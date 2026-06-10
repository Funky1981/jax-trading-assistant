CREATE TABLE IF NOT EXISTS multi_analyst_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    symbol TEXT NOT NULL,
    fundamental_snapshot_id UUID NULL,
    technical_snapshot_id UUID NULL,
    analyst_decision_id UUID NULL,
    fundamental_verdict TEXT NOT NULL,
    fundamental_score NUMERIC NOT NULL,
    technical_verdict TEXT NOT NULL,
    technical_score NUMERIC NOT NULL,
    risk_verdict TEXT NOT NULL,
    risk_score NUMERIC NOT NULL,
    risk_hard_blocks TEXT[] NOT NULL DEFAULT '{}',
    review_decision TEXT NOT NULL,
    candidate_score NUMERIC NOT NULL,
    approval_required BOOLEAN NOT NULL DEFAULT TRUE,
    review_reasons TEXT[] NOT NULL DEFAULT '{}',
    llm_summary TEXT NULL,
    llm_override_attempted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_multi_analyst_reviews_risk_verdict CHECK (
        risk_verdict IN ('pass', 'fail', 'insufficient_evidence')
    ),
    CONSTRAINT chk_multi_analyst_reviews_review_decision CHECK (
        review_decision IN (
            'candidate_allowed',
            'candidate_rejected',
            'watch_only',
            'insufficient_evidence',
            'manual_review_only'
        )
    ),
    CONSTRAINT chk_multi_analyst_reviews_candidate_score CHECK (
        candidate_score >= 0 AND candidate_score <= 100
    )
);

CREATE INDEX IF NOT EXISTS idx_multi_analyst_reviews_event
    ON multi_analyst_reviews(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_multi_analyst_reviews_symbol
    ON multi_analyst_reviews(symbol, created_at DESC);
