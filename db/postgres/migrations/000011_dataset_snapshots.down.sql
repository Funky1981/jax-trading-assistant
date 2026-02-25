DROP TRIGGER IF EXISTS trg_dataset_snapshots_updated_at ON dataset_snapshots;
DROP FUNCTION IF EXISTS update_dataset_snapshots_updated_at();

DROP INDEX IF EXISTS idx_dataset_snapshot_links_linked_at;
DROP INDEX IF EXISTS idx_dataset_snapshot_links_runref;
DROP INDEX IF EXISTS idx_dataset_snapshots_hash;
DROP INDEX IF EXISTS idx_dataset_snapshots_last_seen;

ALTER TABLE IF EXISTS dataset_snapshot_links
    DROP CONSTRAINT IF EXISTS chk_dataset_snapshot_links_run_type;

DROP TABLE IF EXISTS dataset_snapshot_links;
DROP TABLE IF EXISTS dataset_snapshots;
