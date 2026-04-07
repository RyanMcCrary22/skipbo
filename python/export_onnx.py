"""
Export a MaskablePPO model to ONNX format for native Go inference.

Usage:
    python export_onnx.py --model ../models/v1_baseline_1M.zip --output ../models/v1_baseline.onnx
"""

import argparse
import numpy as np
import torch

try:
    from sb3_contrib import MaskablePPO
except ImportError:
    print("ERROR: sb3_contrib not installed. Run: pip install -r requirements.txt")
    exit(1)


def export_to_onnx(model_path: str, output_path: str):
    """Load a MaskablePPO .zip and export the policy network to ONNX."""

    print(f"Loading model from {model_path}...")
    model = MaskablePPO.load(model_path)

    # Extract the policy network (MlpPolicy).
    policy = model.policy
    policy.eval()

    # Get the observation dimension from the model.
    obs_dim = model.observation_space.shape[0]
    print(f"  Observation dim: {obs_dim}")
    print(f"  Action dim: {model.action_space.n}")

    # Create a dummy input matching the observation space.
    dummy_obs = torch.zeros(1, obs_dim, dtype=torch.float32)

    # The MlpPolicy has an internal method to extract features and compute
    # action logits. We need to trace through the actual forward path.
    # The policy's _predict or forward isn't directly ONNX-friendly, so
    # we wrap the key submodules.

    class PolicyWrapper(torch.nn.Module):
        """Wraps the SB3 policy to expose a clean forward(obs) -> logits."""

        def __init__(self, policy):
            super().__init__()
            self.mlp_extractor = policy.mlp_extractor
            self.action_net = policy.action_net

        def forward(self, obs):
            # policy.extract_features returns preprocessed features.
            # For MlpPolicy without feature extractor, obs goes straight
            # into mlp_extractor.
            pi_features, _ = self.mlp_extractor(obs)
            logits = self.action_net(pi_features)
            return logits

    wrapper = PolicyWrapper(policy)
    wrapper.eval()

    # Verify the wrapper produces valid output.
    with torch.no_grad():
        test_logits = wrapper(dummy_obs)
        print(f"  Output shape: {test_logits.shape}")
        print(f"  Sample logits: {test_logits[0, :5].numpy()}")

    # Export to ONNX.
    print(f"\nExporting to {output_path}...")
    torch.onnx.export(
        wrapper,
        dummy_obs,
        output_path,
        input_names=["observation"],
        output_names=["action_logits"],
        dynamic_axes={
            "observation": {0: "batch_size"},
            "action_logits": {0: "batch_size"},
        },
        opset_version=17,
    )

    print(f"✅ ONNX model saved to {output_path}")

    # Quick verification: load the ONNX model and compare outputs.
    try:
        import onnxruntime as ort

        sess = ort.InferenceSession(output_path)
        onnx_input = {"observation": dummy_obs.numpy()}
        onnx_logits = sess.run(None, onnx_input)[0]

        diff = np.abs(test_logits.numpy() - onnx_logits).max()
        print(f"   Max diff (PyTorch vs ONNX): {diff:.8f}")
        if diff < 1e-5:
            print("   ✅ ONNX output matches PyTorch output!")
        else:
            print("   ⚠️  Outputs differ — check the export.")
    except ImportError:
        print("   (onnxruntime not installed — skipping verification)")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Export MaskablePPO to ONNX")
    parser.add_argument("--model", type=str, required=True,
                        help="Path to the .zip model file")
    parser.add_argument("--output", type=str, required=True,
                        help="Path for the output .onnx file")
    args = parser.parse_args()

    export_to_onnx(args.model, args.output)
