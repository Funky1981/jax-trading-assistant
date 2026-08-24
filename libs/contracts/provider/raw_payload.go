package provider

import (
	"fmt"
	"mime"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const RawPayloadRefContractV1 canonical.ContractVersion = "jax.provider_raw_payload_ref/v1"

// RawPayloadID identifies one acquisition event. It is deliberately not
// content-derived: two receipts of identical bytes remain distinct history.
type RawPayloadID string

type RawPayloadByteForm string

const (
	// RawPayloadByteFormEntityBody is the complete entity/body byte sequence at
	// the adapter capture point, after protocol framing but before parsing,
	// character decoding, or provider-specific normalization.
	RawPayloadByteFormEntityBody RawPayloadByteForm = "ENTITY_BODY"
	// RawPayloadByteFormMultipartPart identifies one retained part of a
	// multipart entity. The containing payload must be related explicitly.
	RawPayloadByteFormMultipartPart RawPayloadByteForm = "MULTIPART_PART"
)

type ContentCodingState string

const (
	// ContentCodingIdentity means no content coding changed the retained bytes.
	ContentCodingIdentity ContentCodingState = "IDENTITY"
	// ContentCodingEncoded means the retained bytes still use ContentCoding.
	ContentCodingEncoded ContentCodingState = "ENCODED"
	// ContentCodingDecoded means the transport decoded ContentCoding before the
	// adapter captured the retained bytes.
	ContentCodingDecoded ContentCodingState = "DECODED"
)

type RawPayloadRelationKind string

const (
	RawPayloadRelationDecodedFrom RawPayloadRelationKind = "CONTENT_DECODED_FROM"
	RawPayloadRelationPartOf      RawPayloadRelationKind = "MULTIPART_PART_OF"
)

// RawPayloadRelation keeps separately retained transport/entity
// representations related without pretending that they have the same digest.
type RawPayloadRelation struct {
	Kind      RawPayloadRelationKind `json:"kind"`
	PayloadID RawPayloadID           `json:"payload_id"`
}

// RawPayloadCapture states which exact representation was hashed. Character
// encoding is metadata only; the bytes are never decoded or re-encoded here.
type RawPayloadCapture struct {
	ByteForm           RawPayloadByteForm  `json:"byte_form"`
	ContentCoding      string              `json:"content_coding,omitempty"`
	ContentCodingState ContentCodingState  `json:"content_coding_state"`
	CharacterEncoding  string              `json:"character_encoding,omitempty"`
	RelatedTo          *RawPayloadRelation `json:"related_to,omitempty"`
}

type RawPayloadRetentionClass string

const (
	// RawPayloadRetentionReplayAudit requires preservation for provenance,
	// audit, and replay. No calendar duration or deletion policy is implied.
	RawPayloadRetentionReplayAudit RawPayloadRetentionClass = "REPLAY_AUDIT_REQUIRED"
)

type RawPayloadRedistribution string

const (
	RawPayloadRedistributionNotAuthorized RawPayloadRedistribution = "NOT_AUTHORIZED"
	RawPayloadRedistributionRestricted    RawPayloadRedistribution = "RESTRICTED"
	RawPayloadRedistributionAuthorized    RawPayloadRedistribution = "AUTHORIZED"
)

// RawPayloadRetentionPolicy expresses preservation and redistribution intent,
// not provider-specific licence terms or lifecycle automation.
type RawPayloadRetentionPolicy struct {
	Class          RawPayloadRetentionClass  `json:"class"`
	Policy         canonical.VersionIdentity `json:"policy"`
	Redistribution RawPayloadRedistribution  `json:"redistribution"`
}

// RawPayloadRef is an immutable acquisition reference. It contains no payload
// bytes and no physical storage location. Content identifies the exact bytes;
// ID distinguishes the acquisition event from content-level deduplication.
type RawPayloadRef struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	ID              RawPayloadID                `json:"id"`
	Content         canonical.ContentIdentity   `json:"content"`
	Provider        canonical.ProviderIdentity  `json:"provider"`
	CapabilityID    CapabilityID                `json:"capability_id"`
	Raw             RawRepresentation           `json:"raw"`
	Capture         RawPayloadCapture           `json:"capture"`
	Source          *canonical.SourceIdentity   `json:"source,omitempty"`
	Revision        *canonical.RevisionIdentity `json:"revision,omitempty"`
	ReceivedAt      time.Time                   `json:"received_at"`
	SizeBytes       int64                       `json:"size_bytes"`
	Retention       RawPayloadRetentionPolicy   `json:"retention"`
}

