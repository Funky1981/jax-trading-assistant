CREATE TABLE IF NOT EXISTS news_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_event_id TEXT,
    source_provider TEXT NOT NULL DEFAULT 'jax',
    source_family TEXT NOT NULL,
    symbol TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    url TEXT,
    author TEXT,
    published_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_news_items_symbol_published ON news_items(symbol, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_items_source_family ON news_items(source_family);

CREATE TABLE IF NOT EXISTS sentiment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    news_item_id UUID NOT NULL REFERENCES news_items(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    provider_mode TEXT NOT NULL,
    provider_name TEXT NOT NULL DEFAULT 'jax-local',
    score NUMERIC NOT NULL,
    label TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    drivers JSONB NOT NULL DEFAULT '[]'::jsonb,
    limitations JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    scored_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    degraded BOOLEAN NOT NULL DEFAULT FALSE,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_sentiment_events_symbol_scored ON sentiment_events(symbol, scored_at DESC);
CREATE INDEX IF NOT EXISTS idx_sentiment_events_news_item ON sentiment_events(news_item_id);

CREATE TABLE IF NOT EXISTS sentiment_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    window TEXT NOT NULL,
    provider_mode TEXT NOT NULL,
    score NUMERIC NOT NULL DEFAULT 0,
    label TEXT NOT NULL DEFAULT 'mixed',
    confidence NUMERIC NOT NULL DEFAULT 0,
    source_count INT NOT NULL DEFAULT 0,
    source_groups JSONB NOT NULL DEFAULT '{}'::jsonb,
    price_agreement TEXT NOT NULL DEFAULT 'unknown',
    top_drivers JSONB NOT NULL DEFAULT '[]'::jsonb,
    limitations JSONB NOT NULL DEFAULT '[]'::jsonb,
    state TEXT NOT NULL DEFAULT 'available',
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    stale_after TIMESTAMPTZ,
    UNIQUE(symbol, window, provider_mode)
);

CREATE INDEX IF NOT EXISTS idx_sentiment_aggregates_symbol_window ON sentiment_aggregates(symbol, window);
CREATE INDEX IF NOT EXISTS idx_sentiment_aggregates_state ON sentiment_aggregates(state);

CREATE TABLE IF NOT EXISTS opportunity_sentiment_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_type TEXT NOT NULL,
    opportunity_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    aggregate_id UUID REFERENCES sentiment_aggregates(id) ON DELETE SET NULL,
    snapshot JSONB NOT NULL,
    snapshot_reason TEXT NOT NULL DEFAULT 'created',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_opportunity_sentiment_snapshots_opportunity
    ON opportunity_sentiment_snapshots(opportunity_type, opportunity_id, created_at DESC);

CREATE TABLE IF NOT EXISTS sentiment_alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL DEFAULT 'default',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    trigger_type TEXT NOT NULL,
    symbol TEXT,
    minimum_move NUMERIC NOT NULL DEFAULT 0.25,
    minimum_confidence NUMERIC NOT NULL DEFAULT 0.5,
    channels JSONB NOT NULL DEFAULT '["in_app"]'::jsonb,
    cooldown_seconds INT NOT NULL DEFAULT 86400,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_key TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    severity TEXT,
    primary_symbol TEXT,
    sentiment_trigger_type TEXT,
    entity_type TEXT,
    entity_id TEXT,
    route TEXT NOT NULL DEFAULT '/ai-trading',
    channels JSONB NOT NULL DEFAULT '["in_app"]'::jsonb,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(identity_key)
);

CREATE INDEX IF NOT EXISTS idx_notification_events_created ON notification_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_events_kind ON notification_events(kind);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id TEXT PRIMARY KEY DEFAULT 'default',
    in_app_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    desktop_web_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mobile_push_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sentiment_channels JSONB NOT NULL DEFAULT '{"in_app": true, "desktop_web": false, "mobile_push": false}'::jsonb,
    browser_permission_state TEXT NOT NULL DEFAULT 'unknown',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS approval_override_reasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
    approval_id UUID REFERENCES candidate_approvals(id) ON DELETE SET NULL,
    decision TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    note_redacted TEXT,
    sentiment_evidence_viewed BOOLEAN NOT NULL DEFAULT FALSE,
    sentiment_snapshot_id UUID REFERENCES opportunity_sentiment_snapshots(id) ON DELETE SET NULL,
    decided_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_approval_override_reasons_candidate ON approval_override_reasons(candidate_id);
CREATE INDEX IF NOT EXISTS idx_approval_override_reasons_code ON approval_override_reasons(reason_code);

CREATE TABLE IF NOT EXISTS sentiment_paper_live_handoffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID REFERENCES backtest_runs(id) ON DELETE SET NULL,
    target_mode TEXT NOT NULL DEFAULT 'paper',
    setup_name TEXT NOT NULL,
    sentiment_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    live_ready BOOLEAN NOT NULL DEFAULT FALSE,
    created_by TEXT NOT NULL DEFAULT 'operator',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sentiment_paper_live_handoffs_backtest ON sentiment_paper_live_handoffs(backtest_run_id);
