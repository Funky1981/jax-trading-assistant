package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestResearchMemoryMigrationDefinesExactProvenanceKeyedTable(t *testing.T) {
	data, err := os.ReadFile("000058_research_memory_entries.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{"create table if not exists research_memory_entries", "validity_digest text not null", "payload jsonb not null", "unique (validity_digest)", "semantic result caching is intentionally disabled"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
