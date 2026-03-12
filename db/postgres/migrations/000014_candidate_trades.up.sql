-- candidate_trades: tracks setups detected by the always-on watcher.
-- A candidate is distinct from a final executed trade.

CREATE TABLE IF NOT EXISTS candidate_trades (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_instance_id UUID NOT NULL,
    symbol              TEXT NOT NULL,
    signal_type         TEXT NOT NULL CHECK (signal_type IN ('BUY', 'SELL')),
    status              TEXT NOT NULL DEFAULT 'detected'
                            CHECK (status IN (
                                'detected', 'qualified', 'blocked',
                                'awaiting_approval', 'approved', 'rejected',
                                'expired', 'submitted', 'filled', 'cancelled'
                            )),
    entry_price         NUMERIC(18,6),
    stop_loss           NUMERIC(18,6),
    take_profit         NUMERIC(18,6),
    confidence          NUMERIC(5,4) CHECK (confidence >= 0 AND confidence <= 1),
    reasoning           TEXT,
    block_reason        TEXT,
    session_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    expires_at          TIMESTAMPTZ,
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    qualified_at        TIMESTAMPTZ,
    blocked_at          TIMESTAMPTZ,
    submitted_at        TIMESTAMPTZ,
    filled_at           TIMESTAMPTZ,
    data_provenance     TEXT NOT NULL DEFAULT 'unknown',
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent duplicate open candidates for the same symbol/instance
CREATE UNIQUE INDEX IF NOT EXISTS uq_candidate_open_per_instance_symbol
    ON candidate_trades (strategy_instance_id, symbol, session_date)
    WHERE status IN ('detected', 'qualified', 'awaiting_approval', 'approved');

CREATE INDEX IF NOT EXISTS idx_candidate_trades_status ON candidate_trades (status);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_symbol ON candidate_trades (symbol);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_instance ON candidate_trades (strategy_instance_id);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_detected_at ON candidate_trades (detected_at DESC);

-- candidate_events: audit log of status transitions
CREATE TABLE IF NOT EXISTS candidate_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id    UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    from_status     TEXT,
    to_status       TEXT,
    detail          JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_candidate_events_candidate ON candidate_events (candidate_id, occurred_at DESC);
