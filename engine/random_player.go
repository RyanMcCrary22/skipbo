package engine

import "math/rand/v2"

// RandomPlayer is a simple AI that picks random valid actions.
// It serves as a baseline opponent, a testing tool, and a demonstration
// of how to implement the Player interface.
type RandomPlayer struct {
	name string
	rng  *rand.Rand
}

// NewRandomPlayer creates a random-playing AI with the given name and seed.
func NewRandomPlayer(name string, seed uint64) *RandomPlayer {
	return &RandomPlayer{
		name: name,
		rng:  rand.New(rand.NewPCG(seed, 0)),
	}
}

// Name returns the player's display name.
func (p *RandomPlayer) Name() string { return p.name }

// ChooseAction examines the visible game state and picks a valid action.
// It prioritizes stock pile plays (the only path to winning), then hand
// plays, then discard plays, and discards when no plays are available.
func (p *RandomPlayer) ChooseAction(view *GameView) (Action, error) {
	// Collect all valid play actions.
	var plays []Action

	// Try playing from stock to each building pile (highest priority).
	if view.StockTop != nil {
		for i := 0; i < MaxBuildingPiles; i++ {
			if view.StockTop.CanPlayOn(view.BuildingPiles[i].NextNeeded) {
				plays = append(plays, PlayFromStock(i))
			}
		}
	}

	// If stock plays exist, always prefer them.
	if len(plays) > 0 {
		return plays[p.rng.IntN(len(plays))], nil
	}

	// Try playing from hand to each building pile.
	for hi, card := range view.Hand {
		for bi := 0; bi < MaxBuildingPiles; bi++ {
			if card.CanPlayOn(view.BuildingPiles[bi].NextNeeded) {
				plays = append(plays, PlayFromHand(hi, bi))
			}
		}
	}

	// Try playing from discard tops to each building pile.
	for di := 0; di < MaxDiscardPiles; di++ {
		pile := view.DiscardPiles[di]
		if len(pile) == 0 {
			continue
		}
		top := pile[len(pile)-1]
		for bi := 0; bi < MaxBuildingPiles; bi++ {
			if top.CanPlayOn(view.BuildingPiles[bi].NextNeeded) {
				plays = append(plays, PlayFromDiscard(di, bi))
			}
		}
	}

	// If we have valid plays, pick one at random.
	if len(plays) > 0 {
		return plays[p.rng.IntN(len(plays))], nil
	}

	// No valid plays — must discard from hand.
	if len(view.Hand) > 0 {
		handIdx := p.rng.IntN(len(view.Hand))
		discIdx := p.rng.IntN(MaxDiscardPiles)
		return DiscardFromHand(handIdx, discIdx), nil
	}

	// Fallback (should not be reachable with correct game logic).
	return DiscardFromHand(0, 0), nil
}
