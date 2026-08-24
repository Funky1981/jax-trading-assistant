package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"jax-trading-assistant/libs/contracts/canonical"
)

type NormalizationRequest struct {
	RawRef     RawPayloadRef
	Bytes      []byte
	Target     canonical.ContractSchemaRef
	Normalizer canonical.ComponentIdentity
}

type StoredNormalizationRequest struct {
	RawRef     RawPayloadRef
	Target     canonical.ContractSchemaRef
	Normalizer canonical.ComponentIdentity
}

type NormalizationPipeline struct {
	providers   *Registry
	normalizers *NormalizerRegistry
}

func NewNormalizationPipeline(providers *Registry, normalizers *NormalizerRegistry) (*NormalizationPipeline, error) {
	if providers == nil || providers.ContractVersion() != RegistryContractV1 {
		return nil, &NormalizationError{Stage: NormalizationStageRouting, Code: NormalizationErrorUnsupportedProvider, Detail: "a valid provider registry is required"}
	}
	if normalizers == nil || normalizers.providers != providers {
		return nil, &NormalizationError{Stage: NormalizationStageRouting, Code: NormalizationErrorUnknownNormalizer, Detail: "normalizer registry must be bound to the same provider registry"}
	}
	return &NormalizationPipeline{providers: providers, normalizers: normalizers}, nil
}

// NormalizeStored retrieves and verifies exact bytes through the WP-02.02
// storage port before entering the normalization pipeline. It performs no raw
// acquisition, provider call, fallback, or retry.
func (pipeline *NormalizationPipeline) NormalizeStored(ctx context.Context, store RawPayloadStore, request StoredNormalizationRequest) (NormalizationResult, error) {
	bytes, err := RetrieveRawPayload(ctx, store, request.RawRef)
	if err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, request.RawRef, "raw payload could not be retrieved and verified", err)
	}
	return pipeline.Normalize(ctx, NormalizationRequest{RawRef: request.RawRef, Bytes: bytes, Target: request.Target, Normalizer: request.Normalizer})
}

