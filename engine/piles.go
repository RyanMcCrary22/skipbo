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
	ErrPileEmpty      = errors.New("pile is empty")
	ErrIllegalPlay    = errors.New("illegal play on building pile")
	ErrHandFull       = errors.New("hand is full")
	ErrCardNotInHand  = errors.New("card not found in hand")
	ErrMaxDiscardPile = errors.New("maximum discard piles reached")
)

// ---------------------------------------------------------------------------
// StockPile
// ---------------------------------------------------------------------------

// StockPile is a player's stock pile — an ordered stack with only the top
// card visible. The game is won when a player empties their stock pile.
type StockPile struct {
	cards []Card // index 0 = bottom, last = top (face-up)
}

// NewStockPile creates a stock pile from the given cards.
// The last card in the slice becomes the face-up top card.
func NewStockPile(cards []Card) *StockPile {
	owned := make([]Card, len(cards))
	copy(owned, cards)
	return &StockPile{cards: owned}
}

// Top returns the face-up top card without removing it.
func (s *StockPile) Top() (Card, bool) {
	if len(s.cards) == 0 {
		return Card{}, false
	}
	return s.cards[len(s.cards)-1], true
}

// Pop removes and returns the top card.
func (s *StockPile) Pop() (Card, bool) {
	if len(s.cards) == 0 {
		return Card{}, false
	}
	top := s.cards[len(s.cards)-1]
	s.cards = s.cards[:len(s.cards)-1]
	return top, true
}

// Len returns the number of cards remaining.
func (s *StockPile) Len() int { return len(s.cards) }

// IsEmpty reports whether the stock pile is empty (win condition).
func (s *StockPile) IsEmpty() bool { return len(s.cards) == 0 }

// ---------------------------------------------------------------------------
// BuildingPile
// ---------------------------------------------------------------------------

// BuildingPile is one of the 4 shared piles in the center. Cards must be
// played in ascending order 1→12. When 12 is reached, the pile is complete
// and should be cleared (cards recycled into the draw pile).
type BuildingPile struct {
	cards []Card
}

// NextNeeded returns the CardValue required to play on this pile.
// An empty pile needs a 1; a pile with top card N needs N+1.
func (b *BuildingPile) NextNeeded() CardValue {
	if len(b.cards) == 0 {
		return MinValue
	}
	return CardValue(len(b.cards)) + 1
}

// Play attempts to place a card on this building pile.
// Returns an error if the card cannot legally be played.
func (b *BuildingPile) Play(c Card) error {
	needed := b.NextNeeded()
	if !c.CanPlayOn(needed) {
		return fmt.Errorf("%w: pile needs %s, got %s", ErrIllegalPlay, needed, c)
	}
	b.cards = append(b.cards, c)
	return nil
}

// IsComplete reports whether the pile has reached 12 and should be cleared.
func (b *BuildingPile) IsComplete() bool {
	return len(b.cards) == int(MaxValue)
}

// Clear removes and returns all cards from the pile (for recycling into
// the draw pile). Returns nil if the pile is empty.
func (b *BuildingPile) Clear() []Card {
	if len(b.cards) == 0 {
		return nil
	}
	cleared := b.cards
	b.cards = nil
	return cleared
}

// Len returns the number of cards in the pile.
func (b *BuildingPile) Len() int { return len(b.cards) }

// TopValue returns the effective value of the top card, or 0 if empty.
// This is the sequence position (1–12), not the raw card value
// (which could be SkipBo for wilds).
func (b *BuildingPile) TopValue() CardValue {
	return CardValue(len(b.cards))
}

// ---------------------------------------------------------------------------
// DiscardPile
// ---------------------------------------------------------------------------

// DiscardPile is one of a player's personal discard piles.
// Any card can be placed on any discard pile. Only the top card is playable.
type DiscardPile struct {
	cards []Card
}

// Push places a card on top of this discard pile.
func (d *DiscardPile) Push(c Card) {
	d.cards = append(d.cards, c)
}

// Top returns the top card without removing it.
func (d *DiscardPile) Top() (Card, bool) {
	if len(d.cards) == 0 {
		return Card{}, false
	}
	return d.cards[len(d.cards)-1], true
}

// Pop removes and returns the top card.
func (d *DiscardPile) Pop() (Card, bool) {
	if len(d.cards) == 0 {
		return Card{}, false
	}
	top := d.cards[len(d.cards)-1]
	d.cards = d.cards[:len(d.cards)-1]
	return top, true
}

// Len returns the number of cards.
func (d *DiscardPile) Len() int { return len(d.cards) }

