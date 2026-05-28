CREATE TABLE IF NOT EXISTS ai_scanner_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    enabled BOOLEAN NOT NULL,
    asset_scope TEXT NOT NULL,
    symbols JSONB NOT NULL DEFAULT '[]'::jsonb,
    universe_preset TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    minimum_confidence DOUBLE PRECISION NOT NULL,
    sentiment JSONB NOT NULL,
    status TEXT NOT NULL,
    last_scan_completed_at TIMESTAMPTZ,
    next_scan_at TIMESTAMPTZ,
    channels JSONB NOT NULL,
    policy JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_scanner_settings_next_scan_at
    ON ai_scanner_settings (next_scan_at);
