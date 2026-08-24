package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"sync"

	"jax-trading-assistant/libs/contracts/canonical"
)

type RawPayloadErrorCode string

const (
	RawPayloadErrorUnsupportedVersion    RawPayloadErrorCode = "unsupported_reference_version"
	RawPayloadErrorInvalidReference      RawPayloadErrorCode = "invalid_reference"
	RawPayloadErrorUnknownProvider       RawPayloadErrorCode = "unknown_provider"
	RawPayloadErrorProviderMismatch      RawPayloadErrorCode = "provider_identity_mismatch"
	RawPayloadErrorUndeclaredCapability  RawPayloadErrorCode = "undeclared_capability"
	RawPayloadErrorCapabilityUnavailable RawPayloadErrorCode = "capability_unavailable"
	RawPayloadErrorRawSchemaMismatch     RawPayloadErrorCode = "raw_schema_mismatch"
	RawPayloadErrorUnsupportedMediaType  RawPayloadErrorCode = "unsupported_media_type"
	RawPayloadErrorInvalidRetention      RawPayloadErrorCode = "invalid_retention_policy"
	RawPayloadErrorIncompletePayload     RawPayloadErrorCode = "incomplete_payload"
	RawPayloadErrorPersistenceFailed     RawPayloadErrorCode = "persistence_failed"
	RawPayloadErrorIdentityConflict      RawPayloadErrorCode = "duplicate_identity_conflict"
	RawPayloadErrorRetrievalMissing      RawPayloadErrorCode = "retrieval_missing"
	RawPayloadErrorDigestMismatch        RawPayloadErrorCode = "digest_mismatch"
	RawPayloadErrorSizeMismatch          RawPayloadErrorCode = "size_mismatch"
)

// RawPayloadError is safe for operator-visible failure handling. Detail must
// never contain payload bytes, credentials, request headers, or secrets.
type RawPayloadError struct {
	Code      RawPayloadErrorCode
	PayloadID RawPayloadID
	Detail    string
	Cause     error
}

func (err *RawPayloadError) Error() string {
	return fmt.Sprintf("raw payload %s: id=%q %s", err.Code, err.PayloadID, err.Detail)
}

func (err *RawPayloadError) Unwrap() error { return err.Cause }

// RawPayloadStore is the minimal immutable storage port. Put must make either
// the complete bytes available or return an error; it must never replace an
// existing acquisition identity with different reference meaning. Get must
// return the bytes associated with the exact typed reference.
type RawPayloadStore interface {
	Put(context.Context, RawPayloadRef, []byte) (RawPayloadLocation, error)
	Get(context.Context, RawPayloadRef) ([]byte, error)
}

