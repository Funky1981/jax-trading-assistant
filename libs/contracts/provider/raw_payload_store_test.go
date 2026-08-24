package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

func TestPersistRetrieveVerifyAndContentDeduplication(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	store := NewMemoryRawPayloadStore()
	payload := []byte(`{"same":"content"}`)
	firstRequest := validRawPayloadRequest(definition, "rpa_acquisition_one", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	secondRequest := validRawPayloadRequest(definition, "rpa_acquisition_two", time.Date(2026, 8, 24, 10, 5, 0, 0, time.UTC))

	first, err := PersistRawPayload(context.Background(), registry, store, firstRequest, payload)
	if err != nil {
		t.Fatalf("first PersistRawPayload() error = %v", err)
	}
	second, err := PersistRawPayload(context.Background(), registry, store, secondRequest, payload)
	if err != nil {
		t.Fatalf("second PersistRawPayload() error = %v", err)
	}
	if first.Ref.ID == second.Ref.ID || first.Ref.ReceivedAt.Equal(second.Ref.ReceivedAt) {
		t.Fatal("identical content collapsed distinct acquisition history")
	}
	if first.Ref.Content != second.Ref.Content || first.Location != second.Location {
		t.Fatal("identical content was not content-deduplicated")
	}
	if len(store.records) != 2 || len(store.blobs) != 1 {
		t.Fatalf("records=%d blobs=%d, want 2 acquisitions and 1 blob", len(store.records), len(store.blobs))
	}
	retrieved, err := RetrieveRawPayload(context.Background(), store, first.Ref)
	if err != nil {
		t.Fatalf("RetrieveRawPayload() error = %v", err)
	}
	if string(retrieved) != string(payload) {
		t.Fatalf("retrieved payload = %q", retrieved)
	}
	retrieved[0] = 'X'
	if err := VerifyRawPayload(context.Background(), store, first.Ref); err != nil {
		t.Fatalf("caller mutation changed stored bytes: %v", err)
	}
}

func TestDuplicateAcquisitionIsIdempotentAndConflictsOnChangedContent(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	store := NewMemoryRawPayloadStore()
	request := validRawPayloadRequest(definition, "rpa_idempotent", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	payload := []byte(`{"value":1}`)
	first, err := PersistRawPayload(context.Background(), registry, store, request, payload)
	if err != nil {
		t.Fatalf("first PersistRawPayload() error = %v", err)
	}
	second, err := PersistRawPayload(context.Background(), registry, store, request, payload)
	if err != nil {
		t.Fatalf("idempotent PersistRawPayload() error = %v", err)
	}
	if !rawPayloadRefsEqual(first.Ref, second.Ref) || first.Location != second.Location {
		t.Fatal("idempotent write did not return the existing immutable result")
	}
	_, err = PersistRawPayload(context.Background(), registry, store, request, []byte(`{"value":2}`))
	assertRawPayloadErrorCode(t, err, RawPayloadErrorIdentityConflict)
}

func TestTamperedOrMissingStoredBytesFailVerification(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	store := NewMemoryRawPayloadStore()
	request := validRawPayloadRequest(definition, "rpa_tamper", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	descriptor, err := PersistRawPayload(context.Background(), registry, store, request, []byte("original"))
	if err != nil {
		t.Fatalf("PersistRawPayload() error = %v", err)
	}
	store.blobs[descriptor.Ref.Content.Digest.Value] = []byte("tampered")
	assertRawPayloadErrorCode(t, VerifyRawPayload(context.Background(), store, descriptor.Ref), RawPayloadErrorDigestMismatch)

	missing := cloneRawPayloadRef(descriptor.Ref)
	missing.ID = "rpa_missing"
	assertRawPayloadErrorCode(t, VerifyRawPayload(context.Background(), store, missing), RawPayloadErrorRetrievalMissing)
}

func TestPersistenceFailsClosedForRegistryAndRepresentationMismatch(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	base := validRawPayloadRequest(definition, "rpa_mismatch", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	payload := []byte("payload")

	tests := []struct {
		name string
		edit func(*RawPayloadPersistenceRequest)
		code RawPayloadErrorCode
	}{
		{"unknown_provider", func(request *RawPayloadPersistenceRequest) { request.Provider.ID = "pvd_unknown" }, RawPayloadErrorUnknownProvider},
		{"provider_identity_mismatch", func(request *RawPayloadPersistenceRequest) { request.Provider.Namespace = "other.provider" }, RawPayloadErrorProviderMismatch},
		{"undeclared_capability", func(request *RawPayloadPersistenceRequest) { request.Capability = CapabilityNewsArticle }, RawPayloadErrorUndeclaredCapability},
		{"raw_schema_mismatch", func(request *RawPayloadPersistenceRequest) { request.Raw.Schema.Value = "other/v1" }, RawPayloadErrorRawSchemaMismatch},
		{"unsupported_media_type", func(request *RawPayloadPersistenceRequest) { request.Raw.MediaType = "text/plain" }, RawPayloadErrorUnsupportedMediaType},
		{"incomplete", func(request *RawPayloadPersistenceRequest) { request.Complete = false }, RawPayloadErrorIncompletePayload},
		{"invalid_retention", func(request *RawPayloadPersistenceRequest) { request.Retention.Class = "TEMPORARY" }, RawPayloadErrorInvalidRetention},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := base
			request.Provider.ExternalID = cloneExternalID(base.Provider.ExternalID)
			request.Source = cloneSourceIdentity(base.Source)
			request.Revision = cloneRevisionIdentity(base.Revision)
			test.edit(&request)
			_, err := PersistRawPayload(context.Background(), registry, NewMemoryRawPayloadStore(), request, payload)
			assertRawPayloadErrorCode(t, err, test.code)
		})
	}
}

func TestPersistenceRejectsKnownButUnavailableCapability(t *testing.T) {
	registry := mustRegistry(t)
	definition := validDefinition(CapabilityCorporateFiling)
	definition.Capabilities[0].Support = SupportUnavailable
	if err := registry.Register(definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	request := validRawPayloadRequest(definition, "rpa_unavailable", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	_, err := PersistRawPayload(context.Background(), registry, NewMemoryRawPayloadStore(), request, []byte("payload"))
	assertRawPayloadErrorCode(t, err, RawPayloadErrorCapabilityUnavailable)
}

func TestPersistenceAcceptsMediaTypeParametersForDeclaredBaseType(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	request := validRawPayloadRequest(definition, "rpa_media_parameter", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	request.Raw.MediaType = "application/json; charset=utf-8"
	if _, err := PersistRawPayload(context.Background(), registry, NewMemoryRawPayloadStore(), request, []byte("payload")); err != nil {
		t.Fatalf("PersistRawPayload() rejected declared base media type parameters: %v", err)
	}
}

func TestPersistenceDoesNotPublishReferenceOnWriteOrIncompleteStoreFailure(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	request := validRawPayloadRequest(definition, "rpa_failed_write", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))

	descriptor, err := PersistRawPayload(context.Background(), registry, failingRawPayloadStore{}, request, []byte("payload"))
	if descriptor.Ref.ID != "" {
		t.Fatalf("failed write published reference %#v", descriptor)
	}
	assertRawPayloadErrorCode(t, err, RawPayloadErrorPersistenceFailed)

	descriptor, err = PersistRawPayload(context.Background(), registry, truncatingRawPayloadStore{}, request, []byte("payload"))
	if descriptor.Ref.ID != "" {
		t.Fatalf("truncated write published reference %#v", descriptor)
	}
	assertRawPayloadErrorCode(t, err, RawPayloadErrorSizeMismatch)
}

func TestUnsupportedReferenceVersionFailsBeforeRetrieval(t *testing.T) {
	_, definition := rawPayloadRegistry(t)
	ref := rawPayloadRefFromRequest(validRawPayloadRequest(definition, "rpa_version", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)), []byte("payload"))
	ref.ContractVersion = "jax.provider_raw_payload_ref/v2"
	assertRawPayloadErrorCode(t, VerifyRawPayload(context.Background(), NewMemoryRawPayloadStore(), ref), RawPayloadErrorUnsupportedVersion)
}

type failingRawPayloadStore struct{}

func (failingRawPayloadStore) Put(context.Context, RawPayloadRef, []byte) (RawPayloadLocation, error) {
	return RawPayloadLocation{}, errors.New("deterministic write failure")
}

func (failingRawPayloadStore) Get(context.Context, RawPayloadRef) ([]byte, error) {
	return nil, errors.New("not stored")
}

type truncatingRawPayloadStore struct{}

func (truncatingRawPayloadStore) Put(_ context.Context, _ RawPayloadRef, payload []byte) (RawPayloadLocation, error) {
	return validTestLocation(), nil
}

func (truncatingRawPayloadStore) Get(_ context.Context, _ RawPayloadRef) ([]byte, error) {
	return []byte("payloa"), nil
}

func validTestLocation() RawPayloadLocation {
	return RawPayloadLocation{Store: canonicalVersion("jax.raw_payload_store", "test/v1"), Key: "sha256/test"}
}

func canonicalVersion(namespace, value string) canonical.VersionIdentity {
	return canonical.VersionIdentity{Namespace: namespace, Value: value}
}

func cloneExternalID(value *canonical.ExternalID) *canonical.ExternalID {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func assertRawPayloadErrorCode(t *testing.T, err error, want RawPayloadErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	var rawErr *RawPayloadError
	if !errors.As(err, &rawErr) {
		t.Fatalf("error type = %T, want *RawPayloadError: %v", err, err)
	}
	if rawErr.Code != want {
		t.Fatalf("error code = %q, want %q: %v", rawErr.Code, want, err)
	}
}
