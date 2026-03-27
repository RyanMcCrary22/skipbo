// Package engine implements the core Skip-Bo game logic.
// It is intentionally free of I/O concerns so that it can be driven
// by any frontend (CLI, GUI, gRPC agent, tests, etc.).
package engine

import (
	"fmt"
	"math/rand/v2"
)

// ---------------------------------------------------------------------------
// Card
// ---------------------------------------------------------------------------

// CardValue represents the face value of a card (1–12) or the Skip-Bo wild.
type CardValue int

const (
	SkipBo CardValue = 0 // Wild — can represent any value 1–12.
)

// MinValue and MaxValue bound the numbered card range.
const (
	MinValue CardValue = 1
	MaxValue CardValue = 12
)

// IsWild reports whether the card is a Skip-Bo wild card.
func (v CardValue) IsWild() bool { return v == SkipBo }

// IsValid reports whether v is a legal card value (wild or 1–12).
func (v CardValue) IsValid() bool {
	return v == SkipBo || (v >= MinValue && v <= MaxValue)
}

// String returns a human-readable representation, e.g. "1", "12", "SB".
func (v CardValue) String() string {
	if v == SkipBo {
		return "SB"
	}
	return fmt.Sprintf("%d", int(v))
}

// Card is a single Skip-Bo card. It is intentionally a value type (small,
// cheaply copied) so that slices of cards are easy to work with.
type Card struct {
	Value CardValue
}

// NewCard creates a card with the given value. It panics if v is invalid
// so that programming errors surface loudly during development.
func NewCard(v CardValue) Card {
	if !v.IsValid() {
		panic(fmt.Sprintf("engine: invalid card value %d", v))
	}
	return Card{Value: v}
}

// String delegates to the underlying CardValue.
func (c Card) String() string { return c.Value.String() }

// CanPlayOn reports whether this card can legally be placed on a building
// pile whose current top value is topValue. A building pile expecting
// nextValue accepts a numbered card equal to nextValue or any wild card.
func (c Card) CanPlayOn(nextNeeded CardValue) bool {
	return c.Value.IsWild() || c.Value == nextNeeded
}

// ---------------------------------------------------------------------------
// Deck constants
// ---------------------------------------------------------------------------

const (
	// CopiesPerValue is the number of copies of each numbered card (1–12).
	CopiesPerValue = 12

	// WildCount is the number of Skip-Bo wild cards in a standard deck.
	WildCount = 18

	// DeckSize is the total number of cards: 12 values × 12 copies + 18 wilds.
	DeckSize = int(MaxValue)*CopiesPerValue + WildCount // 162
)

// ---------------------------------------------------------------------------
// Deck
// ---------------------------------------------------------------------------

// Deck holds an ordered collection of cards. The zero value is usable but
// empty; use NewStandardDeck to create a full 162-card deck.
type Deck struct {
	Cards []Card
}

// NewStandardDeck builds a standard 162-card Skip-Bo deck:
//   - 12 copies each of cards 1 through 12  (144 numbered)
//   - 18 Skip-Bo wild cards
//
// The deck is returned in a deterministic, unshuffled order so that tests
// can make assertions about the raw deck. Call Shuffle to randomize.
func NewStandardDeck() *Deck {
	cards := make([]Card, 0, DeckSize)

	// Numbered cards: 12 copies × 12 values.
	for v := MinValue; v <= MaxValue; v++ {
		for i := 0; i < CopiesPerValue; i++ {
			cards = append(cards, NewCard(v))
		}
	}

	// Wild cards.
	for i := 0; i < WildCount; i++ {
		cards = append(cards, NewCard(SkipBo))
	}

	return &Deck{Cards: cards}
}

// Len returns the number of cards remaining in the deck.
func (d *Deck) Len() int { return len(d.Cards) }

// IsEmpty reports whether the deck has no cards.
func (d *Deck) IsEmpty() bool { return len(d.Cards) == 0 }

// Shuffle randomizes the deck order using the provided RNG source.
// Accepting an *rand.Rand makes games deterministic when seeded,
// which is essential for reproducible tests and training runs.
func (d *Deck) Shuffle(rng *rand.Rand) {
	rng.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Draw removes and returns the top card (last element for O(1) pop).
// Returns the card and true, or a zero Card and false if the deck is empty.
func (d *Deck) Draw() (Card, bool) {
	if d.IsEmpty() {
		return Card{}, false
	}
	top := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return top, true
}

// DrawN removes and returns up to n cards from the top of the deck.
// If fewer than n cards remain, all remaining cards are returned.
func (d *Deck) DrawN(n int) []Card {
	if n <= 0 {
		return nil
	}
	if n > len(d.Cards) {
		n = len(d.Cards)
	}
	// Take from the end (top of deck).
	start := len(d.Cards) - n
	drawn := make([]Card, n)
	copy(drawn, d.Cards[start:])
	d.Cards = d.Cards[:start]
	return drawn
}

// AddToBottom places the given cards at the bottom of the deck.
// Used when reshuffling completed building piles back into the draw pile.
func (d *Deck) AddToBottom(cards []Card) {
	d.Cards = append(cards, d.Cards...)
}

// AddToTop places the given cards on top of the deck.
func (d *Deck) AddToTop(cards []Card) {
	d.Cards = append(d.Cards, cards...)
}
