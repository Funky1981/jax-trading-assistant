package main

import "testing"

func TestBuildAIEvidenceInputContainsOnlyStructuredEvidence(t *testing.T) {
	bundle := researchEvidenceBundle{
		EventID:    "event-1",
		Symbol:     "SMH",
		EventType:  "semiconductor_ai",
		Headline:   "Nvidia chip demand surges",
		Source:     "finnhub",
		EventTime:  "2026-01-02T15:04:05Z",
		WhyThisETF: "SMH maps to semiconductor_ai",
		PriceReaction: evidencePriceReaction{
			Pre1H:   0.001,
			Post15M: 0.012,
			Post1H:  0.018,
		},
		PricedIn: evidencePricedIn{
			Verdict: "not_priced_in",
			Score:   0.15,
			Reason:  "pre-event drift was small and post-event confirmation was strong",
		},
		Guardrails: evidenceGuardrails{
			AllowlistPass:    true,
			SpreadPass:       true,
			StaleQuotePass:   true,
			PaperModePass:    true,
			ApprovalRequired: true,
		},
		BeginnerSummary: evidenceBeginnerSummary{
			WhatHappened: "Nvidia chip demand surged.",
			WalkAway:     "Jax walks away if guardrails fail.",
		},
		DetailedFields: map[string]any{
			"raw_provider_payload": map[string]any{"should_not": "leak"},
			"windows":              []string{"internal"},
		},
	}

	input := buildAIResearchInput(bundle)
	if input.EventID != bundle.EventID || input.Symbol != "SMH" {
		t.Fatalf("identity fields not copied: %#v", input)
	}
	if input.Evidence.PriceReaction.Post15M == 0 {
		t.Fatalf("price reaction missing from input: %#v", input.Evidence.PriceReaction)
	}
	if input.RawEvidence != nil {
		t.Fatalf("AI input must not include raw/detail fields, got %#v", input.RawEvidence)
	}
	if !input.Policy.ApprovalRequired || input.Policy.AdvisoryOnly != true {
		t.Fatalf("policy boundary missing: %#v", input.Policy)
	}
}

func TestValidateAIResearchOutputRejectsTradeWhenEvidenceHardRejects(t *testing.T) {
	bundle := researchEvidenceBundle{
		EventID: "event-1",
		Symbol:  "SPY",
		PricedIn: evidencePricedIn{
			Verdict:           "priced_in",
			Score:             0.85,
			Reason:            "pre-event drift already exceeded threshold",
			HardReject:        true,
			HardRejectReasons: []string{"priced_in"},
		},
		Guardrails: evidenceGuardrails{
			AllowlistPass:    true,
			SpreadPass:       true,
			StaleQuotePass:   true,
			PaperModePass:    true,
			ApprovalRequired: true,
			HardReject:       true,
		},
	}
	output := AIResearchOutput{
		Decision:            "trade",
		Confidence:          0.9,
		PlainEnglishSummary: "Trade it.",
		WhyThisETF:          "SPY is relevant.",
		PricedInView:        "Looks fine.",
		MainRisks:           []string{"priced in"},
		ApprovalMessage:     "Approve this trade.",
	}

	result := validateAIResearchOutput(bundle, output)
	if result.Accepted {
		t.Fatalf("expected hard reject override, got accepted result %#v", result)
	}
	if !containsString(result.Reasons, "hard_reject_override") {
		t.Fatalf("expected hard_reject_override reason, got %#v", result.Reasons)
	}
	if result.EffectiveDecision != "reject" {
		t.Fatalf("effective decision = %q, want reject", result.EffectiveDecision)
	}
}

func TestValidateAIResearchOutputRejectsInvalidOrUnsupportedOutput(t *testing.T) {
	bundle := researchEvidenceBundle{
		EventID: "event-1",
		Symbol:  "QQQ",
		Guardrails: evidenceGuardrails{
			AllowlistPass:    true,
			SpreadPass:       true,
			StaleQuotePass:   true,
			PaperModePass:    true,
			ApprovalRequired: true,
		},
	}
	output := AIResearchOutput{
		Decision:        "execute",
		Confidence:      1.2,
		WhyThisETF:      "QQQ is relevant.",
		PricedInView:    "not priced in",
		ApprovalMessage: "submit broker order now",
	}

	result := validateAIResearchOutput(bundle, output)
	if result.Accepted {
		t.Fatalf("expected invalid output rejection, got %#v", result)
	}
	for _, reason := range []string{"invalid_decision", "confidence_out_of_range", "missing_plain_english_summary", "forbidden_execution_language"} {
		if !containsString(result.Reasons, reason) {
			t.Fatalf("missing reason %q in %#v", reason, result.Reasons)
		}
	}
}

func TestValidateAIResearchOutputAcceptsAdvisoryWaitDecision(t *testing.T) {
	bundle := researchEvidenceBundle{
		EventID: "event-1",
		Symbol:  "TLT",
		PricedIn: evidencePricedIn{
			Verdict: "partially_priced_in",
			Score:   0.52,
			Reason:  "some pre-event drift existed",
		},
		Guardrails: evidenceGuardrails{
			AllowlistPass:    true,
			SpreadPass:       true,
			StaleQuotePass:   true,
			PaperModePass:    true,
			ApprovalRequired: true,
		},
	}
	output := AIResearchOutput{
		Decision:            "wait",
		Confidence:          0.62,
		PlainEnglishSummary: "Wait for cleaner confirmation.",
		WhyThisETF:          "TLT is rates-sensitive.",
		PricedInView:        "Partially priced in.",
		MainRisks:           []string{"rates can reverse"},
		WalkAwayReason:      "spread widens",
		ApprovalMessage:     "Watch TLT; approval is still required before any trade.",
	}

	result := validateAIResearchOutput(bundle, output)
	if !result.Accepted {
		t.Fatalf("expected accepted advisory output, got %#v", result)
	}
	if result.EffectiveDecision != "wait" {
		t.Fatalf("effective decision = %q, want wait", result.EffectiveDecision)
	}
}

func TestAttachAIResearchRecommendationStoresAuditableDecision(t *testing.T) {
	bundle := researchEvidenceBundle{
		EventID: "event-1",
		Symbol:  "TLT",
		Guardrails: evidenceGuardrails{
			AllowlistPass:    true,
			SpreadPass:       true,
			StaleQuotePass:   true,
			PaperModePass:    true,
			ApprovalRequired: true,
		},
	}
	output := AIResearchOutput{
		Decision:            "reject",
		Confidence:          0.74,
		PlainEnglishSummary: "Reject because confirmation is weak.",
		WhyThisETF:          "TLT is rates-sensitive.",
		PricedInView:        "Unclear.",
		MainRisks:           []string{"mixed reaction"},
		WalkAwayReason:      "unclear reaction",
		ApprovalMessage:     "No trade should be approved from this evidence.",
	}
	validation := validateAIResearchOutput(bundle, output)

	updated := attachAIResearchRecommendation(bundle, output, validation)
	raw, ok := updated.DetailedFields["ai_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected ai_recommendation detail, got %#v", updated.DetailedFields)
	}
	if raw["decision"] != "reject" || raw["accepted"] != true || raw["effective_decision"] != "reject" {
		t.Fatalf("unexpected stored recommendation: %#v", raw)
	}
	if raw["advisory_only"] != true {
		t.Fatalf("stored recommendation must preserve advisory-only boundary: %#v", raw)
	}
}