// PersistRawPayload validates registry attribution, hashes the exact supplied
// bytes, persists them, reads them back, and verifies them before publishing a
// successful descriptor. A failed operation returns no accepted reference.
func PersistRawPayload(ctx context.Context, registry *Registry, store RawPayloadStore, request RawPayloadPersistenceRequest, payload []byte) (RawPayloadDescriptor, error) {
	if !request.Complete || len(payload) == 0 {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorIncompletePayload, request.ID, "payload must be complete and non-empty", nil)
	}
	if err := request.Retention.Validate(); err != nil {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorInvalidRetention, request.ID, "retention declaration is invalid", err)
	}
	capability, err := registryCapability(registry, request.ID, request.Provider, request.Capability)
	if err != nil {
		return RawPayloadDescriptor{}, err
	}
	if capability.Support != SupportSupported {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorCapabilityUnavailable, request.ID, "declared capability is not statically supported", nil)
	}
	if err := validateRawRepresentationMatch(capability.Raw, request.Raw); err != nil {
		return RawPayloadDescriptor{}, rawPayloadError(rawRepresentationErrorCode(capability.Raw, request.Raw), request.ID, "received representation does not match the declared capability", err)
	}

	ref := RawPayloadRef{
		ContractVersion: RawPayloadRefContractV1,
		ID:              request.ID,
		Content:         canonical.RawContentIdentity(payload),
		Provider:        cloneProviderIdentity(request.Provider),
		CapabilityID:    request.Capability,
		Raw:             request.Raw,
		Capture:         request.Capture,
		Source:          cloneSourceIdentity(request.Source),
		Revision:        cloneRevisionIdentity(request.Revision),
		ReceivedAt:      request.ReceivedAt,
		SizeBytes:       int64(len(payload)),
		Retention:       request.Retention,
	}
	if err := ref.Validate(); err != nil {
		code := RawPayloadErrorInvalidReference
		if ref.ContractVersion != RawPayloadRefContractV1 {
			code = RawPayloadErrorUnsupportedVersion
		}
		return RawPayloadDescriptor{}, rawPayloadError(code, request.ID, "reference metadata is invalid", err)
	}
	if store == nil {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorPersistenceFailed, request.ID, "raw payload store is required", nil)
	}
	location, err := store.Put(ctx, ref, payload)
	if err != nil {
		return RawPayloadDescriptor{}, normalizeStoreError(RawPayloadErrorPersistenceFailed, request.ID, "store write failed", err)
	}
	stored, err := store.Get(ctx, ref)
	if err != nil {
		return RawPayloadDescriptor{}, normalizeStoreError(RawPayloadErrorPersistenceFailed, request.ID, "stored payload could not be verified", err)
	}
	if err := verifyRawPayloadBytes(ref, stored); err != nil {
		return RawPayloadDescriptor{}, err
	}
	if !bytes.Equal(payload, stored) {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorDigestMismatch, request.ID, "stored bytes differ from supplied bytes", nil)
	}
	descriptor := RawPayloadDescriptor{Ref: ref, Location: location}
	if err := descriptor.Validate(); err != nil {
		return RawPayloadDescriptor{}, rawPayloadError(RawPayloadErrorPersistenceFailed, request.ID, "store returned an invalid logical location", err)
	}
	return descriptor, nil
}

// RetrieveRawPayload returns only bytes that match the immutable reference.
func RetrieveRawPayload(ctx context.Context, store RawPayloadStore, ref RawPayloadRef) ([]byte, error) {
	if err := ref.Validate(); err != nil {
		code := RawPayloadErrorInvalidReference
		if ref.ContractVersion != RawPayloadRefContractV1 {
			code = RawPayloadErrorUnsupportedVersion
		}
		return nil, rawPayloadError(code, ref.ID, "reference is invalid", err)
	}
	if store == nil {
		return nil, rawPayloadError(RawPayloadErrorRetrievalMissing, ref.ID, "raw payload store is required", nil)
	}
	payload, err := store.Get(ctx, ref)
	if err != nil {
		return nil, normalizeStoreError(RawPayloadErrorRetrievalMissing, ref.ID, "store retrieval failed", err)
	}
	if err := verifyRawPayloadBytes(ref, payload); err != nil {
		return nil, err
	}
	return append([]byte(nil), payload...), nil
}

// VerifyRawPayload verifies retrievability, size, and the accepted Phase 01
// exact-byte content identity without exposing the bytes to logs.
func VerifyRawPayload(ctx context.Context, store RawPayloadStore, ref RawPayloadRef) error {
	_, err := RetrieveRawPayload(ctx, store, ref)
	return err
}

func verifyRawPayloadBytes(ref RawPayloadRef, payload []byte) error {
	if int64(len(payload)) != ref.SizeBytes {
		return rawPayloadError(RawPayloadErrorSizeMismatch, ref.ID, "retrieved byte count does not match the reference", nil)
	}
	if err := ref.Content.Digest.VerifyBytes(payload); err != nil {
		return rawPayloadError(RawPayloadErrorDigestMismatch, ref.ID, "retrieved bytes failed content verification", err)
	}
	return nil
}

func registryCapability(registry *Registry, payloadID RawPayloadID, identity canonical.ProviderIdentity, capabilityID CapabilityID) (Capability, error) {
	if registry == nil {
		return Capability{}, rawPayloadError(RawPayloadErrorUnknownProvider, payloadID, "provider registry is required", nil)
	}
	capability, err := registry.Capability(identity, capabilityID)
	if err == nil {
		return capability, nil
	}
	var registryErr *RegistryError
	if !errors.As(err, &registryErr) {
		return Capability{}, rawPayloadError(RawPayloadErrorUnknownProvider, payloadID, "provider lookup failed", err)
	}
	switch registryErr.Code {
	case ErrorProviderIdentityMismatch, ErrorInvalidProviderIdentity:
		return Capability{}, rawPayloadError(RawPayloadErrorProviderMismatch, payloadID, "provider identity does not match the registry", err)
	case ErrorUnknownCapability:
		return Capability{}, rawPayloadError(RawPayloadErrorUndeclaredCapability, payloadID, "capability is not declared by the provider", err)
	default:
		return Capability{}, rawPayloadError(RawPayloadErrorUnknownProvider, payloadID, "provider is not registered", err)
	}
}

