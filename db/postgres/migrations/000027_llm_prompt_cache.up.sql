CREATE TABLE IF NOT EXISTS llm_prompt_cache (
    cache_key TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    correlation_id TEXT,
    source_hash TEXT NOT NULL,
    response_text TEXT NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    response_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_prompt_cache_expires_at
    ON llm_prompt_cache (expires_at);

CREATE INDEX IF NOT EXISTS idx_llm_prompt_cache_task_type
    ON llm_prompt_cache (task_type);
