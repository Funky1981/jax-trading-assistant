package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestStructuredTradeCandidateModelMigrationAddsReviewSafetyFields(t *testing.T) {
	data, err := os.ReadFile("000043_structured_trade_candidate_model.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)

	requiredFragments := []string{
		"ALTER TABLE candidate_trades",
		"ADD COLUMN IF NOT EXISTS setup_type",
		"ADD COLUMN IF NOT EXISTS direction",
		"ADD COLUMN IF NOT EXISTS catalyst_summary",
		"ADD COLUMN IF NOT EXISTS supporting_evidence_summary",
		"ADD COLUMN IF NOT EXISTS contradictory_evidence_summary",
		"ADD COLUMN IF NOT EXISTS invalidation_reason",
		"ADD COLUMN IF NOT EXISTS slippage_allowance",
		"ADD COLUMN IF NOT EXISTS max_slippage_adjusted_loss",
		"ADD COLUMN IF NOT EXISTS risk_status TEXT NOT NULL DEFAULT 'pending'",
		"ADD COLUMN IF NOT EXISTS human_approval_required BOOLEAN NOT NULL DEFAULT TRUE",
		"ADD COLUMN IF NOT EXISTS gate_status TEXT NOT NULL DEFAULT 'not_evaluated'",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
