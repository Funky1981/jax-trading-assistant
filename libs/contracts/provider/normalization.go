package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"jax-trading-assistant/libs/contracts/canonical"
)

const NormalizerDescriptorV1 canonical.ContractVersion = "jax.provider_normalizer/v1"

type NormalizationStage string

const (
	NormalizationStageRawVerification      NormalizationStage = "RAW_VERIFICATION"
	NormalizationStageRouting              NormalizationStage = "ROUTING"
	NormalizationStageParsing              NormalizationStage = "PARSING"
	NormalizationStageMapping              NormalizationStage = "NORMALIZATION"
	NormalizationStageCanonicalValidation  NormalizationStage = "CANONICAL_VALIDATION"
	NormalizationStageProvenanceValidation NormalizationStage = "PROVENANCE_VALIDATION"
	NormalizationStageAcceptance           NormalizationStage = "ACCEPTANCE"
)

type NormalizationErrorCode string

const (
	NormalizationErrorRawContentVerification    NormalizationErrorCode = "raw_content_verification_failure"
	NormalizationErrorUnsupportedProvider       NormalizationErrorCode = "unsupported_provider"
	NormalizationErrorUnsupportedCapability     NormalizationErrorCode = "unsupported_capability"
	NormalizationErrorUnsupportedRawSchema      NormalizationErrorCode = "unsupported_raw_schema"
	NormalizationErrorUnsupportedRepresentation NormalizationErrorCode = "unsupported_representation"
	NormalizationErrorParserFailure             NormalizationErrorCode = "parser_failure"
	NormalizationErrorRequiredFieldMissing      NormalizationErrorCode = "required_provider_field_missing"
	NormalizationErrorInvalidProviderValue      NormalizationErrorCode = "invalid_provider_value"
	NormalizationErrorAmbiguousProviderValue    NormalizationErrorCode = "ambiguous_provider_value"
	NormalizationErrorUnmappableProviderValue   NormalizationErrorCode = "unmappable_provider_value"
	NormalizationErrorIdentityResolution        NormalizationErrorCode = "identity_resolution_failure"
	NormalizationErrorCanonicalConstruction     NormalizationErrorCode = "canonical_construction_failure"
	NormalizationErrorCanonicalValidation       NormalizationErrorCode = "canonical_validation_failure"
	NormalizationErrorProvenanceValidation      NormalizationErrorCode = "provenance_validation_failure"
	NormalizationErrorUnsupportedTargetVersion  NormalizationErrorCode = "unsupported_target_canonical_version"
	NormalizationErrorNormalizerVersionMismatch NormalizationErrorCode = "normalizer_version_mismatch"
	NormalizationErrorUnknownNormalizer         NormalizationErrorCode = "unknown_normalizer"
	NormalizationErrorAmbiguousNormalizer       NormalizationErrorCode = "ambiguous_normalizer_registration"
	NormalizationErrorLossMetadata              NormalizationErrorCode = "loss_metadata_inconsistent"
	NormalizationErrorNonDeterministic          NormalizationErrorCode = "non_deterministic_output"
)

// NormalizationError is safe for operator-visible handling. Detail must remain
// bounded and must not contain payload bytes, credentials, headers, or secrets.
type NormalizationError struct {
	Stage        NormalizationStage
	Code         NormalizationErrorCode
	PayloadID    RawPayloadID
	ProviderID   string
	CapabilityID CapabilityID
	Detail       string
	Cause        error
}

func (err *NormalizationError) Error() string {
	return fmt.Sprintf("provider normalization %s/%s: payload=%q provider=%q capability=%q %s", err.Stage, err.Code, err.PayloadID, err.ProviderID, err.CapabilityID, err.Detail)
}

func (err *NormalizationError) Unwrap() error { return err.Cause }

// NormalizerDescriptor binds one deterministic mapping implementation to one
// provider, Jax capability, provider-raw representation, and canonical target.
// Component is the mapping/normalizer version that can affect canonical meaning.
type NormalizerDescriptor struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	Provider        canonical.ProviderIdentity  `json:"provider"`
	CapabilityID    CapabilityID                `json:"capability_id"`
	Raw             RawRepresentation           `json:"raw"`
	Component       canonical.ComponentIdentity `json:"component"`
	Target          canonical.ContractSchemaRef `json:"target"`
}

