package domain

// PortfolioConstraints defines risk limits at the portfolio level
type PortfolioConstraints struct {
	// MaxPositionSize is the maximum dollar amount for a single position
	MaxPositionSize float64 `json:"max_position_size"`

	// MaxPositions is the maximum number of concurrent open positions
	MaxPositions int `json:"max_positions"`

	// MaxSectorExposure is the maximum exposure to a single sector (as % of portfolio)
	MaxSectorExposure float64 `json:"max_sector_exposure"`

	// MaxCorrelatedExposure limits exposure to highly correlated assets
	MaxCorrelatedExposure float64 `json:"max_correlated_exposure"`

	// MaxPortfolioRisk is the maximum total portfolio risk (as % of account)
	MaxPortfolioRisk float64 `json:"max_portfolio_risk"`

	// MaxDrawdown is the maximum acceptable drawdown before halting trading
	MaxDrawdown float64 `json:"max_drawdown"`

	// MinAccountSize is the minimum account size required to continue trading
	MinAccountSize float64 `json:"min_account_size"`
}

// PositionLimits defines limits for individual positions
type PositionLimits struct {
	// MaxRiskPerTrade is the maximum risk per trade (as % of account)
	MaxRiskPerTrade float64 `json:"max_risk_per_trade"`

	// MinRiskPerTrade is the minimum risk per trade (to avoid dust trades)
	MinRiskPerTrade float64 `json:"min_risk_per_trade"`

	// MaxLeverage is the maximum leverage allowed
	MaxLeverage float64 `json:"max_leverage"`

	// MinStopDistance is the minimum stop distance (as % of entry price)
	MinStopDistance float64 `json:"min_stop_distance"`

	// MaxStopDistance is the maximum stop distance (as % of entry price)
	MaxStopDistance float64 `json:"max_stop_distance"`
}

// RiskSizingModel defines different position sizing strategies
type RiskSizingModel string

const (
	// FixedFractional risks a fixed percentage of account per trade
	FixedFractional RiskSizingModel = "fixed_fractional"

	// FixedRatio increases position size as profits accumulate
	FixedRatio RiskSizingModel = "fixed_ratio"

	// KellyCriterion uses the Kelly formula for optimal bet sizing
	KellyCriterion RiskSizingModel = "kelly_criterion"

	// VolatilityAdjusted scales position size based on asset volatility
	VolatilityAdjusted RiskSizingModel = "volatility_adjusted"
)

// PortfolioState represents the current state of the portfolio
type PortfolioState struct {
	AccountSize     float64            `json:"account_size"`
	Cash            float64            `json:"cash"`
	EquityValue     float64            `json:"equity_value"`
	OpenPositions   int                `json:"open_positions"`
	TotalExposure   float64            `json:"total_exposure"`
	TotalRisk       float64            `json:"total_risk"`
	SectorExposure  map[string]float64 `json:"sector_exposure"`
	CurrentDrawdown float64            `json:"current_drawdown"`
	PeakEquity      float64            `json:"peak_equity"`
}

// RiskCheckResult contains the result of a risk validation check
type RiskCheckResult struct {
	Allowed      bool     `json:"allowed"`
	Reason       string   `json:"reason"`
	Violations   []string `json:"violations"`
	RiskMetrics  RiskMetrics `json:"risk_metrics"`
}

// RiskMetrics contains calculated risk metrics
type RiskMetrics struct {
	PositionRisk    float64 `json:"position_risk"`     // Risk for this specific position
	PortfolioRisk   float64 `json:"portfolio_risk"`    // Total portfolio risk including this position
	PositionSize    int     `json:"position_size"`     // Number of shares/contracts
	DollarRisk      float64 `json:"dollar_risk"`       // Dollar amount at risk
	RiskPerUnit     float64 `json:"risk_per_unit"`     // Risk per share/contract
	Leverage        float64 `json:"leverage"`          // Position leverage
	StopDistance    float64 `json:"stop_distance"`     // Stop distance as % of entry
	SectorExposure  float64 `json:"sector_exposure"`   // Sector exposure after this trade
}
