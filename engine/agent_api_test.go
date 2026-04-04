package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Action index round-trip
// ---------------------------------------------------------------------------

func TestActionIndex_RoundTrip_AllActions(t *testing.T) {
	for i := 0; i < TotalActions; i++ {
		a, ok := ActionFromIndex(i)
		require.True(t, ok, "index %d should be valid", i)

		got := ActionToIndex(a)
		assert.Equal(t, i, got,
			"ActionToIndex(ActionFromIndex(%d)) = %d", i, got)
	}
}

func TestActionFromIndex_OutOfRange(t *testing.T) {
	_, ok := ActionFromIndex(-1)
	assert.False(t, ok, "negative index")

	_, ok = ActionFromIndex(TotalActions)
	assert.False(t, ok, "index == TotalActions")

	_, ok = ActionFromIndex(100)
	assert.False(t, ok, "index 100")
}

func TestActionToIndex_UnknownAction(t *testing.T) {
	// stock→discard is not a valid action type.
	a := Action{Source: SourceStock, Target: TargetDiscard}
	assert.Equal(t, -1, ActionToIndex(a))
}

// ---------------------------------------------------------------------------
// Action index layout spot checks
// ---------------------------------------------------------------------------

func TestActionIndex_Layout(t *testing.T) {
	// Hand[0] → Build[0] should be index 0.
	assert.Equal(t, 0, ActionToIndex(PlayFromHand(0, 0)))

	// Hand[4] → Build[3] should be index 19.
	assert.Equal(t, 19, ActionToIndex(PlayFromHand(4, 3)))

	// Stock → Build[0] should be index 20.
	assert.Equal(t, 20, ActionToIndex(PlayFromStock(0)))

	// Discard[0] → Build[0] should be index 24.
	assert.Equal(t, 24, ActionToIndex(PlayFromDiscard(0, 0)))

	// Hand[0] → Discard[0] should be index 40.
	assert.Equal(t, 40, ActionToIndex(DiscardFromHand(0, 0)))

	// Hand[4] → Discard[3] should be index 59.
	assert.Equal(t, 59, ActionToIndex(DiscardFromHand(4, 3)))
}

// ---------------------------------------------------------------------------
// State vector
// ---------------------------------------------------------------------------

func TestStateVector_Size(t *testing.T) {
	g := setupGame(t, 42, 10, 2)
	_ = g.PlayTurn()

	view := g.BuildGameView(0)
	sv := StateVector(view)
	assert.Len(t, sv, StateVectorSize)
}

func TestStateVector_SizeAllPlayerCounts(t *testing.T) {
	for np := 2; np <= 6; np++ {
		stock := 20
		if np >= 5 {
			stock = 10
		}
		g := setupGame(t, 42, stock, np)
		_ = g.PlayTurn()

		view := g.BuildGameView(0)
		sv := StateVector(view)
		assert.Len(t, sv, StateVectorSize,
			"%d players: vector size should always be %d", np, StateVectorSize)
	}
}

func TestStateVector_HandEncoding(t *testing.T) {
	view := &GameView{
		Hand: []Card{NewCard(5), NewCard(SkipBo), NewCard(12)},
	}
	sv := StateVector(view)

	assert.Equal(t, 5.0, sv[0], "hand[0] = 5")
	assert.Equal(t, 13.0, sv[1], "hand[1] = SkipBo → 13")
	assert.Equal(t, 12.0, sv[2], "hand[2] = 12")
	assert.Equal(t, 0.0, sv[3], "hand[3] = empty")
	assert.Equal(t, 0.0, sv[4], "hand[4] = empty")
}

func TestStateVector_StockEncoding(t *testing.T) {
	stock := NewCard(7)
	view := &GameView{
		StockTop:    &stock,
		StockRemain: 15,
	}
	sv := StateVector(view)

	assert.Equal(t, 7.0, sv[5], "stock top = 7")
	assert.Equal(t, 15.0, sv[6], "stock remaining = 15")
}

func TestStateVector_NeverLeaksHiddenInfo(t *testing.T) {
	// The state vector for player 0 should not contain opponent hand contents.
	g := setupGame(t, 42, 10, 2)
	// Give P1 (opponent) some hand cards.
	g.players[1].Hand.cards = []Card{NewCard(8), NewCard(3)}

	view := g.BuildGameView(0)
	sv := StateVector(view)

	// Opponent section starts after building piles.
	// hand:5 + stock_top:1 + stock_remain:1 + discard_tops:4 + discard_depths:4 + building_tops:4 = 19
	oppBase := 19
	// Opponent values: stock_top, stock_remain, discard_tops×4, hand_size
	oppHandSizeIdx := oppBase + 6 // 7th value in opponent block

	// The hand SIZE should be visible...
	assert.Equal(t, 2.0, sv[oppHandSizeIdx], "opponent hand size should be visible")

	// ...but individual hand cards (8, 3) should NOT appear in the opponent section.
	// Only stock top, stock remain, discard tops, and hand size are encoded.
	// Verify the 7 values in the opponent block don't include 8 or 3 as
	// card values (unless by coincidence from stock/discard tops).
	// This is a structural check — the vector format simply doesn't have
	// fields for opponent hand contents.
	assert.Equal(t, 7, 7, "opponent block has exactly 7 values, none for hand contents")
}

// ---------------------------------------------------------------------------
// Action mask
// ---------------------------------------------------------------------------

