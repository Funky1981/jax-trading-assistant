CREATE TABLE IF NOT EXISTS exploratory_paper_lifecycles (
    lifecycle_id TEXT PRIMARY KEY,
    mode TEXT NOT NULL CHECK (mode = 'EXPLORATORY_PAPER'),
    environment TEXT NOT NULL CHECK (environment = 'PAPER'),
    execution_authority TEXT NOT NULL CHECK (execution_authority = 'NONE'),
    broker_execution_allowed BOOLEAN NOT NULL DEFAULT FALSE CHECK (broker_execution_allowed = FALSE),
    maximum_leverage NUMERIC NOT NULL CHECK (maximum_leverage > 0 AND maximum_leverage <= 1),
    candidate_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    issuer_id TEXT NOT NULL,
    instrument_id TEXT NOT NULL,
    thesis_id TEXT NOT NULL UNIQUE,
    thesis_hash TEXT NOT NULL,
    evidence_set_hash TEXT NOT NULL,
    thesis_payload JSONB NOT NULL,
    binding_payload JSONB NOT NULL,
    workflow_id TEXT NOT NULL,
    paper_intent_id TEXT NOT NULL,
    position_id TEXT NOT NULL UNIQUE,
    position_payload JSONB NOT NULL,
    state TEXT NOT NULL,
    operational_state TEXT NOT NULL,
    entry_order_id TEXT NOT NULL,
    entry_fill_id TEXT NOT NULL,
    exit_order_id TEXT,
    exit_fill_id TEXT,
    outcome_id TEXT UNIQUE,
    outcome_payload JSONB,
    policy_versions JSONB NOT NULL,
    next_review_at TIMESTAMPTZ,
    frozen_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    formal_evidence_eligible BOOLEAN NOT NULL DEFAULT FALSE CHECK (formal_evidence_eligible = FALSE),
    CONSTRAINT uq_exploratory_paper_entry_fill UNIQUE (entry_fill_id)
);

CREATE INDEX IF NOT EXISTS idx_exploratory_paper_lifecycle_state
    ON exploratory_paper_lifecycles(state, next_review_at);

CREATE TABLE IF NOT EXISTS exploratory_paper_evidence_reassessments (
    position_id TEXT NOT NULL REFERENCES exploratory_paper_lifecycles(position_id) ON DELETE RESTRICT,
    evidence_id TEXT NOT NULL,
    evidence_fingerprint TEXT NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (position_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS exploratory_paper_reviews (
    review_id TEXT PRIMARY KEY,
    position_id TEXT NOT NULL REFERENCES exploratory_paper_lifecycles(position_id) ON DELETE RESTRICT,
    session_number INTEGER NOT NULL CHECK (session_number BETWEEN 1 AND 5),
    scheduled_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'COMPLETED', 'MISSING_DATA', 'FAILED_CLOSED')),
    idempotency_identity TEXT NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    CONSTRAINT uq_exploratory_paper_review_session UNIQUE (position_id, session_number)
);

CREATE INDEX IF NOT EXISTS idx_exploratory_paper_reviews_due
    ON exploratory_paper_reviews(status, scheduled_at);

CREATE TABLE IF NOT EXISTS exploratory_paper_checkpoints (
    checkpoint_id TEXT PRIMARY KEY,
    position_id TEXT NOT NULL REFERENCES exploratory_paper_lifecycles(position_id) ON DELETE RESTRICT,
    thesis_id TEXT NOT NULL,
    session_number INTEGER NOT NULL CHECK (session_number BETWEEN 1 AND 5),
    observed_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    UNIQUE (position_id, session_number)
);

CREATE TABLE IF NOT EXISTS exploratory_paper_outcomes (
    outcome_id TEXT PRIMARY KEY,
    position_id TEXT NOT NULL UNIQUE REFERENCES exploratory_paper_lifecycles(position_id) ON DELETE RESTRICT,
    thesis_id TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode = 'EXPLORATORY_PAPER'),
    entry_fill_id TEXT NOT NULL,
    exit_fill_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    formal_evidence_eligible BOOLEAN NOT NULL DEFAULT FALSE CHECK (formal_evidence_eligible = FALSE)
);

CREATE OR REPLACE FUNCTION protect_exploratory_thesis_identity()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.thesis_id IS DISTINCT FROM OLD.thesis_id
       OR NEW.thesis_hash IS DISTINCT FROM OLD.thesis_hash
       OR NEW.evidence_set_hash IS DISTINCT FROM OLD.evidence_set_hash
       OR NEW.thesis_payload IS DISTINCT FROM OLD.thesis_payload
       OR NEW.binding_payload IS DISTINCT FROM OLD.binding_payload
       OR NEW.workflow_id IS DISTINCT FROM OLD.workflow_id
       OR NEW.paper_intent_id IS DISTINCT FROM OLD.paper_intent_id
       OR NEW.frozen_at IS DISTINCT FROM OLD.frozen_at
       OR NEW.mode IS DISTINCT FROM OLD.mode
       OR NEW.environment IS DISTINCT FROM OLD.environment
       OR NEW.execution_authority IS DISTINCT FROM OLD.execution_authority
       OR NEW.broker_execution_allowed IS DISTINCT FROM OLD.broker_execution_allowed
       OR NEW.maximum_leverage IS DISTINCT FROM OLD.maximum_leverage
       OR NEW.policy_versions IS DISTINCT FROM OLD.policy_versions
       OR NEW.formal_evidence_eligible IS DISTINCT FROM OLD.formal_evidence_eligible THEN
        RAISE EXCEPTION 'frozen exploratory thesis identity cannot be mutated';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_exploratory_thesis_identity ON exploratory_paper_lifecycles;
CREATE TRIGGER trg_protect_exploratory_thesis_identity
    BEFORE UPDATE ON exploratory_paper_lifecycles
    FOR EACH ROW EXECUTE FUNCTION protect_exploratory_thesis_identity();

CREATE OR REPLACE FUNCTION reject_exploratory_append_only_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'exploratory paper scientific records are append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_reject_exploratory_reassessment_mutation ON exploratory_paper_evidence_reassessments;
CREATE TRIGGER trg_reject_exploratory_reassessment_mutation
    BEFORE UPDATE OR DELETE ON exploratory_paper_evidence_reassessments
    FOR EACH ROW EXECUTE FUNCTION reject_exploratory_append_only_mutation();

DROP TRIGGER IF EXISTS trg_reject_exploratory_checkpoint_mutation ON exploratory_paper_checkpoints;
CREATE TRIGGER trg_reject_exploratory_checkpoint_mutation
    BEFORE UPDATE OR DELETE ON exploratory_paper_checkpoints
    FOR EACH ROW EXECUTE FUNCTION reject_exploratory_append_only_mutation();

DROP TRIGGER IF EXISTS trg_reject_exploratory_outcome_mutation ON exploratory_paper_outcomes;
CREATE TRIGGER trg_reject_exploratory_outcome_mutation
    BEFORE UPDATE OR DELETE ON exploratory_paper_outcomes
    FOR EACH ROW EXECUTE FUNCTION reject_exploratory_append_only_mutation();