// RawPayloadLocation is a logical store/key pair. It intentionally excludes a
// filesystem path, URL, bucket name, database row ID, or credential. A storage
// adapter resolves it to physical placement and may migrate that placement
// without changing RawPayloadRef identity.
type RawPayloadLocation struct {
	Store canonical.VersionIdentity `json:"store"`
	Key   string                    `json:"key"`
}

// RawPayloadDescriptor combines immutable identity with a current retrieval
// hint. Location is not part of RawPayloadRef and is never content identity.
type RawPayloadDescriptor struct {
	Ref      RawPayloadRef      `json:"ref"`
	Location RawPayloadLocation `json:"location"`
}

// RawPayloadPersistenceRequest carries acquisition metadata. Content and size
// are derived from Payload by PersistRawPayload and cannot be supplied by the
// caller.
type RawPayloadPersistenceRequest struct {
	ID         RawPayloadID
	Provider   canonical.ProviderIdentity
	Capability CapabilityID
	Raw        RawRepresentation
	Capture    RawPayloadCapture
	Source     *canonical.SourceIdentity
	Revision   *canonical.RevisionIdentity
	ReceivedAt time.Time
	Retention  RawPayloadRetentionPolicy
	Complete   bool
}

func (capture RawPayloadCapture) Validate() error {
	const contract = "raw_payload_capture"
	switch capture.ByteForm {
	case RawPayloadByteFormEntityBody, RawPayloadByteFormMultipartPart:
	default:
		return invalid(contract, "byte_form", "is not supported")
	}
	if capture.ContentCoding != "" {
		if err := validateToken(contract, "content_coding", capture.ContentCoding); err != nil {
			return err
		}
	}
	switch capture.ContentCodingState {
	case ContentCodingIdentity:
		if capture.ContentCoding != "" {
			return invalid(contract, "content_coding", "must be empty when content_coding_state is IDENTITY")
		}
	case ContentCodingEncoded, ContentCodingDecoded:
		if capture.ContentCoding == "" {
			return invalid(contract, "content_coding", "is required when content coding is encoded or decoded")
		}
	default:
		return invalid(contract, "content_coding_state", "is not supported")
	}
	if capture.CharacterEncoding != "" {
		if err := validateToken(contract, "character_encoding", capture.CharacterEncoding); err != nil {
			return err
		}
	}
	if capture.RelatedTo != nil {
		if err := capture.RelatedTo.Validate(); err != nil {
			return invalid(contract, "related_to", err.Error())
		}
	}
	if capture.ByteForm == RawPayloadByteFormMultipartPart && (capture.RelatedTo == nil || capture.RelatedTo.Kind != RawPayloadRelationPartOf) {
		return invalid(contract, "related_to", "MULTIPART_PART requires a MULTIPART_PART_OF relation")
	}
	if capture.RelatedTo != nil && capture.RelatedTo.Kind == RawPayloadRelationPartOf && capture.ByteForm != RawPayloadByteFormMultipartPart {
		return invalid(contract, "related_to.kind", "MULTIPART_PART_OF requires MULTIPART_PART byte form")
	}
	if capture.RelatedTo != nil && capture.RelatedTo.Kind == RawPayloadRelationDecodedFrom && capture.ContentCodingState != ContentCodingDecoded {
		return invalid(contract, "related_to.kind", "CONTENT_DECODED_FROM requires DECODED content coding state")
	}
	return nil
}

func (relation RawPayloadRelation) Validate() error {
	const contract = "raw_payload_relation"
	switch relation.Kind {
	case RawPayloadRelationDecodedFrom, RawPayloadRelationPartOf:
	default:
		return invalid(contract, "kind", "is not supported")
	}
	return validateRawPayloadID(contract, "payload_id", relation.PayloadID)
}

func (policy RawPayloadRetentionPolicy) Validate() error {
	const contract = "raw_payload_retention_policy"
	if policy.Class != RawPayloadRetentionReplayAudit {
		return invalid(contract, "class", "must require replay/audit retention for an accepted raw payload reference")
	}
	if err := policy.Policy.Validate(); err != nil {
		return invalid(contract, "policy", err.Error())
	}
	switch policy.Redistribution {
	case RawPayloadRedistributionNotAuthorized, RawPayloadRedistributionRestricted, RawPayloadRedistributionAuthorized:
		return nil
	default:
		return invalid(contract, "redistribution", "must state an explicit redistribution boundary")
	}
}

