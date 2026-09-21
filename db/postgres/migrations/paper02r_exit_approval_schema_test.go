package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPaper02RExitApprovalMigrationPreservesHistoricalStatuses(t *testing.T) {
	data, err := os.ReadFile("000076_paper02r_exit_approval_lifecycle.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{
		"drop constraint if exists exploratory_paper_reviews_status_check",
		"exit_recommended",
		"exit_approved",
		"exit_rejected",
		"workflow-bound state",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("PAPER-02R migration missing %q", fragment)
		}
	}
}
