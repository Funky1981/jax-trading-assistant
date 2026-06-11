CREATE TABLE IF NOT EXISTS market_regime_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    as_of_utc TIMESTAMPTZ NOT NULL,
    primary_regime TEXT NOT NULL,
    secondary_regimes TEXT[] NOT NULL DEFAULT '{}',
    confidence NUMERIC NOT NULL,
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    missing_inputs TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_market_regime_snapshots_primary CHECK (
        primary_regime IN (
            'risk_on',
            'risk_off',
            'high_volatility',
            'low_volatility',
            'rates_dominant',
            'growth_dominant',
            'inflation_fear',
            'recession_fear',
            'liquidity_stress',
            'tech_momentum',
            'defensive_rotation',
            'unclear'
        )
    ),
    CONSTRAINT chk_market_regime_snapshots_confidence CHECK (
        confidence >= 0 AND confidence <= 1
    )
);

CREATE TABLE IF NOT EXISTS cross_asset_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NULL REFERENCES macro_events(id) ON DELETE SET NULL,
    playbook_key TEXT NOT NULL,
    as_of_utc TIMESTAMPTZ NOT NULL,
    confirmation_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    asset_results JSONB NOT NULL DEFAULT '{}'::jsonb,
    conflicts TEXT[] NOT NULL DEFAULT '{}',
    missing_assets TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_cross_asset_confirmations_score CHECK (
        confirmation_score >= 0 AND confirmation_score <= 100
    ),
    CONSTRAINT chk_cross_asset_confirmations_verdict CHECK (
        verdict IN (
            'confirmed',
            'partially_confirmed',
            'conflicted',
            'insufficient_data',
            'not_confirmed'
        )
    )
);

CREATE TABLE IF NOT EXISTS economic_calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    country TEXT NOT NULL,
    release_time_utc TIMESTAMPTZ NOT NULL,
    actual NUMERIC NULL,
    forecast NUMERIC NULL,
    previous NUMERIC NULL,
    revised_previous NUMERIC NULL,
    unit TEXT NULL,
    importance TEXT NOT NULL,
    source_url TEXT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_event_id)
);

ALTER TABLE macro_events
    ADD COLUMN IF NOT EXISTS economic_calendar_event_id UUID NULL REFERENCES economic_calendar_events(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS confounder_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    related_event_id UUID NULL,
    confounder_type TEXT NOT NULL,
    affected_symbols TEXT[] NOT NULL DEFAULT '{}',
    headline TEXT NOT NULL,
    event_time_utc TIMESTAMPTZ NOT NULL,
    severity TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    reason TEXT NOT NULL,
    source TEXT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_confounder_events_severity CHECK (
        severity IN ('info', 'low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT chk_confounder_events_confidence CHECK (
        confidence >= 0 AND confidence <= 1
    )
);

CREATE TABLE IF NOT EXISTS event_confounder_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    confounder_event_id UUID NOT NULL REFERENCES confounder_events(id) ON DELETE CASCADE,
    impact TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, confounder_event_id),
    CONSTRAINT chk_event_confounder_links_impact CHECK (
        impact IN (
            'blocks_trade',
            'reduces_confidence',
            'requires_manual_review',
            'reassigns_cause',
            'informational_only'
        )
    )
);

CREATE TABLE IF NOT EXISTS execution_quality_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    as_of_utc TIMESTAMPTZ NOT NULL,
    spread_percent NUMERIC NULL,
    volume_ok BOOLEAN NOT NULL,
    slippage_estimate_percent NUMERIC NULL,
    market_data_fresh BOOLEAN NOT NULL,
    broker_available BOOLEAN NOT NULL,
    event_volatility_state TEXT NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_execution_quality_snapshots_verdict CHECK (
        verdict IN ('good', 'acceptable', 'poor', 'blocked', 'insufficient_data')
    )
);

