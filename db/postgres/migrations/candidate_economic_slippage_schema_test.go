package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCandidateEconomicSlippageMigrationIsAdditiveAndNullableForLegacyInputs(t *testing.T) {
	up, err := os.ReadFile("000079_candidate_economic_slippage_allowance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(up))
	if !strings.Contains(text, "alter table candidate_economic_inputs") || !strings.Contains(text, "add column if not exists slippage_allowance numeric") || !strings.Contains(text, "check (slippage_allowance >= 0)") {
		t.Fatalf("slippage migration does not preserve nullable legacy values and constrain present values: %s", text)
	}
	down, err := os.ReadFile("000079_candidate_economic_slippage_allowance.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(string(down)), "drop column if exists slippage_allowance") {
		t.Fatalf("slippage migration rollback does not remove only its additive column: %s", down)
	}
}
