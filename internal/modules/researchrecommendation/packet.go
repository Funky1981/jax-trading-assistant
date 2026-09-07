package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	EvidencePacketContractV1 = "jax.evidence_packet/v1"
	EvidenceRenderingV1      = "jax.evidence_packet_rendering/v1"
)

type EvidenceKind string

const (
	EvidenceKindInstrument   EvidenceKind = "INSTRUMENT"
	EvidenceKindCompany      EvidenceKind = "COMPANY"
	EvidenceKindMarket       EvidenceKind = "MARKET"
	EvidenceKindMacro        EvidenceKind = "MACRO_CONTEXT"
	EvidenceKindWorldMonitor EvidenceKind = "WORLD_MONITOR"
	EvidenceKindQuant        EvidenceKind = "QUANT"
)

type FreshnessState string

const (
	FreshnessFresh   FreshnessState = "FRESH"
	FreshnessStale   FreshnessState = "STALE"
	FreshnessUnknown FreshnessState = "UNKNOWN"
)

type SourceReference struct {
	SourceID         string     `json:"source_id"`
	Provider         string     `json:"provider"`
	RawReference     string     `json:"raw_reference"`
	RawContentSHA256 string     `json:"raw_content_sha256"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	ObservedAt       *time.Time `json:"observed_at,omitempty"`
	CollectedAt      time.Time  `json:"collected_at"`
	Vintage          string     `json:"vintage,omitempty"`
}

type QuantResultReference struct {
	ResultID            string `json:"result_id"`
	Algorithm           string `json:"algorithm"`
	AlgorithmVersion    string `json:"algorithm_version"`
	FrozenInputSHA256   string `json:"frozen_input_sha256"`
	ResultContentSHA256 string `json:"result_content_sha256"`
}

type EvidenceIdentity struct {
	ID              string                `json:"id"`
	Kind            EvidenceKind          `json:"kind"`
	CanonicalRef    canonical.ContractRef `json:"canonical_ref"`
	Source          SourceReference       `json:"source"`
	IndependenceKey string                `json:"independence_key"`
	Freshness       FreshnessState        `json:"freshness"`
	QuantResult     *QuantResultReference `json:"quant_result,omitempty"`
}

type EvidenceRendering struct {
	Title          string         `json:"title"`
	Summary        string         `json:"summary"`
	Excerpt        string         `json:"excerpt,omitempty"`
	NumericContext []NumericValue `json:"numeric_context,omitempty"`
}

type NumericValue struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
}

type EvidenceItem struct {
	Identity  EvidenceIdentity  `json:"identity"`
	Rendering EvidenceRendering `json:"rendering"`
}

type EvidencePacket struct {
	ContractVersion  string                `json:"contract_version"`
	RenderingVersion string                `json:"rendering_version"`
	ID               string                `json:"id"`
	Subject          canonical.ContractRef `json:"subject"`
	Items            []EvidenceItem        `json:"items"`
	PacketUnknowns   []string              `json:"packet_unknowns,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
}

func NewEvidencePacket(subject canonical.ContractRef, items []EvidenceItem, unknowns []string, createdAt time.Time) (EvidencePacket, error) {
	packet := EvidencePacket{ContractVersion: EvidencePacketContractV1, RenderingVersion: EvidenceRenderingV1, Subject: subject, Items: append([]EvidenceItem(nil), items...), PacketUnknowns: append([]string(nil), unknowns...), CreatedAt: createdAt}
	sort.Slice(packet.Items, func(left, right int) bool { return packet.Items[left].Identity.ID < packet.Items[right].Identity.ID })
	packet.PacketUnknowns = sortedUnique(packet.PacketUnknowns)
	if err := packet.validateWithoutID(); err != nil {
		return EvidencePacket{}, err
	}
	packet.ID = packet.derivedID()
	return packet, packet.Validate()
}

func (packet EvidencePacket) Validate() error {
	if err := packet.validateWithoutID(); err != nil {
		return err
	}
	if !strings.HasPrefix(packet.ID, "epk_") || len(packet.ID) != len("epk_")+64 {
		return fmt.Errorf("packet ID must be an epk_ SHA-256 identity")
	}
	if packet.ID != packet.derivedID() {
		return fmt.Errorf("packet ID does not match source identities")
	}
	return nil
}

func (packet EvidencePacket) validateWithoutID() error {
	if packet.ContractVersion != EvidencePacketContractV1 || packet.RenderingVersion != EvidenceRenderingV1 {
		return fmt.Errorf("unsupported evidence packet or rendering contract")
	}
	if packet.Subject.Kind != canonical.ContractKindInstrument && packet.Subject.Kind != canonical.ContractKindIssuer && packet.Subject.Kind != canonical.ContractKindEvent {
		return fmt.Errorf("packet subject must be an instrument, issuer, or event")
	}
	if strings.TrimSpace(packet.Subject.ID) == "" || strings.TrimSpace(string(packet.Subject.ContractVersion)) == "" {
		return fmt.Errorf("packet subject identity is required")
	}
	if packet.CreatedAt.IsZero() || packet.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("packet created_at must be UTC")
	}
	if len(packet.Items) == 0 {
		return fmt.Errorf("evidence packet requires at least one item")
	}
	seen := map[string]struct{}{}
	for index, item := range packet.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %d: %w", index, err)
		}
		if _, exists := seen[item.Identity.ID]; exists {
			return fmt.Errorf("duplicate evidence item %q", item.Identity.ID)
		}
		seen[item.Identity.ID] = struct{}{}
	}
	return nil
}

