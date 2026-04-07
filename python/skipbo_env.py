"""
Gymnasium-compatible environment wrapping the Go gRPC training server.

This module provides SkipBoEnv, which connects to the Go-based Skip-Bo
engine over gRPC and exposes the standard Gymnasium step/reset interface
used by Stable-Baselines3's MaskablePPO.
"""

import gymnasium as gym
import numpy as np
import grpc

import skipbo_pb2
import skipbo_pb2_grpc

# Constants matching the Go engine's agent_api.go.
STATE_DIM = 54
ACTION_DIM = 60


class SkipBoEnv(gym.Env):
    """Skip-Bo RL environment backed by a Go gRPC server."""

    metadata = {"render_modes": []}

    def __init__(
        self,
        server_addr: str = "localhost:50051",
        num_opponents: int = 1,
        stock_size: int = 30,
        opponent_models: list | None = None,
    ):
        super().__init__()

        self.server_addr = server_addr
        self.num_opponents = num_opponents
        self.stock_size = stock_size
        self.opponent_models = opponent_models if opponent_models else []

        # Connect to the Go training server.
        self.channel = grpc.insecure_channel(server_addr)
        self.stub = skipbo_pb2_grpc.SkipBoEnvStub(self.channel)

        # Gymnasium spaces.
        # State: 54-dim float vector. Values range 0–30 (stock counts).
        self.observation_space = gym.spaces.Box(
            low=0.0, high=30.0, shape=(STATE_DIM,), dtype=np.float32
        )
        # Action: discrete 0–59.
        self.action_space = gym.spaces.Discrete(ACTION_DIM)

        self._current_mask = np.ones(ACTION_DIM, dtype=bool)
        self._game_over = True  # Force reset on first step.
        self._session_id = ""

    def reset(self, seed=None, options=None):
        """Start a new game and return the initial observation."""
        super().reset(seed=seed)

        req = skipbo_pb2.ResetRequest(
            seed=seed if seed is not None else 0,
            num_opponents=self.num_opponents,
            stock_size=self.stock_size,
        )
        if self.opponent_models:
            req.opponent_models.extend(self.opponent_models)
            
        resp = self.stub.Reset(req)

        self._session_id = resp.session_id
        obs = np.array(resp.obs.state, dtype=np.float32)
        self._current_mask = np.array(resp.obs.action_mask, dtype=bool)
        self._game_over = False

        info = {"action_mask": self._current_mask}
        return obs, info

    def step(self, action: int):
        """Execute an action and return (obs, reward, terminated, truncated, info)."""
        # SB3's VecEnv may call step after done without calling reset first.
        # Auto-reset to keep the training loop running.
        if self._game_over or not self._session_id:
            obs, info = self.reset()
            return obs, 0.0, False, False, info

        req = skipbo_pb2.StepRequest(
            session_id=self._session_id,
            action=action
        )
        result = self.stub.Step(req)

        obs = np.array(result.obs.state, dtype=np.float32)
        self._current_mask = np.array(result.obs.action_mask, dtype=bool)

        reward = result.reward
        terminated = result.done
        truncated = False  # We don't truncate games.

        if terminated:
            self._game_over = True

        info = {
            "action_mask": self._current_mask,
            "winner": result.winner,
        }
        if result.info:
            info["turn_number"] = result.info.turn_number
            info["agent_stock_remaining"] = result.info.agent_stock_remaining
            info["played_from_stock"] = result.info.played_from_stock

        return obs, reward, terminated, truncated, info

    def action_masks(self) -> np.ndarray:
        """Return the current legal action mask (for sb3_contrib MaskablePPO)."""
        return self._current_mask

    def close(self):
        """Clean up the gRPC channel."""
        if self.channel:
            self.channel.close()

