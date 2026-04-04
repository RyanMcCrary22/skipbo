package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers: game setup utilities
// ---------------------------------------------------------------------------

// setupGame creates a game with the given parameters using RandomPlayers.
// It fails the test immediately if game creation errors.
func setupGame(t *testing.T, seed uint64, stockSize, numPlayers int) *Game {
	t.Helper()
	players := make([]Player, numPlayers)
	for i := range players {
		players[i] = NewRandomPlayer("P"+string(rune('A'+i)), uint64(i+1))
	}
	cfg := GameConfig{NumPlayers: numPlayers, StockSize: stockSize, Seed: seed}
	game, err := NewGame(cfg, players)
	require.NoError(t, err, "setupGame failed")
	return game
}

// setupMockGame creates a game with MockPlayers so tests can control actions.
func setupMockGame(t *testing.T, seed uint64, stockSize int, mocks ...*MockPlayer) *Game {
	t.Helper()
	players := make([]Player, len(mocks))
	for i, m := range mocks {
		players[i] = m
	}
	cfg := GameConfig{NumPlayers: len(mocks), StockSize: stockSize, Seed: seed}
	game, err := NewGame(cfg, players)
	require.NoError(t, err, "setupMockGame failed")
	return game
}

// countAllCards tallies every card across all game locations.
// This includes the draw pile, all players' hands, stock piles, discard
// piles, and all building piles. The total should always equal DeckSize.
func countAllCards(g *Game) map[CardValue]int {
	counts := make(map[CardValue]int)

	// Draw pile.
	for _, c := range g.drawPile.deck.Cards {
		counts[c.Value]++
	}

	// Building piles.
	for i := 0; i < MaxBuildingPiles; i++ {
		for _, c := range g.buildingPiles[i].cards {
			counts[c.Value]++
		}
	}

	// Each player's hand, stock, and discard piles.
	for i := range g.players {
		ps := &g.players[i]
		for _, c := range ps.Hand.cards {
			counts[c.Value]++
		}
		for _, c := range ps.Stock.cards {
			counts[c.Value]++
		}
		for j := 0; j < MaxDiscardPiles; j++ {
			for _, c := range ps.Discards.Piles[j].cards {
				counts[c.Value]++
			}
		}
	}

	return counts
}

// totalCardCount returns the total number of cards across all locations.
func totalCardCount(g *Game) int {
	total := 0
	for _, n := range countAllCards(g) {
		total += n
	}
	return total
}

// assertCardConservation verifies that no cards were created or destroyed.
func assertCardConservation(t *testing.T, g *Game) {
	t.Helper()
	counts := countAllCards(g)

	for v := MinValue; v <= MaxValue; v++ {
		require.Equal(t, CopiesPerValue, counts[v],
			"card conservation violated for value %d", v)
	}
	require.Equal(t, WildCount, counts[SkipBo],
		"card conservation violated for wilds")
	require.Equal(t, DeckSize, totalCardCount(g),
		"total card count != DeckSize")
}
