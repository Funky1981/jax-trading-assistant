package research

type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func (r DateRange) IsDefined() bool {
	return r.Start != "" && r.End != ""
}

type SlippageModel struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func (m SlippageModel) IsDefined() bool {
	return m.Type != "" && m.Value > 0
}

type FeesModel struct {
	Commission       string `json:"commission"`
	SpreadAssumption string `json:"spread_assumption"`
}

func (m FeesModel) IsDefined() bool {
	return m.Commission != "" && m.SpreadAssumption != ""
}

type BacktestAssumptions struct {
	Execution       string  `json:"execution"`
	PositionSizing  string  `json:"position_sizing"`
	MaxRiskPerTrade float64 `json:"max_risk_per_trade"`
}

func (a BacktestAssumptions) IsDefined() bool {
	return a.Execution != "" && a.PositionSizing != "" && a.MaxRiskPerTrade > 0
}

type PerformanceMetrics struct {
	TotalReturn      float64 `json:"total_return"`
	AnnualisedReturn float64 `json:"annualised_return"`
	Sharpe           float64 `json:"sharpe"`
	Sortino          float64 `json:"sortino"`
	ProfitFactor     float64 `json:"profit_factor"`
}

type DrawdownMetrics struct {
	MaxDrawdown     float64 `json:"max_drawdown"`
	AverageDrawdown float64 `json:"average_drawdown"`
}

func (m DrawdownMetrics) IsDefined() bool {
	return m.MaxDrawdown != 0 || m.AverageDrawdown != 0
}

type BacktestEvidence struct {
	HypothesisID              string              `json:"hypothesis_id"`
	SetupFamily               string              `json:"setup_family"`
	DatasetID                 string              `json:"dataset_id"`
	DatasetHash               string              `json:"dataset_hash"`
	DateRange                 DateRange           `json:"date_range"`
	InstrumentUniverse        []string            `json:"instrument_universe"`
	Benchmark                 string              `json:"benchmark"`
	Assumptions               BacktestAssumptions `json:"assumptions"`
	SlippageModel             SlippageModel       `json:"slippage_model"`
	FeesModel                 FeesModel           `json:"fees_model"`
	InSamplePeriod            DateRange           `json:"in_sample_period"`
	OutOfSamplePeriod         DateRange           `json:"out_of_sample_period"`
	OutOfSampleLimitationNote string              `json:"out_of_sample_limitation_note,omitempty"`
	PerformanceMetrics        PerformanceMetrics  `json:"performance_metrics"`
	DrawdownMetrics           DrawdownMetrics     `json:"drawdown_metrics"`
	WinRate                   float64             `json:"win_rate"`
	AverageWinLoss            float64             `json:"average_win_loss"`
	Expectancy                float64             `json:"expectancy"`
	SampleSize                int                 `json:"sample_size"`
	FailureModes              []string            `json:"failure_modes"`
	RiskRules                 []string            `json:"risk_rules,omitempty"`
	PromotionDecision         PromotionState      `json:"promotion_decision"`
}

func (e BacktestEvidence) HasOutOfSampleEvidence() bool {
	return e.OutOfSamplePeriod.IsDefined()
}

func (e BacktestEvidence) HasOutOfSampleLimitation() bool {
	return e.OutOfSampleLimitationNote != ""
}

func (e BacktestEvidence) HasDefinedRiskRules() bool {
	return len(e.RiskRules) > 0
}