func (item EvidenceItem) Validate() error {
	if !strings.HasPrefix(item.Identity.ID, "epi_") || len(item.Identity.ID) != len("epi_")+64 {
		return fmt.Errorf("evidence item ID must be an epi_ SHA-256 identity")
	}
	switch item.Identity.Kind {
	case EvidenceKindInstrument, EvidenceKindCompany, EvidenceKindMarket, EvidenceKindMacro, EvidenceKindWorldMonitor, EvidenceKindQuant:
	default:
		return fmt.Errorf("unsupported evidence kind %q", item.Identity.Kind)
	}
	if strings.TrimSpace(item.Identity.CanonicalRef.ID) == "" || strings.TrimSpace(string(item.Identity.CanonicalRef.ContractVersion)) == "" {
		return fmt.Errorf("canonical evidence reference is required")
	}
	if strings.TrimSpace(item.Identity.IndependenceKey) == "" {
		return fmt.Errorf("independence key is required")
	}
	switch item.Identity.Freshness {
	case FreshnessFresh, FreshnessStale, FreshnessUnknown:
	default:
		return fmt.Errorf("unsupported freshness state %q", item.Identity.Freshness)
	}
	if err := item.Identity.Source.Validate(); err != nil {
		return err
	}
	if item.Identity.Kind == EvidenceKindQuant {
		if item.Identity.QuantResult == nil {
			return fmt.Errorf("quant evidence requires a Phase-05 result reference")
		}
		if err := item.Identity.QuantResult.Validate(); err != nil {
			return err
		}
	}
	if strings.TrimSpace(item.Rendering.Title) == "" || strings.TrimSpace(item.Rendering.Summary) == "" {
		return fmt.Errorf("evidence rendering requires title and summary")
	}
	for index, numeric := range item.Rendering.NumericContext {
		if strings.TrimSpace(numeric.Metric) == "" || strings.TrimSpace(numeric.Unit) == "" || !finite(numeric.Value) {
			return fmt.Errorf("numeric context %d is invalid", index)
		}
	}
	return nil
}

func (source SourceReference) Validate() error {
	if strings.TrimSpace(source.SourceID) == "" || strings.TrimSpace(source.Provider) == "" || strings.TrimSpace(source.RawReference) == "" || strings.TrimSpace(source.Vintage) == "" && source.ObservedAt == nil && source.PublishedAt == nil {
		return fmt.Errorf("source, provider, raw reference, and temporal identity are required")
	}
	if !validSHA256(source.RawContentSHA256) {
		return fmt.Errorf("raw content SHA-256 must be canonical lowercase hexadecimal")
	}
	if source.CollectedAt.IsZero() || source.CollectedAt.Location() != time.UTC {
		return fmt.Errorf("source collected_at must be UTC")
	}
	for name, value := range map[string]*time.Time{"published_at": source.PublishedAt, "observed_at": source.ObservedAt} {
		if value != nil && (value.IsZero() || value.Location() != time.UTC || value.After(source.CollectedAt)) {
			return fmt.Errorf("source %s must be UTC and no later than collected_at", name)
		}
	}
	return nil
}

func (reference QuantResultReference) Validate() error {
	if strings.TrimSpace(reference.ResultID) == "" || strings.TrimSpace(reference.Algorithm) == "" || strings.TrimSpace(reference.AlgorithmVersion) == "" || !validSHA256(reference.FrozenInputSHA256) || !validSHA256(reference.ResultContentSHA256) {
		return fmt.Errorf("quant result reference requires IDs, versions, and valid SHA-256 identities")
	}
	return nil
}

func (packet EvidencePacket) derivedID() string {
	identities := make([]EvidenceIdentity, len(packet.Items))
	for index, item := range packet.Items {
		identities[index] = item.Identity
	}
	sort.Slice(identities, func(left, right int) bool { return identities[left].ID < identities[right].ID })
	seed, _ := json.Marshal(struct {
		ContractVersion string                `json:"contract_version"`
		Subject         canonical.ContractRef `json:"subject"`
		Identities      []EvidenceIdentity    `json:"identities"`
		Unknowns        []string              `json:"unknowns"`
	}{packet.ContractVersion, packet.Subject, identities, sortedUnique(packet.PacketUnknowns)})
	digest := sha256.Sum256(seed)
	return "epk_" + hex.EncodeToString(digest[:])
}

func (packet EvidencePacket) ContentFingerprint() (string, error) {
	if err := packet.Validate(); err != nil {
		return "", err
	}
	bytes, err := json.Marshal(packet)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(bytes)
	return hex.EncodeToString(digest[:]), nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func sortedUnique(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	result := make([]string, 0, len(copyValues))
	for _, value := range copyValues {
		value = strings.TrimSpace(value)
		if value != "" && (len(result) == 0 || result[len(result)-1] != value) {
			result = append(result, value)
		}
	}
	return result
}
