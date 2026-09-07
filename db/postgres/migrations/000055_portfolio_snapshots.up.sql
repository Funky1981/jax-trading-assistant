CREATE TABLE IF NOT EXISTS portfolio_snapshots (
    snapshot_id TEXT PRIMARY KEY,
    contract_version TEXT NOT NULL,
    identity_algorithm TEXT NOT NULL,
    account_id TEXT NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    provider TEXT NOT NULL,
    currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    synthetic BOOLEAN NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_portfolio_snapshot_capture_order CHECK (captured_at >= as_of)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_account_as_of
    ON portfolio_snapshots(account_id, as_of DESC);

CREATE OR REPLACE FUNCTION protect_portfolio_snapshots_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'portfolio snapshots are immutable observed facts';
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_portfolio_snapshots ON portfolio_snapshots;
CREATE TRIGGER trg_protect_portfolio_snapshots
    BEFORE UPDATE OR DELETE ON portfolio_snapshots
    FOR EACH ROW EXECUTE FUNCTION protect_portfolio_snapshots_append_only();
