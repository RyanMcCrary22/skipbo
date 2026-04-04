package engine

// MockPlayer is a deterministic, reusable Player implementation for tests.
// It can be used in two modes:
//   - Script mode: replays a fixed sequence of actions.
//   - Callback mode: delegates decisions to a user-provided function.
//
// It also records every GameView snapshot it receives so that tests can
// assert exactly what the engine showed the player.
//
// Unlike the test-internal ScriptedPlayer in game_test.go, MockPlayer is
// exported so that GUI, network, and other packages can use it in their
// own tests.
type MockPlayer struct {
	name     string
	actions  []Action
	index    int
	callback func(view *GameView) (Action, error)

	// Views records every GameView snapshot passed to ChooseAction.
	Views []*GameView
}

// MockPlayerOption configures a MockPlayer.
type MockPlayerOption func(*MockPlayer)

// WithActions sets the player to replay the given action sequence.
// After the sequence is exhausted, it falls back to discarding hand[0]→discard[0].
func WithActions(actions ...Action) MockPlayerOption {
	return func(m *MockPlayer) { m.actions = actions }
}

// WithCallback sets the player to delegate decisions to fn.
// This takes precedence over WithActions if both are set.
func WithCallback(fn func(*GameView) (Action, error)) MockPlayerOption {
	return func(m *MockPlayer) { m.callback = fn }
}

// NewMockPlayer creates a MockPlayer with the given name and options.
func NewMockPlayer(name string, opts ...MockPlayerOption) *MockPlayer {
	m := &MockPlayer{name: name}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Name returns the player's display name.
func (m *MockPlayer) Name() string { return m.name }

// ChooseAction records the view and then returns an action.
func (m *MockPlayer) ChooseAction(view *GameView) (Action, error) {
	m.Views = append(m.Views, view)

	// Callback mode takes precedence.
	if m.callback != nil {
		return m.callback(view)
	}

	// Script mode.
	if m.index < len(m.actions) {
		a := m.actions[m.index]
		m.index++
		return a, nil
	}

	// Fallback: discard first hand card to first discard pile.
	return DiscardFromHand(0, 0), nil
}

// Reset clears the recorded views and resets the action index.
func (m *MockPlayer) Reset() {
	m.Views = nil
	m.index = 0
}
