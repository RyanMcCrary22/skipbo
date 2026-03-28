package gui

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// Selection state machine
// ---------------------------------------------------------------------------

// SelectionPhase tracks what the player is choosing.
type SelectionPhase int

const (
	PhaseSelectSource SelectionPhase = iota // Waiting for source click.
	PhaseSelectTarget                       // Source selected, waiting for target.
)

// SelectedSource records what was clicked as the card source.
type SelectedSource struct {
	Source engine.Source
	Index  int
}

// ---------------------------------------------------------------------------
// GameScreen — the main Ebitengine game
// ---------------------------------------------------------------------------

// GameScreen implements ebiten.Game and drives the GUI.
type GameScreen struct {
	fontSrc   *text.GoTextFaceSource

	// Channel-based communication with the engine goroutine.
	viewCh   chan *engine.GameView
	actionCh chan engine.Action

	// Current state.
	currentView  *engine.GameView
	phase        SelectionPhase
	selected     *SelectedSource
	wasMouseDown bool

	// Status / error message.
	statusMsg   string
	statusColor color.Color

	// Game over state.
	gameOverMsg string

	// Animation state for visualizing moves.
	lastAction       *engine.Action
	lastActionPlayer int
	lastActionTime   time.Time
}

// NewGameScreen creates a new GUI game screen.
func NewGameScreen() *GameScreen {
	// Load embedded font.
	fontSrc, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		log.Printf("Warning: could not load font, using fallback: %v", err)
	}

	gs := &GameScreen{
		fontSrc:     fontSrc,
		viewCh:      make(chan *engine.GameView, 1),
		actionCh:    make(chan engine.Action, 1),
		statusMsg:   "Your turn — click a card to play",
		statusColor: colorStatusText,
	}

	return gs
}

// ---------------------------------------------------------------------------
// Layout positions — computed from the screen center
// ---------------------------------------------------------------------------

// Positions for the various card areas.
func buildPileX(i int) float32 {
	startX := float32(ScreenWidth)/2 - float32(engine.MaxBuildingPiles)*(CardWidth+CardSpacing)/2
	return startX + float32(i)*(CardWidth+CardSpacing)
}

const buildPileY = 40

func handCardX(i int, total int) float32 {
	totalWidth := float32(total)*(CardWidth+CardSpacing) - CardSpacing
	startX := float32(ScreenWidth)/2 - totalWidth/2
	return startX + float32(i)*(CardWidth+CardSpacing)
}

const handCardY = 530

const stockX = 40
const stockY = 310

func discardX(i int) float32 {
	return float32(ScreenWidth) - 40 - float32(engine.MaxDiscardPiles-i)*(CardWidth+CardSpacing)
}

const discardY = 310

// Opponent area.
func opponentX(i int, total int) float32 {
	spacing := float32(200)
	totalWidth := float32(total) * spacing
	startX := float32(ScreenWidth)/2 - totalWidth/2
	return startX + float32(i)*spacing
}

const opponentY = 160

// ---------------------------------------------------------------------------
// Ebitengine interface: Update
// ---------------------------------------------------------------------------

func (gs *GameScreen) Update() error {
	// Check for new view from game goroutine.
	select {
	case view := <-gs.viewCh:
		gs.currentView = view
		gs.phase = PhaseSelectSource
		gs.selected = nil
		gs.statusMsg = "Your turn — click a card to play"
		gs.statusColor = colorStatusText
	default:
	}

	if gs.currentView == nil || gs.gameOverMsg != "" {
		return nil
	}

	// Handle mouse clicks.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		gs.wasMouseDown = true
		return nil // Wait for release.
	}

	// Detect click release (simple approach).
	mx, my := ebiten.CursorPosition()
	if !gs.wasMouseDown {
		return nil
	}
	gs.wasMouseDown = false

	fmx, fmy := float32(mx), float32(my)

	switch gs.phase {
	case PhaseSelectSource:
		gs.handleSourceClick(fmx, fmy)
	case PhaseSelectTarget:
		gs.handleTargetClick(fmx, fmy)
	}

	return nil
}

