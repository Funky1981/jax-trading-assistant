package migrations

import (
	"path/filepath"
	"testing"
)

func TestAnalystMemoryCaseStudiesMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000041_analyst_memory_case_studies.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000041_analyst_memory_case_studies.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS analysis_case_studies",
		"macro_event_id UUID NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"event_type TEXT NOT NULL",
		"playbook_key TEXT NOT NULL",
		"technical_snapshot_id UUID NULL",
		"fundamental_snapshot_id UUID NULL",
		"analyst_decision_id UUID NULL",
		"review_id UUID NULL",
		"decision TEXT NOT NULL",
		"expected_outcome TEXT NOT NULL",
		"actual_outcome TEXT NULL",
		"outcome_r NUMERIC NULL",
		"surprise_bucket TEXT NULL",
		"technical_setup TEXT NULL",
		"market_regime TEXT NULL",
		"what_worked TEXT[] NOT NULL",
		"what_failed TEXT[] NOT NULL",
		"lesson TEXT NULL",
		"tags TEXT[] NOT NULL",
		"chk_analysis_case_studies_decision",
		"idx_analysis_case_studies_symbol",
		"idx_analysis_case_studies_event_playbook",
		"CREATE TABLE IF NOT EXISTS analyst_feedback",
		"case_study_id UUID NOT NULL REFERENCES analysis_case_studies(id)",
		"feedback_source TEXT NOT NULL",
		"rating TEXT NOT NULL",
		"comment TEXT NOT NULL",
		"chk_analyst_feedback_rating",
		"idx_analyst_feedback_case_study",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_analyst_feedback_case_study",
		"DROP TABLE IF EXISTS analyst_feedback",
		"DROP TABLE IF EXISTS analysis_case_studies",
	} {
		requireContains(t, down, fragment)
	}
}
