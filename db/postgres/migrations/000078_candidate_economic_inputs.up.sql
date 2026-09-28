CREATE TABLE IF NOT EXISTS candidate_economic_inputs (
    candidate_id UUID PRIMARY KEY REFERENCES candidate_trades(id) ON DELETE RESTRICT,
    contract_version TEXT NOT NULL,
    instrument_id TEXT NOT NULL CHECK (length(btrim(instrument_id)) > 0),
    issuer_id TEXT NOT NULL CHECK (length(btrim(issuer_id)) > 0),
    identity_source TEXT NOT NULL CHECK (length(btrim(identity_source)) > 0),
    identity_policy_version TEXT NOT NULL CHECK (length(btrim(identity_policy_version)) > 0),
    risk_allocation NUMERIC NOT NULL CHECK (risk_allocation > 0 AND risk_allocation <= 1),
    requested_leverage NUMERIC NOT NULL CHECK (requested_leverage > 0 AND requested_leverage <= 1),
    sizing_policy_id TEXT NOT NULL CHECK (length(btrim(sizing_policy_id)) > 0),
    sizing_policy_version TEXT NOT NULL CHECK (length(btrim(sizing_policy_version)) > 0),
    content_identity TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL
);

CREATE OR REPLACE FUNCTION protect_candidate_economic_inputs_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'candidate economic inputs are immutable';
END;
$$;

CREATE TRIGGER trg_candidate_economic_inputs_append_only
    BEFORE UPDATE OR DELETE ON candidate_economic_inputs
    FOR EACH ROW EXECUTE FUNCTION protect_candidate_economic_inputs_append_only();
