package main

import (
	"strings"
)

type AIResearchInput struct {
	EventID     string             `json:"event_id"`
	Symbol      string             `json:"symbol"`
	Evidence    AIResearchEvidence `json:"evidence"`
	Policy      AIResearchPolicy   `json:"policy"`
	RawEvidence map[string]any     `json:"raw_evidence,omitempty"`
}

type AIResearchEvidence struct {
	EventType       string                  `json:"event_type"`
	Headline        string                  `json:"headline"`
	Source          string                  `json:"source"`
	EventTime       string                  `json:"event_time"`
	WhyThisETF      string                  `json:"why_this_etf"`
	PriceReaction   evidencePriceReaction   `json:"price_reaction"`
	PricedIn        evidencePricedIn        `json:"priced_in"`
	Confounders     []evidenceConfounder    `json:"confounders"`
	Guardrails      evidenceGuardrails      `json:"guardrails"`
	BeginnerSummary evidenceBeginnerSummary `json:"beginner_summary"`
}

type AIResearchPolicy struct {
	AdvisoryOnly          bool     `json:"advisory_only"`
	AllowedDecisions      []string `json:"allowed_decisions"`
	ApprovalRequired      bool     `json:"approval_required"`
	ExecutionInstructions bool     `json:"execution_instructions_allowed"`
	DeterministicOverride bool     `json:"deterministic_override_allowed"`
	BlockedIfHardReject   bool     `json:"blocked_if_hard_reject"`
}

type AIResearchOutput struct {
	Decision            string   `json:"decision"`
	Confidence          float64  `json:"confidence"`
	PlainEnglishSummary string   `json:"plain_english_summary"`
	WhyThisETF          string   `json:"why_this_etf"`
	PricedInView        string   `json:"priced_in_view"`
	MainRisks           []string `json:"main_risks"`
	WalkAwayReason      string   `json:"walk_away_reason,omitempty"`
	ApprovalMessage     string   `json:"approval_message"`
}

type AIResearchValidationResult struct {
	Accepted          bool     `json:"accepted"`
	EffectiveDecision string   `json:"effective_decision"`
	Reasons           []string `json:"reasons,omitempty"`
}

func buildAIResearchInput(bundle researchEvidenceBundle) AIResearchInput {
	return AIResearchInput{
		EventID: bundle.EventID,
		Symbol:  bundle.Symbol,
		Evidence: AIResearchEvidence{
			EventType:       bundle.EventType,
			Headline:        bundle.Headline,
			Source:          bundle.Source,
			EventTime:       bundle.EventTime,
			WhyThisETF:      bundle.WhyThisETF,
			PriceReaction:   bundle.PriceReaction,
			PricedIn:        bundle.PricedIn,
			Confounders:     bundle.Confounders,
			Guardrails:      bundle.Guardrails,
			BeginnerSummary: bundle.BeginnerSummary,
		},
		Policy: AIResearchPolicy{
			AdvisoryOnly:          true,
			AllowedDecisions:      []string{"trade", "wait", "reject"},
			ApprovalRequired:      bundle.Guardrails.ApprovalRequired,
			ExecutionInstructions: false,
			DeterministicOverride: false,
			BlockedIfHardReject:   true,
		},
	}
}

