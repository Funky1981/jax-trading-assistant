package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestEvaluationReadyDecisionsMigrationIsImmutableAndVersioned(t *testing.T) {
	data, err := os.ReadFile("000053_evaluation_ready_decisions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(data)
	for _, required := range []string{
		"is_initial BOOLEAN NOT NULL",
		"decision_origin TEXT NOT NULL",
		"uq_genuine_event_initial_ruleset",
		"protect_genuine_event_initial_decision",
		"CREATE TABLE IF NOT EXISTS event_asset_resolutions",
		"resolver_ruleset_version TEXT NOT NULL",
		"known_at_initial_decision_time BOOLEAN NOT NULL",
		"knowable_at_operational_anchor BOOLEAN NOT NULL",
		"uq_event_asset_resolution_ruleset UNIQUE",
		"protect_event_asset_resolution",
		"adjusted_state TEXT NOT NULL",
		"provider_timezone TEXT NOT NULL",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
