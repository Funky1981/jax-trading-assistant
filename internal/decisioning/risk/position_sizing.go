package risk

import "math"

func maxPositionRisk(context PortfolioContext) float64 {
	if !context.hasUsableEquity() {
		return 0
	}
	return context.AccountEquity * context.maxRiskPerTradePct()
}

func maxPositionSize(entry float64, stop float64, maxRisk float64) float64 {
	unitRisk := math.Abs(entry - stop)
	if unitRisk <= 0 || maxRisk <= 0 {
		return 0
	}
	return math.Floor(maxRisk / unitRisk)
}
