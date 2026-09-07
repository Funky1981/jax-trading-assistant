CREATE TABLE IF NOT EXISTS portfolio_risk_decisions (
    decision_id TEXT PRIMARY KEY,
    recommendation_id TEXT NOT NULL,
    portfolio_snapshot_id TEXT,
    analytics_id TEXT,
    policy_id TEXT,
    algorithm TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('ACCEPT','AMEND','REJECT')),
    reason_codes TEXT[] NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_portfolio_risk_decisions_recommendation
    ON portfolio_risk_decisions(recommendation_id, created_at DESC);

CREATE OR REPLACE FUNCTION protect_portfolio_risk_decisions_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'portfolio risk decisions are immutable audit artifacts';
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_portfolio_risk_decisions ON portfolio_risk_decisions;
CREATE TRIGGER trg_protect_portfolio_risk_decisions
    BEFORE UPDATE OR DELETE ON portfolio_risk_decisions
    FOR EACH ROW EXECUTE FUNCTION protect_portfolio_risk_decisions_append_only();
