-- Migration: Add Strategy Artifact Tables
-- Purpose: Implement artifact-based promotion gate for ADR-0012 Phase 4
-- Date: 2026-02-13

-- ============================================================================
-- strategy_artifacts: Immutable strategy definitions with SHA-256 hash
-- ============================================================================
CREATE TABLE IF NOT EXISTS strategy_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id VARCHAR(255) UNIQUE NOT NULL,  -- e.g., "strat_rsi_momentum_2026-02-13T12:34:56Z"
    schema_version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    
    -- Strategy identification
    strategy_name VARCHAR(255) NOT NULL,
    strategy_version VARCHAR(50) NOT NULL,
    code_ref VARCHAR(255),  -- git commit SHA or tag
    
    -- Strategy configuration (JSONB for flexible querying)
    params JSONB NOT NULL,
    
    -- Data window for validation
    data_window_from TIMESTAMPTZ,
    data_window_to TIMESTAMPTZ,
    symbols TEXT[],  -- Array of symbols tested
    
    -- Validation metrics
    validation JSONB,  -- {backtest_run_id, metrics, determinism_seed, report_uri}
    
    -- Risk profile
    risk_profile JSONB NOT NULL,  -- {max_position_pct, max_daily_loss, allowed_order_types}
    
    -- Immutability guarantee
    hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 of canonical JSON
    signature TEXT,  -- Optional KMS signature
    payload JSONB NOT NULL,  -- Complete canonical artifact JSON
    
    -- Audit trail
    created_by VARCHAR(255) NOT NULL,  -- "research-runtime" or user ID
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Indexing for common queries
    CONSTRAINT unique_strategy_version UNIQUE (strategy_name, strategy_version)
);

CREATE INDEX idx_artifacts_strategy_name ON strategy_artifacts(strategy_name);
CREATE INDEX idx_artifacts_created_at ON strategy_artifacts(created_at DESC);
CREATE INDEX idx_artifacts_hash ON strategy_artifacts(hash);

-- ============================================================================
-- artifact_approvals: Approval workflow state machine
-- ============================================================================
CREATE TYPE artifact_approval_state AS ENUM (
    'DRAFT',      -- Initial state, not validated
    'VALIDATED',  -- Golden tests passed
    'REVIEWED',   -- Human review completed
    'APPROVED',   -- Ready for production trader
    'ACTIVE',     -- Currently in use by trader
    'DEPRECATED', -- Still approved but superseded
    'REVOKED'     -- Approval withdrawn, no longer usable
);

CREATE TABLE IF NOT EXISTS artifact_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    
    -- State machine
    state artifact_approval_state NOT NULL DEFAULT 'DRAFT',
    previous_state artifact_approval_state,
    
    -- Approval metadata
    approved_by VARCHAR(255),  -- User or system that approved
    approved_at TIMESTAMPTZ,
    
    -- Validation evidence
    validation_run_id UUID,  -- Links to test execution
    validation_passed BOOLEAN DEFAULT FALSE,
    validation_report_uri TEXT,
    
    -- Review notes
    review_notes TEXT,
    reviewer VARCHAR(255),
    reviewed_at TIMESTAMPTZ,
    
    -- Transition audit
    state_changed_by VARCHAR(255),
    state_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state_change_reason TEXT,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_approval_per_artifact UNIQUE (artifact_id)
);

CREATE INDEX idx_approvals_artifact_id ON artifact_approvals(artifact_id);
CREATE INDEX idx_approvals_state ON artifact_approvals(state);
CREATE INDEX idx_approvals_approved_at ON artifact_approvals(approved_at DESC);

-- ============================================================================
-- artifact_promotions: Audit log of state transitions
-- ============================================================================
CREATE TABLE IF NOT EXISTS artifact_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    
    -- State transition
    from_state artifact_approval_state NOT NULL,
    to_state artifact_approval_state NOT NULL,
    
    -- Transition metadata
    promoted_by VARCHAR(255) NOT NULL,
    promoted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT,
    
    -- Evidence attached to promotion
    validation_data JSONB,  -- Test results, metrics, etc.
    
    CONSTRAINT valid_state_transition CHECK (from_state <> to_state)
);

CREATE INDEX idx_promotions_artifact_id ON artifact_promotions(artifact_id);
CREATE INDEX idx_promotions_promoted_at ON artifact_promotions(promoted_at DESC);

