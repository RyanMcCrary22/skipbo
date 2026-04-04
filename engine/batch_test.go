package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeConfigs(n int, stockSize int) []GameConfig {
	configs := make([]GameConfig, n)
	for i := range configs {
		configs[i] = GameConfig{
			NumPlayers: 2,
			StockSize:  stockSize,
			Seed:       uint64(i + 1),
		}
	}
	return configs
}

func randomFactory(gameIndex int) []Player {
	return []Player{
		NewRandomPlayer("A", uint64(gameIndex*2)),
		NewRandomPlayer("B", uint64(gameIndex*2+1)),
	}
}

// ---------------------------------------------------------------------------
// Sequential batch
// ---------------------------------------------------------------------------

func TestRunBatch_AllGamesComplete(t *testing.T) {
	configs := makeConfigs(20, 10)
	results := RunBatch(configs, randomFactory)

	require.Len(t, results, 20)
	for i, r := range results {
		assert.NoError(t, r.Err, "game %d", i)
		assert.True(t, r.Winner == 0 || r.Winner == 1, "game %d: winner=%d", i, r.Winner)
		assert.Greater(t, r.TurnCount, 0, "game %d: should have >0 turns", i)
		assert.Equal(t, uint64(i+1), r.Seed, "game %d: seed preserved", i)
	}
}

func TestRunBatch_Empty(t *testing.T) {
	results := RunBatch(nil, randomFactory)
	assert.Len(t, results, 0)
}

func TestRunBatch_InvalidConfig(t *testing.T) {
	configs := []GameConfig{{NumPlayers: 1, StockSize: 30, Seed: 1}} // invalid: 1 player
	results := RunBatch(configs, randomFactory)
	require.Len(t, results, 1)
	assert.Error(t, results[0].Err)
	assert.Equal(t, -1, results[0].Winner)
}

// ---------------------------------------------------------------------------
// Parallel batch
// ---------------------------------------------------------------------------

func TestRunBatchParallel_AllGamesComplete(t *testing.T) {
	configs := makeConfigs(50, 10)
	results := RunBatchParallel(configs, randomFactory, 4)

	require.Len(t, results, 50)
	for i, r := range results {
		assert.NoError(t, r.Err, "game %d", i)
		assert.True(t, r.Winner == 0 || r.Winner == 1, "game %d", i)
	}
}

func TestRunBatchParallel_MatchesSequential(t *testing.T) {
	configs := makeConfigs(10, 10)

	seqResults := RunBatch(configs, randomFactory)
	parResults := RunBatchParallel(configs, randomFactory, 4)

	require.Len(t, seqResults, len(parResults))
	for i := range seqResults {
		assert.Equal(t, seqResults[i].Winner, parResults[i].Winner,
			"game %d: parallel winner should match sequential", i)
		assert.Equal(t, seqResults[i].TurnCount, parResults[i].TurnCount,
			"game %d: turn counts should match", i)
	}
}

func TestRunBatchParallel_ZeroConcurrency(t *testing.T) {
	configs := makeConfigs(5, 10)
	results := RunBatchParallel(configs, randomFactory, 0) // defaults to len(configs)

	require.Len(t, results, 5)
	for i, r := range results {
		assert.NoError(t, r.Err, "game %d", i)
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkRunBatch(b *testing.B) {
	configs := makeConfigs(100, 10)
	for i := 0; i < b.N; i++ {
		RunBatch(configs, randomFactory)
	}
}

func BenchmarkRunBatchParallel(b *testing.B) {
	configs := makeConfigs(100, 10)
	for i := 0; i < b.N; i++ {
		RunBatchParallel(configs, randomFactory, 8)
	}
}
