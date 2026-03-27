package gui

import (
	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// GUIPlayer — implements engine.Player via the GUI
// ---------------------------------------------------------------------------

// GUIPlayer bridges the engine.Player interface with the Ebitengine GUI.
// It communicates with the GameScreen via channels: the game engine sends
// GameView snapshots to the GUI, and the GUI sends back chosen actions.
type GUIPlayer struct {
	name   string
	screen *GameScreen
}

// NewGUIPlayer creates a GUI-based player. It must be paired with a
// GameScreen that owns the display and click handling.
func NewGUIPlayer(name string, screen *GameScreen) *GUIPlayer {
	return &GUIPlayer{name: name, screen: screen}
}

func (p *GUIPlayer) Name() string { return p.name }

// ChooseAction blocks until the GUI player clicks a valid action.
// It sends the current view to the GUI thread and waits for a response.
func (p *GUIPlayer) ChooseAction(view *engine.GameView) (engine.Action, error) {
	// Send the view to the GUI for rendering.
	p.screen.viewCh <- view

	// Block until the GUI sends back an action.
	action := <-p.screen.actionCh
	return action, nil
}
