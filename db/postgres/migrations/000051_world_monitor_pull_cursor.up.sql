CREATE TABLE IF NOT EXISTS world_monitor_pull_cursors (
    consumer_name TEXT NOT NULL,
    source_endpoint_identity TEXT NOT NULL,
    last_committed_position BIGINT NOT NULL DEFAULT 0 CHECK (last_committed_position >= 0),
    diagnostic_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer_name, source_endpoint_identity)
);

COMMENT ON TABLE world_monitor_pull_cursors IS
    'Durable consumer positions; advancement must share the transaction that durably ingests and decides the corresponding World Monitor page.';
