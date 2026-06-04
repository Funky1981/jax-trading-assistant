package llmcontext

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PostgresCostRollupJob struct {
	db SQLExecutor
}

func NewPostgresCostRollupJob(db SQLExecutor) PostgresCostRollupJob {
	return PostgresCostRollupJob{db: db}
}

func (j PostgresCostRollupJob) RunWindow(ctx context.Context, from, to time.Time) error {
	if !to.After(from) {
		return fmt.Errorf("rollup window end must be after start")
	}
	if j.db == nil {
		return fmt.Errorf("postgres cost rollup job requires database handle")
	}
	for _, spec := range []struct {
		rollupType string
		keyExpr    string
	}{
		{rollupType: "strategy", keyExpr: "COALESCE(strategy_id, 'unknown')"},
		{rollupType: "symbol", keyExpr: "COALESCE(symbol, 'unknown')"},
		{rollupType: "task_type", keyExpr: "task_type"},
	} {
		if err := j.runOne(ctx, spec.rollupType, spec.keyExpr, from, to); err != nil {
			return err
		}
	}
	return nil
}

func (j PostgresCostRollupJob) runOne(ctx context.Context, rollupType, keyExpr string, from, to time.Time) error {
	_, err := j.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO llm_cost_rollups (
			rollup_type, rollup_key, event_count, candidate_count, approved_count,
			total_input_tokens, total_output_tokens, total_cost_usd,
			paid_calls_avoided, cache_hit_count, headroom_tokens_saved,
			from_ts, to_ts
		)
		SELECT
			$1,
			%s,
			COUNT(DISTINCT event_id) FILTER (WHERE event_id IS NOT NULL),
			COUNT(DISTINCT candidate_id) FILTER (WHERE candidate_id IS NOT NULL),
			0,
			COALESCE(SUM(input_tokens + estimated_input_tokens), 0),
			COALESCE(SUM(output_tokens + estimated_output_tokens), 0),
			COALESCE(SUM(actual_cost_usd + estimated_cost_usd), 0),
			COALESCE(SUM(CASE WHEN blocked THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(cached_input_tokens), 0),
			$2,
			$3
		FROM llm_usage_logs
		WHERE created_at >= $2 AND created_at < $3
		GROUP BY %s
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
	`, keyExpr, keyExpr), rollupType, from, to)
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresCostRollupJob.%s: %w", rollupType, err)
	}
	return nil
}

var _ SQLExecutor = (*sql.DB)(nil)
