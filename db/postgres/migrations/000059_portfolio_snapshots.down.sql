DROP TRIGGER IF EXISTS trg_protect_portfolio_snapshots ON portfolio_snapshots;
DROP FUNCTION IF EXISTS protect_portfolio_snapshots_append_only();
DROP INDEX IF EXISTS idx_portfolio_snapshots_account_as_of;
DROP TABLE IF EXISTS portfolio_snapshots;
