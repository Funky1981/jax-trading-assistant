-- Rollback initial schema

DROP INDEX IF EXISTS idx_trades_event_id;
DROP INDEX IF EXISTS idx_trades_direction;
DROP INDEX IF EXISTS idx_trades_strategy_created_at;
DROP INDEX IF EXISTS idx_trades_symbol_created_at;
DROP INDEX IF EXISTS idx_trades_created_at;

DROP TABLE IF EXISTS trades;

DROP INDEX IF EXISTS idx_events_symbol_time;
DROP INDEX IF EXISTS idx_events_time;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_symbol;

DROP TABLE IF EXISTS events;