func validateRawRepresentationMatch(declared, received RawRepresentation) error {
	if err := received.Validate(); err != nil {
		return err
	}
	if received.Boundary != declared.Boundary || received.Format != declared.Format || received.Schema != declared.Schema {
		return fmt.Errorf("raw boundary, format, and schema must match the declaration")
	}
	if declared.MediaType == "" {
		return nil
	}
	declaredType, _, declaredErr := mime.ParseMediaType(declared.MediaType)
	receivedType, _, receivedErr := mime.ParseMediaType(received.MediaType)
	if declaredErr != nil || receivedErr != nil || declaredType != receivedType {
		return fmt.Errorf("media type must match declared type %q", declaredType)
	}
	return nil
}

func rawRepresentationErrorCode(declared, received RawRepresentation) RawPayloadErrorCode {
	if received.Boundary == declared.Boundary && received.Format == declared.Format && received.Schema == declared.Schema {
		return RawPayloadErrorUnsupportedMediaType
	}
	return RawPayloadErrorRawSchemaMismatch
}

func rawPayloadError(code RawPayloadErrorCode, id RawPayloadID, detail string, cause error) error {
	return &RawPayloadError{Code: code, PayloadID: id, Detail: detail, Cause: cause}
}

func normalizeStoreError(fallback RawPayloadErrorCode, id RawPayloadID, detail string, err error) error {
	var rawErr *RawPayloadError
	if errors.As(err, &rawErr) {
		return err
	}
	return rawPayloadError(fallback, id, detail, err)
}

func cloneSourceIdentity(source *canonical.SourceIdentity) *canonical.SourceIdentity {
	if source == nil {
		return nil
	}
	copySource := *source
	return &copySource
}

func cloneProviderIdentity(provider canonical.ProviderIdentity) canonical.ProviderIdentity {
	copyProvider := provider
	if provider.ExternalID != nil {
		externalID := *provider.ExternalID
		copyProvider.ExternalID = &externalID
	}
	return copyProvider
}

func cloneRevisionIdentity(revision *canonical.RevisionIdentity) *canonical.RevisionIdentity {
	if revision == nil {
		return nil
	}
	copyRevision := *revision
	return &copyRevision
}

type memoryRawPayloadRecord struct {
	ref      RawPayloadRef
	location RawPayloadLocation
}

// MemoryRawPayloadStore is a deterministic, concurrency-safe proof of the
// immutable storage policy. It is not a production durability backend.
type MemoryRawPayloadStore struct {
	mu      sync.RWMutex
	records map[RawPayloadID]memoryRawPayloadRecord
	blobs   map[string][]byte
}

func NewMemoryRawPayloadStore() *MemoryRawPayloadStore {
	return &MemoryRawPayloadStore{
		records: make(map[RawPayloadID]memoryRawPayloadRecord),
		blobs:   make(map[string][]byte),
	}
}

