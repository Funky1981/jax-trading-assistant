package provider

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

func TestRawPayloadIdentityHashesExactReceivedBytes(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	store := NewMemoryRawPayloadStore()
	request := validRawPayloadRequest(definition, "rpa_exact_bytes", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))

	first := []byte("{\r\n  \"value\": 1\r\n}")
	second := []byte("{\n\"value\":1\n}")
	descriptor, err := PersistRawPayload(context.Background(), registry, store, request, first)
	if err != nil {
		t.Fatalf("PersistRawPayload() error = %v", err)
	}
	if descriptor.Ref.Content != canonical.RawContentIdentity(first) {
		t.Fatalf("content identity = %#v, want exact-byte identity", descriptor.Ref.Content)
	}
	if descriptor.Ref.Content == canonical.RawContentIdentity(second) {
		t.Fatal("whitespace/line-ending change did not change content identity")
	}
	if descriptor.Ref.SizeBytes != int64(len(first)) {
		t.Fatalf("size_bytes = %d, want %d", descriptor.Ref.SizeBytes, len(first))
	}
}

func TestRawPayloadReferenceValidationFailsClosed(t *testing.T) {
	_, definition := rawPayloadRegistry(t)
	request := validRawPayloadRequest(definition, "rpa_validation", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	ref := rawPayloadRefFromRequest(request, []byte("payload"))

	tests := []struct {
		name string
		edit func(*RawPayloadRef)
	}{
		{"unsupported_version", func(candidate *RawPayloadRef) { candidate.ContractVersion = "jax.provider_raw_payload_ref/v99" }},
		{"invalid_id", func(candidate *RawPayloadRef) { candidate.ID = "raw_payload" }},
		{"canonical_content", func(candidate *RawPayloadRef) {
			candidate.Content.Representation = canonical.ContentRepresentationCanonicalJSON
			candidate.Content.Canonicalization = canonical.CanonicalJSONIdentityV1
		}},
		{"unknown_capability", func(candidate *RawPayloadRef) { candidate.CapabilityID = "vendor.method" }},
		{"non_utc_receipt", func(candidate *RawPayloadRef) {
			candidate.ReceivedAt = time.Date(2026, 8, 24, 11, 0, 0, 0, time.FixedZone("BST", 3600))
		}},
		{"invalid_size", func(candidate *RawPayloadRef) { candidate.SizeBytes = 0 }},
		{"invalid_retention", func(candidate *RawPayloadRef) { candidate.Retention.Class = "TEMPORARY" }},
		{"unknown_redistribution", func(candidate *RawPayloadRef) { candidate.Retention.Redistribution = "UNKNOWN" }},
		{"self_relation", func(candidate *RawPayloadRef) {
			candidate.Capture.RelatedTo = &RawPayloadRelation{Kind: RawPayloadRelationDecodedFrom, PayloadID: candidate.ID}
			candidate.Capture.ContentCoding = "gzip"
			candidate.Capture.ContentCodingState = ContentCodingDecoded
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneRawPayloadRef(ref)
			test.edit(&candidate)
			assertProviderValidationError(t, candidate.Validate())
		})
	}
}

func TestRawPayloadAndReplayManifestIdentityNamespacesAreDistinct(t *testing.T) {
	const replayManifestPrefix = "rpl_"
	if RawPayloadIDPrefix == replayManifestPrefix {
		t.Fatalf("raw-payload prefix %q collides with ReplayManifest", RawPayloadIDPrefix)
	}

	reservedPrefixes := map[string]string{
		"instrument":               "ins_",
		"issuer":                   "iss_",
		"event":                    "evt_",
		"evidence":                 "evd_",
		"observation":              "obs_",
		"research_run":             "run_",
		"quant_result":             "qnt_",
		"recommendation":           "rec_",
		"source":                   "src_",
		"provider":                 "pvd_",
		"dataset":                  "dset_",
		"dataset_snapshot":         "dss_",
		"component":                "cmp_",
		"provenance":               "pvn_",
		"compatibility_assessment": "cpa_",
		"audit_event":              "aud_",
		"audit_stream":             "ast_",
		"audit_idempotency":        "adi_",
		"correlation":              "cor_",
		"failure":                  "fail_",
		"replay_manifest":          replayManifestPrefix,
	}
	for identity, prefix := range reservedPrefixes {
		if RawPayloadIDPrefix == prefix {
			t.Fatalf("raw-payload prefix %q collides with %s", RawPayloadIDPrefix, identity)
		}
	}

	replay := canonical.ReplayManifest{
		ContractVersion: canonical.ReplayManifestContractV1,
		ID:              "rpl_namespace_probe",
	}
	var canonicalErr *canonical.ValidationError
	if err := replay.Validate(); !errors.As(err, &canonicalErr) || canonicalErr.Field == "id" {
		t.Fatalf("ReplayManifest did not accept its rpl_ identity namespace: %v", err)
	}
	replay.ID = canonical.ReplayManifestID(RawPayloadIDPrefix + "namespace_probe")
	canonicalErr = nil
	if err := replay.Validate(); !errors.As(err, &canonicalErr) || canonicalErr.Field != "id" {
		t.Fatalf("ReplayManifest accepted raw-payload identity namespace %q: %v", RawPayloadIDPrefix, err)
	}

	_, definition := rawPayloadRegistry(t)
	ref := rawPayloadRefFromRequest(validRawPayloadRequest(definition, RawPayloadID(RawPayloadIDPrefix+"namespace_probe"), time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)), []byte("payload"))
	if err := ref.Validate(); err != nil {
		t.Fatalf("RawPayloadRef did not accept its %s identity namespace: %v", RawPayloadIDPrefix, err)
	}
	ref.ID = "rpl_namespace_probe"
	var providerErr *ValidationError
	if err := ref.Validate(); !errors.As(err, &providerErr) || providerErr.Field != "id" {
		t.Fatalf("RawPayloadRef accepted ReplayManifest identity namespace: %v", err)
	}
}

