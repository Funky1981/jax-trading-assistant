ALTER TABLE candidate_trades
    ADD COLUMN IF NOT EXISTS signal_id UUID REFERENCES strategy_signals(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS strategy_id TEXT,
    ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS blocked_reason_code TEXT;

CREATE INDEX IF NOT EXISTS idx_candidate_trades_signal ON candidate_trades (signal_id);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_strategy ON candidate_trades (strategy_id);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_artifact ON candidate_trades (artifact_id);
CREATE INDEX IF NOT EXISTS idx_candidate_trades_blocked_reason_code ON candidate_trades (blocked_reason_code);

ALTER TABLE execution_instructions
    ADD COLUMN IF NOT EXISTS trade_id TEXT REFERENCES trades(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_exec_instructions_trade ON execution_instructions (trade_id);
