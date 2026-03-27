package engine

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrGameOver           = errors.New("game is over")
	ErrNotPlayerTurn      = errors.New("not this player's turn")
	ErrMustDiscard        = errors.New("must discard to end turn")
	ErrAlreadyDiscarded   = errors.New("already discarded this turn")
	ErrInvalidSource      = errors.New("invalid action source")
	ErrInvalidTarget      = errors.New("invalid action target")
	ErrCannotDiscardStock = errors.New("cannot discard from stock pile")
	ErrCannotDiscardDiscard = errors.New("cannot move discard to discard")
)

// ---------------------------------------------------------------------------
// GameConfig
// ---------------------------------------------------------------------------

// StockSize bounds.
const (
	MinStockSize     = 10
	MaxStockSize     = 30
	DefaultStockSize = 30
)

// maxActionRetries is the number of consecutive illegal actions allowed
// before the game aborts. Generous enough for human typos, but prevents
// infinite loops from buggy agents.
const maxActionRetries = 1000

// GameConfig holds the settings for a new game.
type GameConfig struct {
	NumPlayers int    // 2–6.
	StockSize  int    // Cards per player's stock pile (10–30, default 30).
	Seed       uint64 // RNG seed for deterministic games. 0 = use random seed.
}

// DefaultConfig returns a sensible default configuration for the given
// number of players.
func DefaultConfig(numPlayers int) GameConfig {
	return GameConfig{
		NumPlayers: numPlayers,
		StockSize:  DefaultStockSize,
		Seed:       0,
	}
}

