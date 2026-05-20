// Package etfnews provides ETF-specific news classification and symbol mapping.
// It classifies free-text news into market events (panic, sector, rates/inflation)
// and maps each class to the relevant phase-1 ETF symbols.
package etfnews

import "strings"

// Classification is the result of classifying a news event for ETF relevance.
type Classification struct {
	Class       string   `json:"class"`
	Sector      string   `json:"sector,omitempty"`
	Sentiment   string   `json:"sentiment"`
	Materiality string   `json:"materiality"`
	Symbols     []string `json:"symbols"`
	Tags        []string `json:"tags"`
}

// Classify attempts to classify a news event by its title and summary.
// It returns (Classification, true) when the event is ETF-relevant,
// and (Classification{}, false) when no ETF mapping applies.
func Classify(title, summary string) (Classification, bool) {
	text := strings.ToLower(strings.TrimSpace(title + " " + summary))

	switch {
	case containsAny(text, "panic", "selloff", "sell-off", "risk-off", "crash", "market rout", "circuit breaker"):
		return Classification{
			Class:       "market_panic",
			Sentiment:   "bearish",
			Materiality: "high",
			Symbols:     []string{"SPY", "QQQ", "DIA", "IWM"},
			Tags:        []string{"broad_market", "risk_off"},
		}, true

	case containsAny(text, "semiconductor", "chip", "chips", "ai accelerator", "foundry", "tsmc", "nvidia", "amd gpu"):
		return Classification{
			Class:       "sector_news",
			Sector:      "semiconductors",
			Sentiment:   sentimentFromText(text),
			Materiality: "medium",
			Symbols:     []string{"SMH", "SOXX", "XLK"},
			Tags:        []string{"sector", "semiconductors"},
		}, true

	case containsAny(text, "artificial intelligence", "ai spending", "ai investment", "hyperscaler", "cloud capex"):
		return Classification{
			Class:       "sector_news",
			Sector:      "technology",
			Sentiment:   sentimentFromText(text),
			Materiality: "medium",
			Symbols:     []string{"QQQ", "XLK", "SMH"},
			Tags:        []string{"sector", "technology", "ai"},
		}, true

	case containsAny(text, "oil", "crude", "opec", "energy supply", "natural gas", "refinery"):
		return Classification{
			Class:       "sector_news",
			Sector:      "energy",
			Sentiment:   sentimentFromText(text),
			Materiality: "medium",
			Symbols:     []string{"XLE"},
			Tags:        []string{"sector", "energy"},
		}, true

	case containsAny(text, "bank", "banks", "banking", "credit stress", "regional lender", "svb", "fdic", "systemic risk"):
		return Classification{
			Class:       "sector_news",
			Sector:      "financials",
			Sentiment:   sentimentFromText(text),
			Materiality: "medium",
			Symbols:     []string{"XLF"},
			Tags:        []string{"sector", "financials"},
		}, true

	case containsAny(text, "inflation", "cpi", "pce", "core inflation", "treasury yield", "10-year yield",
		"rate cut", "rate hike", "federal reserve", " fed ", "fomc", "central bank", "bond yield",
		"recession fear", "yield curve", "boe rate", "ecb rate"):
		return Classification{
			Class:       "rates_inflation",
			Sentiment:   sentimentFromText(text),
			Materiality: "high",
			Symbols:     []string{"TLT", "GLD", "SPY", "QQQ", "XLF"},
			Tags:        []string{"macro", "rates"},
		}, true

	case containsAny(text, "gold", "safe haven", "geopolitical risk", "war risk", "dollar weakness"):
		return Classification{
			Class:       "macro_risk",
			Sentiment:   sentimentFromText(text),
			Materiality: "medium",
			Symbols:     []string{"GLD", "TLT"},
			Tags:        []string{"macro", "safe_haven"},
		}, true
	}

	return Classification{}, false
}

// sentimentFromText returns "positive", "negative", or "neutral" based on keyword matching.
func sentimentFromText(text string) string {
	if containsAny(text, "beat", "surge", "rally", "strong", "approved", "cut rates", "rate cut",
		"better than expected", "outperform", "rebound", "recovery") {
		return "positive"
	}
	if containsAny(text, "miss", "weak", "falls", "drops", "higher yields", "rate hike",
		"inflation hot", "worse than expected", "underperform", "selloff", "sell-off") {
		return "negative"
	}
	return "neutral"
}

// containsAny returns true if text contains any of the given needle strings.
func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
