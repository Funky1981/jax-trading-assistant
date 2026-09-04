// Package phase03gate contains the smallest provider-neutral demonstration
// contract for the Phase 03 exit gate. It is not the later research packet.
package phase03gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

const PacketContractV1 canonical.ContractVersion = "jax.phase03_exit_packet/v1"

type EvidenceOrigin string

const (
	OriginLiveAcquiredNow         EvidenceOrigin = "LIVE_ACQUIRED_NOW"
	OriginPreviouslyPersistedReal EvidenceOrigin = "PREVIOUSLY_PERSISTED_REAL_SOURCE_EVIDENCE"
	OriginSyntheticFixture        EvidenceOrigin = "SYNTHETIC_FIXTURE"
)

func (origin EvidenceOrigin) IsReal() bool {
	return origin == OriginLiveAcquiredNow || origin == OriginPreviouslyPersistedReal
}

type EvidenceFamily string

const (
	FamilyMarket       EvidenceFamily = "MARKET"
	FamilyCompany      EvidenceFamily = "COMPANY"
	FamilyMacroContext EvidenceFamily = "MACRO_CONTEXT"
	FamilyRelease      EvidenceFamily = "RELEASE_CALENDAR"
)

func (family EvidenceFamily) Validate() error {
	switch family {
	case FamilyMarket, FamilyCompany, FamilyMacroContext, FamilyRelease:
		return nil
	default:
		return fmt.Errorf("unsupported evidence family %q", family)
	}
}

// TemporalMetadata records boundaries without treating one as another. The
// fields are deliberately strings/time values rather than an inferred event
// timestamp; adapters remain responsible for their source-specific semantics.
type TemporalMetadata struct {
	ObservationDate        string     `json:"observation_date,omitempty"`
	ReportDate             string     `json:"report_date,omitempty"`
	FilingDate             string     `json:"filing_date,omitempty"`
	AcceptanceDateTime     *time.Time `json:"acceptance_datetime,omitempty"`
	PublicationTime        *time.Time `json:"publication_time,omitempty"`
	PublicAvailabilityTime *time.Time `json:"public_availability_time,omitempty"`
	AcquiredAt             time.Time  `json:"acquired_at"`
}

func (temporal TemporalMetadata) Validate() error {
	for name, value := range map[string]string{
		"observation_date": temporal.ObservationDate,
		"report_date":      temporal.ReportDate,
		"filing_date":      temporal.FilingDate,
	} {
		if value != "" {
			if len(value) != len("2006-01-02") {
				return fmt.Errorf("%s must use YYYY-MM-DD", name)
			}
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil || parsed.Format("2006-01-02") != value {
				return fmt.Errorf("%s is invalid", name)
			}
		}
	}
	if temporal.AcquiredAt.IsZero() || temporal.AcquiredAt.Location() != time.UTC {
		return errors.New("acquired_at must be a non-zero UTC timestamp")
	}
	for name, value := range map[string]*time.Time{
		"acceptance_datetime":      temporal.AcceptanceDateTime,
		"publication_time":         temporal.PublicationTime,
		"public_availability_time": temporal.PublicAvailabilityTime,
	} {
		if value != nil && (value.IsZero() || value.Location() != time.UTC || value.After(temporal.AcquiredAt)) {
			return fmt.Errorf("%s must be a UTC timestamp no later than acquired_at", name)
		}
	}
	return nil
}

// EvidenceItem links one accepted normalized record to the exact raw
// acquisition and the normalized record's immutable provenance. No provider
// DTO is embedded here.
type EvidenceItem struct {
	Family        EvidenceFamily                 `json:"family"`
	Origin        EvidenceOrigin                 `json:"origin"`
	NormalizedRef canonical.ContractRef          `json:"normalized_ref"`
	RawPayload    providercontract.RawPayloadRef `json:"raw_payload"`
	Provenance    canonical.Provenance           `json:"provenance"`
	Temporal      TemporalMetadata               `json:"temporal"`
}

