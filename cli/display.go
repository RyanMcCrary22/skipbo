// Package cli provides a terminal-based interface for Skip-Bo.
package cli

import (
	"fmt"
	"strings"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// ---------------------------------------------------------------------------
// RenderGameView draws the full game state to a string for terminal display.
// ---------------------------------------------------------------------------

// RenderGameView produces a color-coded, human-readable representation
// of the game state from the current player's perspective.
func RenderGameView(view *engine.GameView) string {
	var b strings.Builder

	// Header.
	b.WriteString(fmt.Sprintf("\n%s═══ SKIP-BO ══ Turn %d ═══%s\n\n",
		colorBold, view.TurnNumber, colorReset))

	// Building piles (shared center).
	b.WriteString(fmt.Sprintf("%s  Building Piles:%s\n", colorBold, colorReset))
	b.WriteString("  ")
	for i := 0; i < engine.MaxBuildingPiles; i++ {
		bp := view.BuildingPiles[i]
		if bp.Size == 0 {
			b.WriteString(fmt.Sprintf("[%s--%s] ", colorDim, colorReset))
		} else {
			b.WriteString(fmt.Sprintf("[%s%2s%s] ",
				colorGreen, bp.TopValue, colorReset))
		}
	}
	b.WriteString(fmt.Sprintf("  %s(draw: %d)%s\n\n",
		colorDim, view.DrawPileSize, colorReset))

	// Opponents.
	if len(view.Opponents) > 0 {
		b.WriteString(fmt.Sprintf("%s  Opponents:%s\n", colorBold, colorReset))
		for _, opp := range view.Opponents {
			stockStr := "--"
			if opp.StockTop != nil {
				stockStr = opp.StockTop.String()
			}
			b.WriteString(fmt.Sprintf("    %s%-10s%s  stock: %s%s%s (%d left)  hand: %d cards  discard tops: ",
				colorCyan, opp.Name, colorReset,
				colorYellow, stockStr, colorReset,
				opp.StockRemain, opp.HandSize))
			for i := 0; i < engine.MaxDiscardPiles; i++ {
				if opp.DiscardTops[i] != nil {
					b.WriteString(fmt.Sprintf("[%s] ", opp.DiscardTops[i]))
				} else {
					b.WriteString(fmt.Sprintf("[%s--%s] ", colorDim, colorReset))
				}
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Separator.
	b.WriteString(fmt.Sprintf("  %s────────────────────────────────%s\n\n", colorDim, colorReset))

	// Current player's stock pile.
	stockStr := "--"
	if view.StockTop != nil {
		stockStr = view.StockTop.String()
	}
	b.WriteString(fmt.Sprintf("  %sYour Stock:%s  %s%s%s  (%d remaining)\n",
		colorBold, colorReset,
		colorYellow+colorBold, stockStr, colorReset,
		view.StockRemain))

	// Current player's hand.
	b.WriteString(fmt.Sprintf("  %sYour Hand:%s   ", colorBold, colorReset))
	for i, card := range view.Hand {
		color := colorWhite
		if card.Value.IsWild() {
			color = colorRed + colorBold
		}
		b.WriteString(fmt.Sprintf("%s%d:%s%s%s  ",
			colorDim, i, colorReset,
			color, card))
	}
	b.WriteString("\n")

	// Current player's discard piles.
	b.WriteString(fmt.Sprintf("  %sYour Discards:%s ", colorBold, colorReset))
	for i := 0; i < engine.MaxDiscardPiles; i++ {
		pile := view.DiscardPiles[i]
		if len(pile) == 0 {
			b.WriteString(fmt.Sprintf("%d:[%s--%s] ", i, colorDim, colorReset))
		} else {
			top := pile[len(pile)-1]
			b.WriteString(fmt.Sprintf("%d:[%s%s%s|%d] ", i,
				colorBlue, top, colorReset, len(pile)))
		}
	}
	b.WriteString("\n\n")

	return b.String()
}

// RenderHelp shows the available commands.
func RenderHelp() string {
	return fmt.Sprintf(`%sCommands:%s
  play hand <idx> build <pile>     Play hand card to building pile
  play stock build <pile>          Play stock top to building pile
  play discard <idx> build <pile>  Play discard top to building pile
  discard <hand_idx> <pile_idx>    Discard from hand to discard pile
  help                             Show this help
  quit                             Quit the game

  Card/pile indices are 0-based.
`, colorBold, colorReset)
}
