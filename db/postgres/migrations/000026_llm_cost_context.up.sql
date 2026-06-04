CREATE TABLE IF NOT EXISTS llm_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type TEXT NOT NULL,
    model_alias TEXT NOT NULL,
    provider_model TEXT NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    estimated_input_tokens INT NOT NULL DEFAULT 0,
    estimated_output_tokens INT NOT NULL DEFAULT 0,
    cached_input_tokens INT NOT NULL DEFAULT 0,
    estimated_cost_usd NUMERIC NOT NULL DEFAULT 0,
    actual_cost_usd NUMERIC NOT NULL DEFAULT 0,
    cache_hit BOOLEAN NOT NULL DEFAULT FALSE,
    cache_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    virtual_key TEXT,
    event_id TEXT,
    candidate_id TEXT,
    strategy_id TEXT,
    symbol TEXT,
    correlation_id TEXT NOT NULL,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    block_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_task_created
    ON llm_usage_logs (task_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_candidate
    ON llm_usage_logs (candidate_id)
    WHERE candidate_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS llm_cost_rollups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rollup_type TEXT NOT NULL,
    rollup_key TEXT NOT NULL,
    event_count INT NOT NULL DEFAULT 0,
    candidate_count INT NOT NULL DEFAULT 0,
    approved_count INT NOT NULL DEFAULT 0,
    total_input_tokens INT NOT NULL DEFAULT 0,
    total_output_tokens INT NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC NOT NULL DEFAULT 0,
    paid_calls_avoided INT NOT NULL DEFAULT 0,
    cache_hit_count INT NOT NULL DEFAULT 0,
    headroom_tokens_saved INT NOT NULL DEFAULT 0,
    from_ts TIMESTAMPTZ NOT NULL,
    to_ts TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_cost_rollups_unique_window
    ON llm_cost_rollups (rollup_type, rollup_key, from_ts, to_ts);
