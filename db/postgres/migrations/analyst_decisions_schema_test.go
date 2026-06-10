package migrations

import (
	"path/filepath"
	"testing"
)

func TestAnalystDecisionsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000039_analyst_decisions.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000039_analyst_decisions.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS analyst_decisions",
		"macro_event_id UUID NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"fundamental_snapshot_id UUID NULL",
		"technical_snapshot_id UUID NULL",
		"evidence_bundle_id UUID NULL",
		"fundamental_score NUMERIC NOT NULL",
		"technical_score NUMERIC NOT NULL",
		"risk_score NUMERIC NOT NULL",
		"confidence_score NUMERIC NOT NULL",
		"candidate_score NUMERIC NOT NULL",
		"decision TEXT NOT NULL",
		"hard_vetoes TEXT[] NOT NULL",
		"reasons TEXT[] NOT NULL",
		"chk_analyst_decisions_decision",
		"chk_analyst_decisions_score",
		"idx_analyst_decisions_event",
		"idx_analyst_decisions_symbol",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_analyst_decisions_symbol",
		"DROP TABLE IF EXISTS analyst_decisions",
	} {
		requireContains(t, down, fragment)
	}
}
