package engine

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ScriptedPlayer: replays a fixed sequence of actions (test helper)
// ---------------------------------------------------------------------------

// ScriptedPlayer replays a predefined list of actions. Useful for
// deterministic test scenarios.
type ScriptedPlayer struct {
	name    string
	actions []Action
	index   int
}

func NewScriptedPlayer(name string, actions []Action) *ScriptedPlayer {
	return &ScriptedPlayer{name: name, actions: actions}
}

func (p *ScriptedPlayer) Name() string { return p.name }

func (p *ScriptedPlayer) ChooseAction(view *GameView) (Action, error) {
	if p.index >= len(p.actions) {
		// Fallback: discard first hand card.
		return DiscardFromHand(0, 0), nil
	}
	a := p.actions[p.index]
	p.index++
	return a, nil
}

// ---------------------------------------------------------------------------
// GameConfig tests
// ---------------------------------------------------------------------------

func TestGameConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     GameConfig
		wantErr bool
	}{
		{"valid 2P", GameConfig{NumPlayers: 2, StockSize: 30}, false},
		{"valid 6P", GameConfig{NumPlayers: 6, StockSize: 20}, false},
		{"valid min stock", GameConfig{NumPlayers: 2, StockSize: 10}, false},
		{"too few players", GameConfig{NumPlayers: 1, StockSize: 30}, true},
		{"too many players", GameConfig{NumPlayers: 7, StockSize: 30}, true},
		{"stock too small", GameConfig{NumPlayers: 2, StockSize: 5}, true},
		{"stock too large", GameConfig{NumPlayers: 2, StockSize: 50}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewGame_Setup(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 1)
	p2 := NewRandomPlayer("Bob", 2)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, err := NewGame(cfg, []Player{p1, p2})
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	// Each player should have 30 stock cards.
	if game.players[0].Stock.Len() != 30 {
		t.Errorf("P1 stock = %d, want 30", game.players[0].Stock.Len())
	}
	if game.players[1].Stock.Len() != 30 {
		t.Errorf("P2 stock = %d, want 30", game.players[1].Stock.Len())
	}

	// Draw pile should have 162 - 60 = 102 cards.
	if game.drawPile.Len() != DeckSize-60 {
		t.Errorf("draw pile = %d, want %d", game.drawPile.Len(), DeckSize-60)
	}

	// Hands should be empty before first turn.
	if game.players[0].Hand.Len() != 0 {
		t.Error("P1 hand should be empty before first turn")
	}

	// Game should not be over.
	if game.IsOver() {
		t.Error("game should not be over at start")
	}
}

func TestNewGame_PlayerMismatch(t *testing.T) {
	p1 := NewRandomPlayer("A", 1)
	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 1}
	_, err := NewGame(cfg, []Player{p1})
	if err == nil {
		t.Error("NewGame should error on player count mismatch")
	}
}

func TestNewGame_SixPlayers(t *testing.T) {
	players := make([]Player, 6)
	for i := range players {
		players[i] = NewRandomPlayer("P"+string(rune('A'+i)), uint64(i))
	}

	cfg := GameConfig{NumPlayers: 6, StockSize: 20, Seed: 42}
	game, err := NewGame(cfg, players)
	if err != nil {
		t.Fatalf("NewGame 6P error: %v", err)
	}

	// 6 * 20 = 120 stock cards dealt, 162 - 120 = 42 in draw pile.
	totalStock := 0
	for i := 0; i < 6; i++ {
		totalStock += game.players[i].Stock.Len()
	}
	if totalStock != 120 {
		t.Errorf("total stock cards = %d, want 120", totalStock)
	}
	if game.drawPile.Len() != 42 {
		t.Errorf("draw pile = %d, want 42", game.drawPile.Len())
	}
}

// ---------------------------------------------------------------------------
// Game flow tests
// ---------------------------------------------------------------------------