func (item EvidenceItem) Validate() error {
	if err := item.Family.Validate(); err != nil {
		return err
	}
	switch item.Origin {
	case OriginLiveAcquiredNow, OriginPreviouslyPersistedReal, OriginSyntheticFixture:
	default:
		return fmt.Errorf("unsupported evidence origin %q", item.Origin)
	}
	if item.NormalizedRef.ID == "" || item.NormalizedRef.ContractVersion == "" {
		return errors.New("normalized evidence reference is required")
	}
	switch item.NormalizedRef.Kind {
	case canonical.ContractKindEvidence:
		if item.NormalizedRef.ContractVersion != canonical.EvidenceContractV1 && item.NormalizedRef.ContractVersion != canonical.EvidenceContractV2 {
			return errors.New("normalized evidence reference has an unsupported evidence version")
		}
		if !strings.HasPrefix(item.NormalizedRef.ID, "evd_") {
			return errors.New("normalized evidence reference has an invalid ID")
		}
	case canonical.ContractKindObservation:
		if item.NormalizedRef.ContractVersion != canonical.ObservationContractV1 && item.NormalizedRef.ContractVersion != canonical.ObservationContractV2 {
			return errors.New("normalized evidence reference has an unsupported observation version")
		}
		if !strings.HasPrefix(item.NormalizedRef.ID, "obs_") {
			return errors.New("normalized evidence reference has an invalid ID")
		}
	default:
		return errors.New("normalized reference must identify canonical evidence or observation")
	}
	if err := item.RawPayload.Validate(); err != nil {
		return fmt.Errorf("raw payload: %w", err)
	}
	if item.RawPayload.Source == nil {
		return errors.New("raw payload source identity is required")
	}
	if err := item.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if err := item.Temporal.Validate(); err != nil {
		return err
	}
	if !provenanceContainsRaw(item.Provenance, item.RawPayload) {
		return errors.New("provenance does not cover the item's raw acquisition")
	}
	return nil
}

