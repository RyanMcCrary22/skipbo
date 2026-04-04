package engine

import (
	"sync"
)

// ---------------------------------------------------------------------------
// Batch Runner — efficient multi-game execution for RL training loops
// ---------------------------------------------------------------------------

// BatchResult holds the outcome of a single game.
type BatchResult struct {
	Winner    int    // Index of the winning player.
	TurnCount int    // Total number of turns in the game.
	Seed      uint64 // The seed used for this game (for reproducibility).
	Err       error  // Non-nil if the game encountered an error.
}

// PlayerFactory creates a fresh set of players for a new game.
// The argument is the game index within the batch (useful for varying seeds).
type PlayerFactory func(gameIndex int) []Player

// RunBatch runs a series of games sequentially and returns their results.
// Each game is constructed from the config at the corresponding index.
// The playerFactory is called once per game to create fresh player instances.
func RunBatch(configs []GameConfig, factory PlayerFactory) []BatchResult {
	results := make([]BatchResult, len(configs))
	for i, cfg := range configs {
		results[i] = runSingleGame(cfg, factory, i)
	}
	return results
}

// RunBatchParallel runs games in parallel with the given concurrency limit.
// Setting concurrency to 0 or negative defaults to len(configs).
func RunBatchParallel(configs []GameConfig, factory PlayerFactory, concurrency int) []BatchResult {
	if concurrency <= 0 {
		concurrency = len(configs)
	}

	results := make([]BatchResult, len(configs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, cfg := range configs {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore slot.

		go func(idx int, c GameConfig) {
			defer wg.Done()
			defer func() { <-sem }() // Release slot.

			results[idx] = runSingleGame(c, factory, idx)
		}(i, cfg)
	}

	wg.Wait()
	return results
}

// runSingleGame runs one game and returns the result.
func runSingleGame(cfg GameConfig, factory PlayerFactory, gameIndex int) BatchResult {
	players := factory(gameIndex)

	game, err := NewGame(cfg, players)
	if err != nil {
		return BatchResult{Winner: -1, Seed: cfg.Seed, Err: err}
	}

	winner, err := game.Run()
	return BatchResult{
		Winner:    winner,
		TurnCount: game.TurnNumber(),
		Seed:      cfg.Seed,
		Err:       err,
	}
}
