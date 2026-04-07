package training

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"time"
	"github.com/google/uuid"

	"github.com/RyanMcCrary22/skipbo/engine"
	pb "github.com/RyanMcCrary22/skipbo/proto/skipbopb"
)

// ---------------------------------------------------------------------------
// Training Server — gRPC service implementing the SkipBoEnv interface
// ---------------------------------------------------------------------------

// GameSession holds the state and channels for a single isolated training game.
type GameSession struct {
	id            string
	game          *engine.Game
	obsCh         chan AgentObs
	actionCh      chan int
	gameDone      chan gameResult
	pendingReward float64
	cancel        context.CancelFunc
	lastActive    time.Time
	resources     []io.Closer
}

// Server implements the SkipBoEnv gRPC service. It maintains concurrent
// game sessions uniquely identified by a combination of a GUID or ID.
type Server struct {
	pb.UnimplementedSkipBoEnvServer

	mu sync.RWMutex

	// Active game sessions.
	sessions map[string]*GameSession

	// Metrics tracking.
	metricsMu sync.Mutex
	metrics   Metrics
}

// Metrics tracks cumulative training statistics.
type Metrics struct {
	TotalGames      int64
	TotalSteps      int64
	AgentWins       int64
	TotalTurns      int64
	StockpilePlays  int64
	IllegalActions  int64
	GamesForAvg     int64 // number of completed games for averaging
}

// gameResult is sent when a game finishes.
type gameResult struct {
	winner int
	err    error
}

// NewServer creates a new training server.
func NewServer() *Server {
	srv := &Server{
		sessions: make(map[string]*GameSession),
	}
	go srv.reapZombieSessions()
	return srv
}

