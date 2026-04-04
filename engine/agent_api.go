package engine

// ---------------------------------------------------------------------------
// Agent Plugin API
//
// This file provides the bridge between the Skip-Bo game engine and external
// AI agents (Python RL, gRPC bots, etc.). It converts the engine's GameView
// into flat numeric representations suitable for neural networks, and maps
// between discrete action indices and Action structs.
//
// Design Principles:
//   - The engine generates the state vector from a GameView, enforcing
//     information boundaries — agents can never see hidden cards.
//   - The action space is a fixed-size discrete set of 60 possible moves.
//   - An action mask indicates which of the 60 actions are currently legal.
// ---------------------------------------------------------------------------

const (
	// HandSlots is the maximum number of cards in a hand.
	HandSlots = MaxHandSize // 5

	// TotalActions is the size of the discrete action space.
	//   Hand→Build:    5×4 = 20
	//   Stock→Build:   1×4 =  4
	//   Discard→Build: 4×4 = 16
	//   Hand→Discard:  5×4 = 20
	//   Total:              = 60
	TotalActions = HandSlots*MaxBuildingPiles + // hand→build   (0–19)
		MaxBuildingPiles + // stock→build  (20–23)
		MaxDiscardPiles*MaxBuildingPiles + // discard→build (24–39)
		HandSlots*MaxDiscardPiles // hand→discard (40–59)

	// StateVectorSize is the length of the flat state vector.
	//   Hand:               5
	//   Stock top:          1
	//   Stock remaining:    1
	//   Own discard tops:   4
	//   Own discard depths: 4
	//   Building pile tops: 4
	//   Opponents (×5 max): 5 × (1 stock top + 1 stock remain + 4 discard tops + 1 hand size) = 35
	//   Total:             54
	StateVectorSize = HandSlots + 1 + 1 + MaxDiscardPiles + MaxDiscardPiles +
		MaxBuildingPiles + 5*7 // 5 + 1 + 1 + 4 + 4 + 4 + 35 = 54

	// maxOpponents is the maximum number of opponents encoded in the vector.
	maxOpponents = 5
)

// Action index layout offsets.
const (
	offHandBuild    = 0                             // [0, 20)
	offStockBuild   = HandSlots * MaxBuildingPiles   // [20, 24)
	offDiscardBuild = offStockBuild + MaxBuildingPiles // [24, 40)
	offHandDiscard  = offDiscardBuild + MaxDiscardPiles*MaxBuildingPiles // [40, 60)
)

// ---------------------------------------------------------------------------
// State Vector
// ---------------------------------------------------------------------------

// cardToFloat converts a CardValue to a float64 for the state vector.
// 0 = empty/none, 1–12 = numbered, 13 = SkipBo wild.
func cardToFloat(v CardValue) float64 {
	if v == SkipBo {
		return 13.0
	}
	return float64(v)
}

// StateVector converts a GameView into a flat numeric vector suitable for
// neural network input. The vector encodes only visible information,
// enforcing the information boundary inherent to Skip-Bo.
//
// The vector layout is fixed-size (StateVectorSize = 54) regardless of the
// number of players; unused opponent slots are zero-padded.
func StateVector(view *GameView) []float64 {
	v := make([]float64, StateVectorSize)
	idx := 0

	// Hand (5 slots, 0-padded if fewer cards).
	for i := 0; i < HandSlots; i++ {
		if i < len(view.Hand) {
			v[idx] = cardToFloat(view.Hand[i].Value)
		}
		idx++
	}

	// Stock top card.
	if view.StockTop != nil {
		v[idx] = cardToFloat(view.StockTop.Value)
	}
	idx++

	// Stock remaining count.
	v[idx] = float64(view.StockRemain)
	idx++

	// Own discard pile tops (4 piles).
	for i := 0; i < MaxDiscardPiles; i++ {
		pile := view.DiscardPiles[i]
		if len(pile) > 0 {
			v[idx] = cardToFloat(pile[len(pile)-1].Value)
		}
		idx++
	}

	// Own discard pile depths (4 piles).
	for i := 0; i < MaxDiscardPiles; i++ {
		v[idx] = float64(len(view.DiscardPiles[i]))
		idx++
	}

	// Building pile tops (4 piles).
	for i := 0; i < MaxBuildingPiles; i++ {
		v[idx] = float64(view.BuildingPiles[i].TopValue)
		idx++
	}

	// Opponents (up to 5, 7 values each).
	for i := 0; i < maxOpponents; i++ {
		if i < len(view.Opponents) {
			opp := view.Opponents[i]
			if opp.StockTop != nil {
				v[idx] = cardToFloat(opp.StockTop.Value)
			}
			idx++
			v[idx] = float64(opp.StockRemain)
			idx++
			for j := 0; j < MaxDiscardPiles; j++ {
				if opp.DiscardTops[j] != nil {
					v[idx] = cardToFloat(opp.DiscardTops[j].Value)
				}
				idx++
			}
			v[idx] = float64(opp.HandSize)
			idx++
		} else {
			// Zero-pad unused opponent slots.
			idx += 7
		}
	}

	return v
}