// IsEmpty reports whether the pile is empty.
func (d *DiscardPile) IsEmpty() bool { return len(d.cards) == 0 }

// Cards returns a copy of the pile contents for display purposes.
func (d *DiscardPile) Cards() []Card {
	out := make([]Card, len(d.cards))
	copy(out, d.cards)
	return out
}

// ---------------------------------------------------------------------------
// DrawPile
// ---------------------------------------------------------------------------

// DrawPile is the shared draw pile in the center. When exhausted, it can
// be replenished from completed building pile cards.
type DrawPile struct {
	deck *Deck
}

// NewDrawPile wraps an existing deck as the draw pile.
func NewDrawPile(deck *Deck) *DrawPile {
	return &DrawPile{deck: deck}
}

// Draw removes and returns the top card. Returns false if empty.
func (dp *DrawPile) Draw() (Card, bool) {
	return dp.deck.Draw()
}

// DrawN removes and returns up to n cards from the top.
func (dp *DrawPile) DrawN(n int) []Card {
	return dp.deck.DrawN(n)
}

// Len returns the number of cards remaining.
func (dp *DrawPile) Len() int { return dp.deck.Len() }

// IsEmpty reports whether the draw pile is exhausted.
func (dp *DrawPile) IsEmpty() bool { return dp.deck.IsEmpty() }

// Replenish adds cards to the bottom and reshuffles the entire pile.
// Called when building piles complete and their cards are recycled.
func (dp *DrawPile) Replenish(cards []Card, rng *rand.Rand) {
	dp.deck.AddToBottom(cards)
	dp.deck.Shuffle(rng)
}

// ---------------------------------------------------------------------------
// Hand
// ---------------------------------------------------------------------------

// MaxHandSize is the maximum number of cards a player can hold.
const MaxHandSize = 5

// Hand represents a player's hand of up to 5 cards.
type Hand struct {
	cards []Card
}

// DrawFrom fills the hand up to MaxHandSize from the given draw pile.
// Returns the number of cards drawn.
func (h *Hand) DrawFrom(dp *DrawPile) int {
	need := MaxHandSize - len(h.cards)
	if need <= 0 {
		return 0
	}
	drawn := dp.DrawN(need)
	h.cards = append(h.cards, drawn...)
	return len(drawn)
}

// Play removes the card at the given index from the hand and returns it.
// Returns an error if the index is out of range.
func (h *Hand) Play(index int) (Card, error) {
	if index < 0 || index >= len(h.cards) {
		return Card{}, fmt.Errorf("%w: index %d, hand size %d", ErrCardNotInHand, index, len(h.cards))
	}
	card := h.cards[index]
	// Remove by swapping with last (order within hand doesn't matter for gameplay).
	h.cards[index] = h.cards[len(h.cards)-1]
	h.cards = h.cards[:len(h.cards)-1]
	return card, nil
}

// Discard removes the card at the given index from the hand.
// This is an alias for Play but reads better at the call site when
// the intent is to discard rather than play on a building pile.
func (h *Hand) Discard(index int) (Card, error) {
	return h.Play(index)
}

// Len returns the number of cards in hand.
func (h *Hand) Len() int { return len(h.cards) }

// IsEmpty reports whether the hand is empty (triggers draw-5-more).
func (h *Hand) IsEmpty() bool { return len(h.cards) == 0 }

// Cards returns a copy of the hand contents for display/AI purposes.
func (h *Hand) Cards() []Card {
	out := make([]Card, len(h.cards))
	copy(out, h.cards)
	return out
}

// Get returns the card at the given index without removing it.
func (h *Hand) Get(index int) (Card, error) {
	if index < 0 || index >= len(h.cards) {
		return Card{}, fmt.Errorf("%w: index %d, hand size %d", ErrCardNotInHand, index, len(h.cards))
	}
	return h.cards[index], nil
}

// ---------------------------------------------------------------------------
// DiscardArea — a player's set of 4 discard piles
// ---------------------------------------------------------------------------

// MaxDiscardPiles is the number of discard piles each player has.
const MaxDiscardPiles = 4

// DiscardArea groups a player's 4 discard piles for convenience.
type DiscardArea struct {
	Piles [MaxDiscardPiles]DiscardPile
}

// Get returns a pointer to the discard pile at the given index (0–3).
func (da *DiscardArea) Get(index int) (*DiscardPile, error) {
	if index < 0 || index >= MaxDiscardPiles {
		return nil, fmt.Errorf("%w: index %d", ErrMaxDiscardPile, index)
	}
	return &da.Piles[index], nil
}