func (s *Server) reapZombieSessions() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		s.mu.Lock()
		for id, sess := range s.sessions {
			if time.Since(sess.lastActive) > 10*time.Minute {
				log.Printf("Reaping zombie game session: %s", id)
				// Cleanly terminate the Game context.
				if sess.cancel != nil {
					sess.cancel()
				}
				// Force cleanup of ONNX or other resources.
				for _, r := range sess.resources {
					r.Close()
				}
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Reset — start a new game
// ---------------------------------------------------------------------------

func (s *Server) Reset(_ context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine game parameters.
	numOpponents := int(req.NumOpponents)
	if len(req.OpponentModels) > 0 {
		numOpponents = len(req.OpponentModels)
	} else if numOpponents <= 0 {
		numOpponents = 1
	}

	stockSize := int(req.StockSize)
	if stockSize < engine.MinStockSize || stockSize > engine.MaxStockSize {
		stockSize = engine.DefaultStockSize
	}
	seed := req.Seed
	if seed == 0 {
		seed = rand.Uint64()
	}

	// Create fresh channels and session.
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	sess := &GameSession{
		id:         sessionID,
		obsCh:      make(chan AgentObs, 1),
		actionCh:   make(chan int, 1),
		gameDone:   make(chan gameResult, 1),
		cancel:     cancel,
		lastActive: time.Now(),
		resources:  []io.Closer{},
	}

	// Build players: agent at seat 0, random opponents after.
	totalPlayers := 1 + numOpponents
	players := make([]engine.Player, totalPlayers)
	players[0] = NewAgentPlayer("PPO Agent", sess.obsCh, sess.actionCh, ctx.Done())
	
	for i := 1; i < totalPlayers; i++ {
		if len(req.OpponentModels) > 0 {
			modelPath := req.OpponentModels[i-1]
			if modelPath != "" && modelPath != "random" {
				fullPath := filepath.Join("models", modelPath)
				p, err := NewOnnxPlayer(fmt.Sprintf("ONNX %d", i), fullPath)
				if err != nil {
					cancel()
					return nil, fmt.Errorf("failed to load onnx model %s: %w", modelPath, err)
				}
				sess.resources = append(sess.resources, p)
				players[i] = p
				continue
			}
		}
		players[i] = engine.NewRandomPlayer(
			fmt.Sprintf("Random %d", i),
			rand.Uint64(),
		)
	}

	cfg := engine.GameConfig{
		NumPlayers: totalPlayers,
		StockSize:  stockSize,
		Seed:       seed,
	}

	game, err := engine.NewGame(cfg, players)
	if err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	// Track stockpile plays, opponent progress, and illegal actions.
	game.OnEvent(func(e engine.GameEvent) {
		s.metricsMu.Lock()
		defer s.metricsMu.Unlock()

		switch {
		case e.Type == engine.EventCardPlayed && e.Action != nil && e.PlayerIndex == 0:
			// Agent played a card.
			if e.Action.Source == engine.SourceStock {
				sess.pendingReward += 5.0 // Stockpile play: primary objective.
				s.metrics.StockpilePlays++
			} else {
				sess.pendingReward += 0.1 // Hand/discard play: keep the game moving.
			}
		case e.Type == engine.EventCardPlayed && e.Action != nil &&
			e.Action.Source == engine.SourceStock && e.PlayerIndex != 0:
			// Opponent played from stockpile — agent should learn to block.
			sess.pendingReward -= 1.0
		case e.Type == engine.EventIllegalAction && e.PlayerIndex == 0:
			s.metrics.IllegalActions++
		}
	})

	sess.game = game
	s.sessions[sessionID] = sess

	// Run the game loop in a background goroutine.
	// It will block whenever the AgentPlayer's ChooseAction is called,
	// waiting for us to feed actions via actionCh.
	go func() {
		winner, err := game.Run()
		sess.gameDone <- gameResult{winner: winner, err: err}
	}()

	// The game loop will immediately start the agent's first turn,
	// draw cards, and call ChooseAction — which sends an obs to obsCh.
	obs := <-sess.obsCh

	return &pb.ResetResponse{
		SessionId: sessionID,
		Obs:       obsToProto(obs),
	}, nil
}

// ---------------------------------------------------------------------------
// Step — execute one agent action
// ---------------------------------------------------------------------------

func (s *Server) Step(_ context.Context, req *pb.StepRequest) (*pb.StepResult, error) {
	s.mu.RLock()
	sess, ok := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid session id, call Reset first")
	}

	s.mu.Lock()
	sess.lastActive = time.Now()
	s.mu.Unlock()

	s.metricsMu.Lock()
	s.metrics.TotalSteps++
	s.metricsMu.Unlock()

	// Send the action to the agent player.
	sess.actionCh <- int(req.Action)

	// Now the game loop continues. It will either:
	// 1. Call ChooseAction again (agent gets another move within this turn,
	//    or after opponents complete their turns) → obs arrives on obsCh.
	// 2. The game ends → result arrives on gameDone.

	select {
	case obs := <-sess.obsCh:
		// Agent's next decision point. Drain accumulated reward.
		s.metricsMu.Lock()
		reward := sess.pendingReward
		sess.pendingReward = 0
		s.metricsMu.Unlock()

		view := sess.game.BuildGameView(0)

		return &pb.StepResult{
			Obs:    obsToProto(obs),
			Reward: reward,
			Done:   false,
			Winner: -1,
			Info: &pb.StepInfo{
				TurnNumber:          int32(sess.game.TurnNumber()),
				AgentStockRemaining: int32(view.StockRemain),
				PlayedFromStock:     reward >= 5.0,
			},
		}, nil

	case result := <-sess.gameDone:
		s.mu.Lock()
		delete(s.sessions, req.SessionId)
		// Close context and models.
		if sess.cancel != nil {
			sess.cancel()
		}
		for _, r := range sess.resources {
			r.Close()
		}
		s.mu.Unlock()

		// Game is over.
		s.metricsMu.Lock()
		defer s.metricsMu.Unlock()

		s.metrics.TotalGames++
		s.metrics.TotalTurns += int64(sess.game.TurnNumber())
		s.metrics.GamesForAvg++

		winner := result.winner
		if winner == 0 {
			s.metrics.AgentWins++
		}

		// Terminal reward: large bonus/penalty + any pending shaped reward.
		reward := sess.pendingReward
		if winner == 0 {
			reward += 100.0
		} else {
			reward -= 100.0
		}

		// Build terminal observation (zeros — game is done).
		termObs := &pb.Observation{
			State:      make([]float64, engine.StateVectorSize),
			ActionMask: make([]bool, engine.TotalActions),
		}

		return &pb.StepResult{
			Obs:    termObs,
			Reward: reward,
			Done:   true,
			Winner: int32(winner),
			Info: &pb.StepInfo{
				TurnNumber: int32(sess.game.TurnNumber()),
			},
		}, nil
	}
}

// ---------------------------------------------------------------------------
// GetMetrics
// ---------------------------------------------------------------------------

func (s *Server) GetMetrics(_ context.Context, _ *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	avgLen := 0.0
	avgStock := 0.0
	if s.metrics.GamesForAvg > 0 {
		avgLen = float64(s.metrics.TotalTurns) / float64(s.metrics.GamesForAvg)
		avgStock = float64(s.metrics.StockpilePlays) / float64(s.metrics.GamesForAvg)
	}

	return &pb.MetricsResponse{
		TotalGames:       s.metrics.TotalGames,
		TotalSteps:       s.metrics.TotalSteps,
		AgentWins:        s.metrics.AgentWins,
		AvgGameLength:    avgLen,
		AvgStockpilePlays: avgStock,
		IllegalActions:   s.metrics.IllegalActions,
	}, nil
}

// (Reward shaping is now handled directly by the event observer and the
// pendingReward accumulator — see Reset and Step methods above.)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func obsToProto(obs AgentObs) *pb.Observation {
	mask := make([]bool, engine.TotalActions)
	copy(mask, obs.ActionMask[:])
	return &pb.Observation{
		State:      obs.State,
		ActionMask: mask,
	}
}
