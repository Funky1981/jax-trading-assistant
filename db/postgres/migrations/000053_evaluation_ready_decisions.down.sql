ALTER TABLE candles
    DROP COLUMN IF EXISTS provider_timezone,
    DROP COLUMN IF EXISTS adjusted_state;

DROP TRIGGER IF EXISTS trg_protect_event_asset_resolution ON event_asset_resolutions;
DROP FUNCTION IF EXISTS protect_event_asset_resolution();
DROP INDEX IF EXISTS idx_event_asset_resolution_review;
DROP INDEX IF EXISTS idx_event_asset_resolution_symbol;
DROP TABLE IF EXISTS event_asset_resolutions;

DROP TRIGGER IF EXISTS trg_protect_genuine_event_initial_decision ON genuine_event_decisions;
DROP FUNCTION IF EXISTS protect_genuine_event_initial_decision();
DROP INDEX IF EXISTS idx_genuine_event_initial_origin_decision;
DROP INDEX IF EXISTS uq_genuine_event_initial_ruleset;
ALTER TABLE genuine_event_decisions
    DROP CONSTRAINT IF EXISTS chk_genuine_event_decision_origin,
    DROP COLUMN IF EXISTS decision_context,
    DROP COLUMN IF EXISTS decision_origin,
    DROP COLUMN IF EXISTS is_initial;