func TestRawPayloadCaptureMakesTransportRepresentationExplicit(t *testing.T) {
	encoded := RawPayloadCapture{
		ByteForm:           RawPayloadByteFormEntityBody,
		ContentCoding:      "gzip",
		ContentCodingState: ContentCodingEncoded,
	}
	if err := encoded.Validate(); err != nil {
		t.Fatalf("encoded capture Validate() error = %v", err)
	}
	decoded := RawPayloadCapture{
		ByteForm:           RawPayloadByteFormEntityBody,
		ContentCoding:      "gzip",
		ContentCodingState: ContentCodingDecoded,
		CharacterEncoding:  "utf-8",
		RelatedTo:          &RawPayloadRelation{Kind: RawPayloadRelationDecodedFrom, PayloadID: "rpa_encoded"},
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded capture Validate() error = %v", err)
	}
	part := RawPayloadCapture{ByteForm: RawPayloadByteFormMultipartPart, ContentCodingState: ContentCodingIdentity}
	assertProviderValidationError(t, part.Validate())
	entityClaimingPart := RawPayloadCapture{
		ByteForm:           RawPayloadByteFormEntityBody,
		ContentCodingState: ContentCodingIdentity,
		RelatedTo:          &RawPayloadRelation{Kind: RawPayloadRelationPartOf, PayloadID: "rpa_multipart"},
	}
	assertProviderValidationError(t, entityClaimingPart.Validate())
}

func TestRawPayloadCanBridgeToImmutableEvidenceAndProvenance(t *testing.T) {
	registry, definition := rawPayloadRegistry(t)
	request := validRawPayloadRequest(definition, "rpa_evidence", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC))
	descriptor, err := PersistRawPayload(context.Background(), registry, NewMemoryRawPayloadStore(), request, []byte(`{"filing":"exact"}`))
	if err != nil {
		t.Fatalf("PersistRawPayload() error = %v", err)
	}
	evidence, err := descriptor.Ref.AsEvidenceRef(canonical.ContractRef{
		Kind:            canonical.ContractKindEvidence,
		ID:              "evd_raw_payload",
		ContractVersion: canonical.EvidenceContractV2,
	})
	if err != nil {
		t.Fatalf("AsEvidenceRef() error = %v", err)
	}
	if evidence.Content != descriptor.Ref.Content || evidence.Provider == nil || !sameProviderIdentity(*evidence.Provider, descriptor.Ref.Provider) || evidence.Source != *descriptor.Ref.Source {
		t.Fatalf("evidence bridge lost raw identity: %#v", evidence)
	}
	input := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidence}
	if err := input.Validate(); err != nil {
		t.Fatalf("raw-backed lineage input Validate() error = %v", err)
	}
	if _, err := canonical.ComputeInputFingerprint([]canonical.LineageInput{input}); err != nil {
		t.Fatalf("ComputeInputFingerprint() error = %v", err)
	}

	withoutSource := cloneRawPayloadRef(descriptor.Ref)
	withoutSource.Source = nil
	if _, err := withoutSource.AsEvidenceRef(evidence.Evidence); err == nil {
		t.Fatal("AsEvidenceRef() fabricated a missing logical source")
	}
}

