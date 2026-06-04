package llmcontext

import (
	"context"
	"fmt"
)

type PGXExecutor interface {
	Exec(ctx context.Context, query string, args ...any) error
}

type PGXUsageLogger struct {
	exec PGXExecutor
}

func NewPGXUsageLogger(exec PGXExecutor) PGXUsageLogger {
	return PGXUsageLogger{exec: exec}
}

func (l PGXUsageLogger) RecordPlanned(pkg PromptPackage, decision CostDecision) error {
	if l.exec == nil {
		return fmt.Errorf("pgx usage logger requires executor")
	}
	err := l.exec.Exec(context.Background(), `
		INSERT INTO llm_usage_logs (
			task_type, model_alias, provider_model,
			estimated_input_tokens, estimated_output_tokens, cached_input_tokens,
			estimated_cost_usd, actual_cost_usd, cache_hit, cache_eligible,
			virtual_key, event_id, candidate_id, strategy_id, symbol, correlation_id,
			blocked, block_reason
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18
		)
	`, pkg.TaskType, pkg.Model, pkg.Model,
		pkg.EstimatedInputTokens, pkg.EstimatedOutputTokens, 0,
		pkg.EstimatedCostUSD, 0.0, false, pkg.CacheEligible,
		"", pkg.EventID, pkg.CandidateID, pkg.StrategyID, pkg.Symbol, pkg.CorrelationID,
		!decision.Allowed, decision.BlockReason)
	if err != nil {
		return fmt.Errorf("llmcontext.PGXUsageLogger.RecordPlanned: %w", err)
	}
	return nil
}

func (l PGXUsageLogger) RecordActual(result LLMResult) error {
	if l.exec == nil {
		return fmt.Errorf("pgx usage logger requires executor")
	}
	total := result.InputTokens + result.OutputTokens
	err := l.exec.Exec(context.Background(), `
		UPDATE llm_usage_logs
		SET input_tokens = $1,
		    output_tokens = $2,
		    total_tokens = $3,
		    cached_input_tokens = $4,
		    actual_cost_usd = $5,
		    cache_hit = $6
		WHERE correlation_id = $7
	`, result.InputTokens, result.OutputTokens, total, result.CachedTokens, result.ActualCostUSD, result.CachedTokens > 0, result.CorrelationID)
	if err != nil {
		return fmt.Errorf("llmcontext.PGXUsageLogger.RecordActual: %w", err)
	}
	return nil
}
