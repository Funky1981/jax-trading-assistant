CREATE TABLE IF NOT EXISTS world_monitor_research_inbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    world_monitor_event_id TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'new',
    rejection_reason TEXT,
    event_type TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT,
    source_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_count INTEGER NOT NULL DEFAULT 0,
    event_time TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    region TEXT,
    possible_affected_etfs JSONB NOT NULL DEFAULT '[]'::jsonb,
    asset_themes JSONB NOT NULL DEFAULT '[]'::jsonb,
    severity TEXT NOT NULL,
    source_tier TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    confidence_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    mapping_reason TEXT NOT NULL,
    dedupe_key TEXT NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE SET NULL,
    research_summary_id UUID REFERENCES research_summaries(id) ON DELETE SET NULL,
    candidate_id UUID REFERENCES candidate_trades(id) ON DELETE SET NULL,
    operator_decision TEXT,
    operator_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_world_monitor_source_event UNIQUE (source, source_event_id),
    CONSTRAINT uq_world_monitor_dedupe_key UNIQUE (dedupe_key),
    CONSTRAINT chk_world_monitor_inbox_status CHECK (
        status IN ('new', 'ignored', 'researching', 'candidate_created', 'rejected', 'archived')
    ),
    CONSTRAINT chk_world_monitor_inbox_severity CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT chk_world_monitor_inbox_source_tier CHECK (
        source_tier IN ('tier1', 'tier2', 'tier3', 'unknown')
    ),
    CONSTRAINT chk_world_monitor_confidence CHECK (
        confidence >= 0 AND confidence <= 1
    ),
    CONSTRAINT chk_world_monitor_source_count CHECK (
        source_count >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_status
    ON world_monitor_research_inbox(status, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_event_time
    ON world_monitor_research_inbox(event_time DESC);
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_normalized_event
    ON world_monitor_research_inbox(normalized_event_id)
    WHERE normalized_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_candidate
    ON world_monitor_research_inbox(candidate_id)
    WHERE candidate_id IS NOT NULL;
