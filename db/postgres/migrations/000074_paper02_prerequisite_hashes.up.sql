ALTER TABLE exploratory_paper_pilots
    ADD COLUMN IF NOT EXISTS eligible_universe_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS risk_policy_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS entry_policy_hash TEXT NOT NULL DEFAULT '';

CREATE OR REPLACE FUNCTION protect_paper02_pilot_identity()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.pilot_id IS DISTINCT FROM OLD.pilot_id
       OR NEW.mode IS DISTINCT FROM OLD.mode
       OR NEW.protocol_version IS DISTINCT FROM OLD.protocol_version
       OR NEW.protocol_hash IS DISTINCT FROM OLD.protocol_hash
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.eligible_universe_hash IS DISTINCT FROM OLD.eligible_universe_hash
       OR NEW.risk_policy_hash IS DISTINCT FROM OLD.risk_policy_hash
       OR NEW.entry_policy_hash IS DISTINCT FROM OLD.entry_policy_hash
       OR NEW.formal_evidence_eligible IS DISTINCT FROM OLD.formal_evidence_eligible THEN
        RAISE EXCEPTION 'PAPER-02 pilot identity is immutable';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_paper02_pilot_identity ON exploratory_paper_pilots;
CREATE TRIGGER trg_protect_paper02_pilot_identity
    BEFORE UPDATE ON exploratory_paper_pilots
    FOR EACH ROW EXECUTE FUNCTION protect_paper02_pilot_identity();
