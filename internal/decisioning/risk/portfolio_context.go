package risk

import "time"

type AccountMode string

const (
	AccountModeResearch AccountMode = "research"
	AccountModePaper    AccountMode = "paper"
	AccountModeLive     AccountMode = "live"
)

const defaultMaxRiskPerTradePct = 0.01

type PortfolioContext struct {
	AccountEquity      float64   `json:"account_equity"`
	CashAvailable      float64   `json:"cash_available"`
	MaxRiskPerTradePct float64   `json:"max_risk_per_trade_pct"`
	AsOf               time.Time `json:"as_of"`
}

func (context PortfolioContext) maxRiskPerTradePct() float64 {
	if context.MaxRiskPerTradePct > 0 {
		return context.MaxRiskPerTradePct
	}
	return defaultMaxRiskPerTradePct
}

func (context PortfolioContext) hasUsableEquity() bool {
	return context.AccountEquity > 0
}
