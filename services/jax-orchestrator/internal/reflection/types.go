package reflection

import (
	"time"

	"jax-trading-assistant/libs/contracts"
)

const (
	DecisionsBank = "trade_decisions"
	OutcomesBank  = "trade_outcomes"
	BeliefsBank   = "strategy_beliefs"
	SourceSystem  = "jax-reflection"
)

type Window struct {
	From time.Time
	To   time.Time
}

type RunConfig struct {
	WindowDays int
	MaxItems   int
	To         time.Time
	DryRun     bool
}

type RunResult struct {
	Beliefs     int
	Retained    int
	Window      Window
	BeliefItems []contracts.MemoryItem
}
