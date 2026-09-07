package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestResearchTaskCheckpointsMigrationDefinesImmutableBoundedCheckpointTable(t *testing.T) {
	data, err := os.ReadFile("000057_research_task_checkpoints.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{"create table if not exists research_task_checkpoints", "checkpoint_id text primary key", "payload jsonb not null", "unique (task_id, checkpoint_version)"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
