CREATE TABLE IF NOT EXISTS macro_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    region TEXT NOT NULL,
    event_time_utc TIMESTAMPTZ NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT,
    actual_value NUMERIC,
    expected_value NUMERIC,
    previous_value NUMERIC,
    unit TEXT,
    surprise_value NUMERIC,
    surprise_percent NUMERIC,
    direction TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'accepted',
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source, source_event_id),
    CONSTRAINT chk_macro_events_direction CHECK (
        direction IN (
            'hawkish_rates',
            'dovish_rates',
            'risk_on',
            'risk_off',
            'inflation_hot',
            'inflation_cool',
            'growth_strong',
            'growth_weak',
            'unclear'
        )
    ),
    CONSTRAINT chk_macro_events_confidence CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT chk_macro_events_status CHECK (
        status IN ('accepted', 'rejected', 'quarantined')
    )
);

CREATE TABLE IF NOT EXISTS macro_event_etf_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    theme TEXT NOT NULL,
    mapping_reason TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, symbol),
    CONSTRAINT chk_macro_event_etf_map_confidence CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE INDEX IF NOT EXISTS idx_macro_events_type_time
    ON macro_events(event_type, event_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_macro_events_source_event
    ON macro_events(source, source_event_id);
CREATE INDEX IF NOT EXISTS idx_macro_events_status_time
    ON macro_events(status, event_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_macro_event_etf_map_symbol
    ON macro_event_etf_map(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_event_etf_map_event
    ON macro_event_etf_map(macro_event_id);
