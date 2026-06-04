CREATE TABLE IF NOT EXISTS llm_memory_artifacts (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    symbol TEXT,
    strategy_id TEXT,
    summary TEXT NOT NULL,
    source_ids JSONB NOT NULL,
    quality DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_memory_artifacts_lookup
    ON llm_memory_artifacts (task_type, symbol, strategy_id, quality DESC, created_at DESC);