func TestRawPayloadReferenceJSONIsStrictAndStable(t *testing.T) {
	_, definition := rawPayloadRegistry(t)
	ref := rawPayloadRefFromRequest(validRawPayloadRequest(definition, "rpa_json", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)), []byte(`{"value":1}`))
	first, err := EncodeRawPayloadRefJSON(ref)
	if err != nil {
		t.Fatalf("EncodeRawPayloadRefJSON() error = %v", err)
	}
	second, err := EncodeRawPayloadRefJSON(ref)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("reference JSON is not stable: %v", err)
	}
	var decoded RawPayloadRef
	if err := DecodeRawPayloadRefJSON(first, &decoded); err != nil {
		t.Fatalf("DecodeRawPayloadRefJSON() error = %v", err)
	}
	if !rawPayloadRefsEqual(decoded, ref) {
		t.Fatalf("round trip mismatch:\nwant %#v\n got %#v", ref, decoded)
	}

	withSecret := bytes.Replace(first, []byte(`"provider":`), []byte(`"authorization":"secret","provider":`), 1)
	duplicateID := bytes.Replace(first, []byte(`"id":`), []byte(`"id":"rpa_duplicate","id":`), 1)
	withNull := bytes.Replace(first, []byte(`"capture":`), []byte(`"capture":null,"ignored_capture":`), 1)
	unsupported := bytes.Replace(first, []byte(string(RawPayloadRefContractV1)), []byte("jax.provider_raw_payload_ref/v99"), 1)
	for name, raw := range map[string][]byte{
		"secret_unknown_field": withSecret,
		"duplicate_field":      duplicateID,
		"null_or_unknown":      withNull,
		"trailing_value":       append(append([]byte(nil), first...), []byte(` {}`)...),
		"unsupported_version":  unsupported,
		"invalid_utf8":         append(append([]byte(nil), first...), 0xff),
	} {
		t.Run(name, func(t *testing.T) {
			var candidate RawPayloadRef
			if err := DecodeRawPayloadRefJSON(raw, &candidate); err == nil {
				t.Fatal("DecodeRawPayloadRefJSON() accepted invalid input")
			}
		})
	}
	if err := DecodeRawPayloadRefJSON(first, nil); err == nil {
		t.Fatal("DecodeRawPayloadRefJSON() accepted nil destination")
	}
}

func rawPayloadRegistry(t *testing.T) (*Registry, ProviderDefinition) {
	t.Helper()
	registry := mustRegistry(t)
	definition := validDefinition(CapabilityCorporateFiling)
	if err := registry.Register(definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return registry, definition
}

func validRawPayloadRequest(definition ProviderDefinition, id RawPayloadID, receivedAt time.Time) RawPayloadPersistenceRequest {
	source := canonical.SourceIdentity{ID: "src_sec_edgar", Kind: canonical.SourceKindRegulator}
	revision := canonical.RevisionIdentity{Namespace: "sec.accession", Value: "0000000000-26-000001"}
	return RawPayloadPersistenceRequest{
		ID:         id,
		Provider:   definition.Identity,
		Capability: CapabilityCorporateFiling,
		Raw:        definition.Capabilities[0].Raw,
		Capture: RawPayloadCapture{
			ByteForm:           RawPayloadByteFormEntityBody,
			ContentCodingState: ContentCodingIdentity,
			CharacterEncoding:  "utf-8",
		},
		Source:     &source,
		Revision:   &revision,
		ReceivedAt: receivedAt,
		Retention: RawPayloadRetentionPolicy{
			Class:          RawPayloadRetentionReplayAudit,
			Policy:         canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"},
			Redistribution: RawPayloadRedistributionNotAuthorized,
		},
		Complete: true,
	}
}

func rawPayloadRefFromRequest(request RawPayloadPersistenceRequest, payload []byte) RawPayloadRef {
	return RawPayloadRef{
		ContractVersion: RawPayloadRefContractV1,
		ID:              request.ID,
		Content:         canonical.RawContentIdentity(payload),
		Provider:        request.Provider,
		CapabilityID:    request.Capability,
		Raw:             request.Raw,
		Capture:         request.Capture,
		Source:          request.Source,
		Revision:        request.Revision,
		ReceivedAt:      request.ReceivedAt,
		SizeBytes:       int64(len(payload)),
		Retention:       request.Retention,
	}
}

func assertProviderValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("validation accepted invalid value")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *ValidationError: %v", err, err)
	}
}

func TestCloneRawPayloadRefDefensivelyCopiesPointers(t *testing.T) {
	_, definition := rawPayloadRegistry(t)
	ref := rawPayloadRefFromRequest(validRawPayloadRequest(definition, "rpa_clone", time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)), []byte("payload"))
	clone := cloneRawPayloadRef(ref)
	clone.Source.ID = "src_changed"
	clone.Revision.Value = "changed"
	clone.Provider.ExternalID.Value = "changed"
	if reflect.DeepEqual(ref, clone) || ref.Source.ID == clone.Source.ID || ref.Revision.Value == clone.Revision.Value || ref.Provider.ExternalID.Value == clone.Provider.ExternalID.Value {
		t.Fatal("clone shares mutable pointer fields")
	}
}
