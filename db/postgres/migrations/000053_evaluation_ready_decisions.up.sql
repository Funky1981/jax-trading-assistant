ALTER TABLE genuine_event_decisions
    ADD COLUMN IF NOT EXISTS is_initial BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS decision_origin TEXT NOT NULL DEFAULT 'historical_replay',
    ADD COLUMN IF NOT EXISTS decision_context TEXT NOT NULL DEFAULT 'legacy_projection';

UPDATE genuine_event_decisions
SET is_initial = (decision_version = 1),
    decision_origin = CASE
        WHEN decision_version = 1
         AND decision_at >= event_receipt_at
         AND decision_at - event_receipt_at <= INTERVAL '1 minute'
            THEN 'live_origin'
        WHEN decision_version = 1 THEN 'historical_backfill'
        ELSE 'historical_replay'
    END,
    decision_context = CASE
        WHEN decision_version = 1
         AND decision_at >= event_receipt_at
         AND decision_at - event_receipt_at <= INTERVAL '1 minute'
            THEN 'continuous_world_monitor_ingestion'
        WHEN decision_version = 1 THEN 'legacy_bounded_backfill'
        ELSE 'legacy_projection_replay'
    END;

ALTER TABLE genuine_event_decisions
    ADD CONSTRAINT chk_genuine_event_decision_origin
        CHECK (decision_origin IN ('live_origin', 'historical_backfill', 'historical_replay'));

CREATE UNIQUE INDEX IF NOT EXISTS uq_genuine_event_initial_ruleset
    ON genuine_event_decisions(source_inbox_event_id, ruleset_version)
    WHERE is_initial;

CREATE INDEX IF NOT EXISTS idx_genuine_event_initial_origin_decision
    ON genuine_event_decisions(decision_origin, decision, event_receipt_at)
    WHERE is_initial;

CREATE OR REPLACE FUNCTION protect_genuine_event_initial_decision()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.is_initial
       AND OLD.source_url NOT LIKE 'https://example.com/%'
       AND OLD.source_url NOT LIKE 'http://example.com/%' THEN
        RAISE EXCEPTION 'immutable initial genuine-event decisions cannot be deleted';
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.is_initial AND (
        NEW.source_inbox_event_id IS DISTINCT FROM OLD.source_inbox_event_id OR
        NEW.normalized_event_id IS DISTINCT FROM OLD.normalized_event_id OR
        NEW.source_event_identity IS DISTINCT FROM OLD.source_event_identity OR
        NEW.decision IS DISTINCT FROM OLD.decision OR
        NEW.decision_version IS DISTINCT FROM OLD.decision_version OR
        NEW.ruleset_version IS DISTINCT FROM OLD.ruleset_version OR
        NEW.processor_identity IS DISTINCT FROM OLD.processor_identity OR
        NEW.processing_mode IS DISTINCT FROM OLD.processing_mode OR
        NEW.decision_at IS DISTINCT FROM OLD.decision_at OR
        NEW.event_publication_at IS DISTINCT FROM OLD.event_publication_at OR
        NEW.event_collection_at IS DISTINCT FROM OLD.event_collection_at OR
        NEW.event_receipt_at IS DISTINCT FROM OLD.event_receipt_at OR
        NEW.source IS DISTINCT FROM OLD.source OR
        NEW.source_url IS DISTINCT FROM OLD.source_url OR
        NEW.event_type IS DISTINCT FROM OLD.event_type OR
        NEW.severity IS DISTINCT FROM OLD.severity OR
        NEW.evidence_score IS DISTINCT FROM OLD.evidence_score OR
        NEW.evidence_score_source IS DISTINCT FROM OLD.evidence_score_source OR
        NEW.confidence IS DISTINCT FROM OLD.confidence OR
        NEW.affected_assets IS DISTINCT FROM OLD.affected_assets OR
        NEW.unknown_assets IS DISTINCT FROM OLD.unknown_assets OR
        NEW.asset_mapping_provenance IS DISTINCT FROM OLD.asset_mapping_provenance OR
        NEW.reasons IS DISTINCT FROM OLD.reasons OR
        NEW.blocking_reasons IS DISTINCT FROM OLD.blocking_reasons OR
        NEW.missing_evidence IS DISTINCT FROM OLD.missing_evidence OR
        NEW.trust_gate_state IS DISTINCT FROM OLD.trust_gate_state OR
        NEW.risk_review_state IS DISTINCT FROM OLD.risk_review_state OR
        NEW.candidate_id IS DISTINCT FROM OLD.candidate_id OR
        NEW.replay_identity IS DISTINCT FROM OLD.replay_identity OR
        NEW.input_fingerprint IS DISTINCT FROM OLD.input_fingerprint OR
        NEW.replay_metadata IS DISTINCT FROM OLD.replay_metadata OR
        NEW.created_at IS DISTINCT FROM OLD.created_at OR
        NEW.is_initial IS DISTINCT FROM OLD.is_initial OR
        NEW.decision_origin IS DISTINCT FROM OLD.decision_origin OR
        NEW.decision_context IS DISTINCT FROM OLD.decision_context
    ) THEN
        RAISE EXCEPTION 'immutable initial genuine-event decision fields cannot be updated';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_genuine_event_initial_decision ON genuine_event_decisions;
