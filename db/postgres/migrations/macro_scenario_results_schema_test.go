package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroScenarioResultsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000033_macro_scenario_results.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000033_macro_scenario_results.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_scenario_results",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"scenario_key TEXT NOT NULL",
		"candidate_bias TEXT NOT NULL",
		"primary_symbols TEXT[] NOT NULL",
		"secondary_symbols TEXT[] NOT NULL",
		"required_confirmations TEXT[] NOT NULL",
		"expected_reactions JSONB NOT NULL DEFAULT '{}'::jsonb",
		"result TEXT NOT NULL",
		"UNIQUE(macro_event_id, scenario_key)",
		"chk_macro_scenario_result",
		"idx_macro_scenario_results_event",
		"idx_macro_scenario_results_result",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_scenario_results_result",
		"DROP TABLE IF EXISTS macro_scenario_results",
	} {
		requireContains(t, down, fragment)
	}
}
