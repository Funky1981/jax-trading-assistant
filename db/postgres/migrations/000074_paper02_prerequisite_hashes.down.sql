DROP TRIGGER IF EXISTS trg_protect_paper02_pilot_identity ON exploratory_paper_pilots;

ALTER TABLE exploratory_paper_pilots
    DROP COLUMN IF EXISTS eligible_universe_hash,
    DROP COLUMN IF EXISTS risk_policy_hash,
    DROP COLUMN IF EXISTS entry_policy_hash;

CREATE OR REPLACE FUNCTION protect_paper02_pilot_identity()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.pilot_id IS DISTINCT FROM OLD.pilot_id
       OR NEW.mode IS DISTINCT FROM OLD.mode
       OR NEW.protocol_version IS DISTINCT FROM OLD.protocol_version
       OR NEW.protocol_hash IS DISTINCT FROM OLD.protocol_hash
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.formal_evidence_eligible IS DISTINCT FROM OLD.formal_evidence_eligible THEN
        RAISE EXCEPTION 'PAPER-02 pilot identity is immutable';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_protect_paper02_pilot_identity
    BEFORE UPDATE ON exploratory_paper_pilots
    FOR EACH ROW EXECUTE FUNCTION protect_paper02_pilot_identity();
