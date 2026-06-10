package macroevents

import (
	"fmt"
	"strings"

	"jax-trading-assistant/internal/modules/instruments"
)

func ValidateAndNormalizeETFs(symbols []string) ([]ETFMapping, error) {
	normalized := dedupeSymbols(symbols)
	if len(normalized) == 0 {
		return nil, fmt.Errorf("affected_etfs are required")
	}

	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		return nil, fmt.Errorf("ETF catalog unavailable: %w", err)
	}

	mappings := make([]ETFMapping, 0, len(normalized))
	for _, symbol := range normalized {
		if !catalog.IsKnownETF(symbol) {
			return nil, fmt.Errorf("ETF mapping rejected: %s is not an allowed ETF", symbol)
		}
		evaluation := catalog.Evaluate(symbol, "paper")
		if !evaluation.Allowed {
			return nil, fmt.Errorf("ETF mapping rejected: %s: %s", symbol, evaluation.ReasonCode)
		}
		mappings = append(mappings, ETFMapping{
			Symbol:        symbol,
			Theme:         themeForSymbol(symbol),
			MappingReason: "Phase 1 macro ETF allowlist mapping for " + symbol,
			Confidence:    1,
		})
	}
	return mappings, nil
}

func dedupeSymbols(symbols []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(symbols))
	for _, raw := range symbols {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol == "" || seen[symbol] {
			continue
		}
		seen[symbol] = true
		out = append(out, symbol)
	}
	return out
}

func themeForSymbol(symbol string) string {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "QQQ":
		return "growth/technology"
	case "SPY":
		return "broad_market"
	case "IWM":
		return "small_caps"
	case "TLT":
		return "rates_duration"
	case "XLF":
		return "financials"
	case "XLE":
		return "energy"
	case "GLD":
		return "safe_haven"
	default:
		return "macro"
	}
}