func (gs *GameScreen) handleSourceClick(mx, my float32) {
	view := gs.currentView

	// Check hand cards.
	for i := range view.Hand {
		cx := handCardX(i, len(view.Hand))
		if CardHitTest(mx, my, cx, handCardY) {
			gs.selected = &SelectedSource{Source: engine.SourceHand, Index: i}
			gs.phase = PhaseSelectTarget
			gs.statusMsg = fmt.Sprintf("Selected hand card %s — click a pile to play/discard", view.Hand[i])
			gs.statusColor = colorHighlight
			return
		}
	}

	// Check stock pile.
	if view.StockTop != nil && CardHitTest(mx, my, stockX, stockY) {
		gs.selected = &SelectedSource{Source: engine.SourceStock, Index: 0}
		gs.phase = PhaseSelectTarget
		gs.statusMsg = fmt.Sprintf("Selected stock (%s) — click a building pile", view.StockTop)
		gs.statusColor = colorStock
		return
	}

	// Check discard pile tops.
	for i := 0; i < engine.MaxDiscardPiles; i++ {
		dx := discardX(i)
		if len(view.DiscardPiles[i]) > 0 && CardHitTest(mx, my, dx, discardY) {
			gs.selected = &SelectedSource{Source: engine.SourceDiscard, Index: i}
			gs.phase = PhaseSelectTarget
			top := view.DiscardPiles[i][len(view.DiscardPiles[i])-1]
			gs.statusMsg = fmt.Sprintf("Selected discard %d (%s) — click a building pile", i, top)
			gs.statusColor = colorDiscard
			return
		}
	}
}

func (gs *GameScreen) handleTargetClick(mx, my float32) {
	if gs.selected == nil {
		gs.phase = PhaseSelectSource
		return
	}

	// Check building piles.
	for i := 0; i < engine.MaxBuildingPiles; i++ {
		bx := buildPileX(i)
		if CardHitTest(mx, my, bx, buildPileY) {
			var action engine.Action
			switch gs.selected.Source {
			case engine.SourceHand:
				action = engine.PlayFromHand(gs.selected.Index, i)
			case engine.SourceStock:
				action = engine.PlayFromStock(i)
			case engine.SourceDiscard:
				action = engine.PlayFromDiscard(gs.selected.Index, i)
			}
			gs.sendAction(action)
			return
		}
	}

	// Check discard piles (only valid target for hand cards = discard action).
	if gs.selected.Source == engine.SourceHand {
		for i := 0; i < engine.MaxDiscardPiles; i++ {
			dx := discardX(i)
			if CardHitTest(mx, my, dx, discardY) {
				action := engine.DiscardFromHand(gs.selected.Index, i)
				gs.sendAction(action)
				return
			}
		}
	}

	// Clicked elsewhere — cancel selection.
	gs.phase = PhaseSelectSource
	gs.selected = nil
	gs.statusMsg = "Selection cancelled — click a card to play"
	gs.statusColor = colorStatusText
}

func (gs *GameScreen) sendAction(action engine.Action) {
	gs.actionCh <- action
	gs.currentView = nil
	gs.phase = PhaseSelectSource
	gs.selected = nil
	gs.statusMsg = "Waiting..."
	gs.statusColor = colorDimText
}

// SetError sets the error message (called from the game event handler).
func (gs *GameScreen) SetError(msg string) {
	gs.statusMsg = msg
	gs.statusColor = colorError
}

// SetGameOver sets the game over message.
func (gs *GameScreen) SetGameOver(msg string) {
	gs.gameOverMsg = msg
}

func (gs *GameScreen) SetLastAction(a *engine.Action, playerIdx int) {
	if a == nil {
		return
	}
	gs.lastAction = a
	gs.lastActionPlayer = playerIdx
	gs.lastActionTime = time.Now()
}

// ---------------------------------------------------------------------------
// Ebitengine interface: Draw
// ---------------------------------------------------------------------------

func (gs *GameScreen) Draw(screen *ebiten.Image) {
	// Felt background.
	screen.Fill(colorFelt)

	if gs.gameOverMsg != "" {
		DrawLabelCentered(screen, ScreenWidth/2, ScreenHeight/2, gs.gameOverMsg, 36, colorStatusText, gs.fontSrc)
		return
	}

	view := gs.currentView
	if view == nil {
		DrawLabelCentered(screen, ScreenWidth/2, ScreenHeight/2, "Waiting for your turn...", 20, colorDimText, gs.fontSrc)
		return
	}

	gs.drawBuildingPiles(screen, view)
	gs.drawOpponents(screen, view)
	gs.drawStock(screen, view)
	gs.drawHand(screen, view)
	gs.drawDiscards(screen, view)
	gs.drawArrow(screen, view)
	gs.drawStatus(screen)
}

