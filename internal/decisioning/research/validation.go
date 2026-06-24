package research

const MinimumPromisingSampleSize = 30

type ValidationResult struct {
	IsValid                  bool           `json:"is_valid"`
	ValidationErrors         []string       `json:"validation_errors"`
	ValidationWarnings       []string       `json:"validation_warnings"`
	MaxAllowedPromotionState PromotionState `json:"max_allowed_promotion_state"`
	PromotionDecision        PromotionState `json:"promotion_decision"`
	RequiredRemediation      []string       `json:"required_remediation"`
	EvidenceQualityScore     float64        `json:"evidence_quality_score"`
}

func ValidateBacktestEvidence(e BacktestEvidence) ValidationResult {
	result := ValidationResult{
		IsValid:                  true,
		MaxAllowedPromotionState: PromotionBacktestedPromising,
	}

	require := func(ok bool, message string) {
		if ok {
			return
		}
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, message)
		result.RequiredRemediation = append(result.RequiredRemediation, message)
		result.MaxAllowedPromotionState = weakerCap(result.MaxAllowedPromotionState, PromotionBacktestedWeak)
	}

	require(e.DatasetID != "", "dataset_id is required")
	require(e.DatasetHash != "", "dataset_hash is required")
	require(e.DateRange.IsDefined(), "date_range is required")
	require(len(e.InstrumentUniverse) > 0, "instrument_universe is required")
	require(e.Benchmark != "", "benchmark is required")
	require(e.Assumptions.IsDefined(), "assumptions are required")
	require(e.SlippageModel.IsDefined(), "slippage_model is required")
	require(e.FeesModel.IsDefined(), "fees_model is required")
	require(e.InSamplePeriod.IsDefined(), "in_sample_period is required")
	require(e.SampleSize > 0, "sample_size is required")
	require(e.DrawdownMetrics.IsDefined(), "drawdown_metrics are required")
	require(e.Expectancy != 0, "expectancy is required")
	require(len(e.FailureModes) > 0, "failure_modes are required")

	if !e.HasOutOfSampleEvidence() {
		if !e.HasOutOfSampleLimitation() {
			result.IsValid = false
			result.ValidationErrors = append(result.ValidationErrors, "out_of_sample_period or out_of_sample_limitation_note is required")
			result.RequiredRemediation = append(result.RequiredRemediation, "add out-of-sample validation or an explicit limitation note")
		} else {
			result.ValidationWarnings = append(result.ValidationWarnings, "out-of-sample evidence is limited")
		}
		result.MaxAllowedPromotionState = weakerCap(result.MaxAllowedPromotionState, PromotionBacktestedWeak)
	}

	if e.SampleSize > 0 && e.SampleSize < MinimumPromisingSampleSize {
		result.ValidationWarnings = append(result.ValidationWarnings, "sample_size is too small for promising evidence")
		result.RequiredRemediation = append(result.RequiredRemediation, "increase sample size before promotion beyond BACKTESTED_WEAK")
		result.MaxAllowedPromotionState = weakerCap(result.MaxAllowedPromotionState, PromotionBacktestedWeak)
	}

	if e.PromotionDecision == disallowedLiveReady {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, "LIVE_READY is not allowed in the current roadmap")
		result.RequiredRemediation = append(result.RequiredRemediation, "use paper-trading promotion states only")
	}
	if e.PromotionDecision != "" && e.PromotionDecision != disallowedLiveReady && !IsAllowedPromotionState(e.PromotionDecision) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, "promotion_decision is not an allowed research state")
		result.RequiredRemediation = append(result.RequiredRemediation, "choose an allowed research promotion state")
		result.MaxAllowedPromotionState = weakerCap(result.MaxAllowedPromotionState, PromotionBacktestedWeak)
	}

	if e.PromotionDecision == PromotionPaperReady {
		if result.MaxAllowedPromotionState != PromotionBacktestedPromising {
			result.IsValid = false
			result.ValidationErrors = append(result.ValidationErrors, "PAPER_READY requires promising backtest evidence")
			result.RequiredRemediation = append(result.RequiredRemediation, "remediate evidence caps before requesting PAPER_READY")
		}
		if e.SetupFamily == "" {
			result.IsValid = false
			result.ValidationErrors = append(result.ValidationErrors, "PAPER_READY requires setup_family")
			result.RequiredRemediation = append(result.RequiredRemediation, "define setup_family")
		}
		if !e.HasDefinedRiskRules() {
			result.IsValid = false
			result.ValidationErrors = append(result.ValidationErrors, "PAPER_READY requires defined risk rules")
			result.RequiredRemediation = append(result.RequiredRemediation, "define risk rules before paper readiness")
		}
		if result.IsValid {
			result.MaxAllowedPromotionState = PromotionPaperReady
		}
	}

	result.PromotionDecision = capPromotion(e.PromotionDecision, result.MaxAllowedPromotionState)
	result.EvidenceQualityScore = scoreEvidenceQuality(e, result)
	return result
}

func weakerCap(current PromotionState, candidate PromotionState) PromotionState {
	if current == "" {
		return candidate
	}
	if promotionRank[candidate] < promotionRank[current] {
		return candidate
	}
	return current
}

func scoreEvidenceQuality(e BacktestEvidence, result ValidationResult) float64 {
	score := 1.0
	score -= float64(len(result.ValidationErrors)) * 0.15
	score -= float64(len(result.ValidationWarnings)) * 0.05
	if e.HasOutOfSampleEvidence() {
		score += 0.05
	}
	if e.SampleSize >= MinimumPromisingSampleSize {
		score += 0.05
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
