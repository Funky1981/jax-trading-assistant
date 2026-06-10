package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroCandidateTradesMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000036_macro_candidate_trades.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000036_macro_candidate_trades.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_candidate_trades",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"evidence_bundle_id UUID NOT NULL REFERENCES macro_evidence_bundles(id)",
		"side TEXT NOT NULL",
		"entry_type TEXT NOT NULL",
		"risk_percent NUMERIC NOT NULL",
		"status TEXT NOT NULL DEFAULT 'awaiting_human_approval'",
		"walkaway_reasons TEXT[] NOT NULL DEFAULT '{}'",
		"UNIQUE(macro_event_id, evidence_bundle_id, symbol)",
		"chk_macro_candidate_side",
		"chk_macro_candidate_status",
		"idx_macro_candidate_trades_status",
		"idx_macro_candidate_trades_event",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_candidate_trades_event",
		"DROP TABLE IF EXISTS macro_candidate_trades",
	} {
		requireContains(t, down, fragment)
	}
}
