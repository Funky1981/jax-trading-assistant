ALTER TABLE paper_orders
    ADD COLUMN IF NOT EXISTS account_id TEXT;

ALTER TABLE paper_fills
    ADD COLUMN IF NOT EXISTS execution_market_provenance JSONB;