func (descriptor NormalizerDescriptor) Validate() error {
	const contract = "normalizer_descriptor"
	if descriptor.ContractVersion != NormalizerDescriptorV1 {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", NormalizerDescriptorV1))
	}
	if err := descriptor.Provider.Validate(); err != nil {
		return invalid(contract, "provider", err.Error())
	}
	outputs, ok := CanonicalOutputsFor(descriptor.CapabilityID)
	if !ok {
		return invalid(contract, "capability_id", "is not a supported Jax capability")
	}
	if err := descriptor.Raw.Validate(); err != nil {
		return invalid(contract, "raw", err.Error())
	}
	if err := descriptor.Component.Validate(); err != nil {
		return invalid(contract, "component", err.Error())
	}
	if descriptor.Component.Kind != canonical.ComponentKindNormalizer && descriptor.Component.Kind != canonical.ComponentKindMapping {
		return invalid(contract, "component.kind", "must be normalizer or mapping")
	}
	if descriptor.Component.Provider == nil || !sameProviderIdentity(*descriptor.Component.Provider, descriptor.Provider) {
		return invalid(contract, "component.provider", "must identify the provider to which the mapping is bound")
	}
	if err := descriptor.Target.Validate(); err != nil {
		return invalid(contract, "target", err.Error())
	}
	for _, output := range outputs {
		if descriptor.Target == output {
			return nil
		}
	}
	return invalid(contract, "target", "is not an authoritative canonical output for the capability")
}

type FieldDispositionStatus string

const (
	FieldDispositionRepresented           FieldDispositionStatus = "REPRESENTED"
	FieldDispositionIntentionallyOmitted  FieldDispositionStatus = "INTENTIONALLY_OMITTED"
	FieldDispositionUnsupportedUnmappable FieldDispositionStatus = "UNSUPPORTED_UNMAPPABLE"
)

// FieldDisposition records semantic treatment, not a duplicate raw schema.
// Invalid or ambiguous provider information is a typed failure and is never a
// successful disposition.
type FieldDisposition struct {
	ProviderField  string                 `json:"provider_field"`
	Status         FieldDispositionStatus `json:"status"`
	CanonicalField string                 `json:"canonical_field,omitempty"`
	ReasonCode     string                 `json:"reason_code,omitempty"`
}

