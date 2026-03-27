package engine

import (
	"math/rand/v2"
	"testing"
)

// ---------------------------------------------------------------------------
// CardValue tests
// ---------------------------------------------------------------------------

func TestCardValue_IsWild(t *testing.T) {
	if !SkipBo.IsWild() {
		t.Error("SkipBo should be wild")
	}
	for v := MinValue; v <= MaxValue; v++ {
		if v.IsWild() {
			t.Errorf("CardValue %d should not be wild", v)
		}
	}
}

func TestCardValue_IsValid(t *testing.T) {
	valid := []CardValue{SkipBo, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	for _, v := range valid {
		if !v.IsValid() {
			t.Errorf("CardValue %d should be valid", v)
		}
	}

	invalid := []CardValue{-1, 13, 100}
	for _, v := range invalid {
		if v.IsValid() {
			t.Errorf("CardValue %d should be invalid", v)
		}
	}
}

func TestCardValue_String(t *testing.T) {
	tests := []struct {
		v    CardValue
		want string
	}{
		{SkipBo, "SB"},
		{1, "1"},
		{12, "12"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("CardValue(%d).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Card tests
// ---------------------------------------------------------------------------

func TestNewCard_Valid(t *testing.T) {
	for v := MinValue; v <= MaxValue; v++ {
		c := NewCard(v)
		if c.Value != v {
			t.Errorf("NewCard(%d).Value = %d", v, c.Value)
		}
	}
	c := NewCard(SkipBo)
	if c.Value != SkipBo {
		t.Errorf("NewCard(SkipBo).Value = %d, want 0", c.Value)
	}
}

func TestNewCard_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCard(99) should panic")
		}
	}()
	NewCard(99)
}

func TestCard_CanPlayOn(t *testing.T) {
	tests := []struct {
		card       Card
		nextNeeded CardValue
		want       bool
	}{
		// Exact match.
		{NewCard(1), 1, true},
		{NewCard(5), 5, true},
		{NewCard(12), 12, true},

		// Mismatch.
		{NewCard(2), 1, false},
		{NewCard(1), 2, false},
		{NewCard(11), 12, false},

		// Wild always works.
		{NewCard(SkipBo), 1, true},
		{NewCard(SkipBo), 6, true},
		{NewCard(SkipBo), 12, true},
	}
	for _, tt := range tests {
		got := tt.card.CanPlayOn(tt.nextNeeded)
		if got != tt.want {
			t.Errorf("Card(%s).CanPlayOn(%s) = %v, want %v",
				tt.card, tt.nextNeeded, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Deck tests
// ---------------------------------------------------------------------------

func TestNewStandardDeck_Size(t *testing.T) {
	d := NewStandardDeck()
	if d.Len() != DeckSize {
		t.Errorf("deck size = %d, want %d", d.Len(), DeckSize)
	}
}

func TestNewStandardDeck_CardCounts(t *testing.T) {
	d := NewStandardDeck()

	counts := make(map[CardValue]int)
	for _, c := range d.Cards {
		counts[c.Value]++
	}

	// 12 copies of each numbered card.
	for v := MinValue; v <= MaxValue; v++ {
		if counts[v] != CopiesPerValue {
			t.Errorf("count of %d = %d, want %d", v, counts[v], CopiesPerValue)
		}
	}

	// 18 wilds.
	if counts[SkipBo] != WildCount {
		t.Errorf("wild count = %d, want %d", counts[SkipBo], WildCount)
	}
}

func TestDeck_Shuffle_Deterministic(t *testing.T) {
	seed := uint64(42)

	d1 := NewStandardDeck()
	d1.Shuffle(rand.New(rand.NewPCG(seed, 0)))

	d2 := NewStandardDeck()
	d2.Shuffle(rand.New(rand.NewPCG(seed, 0)))

	if d1.Len() != d2.Len() {
		t.Fatalf("decks have different lengths after same-seed shuffle")
	}
	for i := range d1.Cards {
		if d1.Cards[i] != d2.Cards[i] {
			t.Fatalf("card %d differs after same-seed shuffle: %v vs %v",
				i, d1.Cards[i], d2.Cards[i])
		}
	}
}

func TestDeck_Shuffle_Changes_Order(t *testing.T) {
	d := NewStandardDeck()
	original := make([]Card, len(d.Cards))
	copy(original, d.Cards)

	d.Shuffle(rand.New(rand.NewPCG(99, 0)))

	same := true
	for i := range d.Cards {
		if d.Cards[i] != original[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("shuffle should change the order (extremely unlikely for 162 cards)")
	}
}

func TestDeck_Draw(t *testing.T) {
	d := NewStandardDeck()
	initial := d.Len()

	card, ok := d.Draw()
	if !ok {
		t.Fatal("Draw from non-empty deck should succeed")
	}
	if !card.Value.IsValid() {
		t.Errorf("drawn card has invalid value: %v", card.Value)
	}
	if d.Len() != initial-1 {
		t.Errorf("deck size after draw = %d, want %d", d.Len(), initial-1)
	}
}

func TestDeck_Draw_Empty(t *testing.T) {
	d := &Deck{}
	_, ok := d.Draw()
	if ok {
		t.Error("Draw from empty deck should return false")
	}
}

func TestDeck_DrawN(t *testing.T) {
	d := NewStandardDeck()

	drawn := d.DrawN(5)
	if len(drawn) != 5 {
		t.Errorf("DrawN(5) returned %d cards, want 5", len(drawn))
	}
	if d.Len() != DeckSize-5 {
		t.Errorf("deck size after DrawN(5) = %d, want %d", d.Len(), DeckSize-5)
	}
}

func TestDeck_DrawN_MoreThanAvailable(t *testing.T) {
	d := &Deck{Cards: []Card{NewCard(1), NewCard(2), NewCard(3)}}

	drawn := d.DrawN(10)
	if len(drawn) != 3 {
		t.Errorf("DrawN(10) from 3-card deck returned %d cards, want 3", len(drawn))
	}
	if !d.IsEmpty() {
		t.Error("deck should be empty after drawing all cards")
	}
}

func TestDeck_DrawN_Zero(t *testing.T) {
	d := NewStandardDeck()
	drawn := d.DrawN(0)
	if drawn != nil {
		t.Errorf("DrawN(0) should return nil, got %v", drawn)
	}
	if d.Len() != DeckSize {
		t.Error("DrawN(0) should not change deck size")
	}
}

func TestDeck_AddToBottom(t *testing.T) {
	d := &Deck{Cards: []Card{NewCard(5)}}
	d.AddToBottom([]Card{NewCard(1), NewCard(2)})

	if d.Len() != 3 {
		t.Fatalf("deck size = %d, want 3", d.Len())
	}
	// Bottom should now be the added cards.
	if d.Cards[0].Value != 1 || d.Cards[1].Value != 2 {
		t.Errorf("bottom cards = [%s, %s], want [1, 2]", d.Cards[0], d.Cards[1])
	}
	// Original card should still be on top.
	if d.Cards[2].Value != 5 {
		t.Errorf("top card = %s, want 5", d.Cards[2])
	}
}

func TestDeck_AddToTop(t *testing.T) {
	d := &Deck{Cards: []Card{NewCard(1)}}
	d.AddToTop([]Card{NewCard(10), NewCard(11)})

	if d.Len() != 3 {
		t.Fatalf("deck size = %d, want 3", d.Len())
	}
	// Top should be the last added card.
	top, _ := d.Draw()
	if top.Value != 11 {
		t.Errorf("top card = %s, want 11", top)
	}
}

func TestDeck_IsEmpty(t *testing.T) {
	d := &Deck{}
	if !d.IsEmpty() {
		t.Error("new empty deck should be empty")
	}

	d.Cards = append(d.Cards, NewCard(1))
	if d.IsEmpty() {
		t.Error("deck with one card should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Draw-all-then-verify: exhaust the entire deck and confirm totals.
// ---------------------------------------------------------------------------

func TestDeck_ExhaustAll(t *testing.T) {
	d := NewStandardDeck()
	d.Shuffle(rand.New(rand.NewPCG(1, 0)))

	counts := make(map[CardValue]int)
	for !d.IsEmpty() {
		c, ok := d.Draw()
		if !ok {
			t.Fatal("Draw returned false on non-empty deck")
		}
		counts[c.Value]++
	}

	for v := MinValue; v <= MaxValue; v++ {
		if counts[v] != CopiesPerValue {
			t.Errorf("after exhausting deck: count of %d = %d, want %d", v, counts[v], CopiesPerValue)
		}
	}
	if counts[SkipBo] != WildCount {
		t.Errorf("after exhausting deck: wild count = %d, want %d", counts[SkipBo], WildCount)
	}
}
