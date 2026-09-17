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
	runtimeLoop, err := os.ReadFile("000071_exploratory_paper_runtime_loop.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(string(runtimeLoop)), "exit_recommended") {
		t.Fatal("runtime review migration missing durable exit recommendation state")
	}
	queue, err := os.ReadFile("000072_exploratory_paper_entry_queue.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"exploratory_paper_entry_queue", "unique", "paper-only handoff"} {
		if !strings.Contains(strings.ToLower(string(queue)), fragment) {
			t.Fatalf("entry queue migration missing %q", fragment)
		}
	}
	pilot, err := os.ReadFile("000073_paper02_pilot_readiness.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"exploratory_paper_pilots", "READY_FOR_EXTERNAL_REVIEW", "exploratory_paper_opportunities", "exploratory_paper_pilot_evidence", "formal_evidence_eligible = FALSE"} {
		if !strings.Contains(strings.ToLower(string(pilot)), strings.ToLower(fragment)) {
			t.Fatalf("PAPER-02 migration missing %q", fragment)
		}
	}
	hashes, err := os.ReadFile("000074_paper02_prerequisite_hashes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"eligible_universe_hash", "risk_policy_hash", "entry_policy_hash", "PAPER-02 pilot identity is immutable"} {
		if !strings.Contains(strings.ToLower(string(hashes)), strings.ToLower(fragment)) {
			t.Fatalf("PAPER-02 prerequisite migration missing %q", fragment)
		}
	}
}
