package backtest

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"jax-trading-assistant/libs/strategies"
)

// Config holds the configuration for a single deterministic backtest run.
type Config struct {
	StrategyName   string
	Symbols        []string
	StartDate      time.Time
	EndDate        time.Time
	DataSource     strategies.HistoricalDataSource
	Seed           int64
	InitialCapital float64
	RiskPerTrade   float64
}

// Result wraps backtest metrics with deterministic metadata.
type Result struct {
	strategies.BacktestResult
	Symbols    []string
	Seed       int64
	RunID      string
	RunAt      time.Time
	DurationMs int64
}

// Engine executes deterministic backtests over the shared strategies registry.
type Engine struct {
	registry *strategies.Registry
}

func New(registry *strategies.Registry) *Engine {
	return &Engine{registry: registry}
}

func (e *Engine) Run(ctx context.Context, cfg Config) (*Result, error) {
	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
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

	runAt := time.Now().UTC()
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