func validateFieldDispositions(dispositions []FieldDisposition) error {
	if len(dispositions) == 0 {
		return fmt.Errorf("at least one provider-field disposition is required")
	}
	seen := make(map[string]struct{}, len(dispositions))
	represented := false
	for i, disposition := range dispositions {
		field := fmt.Sprintf("dispositions[%d]", i)
		if err := validateText("normalization_candidate", field+".provider_field", disposition.ProviderField, maxShortText); err != nil {
			return err
		}
		if _, ok := seen[disposition.ProviderField]; ok {
			return fmt.Errorf("%s.provider_field duplicates an earlier field", field)
		}
		seen[disposition.ProviderField] = struct{}{}
		switch disposition.Status {
		case FieldDispositionRepresented:
			represented = true
			if err := validateText("normalization_candidate", field+".canonical_field", disposition.CanonicalField, maxShortText); err != nil {
				return err
			}
			if disposition.ReasonCode != "" {
				return fmt.Errorf("%s.reason_code must be empty for represented information", field)
			}
		case FieldDispositionIntentionallyOmitted, FieldDispositionUnsupportedUnmappable:
			if disposition.CanonicalField != "" {
				return fmt.Errorf("%s.canonical_field must be empty for information not represented", field)
			}
			if err := validateCode("normalization_candidate", field+".reason_code", disposition.ReasonCode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s.status is not supported", field)
		}
	}
	if !represented {
		return fmt.Errorf("at least one provider field must be faithfully represented")
	}
	return nil
}

// NormalizationInput contains an already accepted raw reference and the exact
// retrieved bytes. Pipeline.Normalize independently verifies the bytes before
// invoking a normalizer. The normalizer must not acquire data itself.
type NormalizationInput struct {
	RawRef RawPayloadRef
	Bytes  []byte
}

// NormalizationCandidate is unaccepted until the pipeline completes canonical
// and provenance validation. A normalizer must return no usable candidate when
// it returns an error.
type NormalizationCandidate struct {
	Record       canonical.Contract
	Revision     canonical.RevisionIdentity
	Dispositions []FieldDisposition
}

// Normalizer is the provider-adapter boundary. Implementations parse into
// provider-owned typed representations and map those values without fetching,
// inference, retries, fallback, health checks, or runtime provider selection.
type Normalizer interface {
	Descriptor() NormalizerDescriptor
	Normalize(context.Context, NormalizationInput) (NormalizationCandidate, error)
}

type NormalizationStatus string

const NormalizationStatusAccepted NormalizationStatus = "ACCEPTED"

type NormalizationQuality string

// NormalizationQualityValidated is a validation result, not provider runtime
// health, freshness, TTL, or last-known-good state.
const NormalizationQualityValidated NormalizationQuality = "CANONICAL_VALIDATED"

type NormalizationValidation struct {
	RawVerified         bool `json:"raw_verified"`
	Parsed              bool `json:"parsed"`
	Mapped              bool `json:"mapped"`
	CanonicalValidated  bool `json:"canonical_validated"`
	ProvenanceValidated bool `json:"provenance_validated"`
}

// NormalizationResult is an in-memory acceptance envelope, not a second
// canonical business model. Rejections return the zero result plus a typed
// error, so partially valid canonical data cannot masquerade as accepted.
type NormalizationResult struct {
	Status       NormalizationStatus
	Quality      NormalizationQuality
	RawRef       RawPayloadRef
	Normalizer   canonical.ComponentIdentity
	Target       canonical.ContractSchemaRef
	Output       canonical.ImmutableContractRef
	Validation   NormalizationValidation
	Dispositions []FieldDisposition
	Record       canonical.Contract
}

func normalizationError(stage NormalizationStage, code NormalizationErrorCode, ref RawPayloadRef, detail string, cause error) error {
	return &NormalizationError{
		Stage: stage, Code: code, PayloadID: ref.ID, ProviderID: ref.Provider.ID,
		CapabilityID: ref.CapabilityID, Detail: detail, Cause: cause,
	}
}

func asNormalizationError(err error) (*NormalizationError, bool) {
	var target *NormalizationError
	ok := errors.As(err, &target)
	return target, ok
}

func canonicalRecordRef(record canonical.Contract) (canonical.ContractRef, error) {
	switch value := record.(type) {
	case canonical.Instrument:
		return canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case *canonical.Instrument:
		if value == nil {
			return canonical.ContractRef{}, fmt.Errorf("record is nil")
		}
		return canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case canonical.Issuer:
		return canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case *canonical.Issuer:
		if value == nil {
			return canonical.ContractRef{}, fmt.Errorf("record is nil")
		}
		return canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case canonical.Event:
		return canonical.ContractRef{Kind: canonical.ContractKindEvent, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case *canonical.Event:
		if value == nil {
			return canonical.ContractRef{}, fmt.Errorf("record is nil")
		}
		return canonical.ContractRef{Kind: canonical.ContractKindEvent, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case canonical.Evidence:
		return canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case *canonical.Evidence:
		if value == nil {
			return canonical.ContractRef{}, fmt.Errorf("record is nil")
		}
		return canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case canonical.Observation:
		return canonical.ContractRef{Kind: canonical.ContractKindObservation, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	case *canonical.Observation:
		if value == nil {
			return canonical.ContractRef{}, fmt.Errorf("record is nil")
		}
		return canonical.ContractRef{Kind: canonical.ContractKindObservation, ID: string(value.ID), ContractVersion: value.ContractVersion}, nil
	default:
		return canonical.ContractRef{}, fmt.Errorf("record type %T is not an external-provider canonical family", record)
	}
}

func sameComponentIdentity(left, right canonical.ComponentIdentity) bool {
	return reflect.DeepEqual(left, right)
}

func cloneComponentIdentity(component canonical.ComponentIdentity) canonical.ComponentIdentity {
	copyComponent := component
	if component.Provider != nil {
		provider := cloneProviderIdentity(*component.Provider)
		copyComponent.Provider = &provider
	}
	if component.Content != nil {
		content := *component.Content
		copyComponent.Content = &content
	}
	return copyComponent
}
