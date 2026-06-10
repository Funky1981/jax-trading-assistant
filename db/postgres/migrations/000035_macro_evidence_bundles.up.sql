CREATE TABLE IF NOT EXISTS macro_evidence_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    status TEXT NOT NULL,
    verdict TEXT NOT NULL,
    summary TEXT NOT NULL,
    evidence JSONB NOT NULL,
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    walkaway_reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, symbol),
    CONSTRAINT chk_macro_evidence_bundle_verdict CHECK (
        verdict IN ('candidate_allowed', 'candidate_blocked', 'watch_only', 'insufficient_evidence')
    )
);

CREATE INDEX IF NOT EXISTS idx_macro_evidence_bundles_event
    ON macro_evidence_bundles(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_evidence_bundles_verdict
    ON macro_evidence_bundles(verdict, created_at DESC);
