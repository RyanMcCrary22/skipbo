package engine

import (
	"math/rand/v2"
	"testing"
)

// ---------------------------------------------------------------------------
// StockPile tests
// ---------------------------------------------------------------------------

func TestStockPile_NewAndTop(t *testing.T) {
	cards := []Card{NewCard(3), NewCard(7), NewCard(1)}
	sp := NewStockPile(cards)

	if sp.Len() != 3 {
		t.Fatalf("stock pile len = %d, want 3", sp.Len())
	}

	top, ok := sp.Top()
	if !ok || top.Value != 1 {
		t.Errorf("Top() = (%s, %v), want (1, true)", top, ok)
	}
}

func TestStockPile_Pop(t *testing.T) {
	sp := NewStockPile([]Card{NewCard(5), NewCard(10)})

	c, ok := sp.Pop()
	if !ok || c.Value != 10 {
		t.Errorf("Pop() = (%s, %v), want (10, true)", c, ok)
	}
	if sp.Len() != 1 {
		t.Errorf("len after pop = %d, want 1", sp.Len())
	}

	// Next top should be 5.
	top, _ := sp.Top()
	if top.Value != 5 {
		t.Errorf("next top = %s, want 5", top)
	}
}

func TestStockPile_Empty(t *testing.T) {
	sp := NewStockPile([]Card{NewCard(1)})
	sp.Pop()

	if !sp.IsEmpty() {
		t.Error("stock pile should be empty after popping only card")
	}
	_, ok := sp.Pop()
	if ok {
		t.Error("Pop on empty stock pile should return false")
	}
}

func TestStockPile_Isolation(t *testing.T) {
	// Ensure NewStockPile copies the slice so external mutations don't affect it.
	cards := []Card{NewCard(1), NewCard(2)}
	sp := NewStockPile(cards)
	cards[0] = NewCard(12)

	top, _ := sp.Top()
	if top.Value != 2 {
		t.Error("stock pile should not be affected by external slice mutation")
	}
}

// ---------------------------------------------------------------------------
// BuildingPile tests
// ---------------------------------------------------------------------------

func TestBuildingPile_EmptyNeedsOne(t *testing.T) {
	bp := &BuildingPile{}
	if bp.NextNeeded() != 1 {
		t.Errorf("empty pile NextNeeded = %d, want 1", bp.NextNeeded())
	}
}

func TestBuildingPile_SequentialPlay(t *testing.T) {
	bp := &BuildingPile{}

	for v := MinValue; v <= MaxValue; v++ {
		if err := bp.Play(NewCard(v)); err != nil {
			t.Fatalf("Play(%d) failed: %v", v, err)
		}
	}

	if !bp.IsComplete() {
		t.Error("pile should be complete after playing 1–12")
	}
	if bp.Len() != 12 {
		t.Errorf("complete pile len = %d, want 12", bp.Len())
	}
}

func TestBuildingPile_RejectWrongCard(t *testing.T) {
	bp := &BuildingPile{}

	// Pile needs 1, try to play 2.
	if err := bp.Play(NewCard(2)); err == nil {
		t.Error("should reject playing 2 on empty pile")
	}

	// Play 1, then try to play 3 (needs 2).
	bp.Play(NewCard(1))
	if err := bp.Play(NewCard(3)); err == nil {
		t.Error("should reject playing 3 when pile needs 2")
	}
}

func TestBuildingPile_WildCard(t *testing.T) {
	bp := &BuildingPile{}

	// Wild can start a pile (acts as 1).
	if err := bp.Play(NewCard(SkipBo)); err != nil {
		t.Fatalf("wild should be playable on empty pile: %v", err)
	}

	// Now pile needs 2; wild works.
	if err := bp.Play(NewCard(SkipBo)); err != nil {
		t.Fatalf("wild should be playable as 2: %v", err)
	}

	// Now pile needs 3; numbered 3 works.
	if err := bp.Play(NewCard(3)); err != nil {
		t.Fatalf("3 should be playable: %v", err)
	}
}

func TestBuildingPile_Clear(t *testing.T) {
	bp := &BuildingPile{}
	for v := MinValue; v <= MaxValue; v++ {
		bp.Play(NewCard(v))
	}

	cleared := bp.Clear()
	if len(cleared) != 12 {
		t.Errorf("cleared %d cards, want 12", len(cleared))
	}
	if bp.Len() != 0 {
		t.Errorf("pile len after clear = %d, want 0", bp.Len())
	}
	if bp.NextNeeded() != 1 {
		t.Errorf("cleared pile NextNeeded = %d, want 1", bp.NextNeeded())
	}
}

func TestBuildingPile_ClearEmpty(t *testing.T) {
	bp := &BuildingPile{}
	cleared := bp.Clear()
	if cleared != nil {
		t.Error("Clear on empty pile should return nil")
	}
}

