-- Migration 006: Strategy Artifact Tables (ADR-0012 Phase 4)

-- Idempotent ENUM creation (CREATE TYPE does not support IF NOT EXISTS in PostgreSQL)
DO $$ BEGIN
    CREATE TYPE artifact_approval_state AS ENUM (
        'DRAFT',
        'VALIDATED',
        'REVIEWED',
        'APPROVED',
        'ACTIVE',
        'DEPRECATED',
        'REVOKED'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS strategy_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id VARCHAR(255) UNIQUE NOT NULL,
    schema_version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    strategy_name VARCHAR(255) NOT NULL,
    strategy_version VARCHAR(50) NOT NULL,
    code_ref VARCHAR(255),
    params JSONB NOT NULL,
    data_window_from TIMESTAMPTZ,
    data_window_to TIMESTAMPTZ,
    symbols TEXT[],
    validation JSONB,
    risk_profile JSONB NOT NULL,
    hash VARCHAR(64) NOT NULL UNIQUE,
    signature TEXT,
    payload JSONB NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_strategy_version UNIQUE (strategy_name, strategy_version)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_strategy_name ON strategy_artifacts(strategy_name);
CREATE INDEX IF NOT EXISTS idx_artifacts_created_at ON strategy_artifacts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_hash ON strategy_artifacts(hash);

CREATE TABLE IF NOT EXISTS artifact_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    state artifact_approval_state NOT NULL DEFAULT 'DRAFT',
    previous_state artifact_approval_state,
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    validation_run_id UUID,
    validation_passed BOOLEAN DEFAULT FALSE,
    validation_report_uri TEXT,
    review_notes TEXT,
    reviewer VARCHAR(255),
    reviewed_at TIMESTAMPTZ,
    state_changed_by VARCHAR(255),
    state_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state_change_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_approval_per_artifact UNIQUE (artifact_id)
);

CREATE INDEX IF NOT EXISTS idx_approvals_artifact_id ON artifact_approvals(artifact_id);
CREATE INDEX IF NOT EXISTS idx_approvals_state ON artifact_approvals(state);
CREATE INDEX IF NOT EXISTS idx_approvals_approved_at ON artifact_approvals(approved_at DESC);

CREATE TABLE IF NOT EXISTS artifact_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    from_state artifact_approval_state NOT NULL,
    to_state artifact_approval_state NOT NULL,
    promoted_by VARCHAR(255) NOT NULL,
    promoted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT,
    validation_data JSONB,
    CONSTRAINT valid_state_transition CHECK (from_state <> to_state)
);

CREATE INDEX IF NOT EXISTS idx_promotions_artifact_id ON artifact_promotions(artifact_id);
CREATE INDEX IF NOT EXISTS idx_promotions_promoted_at ON artifact_promotions(promoted_at DESC);

CREATE TABLE IF NOT EXISTS artifact_validation_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    run_id UUID UNIQUE NOT NULL,
    test_type VARCHAR(50) NOT NULL,
    passed BOOLEAN NOT NULL,
    metrics JSONB,
    errors TEXT[],
    warnings TEXT[],
    determinism_seed INTEGER,
    test_environment JSONB,
    report_uri TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    duration_seconds NUMERIC,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_validation_artifact_id ON artifact_validation_reports(artifact_id);
CREATE INDEX IF NOT EXISTS idx_validation_run_id ON artifact_validation_reports(run_id);
CREATE INDEX IF NOT EXISTS idx_validation_completed_at ON artifact_validation_reports(completed_at DESC);

ALTER TABLE strategy_signals
    ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES strategy_artifacts(id);
CREATE INDEX IF NOT EXISTS idx_signals_artifact_id ON strategy_signals(artifact_id);

ALTER TABLE trades
    ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES strategy_artifacts(id),
    ADD COLUMN IF NOT EXISTS artifact_hash VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_trades_artifact_id ON trades(artifact_id);
CREATE INDEX IF NOT EXISTS idx_trades_artifact_hash ON trades(artifact_hash);

CREATE OR REPLACE VIEW approved_artifacts AS
SELECT
    a.id, a.artifact_id, a.strategy_name, a.strategy_version,
    a.hash, a.params, a.risk_profile, a.created_at,
    ap.state, ap.approved_by, ap.approved_at
FROM strategy_artifacts a
JOIN artifact_approvals ap ON a.id = ap.artifact_id
WHERE ap.state IN ('APPROVED', 'ACTIVE') AND ap.state <> 'REVOKED'
ORDER BY a.strategy_name, a.created_at DESC;

CREATE OR REPLACE VIEW latest_artifacts AS
SELECT DISTINCT ON (strategy_name)
    id, artifact_id, strategy_name, strategy_version, hash, params, created_at, created_by
FROM strategy_artifacts
ORDER BY strategy_name, created_at DESC;

CREATE OR REPLACE VIEW artifact_history AS
SELECT
    a.artifact_id, a.strategy_name, a.strategy_version,
    p.from_state, p.to_state, p.promoted_by, p.promoted_at, p.reason
FROM artifact_promotions p
JOIN strategy_artifacts a ON p.artifact_id = a.id
ORDER BY p.promoted_at DESC;

CREATE OR REPLACE FUNCTION update_artifact_approvals_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_artifact_approvals_updated_at ON artifact_approvals;
CREATE TRIGGER trigger_artifact_approvals_updated_at
BEFORE UPDATE ON artifact_approvals
FOR EACH ROW EXECUTE FUNCTION update_artifact_approvals_updated_at();
