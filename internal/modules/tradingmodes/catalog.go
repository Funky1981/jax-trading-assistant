// Package tradingmodes provides a static catalog of supported trading modes
// and their associated strategy references, risk defaults, and required data.
package tradingmodes

// Catalog holds all available trading modes.
type Catalog struct {
	Modes []Mode `json:"modes"`
}

// Mode describes a trading mode: asset class, execution policy, universe,
// risk defaults, and the strategy types that belong to it.
type Mode struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	AssetClass      string        `json:"assetClass"`
	RuntimeMode     string        `json:"runtimeMode"`
	ExecutionPolicy string        `json:"executionPolicy"`
	Universe        []string      `json:"universe"`
	RequiredData    []string      `json:"requiredData"`
	RiskDefaults    RiskDefaults  `json:"riskDefaults"`
	Strategies      []StrategyRef `json:"strategies"`
}

// StrategyRef is a lightweight reference from a mode to a strategy type,
// including human-readable metadata and a seed default config.
type StrategyRef struct {
	StrategyTypeID string         `json:"strategyTypeId"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	DefaultConfig  map[string]any `json:"defaultConfig"`
}

// RiskDefaults holds mode-level risk parameters applied to all strategy instances
// created inside this mode.
type RiskDefaults struct {
	MaxTradesPerDay  int     `json:"maxTradesPerDay"`
	MaxOpenPositions int     `json:"maxOpenPositions"`
	RiskPerTradePct  float64 `json:"riskPerTradePct"`
	MinConfidence    float64 `json:"minConfidence"`
	FlattenBy        string  `json:"flattenBy"`
	ApprovalRequired bool    `json:"approvalRequired"`
}

// DefaultCatalog returns the built-in trading mode catalog.
// Modes are paper-only for phase 1; live trading is not supported.
func DefaultCatalog() Catalog {
	etfUniverse := []string{"SPY", "QQQ", "DIA", "IWM", "XLK", "XLF", "XLE", "SMH", "SOXX", "TLT", "GLD"}
	return Catalog{Modes: []Mode{
		{
			ID:              "etf_news_paper",
			Name:            "ETF News Paper",
			Description:     "Paper-only ETF strategies driven by confirmed news, macro context, chart structure, volatility, and ETF guardrails.",
			AssetClass:      "ETF",
			RuntimeMode:     "paper",
			ExecutionPolicy: "candidate_approval_only",
			Universe:        etfUniverse,
			RequiredData:    []string{"quotes", "candles_1m", "candles_5m", "news", "event_classification"},
			RiskDefaults: RiskDefaults{
				MaxTradesPerDay:  3,
				MaxOpenPositions: 1,
				RiskPerTradePct:  0.25,
				MinConfidence:    0.65,
				FlattenBy:        "15:55",
				ApprovalRequired: true,
			},
			Strategies: []StrategyRef{
				{
					StrategyTypeID: "etf_news_market_panic_reversal_v1",
					Name:           "Market Panic Reversal",
					Description:    "Looks for broad ETF rebound setups after panic news and price stabilization.",
					DefaultConfig: map[string]any{
						"symbols": []string{"SPY", "QQQ", "DIA", "IWM"},
						"parameters": map[string]any{
							"minDropPct":         1.2,
							"minConfirmations":   2,
							"minVolumeMultiple":  1.2,
							"stabilizationBars":  3,
							"atrStopMultiple":    1.1,
							"rewardRiskMultiple": 1.5,
						},
					},
				},
				{
					StrategyTypeID: "etf_news_sector_momentum_v1",
					Name:           "Sector News Momentum",
					Description:    "Maps confirmed sector news into plain ETF momentum candidates.",
					DefaultConfig: map[string]any{
						"symbols": []string{"QQQ", "XLK", "SMH", "SOXX", "XLE", "XLF", "IWM", "GLD"},
						"parameters": map[string]any{
							"minConfirmations":   2,
							"minMovePct":         0.4,
							"minVolumeMultiple":  1.2,
							"atrStopMultiple":    1.1,
							"rewardRiskMultiple": 1.5,
						},
					},
				},
				{
					StrategyTypeID: "etf_news_rates_bonds_rotation_v1",
					Name:           "Rates and Bonds Rotation",
					Description:    "Evaluates rates and inflation news for TLT, GLD, SPY, QQQ, and XLF candidates.",
					DefaultConfig: map[string]any{
						"symbols": []string{"TLT", "GLD", "SPY", "QQQ", "XLF"},
						"parameters": map[string]any{
							"minConfirmations":   2,
							"minMovePct":         0.35,
							"minVolumeMultiple":  1.1,
							"atrStopMultiple":    1.0,
							"rewardRiskMultiple": 1.5,
						},
					},
				},
			},
		},
		{
			ID:              "research_only",
			Name:            "Research Only",
			Description:     "Backtest and review strategies without live scanning or broker submission.",
			AssetClass:      "MULTI",
			RuntimeMode:     "research",
			ExecutionPolicy: "no_execution",
			Universe:        nil,
			RequiredData:    []string{"datasets"},
			RiskDefaults:    RiskDefaults{ApprovalRequired: true},
			Strategies:      nil,
		},
	}}
}

// Get returns the mode with the given ID, or false if not found.
func (c Catalog) Get(id string) (Mode, bool) {
	for _, mode := range c.Modes {
		if mode.ID == id {
			return mode, true
		}
	}
	return Mode{}, false
}
