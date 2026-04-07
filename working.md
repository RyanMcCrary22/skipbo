# Skip-Bo RL Project: State Summary & Handover

This document summarizes the current state of the Skip-Bo reinforcement learning project for use when resuming a session.

## Project Goal
Develop a high-performance, scalable Skip-Bo training infrastructure using a Go-based game engine and Python-based RL agents (Stable-Baselines3).

## Current Architecture
- **Engine (Go)**: High-speed game logic with invariant testing (card conservation, wild-card tracking).
- **Inference (Go)**: Native `OnnxPlayer` using `onnxruntime` for self-play without Python round-trips for opponents.
- **Training (Python)**: `MaskablePPO` (sb3-contrib) wrapping a gRPC client to the Go server.
- **Communication**: gRPC defined in `proto/skipbo.proto`.

## Recent Accomplishments
1. **Memory Stability**: Fixed a critical leak in `training/server.go`. Zombie sessions are now reaped after 10 minutes of inactivity using a background goroutine and `context.CancelFunc`.
2. **ONNX Self-Play**: Updated the gRPC protocol and training scripts to allow agents to play against specific older models (e.g., `v4_28M.onnx`) by filename.
3. **Linker Resolution**: Successfully linked the Go server to `libonnxruntime.dylib` on macOS (installed via Homebrew at `/opt/homebrew/lib/`).

## How to Run
### 1. Start Go Training Server
```bash
go run ./cmd/train/ --onnx-lib /opt/homebrew/lib/libonnxruntime.dylib
```
### 2. Run Training against ONNX Opponents
```bash
make train STEPS=500000 OPPONENT_MODELS=random,v4_28M.onnx
```

## Immediate Next Step (High Priority)
### **Model Caching Layer**
Currently, the Go server reloads the `.onnx` model from disk **every single game** during the `Reset` call. This will bottleneck high-volume training.
- **Plan**: Modify `training/server.go` to include a thread-safe cache (`map[string]*OnnxPlayer`) for shared model sessions.
- **Action**: Grab the pointer from the cache instead of initializing a new session if the model has already been loaded once.

## Files of Interest
- `training/server.go`: Core gRPC server logic and session management.
- `training/onnx_player.go`: Go implementation of the ONNX agent.
- `python/train.py`: Main training orchestration.
- `python/skipbo_env.py`: Gymnasium wrapper.
- `proto/skipbo.proto`: API definitions.

## Context for Next Agent
The project is stabilized and ready for performance optimization. We just verified that a 2000-step run works with an ONNX opponent, but disk I/O will be the next bottleneck without the cache.
