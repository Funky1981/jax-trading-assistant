ALTER TABLE candles
	ADD COLUMN IF NOT EXISTS timeframe TEXT NOT NULL DEFAULT 'unknown',
	ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'unknown',
	ADD COLUMN IF NOT EXISTS timestamp_semantics TEXT NOT NULL DEFAULT 'unknown',
	ADD COLUMN IF NOT EXISTS regular_trading_hours BOOLEAN,
	ADD COLUMN IF NOT EXISTS market_data_classification TEXT NOT NULL DEFAULT 'unknown',
	ADD COLUMN IF NOT EXISTS ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE candles
	ALTER COLUMN open TYPE NUMERIC(18,6) USING open::numeric(18,6),
	ALTER COLUMN high TYPE NUMERIC(18,6) USING high::numeric(18,6),
	ALTER COLUMN low TYPE NUMERIC(18,6) USING low::numeric(18,6),
	ALTER COLUMN close TYPE NUMERIC(18,6) USING close::numeric(18,6),
	ALTER COLUMN vwap TYPE NUMERIC(18,6) USING vwap::numeric(18,6);

ALTER TABLE candles DROP CONSTRAINT IF EXISTS chk_candles_genuine_source;
ALTER TABLE candles ADD CONSTRAINT chk_candles_genuine_source
	CHECK (UPPER(source) NOT IN ('TEST', 'SYNTHETIC', 'FIXTURE'));

CREATE UNIQUE INDEX IF NOT EXISTS uq_candles_logical_observation
	ON candles(symbol, timeframe, source, timestamp);

CREATE INDEX IF NOT EXISTS idx_candles_genuine_lookup
	ON candles(symbol, timeframe, timestamp DESC)
	WHERE source <> 'unknown';
