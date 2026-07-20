DROP INDEX IF EXISTS idx_candles_genuine_lookup;
DROP INDEX IF EXISTS uq_candles_logical_observation;
ALTER TABLE candles DROP CONSTRAINT IF EXISTS chk_candles_genuine_source;
ALTER TABLE candles
	DROP COLUMN IF EXISTS ingested_at,
	DROP COLUMN IF EXISTS market_data_classification,
	DROP COLUMN IF EXISTS regular_trading_hours,
	DROP COLUMN IF EXISTS timestamp_semantics,
	DROP COLUMN IF EXISTS source,
	DROP COLUMN IF EXISTS timeframe;
ALTER TABLE candles
	ALTER COLUMN open TYPE DOUBLE PRECISION USING open::double precision,
	ALTER COLUMN high TYPE DOUBLE PRECISION USING high::double precision,
	ALTER COLUMN low TYPE DOUBLE PRECISION USING low::double precision,
	ALTER COLUMN close TYPE DOUBLE PRECISION USING close::double precision,
	ALTER COLUMN vwap TYPE DOUBLE PRECISION USING vwap::double precision;