-- ============================================================================
-- artifact_validation_reports: Detailed test results
-- ============================================================================
CREATE TABLE IF NOT EXISTS artifact_validation_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE CASCADE,
    
    -- Test execution metadata
    run_id UUID UNIQUE NOT NULL,
    test_type VARCHAR(50) NOT NULL,  -- 'golden', 'replay', 'backtest', etc.
    passed BOOLEAN NOT NULL,
    
    -- Test results
    metrics JSONB,  -- {sharpe, max_drawdown, win_rate, etc.}
    errors TEXT[],  -- Array of error messages
    warnings TEXT[],
    
    -- Execution environment
    determinism_seed INTEGER,
    test_environment JSONB,  -- {runtime_version, git_commit, etc.}
    
    -- Report storage
    report_uri TEXT,  -- S3/MinIO link to detailed HTML/JSON report
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    duration_seconds NUMERIC,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_validation_artifact_id ON artifact_validation_reports(artifact_id);
CREATE INDEX idx_validation_run_id ON artifact_validation_reports(run_id);
CREATE INDEX idx_validation_completed_at ON artifact_validation_reports(completed_at DESC);

-- ============================================================================
-- Modify existing tables to link to artifacts
-- ============================================================================

-- Add artifact_id to strategy_signals
ALTER TABLE strategy_signals 
ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES strategy_artifacts(id);

CREATE INDEX IF NOT EXISTS idx_signals_artifact_id ON strategy_signals(artifact_id);

-- Add artifact_id to trades
ALTER TABLE trades 
ADD COLUMN IF NOT EXISTS artifact_id UUID REFERENCES strategy_artifacts(id),
ADD COLUMN IF NOT EXISTS artifact_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_trades_artifact_id ON trades(artifact_id);
CREATE INDEX IF NOT EXISTS idx_trades_artifact_hash ON trades(artifact_hash);

-- ============================================================================
-- Helper views for common queries
-- ============================================================================

-- View: Approved artifacts ready for production
CREATE OR REPLACE VIEW approved_artifacts AS
SELECT 
    a.id,
    a.artifact_id,
    a.strategy_name,
    a.strategy_version,
    a.hash,
    a.params,
    a.risk_profile,
    a.created_at,
    ap.state,
    ap.approved_by,
    ap.approved_at
FROM strategy_artifacts a
JOIN artifact_approvals ap ON a.id = ap.artifact_id
WHERE ap.state IN ('APPROVED', 'ACTIVE')
  AND ap.state <> 'REVOKED'
ORDER BY a.strategy_name, a.created_at DESC;

-- View: Latest artifact per strategy
CREATE OR REPLACE VIEW latest_artifacts AS
SELECT DISTINCT ON (strategy_name)
    id,
    artifact_id,
    strategy_name,
    strategy_version,
    hash,
    params,
    created_at,
    created_by
FROM strategy_artifacts
ORDER BY strategy_name, created_at DESC;

-- View: Artifact promotion history
CREATE OR REPLACE VIEW artifact_history AS
SELECT 
    a.artifact_id,
    a.strategy_name,
    a.strategy_version,
    p.from_state,
    p.to_state,
    p.promoted_by,
    p.promoted_at,
    p.reason
FROM artifact_promotions p
JOIN strategy_artifacts a ON p.artifact_id = a.id
ORDER BY p.promoted_at DESC;

-- ============================================================================
-- Triggers for automatic timestamp updates
-- ============================================================================

CREATE OR REPLACE FUNCTION update_artifact_approvals_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_artifact_approvals_updated_at
BEFORE UPDATE ON artifact_approvals
FOR EACH ROW
EXECUTE FUNCTION update_artifact_approvals_updated_at();

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE strategy_artifacts IS 'Immutable strategy definitions with SHA-256 hash for artifact-based promotion gate (ADR-0012 Phase 4)';
COMMENT ON TABLE artifact_approvals IS 'Approval workflow state machine: DRAFT → VALIDATED → REVIEWED → APPROVED → ACTIVE';
COMMENT ON TABLE artifact_promotions IS 'Audit log of all state transitions for artifacts';
COMMENT ON TABLE artifact_validation_reports IS 'Detailed test results and metrics for artifact validation';

COMMENT ON COLUMN strategy_artifacts.hash IS 'SHA-256 hash of canonical JSON payload for immutability verification';
COMMENT ON COLUMN strategy_artifacts.payload IS 'Complete canonical artifact JSON for reproducibility';
COMMENT ON COLUMN artifact_approvals.validation_run_id IS 'Links to golden test execution in artifact_validation_reports';

-- ============================================================================
-- Seed data: Migrate existing strategies to artifacts (DRAFT state)
-- ============================================================================

-- This will be run manually after migration to create initial artifacts
-- from existing strategy configurations in libs/strategies

-- Example for future reference:
-- INSERT INTO strategy_artifacts (
--     artifact_id, strategy_name, strategy_version, 
--     params, risk_profile, hash, payload, created_by
-- ) VALUES (
--     'strat_rsi_momentum_2026-02-13T00:00:00Z',
--     'rsi_momentum',
--     '1.0.0',
--     '{"rsi_period": 14, "buy_threshold": 30, "sell_threshold": 70}'::jsonb,
--     '{"max_position_pct": 0.20, "max_daily_loss": 1000, "allowed_order_types": ["LMT"]}'::jsonb,
--     'sha256hash...',
--     '{...}'::jsonb,
--     'migration-script'
-- );
