ALTER TABLE paper_fills
    DROP COLUMN IF EXISTS execution_market_provenance;

ALTER TABLE paper_orders
    DROP COLUMN IF EXISTS account_id;
