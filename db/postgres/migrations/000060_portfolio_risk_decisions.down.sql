DROP TRIGGER IF EXISTS trg_protect_portfolio_risk_decisions ON portfolio_risk_decisions;
DROP FUNCTION IF EXISTS protect_portfolio_risk_decisions_append_only();
DROP INDEX IF EXISTS idx_portfolio_risk_decisions_recommendation;
DROP TABLE IF EXISTS portfolio_risk_decisions;
