package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestGenuineEventDecisionsMigrationDefinesVersionedContract(t *testing.T) {
	data, err := os.ReadFile("000049_genuine_event_decisions.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS genuine_event_decisions",
		"source_inbox_event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id)",
		"decision TEXT NOT NULL",
		"ruleset_version TEXT NOT NULL",
		"processing_mode TEXT NOT NULL DEFAULT 'deterministic'",
		"unknown_assets BOOLEAN NOT NULL",
		"asset_mapping_provenance JSONB NOT NULL",
		"replay_identity TEXT NOT NULL",
		"CONSTRAINT chk_genuine_event_candidate_link",
		"CONSTRAINT uq_genuine_event_replay_identity UNIQUE",
		"CONSTRAINT uq_genuine_event_version UNIQUE (source_inbox_event_id, decision_version)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_genuine_event_current_ruleset",
	} {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	projection, err := os.ReadFile("000050_genuine_event_current_projection.up.sql")
	if err != nil {
		t.Fatalf("read current-projection migration: %v", err)
	}
	if !strings.Contains(string(projection), "ON genuine_event_decisions(source_inbox_event_id)\n    WHERE is_current") {
		t.Fatal("current-projection migration does not enforce one current decision per event")
	}
}
