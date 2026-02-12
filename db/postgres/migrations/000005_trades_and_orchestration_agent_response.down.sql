-- Roll back trades and orchestration run extensions

ALTER TABLE orchestration_runs
  DROP COLUMN IF EXISTS agent_response;

ALTER TABLE trades
  DROP COLUMN IF EXISTS avg_fill_price,
  DROP COLUMN IF EXISTS filled_qty,
  DROP COLUMN IF EXISTS order_status,
  DROP COLUMN IF EXISTS signal_id;

