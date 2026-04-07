.PHONY: test test-race test-unit bench play-gui play-cli simulate proto train train-server eval export-onnx

# Run all the unit and integration tests
test:
	go test ./... -v

# Run tests with the race detector enabled
test-race:
	go test -race ./... -v

# Run only unit tests (skip integration tests)
test-unit:
	go test ./engine/... -v -run 'Test[^I]'

# Run benchmarks
bench:
	go test ./engine/... -bench=. -benchmem -run=^$$

# Play against 1 bot in the Graphical Interface (Felt Table + Cards)
# Bots wait 2 seconds between moves so you can see what they're doing.
play-gui:
	go run ./cmd/skipbo/ --mode gui --humans 1 --players 2 --stock 10 --bot-delay 2s

# Play against 1 bot in the Command Line Interface (Terminal ASCII)
play-cli:
	go run ./cmd/skipbo/ --mode cli --humans 1 --players 2 --stock 10

# Watch 4 AI agents play against each other at lightning speed
simulate:
	go run ./cmd/skipbo/ --mode headless --humans 0 --players 4 --stock 30

# ---------------------------------------------------------------------------
# RL Training
# ---------------------------------------------------------------------------

# Regenerate gRPC stubs (Go + Python) from proto/skipbo.proto
proto:
	protoc --go_out=proto/skipbopb --go_opt=paths=source_relative \
	       --go-grpc_out=proto/skipbopb --go-grpc_opt=paths=source_relative \
	       -I proto proto/skipbo.proto
	.venv/bin/python -m grpc_tools.protoc --python_out=python/ --grpc_python_out=python/ \
	       -I proto proto/skipbo.proto

# Start the gRPC training server (Go)
train-server:
	go run ./cmd/train/ --port 50051

# Run PPO training (Python) — start the server first with `make train-server`
# Optional: Set total steps with STEPS=... (default: 500000)
# Optional: Set number of processes with PROCS=... (default: 1)
# Optional: Resume training from an existing model with `make train LOAD=path/to/model.zip`
STEPS ?= 500000
PROCS ?= 1
OPPONENTS ?= 1
OPPONENT_MODELS ?= ""
train:
	@if [ -n "$(LOAD)" ]; then \
		cd python && ../.venv/bin/python train.py --timesteps $(STEPS) --procs $(PROCS) --opponents $(OPPONENTS) --opponent-models $(OPPONENT_MODELS) --load $(LOAD); \
	else \
		cd python && ../.venv/bin/python train.py --timesteps $(STEPS) --procs $(PROCS) --opponents $(OPPONENTS) --opponent-models $(OPPONENT_MODELS); \
	fi

# Evaluate a trained model — pass MODEL=path/to/model.zip
eval:
	cd python && ../.venv/bin/python evaluate.py --model $(MODEL) --games 500

# Export a trained model to ONNX format
export-onnx:
	cd python && ../.venv/bin/python export_onnx.py --model ../models/v4_28M.zip --output ../models/v4_28M.onnx
