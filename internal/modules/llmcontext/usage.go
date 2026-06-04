package llmcontext

import "time"

type MemoryUsageLogger struct {
	records []UsageRecord
}

func NewMemoryUsageLogger() *MemoryUsageLogger {
	return &MemoryUsageLogger{}
}

func (l *MemoryUsageLogger) RecordPlanned(pkg PromptPackage, decision CostDecision) error {
	l.records = append(l.records, UsageRecord{
		TaskType:              pkg.TaskType,
		ModelAlias:            pkg.Model,
		EstimatedInputTokens:  pkg.EstimatedInputTokens,
		EstimatedOutputTokens: pkg.EstimatedOutputTokens,
		EstimatedCostUSD:      pkg.EstimatedCostUSD,
		EventID:               pkg.EventID,
		CandidateID:           pkg.CandidateID,
		StrategyID:            pkg.StrategyID,
		Symbol:                pkg.Symbol,
		CorrelationID:         pkg.CorrelationID,
		Blocked:               !decision.Allowed,
		BlockReason:           decision.BlockReason,
		CreatedAt:             time.Now().UTC(),
	})
	return nil
}

func (l *MemoryUsageLogger) RecordActual(result LLMResult) error {
	for i := range l.records {
		if l.records[i].CorrelationID == result.CorrelationID {
			l.records[i].ActualInputTokens = result.InputTokens
			l.records[i].ActualOutputTokens = result.OutputTokens
			l.records[i].CachedInputTokens = result.CachedTokens
			l.records[i].ActualCostUSD = result.ActualCostUSD
			l.records[i].CacheHit = result.CachedTokens > 0
			return nil
		}
	}
	l.records = append(l.records, UsageRecord{
		CorrelationID:      result.CorrelationID,
		ActualInputTokens:  result.InputTokens,
		ActualOutputTokens: result.OutputTokens,
		CachedInputTokens:  result.CachedTokens,
		ActualCostUSD:      result.ActualCostUSD,
		CacheHit:           result.CachedTokens > 0,
		CreatedAt:          time.Now().UTC(),
	})
	return nil
}

func (l *MemoryUsageLogger) Records() []UsageRecord {
	out := make([]UsageRecord, len(l.records))
	copy(out, l.records)
	return out
}

func RollupCosts(records []UsageRecord, rollupType, rollupKey string, from, to time.Time) CostRollup {
	events := map[string]bool{}
	candidates := map[string]bool{}
	rollup := CostRollup{RollupType: rollupType, RollupKey: rollupKey, From: from, To: to}
	for _, record := range records {
		rollup.TotalInputTokens += record.EstimatedInputTokens + record.ActualInputTokens
		rollup.TotalOutputTokens += record.EstimatedOutputTokens + record.ActualOutputTokens
		rollup.TotalCostUSD += record.EstimatedCostUSD
		if record.ActualCostUSD > 0 {
			rollup.TotalCostUSD += record.ActualCostUSD
		}
		if record.EventID != "" {
			events[record.EventID] = true
		}
		if record.CandidateID != "" {
			candidates[record.CandidateID] = true
		}
		if record.Blocked {
			rollup.PaidCallsAvoided++
		}
		rollup.HeadroomTokensSaved += record.CachedInputTokens
	}
	rollup.EventCount = len(events)
	rollup.CandidateCount = len(candidates)
	return rollup
}
