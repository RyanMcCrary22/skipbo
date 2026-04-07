package training

import (
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// OnnxPlayer — a Player that runs ONNX model inference for action selection.
//
// This player loads a trained PPO policy exported to ONNX format and uses
// it to select actions during gameplay. It converts the GameView to a state
// vector, runs inference, masks illegal actions, and picks the best legal
// action via argmax.
// ---------------------------------------------------------------------------

// OnnxPlayer implements the engine.Player interface using an ONNX model.
type OnnxPlayer struct {
	name    string
	session *ort.AdvancedSession
	input   *ort.Tensor[float32]
	output  *ort.Tensor[float32]
}

// NewOnnxPlayer creates a new OnnxPlayer that loads the given ONNX model file.
// Call InitOnnxRuntime() before creating any OnnxPlayer instances.
func NewOnnxPlayer(name string, modelPath string) (*OnnxPlayer, error) {
	// Create input tensor [1, 54].
	inputShape := ort.NewShape(1, int64(engine.StateVectorSize))
	input, err := ort.NewEmptyTensor[float32](inputShape)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}

	// Create output tensor [1, 60].
	outputShape := ort.NewShape(1, int64(engine.TotalActions))
	output, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		input.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	// Create the ONNX session.
	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"observation"},
		[]string{"action_logits"},
		[]ort.ArbitraryTensor{input},
		[]ort.ArbitraryTensor{output},
		nil,
	)
	if err != nil {
		input.Destroy()
		output.Destroy()
		return nil, fmt.Errorf("failed to create ONNX session for %s: %w", modelPath, err)
	}

	return &OnnxPlayer{
		name:    name,
		session: session,
		input:   input,
		output:  output,
	}, nil
}

// Name returns the player's display name.
func (p *OnnxPlayer) Name() string { return p.name }

// ChooseAction uses the ONNX model to select the best legal action.
func (p *OnnxPlayer) ChooseAction(view *engine.GameView) (engine.Action, error) {
	// 1. Convert GameView to state vector.
	stateVec := engine.StateVector(view)

	// 2. Fill the input tensor.
	inputData := p.input.GetData()
	for i, v := range stateVec {
		inputData[i] = float32(v)
	}

	// 3. Run inference.
	if err := p.session.Run(); err != nil {
		return engine.Action{}, fmt.Errorf("ONNX inference failed: %w", err)
	}

	// 4. Get action logits from output tensor.
	logits := p.output.GetData()

	// 5. Compute action mask.
	mask := engine.ActionMask(view)

	// 6. Pick the best legal action (masked argmax).
	bestIdx := -1
	bestVal := math.Inf(-1)
	for i := 0; i < engine.TotalActions; i++ {
		if mask[i] && float64(logits[i]) > bestVal {
			bestVal = float64(logits[i])
			bestIdx = i
		}
	}

	if bestIdx < 0 {
		// No legal action found — shouldn't happen in normal play.
		// Fall back to first available discard.
		for i := 0; i < engine.TotalActions; i++ {
			if mask[i] {
				bestIdx = i
				break
			}
		}
		if bestIdx < 0 {
			return engine.Action{}, fmt.Errorf("no legal actions available")
		}
	}

	action, ok := engine.ActionFromIndex(bestIdx)
	if !ok {
		return engine.Action{}, fmt.Errorf("invalid action index: %d", bestIdx)
	}

	return action, nil
}

// Close frees the underlying ONNX runtime session memory.
// It is critical to call this when the player is no longer needed.
func (p *OnnxPlayer) Close() error {
	var err error
	if p.session != nil {
		err = p.session.Destroy()
		p.session = nil
	}
	if p.input != nil {
		p.input.Destroy()
		p.input = nil
	}
	if p.output != nil {
		p.output.Destroy()
		p.output = nil
	}
	return err
}

// ---------------------------------------------------------------------------
// ONNX Runtime initialization helpers
// ---------------------------------------------------------------------------

// InitOnnxRuntime initializes the ONNX Runtime shared library.
// Must be called once before creating any OnnxPlayer instances.
// libraryPath should point to the onnxruntime shared library file
// (e.g., "libonnxruntime.dylib" on macOS, "libonnxruntime.so" on Linux).
func InitOnnxRuntime(libraryPath string) error {
	ort.SetSharedLibraryPath(libraryPath)
	return ort.InitializeEnvironment()
}

// DestroyOnnxRuntime cleans up the ONNX Runtime environment.
// Call this when the application is shutting down.
func DestroyOnnxRuntime() error {
	return ort.DestroyEnvironment()
}
