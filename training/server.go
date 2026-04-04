package training

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/RyanMcCrary22/skipbo/engine"
	pb "github.com/RyanMcCrary22/skipbo/proto/skipbopb"
)

// ---------------------------------------------------------------------------
// Training Server — gRPC service implementing the SkipBoEnv interface
// ---------------------------------------------------------------------------

// Server implements the SkipBoEnv gRPC service. It manages a single game
// session at a time, bridging the external agent (Python PPO) to the engine.
type Server struct {
	pb.UnimplementedSkipBoEnvServer

	mu sync.Mutex

	// Current game state.
	game     *engine.Game
	obsCh    chan AgentObs
	actionCh chan int
	gameDone chan gameResult

	// Per-step reward accumulator (written by event observer, drained by Step).
	pendingReward float64

	// Metrics tracking.
	metrics Metrics
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
	return &Server{
		obsCh:    make(chan AgentObs, 1),
		actionCh: make(chan int, 1),
		gameDone: make(chan gameResult, 1),
	}
}

// ---------------------------------------------------------------------------
// Reset — start a new game
// ---------------------------------------------------------------------------

func (s *Server) Reset(_ context.Context, req *pb.ResetRequest) (*pb.Observation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine game parameters.
	numOpponents := int(req.NumOpponents)
	if numOpponents <= 0 {
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

	// Create fresh channels for this game session.
	s.obsCh = make(chan AgentObs, 1)
	s.actionCh = make(chan int, 1)
	s.gameDone = make(chan gameResult, 1)

	// Build players: agent at seat 0, random opponents after.
	totalPlayers := 1 + numOpponents
	players := make([]engine.Player, totalPlayers)
	players[0] = NewAgentPlayer("PPO Agent", s.obsCh, s.actionCh)
	for i := 1; i < totalPlayers; i++ {
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
		switch {
		case e.Type == engine.EventCardPlayed && e.Action != nil && e.PlayerIndex == 0:
			// Agent played a card.
			if e.Action.Source == engine.SourceStock {
				s.pendingReward += 5.0 // Stockpile play: primary objective.
				s.metrics.StockpilePlays++
			} else {
				s.pendingReward += 0.1 // Hand/discard play: keep the game moving.
			}
		case e.Type == engine.EventCardPlayed && e.Action != nil &&
			e.Action.Source == engine.SourceStock && e.PlayerIndex != 0:
			// Opponent played from stockpile — agent should learn to block.
			s.pendingReward -= 1.0
		case e.Type == engine.EventIllegalAction && e.PlayerIndex == 0:
			s.metrics.IllegalActions++
		}
	})

	s.game = game

	// Run the game loop in a background goroutine.
	// It will block whenever the AgentPlayer's ChooseAction is called,
	// waiting for us to feed actions via actionCh.
	go func() {
		winner, err := game.Run()
		s.gameDone <- gameResult{winner: winner, err: err}
	}()

	// The game loop will immediately start the agent's first turn,
	// draw cards, and call ChooseAction — which sends an obs to obsCh.
	obs := <-s.obsCh

	return obsToProto(obs), nil
}

// ---------------------------------------------------------------------------
// Step — execute one agent action
// ---------------------------------------------------------------------------

func (s *Server) Step(_ context.Context, req *pb.StepRequest) (*pb.StepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.game == nil {
		return nil, fmt.Errorf("no active game, call Reset first")
	}

	s.metrics.TotalSteps++

	// Send the action to the agent player.
	s.actionCh <- int(req.Action)

	// Now the game loop continues. It will either:
	// 1. Call ChooseAction again (agent gets another move within this turn,
	//    or after opponents complete their turns) → obs arrives on obsCh.
	// 2. The game ends → result arrives on gameDone.

	select {
	case obs := <-s.obsCh:
		// Agent's next decision point. Drain accumulated reward.
		view := s.game.BuildGameView(0)
		reward := s.pendingReward
		s.pendingReward = 0

		return &pb.StepResult{
			Obs:    obsToProto(obs),
			Reward: reward,
			Done:   false,
			Winner: -1,
			Info: &pb.StepInfo{
				TurnNumber:          int32(s.game.TurnNumber()),
				AgentStockRemaining: int32(view.StockRemain),
				PlayedFromStock:     reward >= 5.0,
			},
		}, nil

	case result := <-s.gameDone:
		// Game is over.
		s.metrics.TotalGames++
		s.metrics.TotalTurns += int64(s.game.TurnNumber())
		s.metrics.GamesForAvg++

		winner := result.winner
		if winner == 0 {
			s.metrics.AgentWins++
		}

		// Terminal reward: large bonus/penalty + any pending shaped reward.
		reward := s.pendingReward
		s.pendingReward = 0
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

		s.game = nil // Clear the session.

		return &pb.StepResult{
			Obs:    termObs,
			Reward: reward,
			Done:   true,
			Winner: int32(winner),
			Info: &pb.StepInfo{
				TurnNumber: int32(s.metrics.TotalTurns),
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
