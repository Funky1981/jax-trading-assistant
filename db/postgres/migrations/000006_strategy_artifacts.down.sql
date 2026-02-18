-- Rollback migration 006: Strategy Artifact Tables

DROP TRIGGER IF EXISTS trigger_artifact_approvals_updated_at ON artifact_approvals;
DROP FUNCTION IF EXISTS update_artifact_approvals_updated_at();

DROP VIEW IF EXISTS artifact_history;
DROP VIEW IF EXISTS latest_artifacts;
DROP VIEW IF EXISTS approved_artifacts;

ALTER TABLE trades
    DROP COLUMN IF EXISTS artifact_hash,
    DROP COLUMN IF EXISTS artifact_id;

ALTER TABLE strategy_signals
    DROP COLUMN IF EXISTS artifact_id;

DROP TABLE IF EXISTS artifact_validation_reports;
DROP TABLE IF EXISTS artifact_promotions;
DROP TABLE IF EXISTS artifact_approvals;
DROP TABLE IF EXISTS strategy_artifacts;

DROP TYPE IF EXISTS artifact_approval_state;