// Normalize executes verify -> route -> parse/map -> canonical validation ->
// provenance validation -> acceptance. Any failure returns a zero result.
func (pipeline *NormalizationPipeline) Normalize(ctx context.Context, request NormalizationRequest) (NormalizationResult, error) {
	ref := request.RawRef
	if pipeline == nil || pipeline.providers == nil || pipeline.normalizers == nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnknownNormalizer, ref, "normalization pipeline is not initialized", nil)
	}
	if err := ctx.Err(); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "context ended before raw verification", err)
	}
	if err := ref.Validate(); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "raw payload reference is invalid", err)
	}
	if err := verifyRawPayloadBytes(ref, request.Bytes); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "exact raw bytes do not match the immutable reference", err)
	}

	capability, err := pipeline.providers.Capability(ref.Provider, ref.CapabilityID)
	if err != nil {
		return NormalizationResult{}, mapRegistryNormalizationError(ref, err)
	}
	if capability.Support != SupportSupported {
		return NormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedCapability, ref, "capability is not statically supported", nil)
	}
	if err := validateRawRepresentationMatch(capability.Raw, ref.Raw); err != nil {
		code := NormalizationErrorUnsupportedRawSchema
		if ref.Raw.Boundary == capability.Raw.Boundary && ref.Raw.Format == capability.Raw.Format && ref.Raw.Schema == capability.Raw.Schema {
			code = NormalizationErrorUnsupportedRepresentation
		}
		return NormalizationResult{}, normalizationError(NormalizationStageRouting, code, ref, "raw representation does not match the registered capability", err)
	}
	if err := validateNormalizationTarget(capability, request.Target); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedTargetVersion, ref, "target canonical schema is not declared for the capability", err)
	}

	normalizer, descriptor, err := pipeline.normalizers.selectNormalizer(ref, request.Target, request.Normalizer)
	if err != nil {
		return NormalizationResult{}, err
	}
	input := NormalizationInput{RawRef: cloneRawPayloadRef(ref), Bytes: append([]byte(nil), request.Bytes...)}
	candidate, err := normalizer.Normalize(ctx, input)
	if err != nil {
		if typed, ok := asNormalizationError(err); ok {
			copyError := *typed
			copyError.PayloadID = ref.ID
			copyError.ProviderID = ref.Provider.ID
			copyError.CapabilityID = ref.CapabilityID
			copyError.Detail = safeAdapterFailureDetail(copyError.Code)
			return NormalizationResult{}, &copyError
		}
		return NormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "normalizer failed without a typed provider-safe error", err)
	}
	if candidate.Record == nil || (reflect.ValueOf(candidate.Record).Kind() == reflect.Ptr && reflect.ValueOf(candidate.Record).IsNil()) {
		return NormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "normalizer returned no canonical candidate", nil)
	}
	if err := validateFieldDispositions(candidate.Dispositions); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorLossMetadata, ref, "normalizer returned inconsistent information-disposition metadata", err)
	}
	recordRef, err := canonicalRecordRef(candidate.Record)
	if err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "normalizer returned an unsupported canonical family", err)
	}
	if recordRef.Kind != request.Target.Kind || recordRef.ContractVersion != request.Target.Version {
		return NormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "constructed record does not match the exact target family/version", nil)
	}
	if err := candidate.Record.Validate(); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageCanonicalValidation, NormalizationErrorCanonicalValidation, ref, "canonical contract validation failed", err)
	}
	if err := validateRawProvenanceLink(candidate.Record, ref, descriptor.Component); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageProvenanceValidation, NormalizationErrorProvenanceValidation, ref, "canonical lineage does not match the exact raw input and normalizer", err)
	}
	content, err := canonical.CanonicalContractContentIdentity(candidate.Record)
	if err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorCanonicalConstruction, ref, "canonical output bytes could not be identified", err)
	}
	output := canonical.ImmutableContractRef{Contract: recordRef, Revision: candidate.Revision, Content: content}
	if err := output.Validate(); err != nil {
		return NormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorCanonicalConstruction, ref, "canonical output reference is invalid", err)
	}

	return NormalizationResult{
		Status: NormalizationStatusAccepted, Quality: NormalizationQualityValidated,
		RawRef: cloneRawPayloadRef(ref), Normalizer: cloneComponentIdentity(descriptor.Component), Target: request.Target,
		Output:       output,
		Validation:   NormalizationValidation{RawVerified: true, Parsed: true, Mapped: true, CanonicalValidated: true, ProvenanceValidated: true},
		Dispositions: append([]FieldDisposition(nil), candidate.Dispositions...), Record: candidate.Record,
	}, nil
}

func safeAdapterFailureDetail(code NormalizationErrorCode) string {
	switch code {
	case NormalizationErrorParserFailure:
		return "provider representation could not be parsed"
	case NormalizationErrorRequiredFieldMissing:
		return "provider representation is missing required information"
	case NormalizationErrorInvalidProviderValue:
		return "provider representation contains an invalid value"
	case NormalizationErrorAmbiguousProviderValue:
		return "provider representation contains an ambiguous value"
	case NormalizationErrorUnmappableProviderValue:
		return "provider representation contains an unmappable value"
	case NormalizationErrorIdentityResolution:
		return "canonical identity resolution failed"
	case NormalizationErrorCanonicalConstruction:
		return "canonical candidate construction failed"
	case NormalizationErrorProvenanceValidation:
		return "normalizer could not construct valid immutable lineage"
	default:
		return "normalizer rejected provider input"
	}
}

// VerifyDeterministicNormalization runs an identical request twice and rejects
// any canonical-byte, immutable-output, or loss-metadata difference. It is a
// gate/proof helper; normal production acceptance remains one Normalize call.
func VerifyDeterministicNormalization(ctx context.Context, pipeline *NormalizationPipeline, request NormalizationRequest) (NormalizationResult, error) {
	first, err := pipeline.Normalize(ctx, request)
	if err != nil {
		return NormalizationResult{}, err
	}
	second, err := pipeline.Normalize(ctx, request)
	if err != nil {
		return NormalizationResult{}, err
	}
	firstBytes, firstErr := canonical.CanonicalContractBytes(first.Record)
	secondBytes, secondErr := canonical.CanonicalContractBytes(second.Record)
	if firstErr != nil || secondErr != nil || !bytes.Equal(firstBytes, secondBytes) || first.Output != second.Output || !reflect.DeepEqual(first.Dispositions, second.Dispositions) {
		return NormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorNonDeterministic, request.RawRef, "identical raw input and mapping identity produced different accepted output", errors.Join(firstErr, secondErr))
	}
	return first, nil
}

