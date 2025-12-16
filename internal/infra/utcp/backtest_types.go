package utcp

import "time"

const (
	BacktestProviderID      = "backtest"
	ToolBacktestRunStrategy = "backtest.run_strategy"
	ToolBacktestGetRun      = "backtest.get_run"
)

type RunStrategyInput struct {
	StrategyConfigID string    `json:"strategyConfigId"`
	Symbols          []string  `json:"symbols"`
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
}

type BacktestStats struct {
	Trades      int     `json:"trades"`
	WinRate     float64 `json:"winRate"`
	AvgR        float64 `json:"avgR"`
	MaxDrawdown float64 `json:"maxDrawdown"`
	Sharpe      float64 `json:"sharpe"`
}

type RunStrategyOutput struct {
	RunID string        `json:"runId"`
	Stats BacktestStats `json:"stats"`
}

type GetRunInput struct {
	RunID string `json:"runId"`
}

type RunBySymbol struct {
	Symbol  string  `json:"symbol"`
	Trades  int     `json:"trades"`
	WinRate float64 `json:"winRate"`
}

type GetRunOutput struct {
	RunID    string        `json:"runId"`
	Stats    BacktestStats `json:"stats"`
	BySymbol []RunBySymbol `json:"bySymbol"`
}
