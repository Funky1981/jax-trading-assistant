CREATE TABLE IF NOT EXISTS paper_fills (
    fill_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    order_id TEXT NOT NULL REFERENCES paper_orders(order_id) ON DELETE RESTRICT,
    paper_intent_id TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    instrument_id TEXT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('LONG', 'SHORT')),
    quantity NUMERIC NOT NULL CHECK (quantity > 0),
    price NUMERIC NOT NULL CHECK (price > 0),
    cost_model_id TEXT NOT NULL,
    mid_price NUMERIC NOT NULL CHECK (mid_price > 0),
    spread_cost NUMERIC NOT NULL CHECK (spread_cost >= 0),
    slippage_cost NUMERIC NOT NULL CHECK (slippage_cost >= 0),
    commission NUMERIC NOT NULL CHECK (commission >= 0),
    filled_at TIMESTAMPTZ NOT NULL,
    tick_id TEXT NOT NULL,
    UNIQUE (order_id, tick_id)
);

CREATE OR REPLACE FUNCTION reject_paper_fill_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'paper fills are append-only';
END;
$$;

CREATE TRIGGER trg_paper_fills_append_only
    BEFORE UPDATE OR DELETE ON paper_fills
    FOR EACH ROW EXECUTE FUNCTION reject_paper_fill_mutation();
