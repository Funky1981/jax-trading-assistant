package researchrecommendation

import (
	"fmt"
)

const ConfidenceAssessmentContractV1 = "jax.confidence_assessment/v1"

type CalibrationStatus string

const (
	CalibrationNotYetCalibrated CalibrationStatus = "NOT_YET_CALIBRATED"
	CalibrationNotApplicable    CalibrationStatus = "NOT_APPLICABLE"
)

// ConfidenceAssessment deliberately does not expose a calibrated probability.
// ModelScore is an uncalibrated model-reported signal and must not be used as
// empirical win probability until a later evaluation phase proves calibration.
type ConfidenceAssessment struct {
	ContractVersion        string            `json:"contract_version"`
	CalibrationStatus      CalibrationStatus `json:"calibration_status"`
	Scope                  string            `json:"scope"`
	EvidenceSufficient     bool              `json:"evidence_sufficient"`
	IndependentSourceCount int               `json:"independent_source_count"`
	ContradictionCount     int               `json:"contradiction_count"`
	UnknownCount           int               `json:"unknown_count"`
	ModelScore             *float64          `json:"model_score,omitempty"`
	Uncertainty            []string          `json:"uncertainty"`
}

func NewConfidenceAssessment(modelScore *float64, evidenceSufficient bool, independentSources, contradictions, unknowns int, uncertainty []string) (ConfidenceAssessment, error) {
	assessment := ConfidenceAssessment{ContractVersion: ConfidenceAssessmentContractV1, CalibrationStatus: CalibrationNotYetCalibrated, Scope: "RESEARCH_INTERPRETATION_ONLY", EvidenceSufficient: evidenceSufficient, IndependentSourceCount: independentSources, ContradictionCount: contradictions, UnknownCount: unknowns, ModelScore: modelScore, Uncertainty: sortedUnique(uncertainty)}
	if err := assessment.Validate(); err != nil {
		return ConfidenceAssessment{}, err
	}
	return assessment, nil
}

func (assessment ConfidenceAssessment) Validate() error {
	if assessment.ContractVersion != ConfidenceAssessmentContractV1 || assessment.Scope != "RESEARCH_INTERPRETATION_ONLY" || assessment.IndependentSourceCount < 0 || assessment.ContradictionCount < 0 || assessment.UnknownCount < 0 || len(assessment.Uncertainty) == 0 {
		return fmt.Errorf("confidence assessment contract, scope, counts, and uncertainty are required")
	}
	if assessment.CalibrationStatus != CalibrationNotYetCalibrated && assessment.CalibrationStatus != CalibrationNotApplicable {
		return fmt.Errorf("unsupported calibration status")
	}
	if assessment.ModelScore != nil && (*assessment.ModelScore < 0 || *assessment.ModelScore > 1 || !finite(*assessment.ModelScore)) {
		return fmt.Errorf("uncalibrated model score must be finite and between zero and one")
	}
	copyUncertainty := sortedUnique(assessment.Uncertainty)
	if len(copyUncertainty) != len(assessment.Uncertainty) {
		return fmt.Errorf("confidence uncertainty entries must be unique and non-empty")
	}
	return nil
}
