package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Action constructor tests
// ---------------------------------------------------------------------------

func TestPlayFromHand_Fields(t *testing.T) {
	a := PlayFromHand(2, 3)
	assert.Equal(t, SourceHand, a.Source)
	assert.Equal(t, 2, a.SourceIndex)
	assert.Equal(t, TargetBuild, a.Target)
	assert.Equal(t, 3, a.TargetIndex)
}

func TestPlayFromStock_Fields(t *testing.T) {
	a := PlayFromStock(1)
	assert.Equal(t, SourceStock, a.Source)
	assert.Equal(t, 0, a.SourceIndex, "stock has no source index")
	assert.Equal(t, TargetBuild, a.Target)
	assert.Equal(t, 1, a.TargetIndex)
}

func TestPlayFromDiscard_Fields(t *testing.T) {
	a := PlayFromDiscard(2, 0)
	assert.Equal(t, SourceDiscard, a.Source)
	assert.Equal(t, 2, a.SourceIndex)
	assert.Equal(t, TargetBuild, a.Target)
	assert.Equal(t, 0, a.TargetIndex)
}

func TestDiscardFromHand_Fields(t *testing.T) {
	a := DiscardFromHand(4, 3)
	assert.Equal(t, SourceHand, a.Source)
	assert.Equal(t, 4, a.SourceIndex)
	assert.Equal(t, TargetDiscard, a.Target)
	assert.Equal(t, 3, a.TargetIndex)
}

// ---------------------------------------------------------------------------
// IsDiscard tests
// ---------------------------------------------------------------------------

func TestAction_IsDiscard(t *testing.T) {
	tests := []struct {
		name string
		a    Action
		want bool
	}{
		{"hand to build", PlayFromHand(0, 0), false},
		{"stock to build", PlayFromStock(0), false},
		{"discard to build", PlayFromDiscard(0, 0), false},
		{"hand to discard", DiscardFromHand(0, 0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.IsDiscard())
		})
	}
}

// ---------------------------------------------------------------------------
// String representation tests
// ---------------------------------------------------------------------------

func TestSource_String(t *testing.T) {
	assert.Equal(t, "hand", SourceHand.String())
	assert.Equal(t, "stock", SourceStock.String())
	assert.Equal(t, "discard", SourceDiscard.String())
	assert.Contains(t, Source(99).String(), "99", "unknown source should include the int value")
}

func TestTarget_String(t *testing.T) {
	assert.Equal(t, "build", TargetBuild.String())
	assert.Equal(t, "discard", TargetDiscard.String())
	assert.Contains(t, Target(99).String(), "99", "unknown target should include the int value")
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		name string
		a    Action
		want string
	}{
		{"hand to build", PlayFromHand(2, 1), "hand[2] → build[1]"},
		{"stock to build", PlayFromStock(3), "stock → build[3]"},
		{"discard to build", PlayFromDiscard(0, 2), "discard[0] → build[2]"},
		{"hand to discard", DiscardFromHand(1, 3), "hand[1] → discard[3]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.a.String())
		})
	}
}
