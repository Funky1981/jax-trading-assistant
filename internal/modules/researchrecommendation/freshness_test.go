package researchrecommendation

import (
	"testing"
	"time"
)

func TestFreshnessGatePassesRequiredFreshEvidence(t *testing.T) {
	packet, _ := researchFixture(t)
	decision, err := EvaluateFreshnessAndSufficiency(packet, FreshnessPolicy{AsOf: packetTime().Add(2 * time.Hour), Rules: []FreshnessRule{{Kind: EvidenceKindMarket, MaxAge: 24 * time.Hour, Required: true}, {Kind: EvidenceKindCompany, MaxAge: 48 * time.Hour, Required: true}}, MinimumEvidenceItems: 2, MinimumIndependentSources: 2})
	if err != nil || !decision.Sufficient || decision.State != FreshnessFresh {
		t.Fatalf("freshness decision=%+v err=%v", decision, err)
	}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFreshnessGateRejectsStaleUnknownAndMissingEvidence(t *testing.T) {
	packet, _ := researchFixture(t)
	packet.Items[0].Identity.Freshness = FreshnessStale
	packet.Items[1].Identity.Freshness = FreshnessUnknown
	var err error
	packet, err = NewEvidencePacket(packet.Subject, packet.Items, packet.PacketUnknowns, packet.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := EvaluateFreshnessAndSufficiency(packet, FreshnessPolicy{AsOf: packetTime().Add(2 * time.Hour), Rules: []FreshnessRule{{Kind: EvidenceKindMarket, MaxAge: time.Hour, Required: true}, {Kind: EvidenceKindCompany, MaxAge: time.Hour, Required: true}, {Kind: EvidenceKindMacro, MaxAge: time.Hour, Required: true}}, MinimumEvidenceItems: 3, MinimumIndependentSources: 3})
	if err != nil || decision.Sufficient || len(decision.StaleIDs) != 1 || len(decision.UnknownIDs) != 1 || len(decision.MissingKinds) != 1 {
		t.Fatalf("freshness fail-open: decision=%+v err=%v", decision, err)
	}
}