func (ref RawPayloadRef) Validate() error {
	const contract = "raw_payload_ref"
	if ref.ContractVersion != RawPayloadRefContractV1 {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", RawPayloadRefContractV1))
	}
	if err := validateRawPayloadID(contract, "id", ref.ID); err != nil {
		return err
	}
	if err := ref.Content.Validate(); err != nil {
		return invalid(contract, "content", err.Error())
	}
	if ref.Content.Representation != canonical.ContentRepresentationRawBytes {
		return invalid(contract, "content.representation", "must be raw_bytes")
	}
	if err := ref.Provider.Validate(); err != nil {
		return invalid(contract, "provider", err.Error())
	}
	if _, _, ok := capabilitySpecification(ref.CapabilityID); !ok {
		return invalid(contract, "capability_id", "is not a supported Jax capability")
	}
	if err := ref.Raw.Validate(); err != nil {
		return invalid(contract, "raw", err.Error())
	}
	if err := ref.Capture.Validate(); err != nil {
		return invalid(contract, "capture", err.Error())
	}
	if ref.Source != nil {
		if err := ref.Source.Validate(); err != nil {
			return invalid(contract, "source", err.Error())
		}
	}
	if ref.Revision != nil {
		if err := ref.Revision.Validate(); err != nil {
			return invalid(contract, "revision", err.Error())
		}
	}
	_, offset := ref.ReceivedAt.Zone()
	if ref.ReceivedAt.IsZero() || offset != 0 || ref.ReceivedAt.Year() < 0 || ref.ReceivedAt.Year() > 9999 {
		return invalid(contract, "received_at", "is required and must use UTC")
	}
	if ref.SizeBytes <= 0 {
		return invalid(contract, "size_bytes", "must be positive")
	}
	if err := ref.Retention.Validate(); err != nil {
		return invalid(contract, "retention", err.Error())
	}
	if ref.Capture.RelatedTo != nil && ref.Capture.RelatedTo.PayloadID == ref.ID {
		return invalid(contract, "capture.related_to.payload_id", "must not refer to the payload itself")
	}
	return nil
}

func (location RawPayloadLocation) Validate() error {
	const contract = "raw_payload_location"
	if err := location.Store.Validate(); err != nil {
		return invalid(contract, "store", err.Error())
	}
	if err := validateText(contract, "key", location.Key, 1024); err != nil {
		return err
	}
	if strings.Contains(location.Key, "://") || strings.HasPrefix(location.Key, "/") || strings.HasPrefix(location.Key, `\`) || (len(location.Key) > 1 && location.Key[1] == ':') {
		return invalid(contract, "key", "must be an opaque logical key, not a URL or machine-local absolute path")
	}
	return nil
}

func (descriptor RawPayloadDescriptor) Validate() error {
	if err := descriptor.Ref.Validate(); err != nil {
		return err
	}
	return descriptor.Location.Validate()
}

// AsEvidenceRef bridges a source-attributed raw payload into the accepted
// canonical immutable-evidence vocabulary without creating a canonical
// Evidence record or performing normalization.
func (ref RawPayloadRef) AsEvidenceRef(evidence canonical.ContractRef) (canonical.EvidenceRef, error) {
	if err := ref.Validate(); err != nil {
		return canonical.EvidenceRef{}, err
	}
	if ref.Source == nil {
		return canonical.EvidenceRef{}, invalid("raw_payload_ref", "source", "is required to create an evidence reference; source identity must not be fabricated")
	}
	if ref.Revision == nil {
		return canonical.EvidenceRef{}, invalid("raw_payload_ref", "revision", "is required to create an evidence reference")
	}
	providerIdentity := cloneProviderIdentity(ref.Provider)
	evidenceRef := canonical.EvidenceRef{
		ContractVersion: canonical.EvidenceRefContractV1,
		Evidence:        evidence,
		Content:         ref.Content,
		Source:          *ref.Source,
		Provider:        &providerIdentity,
		Revision:        *ref.Revision,
		CollectedAt:     ref.ReceivedAt,
	}
	if err := evidenceRef.Validate(); err != nil {
		return canonical.EvidenceRef{}, invalid("raw_payload_ref", "evidence", err.Error())
	}
	return evidenceRef, nil
}

func validateRawPayloadID(contract, field string, id RawPayloadID) error {
	value := string(id)
	if !strings.HasPrefix(value, "rpl_") || len(value) <= len("rpl_") {
		return invalid(contract, field, "must use the rpl_ Jax raw-payload prefix")
	}
	return validateCode(contract, field, value)
}

func validateToken(contract, field, value string) error {
	if strings.ToLower(value) != value {
		return invalid(contract, field, "must use lower-case token syntax")
	}
	if _, _, err := mime.ParseMediaType("application/x-jax; value=" + value); err != nil {
		return invalid(contract, field, "must be a valid lower-case token")
	}
	return nil
}