func (gs *GameScreen) drawBuildingPiles(screen *ebiten.Image, view *engine.GameView) {
	DrawLabel(screen, float64(buildPileX(0)), buildPileY-18, "Building Piles", 14, colorDimText, gs.fontSrc)
	for i := 0; i < engine.MaxBuildingPiles; i++ {
		bx := buildPileX(i)
		bp := view.BuildingPiles[i]

		highlighted := gs.phase == PhaseSelectTarget

		if bp.Size == 0 {
			DrawEmptySlot(screen, bx, buildPileY, fmt.Sprintf("needs\n%d", bp.NextNeeded), colorBuild, gs.fontSrc)
		} else {
			// Create a temporary card for display.
			c := engine.NewCard(bp.TopValue)
			DrawCard(screen, bx, buildPileY, &c, highlighted, gs.fontSrc)
		}
		DrawPileCount(screen, bx, buildPileY, bp.Size, gs.fontSrc)
	}

	// Draw pile count.
	DrawLabel(screen, float64(buildPileX(engine.MaxBuildingPiles-1)+CardWidth+20), buildPileY+30,
		fmt.Sprintf("Draw: %d", view.DrawPileSize), 12, colorDimText, gs.fontSrc)
}

func (gs *GameScreen) drawOpponents(screen *ebiten.Image, view *engine.GameView) {
	if len(view.Opponents) == 0 {
		return
	}
	DrawLabelCentered(screen, ScreenWidth/2, opponentY-20, "Opponents", 14, colorDimText, gs.fontSrc)

	for i, opp := range view.Opponents {
		ox := opponentX(i, len(view.Opponents))

		// Name.
		DrawLabelCentered(screen, float64(ox)+CardWidth/2, opponentY-5, opp.Name, 12, colorStatusText, gs.fontSrc)

		// Stock pile (face-up top).
		if opp.StockTop != nil {
			DrawCard(screen, ox, opponentY+10, opp.StockTop, false, gs.fontSrc)
		} else {
			DrawEmptySlot(screen, ox, opponentY+10, "stock", colorEmpty, gs.fontSrc)
		}
		DrawPileCount(screen, ox, opponentY+10, opp.StockRemain, gs.fontSrc)

		// Discard tops (small indicators).
		for j := 0; j < engine.MaxDiscardPiles; j++ {
			dx := ox + float32(j)*16
			dy := float32(opponentY) + CardHeight + 18

			if opp.DiscardTops[j] != nil {
				// Small card indicator.
				vector.DrawFilledRect(screen, dx, dy, 14, 18, colorCardFace, false)
				vector.StrokeRect(screen, dx, dy, 14, 18, 1, colorCardBorder, false)
				DrawLabelCentered(screen, float64(dx)+7, float64(dy)+9,
					opp.DiscardTops[j].String(), 9, colorCardBlue, gs.fontSrc)
			}
		}

		// Hand size indicator.
		DrawLabel(screen, float64(ox)+CardWidth+8, float64(opponentY)+30,
			fmt.Sprintf("🃏%d", opp.HandSize), 11, colorDimText, gs.fontSrc)
	}
}

func (gs *GameScreen) drawStock(screen *ebiten.Image, view *engine.GameView) {
	DrawLabel(screen, stockX, stockY-18, "Your Stock", 14, colorStock, gs.fontSrc)

	highlighted := gs.selected != nil && gs.selected.Source == engine.SourceStock

	if view.StockTop != nil {
		DrawCard(screen, stockX, stockY, view.StockTop, highlighted, gs.fontSrc)
	} else {
		DrawEmptySlot(screen, stockX, stockY, "empty!", colorEmpty, gs.fontSrc)
	}
	DrawPileCount(screen, stockX, stockY, view.StockRemain, gs.fontSrc)
}

func (gs *GameScreen) drawHand(screen *ebiten.Image, view *engine.GameView) {
	if len(view.Hand) == 0 {
		return
	}
	DrawLabelCentered(screen, ScreenWidth/2, handCardY-18, "Your Hand", 14, colorStatusText, gs.fontSrc)

	for i, card := range view.Hand {
		cx := handCardX(i, len(view.Hand))
		highlighted := gs.selected != nil && gs.selected.Source == engine.SourceHand && gs.selected.Index == i
		c := card // Copy for pointer.
		DrawCard(screen, cx, handCardY, &c, highlighted, gs.fontSrc)
	}
}

func (gs *GameScreen) drawDiscards(screen *ebiten.Image, view *engine.GameView) {
	DrawLabel(screen, float64(discardX(0)), discardY-18, "Your Discards", 14, colorDiscard, gs.fontSrc)

	for i := 0; i < engine.MaxDiscardPiles; i++ {
		dx := discardX(i)
		pile := view.DiscardPiles[i]

		highlighted := gs.selected != nil && gs.selected.Source == engine.SourceDiscard && gs.selected.Index == i
		targetHighlight := gs.phase == PhaseSelectTarget && gs.selected != nil && gs.selected.Source == engine.SourceHand

		if len(pile) == 0 {
			if targetHighlight {
				DrawEmptySlot(screen, dx, discardY, "discard\nhere", colorHighlight, gs.fontSrc)
			} else {
				DrawEmptySlot(screen, dx, discardY, fmt.Sprintf("D%d", i), colorEmpty, gs.fontSrc)
			}
		} else {
			top := pile[len(pile)-1]
			DrawCard(screen, dx, discardY, &top, highlighted || targetHighlight, gs.fontSrc)
			DrawPileCount(screen, dx, discardY, len(pile), gs.fontSrc)
		}
	}
}

