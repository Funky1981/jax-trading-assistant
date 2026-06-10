CREATE TABLE IF NOT EXISTS fundamental_analysis_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    symbol TEXT NOT NULL,
    analysis_time_utc TIMESTAMPTZ NOT NULL,
    event_summary TEXT NOT NULL,
    expected_market_impact TEXT NOT NULL,
    affected_themes TEXT[] NOT NULL DEFAULT '{}',
    cross_market_checks JSONB NOT NULL DEFAULT '[]'::jsonb,
    confounders JSONB NOT NULL DEFAULT '[]'::jsonb,
    fundamental_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (macro_event_id, symbol),
    CONSTRAINT chk_fundamental_analysis_snapshots_verdict CHECK (
        verdict IN (
            'strong_bullish',
            'moderate_bullish',
            'neutral',
            'moderate_bearish',
            'strong_bearish',
            'conflicted',
            'insufficient_data'
        )
    ),
    CONSTRAINT chk_fundamental_analysis_snapshots_score CHECK (
        fundamental_score >= 0 AND fundamental_score <= 100
    )
);

CREATE INDEX IF NOT EXISTS idx_fundamental_analysis_snapshots_event
    ON fundamental_analysis_snapshots(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fundamental_analysis_snapshots_symbol_timeframe
    ON fundamental_analysis_snapshots(symbol, created_at DESC);
