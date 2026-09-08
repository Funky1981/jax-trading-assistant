CREATE TABLE IF NOT EXISTS paper_soak_runs (
    run_id TEXT PRIMARY KEY,
    protocol_id TEXT NOT NULL,
    protocol_version TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    current_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('RUNNING', 'COMPLETED', 'FAILED')),
    evidence_class TEXT NOT NULL,
    actual_forward_duration INTERVAL NOT NULL DEFAULT INTERVAL '0 seconds',
    recommendations BIGINT NOT NULL DEFAULT 0 CHECK (recommendations >= 0),
    eligible_paper_intents BIGINT NOT NULL DEFAULT 0 CHECK (eligible_paper_intents >= 0),
    orders BIGINT NOT NULL DEFAULT 0 CHECK (orders >= 0),
    fills BIGINT NOT NULL DEFAULT 0 CHECK (fills >= 0),
    partial_fills BIGINT NOT NULL DEFAULT 0 CHECK (partial_fills >= 0),
    reconciliation_failures BIGINT NOT NULL DEFAULT 0 CHECK (reconciliation_failures >= 0),
    breaker_events BIGINT NOT NULL DEFAULT 0 CHECK (breaker_events >= 0),
    crashes BIGINT NOT NULL DEFAULT 0 CHECK (crashes >= 0),
    duplicate_fills BIGINT NOT NULL DEFAULT 0 CHECK (duplicate_fills >= 0),
    failure_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_paper_soak_time_order CHECK (current_at >= started_at)
);

CREATE TABLE IF NOT EXISTS paper_soak_events (
    event_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES paper_soak_runs(run_id) ON DELETE RESTRICT,
    sequence BIGINT NOT NULL CHECK (sequence > 0),
    observed_at TIMESTAMPTZ NOT NULL,
    observation JSONB NOT NULL,
    UNIQUE (run_id, sequence)
);

CREATE OR REPLACE FUNCTION reject_paper_soak_event_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'paper soak events are append-only';
END;
$$;

CREATE TRIGGER trg_paper_soak_events_append_only
    BEFORE UPDATE OR DELETE ON paper_soak_events
    FOR EACH ROW EXECUTE FUNCTION reject_paper_soak_event_mutation();
