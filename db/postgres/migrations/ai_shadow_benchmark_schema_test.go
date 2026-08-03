package migrations_test

import (
	"os"
	"strings"
	"testing"
)

func TestAIShadowBenchmarkMigrationIsIsolated(t *testing.T) {
	raw, err := os.ReadFile("000054_ai_shadow_benchmark.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(raw))
	for _, table := range []string{"ai_shadow_benchmark_runs", "ai_shadow_benchmark_attempts", "ai_shadow_benchmark_results"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Fatalf("missing isolated table %s", table)
		}
	}
	for _, prohibited := range []string{
		"insert into genuine_event_decisions", "update genuine_event_decisions",
		"insert into candidate_trades", "insert into candidate_approvals",
		"insert into candidate_paper_tickets", "insert into execution_instructions",
		"insert into order_intents", "insert into trades", "insert into fills",
	} {
		if strings.Contains(sql, prohibited) {
			t.Fatalf("migration contains prohibited write: %s", prohibited)
		}
	}
	if !strings.Contains(sql, "append-only") {
		t.Fatal("AI shadow attempts and results must be append-only")
	}
}
