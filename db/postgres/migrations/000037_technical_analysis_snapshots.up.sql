CREATE TABLE IF NOT EXISTS technical_analysis_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    symbol TEXT NOT NULL,
    analysis_time_utc TIMESTAMPTZ NOT NULL,
    timeframe TEXT NOT NULL,
    trend_state TEXT NOT NULL,
    structure_state TEXT NOT NULL,
    key_levels JSONB NOT NULL DEFAULT '{}'::jsonb,
    event_reaction JSONB NOT NULL DEFAULT '{}'::jsonb,
    volume_volatility JSONB NOT NULL DEFAULT '{}'::jsonb,
    relative_strength JSONB NOT NULL DEFAULT '{}'::jsonb,
    technical_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    invalidation_rules TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (macro_event_id, symbol, timeframe),
    CONSTRAINT chk_technical_analysis_snapshots_verdict CHECK (
        verdict IN (
            'confirmed_bullish',
            'confirmed_bearish',
            'watch_only',
            'no_confirmation',
            'conflicting',
            'too_extended',
            'whipsaw',
            'insufficient_data'
        )
    ),
    CONSTRAINT chk_technical_analysis_snapshots_score CHECK (
        technical_score >= 0 AND technical_score <= 100
    )
);

CREATE INDEX IF NOT EXISTS idx_technical_analysis_snapshots_event
    ON technical_analysis_snapshots(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_technical_analysis_snapshots_symbol_timeframe
    ON technical_analysis_snapshots(symbol, timeframe, created_at DESC);
