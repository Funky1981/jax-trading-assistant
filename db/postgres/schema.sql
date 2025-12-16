-- Initial schema for Postgres-backed storage provider.

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  symbol TEXT NOT NULL,
  type TEXT NOT NULL,
  time TIMESTAMPTZ NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS trades (
  id TEXT PRIMARY KEY,
  symbol TEXT NOT NULL,
  direction TEXT NOT NULL,
  entry DOUBLE PRECISION NOT NULL,
  stop DOUBLE PRECISION NOT NULL,
  targets JSONB NOT NULL DEFAULT '[]'::jsonb,
  event_id TEXT NULL REFERENCES events(id),
  strategy_id TEXT NOT NULL,
  notes TEXT NULL,
  risk JSONB NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_symbol_created_at ON trades(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_strategy_created_at ON trades(strategy_id, created_at DESC);

