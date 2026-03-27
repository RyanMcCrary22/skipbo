// Package gui provides an Ebitengine-based graphical interface for Skip-Bo.
package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// Colors
// ---------------------------------------------------------------------------

var (
	colorFelt       = color.RGBA{0x1B, 0x5E, 0x20, 0xFF} // Dark green felt.
	colorCardFace   = color.RGBA{0xFA, 0xFA, 0xFA, 0xFF} // Off-white card.
	colorCardBorder = color.RGBA{0x42, 0x42, 0x42, 0xFF} // Dark gray border.
	colorCardRed    = color.RGBA{0xD3, 0x2F, 0x2F, 0xFF} // Wild card red.
	colorCardBlue   = color.RGBA{0x1E, 0x88, 0xE5, 0xFF} // Normal card blue.
	colorHighlight  = color.RGBA{0xFF, 0xD5, 0x4F, 0xCC} // Selection highlight.
	colorEmpty      = color.RGBA{0x2E, 0x7D, 0x32, 0xFF} // Empty pile slot.
	colorStock      = color.RGBA{0xFF, 0xA0, 0x00, 0xFF} // Stock pile orange.
	colorDiscard    = color.RGBA{0x64, 0xB5, 0xF6, 0xFF} // Discard pile blue.
	colorBuild      = color.RGBA{0x81, 0xC7, 0x84, 0xFF} // Build pile light green.
	colorError      = color.RGBA{0xFF, 0x50, 0x50, 0xFF} // Error text red.
	colorStatusText = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF} // White text.
	colorDimText    = color.RGBA{0xA0, 0xA0, 0xA0, 0xFF} // Dimmed text.
)

// ---------------------------------------------------------------------------
// Card dimensions and layout
// ---------------------------------------------------------------------------

const (
	CardWidth   = 60
	CardHeight  = 84
	CardRadius  = 6
	CardSpacing = 8

	ScreenWidth  = 900
	ScreenHeight = 700
)

// ---------------------------------------------------------------------------
// DrawCard renders a single card at the given position.
// ---------------------------------------------------------------------------

func DrawCard(screen *ebiten.Image, x, y float32, card *engine.Card, highlighted bool, faceSource *text.GoTextFaceSource) {
	// Card background.
	vector.DrawFilledRect(screen, x, y, CardWidth, CardHeight, colorCardFace, false)

	// Border.
	strokeWidth := float32(2)
	if highlighted {
		strokeWidth = 3
		vector.DrawFilledRect(screen, x-3, y-3, CardWidth+6, CardHeight+6, colorHighlight, false)
		vector.DrawFilledRect(screen, x, y, CardWidth, CardHeight, colorCardFace, false)
	}
	vector.StrokeRect(screen, x, y, CardWidth, CardHeight, strokeWidth, colorCardBorder, false)

	if card == nil {
		// Empty pile placeholder.
		vector.DrawFilledRect(screen, x, y, CardWidth, CardHeight, colorEmpty, false)
		vector.StrokeRect(screen, x, y, CardWidth, CardHeight, 1, colorCardBorder, false)
		return
	}

	// Card value text.
	label := card.Value.String()
	textColor := colorCardBlue
	if card.Value.IsWild() {
		textColor = colorCardRed
	}

	if faceSource != nil {
		face := &text.GoTextFace{
			Source: faceSource,
			Size:   22,
		}
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(float64(x)+float64(CardWidth)/2, float64(y)+float64(CardHeight)/2)
		opts.ColorScale.ScaleWithColor(textColor)
		opts.PrimaryAlign = text.AlignCenter
		opts.SecondaryAlign = text.AlignCenter
		text.Draw(screen, label, face, opts)
	}
}

