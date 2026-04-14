# AI Skip-Bo

Welcome to AI Skip-Bo! This project is a digital playground for the classic card game Skip-Bo ([learn more here](https://service.mattel.com/instruction_sheets/42050.pdf)), a game I recently picked up from a friend. After getting hooked, I decided to put my new AI knowledge to the test by building a complete game engine and training an intelligent agent to master it. 

This repository contains everything from a highly optimized game engine to the complete reinforcement learning infrastructure used to train the AI.

So far I have used this relatively simple learning set up to train an agent against a the random player defined in `engine/random_player.go` and have capped out at a win rate around 50%. For a game based so heavily on chance, I think this is promising and more advanced learning strategies including 3+ player games and self training against learned models could take the agent further. 

Next steps are to use onnx, to integrate learned model players into the Go side for my agent to train against.

## Project Architecture

This project is built across multiple layers to efficiently separate the game logic from the reinforcement learning training process:

- **Game Engine (Go)**: A high-speed, thread-safe game engine written in Go. It handles all the state logic, move validation, card conservation, and rule invariants. The Go engine provides the lightning-fast simulation execution required to train agents over millions of games.
- **Training Server & gRPC Connection (Go)**: The vital bridge between the Python machine learning stack and the Go game engine. It exposes the game environment to Python via a gRPC connection (defined using Protocol Buffers), managing training sessions and securely running ONNX model inference for self-play opponents.
- **AI Agent Training (Python)**: The reinforcement learning side. Utilizing Python and Stable-Baselines3 (`MaskablePPO`), these scripts wrap the gRPC client to interact with the game environment, applying action masking to maintain valid play and logging metrics robustly for training analysis.

## Getting Started

### Clone the Repository
Start by cloning the repository locally:
```bash
git clone https://github.com/RyanMcCrary22/skipbo.git
cd skipbo
```

### Install Dependencies
1. **Python Virtual Environment**: Set up the environment for the training scripts.
   ```bash
   python -m venv .venv
   source .venv/bin/activate
   pip install -r python/requirements.txt
   ```
2. **Go Modules**: Download Go dependencies securely.
   ```bash
   go mod download
   ```

## What Can You Run?

The repository has quite a few features implemented directly as easily executable `Makefile` targets. 

### Play the Game!
You can play a game interactively against random players out of the box:
- **`make play-gui`**: Opens a visual Ebitengine-based Graphical User Interface. Features a felt table and card rendering. Bots delay 2 seconds between moves so you can follow the action!
- **`make play-cli`**: Opens a straightforward un-rendered ASCII game in your terminal playing against a bot.

### Watch the Math
- **`make simulate`**: Run a headless simulation to spectate 4 AI agents playing a game against each other entirely autonomously as fast as the engine allows.

### Train the AI
If you wish to initiate the reinforcement learning stack and train the agent:
1. **Start the gRPC Server**: Keep the Go engine open to serve requests.
   ```bash
   make train-server
   ```
2. **Execute Python Training**: In a separate terminal, launch the main script:
   ```bash
   source .venv/bin/activate
   make train STEPS=500000
   ```
*(You can also pass arguments via the make command like `LOAD=path/to/model` or specific amounts of `PROCS` as needed.)*

---

Take a look around the core logic inside the `engine/` namespace, dive into the agent tuning within `python/train.py`, or simply play the game yourself!
