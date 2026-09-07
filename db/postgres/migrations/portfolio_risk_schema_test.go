package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPortfolioSnapshotMigrationDefinesImmutableObservedFacts(t *testing.T) {
	data, err := os.ReadFile("000055_portfolio_snapshots.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{"create table if not exists portfolio_snapshots", "snapshot_id text primary key", "payload jsonb not null", "captured_at >= as_of", "portfolio snapshots are immutable"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("portfolio snapshot migration missing %q", fragment)
		}
	}
}

func TestPortfolioRiskDecisionMigrationDefinesImmutableAuditArtifacts(t *testing.T) {
	data, err := os.ReadFile("000056_portfolio_risk_decisions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{"create table if not exists portfolio_risk_decisions", "decision_id text primary key", "reason_codes text[] not null", "payload jsonb not null", "portfolio risk decisions are immutable"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("risk decision migration missing %q", fragment)
		}
	}
}
