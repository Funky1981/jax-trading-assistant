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
	DatasetID        string    `json:"datasetId,omitempty"`
	Seed             int64     `json:"seed,omitempty"`
	InitialCapital   float64   `json:"initialCapital,omitempty"`
	RiskPerTrade     float64   `json:"riskPerTrade,omitempty"`
	InstanceID       string    `json:"instanceId,omitempty"`
	FlowID           string    `json:"flowId,omitempty"`
}

type BacktestStats struct {
	Trades       int     `json:"trades"`
	WinRate      float64 `json:"winRate"`
	AvgR         float64 `json:"avgR"`
	MaxDrawdown  float64 `json:"maxDrawdown"`
	Sharpe       float64 `json:"sharpe"`
	FinalCapital float64 `json:"finalCapital,omitempty"`
	TotalReturn  float64 `json:"totalReturn,omitempty"`
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
	RunID       string          `json:"runId"`
	Stats       BacktestStats   `json:"stats"`
	BySymbol    []RunBySymbol   `json:"bySymbol"`
	StartedAt   time.Time       `json:"startedAt,omitempty"`
	CompletedAt time.Time       `json:"completedAt,omitempty"`
	Trades      []BacktestTrade `json:"trades,omitempty"`
}

type BacktestTrade struct {
	Symbol     string    `json:"symbol"`
	Direction  string    `json:"direction"`
	EntryDate  time.Time `json:"entryDate"`
	ExitDate   time.Time `json:"exitDate"`
	EntryPrice float64   `json:"entryPrice"`
	ExitPrice  float64   `json:"exitPrice"`
	Quantity   int       `json:"quantity"`
	PnL        float64   `json:"pnl"`
	PnLPct     float64   `json:"pnlPct"`
	RMultiple  float64   `json:"rMultiple"`
	ExitReason string    `json:"exitReason"`
}
