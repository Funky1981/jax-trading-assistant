DROP INDEX IF EXISTS idx_quotes_provider_timestamp;

ALTER TABLE quotes
    DROP COLUMN IF EXISTS received_at,
    DROP COLUMN IF EXISTS provider;
