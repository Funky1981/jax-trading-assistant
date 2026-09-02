CREATE TABLE IF NOT EXISTS raw_payload_contents (
    digest_algorithm TEXT NOT NULL,
    content_digest TEXT PRIMARY KEY,
    representation TEXT NOT NULL,
    payload BYTEA NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_raw_payload_content_algorithm CHECK (digest_algorithm = 'sha256'),
    CONSTRAINT chk_raw_payload_content_digest CHECK (content_digest ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_raw_payload_content_representation CHECK (representation = 'raw_bytes'),
    CONSTRAINT chk_raw_payload_content_size CHECK (size_bytes > 0),
    CONSTRAINT chk_raw_payload_content_byte_count CHECK (octet_length(payload) = size_bytes)
);

CREATE TABLE IF NOT EXISTS raw_payload_acquisitions (
    payload_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    content_digest TEXT NOT NULL REFERENCES raw_payload_contents(content_digest) ON DELETE RESTRICT,
    content_algorithm TEXT NOT NULL,
    content_representation TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    provider_namespace TEXT NOT NULL,
    provider_external_id JSONB,
    capability_id TEXT NOT NULL,
    raw_boundary TEXT NOT NULL,
    raw_format TEXT NOT NULL,
    raw_schema_namespace TEXT NOT NULL,
    raw_schema_value TEXT NOT NULL,
    raw_media_type TEXT,
    capture_byte_form TEXT NOT NULL,
    capture_content_coding TEXT,
    capture_content_coding_state TEXT NOT NULL,
    capture_character_encoding TEXT,
    capture_related_to JSONB,
    source_id TEXT,
    source_kind TEXT,
    revision_namespace TEXT,
    revision_value TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    size_bytes BIGINT NOT NULL,
    retention_class TEXT NOT NULL,
    retention_policy_namespace TEXT NOT NULL,
    retention_policy_value TEXT NOT NULL,
    retention_redistribution TEXT NOT NULL,
    reference JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_raw_payload_acquisition_id CHECK (payload_id ~ '^rpa_[A-Za-z0-9][A-Za-z0-9_.-]*$'),
    CONSTRAINT chk_raw_payload_acquisition_version CHECK (contract_version = 'jax.provider_raw_payload_ref/v1'),
    CONSTRAINT chk_raw_payload_acquisition_content_algorithm CHECK (content_algorithm = 'sha256'),
    CONSTRAINT chk_raw_payload_acquisition_content_representation CHECK (content_representation = 'raw_bytes'),
    CONSTRAINT chk_raw_payload_acquisition_size CHECK (size_bytes > 0),
    CONSTRAINT chk_raw_payload_acquisition_source_pair CHECK ((source_id IS NULL) = (source_kind IS NULL)),
    CONSTRAINT chk_raw_payload_acquisition_revision_pair CHECK ((revision_namespace IS NULL) = (revision_value IS NULL)),
    CONSTRAINT chk_raw_payload_acquisition_coding CHECK (
        (capture_content_coding_state = 'IDENTITY' AND capture_content_coding IS NULL)
        OR (capture_content_coding_state IN ('ENCODED', 'DECODED') AND capture_content_coding IS NOT NULL)
    ),
    CONSTRAINT chk_raw_payload_acquisition_retention CHECK (retention_class = 'REPLAY_AUDIT_REQUIRED'),
    CONSTRAINT chk_raw_payload_acquisition_redistribution CHECK (retention_redistribution IN ('NOT_AUTHORIZED', 'RESTRICTED', 'AUTHORIZED'))
);

CREATE INDEX IF NOT EXISTS idx_raw_payload_acquisitions_content_digest
    ON raw_payload_acquisitions(content_digest);
CREATE INDEX IF NOT EXISTS idx_raw_payload_acquisitions_provider_received
    ON raw_payload_acquisitions(provider_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_raw_payload_acquisitions_source_revision
    ON raw_payload_acquisitions(source_id, revision_namespace, revision_value);

CREATE OR REPLACE FUNCTION protect_raw_payload_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'raw payload evidence is append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_raw_payload_contents ON raw_payload_contents;
CREATE TRIGGER trg_protect_raw_payload_contents
    BEFORE UPDATE OR DELETE ON raw_payload_contents
    FOR EACH ROW EXECUTE FUNCTION protect_raw_payload_append_only();

DROP TRIGGER IF EXISTS trg_protect_raw_payload_acquisitions ON raw_payload_acquisitions;
CREATE TRIGGER trg_protect_raw_payload_acquisitions
    BEFORE UPDATE OR DELETE ON raw_payload_acquisitions
    FOR EACH ROW EXECUTE FUNCTION protect_raw_payload_append_only();
