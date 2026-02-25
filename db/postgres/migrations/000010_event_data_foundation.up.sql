-- Phase 2: Event data foundation (raw ingestion, normalization, symbol mapping).

CREATE TABLE IF NOT EXISTS event_sources (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    provider_type TEXT NOT NULL DEFAULT 'external',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 100,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_raw (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id TEXT NOT NULL REFERENCES event_sources(id) ON DELETE RESTRICT,
    source_event_id TEXT NOT NULL,
    event_kind TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    symbol TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash TEXT NOT NULL,
    flow_id TEXT,
    data_source_type TEXT NOT NULL DEFAULT 'real',
    source_provider TEXT,
    is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    synthetic_reason TEXT,
    provenance_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, source_event_id)
);

CREATE TABLE IF NOT EXISTS event_normalized (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raw_event_id UUID NOT NULL REFERENCES event_raw(id) ON DELETE CASCADE,
    canonical_key TEXT NOT NULL UNIQUE,
    event_kind TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    severity TEXT NOT NULL DEFAULT 'unknown',
    event_time TIMESTAMPTZ NOT NULL,
    source_id TEXT NOT NULL REFERENCES event_sources(id) ON DELETE RESTRICT,
    primary_symbol TEXT,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    data_source_type TEXT NOT NULL DEFAULT 'real',
    source_provider TEXT,
    is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    synthetic_reason TEXT,
    provenance_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_symbol_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    normalized_event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    relevance DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    mapping_method TEXT NOT NULL DEFAULT 'provider_symbol',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (normalized_event_id, symbol)
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_event_raw_data_source_type'
    ) THEN
        ALTER TABLE event_raw
            ADD CONSTRAINT chk_event_raw_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_event_normalized_data_source_type'
    ) THEN
        ALTER TABLE event_normalized
            ADD CONSTRAINT chk_event_normalized_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_event_sources_enabled_priority
    ON event_sources(enabled, priority, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_event_raw_kind_time
    ON event_raw(event_kind, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_event_raw_symbol_time
    ON event_raw(symbol, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_event_raw_flow_id
    ON event_raw(flow_id);
CREATE INDEX IF NOT EXISTS idx_event_normalized_kind_time
    ON event_normalized(event_kind, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_event_normalized_source_time
    ON event_normalized(source_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_event_normalized_symbol_time
    ON event_normalized(primary_symbol, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_event_normalized_raw_event
    ON event_normalized(raw_event_id);
CREATE INDEX IF NOT EXISTS idx_event_symbol_map_symbol_time
    ON event_symbol_map(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_event_symbol_map_primary
    ON event_symbol_map(normalized_event_id, is_primary);

CREATE OR REPLACE FUNCTION update_event_sources_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_event_sources_updated_at ON event_sources;
CREATE TRIGGER trg_event_sources_updated_at
BEFORE UPDATE ON event_sources
FOR EACH ROW EXECUTE FUNCTION update_event_sources_updated_at();

INSERT INTO event_sources (id, display_name, provider_type, enabled, priority, metadata)
VALUES
    ('polygon', 'Polygon', 'external', TRUE, 10, '{"api":"polygon"}'::jsonb),
    ('finnhub', 'Finnhub', 'external', TRUE, 20, '{"api":"finnhub"}'::jsonb),
    ('calendar', 'Economic Calendar', 'calendar', TRUE, 30, '{"api":"calendar"}'::jsonb)
ON CONFLICT (id) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    provider_type = EXCLUDED.provider_type,
    enabled = EXCLUDED.enabled,
    priority = EXCLUDED.priority,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();
