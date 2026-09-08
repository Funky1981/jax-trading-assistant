package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase11PaperSchemasDeclareIsolatedAppendOnlyDomain(t *testing.T) {
	checks := map[string][]string{
		"000064_paper_orders.up.sql":        {"environment TEXT NOT NULL CHECK (environment = 'PAPER')", "paper_intent_id TEXT NOT NULL UNIQUE", "filled_quantity + remaining_quantity = quantity"},
		"000065_paper_fills.up.sql":         {"REFERENCES paper_orders", "CREATE TRIGGER trg_paper_fills_append_only", "UNIQUE (order_id, tick_id)"},
		"000066_paper_ledger_events.up.sql": {"CREATE TABLE IF NOT EXISTS paper_accounts", "environment TEXT NOT NULL CHECK (environment = 'PAPER')", "CREATE TRIGGER trg_paper_ledger_events_append_only"},
		"000067_paper_soak_runs.up.sql":     {"CREATE TABLE IF NOT EXISTS paper_soak_runs", "CREATE TABLE IF NOT EXISTS paper_soak_events", "CREATE TRIGGER trg_paper_soak_events_append_only"},
	}
	for name, needles := range checks {
		data, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q", name, needle)
			}
		}
	}
}
