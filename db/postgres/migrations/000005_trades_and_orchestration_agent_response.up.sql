-- Extend trades and orchestration runs for paper trading lifecycle tracking

ALTER TABLE trades
  ADD COLUMN IF NOT EXISTS signal_id UUID,
  ADD COLUMN IF NOT EXISTS order_status TEXT,
  ADD COLUMN IF NOT EXISTS filled_qty INTEGER,
  ADD COLUMN IF NOT EXISTS avg_fill_price DOUBLE PRECISION;

CREATE INDEX IF NOT EXISTS idx_trades_signal_id ON trades(signal_id);

ALTER TABLE orchestration_runs
  ADD COLUMN IF NOT EXISTS agent_response JSONB;