CREATE TABLE IF NOT EXISTS position_size_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NULL,
    symbol TEXT NOT NULL,
    account_equity NUMERIC NOT NULL,
    entry_price NUMERIC NOT NULL,
    stop_price NUMERIC NOT NULL,
    risk_percent NUMERIC NOT NULL,
    cash_risk NUMERIC NOT NULL,
    position_size NUMERIC NOT NULL,
    adjusted_risk_percent NUMERIC NOT NULL,
    adjustments TEXT[] NOT NULL DEFAULT '{}',
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_position_size_recommendations_verdict CHECK (
        verdict IN ('allowed', 'reduced', 'blocked', 'insufficient_data')
    ),
    CONSTRAINT chk_position_size_recommendations_risk CHECK (
        risk_percent >= 0 AND adjusted_risk_percent >= 0
    )
);

CREATE TABLE IF NOT EXISTS strategy_playbook_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NULL,
    candidate_id UUID NULL,
    playbook_key TEXT NOT NULL,
    matched BOOLEAN NOT NULL,
    result TEXT NOT NULL,
    reasons TEXT[] NOT NULL DEFAULT '{}',
    failed_rules TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_strategy_playbook_results_result CHECK (
        result IN (
            'matched_allowed',
            'matched_watch_only',
            'matched_blocked',
            'no_strategy_match'
        )
    )
);

CREATE TABLE IF NOT EXISTS walkaway_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NULL,
    symbol TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    evidence_refs JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_walkaway_decisions_severity CHECK (
        severity IN ('info', 'warning', 'blocker', 'critical')
    )
);

CREATE TABLE IF NOT EXISTS trade_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    strategy_key TEXT NOT NULL,
    entry_price NUMERIC NULL,
    exit_price NUMERIC NULL,
    stop_price NUMERIC NULL,
    target_price NUMERIC NULL,
    mfe_r NUMERIC NULL,
    mae_r NUMERIC NULL,
    final_r NUMERIC NULL,
    outcome TEXT NOT NULL,
    what_worked TEXT[] NOT NULL DEFAULT '{}',
    what_failed TEXT[] NOT NULL DEFAULT '{}',
    lesson TEXT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_trade_reviews_outcome CHECK (
        outcome IN (
            'win',
            'loss',
            'breakeven',
            'avoided_good_trade',
            'avoided_bad_trade',
            'invalidated_before_entry',
            'manual_reject_correct',
            'manual_reject_incorrect'
        )
    )
);

CREATE TABLE IF NOT EXISTS risk_simulation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_key TEXT NULL,
    input_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    simulation_count INT NOT NULL,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_risk_simulation_runs_count CHECK (
        simulation_count >= 0
    )
);

CREATE OR REPLACE VIEW strategy_performance_summary AS
SELECT
    strategy_key,
    COUNT(*)::INT AS trades,
    AVG(final_r)::NUMERIC AS avg_r,
    AVG(CASE WHEN final_r > 0 THEN 1.0 ELSE 0.0 END)::NUMERIC AS win_rate
FROM trade_reviews
GROUP BY strategy_key;

CREATE OR REPLACE VIEW robust_event_funnel_summary AS
SELECT
    (SELECT COUNT(*) FROM macro_events)::INT AS events_analyzed,
    (SELECT COUNT(*) FROM macro_candidate_trades)::INT AS candidates_created,
    (SELECT COUNT(*) FROM walkaway_decisions WHERE severity IN ('blocker', 'critical'))::INT AS blocking_walkaways,
    (SELECT COUNT(*) FROM trade_reviews)::INT AS reviewed_trades;

CREATE INDEX IF NOT EXISTS idx_market_regime_snapshots_as_of
    ON market_regime_snapshots(as_of_utc DESC);
CREATE INDEX IF NOT EXISTS idx_cross_asset_confirmations_event
    ON cross_asset_confirmations(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_economic_calendar_events_type_time
    ON economic_calendar_events(event_type, country, release_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_confounder_events_time
    ON confounder_events(event_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_event_confounder_links_event
    ON event_confounder_links(event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_execution_quality_snapshots_symbol
    ON execution_quality_snapshots(symbol, as_of_utc DESC);
CREATE INDEX IF NOT EXISTS idx_position_size_recommendations_candidate
    ON position_size_recommendations(candidate_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_playbook_results_candidate
    ON strategy_playbook_results(candidate_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_walkaway_decisions_event
    ON walkaway_decisions(event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trade_reviews_strategy
    ON trade_reviews(strategy_key, reviewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_simulation_runs_strategy
    ON risk_simulation_runs(strategy_key, created_at DESC);
