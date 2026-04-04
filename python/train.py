"""
PPO training script for the Skip-Bo agent.

Trains a MaskablePPO agent against random opponents via the Go gRPC server.
Logs metrics to TensorBoard and saves a training_report.json at the end.

Usage:
    # Start the Go server first:
    go run ./cmd/train/ --port 50051

    # Then run training:
    python train.py --timesteps 500000

    # Monitor with TensorBoard:
    tensorboard --logdir runs/
"""

import argparse
import json
import os
import platform
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

import grpc
import numpy as np

# Import after checking deps exist.
try:
    from sb3_contrib import MaskablePPO
    from sb3_contrib.common.wrappers import ActionMasker
    from stable_baselines3.common.callbacks import (
        BaseCallback,
        CheckpointCallback,
    )
except ImportError:
    print("ERROR: sb3_contrib not installed. Run: pip install -r requirements.txt")
    sys.exit(1)

from skipbo_env import SkipBoEnv
import skipbo_pb2
import skipbo_pb2_grpc


# ---------------------------------------------------------------------------
# Custom callback for checkpoint evaluation
# ---------------------------------------------------------------------------

class EvalAndLogCallback(BaseCallback):
    """Evaluates the agent every `eval_freq` steps and logs results."""

    def __init__(
        self,
        eval_freq: int = 50_000,
        n_eval_games: int = 100,
        run_dir: str = "runs/default",
        server_addr: str = "localhost:50051",
        verbose: int = 1,
    ):
        super().__init__(verbose)
        self.eval_freq = eval_freq
        self.n_eval_games = n_eval_games
        self.run_dir = Path(run_dir)
        self.checkpoints_dir = self.run_dir / "checkpoints"
        self.checkpoints_dir.mkdir(parents=True, exist_ok=True)
        self.server_addr = server_addr
        self.eval_history = []

    def _on_step(self) -> bool:
        if self.num_timesteps % self.eval_freq == 0 and self.num_timesteps > 0:
            self._evaluate()
        return True

    def _evaluate(self):
        """Run N games and compute win rate."""
        env = SkipBoEnv(server_addr=self.server_addr)
        wins = 0
        total_turns = 0
        total_stock_plays = 0

        for _ in range(self.n_eval_games):
            obs, info = env.reset()
            done = False
            game_stock_plays = 0
            while not done:
                mask = env.action_masks()
                action, _ = self.model.predict(
                    obs, deterministic=True, action_masks=mask
                )
                obs, reward, done, truncated, info = env.step(int(action))
                if info.get("played_from_stock", False):
                    game_stock_plays += 1

            if info.get("winner", -1) == 0:
                wins += 1
            total_turns += info.get("turn_number", 0)
            total_stock_plays += game_stock_plays

        env.close()

        win_rate = wins / self.n_eval_games
        avg_turns = total_turns / self.n_eval_games if self.n_eval_games > 0 else 0
        avg_stock = total_stock_plays / self.n_eval_games if self.n_eval_games > 0 else 0

        # Log to TensorBoard.
        self.logger.record("eval/win_rate", win_rate)
        self.logger.record("eval/avg_game_length", avg_turns)
        self.logger.record("eval/avg_stockpile_plays", avg_stock)

        # Save checkpoint.
        step = self.num_timesteps
        model_path = self.checkpoints_dir / f"model_{step}.zip"
        self.model.save(str(model_path))

        # Save eval snapshot.
        snapshot = {
            "step": step,
            "win_rate": win_rate,
            "avg_game_length": avg_turns,
            "avg_stockpile_plays": avg_stock,
            "model_path": str(model_path),
        }
        eval_path = self.checkpoints_dir / f"eval_{step}.json"
        with open(eval_path, "w") as f:
            json.dump(snapshot, f, indent=2)

        self.eval_history.append(snapshot)

        if self.verbose:
            print(
                f"\n📊 Eval @ step {step}: "
                f"win_rate={win_rate:.2%}, "
                f"avg_turns={avg_turns:.1f}, "
                f"avg_stock_plays={avg_stock:.1f}"
            )


# ---------------------------------------------------------------------------
# Action mask wrapper for SB3
# ---------------------------------------------------------------------------

def mask_fn(env: SkipBoEnv) -> np.ndarray:
    """Returns the current action mask from the environment."""
    return env.action_masks()


# ---------------------------------------------------------------------------
# Hardware detection
# ---------------------------------------------------------------------------

def get_hardware_info() -> dict:
    """Gather hardware info for the training report."""
    info = {
        "chip": platform.processor() or platform.machine(),
        "system": platform.system(),
        "python_version": platform.python_version(),
    }
    try:
        import psutil
        info["ram_gb"] = round(psutil.virtual_memory().total / (1024**3), 1)
        info["cores"] = psutil.cpu_count(logical=True)
    except ImportError:
        info["ram_gb"] = "unknown"
        info["cores"] = os.cpu_count()
    return info


