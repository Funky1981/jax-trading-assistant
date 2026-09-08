// Package advancedquant contains bounded, research-only contracts for Phase 12.
//
// The package deliberately uses the existing Go modular-monolith and standard
// library. It does not add a Python runtime, Qlib, RD-Agent, hosted inference,
// or an external experiment service.
package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const SuitabilityContractV1 = "jax.phase12.suitability/v1"

type Candidate string

const (
	CandidateExistingJax Candidate = "EXISTING_JAX_RESEARCH_ARCHITECTURE"
	CandidateQlib        Candidate = "QLIB"
	CandidateRDAgent     Candidate = "RD_AGENT"
)

type Decision string

const (
	DecisionRecommended  Decision = "RECOMMENDED"
	DecisionNotJustified Decision = "NOT_JUSTIFIED"
)

// SuitabilityAssessment is a frozen concept-level assessment. It is not an
// adoption or deployment instruction.
type SuitabilityAssessment struct {
	ContractVersion       string    `json:"contract_version"`
	AssessmentID          string    `json:"assessment_id"`
	Candidate             Candidate `json:"candidate"`
	Decision              Decision  `json:"decision"`
	LeakageSafety         string    `json:"leakage_safety"`
	Reproducibility       string    `json:"reproducibility"`
	OperationalImpact     string    `json:"operational_impact"`
	IntegrationImpact     string    `json:"integration_impact"`
	LicensingReview       string    `json:"licensing_review"`
	DataRequirements      string    `json:"data_requirements"`
	DecisionRationale     string    `json:"decision_rationale"`
	ProductionDependency  bool      `json:"production_dependency"`
	ExternalServiceNeeded bool      `json:"external_service_needed"`
}

func DefaultSuitabilityAssessments() []SuitabilityAssessment {
	return []SuitabilityAssessment{
		newAssessment(CandidateExistingJax, DecisionRecommended,
			"Existing typed contracts and deterministic replay can carry the bounded event study.",
			"No new runtime; reuse existing dataset, replay, evaluation and experiment stores.",
			"Existing Jax contracts are the least operationally complex route."),
		newAssessment(CandidateQlib, DecisionNotJustified,
			"Qlib offers broad research functions, but its point-in-time integration and Python data/runtime boundary are not demonstrated for this dataset.",
			"Adoption would add a consequential dependency before a deterministic baseline establishes need.",
			"Suitability spike only; no production dependency is approved."),
		newAssessment(CandidateRDAgent, DecisionNotJustified,
			"RD-Agent automates hypothesis search, but automation is unnecessary before the registered event study and would increase leakage and multiple-testing risk.",
			"An agent runtime would add orchestration and reproducibility burden without a validated need.",
			"Suitability spike only; no production dependency is approved."),
	}
}

func newAssessment(candidate Candidate, decision Decision, leakage, operational, rationale string) SuitabilityAssessment {
	a := SuitabilityAssessment{
		ContractVersion:       SuitabilityContractV1,
		Candidate:             candidate,
		Decision:              decision,
		LeakageSafety:         leakage,
		Reproducibility:       "Frozen dataset, algorithm and configuration identities are required.",
		OperationalImpact:     operational,
		IntegrationImpact:     "Use existing Jax Go package boundaries and immutable artifacts.",
		LicensingReview:       "No adoption is made; any future dependency requires a separate license review.",
		DataRequirements:      "HYP-EVENT-001A daily SEC/Alpaca panel and event-time features.",
		DecisionRationale:     rationale,
		ProductionDependency:  decision == DecisionRecommended && candidate != CandidateExistingJax,
		ExternalServiceNeeded: false,
	}
	a.AssessmentID = assessmentID(a)
	return a
}

func (a SuitabilityAssessment) Validate() error {
	if a.ContractVersion != SuitabilityContractV1 || !strings.HasPrefix(a.AssessmentID, "suit_") || len(a.AssessmentID) != len("suit_")+64 {
		return fmt.Errorf("suitability assessment identity or contract is invalid")
	}
	for _, value := range []string{string(a.Candidate), string(a.Decision), a.LeakageSafety, a.Reproducibility, a.OperationalImpact, a.IntegrationImpact, a.LicensingReview, a.DataRequirements, a.DecisionRationale} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("suitability assessment contains an empty required field")
		}
	}
	switch a.Candidate {
	case CandidateExistingJax, CandidateQlib, CandidateRDAgent:
	default:
		return fmt.Errorf("unsupported suitability candidate %q", a.Candidate)
	}
	switch a.Decision {
	case DecisionRecommended, DecisionNotJustified:
	default:
		return fmt.Errorf("unsupported suitability decision %q", a.Decision)
	}
	if a.Candidate != CandidateExistingJax && (a.Decision != DecisionNotJustified || a.ProductionDependency || a.ExternalServiceNeeded) {
		return fmt.Errorf("external suitability candidates cannot be adopted by this spike")
	}
	if a.AssessmentID != assessmentID(a) {
		return fmt.Errorf("suitability assessment identity does not match contents")
	}
	return nil
}

func assessmentID(a SuitabilityAssessment) string {
	a.AssessmentID = ""
	b, _ := json.Marshal(a)
	digest := sha256.Sum256(b)
	return "suit_" + hex.EncodeToString(digest[:])
}
