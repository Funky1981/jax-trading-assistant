ALTER TABLE quotes
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ;

UPDATE quotes
SET received_at = updated_at
WHERE received_at IS NULL;

ALTER TABLE quotes
    ALTER COLUMN received_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_quotes_provider_timestamp
    ON quotes(provider, timestamp DESC, received_at DESC);