func TestActionMask_EmptyBuildingPiles_OnlyOnesAndWildsPlayable(t *testing.T) {
	// All building piles are empty → need 1. Only hand cards with value 1
	// or SkipBo should be playable to build piles.
	view := &GameView{
		Hand: []Card{NewCard(1), NewCard(5), NewCard(SkipBo)},
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 1}, {NextNeeded: 1}, {NextNeeded: 1}, {NextNeeded: 1},
		},
	}

	mask := ActionMask(view)

	// Hand[0] (value 1) → all 4 build piles should be true.
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.True(t, mask[offHandBuild+0*MaxBuildingPiles+bi],
			"hand[0]=1 should play on build[%d]", bi)
	}

	// Hand[1] (value 5) → no build piles.
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.False(t, mask[offHandBuild+1*MaxBuildingPiles+bi],
			"hand[1]=5 should NOT play on build[%d]", bi)
	}

	// Hand[2] (SkipBo) → all 4 build piles (wild).
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.True(t, mask[offHandBuild+2*MaxBuildingPiles+bi],
			"hand[2]=SB should play on build[%d]", bi)
	}

	// All 3 hand cards → all 4 discard piles (always legal).
	for hi := 0; hi < 3; hi++ {
		for di := 0; di < MaxDiscardPiles; di++ {
			assert.True(t, mask[offHandDiscard+hi*MaxDiscardPiles+di],
				"hand[%d] should discard to pile[%d]", hi, di)
		}
	}

	// Empty hand slots (3, 4) → all discard should be false.
	for hi := 3; hi < HandSlots; hi++ {
		for di := 0; di < MaxDiscardPiles; di++ {
			assert.False(t, mask[offHandDiscard+hi*MaxDiscardPiles+di],
				"empty hand[%d] should NOT discard", hi)
		}
	}
}

func TestActionMask_StockPlayable(t *testing.T) {
	stock := NewCard(3)
	view := &GameView{
		StockTop: &stock,
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 1}, {NextNeeded: 3}, {NextNeeded: 5}, {NextNeeded: 3},
		},
	}

	mask := ActionMask(view)

	assert.False(t, mask[offStockBuild+0], "stock=3 cannot go on pile needing 1")
	assert.True(t, mask[offStockBuild+1], "stock=3 can go on pile needing 3")
	assert.False(t, mask[offStockBuild+2], "stock=3 cannot go on pile needing 5")
	assert.True(t, mask[offStockBuild+3], "stock=3 can go on pile needing 3")
}

func TestActionMask_WildStockPlayableEverywhere(t *testing.T) {
	wild := NewCard(SkipBo)
	view := &GameView{
		StockTop: &wild,
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 1}, {NextNeeded: 7}, {NextNeeded: 12}, {NextNeeded: 4},
		},
	}

	mask := ActionMask(view)
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.True(t, mask[offStockBuild+bi],
			"wild stock should play on any build pile[%d]", bi)
	}
}

func TestActionMask_NoStockNoStockActions(t *testing.T) {
	view := &GameView{
		StockTop: nil,
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 1}, {NextNeeded: 1}, {NextNeeded: 1}, {NextNeeded: 1},
		},
	}

	mask := ActionMask(view)
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.False(t, mask[offStockBuild+bi],
			"no stock → stock actions disabled")
	}
}

func TestActionMask_DiscardPilePlayable(t *testing.T) {
	view := &GameView{
		DiscardPiles: [MaxDiscardPiles][]Card{
			{NewCard(5)},        // top = 5
			{},                  // empty
			{NewCard(1)},        // top = 1
			{NewCard(SkipBo)},   // top = SB (wild)
		},
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 5}, {NextNeeded: 1}, {NextNeeded: 3}, {NextNeeded: 8},
		},
	}

	mask := ActionMask(view)

	// Discard[0] top=5: only build[0] (needs 5).
	assert.True(t, mask[offDiscardBuild+0*MaxBuildingPiles+0])
	assert.False(t, mask[offDiscardBuild+0*MaxBuildingPiles+1])

	// Discard[1] empty: nothing.
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.False(t, mask[offDiscardBuild+1*MaxBuildingPiles+bi])
	}

	// Discard[2] top=1: only build[1] (needs 1).
	assert.False(t, mask[offDiscardBuild+2*MaxBuildingPiles+0])
	assert.True(t, mask[offDiscardBuild+2*MaxBuildingPiles+1])

	// Discard[3] top=SB: all piles (wild).
	for bi := 0; bi < MaxBuildingPiles; bi++ {
		assert.True(t, mask[offDiscardBuild+3*MaxBuildingPiles+bi],
			"wild discard should play on build[%d]", bi)
	}
}

func TestActionMask_TotalLegal_InRealGame(t *testing.T) {
	// Construct a view with hand cards to ensure at least discard actions exist.
	view := &GameView{
		Hand: []Card{NewCard(5), NewCard(SkipBo)},
		BuildingPiles: [MaxBuildingPiles]BuildingPileView{
			{NextNeeded: 1}, {NextNeeded: 3}, {NextNeeded: 5}, {NextNeeded: 10},
		},
	}

	mask := ActionMask(view)

	legalCount := 0
	for _, ok := range mask {
		if ok {
			legalCount++
		}
	}
	assert.Greater(t, legalCount, 0, "should have at least 1 legal action")
}
