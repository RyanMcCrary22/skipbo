package engine

import "fmt"

// ---------------------------------------------------------------------------
// Action sources — where a card comes from
// ---------------------------------------------------------------------------

// Source identifies where a card is being played from.
type Source int

const (
	SourceHand    Source = iota // From the player's hand (by index).
	SourceStock                // From the top of the player's stock pile.
	SourceDiscard              // From the top of one of the player's discard piles (by index).
)

// String returns a human-readable representation of the source.
func (s Source) String() string {
	switch s {
	case SourceHand:
		return "hand"
	case SourceStock:
		return "stock"
	case SourceDiscard:
		return "discard"
	default:
		return fmt.Sprintf("Source(%d)", s)
	}
}

// ---------------------------------------------------------------------------
// Action targets — where a card goes
// ---------------------------------------------------------------------------

// Target identifies where a card is being played to.
type Target int

const (
	TargetBuild   Target = iota // Onto one of the 4 shared building piles (by index).
	TargetDiscard               // Onto one of the player's 4 discard piles (by index, ends turn).
)

// String returns a human-readable representation of the target.
func (t Target) String() string {
	switch t {
	case TargetBuild:
		return "build"
	case TargetDiscard:
		return "discard"
	default:
		return fmt.Sprintf("Target(%d)", t)
	}
}

// ---------------------------------------------------------------------------
// Action
// ---------------------------------------------------------------------------

// Action describes a single play within a turn.
//
// Examples:
//
//	Play hand card index 2 onto building pile 0:
//	  Action{Type: SourceHand, SourceIndex: 2, Target: TargetBuild, TargetIndex: 0}
//
//	Play stock pile top onto building pile 1:
//	  Action{Type: SourceStock, Target: TargetBuild, TargetIndex: 1}
//
//	Play discard pile 3 top onto building pile 2:
//	  Action{Type: SourceDiscard, SourceIndex: 3, Target: TargetBuild, TargetIndex: 2}
//
//	Discard hand card index 0 to discard pile 1 (ends turn):
//	  Action{Type: SourceHand, SourceIndex: 0, Target: TargetDiscard, TargetIndex: 1}
type Action struct {
	Source      Source // Where the card comes from.
	SourceIndex int   // Index within the source (hand card index, or discard pile index).
	Target      Target // Where the card goes.
	TargetIndex int    // Index of the target pile (building pile 0–3 or discard pile 0–3).
}

// IsDiscard reports whether this action is a discard (ends the turn).
func (a Action) IsDiscard() bool {
	return a.Target == TargetDiscard
}

// String returns a human-readable representation of the action.
func (a Action) String() string {
	src := a.Source.String()
	if a.Source == SourceHand || a.Source == SourceDiscard {
		src = fmt.Sprintf("%s[%d]", src, a.SourceIndex)
	}
	return fmt.Sprintf("%s → %s[%d]", src, a.Target, a.TargetIndex)
}

// ---------------------------------------------------------------------------
// Convenience constructors for common actions
// ---------------------------------------------------------------------------

// PlayFromHand creates an action to play a hand card onto a building pile.
func PlayFromHand(handIndex, buildPileIndex int) Action {
	return Action{
		Source:      SourceHand,
		SourceIndex: handIndex,
		Target:      TargetBuild,
		TargetIndex: buildPileIndex,
	}
}

// PlayFromStock creates an action to play the stock pile top onto a building pile.
func PlayFromStock(buildPileIndex int) Action {
	return Action{
		Source:      SourceStock,
		Target:      TargetBuild,
		TargetIndex: buildPileIndex,
	}
}

// PlayFromDiscard creates an action to play a discard pile top onto a building pile.
func PlayFromDiscard(discardIndex, buildPileIndex int) Action {
	return Action{
		Source:      SourceDiscard,
		SourceIndex: discardIndex,
		Target:      TargetBuild,
		TargetIndex: buildPileIndex,
	}
}

// DiscardFromHand creates an action to discard a hand card to a discard pile.
func DiscardFromHand(handIndex, discardPileIndex int) Action {
	return Action{
		Source:      SourceHand,
		SourceIndex: handIndex,
		Target:      TargetDiscard,
		TargetIndex: discardPileIndex,
	}
}
