CREATE TABLE IF NOT EXISTS evidence_subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id TEXT NOT NULL UNIQUE,
    deterministic_subject_key TEXT NOT NULL UNIQUE,
    subject_type TEXT NOT NULL,
    canonical_label TEXT NOT NULL,
    current_decision TEXT NOT NULL DEFAULT 'NO_TRADE',
    current_decision_reason TEXT NOT NULL DEFAULT 'awaiting deterministic evaluation',
    current_missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    first_observed_at TIMESTAMPTZ NOT NULL,
    latest_evidence_at TIMESTAMPTZ NOT NULL,
    latest_evaluation_at TIMESTAMPTZ,
    ruleset_version TEXT NOT NULL,
    resolved_assets TEXT[] NOT NULL DEFAULT '{}',
    unknown_assets BOOLEAN NOT NULL DEFAULT TRUE,
    candidate_id UUID REFERENCES candidate_trades(id) ON DELETE RESTRICT,
    projection_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_evidence_subject_decision CHECK (current_decision IN ('NO_TRADE', 'WATCH', 'CANDIDATE')),
    CONSTRAINT chk_evidence_subject_assets CHECK (
        (unknown_assets AND cardinality(resolved_assets) = 0)
        OR (NOT unknown_assets AND cardinality(resolved_assets) > 0)
    ),
    CONSTRAINT chk_evidence_subject_candidate CHECK (
        (current_decision = 'CANDIDATE' AND candidate_id IS NOT NULL)
        OR (current_decision IN ('NO_TRADE', 'WATCH') AND candidate_id IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS evidence_subject_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES evidence_subjects(id) ON DELETE CASCADE,
    genuine_event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    relationship_type TEXT NOT NULL,
    association_reason TEXT NOT NULL,
    source_independence TEXT NOT NULL,
    source_group_key TEXT NOT NULL,
    evidence_contribution TEXT NOT NULL,
    contradiction_state TEXT NOT NULL DEFAULT 'neutral',
    publication_at TIMESTAMPTZ NOT NULL,
    receipt_at TIMESTAMPTZ NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_evidence_subject_event UNIQUE (genuine_event_id),
    CONSTRAINT chk_evidence_relationship CHECK (relationship_type IN ('originating', 'corroborating', 'contradicting', 'duplicate', 'context')),
    CONSTRAINT chk_evidence_independence CHECK (source_independence IN ('primary', 'independent', 'not_independent', 'unknown')),
    CONSTRAINT chk_evidence_contradiction CHECK (contradiction_state IN ('corroborates', 'contradicts', 'neutral'))
);

CREATE TABLE IF NOT EXISTS evidence_subject_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES evidence_subjects(id) ON DELETE CASCADE,
    previous_decision TEXT NOT NULL,
    new_decision TEXT NOT NULL,
    deterministic_reason TEXT NOT NULL,
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    evidence_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    evidence_set_fingerprint TEXT NOT NULL,
    ruleset_version TEXT NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL,
    triggering_event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    idempotency_identity TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_evidence_evaluation_previous CHECK (previous_decision IN ('NO_TRADE', 'WATCH', 'CANDIDATE')),
    CONSTRAINT chk_evidence_evaluation_new CHECK (new_decision IN ('NO_TRADE', 'WATCH', 'CANDIDATE')),
    CONSTRAINT uq_evidence_subject_input UNIQUE (subject_id, ruleset_version, evidence_set_fingerprint)
);

CREATE TABLE IF NOT EXISTS evidence_subject_candidates (
    subject_id UUID PRIMARY KEY REFERENCES evidence_subjects(id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL UNIQUE REFERENCES candidate_trades(id) ON DELETE RESTRICT,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    link_reason TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_evidence_subject_current_decision
    ON evidence_subjects(current_decision, latest_evaluation_at DESC);
CREATE INDEX IF NOT EXISTS idx_evidence_subject_events_subject_time
    ON evidence_subject_events(subject_id, publication_at DESC, genuine_event_id);
CREATE INDEX IF NOT EXISTS idx_evidence_subject_events_source_group
    ON evidence_subject_events(subject_id, source_group_key);
CREATE INDEX IF NOT EXISTS idx_evidence_subject_evaluations_subject_time
    ON evidence_subject_evaluations(subject_id, evaluated_at DESC, id DESC);
