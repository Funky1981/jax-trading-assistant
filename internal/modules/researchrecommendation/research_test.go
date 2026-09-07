package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestStructuredResearchOutputValidatesSuppliedEvidenceAndProvenance(t *testing.T) {
	packet, plan := researchFixture(t)
	raw := `{"thesis":"bounded thesis","bull_case":["a"],"bear_case":["b"]}`
	digest := sha256.Sum256([]byte(raw))
	output, err := NewStructuredResearchOutput(plan, packet, StructuredResearchOutput{
		ContractVersion: ResearchOutputContractV1, ContextPlanID: plan.ID, Subject: packet.Subject.ID, Thesis: "The supplied evidence supports a cautious view.", ThesisEvidenceIDs: []string{packet.Items[0].Identity.ID},
		BullCase:               []ResearchClaim{{Statement: "Market evidence supports the thesis.", EvidenceIDs: []string{packet.Items[0].Identity.ID}}},
		BearCase:               []ResearchClaim{{Statement: "Company evidence is a counterweight.", EvidenceIDs: []string{packet.Items[1].Identity.ID}}},
		Contradictions:         []ResearchClaim{{Statement: "Sources disagree on timing.", EvidenceIDs: []string{packet.Items[0].Identity.ID, packet.Items[1].Identity.ID}}},
		Unknowns:               []ResearchUnknown{{Statement: "The next filing outcome is unknown."}},
		InvalidationConditions: []InvalidationCondition{{Condition: "Invalidate if the supplied market context reverses."}},
		Inference:              InferenceProvenance{Provider: "deterministic-fixture", Model: "fixture-model", PromptVersion: "prompt-v1", SystemVersion: "system-v1", OutputContractVersion: ResearchOutputContractV1, RequestID: "req-fixture", RawResponse: raw, RawResponseSHA256: hex.EncodeToString(digest[:]), CapturedAt: packetTime(), InputTokens: 10, OutputTokens: 8, ActualCostUSD: 0, UsageComplete: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(output.ID, "rso_") || output.Inference.RawResponseSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("output identity/provenance missing: %+v", output)
	}
}

func TestStructuredResearchOutputRejectsFabricatedEvidenceAndIncompleteArtifacts(t *testing.T) {
	packet, plan := researchFixture(t)
	output := validResearchOutput(t, packet, plan)
	output.BullCase[0].EvidenceIDs = []string{"epi_" + strings.Repeat("f", 64)}
	if err := ValidateStructuredResearchOutput(plan, packet, output); err == nil {
		t.Fatal("fabricated evidence reference accepted")
	}
	output = validResearchOutput(t, packet, plan)
	output.Inference.RawResponse = "changed without changing hash"
	if err := ValidateStructuredResearchOutput(plan, packet, output); err == nil {
		t.Fatal("tampered raw inference artifact accepted")
	}
	output = validResearchOutput(t, packet, plan)
	output.ID = "rso_" + strings.Repeat("f", 64)
	if err := ValidateStructuredResearchOutput(plan, packet, output); err == nil {
		t.Fatal("unbound research output identity accepted")
	}
	output = validResearchOutput(t, packet, plan)
	output.BullCase[0].Statement = "tampered claim"
	if err := ValidateStructuredResearchOutput(plan, packet, output); err == nil {
		t.Fatal("tampered structured claim accepted")
	}
}

func researchFixture(t *testing.T) (EvidencePacket, ContextPlan) {
	t.Helper()
	items := []EvidenceItem{packetItem("market", EvidenceKindMarket, "market"), packetItem("company", EvidenceKindCompany, "company")}
	packet, err := NewEvidencePacket(instrumentRef(), items, []string{"issuer guidance remains unresolved"}, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildContext(ResearchTask{TaskID: "research-task", Subject: "JAX", Objective: "assess", RequiredOutput: "structured research", Budget: testBudget()}, packet)
	if err != nil {
		t.Fatal(err)
	}
	return packet, plan
}

func validResearchOutput(t *testing.T, packet EvidencePacket, plan ContextPlan) StructuredResearchOutput {
	t.Helper()
	raw := "fixture raw response"
	digest := sha256.Sum256([]byte(raw))
	output, err := NewStructuredResearchOutput(plan, packet, StructuredResearchOutput{ContractVersion: ResearchOutputContractV1, ContextPlanID: plan.ID, Subject: packet.Subject.ID, Thesis: "thesis", ThesisEvidenceIDs: []string{packet.Items[0].Identity.ID}, BullCase: []ResearchClaim{{Statement: "bull", EvidenceIDs: []string{packet.Items[0].Identity.ID}}}, BearCase: []ResearchClaim{{Statement: "bear", EvidenceIDs: []string{packet.Items[1].Identity.ID}}}, Unknowns: []ResearchUnknown{{Statement: "unknown"}}, InvalidationConditions: []InvalidationCondition{{Condition: "condition"}}, Inference: InferenceProvenance{Provider: "fixture", Model: "fixture", PromptVersion: "p1", SystemVersion: "s1", OutputContractVersion: ResearchOutputContractV1, RequestID: "req", RawResponse: raw, RawResponseSHA256: hex.EncodeToString(digest[:]), CapturedAt: packetTime(), UsageComplete: true}})
	if err != nil {
		t.Fatal(err)
	}
	return output
}
