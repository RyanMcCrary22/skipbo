package engine

// ---------------------------------------------------------------------------
// Player — the core interface for anything that makes decisions
// ---------------------------------------------------------------------------

// Player represents any entity that can play Skip-Bo: a human via CLI,
// a GUI click handler, an AI agent over gRPC, or a test mock.
//
// The interface is intentionally minimal so that new player types
// (agent protocols, network players, etc.) are easy to implement.
type Player interface {
	// ChooseAction is called when it is the player's turn.
	// The player examines the visible game state and returns an action.
	// This is called repeatedly within a turn until the player discards
	// (ending the turn) or wins.
	//
	// Returning an error signals an unrecoverable issue (disconnect, etc.)
	// and will abort the game.
	ChooseAction(view *GameView) (Action, error)

	// Name returns a display name for the player.
	Name() string
}

// ---------------------------------------------------------------------------
// GameView — the visible game state from a player's perspective
// ---------------------------------------------------------------------------

// GameView provides a read-only snapshot of the game state visible to a
// specific player. It enforces imperfect information by hiding other
// players' hands and face-down stock cards.
type GameView struct {
	// Current player's own state.
	Hand         []Card            // Cards in hand.
	StockTop     *Card             // Face-up top card of stock pile (nil if empty).
	StockRemain  int               // Number of cards remaining in stock pile.
	DiscardPiles [MaxDiscardPiles][]Card // Full contents of own discard piles (all visible).

	// Shared state.
	BuildingPiles [MaxBuildingPiles]BuildingPileView // The 4 center building piles.
	DrawPileSize  int                                // Cards remaining in draw pile.

	// Other players' visible state.
	Opponents []OpponentView

	// Meta.
	CurrentPlayer int // Index of the current player (0-based).
	PlayerIndex   int // This player's index.
	TurnNumber    int // Turn counter (for logging/debugging).
}

// MaxBuildingPiles is the number of shared building piles.
const MaxBuildingPiles = 4

// BuildingPileView is a read-only view of a building pile.
type BuildingPileView struct {
	TopValue  CardValue // Effective top value (0 if empty).
	NextNeeded CardValue // The value needed to play on this pile.
	Size      int       // Number of cards in the pile.
}

// OpponentView contains the visible information about another player.
type OpponentView struct {
	Name         string             // Display name.
	StockTop     *Card              // Visible top of their stock pile.
	StockRemain  int                // Cards remaining in their stock.
	DiscardTops  [MaxDiscardPiles]*Card // Top card of each discard pile (nil if empty).
	HandSize     int                // How many cards they're holding (but not what).
}

// ---------------------------------------------------------------------------
// GameEvent — notifications about game state changes
// ---------------------------------------------------------------------------

// GameEventType categorizes a game event.
type GameEventType int

const (
	EventCardPlayed    GameEventType = iota // A card was played on a building pile.
	EventCardDiscarded                      // A card was discarded.
	EventPileCompleted                      // A building pile reached 12 and was cleared.
	EventTurnStarted                        // A player's turn began.
	EventTurnEnded                          // A player's turn ended.
	EventHandRefilled                       // A player drew cards (hand was empty, got 5 more).
	EventGameOver                           // The game ended.
	EventIllegalAction                      // A player attempted an illegal action.
)

// GameEvent carries information about a state change.
// Observers (GUI, logging, etc.) can watch these without coupling to the engine.
type GameEvent struct {
	Type        GameEventType
	PlayerIndex int       // Which player is involved.
	Card        *Card     // The card involved (if applicable).
	PileIndex   int       // Which pile (building or discard) is involved.
	Message     string    // Human-readable description.
}
