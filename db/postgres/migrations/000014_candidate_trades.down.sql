DROP INDEX IF EXISTS idx_candidate_events_candidate;
DROP TABLE IF EXISTS candidate_events;
DROP INDEX IF EXISTS idx_candidate_trades_detected_at;
DROP INDEX IF EXISTS idx_candidate_trades_instance;
DROP INDEX IF EXISTS idx_candidate_trades_symbol;
DROP INDEX IF EXISTS idx_candidate_trades_status;
DROP INDEX IF EXISTS uq_candidate_open_per_instance_symbol;
DROP TABLE IF EXISTS candidate_trades;
