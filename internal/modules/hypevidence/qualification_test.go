package hypevidence

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func qualificationFixture(t *testing.T, decision QualificationDecision) QualificationArtifact {
	t.Helper()
	a, err := NewQualificationArtifact(QualificationArtifact{
		SampleIdentity: "sample-v1", SampleSelectionPolicy: "development-only-stratified-v1", EventIDs: []string{"evt-1"},
		ClassifierVersion: DirectionContractVersion, PromptVersion: PromptIdentity(), Provider: "openai", Model: "gpt-5.6-luna",
		Attempts:             []QualificationAttempt{{EventID: "evt-1", AttemptNumber: 1, RequestID: "req-1", ResponseID: "resp-1", UsageArtifactID: "usage-1", RawResponseSHA256: SHA256Hex([]byte("response")), InputTokens: 100, OutputTokens: 10, TotalTokens: 110, StructuredValid: true, EvidenceAnchorsValid: true, FutureInfoCheck: true, InjectionCheck: true, AbstentionCheck: true}},
		SemanticReviewMethod: "blinded-development-review-v1", StructuredValidCount: 1, GroundedCount: 1, FutureInfoPassCount: 1, InjectionPassCount: 1, AbstentionPassCount: 1,
		Metrics: map[string]float64{"structured_valid_rate": 1}, Decision: decision, DecisionAt: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestQualificationCompatibilityFailsClosed(t *testing.T) {
	a := qualificationFixture(t, QualificationPass)
	if err := a.Compatible(DirectionContractVersion, PromptIdentity(), "openai", "gpt-5.6-luna"); err != nil {
		t.Fatal(err)
	}
	if err := a.Compatible(DirectionContractVersion, "changed", "openai", "gpt-5.6-luna"); err == nil {
		t.Fatal("changed prompt reused qualification")
	}
	if err := a.Compatible(DirectionContractVersion, PromptIdentity(), "openai", "other"); err == nil {
		t.Fatal("changed model reused qualification")
	}
}

func TestQualificationFailureAndMissingAttemptUsageAreRejected(t *testing.T) {
	failed := qualificationFixture(t, QualificationFail)
	if err := failed.Compatible(DirectionContractVersion, PromptIdentity(), "openai", "gpt-5.6-luna"); err == nil {
		t.Fatal("failed qualification was accepted")
	}
	broken := qualificationFixture(t, QualificationPass)
	broken.Attempts[0].UsageArtifactID = ""
	broken.ID = qualificationArtifactID(broken)
	if err := broken.Validate(); err == nil {
		t.Fatal("qualification without durable usage was accepted")
	}
}

func TestQualificationArtifactPersistenceIsImmutable(t *testing.T) {
	artifact := qualificationFixture(t, QualificationPass)
	path := filepath.Join(t.TempDir(), "qualification.json")
	if err := WriteQualificationArtifact(path, artifact); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadQualificationArtifact(path)
	if err != nil || loaded.ID != artifact.ID {
		t.Fatalf("qualification artifact did not round-trip: %+v %v", loaded, err)
	}
	artifact.Decision = QualificationFail
	artifact.ID = qualificationArtifactID(artifact)
	if err := WriteQualificationArtifact(path, artifact); err == nil {
		t.Fatal("qualification artifact was overwritten")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
