-- candidate_approvals: records human approval decisions for candidate trades.

CREATE TABLE IF NOT EXISTS candidate_approvals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
    decision            TEXT NOT NULL CHECK (decision IN ('approved', 'rejected', 'snoozed', 'reanalysis_requested')),
    approved_by         TEXT NOT NULL,
    notes               TEXT,
    expiry_at           TIMESTAMPTZ,
    snooze_until        TIMESTAMPTZ,
    reanalysis_requested BOOLEAN NOT NULL DEFAULT FALSE,
    decided_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_candidate_approvals_candidate ON candidate_approvals (candidate_id);
CREATE INDEX IF NOT EXISTS idx_candidate_approvals_decision ON candidate_approvals (decision);
CREATE INDEX IF NOT EXISTS idx_candidate_approvals_decided_at ON candidate_approvals (decided_at DESC);

-- execution_instructions: approved candidates that are ready for the execution engine.
-- Only populated after explicit human approval.

CREATE TABLE IF NOT EXISTS execution_instructions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    approval_id         UUID NOT NULL REFERENCES candidate_approvals(id) ON DELETE RESTRICT,
    candidate_id        UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE RESTRICT,
    symbol              TEXT NOT NULL,
    signal_type         TEXT NOT NULL CHECK (signal_type IN ('BUY', 'SELL')),
    entry_price         NUMERIC(18,6),
    stop_loss           NUMERIC(18,6),
    take_profit         NUMERIC(18,6),
    status              TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'submitted', 'filled', 'rejected', 'cancelled')),
    broker_order_id     TEXT,
    fill_price          NUMERIC(18,6),
    fill_qty            INTEGER,
    error_message       TEXT,
    submitted_at        TIMESTAMPTZ,
    filled_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exec_instructions_candidate ON execution_instructions (candidate_id);
CREATE INDEX IF NOT EXISTS idx_exec_instructions_status ON execution_instructions (status);
CREATE INDEX IF NOT EXISTS idx_exec_instructions_created ON execution_instructions (created_at DESC);
