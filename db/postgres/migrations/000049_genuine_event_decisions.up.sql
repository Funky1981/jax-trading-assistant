CREATE TABLE IF NOT EXISTS genuine_event_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_inbox_event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE RESTRICT,
    source_event_identity TEXT NOT NULL,
    decision TEXT NOT NULL,
    decision_version INTEGER NOT NULL,
    ruleset_version TEXT NOT NULL,
    processor_identity TEXT NOT NULL,
    processing_mode TEXT NOT NULL DEFAULT 'deterministic',
    decision_at TIMESTAMPTZ NOT NULL,
    event_publication_at TIMESTAMPTZ NOT NULL,
    event_collection_at TIMESTAMPTZ,
    event_receipt_at TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    source_url TEXT NOT NULL,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    evidence_score NUMERIC(8,6) NOT NULL,
    evidence_score_source TEXT NOT NULL,
    confidence NUMERIC(8,6) NOT NULL,
    affected_assets TEXT[] NOT NULL DEFAULT '{}',
    unknown_assets BOOLEAN NOT NULL,
    asset_mapping_provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    blocking_reasons TEXT[] NOT NULL DEFAULT '{}',
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    trust_gate_state TEXT NOT NULL,
    risk_review_state TEXT NOT NULL,
    candidate_id UUID REFERENCES candidate_trades(id) ON DELETE RESTRICT,
    replay_identity TEXT NOT NULL,
    input_fingerprint TEXT NOT NULL,
    replay_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_genuine_event_decision CHECK (decision IN ('NO_TRADE', 'WATCH', 'CANDIDATE')),
    CONSTRAINT chk_genuine_event_processing_mode CHECK (processing_mode = 'deterministic'),
    CONSTRAINT chk_genuine_event_decision_version CHECK (decision_version > 0),
    CONSTRAINT chk_genuine_event_decision_score CHECK (evidence_score >= 0 AND evidence_score <= 1),
    CONSTRAINT chk_genuine_event_decision_confidence CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT chk_genuine_event_unknown_assets CHECK (
        (unknown_assets AND cardinality(affected_assets) = 0)
        OR (NOT unknown_assets AND cardinality(affected_assets) > 0)
    ),
    CONSTRAINT chk_genuine_event_candidate_link CHECK (
        (decision = 'CANDIDATE' AND candidate_id IS NOT NULL)
        OR (decision IN ('NO_TRADE', 'WATCH') AND candidate_id IS NULL)
    ),
    CONSTRAINT uq_genuine_event_replay_identity UNIQUE (replay_identity),
    CONSTRAINT uq_genuine_event_version UNIQUE (source_inbox_event_id, decision_version),
    CONSTRAINT uq_genuine_event_input UNIQUE (source_inbox_event_id, ruleset_version, input_fingerprint)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_genuine_event_current_ruleset
    ON genuine_event_decisions(source_inbox_event_id)
    WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_genuine_event_current_decision
    ON genuine_event_decisions(decision, decision_at DESC)
    WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_genuine_event_normalized
    ON genuine_event_decisions(normalized_event_id)
    WHERE normalized_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_genuine_event_candidate
    ON genuine_event_decisions(candidate_id)
    WHERE candidate_id IS NOT NULL;