func TestBuildingPile_TopValue(t *testing.T) {
	bp := &BuildingPile{}
	if bp.TopValue() != 0 {
		t.Errorf("empty pile TopValue = %d, want 0", bp.TopValue())
	}

	bp.Play(NewCard(1))
	if bp.TopValue() != 1 {
		t.Errorf("TopValue after playing 1 = %d, want 1", bp.TopValue())
	}

	bp.Play(NewCard(SkipBo)) // Acts as 2.
	if bp.TopValue() != 2 {
		t.Errorf("TopValue after wild (as 2) = %d, want 2", bp.TopValue())
	}
}

// ---------------------------------------------------------------------------
// DiscardPile tests
// ---------------------------------------------------------------------------

func TestDiscardPile_PushAndTop(t *testing.T) {
	dp := &DiscardPile{}

	if !dp.IsEmpty() {
		t.Error("new discard pile should be empty")
	}

	dp.Push(NewCard(5))
	dp.Push(NewCard(8))

	top, ok := dp.Top()
	if !ok || top.Value != 8 {
		t.Errorf("Top() = (%s, %v), want (8, true)", top, ok)
	}
	if dp.Len() != 2 {
		t.Errorf("len = %d, want 2", dp.Len())
	}
}

func TestDiscardPile_Pop(t *testing.T) {
	dp := &DiscardPile{}
	dp.Push(NewCard(3))
	dp.Push(NewCard(7))

	c, ok := dp.Pop()
	if !ok || c.Value != 7 {
		t.Errorf("Pop() = (%s, %v), want (7, true)", c, ok)
	}

	c, ok = dp.Pop()
	if !ok || c.Value != 3 {
		t.Errorf("Pop() = (%s, %v), want (3, true)", c, ok)
	}

	_, ok = dp.Pop()
	if ok {
		t.Error("Pop on empty discard pile should return false")
	}
}

func TestDiscardPile_AnyCardAllowed(t *testing.T) {
	dp := &DiscardPile{}
	// Any card can go on any discard pile, in any order.
	dp.Push(NewCard(12))
	dp.Push(NewCard(1))
	dp.Push(NewCard(SkipBo))
	dp.Push(NewCard(5))

	if dp.Len() != 4 {
		t.Errorf("len = %d, want 4", dp.Len())
	}
}

func TestDiscardPile_Cards_ReturnsACopy(t *testing.T) {
	dp := &DiscardPile{}
	dp.Push(NewCard(1))
	dp.Push(NewCard(2))

	cards := dp.Cards()
	cards[0] = NewCard(12) // Mutate the copy.

	top, _ := dp.Top()
	if top.Value != 2 {
		t.Error("Cards() should return a copy, not a reference")
	}
}

// ---------------------------------------------------------------------------
// DrawPile tests
// ---------------------------------------------------------------------------

func TestDrawPile_DrawAndLen(t *testing.T) {
	deck := &Deck{Cards: []Card{NewCard(1), NewCard(2), NewCard(3)}}
	dp := NewDrawPile(deck)

	if dp.Len() != 3 {
		t.Fatalf("draw pile len = %d, want 3", dp.Len())
	}

	c, ok := dp.Draw()
	if !ok || c.Value != 3 {
		t.Errorf("Draw() = (%s, %v), want (3, true)", c, ok)
	}
	if dp.Len() != 2 {
		t.Errorf("len after draw = %d, want 2", dp.Len())
	}
}

func TestDrawPile_DrawN(t *testing.T) {
	deck := &Deck{Cards: []Card{NewCard(1), NewCard(2), NewCard(3), NewCard(4), NewCard(5)}}
	dp := NewDrawPile(deck)

	drawn := dp.DrawN(3)
	if len(drawn) != 3 {
		t.Errorf("DrawN(3) returned %d cards", len(drawn))
	}
	if dp.Len() != 2 {
		t.Errorf("len after DrawN(3) = %d, want 2", dp.Len())
	}
}

func TestDrawPile_Empty(t *testing.T) {
	dp := NewDrawPile(&Deck{})
	if !dp.IsEmpty() {
		t.Error("empty draw pile should report IsEmpty")
	}
	_, ok := dp.Draw()
	if ok {
		t.Error("Draw on empty draw pile should return false")
	}
}

func TestDrawPile_Replenish(t *testing.T) {
	dp := NewDrawPile(&Deck{})

	recycled := []Card{NewCard(4), NewCard(5), NewCard(6)}
	rng := rand.New(rand.NewPCG(42, 0))
	dp.Replenish(recycled, rng)

	if dp.Len() != 3 {
		t.Errorf("len after replenish = %d, want 3", dp.Len())
	}

	// Draw all and verify same card set.
	counts := make(map[CardValue]int)
	for !dp.IsEmpty() {
		c, _ := dp.Draw()
		counts[c.Value]++
	}
	for _, v := range []CardValue{4, 5, 6} {
		if counts[v] != 1 {
			t.Errorf("after replenish: count of %d = %d, want 1", v, counts[v])
		}
	}
}

