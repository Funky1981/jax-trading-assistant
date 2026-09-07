CREATE TABLE IF NOT EXISTS workflow_instances (
    workflow_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    state TEXT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    recommendation_id TEXT NOT NULL,
    risk_decision_id TEXT NOT NULL,
    portfolio_snapshot_id TEXT NOT NULL,
    analytics_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    proposal_id TEXT,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_workflow_instances_utc_order CHECK (updated_at >= created_at)
);

CREATE INDEX IF NOT EXISTS idx_workflow_instances_state_updated
    ON workflow_instances(state, updated_at DESC);
