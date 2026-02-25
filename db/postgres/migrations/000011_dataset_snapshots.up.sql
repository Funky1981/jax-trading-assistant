-- Phase 3: Dataset snapshots and run linkage for provenance integrity.

CREATE TABLE IF NOT EXISTS dataset_snapshots (
    dataset_id TEXT PRIMARY KEY,
    dataset_hash TEXT NOT NULL,
    dataset_name TEXT,
    symbol TEXT,
    source TEXT,
    schema_ver TEXT,
    record_count INTEGER,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    file_path TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dataset_snapshot_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id TEXT NOT NULL REFERENCES dataset_snapshots(dataset_id) ON DELETE CASCADE,
    run_type TEXT NOT NULL,
    run_ref_id TEXT NOT NULL,
    observed_hash TEXT NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (dataset_id, run_type, run_ref_id)
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_dataset_snapshot_links_run_type'
    ) THEN
        ALTER TABLE dataset_snapshot_links
            ADD CONSTRAINT chk_dataset_snapshot_links_run_type
            CHECK (run_type IN ('run', 'backtest_run'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_dataset_snapshots_last_seen
    ON dataset_snapshots(last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_dataset_snapshots_hash
    ON dataset_snapshots(dataset_hash);
CREATE INDEX IF NOT EXISTS idx_dataset_snapshot_links_runref
    ON dataset_snapshot_links(run_type, run_ref_id);
CREATE INDEX IF NOT EXISTS idx_dataset_snapshot_links_linked_at
    ON dataset_snapshot_links(linked_at DESC);

CREATE OR REPLACE FUNCTION update_dataset_snapshots_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_dataset_snapshots_updated_at ON dataset_snapshots;
CREATE TRIGGER trg_dataset_snapshots_updated_at
BEFORE UPDATE ON dataset_snapshots
FOR EACH ROW EXECUTE FUNCTION update_dataset_snapshots_updated_at();
