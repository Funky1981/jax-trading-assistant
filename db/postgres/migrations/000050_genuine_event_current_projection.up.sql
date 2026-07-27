DROP INDEX IF EXISTS uq_genuine_event_current_ruleset;

ALTER TABLE genuine_event_decisions
    DROP CONSTRAINT IF EXISTS uq_genuine_event_version;

ALTER TABLE genuine_event_decisions
    ADD CONSTRAINT uq_genuine_event_version UNIQUE (source_inbox_event_id, decision_version);

CREATE UNIQUE INDEX uq_genuine_event_current_ruleset
    ON genuine_event_decisions(source_inbox_event_id)
    WHERE is_current;
