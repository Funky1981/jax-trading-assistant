package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/researchrecommendation"
	"jax-trading-assistant/libs/contracts/canonical"
)

func TestReplayHistoricalCaseReconstructsExactPhase06Artifact(t *testing.T) {
	caseFile := replayFixture(t)
	result, err := ReplayHistoricalCase(caseFile)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != ReplayModeArtifact || !result.Reconstructed || result.CaseID != caseFile.ID || result.ContextPlanID != caseFile.ContextPlan.ID || result.ResearchOutputID != caseFile.Research.ID || result.RecommendationID != caseFile.Recommendation.ID || len(result.AvailableIDs) != len(caseFile.Packet.Items) {
		t.Fatalf("unexpected replay result: %+v", result)
	}
}

func TestReplayRejectsFutureEvidenceAndTamperedFrozenArtifacts(t *testing.T) {
	caseFile := replayFixture(t)
	caseFile.Knowledge[0].AvailableAt = caseFile.DecisionAt.Add(time.Minute)
	caseFile = reissuedCase(t, caseFile, caseFile.DecisionAt, caseFile.Knowledge)
	if _, err := ReplayHistoricalCase(caseFile); !errors.Is(err, ErrFutureEvidence) {
		t.Fatalf("future evidence was accepted: %v", err)
	}
	caseFile = replayFixture(t)
	caseFile.Research.Thesis = "hindsight changed thesis"
	if _, err := ReplayHistoricalCase(caseFile); err == nil {
		t.Fatal("tampered research artifact was accepted")
	}
}

func TestReplayRejectsFuturePublicationAndCollectionTimes(t *testing.T) {
	caseFile := replayFixture(t)
	caseFile = reissuedCase(t, caseFile, caseFile.DecisionAt.Add(-3*time.Hour), caseFile.Knowledge)
	if _, err := ReplayHistoricalCase(caseFile); !errors.Is(err, ErrFutureEvidence) {
		t.Fatalf("future publication/collection was accepted: %v", err)
	}
}

func reissuedCase(t *testing.T, original HistoricalCase, decisionAt time.Time, knowledge []EvidenceKnowledge) HistoricalCase {
	t.Helper()
	caseFile, err := NewHistoricalCase(original.CaseVersion, decisionAt, original.Task, original.Packet, knowledge, original.ContextPlan, original.Research, original.Recommendation)
	if err != nil {
		t.Fatal(err)
	}
	return caseFile
}

func replayFixture(t *testing.T) HistoricalCase {
	t.Helper()
	decisionAt := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	items := []researchrecommendation.EvidenceItem{
		replayItem("market", researchrecommendation.EvidenceKindMarket, "market", decisionAt),
		replayItem("company", researchrecommendation.EvidenceKindCompany, "company", decisionAt),
	}
	packet, err := researchrecommendation.NewEvidencePacket(replayInstrument(), items, []string{"future outcome is not known"}, decisionAt)
	if err != nil {
		t.Fatal(err)
	}
	task := researchrecommendation.ResearchTask{TaskID: "replay-task", Subject: packet.Subject.ID, Objective: "reconstruct historical research", RequiredOutput: "structured recommendation", Budget: replayBudget()}
	plan, err := researchrecommendation.BuildContext(task, packet)
	if err != nil {
		t.Fatal(err)
	}
	raw := `{"historical":"artifact"}`
	digest := sha256.Sum256([]byte(raw))
	research, err := researchrecommendation.NewStructuredResearchOutput(plan, packet, researchrecommendation.StructuredResearchOutput{
		ContractVersion: researchrecommendation.ResearchOutputContractV1, ContextPlanID: plan.ID, Subject: packet.Subject.ID, Thesis: "historical evidence supports watch", ThesisEvidenceIDs: []string{packet.Items[0].Identity.ID},
		BullCase: []researchrecommendation.ResearchClaim{{Statement: "market support", EvidenceIDs: []string{packet.Items[0].Identity.ID}}}, BearCase: []researchrecommendation.ResearchClaim{{Statement: "company uncertainty", EvidenceIDs: []string{packet.Items[1].Identity.ID}}}, Unknowns: []researchrecommendation.ResearchUnknown{{Statement: "outcome unknown"}}, InvalidationConditions: []researchrecommendation.InvalidationCondition{{Condition: "invalidate on reversal"}}, Inference: researchrecommendation.InferenceProvenance{Provider: "fixture", Model: "fixture", PromptVersion: "p1", SystemVersion: "s1", OutputContractVersion: researchrecommendation.ResearchOutputContractV1, RequestID: "replay-request", RawResponse: raw, RawResponseSHA256: hex.EncodeToString(digest[:]), CapturedAt: decisionAt, UsageComplete: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	eligibility, err := researchrecommendation.EvaluateEligibility(researchrecommendation.EligibilityInput{InstrumentResolved: true, ResearchValid: true, ContextComplete: true, EvidenceSufficient: true, FreshnessAcceptable: true, QuantContextComplete: true, ModelRouteAllowed: true, RequestedDisposition: researchrecommendation.DispositionWatch})
	if err != nil {
		t.Fatal(err)
	}
	confidence, err := researchrecommendation.NewConfidenceAssessment(nil, true, 2, 0, 1, []string{"not calibrated"})
	if err != nil {
		t.Fatal(err)
	}
	recommendation, err := researchrecommendation.BuildRecommendation(packet, plan, research, eligibility, confidence, decisionAt)
	if err != nil {
		t.Fatal(err)
	}
	knowledge := make([]EvidenceKnowledge, 0, len(packet.Items))
	for _, item := range packet.Items {
		knowledge = append(knowledge, EvidenceKnowledge{EvidenceID: item.Identity.ID, AvailableAt: decisionAt.Add(-time.Hour), Basis: "published and accepted before decision"})
	}
	caseFile, err := NewHistoricalCase("fixture-v1", decisionAt, task, packet, knowledge, plan, research, recommendation)
	if err != nil {
		t.Fatal(err)
	}
	return caseFile
}

func replayItem(id string, kind researchrecommendation.EvidenceKind, independence string, at time.Time) researchrecommendation.EvidenceItem {
	digest := sha256.Sum256([]byte(id))
	published := at.Add(-2 * time.Hour)
	return researchrecommendation.EvidenceItem{Identity: researchrecommendation.EvidenceIdentity{ID: "epi_" + hex.EncodeToString(digest[:]), Kind: kind, CanonicalRef: canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: "evd_" + id, ContractVersion: canonical.EvidenceContractV1}, IndependenceKey: independence, Freshness: researchrecommendation.FreshnessFresh, Source: researchrecommendation.SourceReference{SourceID: "src_" + id, Provider: "fixture", RawReference: "fixture/" + id, RawContentSHA256: strings.Repeat("a", 64), PublishedAt: &published, CollectedAt: at}}, Rendering: researchrecommendation.EvidenceRendering{Title: "Historical " + id, Summary: "Frozen historical evidence for " + id}}
}

func replayInstrument() canonical.ContractRef {
	return canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_replay", ContractVersion: canonical.InstrumentContractV1}
}

func replayBudget() researchrecommendation.ContextBudget {
	return researchrecommendation.ContextBudget{TargetInputTokens: 20, MaxInputTokens: 2000, MaxOutputTokens: 100, MaxReasoningTokens: 50, MaxEvidenceItems: 10, MaxChunks: 10, MaxRetries: 1, MaximumModelTier: "local-small", MaxEstimatedCostUSD: 1}
}
