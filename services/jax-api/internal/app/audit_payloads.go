package app

import (
	"sort"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

func redactEventPayload(e domain.Event) map[string]any {
	return map[string]any{
		"eventId":     e.ID,
		"symbol":      e.Symbol,
		"type":        e.Type,
		"time":        e.Time,
		"payloadKeys": mapKeys(e.Payload),
	}
}

func redactTradeSetupPayload(s domain.TradeSetup) map[string]any {
	return map[string]any{
		"tradeId":     s.ID,
		"symbol":      s.Symbol,
		"direction":   s.Direction,
		"strategyId":  s.StrategyID,
		"eventId":     s.EventID,
		"targets":     len(s.Targets),
		"hasEntry":    s.Entry != 0,
		"hasStop":     s.Stop != 0,
		"notesSet":    s.Notes != "",
		"researchSet": s.Research != nil,
	}
}

func redactRiskResultPayload(r domain.RiskResult) map[string]any {
	return map[string]any{
		"positionSize": r.PositionSize,
		"hasRMultiple": r.RMultiple != 0,
		"riskSet":      r.TotalRisk != 0,
	}
}

func redactRiskInputPayload(accountSize float64, riskPercent float64, entry float64, stop float64, target *float64) map[string]any {
	return map[string]any{
		"hasAccountSize": accountSize != 0,
		"hasRiskPercent": riskPercent != 0,
		"hasEntry":       entry != 0,
		"hasStop":        stop != 0,
		"hasTarget":      target != nil,
	}
}

func mapKeys(payload map[string]any) []string {
	if len(payload) == 0 {
		return nil
	}
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
