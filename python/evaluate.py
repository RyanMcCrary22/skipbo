"""
Evaluate a trained Skip-Bo PPO agent.

Loads a saved model checkpoint and plays N games against random opponents,
reporting win rate, average game length, and stockpile efficiency.

Usage:
    # Start the Go server first:
    go run ./cmd/train/ --port 50051

    # Then evaluate:
    python evaluate.py --model runs/<run_id>/final_model.zip --games 500
"""

import argparse
import json
import sys
from pathlib import Path

import numpy as np

try:
    from sb3_contrib import MaskablePPO
except ImportError:
    print("ERROR: sb3_contrib not installed. Run: pip install -r requirements.txt")
    sys.exit(1)

from skipbo_env import SkipBoEnv


def evaluate(model_path: str, server_addr: str, n_games: int, stock_size: int):
    """Play n_games and report statistics."""
    print(f"🎯 Evaluating {model_path}")
    print(f"   Games:  {n_games}")
    print(f"   Server: {server_addr}")
    print()

    model = MaskablePPO.load(model_path)
    env = SkipBoEnv(server_addr=server_addr, stock_size=stock_size)

    wins = 0
    game_lengths = []
    stock_plays_per_game = []

    for i in range(n_games):
        obs, info = env.reset()
        done = False
        stock_plays = 0
        steps = 0

        while not done:
            mask = env.action_masks()
            action, _ = model.predict(obs, deterministic=True, action_masks=mask)
            obs, reward, done, truncated, info = env.step(int(action))
            steps += 1
            if info.get("played_from_stock", False):
                stock_plays += 1

        if info.get("winner", -1) == 0:
            wins += 1

        turn_num = info.get("turn_number", steps)
        game_lengths.append(turn_num)
        stock_plays_per_game.append(stock_plays)

        if (i + 1) % 50 == 0:
            wr = wins / (i + 1)
            print(f"  [{i+1}/{n_games}] win_rate={wr:.2%}")

    env.close()

    # Results.
    win_rate = wins / n_games
    avg_length = np.mean(game_lengths)
    avg_stock = np.mean(stock_plays_per_game)

    print()
    print(f"═══════════════════════════════════════")
    print(f"  Results ({n_games} games)")
    print(f"═══════════════════════════════════════")
    print(f"  Win Rate:              {win_rate:.2%} ({wins}/{n_games})")
    print(f"  Avg Game Length:       {avg_length:.1f} turns")
    print(f"  Avg Stockpile Plays:   {avg_stock:.1f} per game")
    print(f"  Min/Max Game Length:   {min(game_lengths)}/{max(game_lengths)}")
    print(f"═══════════════════════════════════════")

    # Save results.
    results = {
        "model_path": model_path,
        "n_games": n_games,
        "win_rate": win_rate,
        "wins": wins,
        "avg_game_length": round(float(avg_length), 2),
        "avg_stockpile_plays": round(float(avg_stock), 2),
        "min_game_length": int(min(game_lengths)),
        "max_game_length": int(max(game_lengths)),
    }

    out_path = Path(model_path).parent / "eval_results.json"
    with open(out_path, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\n📝 Results saved to {out_path}")

    return results


def main():
    parser = argparse.ArgumentParser(description="Evaluate a trained Skip-Bo agent")
    parser.add_argument("--model", type=str, required=True,
                        help="Path to saved model .zip")
    parser.add_argument("--server", type=str, default="localhost:50051",
                        help="gRPC server address")
    parser.add_argument("--games", type=int, default=500,
                        help="Number of evaluation games")
    parser.add_argument("--stock-size", type=int, default=30,
                        help="Stock pile size")
    args = parser.parse_args()

    evaluate(args.model, args.server, args.games, args.stock_size)


if __name__ == "__main__":
    main()