func validateAIResearchOutput(bundle researchEvidenceBundle, output AIResearchOutput) AIResearchValidationResult {
	result := AIResearchValidationResult{
		Accepted:          true,
		EffectiveDecision: strings.ToLower(strings.TrimSpace(output.Decision)),
	}
	if result.EffectiveDecision == "" {
		result.EffectiveDecision = "reject"
	}
	if !validAIDecision(result.EffectiveDecision) {
		result.Reasons = append(result.Reasons, "invalid_decision")
	}
	if output.Confidence < 0 || output.Confidence > 1 {
		result.Reasons = append(result.Reasons, "confidence_out_of_range")
	}
	if strings.TrimSpace(output.PlainEnglishSummary) == "" {
		result.Reasons = append(result.Reasons, "missing_plain_english_summary")
	}
	if strings.TrimSpace(output.WhyThisETF) == "" {
		result.Reasons = append(result.Reasons, "missing_why_this_etf")
	}
	if strings.TrimSpace(output.PricedInView) == "" {
		result.Reasons = append(result.Reasons, "missing_priced_in_view")
	}
	if len(output.MainRisks) == 0 {
		result.Reasons = append(result.Reasons, "missing_main_risks")
	}
	if strings.TrimSpace(output.ApprovalMessage) == "" {
		result.Reasons = append(result.Reasons, "missing_approval_message")
	}
	if containsForbiddenExecutionLanguage(output) {
		result.Reasons = append(result.Reasons, "forbidden_execution_language")
	}
	if result.EffectiveDecision == "trade" && hasFailedDeterministicGuardrail(bundle) {
		result.Reasons = append(result.Reasons, "failed_guardrail_override")
	}
	if result.EffectiveDecision == "trade" && (bundle.PricedIn.HardReject || bundle.Guardrails.HardReject) {
		result.Reasons = append(result.Reasons, "hard_reject_override")
	}
	if result.EffectiveDecision == "trade" && (bundle.PricedIn.Verdict == "priced_in" || bundle.PricedIn.Verdict == "unclear") {
		result.Reasons = append(result.Reasons, "priced_in_override")
	}
	if len(result.Reasons) > 0 {
		result.Accepted = false
		if containsString(result.Reasons, "hard_reject_override") ||
			containsString(result.Reasons, "failed_guardrail_override") ||
			containsString(result.Reasons, "priced_in_override") {
			result.EffectiveDecision = "reject"
		}
	}
	return result
}

func attachAIResearchRecommendation(bundle researchEvidenceBundle, output AIResearchOutput, validation AIResearchValidationResult) researchEvidenceBundle {
	if bundle.DetailedFields == nil {
		bundle.DetailedFields = map[string]any{}
	}
	bundle.DetailedFields["ai_recommendation"] = map[string]any{
		"decision":              strings.ToLower(strings.TrimSpace(output.Decision)),
		"effective_decision":    validation.EffectiveDecision,
		"confidence":            output.Confidence,
		"accepted":              validation.Accepted,
		"validation_reasons":    append([]string(nil), validation.Reasons...),
		"plain_english_summary": output.PlainEnglishSummary,
		"why_this_etf":          output.WhyThisETF,
		"priced_in_view":        output.PricedInView,
		"main_risks":            append([]string(nil), output.MainRisks...),
		"walk_away_reason":      output.WalkAwayReason,
		"approval_message":      output.ApprovalMessage,
		"advisory_only":         true,
		"approval_required":     bundle.Guardrails.ApprovalRequired,
	}
	return bundle
}

func validAIDecision(decision string) bool {
	switch decision {
	case "trade", "wait", "reject":
		return true
	default:
		return false
	}
}

func hasFailedDeterministicGuardrail(bundle researchEvidenceBundle) bool {
	return !bundle.Guardrails.AllowlistPass ||
		!bundle.Guardrails.SpreadPass ||
		!bundle.Guardrails.StaleQuotePass ||
		!bundle.Guardrails.PaperModePass
}

func containsForbiddenExecutionLanguage(output AIResearchOutput) bool {
	text := strings.ToLower(strings.Join([]string{
		output.Decision,
		output.PlainEnglishSummary,
		output.WhyThisETF,
		output.PricedInView,
		strings.Join(output.MainRisks, " "),
		output.WalkAwayReason,
		output.ApprovalMessage,
	}, " "))
	for _, phrase := range []string{
		"submit broker order",
		"place order",
		"execute order",
		"approve this trade",
		"approved trade",
		"increase position size",
		"enable live trading",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}
