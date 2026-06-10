package migrations

import (
	"path/filepath"
	"testing"
)

func TestMultiAnalystReviewsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000040_multi_analyst_reviews.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000040_multi_analyst_reviews.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS multi_analyst_reviews",
		"macro_event_id UUID NULL REFERENCES macro_events(id)",
		"symbol TEXT NOT NULL",
		"fundamental_snapshot_id UUID NULL",
		"technical_snapshot_id UUID NULL",
		"analyst_decision_id UUID NULL",
		"fundamental_verdict TEXT NOT NULL",
		"fundamental_score NUMERIC NOT NULL",
		"technical_verdict TEXT NOT NULL",
		"technical_score NUMERIC NOT NULL",
		"risk_verdict TEXT NOT NULL",
		"risk_score NUMERIC NOT NULL",
		"risk_hard_blocks TEXT[] NOT NULL",
		"review_decision TEXT NOT NULL",
		"candidate_score NUMERIC NOT NULL",
		"approval_required BOOLEAN NOT NULL",
		"review_reasons TEXT[] NOT NULL",
		"llm_override_attempted BOOLEAN NOT NULL",
		"chk_multi_analyst_reviews_risk_verdict",
		"chk_multi_analyst_reviews_review_decision",
		"chk_multi_analyst_reviews_candidate_score",
		"idx_multi_analyst_reviews_event",
		"idx_multi_analyst_reviews_symbol",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_multi_analyst_reviews_symbol",
		"DROP TABLE IF EXISTS multi_analyst_reviews",
	} {
		requireContains(t, down, fragment)
	}
}
