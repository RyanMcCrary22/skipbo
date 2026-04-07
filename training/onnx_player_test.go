package training

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOnnxPlayer(t *testing.T) {
	// Find libonnxruntime library
	libName := "libonnxruntime.so"
	if runtime.GOOS == "darwin" {
		libName = "libonnxruntime.dylib"
	} else if runtime.GOOS == "windows" {
		libName = "onnxruntime.dll"
	}
	
	// We assume it's installed globally or available in the path,
	// but onnxruntime usually requires specific library paths.
	// For testing, we just try to initialize and check if we get a library not found error.
	
	// actually for a quick smoke test we just check if it fails gracefully
	err := InitOnnxRuntime(libName)
	if err != nil {
		t.Skipf("Skipping ONNX tests since onnxruntime lib %s not found: %v", libName, err)
	}
	defer DestroyOnnxRuntime()

	modelPath := filepath.Join("..", "models", "v4_28M.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping tests because %s does not exist", modelPath)
	}

	player, err := NewOnnxPlayer("TestBot", modelPath)
	if err != nil {
		t.Fatalf("Failed to load OnnxPlayer: %v", err)
	}
	defer player.Close()

	if player.Name() != "TestBot" {
		t.Errorf("Expected name TestBot, got %s", player.Name())
	}
}