// ---------------------------------------------------------------------------
// Hand tests
// ---------------------------------------------------------------------------

func TestHand_DrawFrom(t *testing.T) {
	deck := &Deck{Cards: []Card{
		NewCard(1), NewCard(2), NewCard(3), NewCard(4), NewCard(5),
		NewCard(6), NewCard(7),
	}}
	dp := NewDrawPile(deck)
	h := &Hand{}

	n := h.DrawFrom(dp)
	if n != 5 {
		t.Errorf("DrawFrom drew %d cards, want 5", n)
	}
	if h.Len() != 5 {
		t.Errorf("hand len = %d, want 5", h.Len())
	}
	if dp.Len() != 2 {
		t.Errorf("draw pile len = %d, want 2", dp.Len())
	}

	// Drawing again should draw 0 (hand already full).
	n = h.DrawFrom(dp)
	if n != 0 {
		t.Errorf("DrawFrom on full hand drew %d, want 0", n)
	}
}

func TestHand_DrawFrom_PartialDeck(t *testing.T) {
	deck := &Deck{Cards: []Card{NewCard(1), NewCard(2)}}
	dp := NewDrawPile(deck)
	h := &Hand{}

	n := h.DrawFrom(dp)
	if n != 2 {
		t.Errorf("DrawFrom drew %d cards, want 2", n)
	}
	if h.Len() != 2 {
		t.Errorf("hand len = %d, want 2", h.Len())
	}
}

func TestHand_Play(t *testing.T) {
	h := &Hand{cards: []Card{NewCard(3), NewCard(7), NewCard(11)}}

	card, err := h.Play(1) // Play index 1 (the 7).
	if err != nil {
		t.Fatalf("Play(1) error: %v", err)
	}
	if card.Value != 7 {
		t.Errorf("played card = %s, want 7", card)
	}
	if h.Len() != 2 {
		t.Errorf("hand len after play = %d, want 2", h.Len())
	}
}

func TestHand_Play_OutOfRange(t *testing.T) {
	h := &Hand{cards: []Card{NewCard(1)}}
	_, err := h.Play(5)
	if err == nil {
		t.Error("Play with out-of-range index should error")
	}
	_, err = h.Play(-1)
	if err == nil {
		t.Error("Play with negative index should error")
	}
}

func TestHand_IsEmpty(t *testing.T) {
	h := &Hand{}
	if !h.IsEmpty() {
		t.Error("new hand should be empty")
	}

	h.cards = append(h.cards, NewCard(1))
	if h.IsEmpty() {
		t.Error("hand with card should not be empty")
	}
}

func TestHand_Cards_ReturnsACopy(t *testing.T) {
	h := &Hand{cards: []Card{NewCard(1), NewCard(2)}}
	cards := h.Cards()
	cards[0] = NewCard(12)

	c, _ := h.Get(0)
	if c.Value != 1 {
		t.Error("Cards() should return a copy")
	}
}

func TestHand_Get(t *testing.T) {
	h := &Hand{cards: []Card{NewCard(5), NewCard(10)}}

	c, err := h.Get(0)
	if err != nil || c.Value != 5 {
		t.Errorf("Get(0) = (%s, %v), want (5, nil)", c, err)
	}

	_, err = h.Get(2)
	if err == nil {
		t.Error("Get out of range should error")
	}
}

// ---------------------------------------------------------------------------
// DiscardArea tests
// ---------------------------------------------------------------------------

func TestDiscardArea_Get(t *testing.T) {
	da := &DiscardArea{}

	for i := 0; i < MaxDiscardPiles; i++ {
		pile, err := da.Get(i)
		if err != nil {
			t.Fatalf("Get(%d) error: %v", i, err)
		}
		pile.Push(NewCard(CardValue(i + 1)))
	}

	// Verify each pile got the right card.
	for i := 0; i < MaxDiscardPiles; i++ {
		pile, _ := da.Get(i)
		top, ok := pile.Top()
		if !ok || top.Value != CardValue(i+1) {
			t.Errorf("pile %d top = (%s, %v), want (%d, true)", i, top, ok, i+1)
		}
	}
}

func TestDiscardArea_Get_OutOfRange(t *testing.T) {
	da := &DiscardArea{}

	_, err := da.Get(-1)
	if err == nil {
		t.Error("Get(-1) should error")
	}

	_, err = da.Get(MaxDiscardPiles)
	if err == nil {
		t.Errorf("Get(%d) should error", MaxDiscardPiles)
	}
}
