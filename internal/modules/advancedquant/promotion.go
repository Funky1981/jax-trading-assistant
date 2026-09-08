package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const PromotionContractV1 = "jax.phase12.promotion_gate/v1"

type PromotionStatus string

const (
	PromotionClosed   PromotionStatus = "PROMOTION_CLOSED"
	PromotionEligible PromotionStatus = "PROMOTION_ELIGIBLE_FOR_SEPARATE_REVIEW"
)

type PromotionEvidence struct {
	HypothesisID            string `json:"hypothesis_id"`
	ExperimentID            string `json:"experiment_id"`
	DatasetID               string `json:"dataset_id"`
	DatasetHash             string `json:"dataset_hash"`
	BaselineBeaten          bool   `json:"baseline_beaten"`
	Reproducible            bool   `json:"reproducible"`
	LeakageReviewPassed     bool   `json:"leakage_review_passed"`
	CostSensitivityPassed   bool   `json:"cost_sensitivity_passed"`
	DriftMonitoringDefined  bool   `json:"drift_monitoring_defined"`
	FailureBehaviourDefined bool   `json:"failure_behaviour_defined"`
	SurvivorshipControlled  bool   `json:"survivorship_controlled"`
	FinalHoldoutSealed      bool   `json:"final_holdout_sealed"`
	ForwardPaperDays        int    `json:"forward_paper_days"`
	ForwardPaperOrders      int    `json:"forward_paper_orders"`
}

type PromotionResult struct {
	ContractVersion               string            `json:"contract_version"`
	ID                            string            `json:"id"`
	Status                        PromotionStatus   `json:"status"`
	Evidence                      PromotionEvidence `json:"evidence"`
	ReasonCodes                   []string          `json:"reason_codes"`
	RecommendationMutationAllowed bool              `json:"recommendation_mutation_allowed"`
}

func EvaluatePromotion(evidence PromotionEvidence) (PromotionResult, error) {
	if strings.TrimSpace(evidence.HypothesisID) == "" || strings.TrimSpace(evidence.ExperimentID) == "" || strings.TrimSpace(evidence.DatasetID) == "" || !validSHA256(evidence.DatasetHash) || evidence.ForwardPaperDays < 0 || evidence.ForwardPaperOrders < 0 {
		return PromotionResult{}, fmt.Errorf("promotion evidence is incomplete")
	}
	result := PromotionResult{ContractVersion: PromotionContractV1, Status: PromotionEligible, Evidence: evidence, RecommendationMutationAllowed: false}
	result.ReasonCodes = promotionReasonCodes(evidence)
	if len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != "SEPARATE_RECOMMENDATION_REVIEW_REQUIRED" {
		result.Status = PromotionClosed
	}
	result.ID = promotionID(result)
	return result, nil
}

func (r PromotionResult) Validate() error {
	if r.ContractVersion != PromotionContractV1 || !validHashID(r.ID, "promote_") || (r.Status != PromotionClosed && r.Status != PromotionEligible) || r.RecommendationMutationAllowed || len(r.ReasonCodes) == 0 {
		return fmt.Errorf("promotion result is invalid or grants recommendation mutation")
	}
	if r.ID != promotionID(r) {
		return fmt.Errorf("promotion result identity does not match evidence")
	}
	expected := promotionReasonCodes(r.Evidence)
	if len(expected) != len(r.ReasonCodes) {
		return fmt.Errorf("promotion reason codes do not match evidence")
	}
	for index := range expected {
		if expected[index] != r.ReasonCodes[index] {
			return fmt.Errorf("promotion reason code mismatch")
		}
	}
	if (len(expected) == 1 && expected[0] == "SEPARATE_RECOMMENDATION_REVIEW_REQUIRED") != (r.Status == PromotionEligible) {
		return fmt.Errorf("promotion status does not match reason codes")
	}
	return nil
}

func promotionReasonCodes(evidence PromotionEvidence) []string {
	codes := []string{}
	if evidence.FinalHoldoutSealed {
		codes = append(codes, "FINAL_HOLDOUT_SEALED")
	}
	if !evidence.SurvivorshipControlled {
		codes = append(codes, "SURVIVORSHIP_LIMITATION")
	}
	if evidence.ForwardPaperDays == 0 || evidence.ForwardPaperOrders == 0 {
		codes = append(codes, "NO_FORWARD_PAPER_EVIDENCE")
	}
	checks := []struct {
		ok   bool
		code string
	}{{evidence.BaselineBeaten, "BASELINE_NOT_BEATEN"}, {evidence.Reproducible, "NOT_REPRODUCIBLE"}, {evidence.LeakageReviewPassed, "LEAKAGE_REVIEW_FAILED"}, {evidence.CostSensitivityPassed, "COST_SENSITIVITY_FAILED"}, {evidence.DriftMonitoringDefined, "DRIFT_MONITORING_MISSING"}, {evidence.FailureBehaviourDefined, "FAILURE_BEHAVIOUR_MISSING"}}
	for _, check := range checks {
		if !check.ok {
			codes = append(codes, check.code)
		}
	}
	if len(codes) == 0 {
		return []string{"SEPARATE_RECOMMENDATION_REVIEW_REQUIRED"}
	}
	return codes
}

func promotionID(r PromotionResult) string {
	r.ID = ""
	b, _ := json.Marshal(r)
	d := sha256.Sum256(b)
	return "promote_" + hex.EncodeToString(d[:])
}
