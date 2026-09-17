package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestExploratoryPaperMigrationsPreserveIdentityAndFirewall(t *testing.T) {
	lifecycle, err := os.ReadFile("000069_exploratory_paper_lifecycle.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(lifecycle))
	for _, fragment := range []string{
		"create table if not exists exploratory_paper_lifecycles",
		"formal_evidence_eligible boolean not null default false check (formal_evidence_eligible = false)",
		"create table if not exists exploratory_paper_evidence_reassessments",
		"primary key (position_id, evidence_id)",
		"create table if not exists exploratory_paper_reviews",
		"unique (position_id, session_number)",
		"create table if not exists exploratory_paper_checkpoints",
		"create table if not exists exploratory_paper_outcomes",
		"frozen exploratory thesis identity cannot be mutated",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("exploratory paper migration missing %q", fragment)
		}
	}
	executedPrice, err := os.ReadFile("000070_paper_fill_executed_price.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(string(executedPrice)), "add column if not exists executed_price") {
		t.Fatal("paper fill executed-price migration missing")
	}
}
