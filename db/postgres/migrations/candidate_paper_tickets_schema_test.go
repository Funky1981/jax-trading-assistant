package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCandidatePaperTicketsMigrationCreatesPaperOnlyReviewTable(t *testing.T) {
	data, err := os.ReadFile("000045_candidate_paper_tickets.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)

	requiredFragments := []string{
		"CREATE TABLE IF NOT EXISTS candidate_paper_tickets",
		"paper_ticket_id TEXT NOT NULL UNIQUE",
		"candidate_id UUID NOT NULL REFERENCES candidate_trades(id)",
		"source_approval_id UUID REFERENCES candidate_approvals(id)",
		"status TEXT NOT NULL DEFAULT 'paper_ticket_created'",
		"paper_only BOOLEAN NOT NULL DEFAULT TRUE",
		"broker_execution_allowed BOOLEAN NOT NULL DEFAULT FALSE",
		"execution_instruction_created BOOLEAN NOT NULL DEFAULT FALSE",
		"live_trading_allowed BOOLEAN NOT NULL DEFAULT FALSE",
		"leverage_allowed BOOLEAN NOT NULL DEFAULT FALSE",
		"CONSTRAINT chk_candidate_paper_tickets_paper_only CHECK (paper_only = TRUE)",
		"CONSTRAINT chk_candidate_paper_tickets_no_broker CHECK (broker_execution_allowed = FALSE)",
		"CONSTRAINT chk_candidate_paper_tickets_no_execution CHECK (execution_instruction_created = FALSE)",
		"CONSTRAINT chk_candidate_paper_tickets_no_live CHECK (live_trading_allowed = FALSE)",
		"CONSTRAINT chk_candidate_paper_tickets_no_leverage CHECK (leverage_allowed = FALSE)",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

func TestCandidatePaperTicketReviewActionsMigrationAddsInternalNotesOnly(t *testing.T) {
	data, err := os.ReadFile("000046_candidate_paper_ticket_review_actions.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)

	requiredFragments := []string{
		"ADD COLUMN IF NOT EXISTS review_notes TEXT",
		"CREATE OR REPLACE FUNCTION append_review_note",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