// ---------------------------------------------------------------------------
// Action Index Mapping
// ---------------------------------------------------------------------------

// ActionFromIndex converts a discrete action index (0–59) into an Action struct.
// Returns a zero Action and false if the index is out of range.
func ActionFromIndex(index int) (Action, bool) {
	if index < 0 || index >= TotalActions {
		return Action{}, false
	}

	switch {
	case index < offStockBuild:
		// Hand → Build: index = handSlot*4 + buildPile
		handSlot := (index - offHandBuild) / MaxBuildingPiles
		buildPile := (index - offHandBuild) % MaxBuildingPiles
		return PlayFromHand(handSlot, buildPile), true

	case index < offDiscardBuild:
		// Stock → Build: index - 20 = buildPile
		buildPile := index - offStockBuild
		return PlayFromStock(buildPile), true

	case index < offHandDiscard:
		// Discard → Build: (index-24) = discardPile*4 + buildPile
		rel := index - offDiscardBuild
		discardPile := rel / MaxBuildingPiles
		buildPile := rel % MaxBuildingPiles
		return PlayFromDiscard(discardPile, buildPile), true

	default:
		// Hand → Discard: (index-40) = handSlot*4 + discardPile
		rel := index - offHandDiscard
		handSlot := rel / MaxDiscardPiles
		discardPile := rel % MaxDiscardPiles
		return DiscardFromHand(handSlot, discardPile), true
	}
}

// ActionToIndex converts an Action to its discrete action index (0–59).
// Returns -1 if the action does not map to any known index.
func ActionToIndex(a Action) int {
	switch {
	case a.Source == SourceHand && a.Target == TargetBuild:
		return offHandBuild + a.SourceIndex*MaxBuildingPiles + a.TargetIndex

	case a.Source == SourceStock && a.Target == TargetBuild:
		return offStockBuild + a.TargetIndex

	case a.Source == SourceDiscard && a.Target == TargetBuild:
		return offDiscardBuild + a.SourceIndex*MaxBuildingPiles + a.TargetIndex

	case a.Source == SourceHand && a.Target == TargetDiscard:
		return offHandDiscard + a.SourceIndex*MaxDiscardPiles + a.TargetIndex

	default:
		return -1
	}
}

// ---------------------------------------------------------------------------
// Action Mask
// ---------------------------------------------------------------------------

// ActionMask returns a boolean array of length TotalActions indicating which
// actions are legal for the current player given the GameView. The mask is
// suitable for masking softmax outputs in PPO/DQN agents.
func ActionMask(view *GameView) [TotalActions]bool {
	var mask [TotalActions]bool

	// Hand → Build (indices 0–19).
	for hi := 0; hi < HandSlots; hi++ {
		if hi >= len(view.Hand) {
			continue
		}
		card := view.Hand[hi]
		for bi := 0; bi < MaxBuildingPiles; bi++ {
			if card.CanPlayOn(view.BuildingPiles[bi].NextNeeded) {
				mask[offHandBuild+hi*MaxBuildingPiles+bi] = true
			}
		}
	}

	// Stock → Build (indices 20–23).
	if view.StockTop != nil {
		for bi := 0; bi < MaxBuildingPiles; bi++ {
			if view.StockTop.CanPlayOn(view.BuildingPiles[bi].NextNeeded) {
				mask[offStockBuild+bi] = true
			}
		}
	}

	// Discard → Build (indices 24–39).
	for di := 0; di < MaxDiscardPiles; di++ {
		pile := view.DiscardPiles[di]
		if len(pile) == 0 {
			continue
		}
		top := pile[len(pile)-1]
		for bi := 0; bi < MaxBuildingPiles; bi++ {
			if top.CanPlayOn(view.BuildingPiles[bi].NextNeeded) {
				mask[offDiscardBuild+di*MaxBuildingPiles+bi] = true
			}
		}
	}

	// Hand → Discard (indices 40–59).
	for hi := 0; hi < HandSlots; hi++ {
		if hi >= len(view.Hand) {
			continue
		}
		for di := 0; di < MaxDiscardPiles; di++ {
			mask[offHandDiscard+hi*MaxDiscardPiles+di] = true
		}
	}

	return mask
}
