package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResearchSummariesEvidenceBundleSchema(t *testing.T) {
	path := filepath.Join("000022_event_study_schema.up.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sql := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS research_summaries",
		"priced_in_view TEXT NOT NULL",
		"evidence JSONB NOT NULL DEFAULT '{}'::jsonb",
		"idx_research_summaries_event_symbol",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("research summary schema missing %q", required)
		}
	}
}
