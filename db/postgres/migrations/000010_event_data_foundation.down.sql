DROP TRIGGER IF EXISTS trg_event_sources_updated_at ON event_sources;
DROP FUNCTION IF EXISTS update_event_sources_updated_at();

DROP INDEX IF EXISTS idx_event_symbol_map_primary;
DROP INDEX IF EXISTS idx_event_symbol_map_symbol_time;
DROP INDEX IF EXISTS idx_event_normalized_raw_event;
DROP INDEX IF EXISTS idx_event_normalized_symbol_time;
DROP INDEX IF EXISTS idx_event_normalized_source_time;
DROP INDEX IF EXISTS idx_event_normalized_kind_time;
DROP INDEX IF EXISTS idx_event_raw_flow_id;
DROP INDEX IF EXISTS idx_event_raw_symbol_time;
DROP INDEX IF EXISTS idx_event_raw_kind_time;
DROP INDEX IF EXISTS idx_event_sources_enabled_priority;

ALTER TABLE IF EXISTS event_normalized
    DROP CONSTRAINT IF EXISTS chk_event_normalized_data_source_type;
ALTER TABLE IF EXISTS event_raw
    DROP CONSTRAINT IF EXISTS chk_event_raw_data_source_type;

DROP TABLE IF EXISTS event_symbol_map;
DROP TABLE IF EXISTS event_normalized;
DROP TABLE IF EXISTS event_raw;
DROP TABLE IF EXISTS event_sources;
