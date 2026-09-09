package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/hypevidence"
)

func TestClassificationParserRejectsUnknownAndTrailingFields(t *testing.T) {
	if _, err := parseClassification(`{"direction":"POSITIVE","reason_code":"GROUNDED_DIRECTION","evidence_anchors":[],"extra":true}`); err == nil {
		t.Fatal("unknown field was accepted")
	}
	if _, err := parseClassification(`{"direction":"POSITIVE","reason_code":"GROUNDED_DIRECTION","evidence_anchors":[]} {}`); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

func TestClassificationValidationRequiresGroundingAndKnownDirection(t *testing.T) {
	p := packet{Documents: []hypevidence.Document{{Filename: "primary.htm"}}}
	if err := validateClassification(classification{Direction: hypevidence.DirectionPositive, ReasonCode: hypevidence.ReasonGroundedDirection}, p, ""); err == nil {
		t.Fatal("ungrounded polarity was accepted")
	}
	if err := validateClassification(classification{Direction: "FORGED", ReasonCode: hypevidence.ReasonGroundedDirection, EvidenceAnchors: []string{"claim"}}, p, ""); err == nil {
		t.Fatal("unsupported direction was accepted")
	}
	if err := validateClassification(classification{Direction: hypevidence.DirectionPositive, ReasonCode: hypevidence.ReasonGroundedDirection, EvidenceAnchors: []string{"document:primary.htm#item-1"}}, p, ""); err != nil {
		t.Fatalf("valid document anchor rejected: %v", err)
	}
}

func TestCostMicrosUsesCachedTokensAndLargeInputTier(t *testing.T) {
	u := usage{InputTokens: 300_000, OutputTokens: 256}
	u.InputDetails.CachedTokens = 10_000
	want := int64(290_000)*400_000/1_000_000 + int64(10_000)*(cachePrice*2)/1_000_000 + (int64(256)*1_800_000+999_999)/1_000_000
	if got := costMicros(u); got != want {
		t.Fatalf("cost=%d want=%d", got, want)
	}
}

func TestBulkInferenceRequiresExactQualificationPass(t *testing.T) {
	dir := t.TempDir()
	artifact, err := hypevidence.NewQualificationArtifact(hypevidence.QualificationArtifact{
		SampleIdentity: "sample-v1", SampleSelectionPolicy: "development-only-stratified-v1", EventIDs: []string{"evt-1"},
		ClassifierVersion: hypevidence.DirectionContractVersion, PromptVersion: hypevidence.PromptIdentity(), Provider: providerName, Model: modelName,
		Attempts:             []hypevidence.QualificationAttempt{{EventID: "evt-1", AttemptNumber: 1, RequestID: "req-1", ResponseID: "resp-1", UsageArtifactID: "usage-1", RawResponseSHA256: hypevidence.SHA256Hex([]byte("raw")), InputTokens: 100, OutputTokens: 10, TotalTokens: 110}},
		SemanticReviewMethod: "blinded-development-review-v1", Decision: hypevidence.QualificationPass, DecisionAt: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "qualification.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireQualification(path); err != nil {
		t.Fatal(err)
	}
	artifact.PromptVersion = "changed"
	b, _ = json.Marshal(artifact)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireQualification(path); err == nil {
		t.Fatal("changed classifier identity reused qualification")
	}
}
