// Package backtest provides a deterministic backtest engine for the research runtime.
// It wraps libs/strategies.Backtester and records the seed used so results can be
// reproduced exactly — a requirement for the ADR-0012 artifact validation gate.
package backtest

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"jax-trading-assistant/libs/strategies"
)

// Config holds the configuration for a single backtest run.
type Config struct {
	// StrategyName must match a strategy registered in the Registry.
	StrategyName string
	// Symbols is the list of ticker symbols to back-test.
	Symbols []string
	// Date range for the back-test.
	StartDate time.Time
	EndDate   time.Time
	// DataSource provides historical candles and indicators.
	DataSource strategies.HistoricalDataSource
	// Seed makes execution deterministic. 0 = auto-generate from wall clock.
	Seed int64
	// InitialCapital in USD; defaults to 100 000 when zero.
	InitialCapital float64
	// RiskPerTrade as a fraction (e.g. 0.01 = 1 %); defaults to 0.01 when zero.
	RiskPerTrade float64
}

// Result embeds the full strategies.BacktestResult and adds metadata required
// for artifact creation (seed, run ID, timing).
type Result struct {
	strategies.BacktestResult
	// Symbols is the list of symbols that were tested. Preserved separately
	// because BacktestResult.Symbol only tracks the last symbol processed.
	Symbols []string
	// Seed that was used. Record this in the artifact for deterministic replay.
	Seed int64
	// RunID is a human-readable identifier for this specific execution.
	RunID string
	// RunAt is the wall-clock time when Run() was called.
	RunAt time.Time
	// DurationMs is how long the backtest took.
	DurationMs int64
}

// Engine wraps libs/strategies.Backtester with determinism seed tracking.
type Engine struct {
	registry *strategies.Registry
}

// New creates a new backtest Engine backed by the given strategy Registry.
func New(registry *strategies.Registry) *Engine {
	return &Engine{registry: registry}
}

// Run executes a deterministic backtest.
//
// The seed is stored in Result.Seed and must be preserved in the artifact's
// ValidationInfo so that any later replay run produces identical trades.
// If cfg.Seed is 0, a seed is chosen from the wall clock.
func (e *Engine) Run(ctx context.Context, cfg Config) (*Result, error) {
	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	// Set global rand seed so strategy internals that use math/rand are deterministic.
	// In future phases this can be replaced with a per-context rand source.
	//nolint:gosec
	rand.Seed(seed) //nolint:staticcheck

	capital := cfg.InitialCapital
	if capital <= 0 {
		capital = 100_000
	}
	risk := cfg.RiskPerTrade
	if risk <= 0 {
		risk = 0.01
	}

	bt := strategies.NewBacktester(e.registry).
		WithCapital(capital).
		WithRiskPerTrade(risk)

	runAt := time.Now()
	inner, err := bt.Run(ctx, strategies.BacktestConfig{
		StrategyID: cfg.StrategyName,
		Symbols:    cfg.Symbols,
		StartDate:  cfg.StartDate,
		EndDate:    cfg.EndDate,
		DataSource: cfg.DataSource,
	})
	if err != nil {
		return nil, fmt.Errorf("backtest run failed for strategy %q: %w", cfg.StrategyName, err)
	}

	return &Result{
		BacktestResult: *inner,
		Symbols:        cfg.Symbols,
		Seed:           seed,
		RunID:          fmt.Sprintf("bt_%s_%d", cfg.StrategyName, seed),
		RunAt:          runAt,
		DurationMs:     time.Since(runAt).Milliseconds(),
	}, nil
}
