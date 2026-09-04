package phase03gate

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

var packetNow = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

func packetIssuer() canonical.Issuer {
	return canonical.Issuer{ContractVersion: canonical.IssuerContractV1, ID: "iss_apple", Type: canonical.IssuerTypeCorporate, Name: "Apple Inc.", Jurisdiction: "US", ExternalIDs: []canonical.ExternalID{{Namespace: "sec.cik", Value: "0000320193"}}, CreatedAt: packetNow}
}

func packetInstrument() canonical.Instrument {
	return canonical.Instrument{ContractVersion: canonical.InstrumentContractV1, ID: "ins_aapl_common", Type: canonical.InstrumentTypeEquity, Name: "Apple Inc. common stock", Currency: "USD", ExternalIDs: []canonical.ExternalID{{Namespace: "ticker.xnas", Value: "AAPL"}}, Issuers: []canonical.InstrumentIssuer{{IssuerID: "iss_apple", Role: canonical.InstrumentIssuerRoleIssuer}}, CreatedAt: packetNow}
}

func packetRaw(id providercontract.RawPayloadID, provider canonical.ProviderIdentity, source canonical.SourceIdentity) providercontract.RawPayloadRef {
	bytes := []byte(string(id) + " exact source bytes")
	revision := canonical.RevisionIdentity{Namespace: "source.response_sha256", Value: canonical.DigestBytes(bytes).Value}
	return providercontract.RawPayloadRef{ContractVersion: providercontract.RawPayloadRefContractV1, ID: id, Content: canonical.RawContentIdentity(bytes), Provider: provider, CapabilityID: providercontract.CapabilityMacroObservation, Raw: providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: "test.source", Value: "v1"}, MediaType: "application/json"}, Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity, CharacterEncoding: "utf-8"}, Source: &source, Revision: &revision, ReceivedAt: packetNow, SizeBytes: int64(len(bytes)), Retention: providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionRestricted}}
}

func packetItem(t *testing.T, family EvidenceFamily, origin EvidenceOrigin, id string) EvidenceItem {
	t.Helper()
	provider := canonical.ProviderIdentity{ID: "pvd_test", Namespace: "test.provider", ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "test"}}
	source := canonical.SourceIdentity{ID: "src_test", Kind: canonical.SourceKindPublisher}
	raw := packetRaw(providercontract.RawPayloadID("rpa_"+id), provider, source)
	evidenceRef, err := raw.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: "evd_" + id, ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		t.Fatal(err)
	}
	lineage := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}
	fingerprint, err := canonical.ComputeInputFingerprint([]canonical.LineageInput{lineage})
	if err != nil {
		t.Fatal(err)
	}
	provenance := canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + strings.Repeat("a", 24), Inputs: []canonical.LineageInput{lineage}, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: "cmp_phase03_gate_test", Kind: canonical.ComponentKindSoftwareBuild, Name: "Phase 03 gate test builder", Version: canonical.VersionIdentity{Namespace: "test", Value: "v1"}}}
	return EvidenceItem{Family: family, Origin: origin, NormalizedRef: canonical.ContractRef{Kind: canonical.ContractKindObservation, ID: "obs_" + id, ContractVersion: canonical.ObservationContractV2}, RawPayload: raw, Provenance: provenance, Temporal: TemporalMetadata{ObservationDate: "2026-09-03", AcquiredAt: packetNow}}
}

func completePacket(t *testing.T, origin EvidenceOrigin) Packet {
	t.Helper()
	items := []EvidenceItem{packetItem(t, FamilyMarket, origin, "market"), packetItem(t, FamilyCompany, origin, "company"), packetItem(t, FamilyMacroContext, origin, "macro")}
	packet, err := NewPacket(packetInstrument(), packetIssuer(), items, []string{"eqc_fixture_diagnostic"}, packetNow)
	if err != nil {
		t.Fatal(err)
	}
	return packet
}

func TestNewPacketStableIDAndCanonicalIdentity(t *testing.T) {
	first := completePacket(t, OriginPreviouslyPersistedReal)
	second := completePacket(t, OriginPreviouslyPersistedReal)
	if first.ID != second.ID {
		t.Fatalf("packet ID changed for identical semantic inputs: %q != %q", first.ID, second.ID)
	}
	if first.Instrument.ID != "ins_aapl_common" || first.Issuer.ID != "iss_apple" {
		t.Fatalf("unexpected identity: %s / %s", first.Instrument.ID, first.Issuer.ID)
	}
}

func TestAssertRealExitConditionFailsClosedForFixtures(t *testing.T) {
	packet := completePacket(t, OriginSyntheticFixture)
	if err := packet.AssertRealExitCondition(); err == nil || !strings.Contains(err.Error(), "real") {
		t.Fatalf("fixture packet was accepted: %v", err)
	}
}

func TestAssertRealExitConditionRequiresAllFamilies(t *testing.T) {
	items := []EvidenceItem{packetItem(t, FamilyMarket, OriginLiveAcquiredNow, "market"), packetItem(t, FamilyCompany, OriginPreviouslyPersistedReal, "company")}
	packet, err := NewPacket(packetInstrument(), packetIssuer(), items, nil, packetNow)
	if err != nil {
		t.Fatal(err)
	}
	if err := packet.AssertRealExitCondition(); err == nil || !strings.Contains(err.Error(), "MACRO_CONTEXT") {
		t.Fatalf("missing macro evidence was not rejected: %v", err)
	}
}

func TestEvidenceItemRequiresRawSourceProvenance(t *testing.T) {
	item := packetItem(t, FamilyMarket, OriginLiveAcquiredNow, "market")
	item.RawPayload.Source = nil
	if err := item.Validate(); err == nil || !strings.Contains(err.Error(), "source identity") {
		t.Fatalf("missing raw source was accepted: %v", err)
	}
}

func TestPacketValidationFailsClosedForTamperedID(t *testing.T) {
	packet := completePacket(t, OriginPreviouslyPersistedReal)
	packet.ID = "p03_tampered"
	if err := packet.Validate(); err == nil || !strings.Contains(err.Error(), "deterministic") {
		t.Fatalf("tampered packet ID was accepted: %v", err)
	}
}
