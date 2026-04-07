package training

import (
	"fmt"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// AgentPlayer — a Player that blocks on a channel for external decisions
// ---------------------------------------------------------------------------

// AgentPlayer implements the engine.Player interface by delegating action
// decisions to an external agent (e.g., a Python PPO process over gRPC).
//
// On each call to ChooseAction, it sends the current observation to the
// `ObsCh` channel and blocks until an action index arrives on `ActionCh`.
// The gRPC server bridges these channels to the remote agent.
type AgentPlayer struct {
	name string

	// ObsCh carries the observation to the gRPC server, which forwards
	// it to the Python agent.
	ObsCh chan<- AgentObs

	// ActionCh receives the action index chosen by the remote agent.
	ActionCh <-chan int
	
	// done is closed when the session is cancelled, unblocking the player.
	done <-chan struct{}
}

// AgentObs bundles the state vector and action mask for one decision point.
type AgentObs struct {
	State      []float64
	ActionMask [engine.TotalActions]bool
}

// NewAgentPlayer creates an AgentPlayer with the given channel pair.
func NewAgentPlayer(name string, obsCh chan<- AgentObs, actionCh <-chan int, done <-chan struct{}) *AgentPlayer {
	return &AgentPlayer{
		name:     name,
		ObsCh:    obsCh,
		ActionCh: actionCh,
		done:     done,
	}
}

// Name implements engine.Player.
func (p *AgentPlayer) Name() string { return p.name }

// ChooseAction implements engine.Player.
// It converts the GameView into an observation, sends it to the external
// agent, and blocks until the agent responds with an action index.
func (p *AgentPlayer) ChooseAction(view *engine.GameView) (engine.Action, error) {
	// Build observation from the engine-generated view.
	obs := AgentObs{
		State:      engine.StateVector(view),
		ActionMask: engine.ActionMask(view),
	}

	// Send observation to the gRPC server goroutine.
	select {
	case p.ObsCh <- obs:
	case <-p.done:
		return engine.Action{}, fmt.Errorf("session cancelled")
	}

	// Block until the agent sends back an action index.
	var actionIdx int
	select {
	case actionIdx = <-p.ActionCh:
	case <-p.done:
		return engine.Action{}, fmt.Errorf("session cancelled")
	}

	// Convert index to Action struct.
	action, ok := engine.ActionFromIndex(actionIdx)
	if !ok {
		// Fallback: discard from hand slot 0 to discard pile 0.
		// The engine will reject it if truly illegal, triggering a retry.
		return engine.DiscardFromHand(0, 0), nil
	}
	return action, nil
}
