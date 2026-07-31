package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestEvidenceSubjectsMigrationDefinesDurableIdempotentContract(t *testing.T) {
	data, err := os.ReadFile("000052_evidence_subjects.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	migration := string(data)
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS evidence_subjects",
		"deterministic_subject_key TEXT NOT NULL UNIQUE",
		"current_missing_evidence TEXT[] NOT NULL",
		"CREATE TABLE IF NOT EXISTS evidence_subject_events",
		"CONSTRAINT uq_evidence_subject_event UNIQUE (genuine_event_id)",
		"source_independence TEXT NOT NULL",
		"CREATE TABLE IF NOT EXISTS evidence_subject_evaluations",
		"idempotency_identity TEXT NOT NULL UNIQUE",
		"CONSTRAINT uq_evidence_subject_input UNIQUE (subject_id, ruleset_version, evidence_set_fingerprint)",
		"CREATE TABLE IF NOT EXISTS evidence_subject_candidates",
	} {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
