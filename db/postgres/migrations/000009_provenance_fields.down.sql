DROP INDEX IF EXISTS idx_strategy_artifacts_is_synthetic;
DROP INDEX IF EXISTS idx_runs_is_synthetic;
DROP INDEX IF EXISTS idx_backtest_runs_is_synthetic;

ALTER TABLE strategy_artifacts DROP CONSTRAINT IF EXISTS chk_strategy_artifacts_data_source_type;
ALTER TABLE run_artifacts DROP CONSTRAINT IF EXISTS chk_run_artifacts_data_source_type;
ALTER TABLE runs DROP CONSTRAINT IF EXISTS chk_runs_data_source_type;
ALTER TABLE backtest_runs DROP CONSTRAINT IF EXISTS chk_backtest_runs_data_source_type;

ALTER TABLE strategy_artifacts
    DROP COLUMN IF EXISTS provenance_verified_at,
    DROP COLUMN IF EXISTS synthetic_reason,
    DROP COLUMN IF EXISTS is_synthetic,
    DROP COLUMN IF EXISTS dataset_hash,
    DROP COLUMN IF EXISTS dataset_id,
    DROP COLUMN IF EXISTS source_provider,
    DROP COLUMN IF EXISTS data_source_type;

ALTER TABLE run_artifacts
    DROP COLUMN IF EXISTS provenance_verified_at,
    DROP COLUMN IF EXISTS synthetic_reason,
    DROP COLUMN IF EXISTS is_synthetic,
    DROP COLUMN IF EXISTS dataset_hash,
    DROP COLUMN IF EXISTS dataset_id,
    DROP COLUMN IF EXISTS source_provider,
    DROP COLUMN IF EXISTS data_source_type;

ALTER TABLE runs
    DROP COLUMN IF EXISTS provenance_verified_at,
    DROP COLUMN IF EXISTS synthetic_reason,
    DROP COLUMN IF EXISTS is_synthetic,
    DROP COLUMN IF EXISTS dataset_hash,
    DROP COLUMN IF EXISTS dataset_id,
    DROP COLUMN IF EXISTS source_provider,
    DROP COLUMN IF EXISTS data_source_type;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS provenance_verified_at,
    DROP COLUMN IF EXISTS synthetic_reason,
    DROP COLUMN IF EXISTS is_synthetic,
    DROP COLUMN IF EXISTS dataset_hash,
    DROP COLUMN IF EXISTS source_provider,
    DROP COLUMN IF EXISTS data_source_type;
