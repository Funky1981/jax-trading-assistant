package canonical

import (
	"fmt"
	"time"
)

type EvidenceType string

const (
	EvidenceTypeArticle         EvidenceType = "article"
	EvidenceTypeFiling          EvidenceType = "filing"
	EvidenceTypeOfficialRelease EvidenceType = "official_release"
	EvidenceTypeTranscript      EvidenceType = "transcript"
	EvidenceTypeDocument        EvidenceType = "document"
	EvidenceTypeDataset         EvidenceType = "dataset"
	EvidenceTypeResearchOutput  EvidenceType = "research_output"
	EvidenceTypeModelOutput     EvidenceType = "model_output"
)

type EvidenceRelationship string

const (
	EvidenceRelationshipSupports    EvidenceRelationship = "supports"
	EvidenceRelationshipContradicts EvidenceRelationship = "contradicts"
	EvidenceRelationshipDescribes   EvidenceRelationship = "describes"
)

// EvidenceLink states how a source-backed Evidence item relates to another
// canonical record. It is not the immutable EvidenceRef/provenance identity
// that WP-01.03 will define.
type EvidenceLink struct {
	Target       ContractRef          `json:"target"`
	Relationship EvidenceRelationship `json:"relationship"`
}

// Evidence is source material supporting, contradicting, or describing an
// event or research conclusion. It does not assert that the source material is
// itself the event or a measured observation.
type Evidence struct {
	ContractVersion ContractVersion `json:"contract_version"`
	ID              EvidenceID      `json:"id"`
	Type            EvidenceType    `json:"type"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary,omitempty"`
	Source          SourceReference `json:"source"`
	Links           []EvidenceLink  `json:"links"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	CollectedAt     time.Time       `json:"collected_at"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (evidence Evidence) Validate() error {
	const contract = "evidence"
	if err := validateVersion(contract, evidence.ContractVersion, EvidenceContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(evidence.ID), "evd_"); err != nil {
		return err
	}
	switch evidence.Type {
	case EvidenceTypeArticle, EvidenceTypeFiling, EvidenceTypeOfficialRelease, EvidenceTypeTranscript,
		EvidenceTypeDocument, EvidenceTypeDataset, EvidenceTypeResearchOutput, EvidenceTypeModelOutput:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateRequiredText(contract, "title", evidence.Title, maxShortText); err != nil {
		return err
	}
	if evidence.Summary != "" {
		if err := validateRequiredText(contract, "summary", evidence.Summary, maxDescription); err != nil {
			return err
		}
	}
	if err := validateSource(contract, "source", evidence.Source); err != nil {
		return err
	}
	if len(evidence.Links) == 0 {
		return invalid(contract, "links", "requires at least one described or supported target")
	}
	allowedTargets := map[ContractKind]bool{
		ContractKindInstrument:     true,
		ContractKindIssuer:         true,
		ContractKindEvent:          true,
		ContractKindObservation:    true,
		ContractKindResearchRun:    true,
		ContractKindQuantResult:    true,
		ContractKindRecommendation: true,
	}
	seen := make(map[string]struct{}, len(evidence.Links))
	for i, link := range evidence.Links {
		field := fmt.Sprintf("links[%d]", i)
		if err := validateContractRef(contract, field+".target", link.Target); err != nil {
			return err
		}
		if !allowedTargets[link.Target.Kind] {
			return invalid(contract, field+".target.kind", "is not allowed as an evidence target")
		}
		switch link.Relationship {
		case EvidenceRelationshipSupports, EvidenceRelationshipContradicts, EvidenceRelationshipDescribes:
		default:
			return invalid(contract, field+".relationship", "is not supported")
		}
		key := string(link.Target.Kind) + "\x00" + link.Target.ID
		if _, ok := seen[key]; ok {
			return invalid(contract, field, "repeats a target with ambiguous evidence relationships")
		}
		seen[key] = struct{}{}
	}
	if err := validateOptionalUTC(contract, "published_at", evidence.PublishedAt); err != nil {
		return err
	}
	if err := validateRequiredUTC(contract, "collected_at", evidence.CollectedAt); err != nil {
		return err
	}
	if evidence.PublishedAt != nil && evidence.CollectedAt.Before(*evidence.PublishedAt) {
		return invalid(contract, "collected_at", "must not precede published_at")
	}
	if err := validateRequiredUTC(contract, "created_at", evidence.CreatedAt); err != nil {
		return err
	}
	if evidence.CreatedAt.Before(evidence.CollectedAt) {
		return invalid(contract, "created_at", "must not precede collected_at")
	}
	return nil
}
