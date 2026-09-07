CREATE TABLE IF NOT EXISTS workflow_breakers (
    name TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    tripped BOOLEAN NOT NULL,
    reason TEXT,
    actor TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_breaker_events (
    event_id TEXT PRIMARY KEY,
    name TEXT NOT NULL REFERENCES workflow_breakers(name) ON DELETE RESTRICT,
    tripped BOOLEAN NOT NULL,
    reason TEXT,
    actor TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_breaker_events_name_time
    ON workflow_breaker_events(name, occurred_at DESC);

CREATE OR REPLACE FUNCTION reject_workflow_breaker_event_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'workflow breaker events are append-only';
END;
$$;

CREATE TRIGGER trg_workflow_breaker_events_append_only
    BEFORE UPDATE OR DELETE ON workflow_breaker_events
    FOR EACH ROW EXECUTE FUNCTION reject_workflow_breaker_event_mutation();
