CREATE TABLE IF NOT EXISTS analysis_case_studies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    symbol TEXT NOT NULL,
    event_type TEXT NOT NULL,
    playbook_key TEXT NOT NULL,
    technical_snapshot_id UUID NULL,
    fundamental_snapshot_id UUID NULL,
    analyst_decision_id UUID NULL,
    review_id UUID NULL,
    decision TEXT NOT NULL,
    expected_outcome TEXT NOT NULL,
    actual_outcome TEXT NULL,
    outcome_r NUMERIC NULL,
    surprise_bucket TEXT NULL,
    technical_setup TEXT NULL,
    market_regime TEXT NULL,
    what_worked TEXT[] NOT NULL DEFAULT '{}',
    what_failed TEXT[] NOT NULL DEFAULT '{}',
    lesson TEXT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_analysis_case_studies_decision CHECK (
        decision IN (
            'candidate_allowed',
            'candidate_rejected',
            'watch_only',
            'insufficient_evidence',
            'manual_review_only'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_analysis_case_studies_symbol
    ON analysis_case_studies(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analysis_case_studies_event_playbook
    ON analysis_case_studies(event_type, playbook_key, created_at DESC);

CREATE TABLE IF NOT EXISTS analyst_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_study_id UUID NOT NULL REFERENCES analysis_case_studies(id) ON DELETE CASCADE,
    feedback_source TEXT NOT NULL,
    rating TEXT NOT NULL,
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_analyst_feedback_rating CHECK (
        rating IN ('accepted', 'rejected', 'helpful', 'not_helpful', 'needs_review')
    )
);

CREATE INDEX IF NOT EXISTS idx_analyst_feedback_case_study
    ON analyst_feedback(case_study_id, created_at DESC);
