package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCandidateEconomicInputsMigrationIsTypedAndAppendOnly(t *testing.T) {
	sql, err := os.ReadFile("000078_candidate_economic_inputs.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(sql))
	for _, fragment := range []string{
		"create table if not exists candidate_economic_inputs",
		"candidate_id uuid primary key references candidate_trades(id) on delete restrict",
		"instrument_id text not null", "issuer_id text not null", "identity_source text not null",
		"identity_policy_version text not null", "risk_allocation > 0 and risk_allocation <= 1",
		"requested_leverage > 0 and requested_leverage <= 1", "content_identity text not null unique",
		"create trigger trg_candidate_economic_inputs_append_only", "before update or delete",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
	if strings.Contains(text, "insert into candidate_economic_inputs") {
		t.Fatal("migration must not backfill historical candidates")
	}
}
