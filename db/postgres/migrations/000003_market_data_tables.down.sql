-- Rollback market data tables

DROP INDEX IF EXISTS idx_candles_timestamp;
DROP INDEX IF EXISTS idx_candles_symbol_timestamp;
DROP TABLE IF EXISTS candles;

DROP INDEX IF EXISTS idx_quotes_updated_at;
DROP TABLE IF EXISTS quotes;
