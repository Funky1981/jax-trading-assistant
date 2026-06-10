package migrations

import (
	"path/filepath"
	"testing"
)

func TestFundamentalAnalysisSnapshotsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000038_fundamental_analysis_snapshots.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000038_fundamental_analysis_snapshots.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS fundamental_analysis_snapshots",
		"macro_event_id UUID NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"analysis_time_utc TIMESTAMPTZ NOT NULL",
		"event_summary TEXT NOT NULL",
		"expected_market_impact TEXT NOT NULL",
		"affected_themes TEXT[] NOT NULL",
		"cross_market_checks JSONB NOT NULL",
		"confounders JSONB NOT NULL",
		"fundamental_score NUMERIC NOT NULL",
		"verdict TEXT NOT NULL",
		"reasons TEXT[] NOT NULL",
		"missing_evidence TEXT[] NOT NULL",
		"UNIQUE (macro_event_id, symbol)",
		"chk_fundamental_analysis_snapshots_verdict",
		"chk_fundamental_analysis_snapshots_score",
		"idx_fundamental_analysis_snapshots_event",
		"idx_fundamental_analysis_snapshots_symbol_timeframe",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_fundamental_analysis_snapshots_symbol_timeframe",
		"DROP TABLE IF EXISTS fundamental_analysis_snapshots",
	} {
		requireContains(t, down, fragment)
	}
}
