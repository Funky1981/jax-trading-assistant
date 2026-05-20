-- Historical ETF event-study schema.
-- Reuses event_normalized, candles, candidate_trades, and related foundation tables.

CREATE TABLE IF NOT EXISTS event_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    window_name TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    price_before NUMERIC,
    price_after NUMERIC,
    return_pct NUMERIC,
    benchmark_symbol TEXT,
    benchmark_return_pct NUMERIC,
    abnormal_return_pct NUMERIC,
    volume_before NUMERIC,
    volume_after NUMERIC,
    volume_change_pct NUMERIC,
    spread_before_bps NUMERIC,
    spread_after_bps NUMERIC,
    volatility_adjusted_move NUMERIC,
    data_quality TEXT NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_event_windows_event_symbol_window UNIQUE (event_id, symbol, window_name),
    CONSTRAINT chk_event_windows_window_order CHECK (window_end > window_start),
    CONSTRAINT chk_event_windows_data_quality CHECK (
        data_quality IN ('complete', 'partial', 'missing', 'synthetic', 'unknown')
    )
);

CREATE TABLE IF NOT EXISTS event_confounders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    confounding_event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    time_distance_minutes INTEGER NOT NULL,
    relevance_score NUMERIC NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_event_confounders_event_confounding_symbol UNIQUE (event_id, confounding_event_id, symbol),
    CONSTRAINT chk_event_confounders_relationship_type CHECK (
        relationship_type IN (
            'macro',
            'sector',
            'company',
            'geopolitical',
            'earnings',
            'rates',
            'commodity',
            'credit',
            'unknown'
        )
    ),
    CONSTRAINT chk_event_confounders_relevance_score CHECK (
        relevance_score >= 0 AND relevance_score <= 1
    )
);

CREATE TABLE IF NOT EXISTS event_priced_in_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    pre_event_1h_return NUMERIC,
    pre_event_4h_return NUMERIC,
    pre_event_1d_return NUMERIC,
    post_event_15m_return NUMERIC,
    post_event_1h_return NUMERIC,
    benchmark_symbol TEXT,
    benchmark_return NUMERIC,
    abnormal_return NUMERIC,
    volume_confirmation_score NUMERIC,
    spread_quality_score NUMERIC,
    priced_in_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_event_priced_in_scores_event_symbol UNIQUE (event_id, symbol),
    CONSTRAINT chk_event_priced_in_scores_score CHECK (
        priced_in_score >= 0 AND priced_in_score <= 1
    ),
    CONSTRAINT chk_event_priced_in_scores_verdict CHECK (
        verdict IN (
            'not_priced_in',
            'partially_priced_in',
            'priced_in',
            'overreaction',
            'unclear'
        )
    )
);

CREATE TABLE IF NOT EXISTS etf_context_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    theme TEXT NOT NULL,
    sector TEXT,
    benchmark_symbol TEXT,
    related_symbols JSONB NOT NULL DEFAULT '[]'::jsonb,
    macro_sensitivity JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes TEXT,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_etf_context_snapshots_validity CHECK (
        valid_to IS NULL OR valid_to > valid_from
    )
);

CREATE TABLE IF NOT EXISTS research_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES event_normalized(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    strategy_id TEXT,
    summary TEXT NOT NULL,
    why_this_etf TEXT NOT NULL,
    what_happened TEXT NOT NULL,
    what_else_mattered TEXT,
    priced_in_view TEXT NOT NULL,
    risk_notes TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_windows_symbol_window
    ON event_windows(symbol, window_name);
CREATE INDEX IF NOT EXISTS idx_event_windows_event_symbol
    ON event_windows(event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_event_windows_symbol_time
    ON event_windows(symbol, window_start DESC, window_end DESC);

CREATE INDEX IF NOT EXISTS idx_event_confounders_event_symbol
    ON event_confounders(event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_event_confounders_symbol_relationship
    ON event_confounders(symbol, relationship_type);
CREATE INDEX IF NOT EXISTS idx_event_confounders_confounding_event
    ON event_confounders(confounding_event_id);

CREATE INDEX IF NOT EXISTS idx_event_priced_in_scores_event_symbol
    ON event_priced_in_scores(event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_event_priced_in_scores_symbol_verdict
    ON event_priced_in_scores(symbol, verdict);

CREATE INDEX IF NOT EXISTS idx_etf_context_snapshots_symbol_validity
    ON etf_context_snapshots(symbol, valid_from DESC, valid_to);
CREATE INDEX IF NOT EXISTS idx_etf_context_snapshots_theme
    ON etf_context_snapshots(theme);

CREATE INDEX IF NOT EXISTS idx_research_summaries_event_symbol
    ON research_summaries(event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_research_summaries_symbol_created
    ON research_summaries(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_research_summaries_strategy_created
    ON research_summaries(strategy_id, created_at DESC)
    WHERE strategy_id IS NOT NULL;
