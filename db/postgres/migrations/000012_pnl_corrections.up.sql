-- Add PnL correction append model for reconciliation (Gate 5)

CREATE TABLE IF NOT EXISTS pnl_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id TEXT REFERENCES trades(id) ON DELETE SET NULL,
    delta DOUBLE PRECISION NOT NULL,
    reason TEXT NOT NULL,
    source TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pnl_corrections_trade_id ON pnl_corrections(trade_id);
CREATE INDEX IF NOT EXISTS idx_pnl_corrections_created_at ON pnl_corrections(created_at DESC);
