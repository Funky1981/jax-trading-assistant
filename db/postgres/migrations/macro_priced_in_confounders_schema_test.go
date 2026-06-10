package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroPricedInConfoundersMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000034_macro_priced_in_confounders.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000034_macro_priced_in_confounders.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_priced_in_scores",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"verdict TEXT NOT NULL",
		"score NUMERIC NOT NULL",
		"reasons TEXT[] NOT NULL",
		"UNIQUE(macro_event_id, symbol)",
		"chk_macro_priced_in_verdict",
		"CREATE TABLE IF NOT EXISTS macro_confounders",
		"confounder_type TEXT NOT NULL",
		"severity TEXT NOT NULL",
		"chk_macro_confounder_severity",
		"idx_macro_priced_in_scores_event",
		"idx_macro_confounders_event",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_confounders_event",
		"DROP TABLE IF EXISTS macro_confounders",
		"DROP TABLE IF EXISTS macro_priced_in_scores",
	} {
		requireContains(t, down, fragment)
	}
}