func validateNormalizationTarget(capability Capability, target canonical.ContractSchemaRef) error {
	if err := target.Validate(); err != nil {
		return err
	}
	for _, output := range capability.CanonicalOutputs {
		if output == target {
			return nil
		}
	}
	return fmt.Errorf("target is not one of the capability's exact canonical outputs")
}

func validateRawProvenanceLink(record canonical.Contract, ref RawPayloadRef, component canonical.ComponentIdentity) error {
	switch value := record.(type) {
	case canonical.Observation:
		return validateObservationRawLink(value, ref, component)
	case *canonical.Observation:
		return validateObservationRawLink(*value, ref, component)
	case canonical.Evidence:
		return validateEvidenceRawLink(value, ref, component)
	case *canonical.Evidence:
		return validateEvidenceRawLink(*value, ref, component)
	case canonical.Instrument, *canonical.Instrument, canonical.Issuer, *canonical.Issuer, canonical.Event, *canonical.Event:
		// These accepted families have no Phase 01 provenance-bearing schema.
		// The acceptance envelope binds their exact output to ref and component.
		return nil
	default:
		return fmt.Errorf("provider output family %T is not supported", record)
	}
}

func validateObservationRawLink(observation canonical.Observation, ref RawPayloadRef, component canonical.ComponentIdentity) error {
	if observation.ContractVersion != canonical.ObservationContractV2 || observation.Provenance == nil {
		return fmt.Errorf("provider observations must use provenance-bearing observation v2")
	}
	if !sameComponentIdentity(observation.Provenance.Producer, component) {
		return fmt.Errorf("observation provenance producer does not match the selected normalizer")
	}
	if ref.Source == nil || ref.Revision == nil {
		return fmt.Errorf("raw source and revision are required; source identity cannot be fabricated")
	}
	if observation.Source.ID != ref.Source.ID || observation.Source.Kind != ref.Source.Kind {
		return fmt.Errorf("observation source does not match the raw logical source")
	}
	for _, input := range observation.Provenance.Inputs {
		if input.Kind != canonical.LineageInputKindEvidence || input.Evidence == nil {
			continue
		}
		evidence := input.Evidence
		if evidence.Content == ref.Content && evidence.Provider != nil && sameProviderIdentity(*evidence.Provider, ref.Provider) &&
			evidence.Source == *ref.Source && evidence.Revision == *ref.Revision && evidence.CollectedAt.Equal(ref.ReceivedAt) {
			return nil
		}
	}
	return fmt.Errorf("observation provenance does not contain the exact raw evidence identity")
}

func validateEvidenceRawLink(evidence canonical.Evidence, ref RawPayloadRef, component canonical.ComponentIdentity) error {
	if evidence.ContractVersion != canonical.EvidenceContractV2 || evidence.ImmutableRef == nil {
		return fmt.Errorf("provider evidence must use evidence v2 with an immutable reference")
	}
	if ref.Source == nil || ref.Revision == nil {
		return fmt.Errorf("raw source and revision are required; source identity cannot be fabricated")
	}
	immutable := evidence.ImmutableRef
	if immutable.Content != ref.Content || immutable.Provider == nil || !sameProviderIdentity(*immutable.Provider, ref.Provider) ||
		immutable.Source != *ref.Source || immutable.Revision != *ref.Revision || !immutable.CollectedAt.Equal(ref.ReceivedAt) {
		return fmt.Errorf("evidence immutable reference does not match the exact raw input")
	}
	if evidence.Provenance != nil && !sameComponentIdentity(evidence.Provenance.Producer, component) {
		return fmt.Errorf("evidence provenance producer does not match the selected normalizer")
	}
	return nil
}
