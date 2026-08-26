package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"jax-trading-assistant/libs/contracts/canonical"
)

const maximumNormalizationBatchRecords = 10000

// BatchNormalizer is the bounded fan-out form of Normalizer for one provider
// payload that deterministically contains multiple canonical records. It keeps
// the accepted provider/capability/raw-schema/target route and mapping identity;
// it does not introduce a second routing or canonical contract model.
//
// Implementations also satisfy Normalizer so they can be registered through
// the accepted NormalizerRegistry. Their single-record Normalize method should
// fail closed when the raw representation cannot unambiguously mean one record.
type BatchNormalizer interface {
	Normalizer
	NormalizeBatch(context.Context, NormalizationInput) ([]NormalizationCandidate, error)
}

// BatchNormalizationResult is an in-memory acceptance envelope for the
// canonical records produced from one exact RawPayloadRef. Every child result
// independently passed the accepted canonical and provenance validation path.
type BatchNormalizationResult struct {
	RawRef     RawPayloadRef
	Normalizer canonical.ComponentIdentity
	Target     canonical.ContractSchemaRef
	Records    []NormalizationResult
}

// NormalizeBatchStored retrieves and verifies exact stored bytes before
// entering the same provider/capability/raw-schema/target route used by
// single-record normalization.
func (pipeline *NormalizationPipeline) NormalizeBatchStored(ctx context.Context, store RawPayloadStore, request StoredNormalizationRequest) (BatchNormalizationResult, error) {
	payload, err := RetrieveRawPayload(ctx, store, request.RawRef)
	if err != nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, request.RawRef, "raw payload could not be retrieved and verified", err)
	}
	return pipeline.NormalizeBatch(ctx, NormalizationRequest{RawRef: request.RawRef, Bytes: payload, Target: request.Target, Normalizer: request.Normalizer})
}

// NormalizeBatch executes the accepted verification, exact routing, mapping,
// canonical validation, provenance validation, and immutable acceptance stages
// for every candidate in a bounded provider payload. Any child failure rejects
// the whole batch and returns a zero result.
func (pipeline *NormalizationPipeline) NormalizeBatch(ctx context.Context, request NormalizationRequest) (BatchNormalizationResult, error) {
	ref := request.RawRef
	if pipeline == nil || pipeline.providers == nil || pipeline.normalizers == nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnknownNormalizer, ref, "normalization pipeline is not initialized", nil)
	}
	if err := ctx.Err(); err != nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "context ended before raw verification", err)
	}
	if err := ref.Validate(); err != nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "raw payload reference is invalid", err)
	}
	if err := verifyRawPayloadBytes(ref, request.Bytes); err != nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRawVerification, NormalizationErrorRawContentVerification, ref, "exact raw bytes do not match the immutable reference", err)
	}

	capability, err := pipeline.providers.Capability(ref.Provider, ref.CapabilityID)
	if err != nil {
		return BatchNormalizationResult{}, mapRegistryNormalizationError(ref, err)
	}
	if capability.Support != SupportSupported {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedCapability, ref, "capability is not statically supported", nil)
	}
	if err := validateRawRepresentationMatch(capability.Raw, ref.Raw); err != nil {
		code := NormalizationErrorUnsupportedRawSchema
		if ref.Raw.Boundary == capability.Raw.Boundary && ref.Raw.Format == capability.Raw.Format && ref.Raw.Schema == capability.Raw.Schema {
			code = NormalizationErrorUnsupportedRepresentation
		}
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRouting, code, ref, "raw representation does not match the registered capability", err)
	}
	if err := validateNormalizationTarget(capability, request.Target); err != nil {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedTargetVersion, ref, "target canonical schema is not declared for the capability", err)
	}

	normalizer, descriptor, err := pipeline.normalizers.selectNormalizer(ref, request.Target, request.Normalizer)
	if err != nil {
		return BatchNormalizationResult{}, err
	}
	batchNormalizer, ok := normalizer.(BatchNormalizer)
	if !ok {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "registered normalizer does not support bounded batch output", nil)
	}
	input := NormalizationInput{RawRef: cloneRawPayloadRef(ref), Bytes: append([]byte(nil), request.Bytes...)}
	candidates, err := batchNormalizer.NormalizeBatch(ctx, input)
	if err != nil {
		return BatchNormalizationResult{}, normalizeBatchAdapterError(ref, err)
	}
	if len(candidates) == 0 || len(candidates) > maximumNormalizationBatchRecords {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "normalizer returned an empty or excessive batch", nil)
	}

	results := make([]NormalizationResult, 0, len(candidates))
	seen := make(map[canonical.ContractRef]struct{}, len(candidates))
	for _, candidate := range candidates {
		accepted, err := acceptBatchCandidate(ref, request.Target, descriptor, candidate)
		if err != nil {
			return BatchNormalizationResult{}, err
		}
		if _, exists := seen[accepted.Output.Contract]; exists {
			return BatchNormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorNonDeterministic, ref, "normalizer returned a duplicate canonical record identity", nil)
		}
		seen[accepted.Output.Contract] = struct{}{}
		results = append(results, accepted)
	}

	return BatchNormalizationResult{
		RawRef: cloneRawPayloadRef(ref), Normalizer: cloneComponentIdentity(descriptor.Component),
		Target: request.Target, Records: results,
	}, nil
}

