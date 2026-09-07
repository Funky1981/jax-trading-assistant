package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

func TestEvidencePacketSeparatesIdentityFromRendering(t *testing.T) {
	item := packetItem("market", EvidenceKindMarket, "source-market")
	packet, err := NewEvidencePacket(canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_jax", ContractVersion: canonical.InstrumentContractV1}, []EvidenceItem{item}, []string{"unresolved issuer alias"}, packetTime())
	if err != nil {
		t.Fatal(err)
	}
	originalID, err := packet.ContentFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	packet.Items[0].Rendering.Summary = "A changed bounded rendering"
	if err := packet.Validate(); err != nil {
		t.Fatal(err)
	}
	if packet.ID != packet.derivedID() {
		t.Fatal("source identity changed when rendering changed")
	}
	changedID, err := packet.ContentFingerprint()
	if err != nil || originalID == changedID {
		t.Fatal("rendering change was not distinguishable in content fingerprint")
	}
}

func TestEvidencePacketRejectsAmbiguousOrMalformedEvidence(t *testing.T) {
	item := packetItem("market", EvidenceKindMarket, "source-market")
	item.Identity.Source.RawContentSHA256 = strings.Repeat("g", 64)
	if _, err := NewEvidencePacket(canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_jax", ContractVersion: canonical.InstrumentContractV1}, []EvidenceItem{item}, nil, packetTime()); err == nil {
		t.Fatal("malformed source fingerprint accepted")
	}
	item = packetItem("market", EvidenceKindMarket, "same-source")
	duplicate := item
	if _, err := NewEvidencePacket(canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_jax", ContractVersion: canonical.InstrumentContractV1}, []EvidenceItem{item, duplicate}, nil, packetTime()); err != nil {
		return
	}
	t.Fatal("duplicate evidence identity accepted")
}

func packetItem(id string, kind EvidenceKind, independence string) EvidenceItem {
	digest := sha256.Sum256([]byte(id))
	return EvidenceItem{
		Identity:  EvidenceIdentity{ID: "epi_" + hex.EncodeToString(digest[:]), Kind: kind, CanonicalRef: canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: "evd_" + id, ContractVersion: canonical.EvidenceContractV1}, IndependenceKey: independence, Freshness: FreshnessFresh, Source: SourceReference{SourceID: "src_" + id, Provider: "provider-" + id, RawReference: "raw/" + id, RawContentSHA256: strings.Repeat("a", 64), PublishedAt: timePtr(packetTime().Add(-time.Hour)), CollectedAt: packetTime()}},
		Rendering: EvidenceRendering{Title: "Evidence " + id, Summary: "Bounded evidence summary " + id, Excerpt: "untrusted source content"},
	}
}

func packetTime() time.Time              { return time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC) }
func timePtr(value time.Time) *time.Time { return &value }
