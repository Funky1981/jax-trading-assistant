CREATE TABLE IF NOT EXISTS analyst_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    symbol TEXT NOT NULL,
    fundamental_snapshot_id UUID NULL,
    technical_snapshot_id UUID NULL,
    evidence_bundle_id UUID NULL,
    fundamental_score NUMERIC NOT NULL,
    technical_score NUMERIC NOT NULL,
    risk_score NUMERIC NOT NULL,
    confidence_score NUMERIC NOT NULL,
    candidate_score NUMERIC NOT NULL,
    decision TEXT NOT NULL,
    hard_vetoes TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_analyst_decisions_decision CHECK (
        decision IN (
            'candidate_allowed',
            'candidate_rejected',
            'watch_only',
            'insufficient_evidence',
            'manual_review_only'
        )
    ),
    CONSTRAINT chk_analyst_decisions_score CHECK (
        candidate_score >= 0 AND candidate_score <= 100
    )
);

CREATE INDEX IF NOT EXISTS idx_analyst_decisions_event
    ON analyst_decisions(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analyst_decisions_symbol
    ON analyst_decisions(symbol, created_at DESC);