func (gs *GameScreen) drawStatus(screen *ebiten.Image) {
	// Status bar at bottom.
	vector.DrawFilledRect(screen, 0, ScreenHeight-40, ScreenWidth, 40, color.RGBA{0x10, 0x10, 0x10, 0xCC}, false)
	DrawLabel(screen, 20, ScreenHeight-16, gs.statusMsg, 14, gs.statusColor, gs.fontSrc)

	// Turn number.
	if gs.currentView != nil {
		turnLabel := fmt.Sprintf("Turn %d", gs.currentView.TurnNumber)
		DrawLabel(screen, ScreenWidth-80, ScreenHeight-16, turnLabel, 12, colorDimText, gs.fontSrc)
	}
}

// ---------------------------------------------------------------------------
// Ebitengine interface: Layout
// ---------------------------------------------------------------------------

func (gs *GameScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// ---------------------------------------------------------------------------
// Animations
// ---------------------------------------------------------------------------

func (gs *GameScreen) drawArrow(screen *ebiten.Image, view *engine.GameView) {
	if gs.lastAction == nil {
		return
	}
	// Fade out arrow after 2 seconds
	elapsed := time.Since(gs.lastActionTime)
	alpha := 255 - int(elapsed.Seconds()*128)
	if alpha <= 0 {
		return
	}
	if alpha > 255 {
		alpha = 255
	}
	
	var x1, y1, x2, y2 float32
	
	// Target
	if gs.lastAction.Target == engine.TargetBuild {
		x2 = buildPileX(gs.lastAction.TargetIndex) + CardWidth/2
		y2 = buildPileY + CardHeight/2
	} else {
		if gs.lastActionPlayer == view.PlayerIndex {
			x2 = discardX(gs.lastAction.TargetIndex) + CardWidth/2
			y2 = discardY + CardHeight/2
		} else {
			oppIdx := gs.getOpponentViewIndex(gs.lastActionPlayer, view)
			if oppIdx >= 0 {
				ox := opponentX(oppIdx, len(view.Opponents))
				x2 = ox + float32(gs.lastAction.TargetIndex)*16 + 7
				y2 = float32(opponentY) + CardHeight + 18 + 9
			}
		}
	}
	
	// Source
	if gs.lastActionPlayer == view.PlayerIndex {
		switch gs.lastAction.Source {
		case engine.SourceHand:
			x1 = handCardX(gs.lastAction.SourceIndex, len(view.Hand)) + CardWidth/2
			y1 = handCardY + CardHeight/2
		case engine.SourceStock:
			x1 = stockX + CardWidth/2
			y1 = stockY + CardHeight/2
		case engine.SourceDiscard:
			x1 = discardX(gs.lastAction.SourceIndex) + CardWidth/2
			y1 = discardY + CardHeight/2
		}
	} else {
		oppIdx := gs.getOpponentViewIndex(gs.lastActionPlayer, view)
		if oppIdx >= 0 {
			ox := opponentX(oppIdx, len(view.Opponents))
			switch gs.lastAction.Source {
			case engine.SourceHand: 
				x1 = float32(ox) + CardWidth + 8
				y1 = float32(opponentY) + 30
			case engine.SourceStock:
				x1 = float32(ox) + CardWidth/2
				y1 = float32(opponentY) + 10 + CardHeight/2
			case engine.SourceDiscard:
				x1 = ox + float32(gs.lastAction.SourceIndex)*16 + 7
				y1 = float32(opponentY) + CardHeight + 18 + 9
			}
		}
	}
	
	if x1 == 0 && y1 == 0 {
		return
	}
	
	c := color.RGBA{0xFF, 0x8C, 0x00, uint8(alpha)}
	vector.StrokeLine(screen, x1, y1, x2, y2, 6, c, true)
	vector.DrawFilledCircle(screen, x2, y2, 12, c, true)
}

func (gs *GameScreen) getOpponentViewIndex(playerIdx int, view *engine.GameView) int {
	if playerIdx < view.PlayerIndex {
		return playerIdx
	} else if playerIdx > view.PlayerIndex {
		return playerIdx - 1
	}
	return -1
}
