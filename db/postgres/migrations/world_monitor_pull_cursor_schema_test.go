package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestWorldMonitorPullCursorMigrationDefinesDurablePosition(t *testing.T) {
	data, err := os.ReadFile("000051_world_monitor_pull_cursor.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS world_monitor_pull_cursors",
		"consumer_name TEXT NOT NULL",
		"source_endpoint_identity TEXT NOT NULL",
		"last_committed_position BIGINT NOT NULL DEFAULT 0",
		"diagnostic_metadata JSONB NOT NULL",
		"PRIMARY KEY (consumer_name, source_endpoint_identity)",
	} {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
