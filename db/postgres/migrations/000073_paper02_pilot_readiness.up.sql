CREATE TABLE IF NOT EXISTS exploratory_paper_pilots (
    pilot_id TEXT PRIMARY KEY,
    mode TEXT NOT NULL CHECK (mode = 'EXPLORATORY_PAPER'),
    status TEXT NOT NULL CHECK (status IN ('DRAFT','READY_FOR_EXTERNAL_REVIEW','ACTIVE','PAUSED','ABORTED','COMPLETED')),
    protocol_version TEXT NOT NULL,
    protocol_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    formal_evidence_eligible BOOLEAN NOT NULL DEFAULT FALSE CHECK (formal_evidence_eligible = FALSE)
);

CREATE TABLE IF NOT EXISTS exploratory_paper_opportunities (
    opportunity_id TEXT PRIMARY KEY,
    pilot_id TEXT NOT NULL REFERENCES exploratory_paper_pilots(pilot_id) ON DELETE RESTRICT,
    event_identity TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL,
    final_classification TEXT NOT NULL,
    payload JSONB NOT NULL,
    formal_evidence_eligible BOOLEAN NOT NULL DEFAULT FALSE CHECK (formal_evidence_eligible = FALSE),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (pilot_id, event_identity)
);

CREATE INDEX IF NOT EXISTS idx_exploratory_paper_opportunities_pilot
    ON exploratory_paper_opportunities(pilot_id, first_seen_at, opportunity_id);

CREATE TABLE IF NOT EXISTS exploratory_paper_opportunity_events (
    event_id TEXT PRIMARY KEY,
    opportunity_id TEXT NOT NULL REFERENCES exploratory_paper_opportunities(opportunity_id) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS exploratory_paper_pilot_evidence (
    pilot_id TEXT NOT NULL REFERENCES exploratory_paper_pilots(pilot_id) ON DELETE RESTRICT,
    opportunity_id TEXT NOT NULL REFERENCES exploratory_paper_opportunities(opportunity_id) ON DELETE RESTRICT,
    evidence_id TEXT NOT NULL,
    phase TEXT NOT NULL CHECK (phase IN ('ENTRY_EVIDENCE','NEW_SUPPORTING_EVIDENCE','NEW_CONTRADICTORY_EVIDENCE','NEW_INVALIDATING_EVIDENCE','NO_NEW_RELEVANT_EVIDENCE','MISSING_EVIDENCE')),
    first_seen_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    formal_evidence_eligible BOOLEAN NOT NULL DEFAULT FALSE CHECK (formal_evidence_eligible = FALSE),
    PRIMARY KEY (opportunity_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS exploratory_paper_pilot_market_observations (
    observation_id TEXT PRIMARY KEY,
    pilot_id TEXT NOT NULL REFERENCES exploratory_paper_pilots(pilot_id) ON DELETE RESTRICT,
    opportunity_id TEXT NOT NULL REFERENCES exploratory_paper_opportunities(opportunity_id) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('ENTRY','REVIEW','EXIT','CHECKPOINT')),
    observed_at TIMESTAMPTZ,
    payload JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS exploratory_paper_pilot_incidents (
    incident_id TEXT PRIMARY KEY,
    pilot_id TEXT NOT NULL REFERENCES exploratory_paper_pilots(pilot_id) ON DELETE RESTRICT,
    severity TEXT NOT NULL,
    code TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    admission_blocked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_exploratory_paper_pilot_incidents_pilot
    ON exploratory_paper_pilot_incidents(pilot_id, occurred_at);

CREATE OR REPLACE FUNCTION protect_paper02_pilot_identity()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.pilot_id IS DISTINCT FROM OLD.pilot_id
       OR NEW.mode IS DISTINCT FROM OLD.mode
       OR NEW.protocol_version IS DISTINCT FROM OLD.protocol_version
       OR NEW.protocol_hash IS DISTINCT FROM OLD.protocol_hash
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.formal_evidence_eligible IS DISTINCT FROM OLD.formal_evidence_eligible THEN
        RAISE EXCEPTION 'PAPER-02 pilot identity is immutable';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_paper02_pilot_identity ON exploratory_paper_pilots;
CREATE TRIGGER trg_protect_paper02_pilot_identity
    BEFORE UPDATE ON exploratory_paper_pilots
    FOR EACH ROW EXECUTE FUNCTION protect_paper02_pilot_identity();

CREATE OR REPLACE FUNCTION reject_paper02_append_only_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'PAPER-02 opportunity/evidence records are append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_reject_paper02_opportunity_event_mutation ON exploratory_paper_opportunity_events;
CREATE TRIGGER trg_reject_paper02_opportunity_event_mutation
    BEFORE UPDATE OR DELETE ON exploratory_paper_opportunity_events
    FOR EACH ROW EXECUTE FUNCTION reject_paper02_append_only_mutation();

DROP TRIGGER IF EXISTS trg_reject_paper02_evidence_mutation ON exploratory_paper_pilot_evidence;
CREATE TRIGGER trg_reject_paper02_evidence_mutation
    BEFORE UPDATE OR DELETE ON exploratory_paper_pilot_evidence
    FOR EACH ROW EXECUTE FUNCTION reject_paper02_append_only_mutation();

COMMENT ON TABLE exploratory_paper_pilots IS
    'PAPER-02 exploratory readiness and activation identity; never formal evidence.';