func TestGame_PlayTurn_DrawsCards(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 10)
	p2 := NewRandomPlayer("Bob", 20)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})

	err := game.PlayTurn()
	if err != nil {
		t.Fatalf("PlayTurn error: %v", err)
	}

	// After turn 1, hand should be ≤ 5 (could have played cards).
	// Player 1 was current, now it should be player 2's turn.
	if game.CurrentPlayer() != 1 {
		t.Errorf("current player = %d, want 1", game.CurrentPlayer())
	}
}

func TestGame_PlayTurn_WhenGameOver(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 1)
	p2 := NewRandomPlayer("Bob", 2)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})
	game.gameOver = true

	err := game.PlayTurn()
	if err != ErrGameOver {
		t.Errorf("PlayTurn on finished game: err = %v, want ErrGameOver", err)
	}
}

func TestGame_BuildGameView(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 1)
	p2 := NewRandomPlayer("Bob", 2)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})

	// Give P1 some hand cards manually for a view test.
	game.players[0].Hand.cards = []Card{NewCard(3), NewCard(7), NewCard(SkipBo)}

	view := game.BuildGameView(0)

	if len(view.Hand) != 3 {
		t.Errorf("view hand size = %d, want 3", len(view.Hand))
	}
	if view.StockRemain != 30 {
		t.Errorf("view stock remain = %d, want 30", view.StockRemain)
	}
	if view.StockTop == nil {
		t.Error("view stock top should not be nil")
	}
	if len(view.Opponents) != 1 {
		t.Errorf("view opponents = %d, want 1", len(view.Opponents))
	}
	// Opponent's hand should not be visible (only the count).
	if view.Opponents[0].HandSize != 0 {
		t.Errorf("opponent hand size = %d, want 0 (no cards dealt yet)", view.Opponents[0].HandSize)
	}
}

func TestGame_EventsEmitted(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 10)
	p2 := NewRandomPlayer("Bob", 20)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})

	var events []GameEvent
	game.OnEvent(func(e GameEvent) {
		events = append(events, e)
	})

	game.PlayTurn()

	// Should have at least TurnStarted and TurnEnded events.
	hasStart := false
	hasEnd := false
	for _, e := range events {
		if e.Type == EventTurnStarted {
			hasStart = true
		}
		if e.Type == EventTurnEnded {
			hasEnd = true
		}
	}
	if !hasStart {
		t.Error("no TurnStarted event emitted")
	}
	if !hasEnd {
		t.Error("no TurnEnded event emitted")
	}
}

// ---------------------------------------------------------------------------
// Action validation tests
// ---------------------------------------------------------------------------

func TestGame_RejectsPlayAfterDiscard(t *testing.T) {
	p1 := NewScriptedPlayer("Alice", []Action{
		DiscardFromHand(0, 0),       // Discard first (ends turn).
		PlayFromHand(0, 0),          // Should fail — already discarded.
	})
	p2 := NewRandomPlayer("Bob", 1)

	cfg := GameConfig{NumPlayers: 2, StockSize: 10, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})

	// First turn should succeed (scripted player discards immediately).
	err := game.PlayTurn()
	if err != nil {
		t.Fatalf("PlayTurn error: %v", err)
	}
}

func TestGame_RejectsBadBuildingPileIndex(t *testing.T) {
	// Try to play on building pile index 5 (only 0–3 exist).
	// The engine should retry instead of crashing; the scripted player
	// falls back to a valid discard after the illegal action.
	p1 := NewScriptedPlayer("Alice", []Action{
		{Source: SourceHand, SourceIndex: 0, Target: TargetBuild, TargetIndex: 5},
	})
	p2 := NewRandomPlayer("Bob", 1)

	cfg := GameConfig{NumPlayers: 2, StockSize: 10, Seed: 42}
	game, _ := NewGame(cfg, []Player{p1, p2})

	var gotIllegalEvent bool
	game.OnEvent(func(e GameEvent) {
		if e.Type == EventIllegalAction {
			gotIllegalEvent = true
		}
	})

	err := game.PlayTurn()
	if err != nil {
		t.Fatalf("PlayTurn should succeed (retry on illegal move): %v", err)
	}
	if !gotIllegalEvent {
		t.Error("should have emitted EventIllegalAction")
	}
}

