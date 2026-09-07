package researchrecommendation

import (
	"testing"
)

func TestEligibilityProducesResearchOnlyCandidateClassification(t *testing.T) {
	decision, err := EvaluateEligibility(EligibilityInput{InstrumentResolved: true, ResearchValid: true, ContextComplete: true, EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true, ModelRouteAllowed: true, RequestedDisposition: DispositionCandidate})
	if err != nil || !decision.Eligible || decision.Disposition != DispositionCandidate {
		t.Fatalf("eligibility=%+v err=%v", decision, err)
	}
	packet, plan := researchFixture(t)
	research := validResearchOutput(t, packet, plan)
	confidence, err := NewConfidenceAssessment(nil, true, 2, 0, 1, []string{"model score is not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := BuildRecommendation(packet, plan, research, decision, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	if recommendation.Disposition != DispositionCandidate || recommendation.ExecutionAuthority != "NONE" || recommendation.Authority != "RESEARCH_DECISION_SUPPORT" {
		t.Fatalf("unsafe recommendation grammar: %+v", recommendation)
	}
}

func TestEligibilityFailsClosedToNoTrade(t *testing.T) {
	decision, err := EvaluateEligibility(EligibilityInput{InstrumentResolved: true, ResearchValid: true, ContextComplete: false, EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true, ModelRouteAllowed: true, RequestedDisposition: DispositionWatch})
	if err != nil || decision.Eligible || decision.Disposition != DispositionNoTrade || len(decision.ReasonCodes) != 1 || decision.ReasonCodes[0] != "context_incomplete" {
		t.Fatalf("failed-open eligibility=%+v err=%v", decision, err)
	}
}

func TestEligibilityRejectsUnknownDisposition(t *testing.T) {
	if _, err := EvaluateEligibility(EligibilityInput{RequestedDisposition: RecommendationDisposition("BUY")}); err == nil {
		t.Fatal("unknown recommendation disposition accepted")
	}
}
