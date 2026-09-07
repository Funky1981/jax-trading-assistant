CREATE TABLE IF NOT EXISTS world_monitor_pull_pages (
    page_id BIGSERIAL PRIMARY KEY,
    consumer_name TEXT NOT NULL,
    source_endpoint_identity TEXT NOT NULL,
    after_position BIGINT NOT NULL CHECK (after_position >= 0),
    next_position BIGINT NOT NULL CHECK (next_position >= after_position),
    event_count INTEGER NOT NULL CHECK (event_count > 0),
    page_digest TEXT NOT NULL CHECK (page_digest ~ '^[0-9a-f]{64}$'),
    raw_payload BYTEA NOT NULL CHECK (octet_length(raw_payload) > 0),
    provider_schema_version TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL,
    UNIQUE (consumer_name, source_endpoint_identity, after_position, page_digest)
);

COMMENT ON TABLE world_monitor_pull_pages IS 'Append-only exact provider response bytes for durable World Monitor cursor pages; normalized event rows are separate.';
COMMENT ON COLUMN world_monitor_pull_pages.page_digest IS 'SHA-256 of raw_payload bytes, before JSON parsing or normalization.';
