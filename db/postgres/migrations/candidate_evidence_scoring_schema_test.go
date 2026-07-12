package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCandidateEvidenceScoringMigrationAddsCandidateScopedEvidenceTables(t *testing.T) {
	data, err := os.ReadFile("000044_candidate_evidence_scoring.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)

	requiredFragments := []string{
		"CREATE TABLE IF NOT EXISTS candidate_evidence_items",
		"candidate_id UUID NOT NULL REFERENCES candidate_trades(id)",
		"source_type TEXT NOT NULL",
		"supports_candidate BOOLEAN NOT NULL DEFAULT FALSE",
		"contradicts_candidate BOOLEAN NOT NULL DEFAULT FALSE",
		"freshness_status TEXT NOT NULL",
		"CREATE TABLE IF NOT EXISTS candidate_evidence_scores",
		"overall_evidence_score NUMERIC(8,6) NOT NULL",
		"evidence_status TEXT NOT NULL",
		"broker_execution_allowed BOOLEAN NOT NULL DEFAULT FALSE",
		"execution_instruction_created BOOLEAN NOT NULL DEFAULT FALSE",
		"approval_granted BOOLEAN NOT NULL DEFAULT FALSE",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
