package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/RyanMcCrary22/skipbo/engine"
)

// ---------------------------------------------------------------------------
// HumanPlayer — implements engine.Player via terminal I/O.
// ---------------------------------------------------------------------------

// HumanPlayer lets a human play Skip-Bo through the terminal.
type HumanPlayer struct {
	name    string
	scanner *bufio.Scanner
}

// NewHumanPlayer creates a new human player reading from stdin.
func NewHumanPlayer(name string) *HumanPlayer {
	return &HumanPlayer{
		name:    name,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (p *HumanPlayer) Name() string { return p.name }

// ChooseAction presents the game state and prompts the player for an action.
func (p *HumanPlayer) ChooseAction(view *engine.GameView) (engine.Action, error) {
	// Display current game state.
	fmt.Print(RenderGameView(view))

	for {
		fmt.Printf("%s> %s", colorGreen, colorReset)
		if !p.scanner.Scan() {
			return engine.Action{}, fmt.Errorf("input stream closed")
		}

		line := strings.TrimSpace(p.scanner.Text())
		if line == "" {
			continue
		}

		action, err := ParseCommand(line)
		if err != nil {
			fmt.Printf("  %s%s%s\n", colorRed, err, colorReset)
			continue
		}

		return action, nil
	}
}

// ---------------------------------------------------------------------------
// Command parsing
// ---------------------------------------------------------------------------

// ParseCommand parses a human-typed command string into an engine.Action.
//
// Supported formats:
//
//	play hand <idx> build <pile>
//	play stock build <pile>
//	play discard <idx> build <pile>
//	discard <hand_idx> <pile_idx>
//	help
//	quit
func ParseCommand(input string) (engine.Action, error) {
	parts := strings.Fields(strings.ToLower(input))
	if len(parts) == 0 {
		return engine.Action{}, fmt.Errorf("empty command")
	}

	switch parts[0] {
	case "help":
		fmt.Print(RenderHelp())
		return engine.Action{}, fmt.Errorf("(showing help, enter a command)")

	case "quit", "exit", "q":
		fmt.Println("Goodbye!")
		os.Exit(0)

	case "play":
		return parsePlayCommand(parts[1:])

	case "discard", "d":
		return parseDiscardCommand(parts[1:])

	default:
		return engine.Action{}, fmt.Errorf("unknown command %q — type 'help' for commands", parts[0])
	}

	return engine.Action{}, fmt.Errorf("unknown command")
}

// parsePlayCommand handles: play hand <i> build <j>, play stock build <j>,
// play discard <i> build <j>
func parsePlayCommand(parts []string) (engine.Action, error) {
	if len(parts) < 2 {
		return engine.Action{}, fmt.Errorf(
			"usage: play hand <idx> build <pile> | play stock build <pile> | play discard <idx> build <pile>")
	}

	switch parts[0] {
	case "hand", "h":
		// play hand <idx> build <pile>
		if len(parts) < 4 {
			return engine.Action{}, fmt.Errorf("usage: play hand <idx> build <pile>")
		}
		handIdx, err := strconv.Atoi(parts[1])
		if err != nil {
			return engine.Action{}, fmt.Errorf("invalid hand index: %s", parts[1])
		}
		if parts[2] != "build" && parts[2] != "b" {
			return engine.Action{}, fmt.Errorf("expected 'build' after hand index, got %q", parts[2])
		}
		buildIdx, err := strconv.Atoi(parts[3])
		if err != nil {
			return engine.Action{}, fmt.Errorf("invalid build pile index: %s", parts[3])
		}
		return engine.PlayFromHand(handIdx, buildIdx), nil

	case "stock", "s":
		// play stock build <pile>
		if len(parts) < 3 {
			return engine.Action{}, fmt.Errorf("usage: play stock build <pile>")
		}
		if parts[1] != "build" && parts[1] != "b" {
			return engine.Action{}, fmt.Errorf("expected 'build' after stock, got %q", parts[1])
		}
		buildIdx, err := strconv.Atoi(parts[2])
		if err != nil {
			return engine.Action{}, fmt.Errorf("invalid build pile index: %s", parts[2])
		}
		return engine.PlayFromStock(buildIdx), nil

	case "discard", "d":
		// play discard <idx> build <pile>
		if len(parts) < 4 {
			return engine.Action{}, fmt.Errorf("usage: play discard <idx> build <pile>")
		}
		discIdx, err := strconv.Atoi(parts[1])
		if err != nil {
			return engine.Action{}, fmt.Errorf("invalid discard pile index: %s", parts[1])
		}
		if parts[2] != "build" && parts[2] != "b" {
			return engine.Action{}, fmt.Errorf("expected 'build' after discard index, got %q", parts[2])
		}
		buildIdx, err := strconv.Atoi(parts[3])
		if err != nil {
			return engine.Action{}, fmt.Errorf("invalid build pile index: %s", parts[3])
		}
		return engine.PlayFromDiscard(discIdx, buildIdx), nil

	default:
		return engine.Action{}, fmt.Errorf("unknown source %q — use hand, stock, or discard", parts[0])
	}
}

// parseDiscardCommand handles: discard <hand_idx> <pile_idx>
func parseDiscardCommand(parts []string) (engine.Action, error) {
	if len(parts) < 2 {
		return engine.Action{}, fmt.Errorf("usage: discard <hand_idx> <pile_idx>")
	}

	handIdx, err := strconv.Atoi(parts[0])
	if err != nil {
		return engine.Action{}, fmt.Errorf("invalid hand index: %s", parts[0])
	}
	pileIdx, err := strconv.Atoi(parts[1])
	if err != nil {
		return engine.Action{}, fmt.Errorf("invalid discard pile index: %s", parts[1])
	}

	return engine.DiscardFromHand(handIdx, pileIdx), nil
}
