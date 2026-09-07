CREATE TABLE IF NOT EXISTS research_task_checkpoints (
    checkpoint_id TEXT PRIMARY KEY CHECK (checkpoint_id ~ '^checkpoint_[0-9a-f]{64}$'),
    task_id TEXT NOT NULL,
    checkpoint_version INTEGER NOT NULL CHECK (checkpoint_version > 0),
    status TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (task_id, checkpoint_version)
);

COMMENT ON TABLE research_task_checkpoints IS 'Immutable structured research task checkpoints; bounded resume state is stored independently from chat transcripts.';
