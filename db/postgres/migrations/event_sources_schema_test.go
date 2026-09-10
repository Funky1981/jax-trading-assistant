package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestRetiredEventSourceMigrationPreservesHistoricalReferences(t *testing.T) {
	up, err := os.ReadFile("000068_retire_redundant_event_source.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("000068_retire_redundant_event_source.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	upText, downText := string(up), string(down)
	for _, fragment := range []string{
		"WHERE id = 'finnhub'",
		"enabled = FALSE",
		"priority = 9999",
		"'retired_reason'",
	} {
		if !strings.Contains(upText, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}
	if strings.Contains(upText, "DELETE FROM event_sources") || strings.Contains(upText, "DROP TABLE") {
		t.Fatal("retirement migration must preserve historical event-source rows")
	}
	for _, fragment := range []string{
		"WHERE id = 'finnhub'",
		"enabled = TRUE",
		"priority = 20",
	} {
		if !strings.Contains(downText, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}
