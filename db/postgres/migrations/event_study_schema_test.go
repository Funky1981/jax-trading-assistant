package migrations

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEventStudySchemaMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000022_event_study_schema.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000022_event_study_schema.down.sql"))

	for _, table := range []string{
		"event_windows",
		"event_confounders",
		"event_priced_in_scores",
		"etf_context_snapshots",
		"research_summaries",
	} {
		requireContains(t, up, "CREATE TABLE IF NOT EXISTS "+table)
		requireContains(t, down, "DROP TABLE IF EXISTS "+table)
	}

	for _, fragment := range []string{
		"UNIQUE (event_id, symbol, window_name)",
		"UNIQUE (event_id, confounding_event_id, symbol)",
		"UNIQUE (event_id, symbol)",
		"chk_event_windows_window_order",
		"chk_event_confounders_relationship_type",
		"chk_event_confounders_relevance_score",
		"chk_event_priced_in_scores_verdict",
		"chk_event_priced_in_scores_score",
		"idx_event_windows_symbol_window",
		"idx_event_windows_event_symbol",
		"idx_event_confounders_event_symbol",
		"idx_event_priced_in_scores_event_symbol",
		"idx_etf_context_snapshots_symbol_validity",
		"idx_research_summaries_event_symbol",
	} {
		requireContains(t, up, fragment)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Dir(file)
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", filepath.Base(path), err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

func requireContains(t *testing.T, content, fragment string) {
	t.Helper()
	if !strings.Contains(content, fragment) {
		t.Fatalf("migration missing fragment %q", fragment)
	}
}
