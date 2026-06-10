package migrations

import (
	"path/filepath"
	"testing"
)

func TestTechnicalAnalysisSnapshotsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000037_technical_analysis_snapshots.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000037_technical_analysis_snapshots.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS technical_analysis_snapshots",
		"macro_event_id UUID NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"analysis_time_utc TIMESTAMPTZ NOT NULL",
		"timeframe TEXT NOT NULL",
		"trend_state TEXT NOT NULL",
		"structure_state TEXT NOT NULL",
		"key_levels JSONB NOT NULL",
		"event_reaction JSONB NOT NULL",
		"volume_volatility JSONB NOT NULL",
		"relative_strength JSONB NOT NULL",
		"technical_score NUMERIC NOT NULL",
		"verdict TEXT NOT NULL",
		"reasons TEXT[] NOT NULL",
		"invalidation_rules TEXT[] NOT NULL",
		"UNIQUE (macro_event_id, symbol, timeframe)",
		"chk_technical_analysis_snapshots_verdict",
		"chk_technical_analysis_snapshots_score",
		"idx_technical_analysis_snapshots_event",
		"idx_technical_analysis_snapshots_symbol_timeframe",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_technical_analysis_snapshots_symbol_timeframe",
		"DROP TABLE IF EXISTS technical_analysis_snapshots",
	} {
		requireContains(t, down, fragment)
	}
}