# ---------------------------------------------------------------------------
# Main training loop
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Train a Skip-Bo PPO agent")
    parser.add_argument("--timesteps", type=int, default=500_000,
                        help="Total training timesteps")
    parser.add_argument("--server", type=str, default="localhost:50051",
                        help="gRPC server address")
    parser.add_argument("--eval-freq", type=int, default=50_000,
                        help="Evaluate every N steps")
    parser.add_argument("--eval-games", type=int, default=100,
                        help="Number of games per evaluation")
    parser.add_argument("--hidden", type=int, nargs="+", default=[128, 128],
                        help="Hidden layer sizes")
    parser.add_argument("--lr", type=float, default=3e-4,
                        help="Learning rate")
    parser.add_argument("--n-steps", type=int, default=2048,
                        help="Steps per rollout")
    parser.add_argument("--batch-size", type=int, default=64,
                        help="Minibatch size")
    parser.add_argument("--n-epochs", type=int, default=10,
                        help="PPO epochs per rollout")
    parser.add_argument("--opponents", type=int, default=1,
                        help="Number of random opponents")
    parser.add_argument("--stock-size", type=int, default=30,
                        help="Stock pile size")
    args = parser.parse_args()

    # Create run directory.
    run_id = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    run_dir = Path("runs") / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    print(f"🎮 Skip-Bo PPO Training")
    print(f"   Run ID:     {run_id}")
    print(f"   Server:     {args.server}")
    print(f"   Timesteps:  {args.timesteps:,}")
    print(f"   Network:    {args.hidden}")
    print(f"   LR:         {args.lr}")
    print(f"   Run dir:    {run_dir}")
    print()

    # Create environment.
    env = SkipBoEnv(
        server_addr=args.server,
        num_opponents=args.opponents,
        stock_size=args.stock_size,
    )
    env = ActionMasker(env, mask_fn)

    # Create PPO agent.
    model = MaskablePPO(
        "MlpPolicy",
        env,
        learning_rate=args.lr,
        n_steps=args.n_steps,
        batch_size=args.batch_size,
        n_epochs=args.n_epochs,
        policy_kwargs={"net_arch": args.hidden},
        tensorboard_log=str(run_dir / "tensorboard"),
        verbose=1,
    )

    # Callbacks.
    eval_callback = EvalAndLogCallback(
        eval_freq=args.eval_freq,
        n_eval_games=args.eval_games,
        run_dir=str(run_dir),
        server_addr=args.server,
    )

    # Train!
    start_time = time.time()
    model.learn(
        total_timesteps=args.timesteps,
        callback=eval_callback,
        tb_log_name="ppo",
    )
    wall_clock = time.time() - start_time

    # Save final model.
    final_path = run_dir / "final_model.zip"
    model.save(str(final_path))
    print(f"\n✅ Final model saved to {final_path}")

    # Fetch server-side metrics.
    try:
        channel = grpc.insecure_channel(args.server)
        stub = skipbo_pb2_grpc.SkipBoEnvStub(channel)
        metrics = stub.GetMetrics(skipbo_pb2.MetricsRequest())
        channel.close()
        server_metrics = {
            "total_games": metrics.total_games,
            "total_steps": metrics.total_steps,
            "agent_wins": metrics.agent_wins,
            "avg_game_length": round(metrics.avg_game_length, 2),
            "avg_stockpile_plays": round(metrics.avg_stockpile_plays, 2),
            "illegal_actions": metrics.illegal_actions,
        }
    except Exception as e:
        print(f"⚠️  Could not fetch server metrics: {e}")
        server_metrics = {}

    # Generate training report.
    report = {
        "run_id": run_id,
        "hardware": get_hardware_info(),
        "hyperparameters": {
            "policy": "MlpPolicy",
            "hidden": args.hidden,
            "n_steps": args.n_steps,
            "batch_size": args.batch_size,
            "n_epochs": args.n_epochs,
            "lr": args.lr,
            "total_timesteps": args.timesteps,
            "num_opponents": args.opponents,
            "stock_size": args.stock_size,
        },
        "results": {
            **server_metrics,
            "final_win_rate": (
                eval_callback.eval_history[-1]["win_rate"]
                if eval_callback.eval_history
                else None
            ),
            "wall_clock_seconds": round(wall_clock, 1),
        },
        "checkpoints": eval_callback.eval_history,
    }

    report_path = run_dir / "training_report.json"
    with open(report_path, "w") as f:
        json.dump(report, f, indent=2)

    print(f"📝 Training report saved to {report_path}")
    print(f"⏱️  Wall clock: {wall_clock:.0f}s ({wall_clock/3600:.1f}h)")
    if eval_callback.eval_history:
        final_wr = eval_callback.eval_history[-1]["win_rate"]
        print(f"🏆 Final win rate: {final_wr:.2%}")

    env.close()


if __name__ == "__main__":
    main()