// validate checks that the config is legal.
func (cfg GameConfig) validate() error {
	if cfg.NumPlayers < 2 || cfg.NumPlayers > 6 {
		return fmt.Errorf("player count %d out of range [2, 6]", cfg.NumPlayers)
	}
	if cfg.StockSize < MinStockSize || cfg.StockSize > MaxStockSize {
		return fmt.Errorf("stock size %d out of range [%d, %d]",
			cfg.StockSize, MinStockSize, MaxStockSize)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PlayerState — the complete mutable state for one player
// ---------------------------------------------------------------------------

// PlayerState holds all the game state belonging to a single player.
type PlayerState struct {
	Player   Player       // The player implementation (human, agent, etc.).
	Stock    *StockPile   // The player's stock pile.
	Hand     Hand         // The player's hand.
	Discards DiscardArea  // The player's 4 discard piles.
}

// ---------------------------------------------------------------------------
// Game
// ---------------------------------------------------------------------------

// Game orchestrates a full Skip-Bo game. It manages the game loop, validates
// all actions, and enforces the rules. The Game struct owns all mutable state;
// Player implementations only see read-only GameView snapshots.
type Game struct {
	config GameConfig
	rng    *rand.Rand

	drawPile      *DrawPile
	buildingPiles [MaxBuildingPiles]BuildingPile
	players       []PlayerState

	currentPlayer int  // Index of the player whose turn it is.
	turnNumber    int  // Incremented each turn.
	hasDiscarded  bool // Whether the current player has discarded this turn.
	gameOver      bool // Whether the game has ended.
	winner        int  // Index of the winning player (-1 if game not over).

	// Observers receive events for GUI updates, logging, etc.
	observers []func(GameEvent)
}

// NewGame creates and sets up a new Skip-Bo game.
// The players slice must match cfg.NumPlayers in length.
func NewGame(cfg GameConfig, players []Player) (*Game, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if len(players) != cfg.NumPlayers {
		return nil, fmt.Errorf("config says %d players but %d provided",
			cfg.NumPlayers, len(players))
	}

	// Create RNG. Seed 0 means "pick something", but we still make it
	// deterministic by using a PCG source so the caller can choose.
	var rng *rand.Rand
	if cfg.Seed != 0 {
		rng = rand.New(rand.NewPCG(cfg.Seed, 0))
	} else {
		// Use a random seed. In production you'd log this for reproducibility.
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}

	// Build and shuffle the deck.
	deck := NewStandardDeck()
	deck.Shuffle(rng)

	// Deal stock piles.
	pstates := make([]PlayerState, cfg.NumPlayers)
	for i := 0; i < cfg.NumPlayers; i++ {
		stockCards := deck.DrawN(cfg.StockSize)
		pstates[i] = PlayerState{
			Player: players[i],
			Stock:  NewStockPile(stockCards),
		}
	}

	// Remaining cards become the draw pile.
	drawPile := NewDrawPile(deck)

	g := &Game{
		config:        cfg,
		rng:           rng,
		drawPile:      drawPile,
		players:       pstates,
		currentPlayer: 0,
		winner:        -1,
	}

	return g, nil
}

// ---------------------------------------------------------------------------
// Observer registration
// ---------------------------------------------------------------------------

// OnEvent registers a callback to be notified of game events.
func (g *Game) OnEvent(fn func(GameEvent)) {
	g.observers = append(g.observers, fn)
}

func (g *Game) emit(e GameEvent) {
	for _, fn := range g.observers {
		fn(e)
	}
}

// ---------------------------------------------------------------------------
// Game accessors
// ---------------------------------------------------------------------------

// IsOver reports whether the game has ended.
func (g *Game) IsOver() bool { return g.gameOver }

// Winner returns the index of the winning player, or -1 if the game is not over.
func (g *Game) Winner() int { return g.winner }

// CurrentPlayer returns the index of the player whose turn it is.
func (g *Game) CurrentPlayer() int { return g.currentPlayer }

// TurnNumber returns the current turn counter.
func (g *Game) TurnNumber() int { return g.turnNumber }

// NumPlayers returns how many players are in the game.
func (g *Game) NumPlayers() int { return len(g.players) }

// PlayerName returns the display name of the given player.
func (g *Game) PlayerName(index int) string {
	return g.players[index].Player.Name()
}

// ---------------------------------------------------------------------------
// GameView construction
// ---------------------------------------------------------------------------

// buildGameView creates a GameView snapshot for the given player index.
func (g *Game) buildGameView(playerIdx int) *GameView {
	ps := &g.players[playerIdx]

	view := &GameView{
		Hand:          ps.Hand.Cards(),
		StockRemain:   ps.Stock.Len(),
		DrawPileSize:  g.drawPile.Len(),
		CurrentPlayer: g.currentPlayer,
		PlayerIndex:   playerIdx,
		TurnNumber:    g.turnNumber,
	}

	// Stock top card.
	if top, ok := ps.Stock.Top(); ok {
		view.StockTop = &top
	}

	// Own discard piles (fully visible).
	for i := 0; i < MaxDiscardPiles; i++ {
		view.DiscardPiles[i] = ps.Discards.Piles[i].Cards()
	}

	// Building piles.
	for i := 0; i < MaxBuildingPiles; i++ {
		bp := &g.buildingPiles[i]
		view.BuildingPiles[i] = BuildingPileView{
			TopValue:   bp.TopValue(),
			NextNeeded: bp.NextNeeded(),
			Size:       bp.Len(),
		}
	}

	// Opponents.
	for i := 0; i < len(g.players); i++ {
		if i == playerIdx {
			continue
		}
		opp := &g.players[i]
		ov := OpponentView{
			Name:        opp.Player.Name(),
			StockRemain: opp.Stock.Len(),
			HandSize:    opp.Hand.Len(),
		}
		if top, ok := opp.Stock.Top(); ok {
			ov.StockTop = &top
		}
		for j := 0; j < MaxDiscardPiles; j++ {
			if top, ok := opp.Discards.Piles[j].Top(); ok {
				ov.DiscardTops[j] = &top
			}
		}
		view.Opponents = append(view.Opponents, ov)
	}

	return view
}

// ---------------------------------------------------------------------------
// Turn execution
// ---------------------------------------------------------------------------

// PlayTurn executes a single player's complete turn.
// It draws cards, then repeatedly asks the player for actions until they
// discard (ending the turn) or win, handling draw-5-more when the hand
// empties mid-turn.
func (g *Game) PlayTurn() error {
	if g.gameOver {
		return ErrGameOver
	}

	ps := &g.players[g.currentPlayer]
	g.hasDiscarded = false
	g.turnNumber++

	g.emit(GameEvent{
		Type:        EventTurnStarted,
		PlayerIndex: g.currentPlayer,
		Message:     fmt.Sprintf("%s's turn begins", ps.Player.Name()),
	})

	// Step 1: Draw up to 5 cards.
	g.drawUpToFive(ps)

	// Step 2: Play loop — player makes actions until they discard or win.
	retries := 0
	for {
		if g.gameOver {
			return nil
		}

		view := g.buildGameView(g.currentPlayer)
		action, err := ps.Player.ChooseAction(view)
		if err != nil {
			// Player-level errors (disconnect, etc.) are fatal.
			return fmt.Errorf("player %s error: %w", ps.Player.Name(), err)
		}

		if err := g.executeAction(g.currentPlayer, action); err != nil {
			// Illegal move — notify observers and let the player try again.
			g.emit(GameEvent{
				Type:        EventIllegalAction,
				PlayerIndex: g.currentPlayer,
				Message:     fmt.Sprintf("Illegal move: %v", err),
			})
			retries++
			if retries > maxActionRetries {
				return fmt.Errorf("player %s exceeded %d illegal action retries",
					ps.Player.Name(), maxActionRetries)
			}
			continue
		}
		retries = 0 // Reset on successful action.

		// If the player discarded, the turn is over.
		if g.hasDiscarded {
			g.emit(GameEvent{
				Type:        EventTurnEnded,
				PlayerIndex: g.currentPlayer,
				Message:     fmt.Sprintf("%s's turn ends", ps.Player.Name()),
			})
			g.advanceTurn()
			return nil
		}

		// If hand is empty after playing cards, draw 5 more and continue.
		if ps.Hand.IsEmpty() {
			g.emit(GameEvent{
				Type:        EventHandRefilled,
				PlayerIndex: g.currentPlayer,
				Message:     fmt.Sprintf("%s played all hand cards, drawing 5 more", ps.Player.Name()),
			})
			g.drawUpToFive(ps)
		}
	}
}

// Run plays the game to completion, alternating turns until someone wins.
// Returns the index of the winner.
func (g *Game) Run() (int, error) {
	for !g.gameOver {
		if err := g.PlayTurn(); err != nil {
			return -1, err
		}
	}
	return g.winner, nil
}

// ---------------------------------------------------------------------------
// Action execution
// ---------------------------------------------------------------------------

// executeAction validates and performs a single action.
func (g *Game) executeAction(playerIdx int, a Action) error {
	ps := &g.players[playerIdx]

	// Validate: can't act after discarding.
	if g.hasDiscarded {
		return ErrAlreadyDiscarded
	}

	// Get the card from the source.
	card, err := g.extractCard(ps, a)
	if err != nil {
		return err
	}

	// Place the card on the target.
	if err := g.placeCard(ps, a, card); err != nil {
		// If placement fails, we need to put the card back.
		// This is tricky for hand cards (order changed), but for correctness
		// we add it back to hand.
		g.returnCard(ps, a, card)
		return err
	}

	// Check if playing from stock emptied it → win.
	if a.Source == SourceStock && ps.Stock.IsEmpty() {
		g.gameOver = true
		g.winner = playerIdx
		g.emit(GameEvent{
			Type:        EventGameOver,
			PlayerIndex: playerIdx,
			Message:     fmt.Sprintf("%s wins!", ps.Player.Name()),
		})
	}

	return nil
}

// extractCard removes and returns the card specified by the action's source.
func (g *Game) extractCard(ps *PlayerState, a Action) (Card, error) {
	switch a.Source {
	case SourceHand:
		if a.Target == TargetBuild || a.Target == TargetDiscard {
			return ps.Hand.Play(a.SourceIndex)
		}
		return Card{}, ErrInvalidTarget

	case SourceStock:
		if a.Target == TargetDiscard {
			return Card{}, ErrCannotDiscardStock
		}
		card, ok := ps.Stock.Pop()
		if !ok {
			return Card{}, ErrPileEmpty
		}
		return card, nil

	case SourceDiscard:
		if a.Target == TargetDiscard {
			return Card{}, ErrCannotDiscardDiscard
		}
		pile, err := ps.Discards.Get(a.SourceIndex)
		if err != nil {
			return Card{}, err
		}
		card, ok := pile.Pop()
		if !ok {
			return Card{}, ErrPileEmpty
		}
		return card, nil

	default:
		return Card{}, ErrInvalidSource
	}
}

// placeCard puts a card onto the target specified by the action.
func (g *Game) placeCard(ps *PlayerState, a Action, card Card) error {
	switch a.Target {
	case TargetBuild:
		if a.TargetIndex < 0 || a.TargetIndex >= MaxBuildingPiles {
			return fmt.Errorf("%w: building pile index %d", ErrInvalidTarget, a.TargetIndex)
		}
		bp := &g.buildingPiles[a.TargetIndex]
		if err := bp.Play(card); err != nil {
			return err
		}

		g.emit(GameEvent{
			Type:        EventCardPlayed,
			PlayerIndex: g.currentPlayer,
			Card:        &card,
			PileIndex:   a.TargetIndex,
			Message:     fmt.Sprintf("%s plays %s on building pile %d", ps.Player.Name(), card, a.TargetIndex),
		})

		// Check for completed pile.
		if bp.IsComplete() {
			cleared := bp.Clear()
			g.drawPile.Replenish(cleared, g.rng)
			g.emit(GameEvent{
				Type:        EventPileCompleted,
				PlayerIndex: g.currentPlayer,
				PileIndex:   a.TargetIndex,
				Message:     fmt.Sprintf("Building pile %d completed and cleared", a.TargetIndex),
			})
		}
		return nil

	case TargetDiscard:
		if a.TargetIndex < 0 || a.TargetIndex >= MaxDiscardPiles {
			return fmt.Errorf("%w: discard pile index %d", ErrInvalidTarget, a.TargetIndex)
		}
		pile, _ := ps.Discards.Get(a.TargetIndex)
		pile.Push(card)
		g.hasDiscarded = true

		g.emit(GameEvent{
			Type:        EventCardDiscarded,
			PlayerIndex: g.currentPlayer,
			Card:        &card,
			PileIndex:   a.TargetIndex,
			Message:     fmt.Sprintf("%s discards %s to discard pile %d", ps.Player.Name(), card, a.TargetIndex),
		})
		return nil

	default:
		return ErrInvalidTarget
	}
}

// returnCard puts a card back to its source when a placement fails.
// This ensures atomicity of action execution.
func (g *Game) returnCard(ps *PlayerState, a Action, card Card) {
	switch a.Source {
	case SourceHand:
		ps.Hand.cards = append(ps.Hand.cards, card)
	case SourceStock:
		// Push back. StockPile doesn't have Push, so we reconstruct.
		ps.Stock.cards = append(ps.Stock.cards, card)
	case SourceDiscard:
		pile, _ := ps.Discards.Get(a.SourceIndex)
		pile.Push(card)
	}
}

// drawUpToFive draws cards from the draw pile to fill the player's hand.
// If the draw pile runs out, no more cards are drawn (the game continues
// with whatever cards are available — complete building piles replenish it).
func (g *Game) drawUpToFive(ps *PlayerState) {
	ps.Hand.DrawFrom(g.drawPile)
}

// advanceTurn moves to the next player.
func (g *Game) advanceTurn() {
	g.currentPlayer = (g.currentPlayer + 1) % len(g.players)
}
