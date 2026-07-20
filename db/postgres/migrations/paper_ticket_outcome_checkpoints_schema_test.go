package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPaperTicketOutcomeCheckpointMigrationIsHypotheticalAndIdempotent(t *testing.T) {
	data, err := os.ReadFile("000047_paper_ticket_outcome_checkpoints.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS paper_ticket_outcome_checkpoints",
		"UNIQUE (paper_ticket_id, checkpoint_name)",
		"hypothetical_entry_price", "calculation_inputs JSONB",
		"ambiguous_same_candle", "pending_market_data", "pending_not_due",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	for _, forbidden := range []string{"execution_instructions", "broker_orders", "order_intents", "INSERT INTO trades"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("migration crosses execution boundary with %q", forbidden)
		}
	}
}
