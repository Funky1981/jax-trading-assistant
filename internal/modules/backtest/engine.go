package backtest

import (
	"context"

	shared "jax-trading-assistant/libs/backtest"
	"jax-trading-assistant/libs/strategies"
)

type Config = shared.Config
type Result = shared.Result

// Engine is a thin orchestration wrapper around the shared libs/backtest engine.
type Engine struct {
	inner *shared.Engine
}

func New(registry *strategies.Registry) *Engine {
	return &Engine{inner: shared.New(registry)}
}

func (e *Engine) Run(ctx context.Context, cfg Config) (*Result, error) {
	return e.inner.Run(ctx, cfg)
}