func (store *MemoryRawPayloadStore) Put(ctx context.Context, ref RawPayloadRef, payload []byte) (RawPayloadLocation, error) {
	if err := ctx.Err(); err != nil {
		return RawPayloadLocation{}, rawPayloadError(RawPayloadErrorPersistenceFailed, ref.ID, "context ended before write", err)
	}
	if store == nil {
		return RawPayloadLocation{}, rawPayloadError(RawPayloadErrorPersistenceFailed, ref.ID, "memory store is nil", nil)
	}
	if err := ref.Validate(); err != nil {
		return RawPayloadLocation{}, rawPayloadError(RawPayloadErrorInvalidReference, ref.ID, "reference is invalid", err)
	}
	if err := verifyRawPayloadBytes(ref, payload); err != nil {
		return RawPayloadLocation{}, err
	}
	location := RawPayloadLocation{
		Store: canonical.VersionIdentity{Namespace: "jax.raw_payload_store", Value: "memory/v1"},
		Key:   "sha256/" + ref.Content.Digest.Value,
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.records[ref.ID]; ok {
		if !rawPayloadRefsEqual(existing.ref, ref) {
			return RawPayloadLocation{}, rawPayloadError(RawPayloadErrorIdentityConflict, ref.ID, "existing acquisition identity has different immutable meaning", nil)
		}
		return existing.location, nil
	}
	if existingBlob, ok := store.blobs[ref.Content.Digest.Value]; ok && !bytes.Equal(existingBlob, payload) {
		return RawPayloadLocation{}, rawPayloadError(RawPayloadErrorDigestMismatch, ref.ID, "content digest aliases different bytes", nil)
	}
	if _, ok := store.blobs[ref.Content.Digest.Value]; !ok {
		store.blobs[ref.Content.Digest.Value] = append([]byte(nil), payload...)
	}
	store.records[ref.ID] = memoryRawPayloadRecord{ref: cloneRawPayloadRef(ref), location: location}
	return location, nil
}

func (store *MemoryRawPayloadStore) Get(ctx context.Context, ref RawPayloadRef) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, rawPayloadError(RawPayloadErrorRetrievalMissing, ref.ID, "context ended before retrieval", err)
	}
	if store == nil {
		return nil, rawPayloadError(RawPayloadErrorRetrievalMissing, ref.ID, "memory store is nil", nil)
	}
	if err := ref.Validate(); err != nil {
		return nil, rawPayloadError(RawPayloadErrorInvalidReference, ref.ID, "reference is invalid", err)
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.records[ref.ID]
	if !ok {
		return nil, rawPayloadError(RawPayloadErrorRetrievalMissing, ref.ID, "acquisition identity is not stored", nil)
	}
	if !rawPayloadRefsEqual(record.ref, ref) {
		return nil, rawPayloadError(RawPayloadErrorIdentityConflict, ref.ID, "stored acquisition reference does not match requested reference", nil)
	}
	payload, ok := store.blobs[ref.Content.Digest.Value]
	if !ok {
		return nil, rawPayloadError(RawPayloadErrorRetrievalMissing, ref.ID, "content bytes are not stored", nil)
	}
	return append([]byte(nil), payload...), nil
}

func rawPayloadRefsEqual(left, right RawPayloadRef) bool {
	return left.ContractVersion == right.ContractVersion && left.ID == right.ID && left.Content == right.Content &&
		left.Provider.ID == right.Provider.ID && left.Provider.Namespace == right.Provider.Namespace && externalIDsEqual(left.Provider.ExternalID, right.Provider.ExternalID) &&
		left.CapabilityID == right.CapabilityID && left.Raw == right.Raw && rawPayloadCapturesEqual(left.Capture, right.Capture) &&
		sourceIdentitiesEqual(left.Source, right.Source) && revisionIdentitiesEqual(left.Revision, right.Revision) &&
		left.ReceivedAt.Equal(right.ReceivedAt) && left.SizeBytes == right.SizeBytes && left.Retention == right.Retention
}

func rawPayloadCapturesEqual(left, right RawPayloadCapture) bool {
	if left.ByteForm != right.ByteForm || left.ContentCoding != right.ContentCoding || left.ContentCodingState != right.ContentCodingState || left.CharacterEncoding != right.CharacterEncoding {
		return false
	}
	if left.RelatedTo == nil || right.RelatedTo == nil {
		return left.RelatedTo == nil && right.RelatedTo == nil
	}
	return *left.RelatedTo == *right.RelatedTo
}

func externalIDsEqual(left, right *canonical.ExternalID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sourceIdentitiesEqual(left, right *canonical.SourceIdentity) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func revisionIdentitiesEqual(left, right *canonical.RevisionIdentity) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func cloneRawPayloadRef(ref RawPayloadRef) RawPayloadRef {
	copyRef := ref
	copyRef.Provider = cloneProviderIdentity(ref.Provider)
	copyRef.Source = cloneSourceIdentity(ref.Source)
	copyRef.Revision = cloneRevisionIdentity(ref.Revision)
	if ref.Capture.RelatedTo != nil {
		relation := *ref.Capture.RelatedTo
		copyRef.Capture.RelatedTo = &relation
	}
	return copyRef
}
