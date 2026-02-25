-- Phase 1: No-fake-data provenance fields for runs and artifacts.

ALTER TABLE backtest_runs
    ADD COLUMN IF NOT EXISTS data_source_type TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS source_provider TEXT,
    ADD COLUMN IF NOT EXISTS dataset_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS synthetic_reason TEXT,
    ADD COLUMN IF NOT EXISTS provenance_verified_at TIMESTAMPTZ;

ALTER TABLE runs
    ADD COLUMN IF NOT EXISTS data_source_type TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS source_provider TEXT,
    ADD COLUMN IF NOT EXISTS dataset_id TEXT,
    ADD COLUMN IF NOT EXISTS dataset_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS synthetic_reason TEXT,
    ADD COLUMN IF NOT EXISTS provenance_verified_at TIMESTAMPTZ;

ALTER TABLE run_artifacts
    ADD COLUMN IF NOT EXISTS data_source_type TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS source_provider TEXT,
    ADD COLUMN IF NOT EXISTS dataset_id TEXT,
    ADD COLUMN IF NOT EXISTS dataset_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS synthetic_reason TEXT,
    ADD COLUMN IF NOT EXISTS provenance_verified_at TIMESTAMPTZ;

ALTER TABLE strategy_artifacts
    ADD COLUMN IF NOT EXISTS data_source_type TEXT NOT NULL DEFAULT 'real',
    ADD COLUMN IF NOT EXISTS source_provider TEXT,
    ADD COLUMN IF NOT EXISTS dataset_id TEXT,
    ADD COLUMN IF NOT EXISTS dataset_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS synthetic_reason TEXT,
    ADD COLUMN IF NOT EXISTS provenance_verified_at TIMESTAMPTZ DEFAULT NOW();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_backtest_runs_data_source_type'
    ) THEN
        ALTER TABLE backtest_runs
            ADD CONSTRAINT chk_backtest_runs_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_runs_data_source_type'
    ) THEN
        ALTER TABLE runs
            ADD CONSTRAINT chk_runs_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_run_artifacts_data_source_type'
    ) THEN
        ALTER TABLE run_artifacts
            ADD CONSTRAINT chk_run_artifacts_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_strategy_artifacts_data_source_type'
    ) THEN
        ALTER TABLE strategy_artifacts
            ADD CONSTRAINT chk_strategy_artifacts_data_source_type
            CHECK (data_source_type IN ('real', 'synthetic', 'unknown'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_is_synthetic ON backtest_runs(is_synthetic);
CREATE INDEX IF NOT EXISTS idx_runs_is_synthetic ON runs(is_synthetic);
CREATE INDEX IF NOT EXISTS idx_strategy_artifacts_is_synthetic ON strategy_artifacts(is_synthetic);
