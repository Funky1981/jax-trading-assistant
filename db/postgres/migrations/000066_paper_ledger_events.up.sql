CREATE TABLE IF NOT EXISTS paper_accounts (
    account_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    environment TEXT NOT NULL CHECK (environment = 'PAPER'),
    currency TEXT NOT NULL,
    initial_cash NUMERIC NOT NULL CHECK (initial_cash > 0),
    cash NUMERIC NOT NULL CHECK (cash >= 0),
    equity NUMERIC NOT NULL,
    realized_pnl NUMERIC NOT NULL,
    fees NUMERIC NOT NULL CHECK (fees >= 0),
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS paper_ledger_events (
    event_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES paper_accounts(account_id) ON DELETE RESTRICT,
    fill_id TEXT NOT NULL UNIQUE REFERENCES paper_fills(fill_id) ON DELETE RESTRICT,
    order_id TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    instrument_id TEXT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('LONG', 'SHORT')),
    quantity NUMERIC NOT NULL CHECK (quantity > 0),
    price NUMERIC NOT NULL CHECK (price > 0),
    fee NUMERIC NOT NULL CHECK (fee >= 0),
    cash_delta NUMERIC NOT NULL,
    realized_pnl NUMERIC NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE OR REPLACE FUNCTION reject_paper_ledger_event_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'paper ledger events are append-only';
END;
$$;

CREATE TRIGGER trg_paper_ledger_events_append_only
    BEFORE UPDATE OR DELETE ON paper_ledger_events
    FOR EACH ROW EXECUTE FUNCTION reject_paper_ledger_event_mutation();
