DROP INDEX IF EXISTS idx_exec_instructions_trade;

ALTER TABLE execution_instructions
    DROP COLUMN IF EXISTS trade_id;

DROP INDEX IF EXISTS idx_candidate_trades_blocked_reason_code;
DROP INDEX IF EXISTS idx_candidate_trades_artifact;
DROP INDEX IF EXISTS idx_candidate_trades_strategy;
DROP INDEX IF EXISTS idx_candidate_trades_signal;

ALTER TABLE candidate_trades
    DROP COLUMN IF EXISTS blocked_reason_code,
    DROP COLUMN IF EXISTS artifact_id,
    DROP COLUMN IF EXISTS strategy_id,
    DROP COLUMN IF EXISTS signal_id;
