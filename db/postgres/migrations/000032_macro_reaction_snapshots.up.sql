CREATE TABLE IF NOT EXISTS macro_reaction_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    pre_price NUMERIC NOT NULL,
    post_price NUMERIC NOT NULL,
    change_abs NUMERIC NOT NULL,
    change_percent NUMERIC NOT NULL,
    high_after NUMERIC,
    low_after NUMERIC,
    volume_ratio NUMERIC,
    atr_ratio NUMERIC,
    direction TEXT NOT NULL,
    confirms_event BOOLEAN NOT NULL,
    too_extended BOOLEAN NOT NULL,
    noisy BOOLEAN NOT NULL,
    reason TEXT NOT NULL,
    raw_candles JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, symbol, timeframe),
    CONSTRAINT chk_macro_reaction_direction CHECK (
        direction IN ('up', 'down', 'flat', 'whipsaw', 'unknown')
    )
);

CREATE INDEX IF NOT EXISTS idx_macro_reaction_snapshots_event
    ON macro_reaction_snapshots(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_reaction_snapshots_symbol_timeframe
    ON macro_reaction_snapshots(symbol, timeframe, created_at DESC);
