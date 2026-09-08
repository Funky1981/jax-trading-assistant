package hypevidence

import (
	"testing"
	"time"
)

func TestDirectionContractIsFrozenAndEventTimeBound(t *testing.T) {
	c := DefaultDirectionClassifierContract()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.OutputTokenCeiling != 256 || c.MaxRetries != 1 {
		t.Fatalf("unexpected bounded limits: %#v", c)
	}
	if c.PromptVersion != PromptIdentity() {
		t.Fatal("prompt identity is not content-bound")
	}
	t.Logf("prompt_overhead_estimated_tokens=%d prompt_version=%s", PromptOverheadTokens(), PromptIdentity())
}

func TestDirectionResultRejectsUngroundedPolarityAndNonUTC(t *testing.T) {
	r := DirectionResult{
		ContractVersion:      DirectionContractVersion,
		HypothesisID:         "HYP-EVENT-001A",
		EventID:              "evt-1",
		Accession:            "0000000000-26-000001",
		EvidencePacketSHA256: "packet-hash",
		AvailabilityCutoff:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("BST", 3600)),
		Direction:            DirectionPositive,
		ReasonCode:           ReasonGroundedDirection,
		ClassifierVersion:    DirectionContractVersion,
		PromptVersion:        PromptIdentity(),
		Provider:             "openai",
		Model:                "gpt-5.6-luna",
		InferenceTimestamp:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		RawResponseSHA256:    "response-hash",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected non-UTC timestamp to be rejected")
	}
	r.AvailabilityCutoff = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := r.Validate(); err == nil {
		t.Fatal("expected ungrounded polarity to be rejected")
	}
	r.EvidenceAnchors = []string{"document:primary#item-1"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDirectionResultAllowsExplicitAbstention(t *testing.T) {
	r := DirectionResult{
		ContractVersion:      DirectionContractVersion,
		HypothesisID:         "HYP-EVENT-001A",
		EventID:              "evt-1",
		Accession:            "0000000000-26-000001",
		EvidencePacketSHA256: "packet-hash",
		AvailabilityCutoff:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Direction:            DirectionInsufficientEvidence,
		ReasonCode:           ReasonInsufficientEvidence,
		EvidenceAnchors:      []string{"document:primary#ambiguous"},
		ClassifierVersion:    DirectionContractVersion,
		PromptVersion:        PromptIdentity(),
		Provider:             "openai",
		Model:                "gpt-5.6-luna",
		InferenceTimestamp:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		RawResponseSHA256:    "response-hash",
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}
