package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroReactionSnapshotsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000032_macro_reaction_snapshots.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000032_macro_reaction_snapshots.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_reaction_snapshots",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"timeframe TEXT NOT NULL",
		"pre_price NUMERIC NOT NULL",
		"post_price NUMERIC NOT NULL",
		"change_abs NUMERIC NOT NULL",
		"change_percent NUMERIC NOT NULL",
		"direction TEXT NOT NULL",
		"confirms_event BOOLEAN NOT NULL",
		"too_extended BOOLEAN NOT NULL",
		"noisy BOOLEAN NOT NULL",
		"raw_candles JSONB NOT NULL DEFAULT '[]'::jsonb",
		"UNIQUE(macro_event_id, symbol, timeframe)",
		"chk_macro_reaction_direction",
		"idx_macro_reaction_snapshots_event",
		"idx_macro_reaction_snapshots_symbol_timeframe",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_reaction_snapshots_symbol_timeframe",
		"DROP TABLE IF EXISTS macro_reaction_snapshots",
	} {
		requireContains(t, down, fragment)
	}
}
