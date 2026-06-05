package migrations

import (
	"path/filepath"
	"testing"
)

func TestWorldMonitorResearchInboxMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000030_world_monitor_research_inbox.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000030_world_monitor_research_inbox.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS world_monitor_research_inbox",
		"world_monitor_event_id TEXT NOT NULL",
		"status TEXT NOT NULL DEFAULT 'new'",
		"severity TEXT NOT NULL",
		"source_tier TEXT NOT NULL",
		"confidence_reasons JSONB NOT NULL DEFAULT '[]'::jsonb",
		"raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"normalized_event_id UUID REFERENCES event_normalized(id)",
		"candidate_id UUID REFERENCES candidate_trades(id)",
		"CONSTRAINT uq_world_monitor_source_event",
		"chk_world_monitor_inbox_status",
		"chk_world_monitor_inbox_severity",
		"chk_world_monitor_inbox_source_tier",
		"idx_world_monitor_inbox_status",
		"idx_world_monitor_inbox_event_time",
		"idx_world_monitor_inbox_normalized_event",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_world_monitor_inbox_normalized_event",
		"DROP TABLE IF EXISTS world_monitor_research_inbox",
	} {
		requireContains(t, down, fragment)
	}
}
