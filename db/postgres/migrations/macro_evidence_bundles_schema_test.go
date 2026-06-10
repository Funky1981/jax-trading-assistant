package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroEvidenceBundlesMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000035_macro_evidence_bundles.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000035_macro_evidence_bundles.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_evidence_bundles",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"status TEXT NOT NULL",
		"verdict TEXT NOT NULL",
		"summary TEXT NOT NULL",
		"evidence JSONB NOT NULL",
		"missing_evidence TEXT[] NOT NULL DEFAULT '{}'",
		"walkaway_reasons TEXT[] NOT NULL DEFAULT '{}'",
		"UNIQUE(macro_event_id, symbol)",
		"chk_macro_evidence_bundle_verdict",
		"idx_macro_evidence_bundles_event",
		"idx_macro_evidence_bundles_verdict",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_evidence_bundles_verdict",
		"DROP TABLE IF EXISTS macro_evidence_bundles",
	} {
		requireContains(t, down, fragment)
	}
}
