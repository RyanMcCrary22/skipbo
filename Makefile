.PHONY: test play-gui play-cli simulate

# Run all the unit and integration tests
test:
	go test ./... -v

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