// DrawCardBack renders a face-down card (for opponent stock piles, etc.)
func DrawCardBack(screen *ebiten.Image, x, y float32) {
	vector.DrawFilledRect(screen, x, y, CardWidth, CardHeight, colorCardRed, false)
	vector.StrokeRect(screen, x, y, CardWidth, CardHeight, 2, colorCardBorder, false)

	// Cross-hatch pattern.
	for i := float32(0); i < CardWidth; i += 10 {
		vector.StrokeLine(screen, x+i, y, x+i, y+CardHeight, 0.5, color.RGBA{0xFF, 0xFF, 0xFF, 0x40}, false)
	}
	for j := float32(0); j < CardHeight; j += 10 {
		vector.StrokeLine(screen, x, y+j, x+CardWidth, y+j, 0.5, color.RGBA{0xFF, 0xFF, 0xFF, 0x40}, false)
	}
}

// DrawEmptySlot draws an empty pile slot with a label.
func DrawEmptySlot(screen *ebiten.Image, x, y float32, label string, slotColor color.Color, faceSource *text.GoTextFaceSource) {
	vector.DrawFilledRect(screen, x, y, CardWidth, CardHeight, slotColor, false)
	vector.StrokeRect(screen, x, y, CardWidth, CardHeight, 1, colorCardBorder, false)

	if faceSource != nil {
		face := &text.GoTextFace{
			Source: faceSource,
			Size:   12,
		}
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(float64(x)+float64(CardWidth)/2, float64(y)+float64(CardHeight)/2)
		opts.ColorScale.ScaleWithColor(colorDimText)
		opts.PrimaryAlign = text.AlignCenter
		opts.SecondaryAlign = text.AlignCenter
		text.Draw(screen, label, face, opts)
	}
}

// DrawLabel draws a text label at the given position.
func DrawLabel(screen *ebiten.Image, x, y float64, label string, size float64, c color.Color, faceSource *text.GoTextFaceSource) {
	if faceSource == nil {
		return
	}
	face := &text.GoTextFace{
		Source: faceSource,
		Size:   size,
	}
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(c)
	text.Draw(screen, label, face, opts)
}

// DrawLabelCentered draws a text label centered at the given position.
func DrawLabelCentered(screen *ebiten.Image, x, y float64, label string, size float64, c color.Color, faceSource *text.GoTextFaceSource) {
	if faceSource == nil {
		return
	}
	face := &text.GoTextFace{
		Source: faceSource,
		Size:   size,
	}
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(c)
	opts.PrimaryAlign = text.AlignCenter
	opts.SecondaryAlign = text.AlignCenter
	text.Draw(screen, label, face, opts)
}

// CardHitTest checks if a point (mx, my) is inside a card at (x, y).
func CardHitTest(mx, my, x, y float32) bool {
	return mx >= x && mx <= x+CardWidth && my >= y && my <= y+CardHeight
}

// cardCountBadge draws a small count badge on a pile.
func DrawPileCount(screen *ebiten.Image, x, y float32, count int, faceSource *text.GoTextFaceSource) {
	if count <= 0 {
		return
	}
	label := fmt.Sprintf("%d", count)

	// Badge background.
	bx := x + CardWidth - 16
	by := y - 4
	vector.DrawFilledCircle(screen, bx+8, by+8, 10, color.RGBA{0x33, 0x33, 0x33, 0xDD}, false)

	if faceSource != nil {
		face := &text.GoTextFace{
			Source: faceSource,
			Size:   11,
		}
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(float64(bx)+8, float64(by)+8)
		opts.ColorScale.ScaleWithColor(colorStatusText)
		opts.PrimaryAlign = text.AlignCenter
		opts.SecondaryAlign = text.AlignCenter
		text.Draw(screen, label, face, opts)
	}
}

// ---------------------------------------------------------------------------
// Helper used by game_screen to get the bounding rect for a card position
// ---------------------------------------------------------------------------

// CardRect returns the bounding rectangle for a card at (x, y).
func CardRect(x, y float32) image.Rectangle {
	return image.Rect(int(x), int(y), int(x+CardWidth), int(y+CardHeight))
}