CREATE TRIGGER trg_protect_genuine_event_initial_decision
    BEFORE UPDATE OR DELETE ON genuine_event_decisions
    FOR EACH ROW EXECUTE FUNCTION protect_genuine_event_initial_decision();

CREATE TABLE IF NOT EXISTS event_asset_resolutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id UUID NOT NULL UNIQUE REFERENCES genuine_event_decisions(id) ON DELETE RESTRICT,
    source_inbox_event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE RESTRICT,
    resolution_status TEXT NOT NULL,
    resolved_symbol TEXT,
    benchmark_symbol TEXT,
    mapping_type TEXT NOT NULL,
    asset_relationship TEXT NOT NULL,
    confidence_class TEXT NOT NULL,
    deterministic_reason TEXT NOT NULL,
    source_fields TEXT[] NOT NULL DEFAULT '{}',
    source_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    resolver_ruleset_version TEXT NOT NULL,
    decision_origin TEXT NOT NULL,
    known_at_initial_decision_time BOOLEAN NOT NULL,
    knowable_at_operational_anchor BOOLEAN NOT NULL,
    ambiguity_reason TEXT,
    rejection_reason TEXT,
    canonical_entity TEXT,
    asset_class TEXT,
    exchange_name TEXT,
    effective_from DATE,
    effective_to DATE,
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    resolution_fingerprint TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_event_asset_resolution_status CHECK (resolution_status IN ('resolved', 'ambiguous', 'unresolved', 'rejected')),
    CONSTRAINT chk_event_asset_relationship CHECK (asset_relationship IN ('direct', 'proxy', 'none')),
    CONSTRAINT chk_event_asset_confidence CHECK (confidence_class IN ('exact', 'high_confidence_deterministic', 'proxy', 'ambiguous', 'unresolved')),
    CONSTRAINT chk_event_asset_symbol_state CHECK (
        (resolution_status = 'resolved' AND resolved_symbol IS NOT NULL AND asset_relationship IN ('direct', 'proxy'))
        OR (resolution_status <> 'resolved' AND resolved_symbol IS NULL AND asset_relationship = 'none')
    ),
    CONSTRAINT uq_event_asset_resolution_ruleset UNIQUE (source_inbox_event_id, resolver_ruleset_version)
);

CREATE INDEX IF NOT EXISTS idx_event_asset_resolution_symbol
    ON event_asset_resolutions(resolved_symbol, created_at)
    WHERE resolution_status = 'resolved';
CREATE INDEX IF NOT EXISTS idx_event_asset_resolution_review
    ON event_asset_resolutions(resolver_ruleset_version, resolution_status, decision_origin);

CREATE OR REPLACE FUNCTION protect_event_asset_resolution()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'event asset resolution provenance is append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_event_asset_resolution ON event_asset_resolutions;
CREATE TRIGGER trg_protect_event_asset_resolution
    BEFORE UPDATE OR DELETE ON event_asset_resolutions
    FOR EACH ROW EXECUTE FUNCTION protect_event_asset_resolution();

ALTER TABLE candles
    ADD COLUMN IF NOT EXISTS adjusted_state TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS provider_timezone TEXT NOT NULL DEFAULT 'UTC';
