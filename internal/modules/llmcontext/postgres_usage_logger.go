package llmcontext

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type PostgresUsageLogger struct {
	db SQLExecutor
}

func NewPostgresUsageLogger(db SQLExecutor) PostgresUsageLogger {
	return PostgresUsageLogger{db: db}
}

func (l PostgresUsageLogger) RecordPlanned(ctx context.Context, record UsageRecord) error {
	if l.db == nil {
		return fmt.Errorf("postgres usage logger requires database handle")
	}
	if record.ProviderModel == "" {
		record.ProviderModel = record.ModelAlias
	}
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO llm_usage_logs (
			task_type, model_alias, provider_model,
			estimated_input_tokens, estimated_output_tokens, cached_input_tokens,
			estimated_cost_usd, actual_cost_usd, cache_hit, cache_eligible,
			virtual_key, event_id, candidate_id, strategy_id, symbol, correlation_id,
			blocked, block_reason
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18
		)
	`, record.TaskType, record.ModelAlias, record.ProviderModel,
		record.EstimatedInputTokens, record.EstimatedOutputTokens, record.CachedInputTokens,
		record.EstimatedCostUSD, record.ActualCostUSD, record.CacheHit, record.CacheEligible,
		record.VirtualKey, record.EventID, record.CandidateID, record.StrategyID, record.Symbol, record.CorrelationID,
		record.Blocked, record.BlockReason)
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresUsageLogger.RecordPlanned: %w", err)
	}
	return nil
}

func (l PostgresUsageLogger) RecordActual(ctx context.Context, result LLMResult) error {
	if l.db == nil {
		return fmt.Errorf("postgres usage logger requires database handle")
	}
	total := result.InputTokens + result.OutputTokens
	cacheHit := result.CachedTokens > 0
	_, err := l.db.ExecContext(ctx, `
		UPDATE llm_usage_logs
		SET input_tokens = $1,
		    output_tokens = $2,
		    total_tokens = $3,
		    cached_input_tokens = $4,
		    actual_cost_usd = $5,
		    cache_hit = $6
		WHERE correlation_id = $7
	`, result.InputTokens, result.OutputTokens, total, result.CachedTokens, result.ActualCostUSD, cacheHit, result.CorrelationID)
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresUsageLogger.RecordActual: %w", err)
	}
	return nil
}

func (l PostgresUsageLogger) UpsertRollup(ctx context.Context, rollup CostRollup) error {
	if l.db == nil {
		return fmt.Errorf("postgres usage logger requires database handle")
	}
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO llm_cost_rollups (
			rollup_type, rollup_key, event_count, candidate_count, approved_count,
			total_input_tokens, total_output_tokens, total_cost_usd,
			paid_calls_avoided, cache_hit_count, headroom_tokens_saved,
			from_ts, to_ts
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
		)
		ON CONFLICT (rollup_type, rollup_key, from_ts, to_ts) DO UPDATE SET
			event_count = EXCLUDED.event_count,
			candidate_count = EXCLUDED.candidate_count,
			approved_count = EXCLUDED.approved_count,
			total_input_tokens = EXCLUDED.total_input_tokens,
			total_output_tokens = EXCLUDED.total_output_tokens,
			total_cost_usd = EXCLUDED.total_cost_usd,
			paid_calls_avoided = EXCLUDED.paid_calls_avoided,
			cache_hit_count = EXCLUDED.cache_hit_count,
			headroom_tokens_saved = EXCLUDED.headroom_tokens_saved,
			created_at = NOW()
	`, rollup.RollupType, rollup.RollupKey, rollup.EventCount, rollup.CandidateCount, rollup.ApprovedCount,
		rollup.TotalInputTokens, rollup.TotalOutputTokens, rollup.TotalCostUSD,
		rollup.PaidCallsAvoided, rollup.CacheHitCount, rollup.HeadroomTokensSaved,
		rollup.From, rollup.To)
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresUsageLogger.UpsertRollup: %w", err)
	}
	return nil
}