type Packet struct {
	ContractVersion canonical.ContractVersion `json:"contract_version"`
	ID              string                    `json:"id"`
	Instrument      canonical.Instrument      `json:"instrument"`
	Issuer          canonical.Issuer          `json:"issuer"`
	Items           []EvidenceItem            `json:"items"`
	DiagnosticIDs   []string                  `json:"diagnostic_ids,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
}

// NewPacket sorts evidence items and derives the ID from stable semantic
// inputs. Acquisition wall-clock values are intentionally excluded from the
// ID so replaying the same source identities remains deterministic.
func NewPacket(instrument canonical.Instrument, issuer canonical.Issuer, items []EvidenceItem, diagnosticIDs []string, createdAt time.Time) (Packet, error) {
	packet := Packet{ContractVersion: PacketContractV1, Instrument: instrument, Issuer: issuer, Items: append([]EvidenceItem(nil), items...), DiagnosticIDs: append([]string(nil), diagnosticIDs...), CreatedAt: createdAt}
	sort.Slice(packet.Items, func(i, j int) bool { return itemKey(packet.Items[i]) < itemKey(packet.Items[j]) })
	sort.Strings(packet.DiagnosticIDs)
	seedItems := make([]struct {
		Family        EvidenceFamily                `json:"family"`
		Origin        EvidenceOrigin                `json:"origin"`
		NormalizedRef canonical.ContractRef         `json:"normalized_ref"`
		RawID         providercontract.RawPayloadID `json:"raw_id"`
		Digest        string                        `json:"digest"`
	}, 0, len(packet.Items))
	for _, item := range packet.Items {
		seedItems = append(seedItems, struct {
			Family        EvidenceFamily                `json:"family"`
			Origin        EvidenceOrigin                `json:"origin"`
			NormalizedRef canonical.ContractRef         `json:"normalized_ref"`
			RawID         providercontract.RawPayloadID `json:"raw_id"`
			Digest        string                        `json:"digest"`
		}{item.Family, item.Origin, item.NormalizedRef, item.RawPayload.ID, item.RawPayload.Content.Digest.Value})
	}
	seed, err := json.Marshal(struct {
		Version     canonical.ContractVersion `json:"version"`
		Instrument  canonical.ContractRef     `json:"instrument"`
		Issuer      canonical.ContractRef     `json:"issuer"`
		Items       any                       `json:"items"`
		Diagnostics []string                  `json:"diagnostics"`
	}{PacketContractV1, refForInstrument(instrument), refForIssuer(issuer), seedItems, packet.DiagnosticIDs})
	if err != nil {
		return Packet{}, err
	}
	packet.ID = "p03_" + canonical.DigestBytes(seed).Value[:24]
	if err := packet.Validate(); err != nil {
		return Packet{}, err
	}
	return packet, nil
}

func (packet Packet) Validate() error {
	if packet.ContractVersion != PacketContractV1 {
		return fmt.Errorf("packet contract version must be %q", PacketContractV1)
	}
	if !strings.HasPrefix(packet.ID, "p03_") || len(packet.ID) <= len("p03_") {
		return errors.New("packet ID must use p03_ prefix")
	}
	if err := packet.Instrument.Validate(); err != nil {
		return fmt.Errorf("instrument: %w", err)
	}
	if err := packet.Issuer.Validate(); err != nil {
		return fmt.Errorf("issuer: %w", err)
	}
	if !instrumentReferencesIssuer(packet.Instrument, packet.Issuer.ID) {
		return errors.New("instrument is not linked to packet issuer")
	}
	if len(packet.Items) == 0 {
		return errors.New("packet requires evidence items")
	}
	if packet.CreatedAt.IsZero() || packet.CreatedAt.Location() != time.UTC {
		return errors.New("created_at must be a non-zero UTC timestamp")
	}
	if expected, err := packetID(packet); err != nil || packet.ID != expected {
		return errors.New("packet ID does not match deterministic packet inputs")
	}
	seen := map[string]bool{}
	for i, item := range packet.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
		key := itemKey(item)
		if seen[key] {
			return fmt.Errorf("duplicate evidence item %q", key)
		}
		seen[key] = true
	}
	return nil
}

func packetID(packet Packet) (string, error) {
	items := append([]EvidenceItem(nil), packet.Items...)
	sort.Slice(items, func(i, j int) bool { return itemKey(items[i]) < itemKey(items[j]) })
	seedItems := make([]struct {
		Family        EvidenceFamily                `json:"family"`
		Origin        EvidenceOrigin                `json:"origin"`
		NormalizedRef canonical.ContractRef         `json:"normalized_ref"`
		RawID         providercontract.RawPayloadID `json:"raw_id"`
		Digest        string                        `json:"digest"`
	}, 0, len(items))
	for _, item := range items {
		seedItems = append(seedItems, struct {
			Family        EvidenceFamily                `json:"family"`
			Origin        EvidenceOrigin                `json:"origin"`
			NormalizedRef canonical.ContractRef         `json:"normalized_ref"`
			RawID         providercontract.RawPayloadID `json:"raw_id"`
			Digest        string                        `json:"digest"`
		}{item.Family, item.Origin, item.NormalizedRef, item.RawPayload.ID, item.RawPayload.Content.Digest.Value})
	}
	diagnostics := append([]string(nil), packet.DiagnosticIDs...)
	sort.Strings(diagnostics)
	seed, err := json.Marshal(struct {
		Version     canonical.ContractVersion `json:"version"`
		Instrument  canonical.ContractRef     `json:"instrument"`
		Issuer      canonical.ContractRef     `json:"issuer"`
		Items       any                       `json:"items"`
		Diagnostics []string                  `json:"diagnostics"`
	}{PacketContractV1, refForInstrument(packet.Instrument), refForIssuer(packet.Issuer), seedItems, diagnostics})
	if err != nil {
		return "", err
	}
	return "p03_" + canonical.DigestBytes(seed).Value[:24], nil
}

// AssertRealExitCondition is the fail-closed Phase 03 gate assertion. A
// fixture may demonstrate contract behaviour but can never satisfy a real
// evidence family requirement.
func (packet Packet) AssertRealExitCondition() error {
	if err := packet.Validate(); err != nil {
		return err
	}
	required := map[EvidenceFamily]bool{FamilyMarket: false, FamilyCompany: false, FamilyMacroContext: false}
	for _, item := range packet.Items {
		if _, ok := required[item.Family]; !ok || !item.Origin.IsReal() {
			continue
		}
		if item.RawPayload.Source == nil || item.RawPayload.Content.Digest.Value == "" {
			return fmt.Errorf("real %s evidence lacks raw source provenance", item.Family)
		}
		required[item.Family] = true
	}
	for family, present := range required {
		if !present {
			return fmt.Errorf("real %s evidence is required", family)
		}
	}
	return nil
}

func provenanceContainsRaw(provenance canonical.Provenance, raw providercontract.RawPayloadRef) bool {
	for _, input := range provenance.Inputs {
		if input.Evidence == nil || input.Evidence.Content.Digest != raw.Content.Digest {
			continue
		}
		if raw.Source != nil && input.Evidence.Source != *raw.Source {
			continue
		}
		if input.Evidence.Provider != nil && !sameProvider(*input.Evidence.Provider, raw.Provider) {
			continue
		}
		return true
	}
	return false
}

func sameProvider(left, right canonical.ProviderIdentity) bool {
	if left.ID != right.ID || left.Namespace != right.Namespace {
		return false
	}
	if left.ExternalID == nil || right.ExternalID == nil {
		return left.ExternalID == nil && right.ExternalID == nil
	}
	return *left.ExternalID == *right.ExternalID
}

func itemKey(item EvidenceItem) string {
	return string(item.Family) + "\x00" + string(item.Origin) + "\x00" + item.NormalizedRef.ID + "\x00" + string(item.RawPayload.ID)
}
func refForInstrument(value canonical.Instrument) canonical.ContractRef {
	return canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(value.ID), ContractVersion: value.ContractVersion}
}
func refForIssuer(value canonical.Issuer) canonical.ContractRef {
	return canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(value.ID), ContractVersion: value.ContractVersion}
}
func instrumentReferencesIssuer(instrument canonical.Instrument, issuer canonical.IssuerID) bool {
	for _, link := range instrument.Issuers {
		if link.IssuerID == issuer {
			return true
		}
	}
	return false
}
