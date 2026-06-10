CREATE TABLE IF NOT EXISTS macro_priced_in_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    verdict TEXT NOT NULL,
    score NUMERIC NOT NULL,
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, symbol),
    CONSTRAINT chk_macro_priced_in_verdict CHECK (
        verdict IN ('not_priced_in', 'partially_priced_in', 'priced_in', 'overreaction', 'unclear')
    ),
    CONSTRAINT chk_macro_priced_in_score CHECK (score >= 0 AND score <= 1)
);

CREATE TABLE IF NOT EXISTS macro_confounders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    confounder_type TEXT NOT NULL,
    headline TEXT NOT NULL,
    source TEXT,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_macro_confounder_severity CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    )
);

CREATE INDEX IF NOT EXISTS idx_macro_priced_in_scores_event
    ON macro_priced_in_scores(macro_event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_macro_priced_in_scores_verdict
    ON macro_priced_in_scores(verdict, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_confounders_event
    ON macro_confounders(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_confounders_severity
    ON macro_confounders(severity, created_at DESC);
