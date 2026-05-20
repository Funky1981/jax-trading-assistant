package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPricedInEngineMigrationAddsDecisionFields(t *testing.T) {
	path := filepath.Join("000023_priced_in_engine.up.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sql := string(data)
	for _, required := range []string{
		"ADD COLUMN IF NOT EXISTS post_event_5m_return",
		"ADD COLUMN IF NOT EXISTS hard_reject",
		"ADD COLUMN IF NOT EXISTS hard_reject_reasons",
		"idx_event_priced_in_scores_hard_reject",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("priced-in migration missing %q", required)
		}
	}
}
