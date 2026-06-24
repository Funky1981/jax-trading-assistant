package research

type EvidenceBundle struct {
	Hypothesis ResearchHypothesis `json:"hypothesis"`
	Backtest   BacktestEvidence   `json:"backtest_evidence"`
}

func ValidateEvidenceBundle(bundle EvidenceBundle) ValidationResult {
	result := ValidateBacktestEvidence(bundle.Backtest)
	if !bundle.Hypothesis.HasPaperReadyDefinition() && result.PromotionDecision == PromotionPaperReady {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, "paper_ready requires a complete research hypothesis")
		result.RequiredRemediation = append(result.RequiredRemediation, "define hypothesis claim, setup family, target assets, entry, exit, risk rule, and expected failure modes")
		result.MaxAllowedPromotionState = PromotionBacktestedPromising
		result.PromotionDecision = PromotionBacktestedPromising
	}
	return result
}
