package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroEventsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000031_macro_events.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000031_macro_events.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_events",
		"source_event_id TEXT NOT NULL",
		"event_type TEXT NOT NULL",
		"event_time_utc TIMESTAMPTZ NOT NULL",
		"surprise_value NUMERIC",
		"surprise_percent NUMERIC",
		"direction TEXT NOT NULL",
		"raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"UNIQUE(source, source_event_id)",
		"chk_macro_events_direction",
		"chk_macro_events_confidence",
		"CREATE TABLE IF NOT EXISTS macro_event_etf_map",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"UNIQUE(macro_event_id, symbol)",
		"chk_macro_event_etf_map_confidence",
		"idx_macro_events_type_time",
		"idx_macro_events_source_event",
		"idx_macro_event_etf_map_symbol",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_event_etf_map_symbol",
		"DROP TABLE IF EXISTS macro_event_etf_map",
		"DROP TABLE IF EXISTS macro_events",
	} {
		requireContains(t, down, fragment)
	}
}
