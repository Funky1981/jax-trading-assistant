package converters

import (
	"strings"

	"jax-trading-assistant/libs/contracts/domain"
	"jax-trading-assistant/libs/strategies"
)

// SignalToDomain converts a strategies.Signal to domain.Signal
func SignalToDomain(strategyID string, sig strategies.Signal) domain.Signal {
	return domain.Signal{
		ID:         "", // Will be set by caller (e.g., UUID)
		Symbol:     sig.Symbol,
		Timestamp:  sig.Timestamp,
		Type:       strings.ToUpper(string(sig.Type)), // Database expects uppercase
		Confidence: sig.Confidence,
		EntryPrice: sig.EntryPrice,
		StopLoss:   sig.StopLoss,
		TakeProfit: sig.TakeProfit,
		Reason:     sig.Reason,
		StrategyID: strategyID,
		Indicators: sig.Indicators,
	}
}

// SignalFromDomain converts a domain.Signal back to strategies.Signal
func SignalFromDomain(sig domain.Signal) strategies.Signal {
	return strategies.Signal{
		Type:       strategies.SignalType(strings.ToLower(sig.Type)), // Convert back to lowercase
		Symbol:     sig.Symbol,
		Timestamp:  sig.Timestamp,
		Confidence: sig.Confidence,
		EntryPrice: sig.EntryPrice,
		StopLoss:   sig.StopLoss,
		TakeProfit: sig.TakeProfit,
		Reason:     sig.Reason,
		Indicators: sig.Indicators,
	}
}
