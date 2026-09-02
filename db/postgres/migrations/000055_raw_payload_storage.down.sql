DROP TRIGGER IF EXISTS trg_protect_raw_payload_acquisitions ON raw_payload_acquisitions;
DROP TRIGGER IF EXISTS trg_protect_raw_payload_contents ON raw_payload_contents;
DROP FUNCTION IF EXISTS protect_raw_payload_append_only();
DROP INDEX IF EXISTS idx_raw_payload_acquisitions_source_revision;
DROP INDEX IF EXISTS idx_raw_payload_acquisitions_provider_received;
DROP INDEX IF EXISTS idx_raw_payload_acquisitions_content_digest;
DROP TABLE IF EXISTS raw_payload_acquisitions;
DROP TABLE IF EXISTS raw_payload_contents;