// VerifyDeterministicBatchNormalization rejects any ordering, count,
// canonical-byte, immutable-output, or disposition change for identical input.
func VerifyDeterministicBatchNormalization(ctx context.Context, pipeline *NormalizationPipeline, request NormalizationRequest) (BatchNormalizationResult, error) {
	first, err := pipeline.NormalizeBatch(ctx, request)
	if err != nil {
		return BatchNormalizationResult{}, err
	}
	second, err := pipeline.NormalizeBatch(ctx, request)
	if err != nil {
		return BatchNormalizationResult{}, err
	}
	if len(first.Records) != len(second.Records) {
		return BatchNormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorNonDeterministic, request.RawRef, "identical raw input produced a different batch size", nil)
	}
	for i := range first.Records {
		left, leftErr := canonical.CanonicalContractBytes(first.Records[i].Record)
		right, rightErr := canonical.CanonicalContractBytes(second.Records[i].Record)
		if leftErr != nil || rightErr != nil || !bytes.Equal(left, right) || first.Records[i].Output != second.Records[i].Output || !reflect.DeepEqual(first.Records[i].Dispositions, second.Records[i].Dispositions) {
			return BatchNormalizationResult{}, normalizationError(NormalizationStageAcceptance, NormalizationErrorNonDeterministic, request.RawRef, "identical raw input and mapping identity produced different accepted batch output", errors.Join(leftErr, rightErr))
		}
	}
	return first, nil
}

func normalizeBatchAdapterError(ref RawPayloadRef, err error) error {
	if typed, ok := asNormalizationError(err); ok {
		copyError := *typed
		copyError.PayloadID = ref.ID
		copyError.ProviderID = ref.Provider.ID
		copyError.CapabilityID = ref.CapabilityID
		copyError.Detail = safeAdapterFailureDetail(copyError.Code)
		return &copyError
	}
	return normalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, ref, "normalizer failed without a typed provider-safe error", err)
}

func acceptBatchCandidate(ref RawPayloadRef, target canonical.ContractSchemaRef, descriptor NormalizerDescriptor, candidate NormalizationCandidate) (NormalizationResult, error) {
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
	if recordRef.Kind != target.Kind || recordRef.ContractVersion != target.Version {
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
		RawRef: cloneRawPayloadRef(ref), Normalizer: cloneComponentIdentity(descriptor.Component), Target: target,
		Output: output, Validation: NormalizationValidation{RawVerified: true, Parsed: true, Mapped: true, CanonicalValidated: true, ProvenanceValidated: true},
		Dispositions: append([]FieldDisposition(nil), candidate.Dispositions...), Record: candidate.Record,
	}, nil
}

func (result BatchNormalizationResult) Validate() error {
	if err := result.RawRef.Validate(); err != nil {
		return fmt.Errorf("batch normalization result: invalid raw reference: %w", err)
	}
	if err := result.Normalizer.Validate(); err != nil {
		return fmt.Errorf("batch normalization result: invalid normalizer: %w", err)
	}
	if err := result.Target.Validate(); err != nil {
		return fmt.Errorf("batch normalization result: invalid target: %w", err)
	}
	if len(result.Records) == 0 || len(result.Records) > maximumNormalizationBatchRecords {
		return fmt.Errorf("batch normalization result: record count is invalid")
	}
	for _, record := range result.Records {
		if record.Status != NormalizationStatusAccepted || record.Quality != NormalizationQualityValidated || record.RawRef.ID != result.RawRef.ID || !sameComponentIdentity(record.Normalizer, result.Normalizer) || record.Target != result.Target {
			return fmt.Errorf("batch normalization result: child acceptance envelope is inconsistent")
		}
	}
	return nil
}
