CREATE TABLE IF NOT EXISTS research_memory_entries (
    memory_id TEXT PRIMARY KEY CHECK (memory_id ~ '^memory_[0-9a-f]{64}$'),
    validity_digest TEXT NOT NULL CHECK (validity_digest ~ '^[0-9a-f]{64}$'),
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    UNIQUE (validity_digest)
);

COMMENT ON TABLE research_memory_entries IS 'Exact provenance-keyed research memory; semantic result caching is intentionally disabled.';
