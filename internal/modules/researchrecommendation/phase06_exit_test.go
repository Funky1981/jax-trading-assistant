package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPhase06ExitProducesReproducibleEvidenceLinkedResearchOnlyOutput(t *testing.T) {
	packet := phase06ExitPacket(t)
	task := ResearchTask{TaskID: "phase06-exit", Subject: packet.Subject.ID, Objective: "assess the evidence", RequiredOutput: "research recommendation", Budget: testBudget(), RequiredIDs: evidenceIDs(packet)}
	plan, err := BuildContext(task, packet)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Oversize || len(plan.SelectedIDs) != len(packet.Items) || len(plan.OmittedIDs) != 0 {
		t.Fatalf("exit context is incomplete: %+v", plan)
	}

	raw := `{"structured_research":"phase-06-fixture"}`
	digest := sha256.Sum256([]byte(raw))
	base := StructuredResearchOutput{
		ContractVersion: ResearchOutputContractV1, ContextPlanID: plan.ID, Subject: packet.Subject.ID,
		Thesis:                 "Evidence supports a cautious watch while cross-source timing remains unresolved.",
		ThesisEvidenceIDs:      []string{itemID(packet, EvidenceKindMarket)},
		BullCase:               []ResearchClaim{{Statement: "Company evidence supports improving fundamentals.", EvidenceIDs: []string{itemID(packet, EvidenceKindCompany)}}},
		BearCase:               []ResearchClaim{{Statement: "Macro conditions remain a counterweight.", EvidenceIDs: []string{itemID(packet, EvidenceKindMacro)}}},
		Contradictions:         []ResearchClaim{{Statement: "Market and world-monitor timing signals diverge.", EvidenceIDs: []string{itemID(packet, EvidenceKindMarket), itemID(packet, EvidenceKindWorldMonitor)}}},
		Unknowns:               []ResearchUnknown{{Statement: "The next catalyst outcome is unknown."}},
		InvalidationConditions: []InvalidationCondition{{Condition: "Invalidate if the quantified trend reverses.", EvidenceIDs: []string{itemID(packet, EvidenceKindQuant)}}},
		Inference:              InferenceProvenance{Provider: "deterministic-fixture", Model: "fixture-model", PromptVersion: "phase06-prompt-v1", SystemVersion: "phase06-system-v1", OutputContractVersion: ResearchOutputContractV1, RequestID: "phase06-request", RawResponse: raw, RawResponseSHA256: hex.EncodeToString(digest[:]), CapturedAt: packetTime(), UsageComplete: true},
	}
	base, err = NewStructuredResearchOutput(plan, packet, base)
	if err != nil {
		t.Fatal(err)
	}
	research, stats, err := ExecuteStructuredResearch(plan, packet, &fixtureResearchProvider{outputs: []StructuredResearchOutput{base}})
	if err != nil || stats.Attempts != 1 {
		t.Fatalf("structured research execution failed: stats=%+v err=%v", stats, err)
	}
	freshness, err := EvaluateFreshnessAndSufficiency(packet, phase06FreshnessPolicy())
	if err != nil || !freshness.Sufficient || freshness.IndependentSourceCount != len(packet.Items) {
		t.Fatalf("exit freshness is not sufficient: decision=%+v err=%v", freshness, err)
	}
	eligibility, err := EvaluateEligibility(EligibilityInput{InstrumentResolved: true, ResearchValid: true, ContextComplete: true, EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true, ModelRouteAllowed: true, RequestedDisposition: DispositionWatch})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := NewConfidenceAssessment(nil, true, freshness.IndependentSourceCount, len(research.Contradictions), len(research.Unknowns), []string{"model score is not calibrated", "catalyst outcome remains unknown"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := BuildRecommendation(packet, plan, research, eligibility, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	model, err := BuildRecommendationReadModel(packet, plan, research, recommendation, freshness)
	if err != nil {
		t.Fatal(err)
	}
	if model.Disposition != DispositionWatch || len(model.BullCase) != 1 || len(model.CounterEvidence) != 1 || len(model.Contradictions) != 1 || len(model.Unknowns) != 1 || len(model.InvalidationConditions) != 1 || len(model.Evidence) != 6 || len(model.QuantContext) != 1 || model.QuantContext[0].Value != 1.25 {
		t.Fatalf("exit output omitted required research surface: %+v", model)
	}
	serialized, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	secondPlan, err := BuildContext(task, packet)
	if err != nil {
		t.Fatal(err)
	}
	secondResearch, err := NewStructuredResearchOutput(secondPlan, packet, base)
	if err != nil {
		t.Fatal(err)
	}
	secondRecommendation, err := BuildRecommendation(packet, secondPlan, secondResearch, eligibility, confidence, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	secondModel, err := BuildRecommendationReadModel(packet, secondPlan, secondResearch, secondRecommendation, freshness)
	if err != nil {
		t.Fatal(err)
	}
	serializedAgain, err := json.Marshal(secondModel)
	if err != nil {
		t.Fatal(err)
	}
	if string(serialized) != string(serializedAgain) || plan.ID != secondPlan.ID || model.ID != secondModel.ID {
		t.Fatal("phase exit output is not reproducible")
	}
	lower := strings.ToLower(string(serialized))
	if strings.Contains(lower, "raw_response\"") || strings.Contains(lower, "approval") || strings.Contains(lower, "order") || strings.Contains(lower, "trade") || strings.Contains(lower, "fill") || model.Authority != "RESEARCH_DECISION_SUPPORT" || model.ExecutionAuthority != "NONE" {
		t.Fatal("research output crossed the execution/approval boundary")
	}
}

func phase06ExitPacket(t *testing.T) EvidencePacket {
	t.Helper()
	kinds := []EvidenceKind{EvidenceKindInstrument, EvidenceKindCompany, EvidenceKindMarket, EvidenceKindMacro, EvidenceKindWorldMonitor, EvidenceKindQuant}
	items := make([]EvidenceItem, 0, len(kinds))
	for _, kind := range kinds {
		item := packetItem(strings.ToLower(string(kind)), kind, "phase06-"+strings.ToLower(string(kind)))
		if kind == EvidenceKindQuant {
			item.Identity.QuantResult = &QuantResultReference{ResultID: "qres_phase06", Algorithm: "phase05_fixture", AlgorithmVersion: "v1", FrozenInputSHA256: strings.Repeat("b", 64), ResultContentSHA256: strings.Repeat("c", 64)}
			item.Rendering.NumericContext = []NumericValue{{Metric: "trend_score", Value: 1.25, Unit: "score"}}
		}
		items = append(items, item)
	}
	packet, err := NewEvidencePacket(instrumentRef(), items, []string{"catalyst timing is unresolved"}, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func evidenceIDs(packet EvidencePacket) []string {
	ids := make([]string, 0, len(packet.Items))
	for _, item := range packet.Items {
		ids = append(ids, item.Identity.ID)
	}
	return ids
}

func itemID(packet EvidencePacket, kind EvidenceKind) string {
	for _, item := range packet.Items {
		if item.Identity.Kind == kind {
			return item.Identity.ID
		}
	}
	return ""
}

func phase06FreshnessPolicy() FreshnessPolicy {
	rules := make([]FreshnessRule, 0, 6)
	for _, kind := range []EvidenceKind{EvidenceKindInstrument, EvidenceKindCompany, EvidenceKindMarket, EvidenceKindMacro, EvidenceKindWorldMonitor, EvidenceKindQuant} {
		rules = append(rules, FreshnessRule{Kind: kind, MaxAge: 24 * time.Hour, Required: true})
	}
	return FreshnessPolicy{AsOf: packetTime(), Rules: rules, MinimumEvidenceItems: 6, MinimumIndependentSources: 6}
}
