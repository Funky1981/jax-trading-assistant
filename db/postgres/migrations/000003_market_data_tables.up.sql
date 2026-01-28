-- Add market data tables for quotes and candles

CREATE TABLE IF NOT EXISTS quotes (
  symbol TEXT PRIMARY KEY,
  price DOUBLE PRECISION NOT NULL,
  bid DOUBLE PRECISION,
  ask DOUBLE PRECISION,
  bid_size BIGINT,
  ask_size BIGINT,
  volume BIGINT,
  timestamp TIMESTAMPTZ NOT NULL,
  exchange TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_quotes_updated_at ON quotes(updated_at DESC);

CREATE TABLE IF NOT EXISTS candles (
  symbol TEXT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  open DOUBLE PRECISION NOT NULL,
  high DOUBLE PRECISION NOT NULL,
  low DOUBLE PRECISION NOT NULL,
  close DOUBLE PRECISION NOT NULL,
  volume BIGINT NOT NULL,
  vwap DOUBLE PRECISION,
  PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_candles_symbol_timestamp ON candles(symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_candles_timestamp ON candles(timestamp DESC);
