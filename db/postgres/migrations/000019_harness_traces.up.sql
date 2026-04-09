ALTER TABLE chat_messages
ADD COLUMN IF NOT EXISTS trace_id TEXT;

CREATE TABLE IF NOT EXISTS harness_traces (
    trace_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    question TEXT NOT NULL,
    tool_names JSONB NOT NULL DEFAULT '[]'::jsonb,
    validator_failures JSONB NOT NULL DEFAULT '[]'::jsonb,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_harness_traces_session_created
    ON harness_traces (session_id, created_at DESC);
