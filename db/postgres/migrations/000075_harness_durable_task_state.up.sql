CREATE TABLE IF NOT EXISTS harness_tasks (
    task_id TEXT PRIMARY KEY,
    objective JSONB NOT NULL,
    latest_state_version INTEGER NOT NULL CHECK (latest_state_version > 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_task_state_versions (
    task_id TEXT NOT NULL REFERENCES harness_tasks(task_id) ON DELETE RESTRICT,
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    state_hash TEXT NOT NULL CHECK (state_hash ~ '^sha256:[0-9a-f]{64}$'),
    payload JSONB NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, state_version),
    UNIQUE (task_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS harness_checkpoints (
    checkpoint_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES harness_tasks(task_id) ON DELETE RESTRICT,
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    content_hash TEXT NOT NULL CHECK (content_hash ~ '^sha256:[0-9a-f]{64}$'),
    payload JSONB NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (task_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS harness_compactions (
    compaction_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES harness_tasks(task_id) ON DELETE RESTRICT,
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    content_hash TEXT NOT NULL CHECK (content_hash ~ '^sha256:[0-9a-f]{64}$'),
    payload JSONB NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (task_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS harness_failure_events (
    event_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES harness_tasks(task_id) ON DELETE RESTRICT,
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_retry_events (
    event_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES harness_tasks(task_id) ON DELETE RESTRICT,
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_harness_state_versions_task
    ON harness_task_state_versions(task_id, state_version);

CREATE INDEX IF NOT EXISTS idx_harness_checkpoints_task
    ON harness_checkpoints(task_id, state_version, checkpoint_id);

CREATE INDEX IF NOT EXISTS idx_harness_compactions_task
    ON harness_compactions(task_id, state_version, compaction_id);

COMMENT ON TABLE harness_tasks IS 'HARNESS-03 durable research task identity and latest-version pointer; independent from PAPER-02 and canonical evidence.';
COMMENT ON TABLE harness_task_state_versions IS 'HARNESS-03 append-only task-state history with optimistic concurrency and idempotency.';
COMMENT ON TABLE harness_checkpoints IS 'HARNESS-03 immutable checkpoint references and structured continuation content.';
COMMENT ON TABLE harness_compactions IS 'HARNESS-03 derived structured compaction records; never canonical task state.';
