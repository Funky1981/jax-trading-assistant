CREATE TABLE IF NOT EXISTS workflow_audit_events (
    event_id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflow_instances(workflow_id) ON DELETE RESTRICT,
    sequence BIGINT NOT NULL CHECK (sequence > 0),
    previous_state TEXT NOT NULL,
    next_state TEXT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    input_fingerprint TEXT NOT NULL,
    content_identity TEXT NOT NULL,
    reason TEXT,
    occurred_at TIMESTAMPTZ NOT NULL,
    transition_version TEXT NOT NULL,
    CONSTRAINT uq_workflow_audit_sequence UNIQUE (workflow_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_workflow_audit_workflow_sequence
    ON workflow_audit_events(workflow_id, sequence);
