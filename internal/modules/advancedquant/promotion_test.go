package advancedquant

import "testing"

func TestPromotionGateRemainsClosedForHYPEvent001A(t *testing.T) {
	result, err := EvaluatePromotion(PromotionEvidence{HypothesisID: "HYP-EVENT-001A", ExperimentID: "exp", DatasetID: "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1", DatasetHash: "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d", BaselineBeaten: true, Reproducible: true, LeakageReviewPassed: true, CostSensitivityPassed: true, DriftMonitoringDefined: true, FailureBehaviourDefined: true, FinalHoldoutSealed: true, ForwardPaperDays: 0, ForwardPaperOrders: 0})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != PromotionClosed || result.RecommendationMutationAllowed {
		t.Fatalf("promotion was not closed: %+v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	if !containsAll(result.ReasonCodes, []string{"FINAL_HOLDOUT_SEALED", "SURVIVORSHIP_LIMITATION", "NO_FORWARD_PAPER_EVIDENCE"}) {
		t.Fatalf("required closure reasons missing: %v", result.ReasonCodes)
	}
}

func TestPromotionGateFailsClosedOnMissingEvidenceAndDetectsTampering(t *testing.T) {
	if _, err := EvaluatePromotion(PromotionEvidence{HypothesisID: "HYP-EVENT-001A", DatasetHash: "bad"}); err == nil {
		t.Fatal("incomplete evidence was accepted")
	}
	result, err := EvaluatePromotion(PromotionEvidence{HypothesisID: "HYP", ExperimentID: "exp", DatasetID: "data", DatasetHash: "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d", BaselineBeaten: true, Reproducible: true, LeakageReviewPassed: true, CostSensitivityPassed: true, DriftMonitoringDefined: true, FailureBehaviourDefined: true, SurvivorshipControlled: true, FinalHoldoutSealed: false, ForwardPaperDays: 10, ForwardPaperOrders: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != PromotionEligible || result.RecommendationMutationAllowed {
		t.Fatalf("eligible result crossed authority boundary: %+v", result)
	}
	result.Status = PromotionClosed
	if err := result.Validate(); err == nil {
		t.Fatal("tampered promotion result was accepted")
	}
}

func containsAll(have, want []string) bool {
	set := map[string]bool{}
	for _, value := range have {
		set[value] = true
	}
	for _, value := range want {
		if !set[value] {
			return false
		}
	}
	return true
}
