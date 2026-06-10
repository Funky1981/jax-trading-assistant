CREATE TABLE IF NOT EXISTS macro_candidate_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    evidence_bundle_id UUID NOT NULL REFERENCES macro_evidence_bundles(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    bias TEXT NOT NULL,
    entry_type TEXT NOT NULL,
    entry_reference_price NUMERIC NOT NULL,
    stop_reference_price NUMERIC NOT NULL,
    target_reference_price NUMERIC NOT NULL,
    risk_percent NUMERIC NOT NULL,
    time_limit TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'awaiting_human_approval',
    created_reason TEXT NOT NULL,
    rejection_reason TEXT,
    walkaway_reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, evidence_bundle_id, symbol),
    CONSTRAINT chk_macro_candidate_side CHECK (
        side IN ('long', 'short_bias', 'watch_only', 'no_trade')
    ),
    CONSTRAINT chk_macro_candidate_entry_type CHECK (
        entry_type IN ('breakout_continuation', 'pullback_retest', 'range_reclaim', 'no_entry')
    ),
    CONSTRAINT chk_macro_candidate_status CHECK (
        status IN ('awaiting_human_approval', 'watch_only', 'blocked', 'rejected', 'archived')
    ),
    CONSTRAINT chk_macro_candidate_risk CHECK (
        risk_percent >= 0 AND risk_percent <= 0.5
    )
);

CREATE INDEX IF NOT EXISTS idx_macro_candidate_trades_status
    ON macro_candidate_trades(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_candidate_trades_event
    ON macro_candidate_trades(macro_event_id, created_at DESC);