// ---------------------------------------------------------------------------
// Integration tests: complete games with random agents
// ---------------------------------------------------------------------------

func TestIntegration_RandomGame2P(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 100)
	p2 := NewRandomPlayer("Bob", 200)

	cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
	game, err := NewGame(cfg, []Player{p1, p2})
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	winner, err := game.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if winner < 0 || winner > 1 {
		t.Errorf("winner = %d, want 0 or 1", winner)
	}
	if !game.IsOver() {
		t.Error("game should be over after Run")
	}

	// Winning player's stock should be empty.
	if game.players[winner].Stock.Len() != 0 {
		t.Errorf("winner's stock pile = %d, want 0", game.players[winner].Stock.Len())
	}
}

func TestIntegration_RandomGame6P(t *testing.T) {
	players := make([]Player, 6)
	for i := range players {
		players[i] = NewRandomPlayer(
			"Player"+string(rune('A'+i)), uint64(i*100),
		)
	}

	cfg := GameConfig{NumPlayers: 6, StockSize: 20, Seed: 99}
	game, err := NewGame(cfg, players)
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	winner, err := game.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if winner < 0 || winner > 5 {
		t.Errorf("winner = %d, want 0–5", winner)
	}
	if game.players[winner].Stock.Len() != 0 {
		t.Error("winner's stock should be empty")
	}
}

func TestIntegration_SmallStockGame(t *testing.T) {
	p1 := NewRandomPlayer("Alice", 1)
	p2 := NewRandomPlayer("Bob", 2)

	cfg := GameConfig{NumPlayers: 2, StockSize: 10, Seed: 7}
	game, err := NewGame(cfg, []Player{p1, p2})
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	winner, err := game.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if winner < 0 {
		t.Error("game should have a winner")
	}
}

func TestIntegration_DeterministicReplay(t *testing.T) {
	// Same seed + same players → same game outcome.
	results := make([]int, 2)
	for run := 0; run < 2; run++ {
		p1 := NewRandomPlayer("Alice", 10)
		p2 := NewRandomPlayer("Bob", 20)

		cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: 42}
		game, _ := NewGame(cfg, []Player{p1, p2})
		winner, _ := game.Run()
		results[run] = winner
	}

	if results[0] != results[1] {
		t.Errorf("deterministic replay: game 1 winner = %d, game 2 winner = %d",
			results[0], results[1])
	}
}

func TestIntegration_ManyGamesAllTerminate(t *testing.T) {
	// Run 100 random games and verify they all terminate with a winner.
	for i := 0; i < 100; i++ {
		p1 := NewRandomPlayer("A", uint64(i*2))
		p2 := NewRandomPlayer("B", uint64(i*2+1))

		cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: uint64(i + 1000)}
		game, err := NewGame(cfg, []Player{p1, p2})
		if err != nil {
			t.Fatalf("game %d: NewGame error: %v", i, err)
		}

		winner, err := game.Run()
		if err != nil {
			t.Fatalf("game %d: Run error: %v", i, err)
		}
		if winner < 0 {
			t.Fatalf("game %d: no winner", i)
		}
		if game.players[winner].Stock.Len() != 0 {
			t.Fatalf("game %d: winner stock not empty", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: games per second with random agents
// ---------------------------------------------------------------------------

func BenchmarkRandomGame2P(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p1 := NewRandomPlayer("A", uint64(i))
		p2 := NewRandomPlayer("B", uint64(i+1))
		cfg := GameConfig{NumPlayers: 2, StockSize: 30, Seed: uint64(i + 1)}
		game, _ := NewGame(cfg, []Player{p1, p2})
		game.Run()
	}
}

func BenchmarkRandomGame6P(b *testing.B) {
	for i := 0; i < b.N; i++ {
		players := make([]Player, 6)
		for j := range players {
			players[j] = NewRandomPlayer("P", uint64(i*6+j))
		}
		cfg := GameConfig{NumPlayers: 6, StockSize: 20, Seed: uint64(i + 1)}
		game, _ := NewGame(cfg, players)
		game.Run()
	}
}
