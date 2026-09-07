package researchrecommendation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRecommendationReadModelIsStableAndResearchOnly(t *testing.T) {
	packet, plan := researchFixture(t)
	research := validResearchOutput(t, packet, plan)
	eligibility, err := EvaluateEligibility(EligibilityInput{
		InstrumentResolved: true, ResearchValid: true, ContextComplete: true,
		EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true,
		ModelRouteAllowed: true, RequestedDisposition: DispositionWatch,
	})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := NewConfidenceAssessment(nil, true, 2, 1, 1, []string{"model score is not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := BuildRecommendation(packet, plan, research, eligibility, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	freshness, err := EvaluateFreshnessAndSufficiency(packet, FreshnessPolicy{
		AsOf: packetTime(), Rules: []FreshnessRule{
			{Kind: EvidenceKindMarket, MaxAge: 24 * time.Hour, Required: true},
			{Kind: EvidenceKindCompany, MaxAge: 24 * time.Hour, Required: true},
		}, MinimumEvidenceItems: 2, MinimumIndependentSources: 2,
	})
	if err != nil || !freshness.Sufficient {
		t.Fatalf("freshness=%+v err=%v", freshness, err)
	}
	model, err := BuildRecommendationReadModel(packet, plan, research, recommendation, freshness)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Validate(); err != nil {
		t.Fatal(err)
	}
	second, err := BuildRecommendationReadModel(packet, plan, research, recommendation, freshness)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(model)
	right, _ := json.Marshal(second)
	if string(left) != string(right) || model.ID != second.ID {
		t.Fatal("read model is not reproducible")
	}
	if strings.Contains(string(left), "raw_response\"") || model.ExecutionAuthority != "NONE" || model.Authority != "RESEARCH_DECISION_SUPPORT" {
		t.Fatal("read model exposed inference payload or execution authority")
	}
	if len(model.Evidence) != len(packet.Items) || len(model.ContextSelectedIDs) != len(plan.SelectedIDs) {
		t.Fatalf("read model omitted evidence/context: %+v", model)
	}
}

func TestRecommendationReadModelRejectsTamperedPacketFingerprint(t *testing.T) {
	packet, plan := researchFixture(t)
	research := validResearchOutput(t, packet, plan)
	eligibility, err := EvaluateEligibility(EligibilityInput{
		InstrumentResolved: true, ResearchValid: true, ContextComplete: true,
		EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true,
		ModelRouteAllowed: true, RequestedDisposition: DispositionNoTrade,
	})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := NewConfidenceAssessment(nil, true, 2, 1, 1, []string{"not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := BuildRecommendation(packet, plan, research, eligibility, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	freshness, err := EvaluateFreshnessAndSufficiency(packet, FreshnessPolicy{
		AsOf: packetTime(), Rules: []FreshnessRule{{Kind: EvidenceKindMarket, MaxAge: 24 * time.Hour}, {Kind: EvidenceKindCompany, MaxAge: 24 * time.Hour}},
		MinimumEvidenceItems: 2, MinimumIndependentSources: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan.PacketContentFingerprint = strings.Repeat("a", 64)
	if _, err := BuildRecommendationReadModel(packet, plan, research, recommendation, freshness); err == nil {
		t.Fatal("tampered packet fingerprint accepted")
	}
}

func TestRecommendationReadModelRejectsTamperedIdentity(t *testing.T) {
	packet, plan := researchFixture(t)
	research := validResearchOutput(t, packet, plan)
	eligibility, err := EvaluateEligibility(EligibilityInput{
		InstrumentResolved: true, ResearchValid: true, ContextComplete: true,
		EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true,
		ModelRouteAllowed: true, RequestedDisposition: DispositionWatch,
	})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := NewConfidenceAssessment(nil, true, 2, 1, 1, []string{"not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := BuildRecommendation(packet, plan, research, eligibility, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	freshness, err := EvaluateFreshnessAndSufficiency(packet, FreshnessPolicy{
		AsOf: packetTime(), Rules: []FreshnessRule{{Kind: EvidenceKindMarket, MaxAge: 24 * time.Hour}, {Kind: EvidenceKindCompany, MaxAge: 24 * time.Hour}},
		MinimumEvidenceItems: 2, MinimumIndependentSources: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	model, err := BuildRecommendationReadModel(packet, plan, research, recommendation, freshness)
	if err != nil {
		t.Fatal(err)
	}
	model.Thesis = "tampered"
	if err := model.Validate(); err == nil {
		t.Fatal("tampered read model identity accepted")
	}
}

func TestBuildRecommendationRejectsNonUTCTimestamp(t *testing.T) {
	packet, plan := researchFixture(t)
	research := validResearchOutput(t, packet, plan)
	eligibility, err := EvaluateEligibility(EligibilityInput{
		InstrumentResolved: true, ResearchValid: true, ContextComplete: true,
		EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true,
		ModelRouteAllowed: true, RequestedDisposition: DispositionWatch,
	})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := NewConfidenceAssessment(nil, true, 2, 1, 1, []string{"not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildRecommendation(packet, plan, research, eligibility, confidence, time.Date(2026, 9, 7, 13, 0, 0, 0, time.FixedZone("BST", 3600))); err == nil {
		t.Fatal("non-UTC recommendation timestamp was silently normalized")
	}
}
