package main

import (
	"testing"

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
	want := int64(290_000)*400_000/1_000_000 + int64(10_000)*cachePrice/1_000_000 + int64(256)*1_800_000/1_000_000
	if got := costMicros(u); got != want {
		t.Fatalf("cost=%d want=%d", got, want)
	}
}
