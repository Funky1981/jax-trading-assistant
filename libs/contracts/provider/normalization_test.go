package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

var (
	syntheticReceivedAt = time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	syntheticPayload    = []byte(`{"series":"GDP","observed_at":"2026-08-24T07:30:00-04:00","value":2.75,"unit":"percent","value_kind":"point","provider_status":"released","display_precision":2}`)
)

type syntheticMacroRecord struct {
	Series           string      `json:"series"`
	ObservedAt       string      `json:"observed_at"`
	Value            json.Number `json:"value"`
	Unit             string      `json:"unit"`
	ValueKind        string      `json:"value_kind"`
	ProviderStatus   string      `json:"provider_status,omitempty"`
	DisplayPrecision *int        `json:"display_precision,omitempty"`
}

type syntheticIdentityResolver interface {
	ResolveObservation(string) (canonical.ContractRef, canonical.SourceReference, error)
}

type fixedSyntheticResolver struct {
	fail bool
}

func (resolver fixedSyntheticResolver) ResolveObservation(series string) (canonical.ContractRef, canonical.SourceReference, error) {
	if resolver.fail {
		return canonical.ContractRef{}, canonical.SourceReference{}, errors.New("synthetic identity catalogue unavailable")
	}
	if series != "GDP" {
		return canonical.ContractRef{}, canonical.SourceReference{}, fmt.Errorf("series is not mapped")
	}
	return canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_us_gdp_series", ContractVersion: canonical.InstrumentContractV1}, canonical.SourceReference{
		ID: "src_synthetic_macro", Kind: canonical.SourceKindDataset,
		ExternalID: &canonical.ExternalID{Namespace: "synthetic.series", Value: series},
	}, nil
}

type syntheticMacroNormalizer struct {
	descriptor NormalizerDescriptor
	resolver   syntheticIdentityResolver
}

func (normalizer *syntheticMacroNormalizer) Descriptor() NormalizerDescriptor {
	return cloneNormalizerDescriptor(normalizer.descriptor)
}

func (normalizer *syntheticMacroNormalizer) Normalize(_ context.Context, input NormalizationInput) (NormalizationCandidate, error) {
	parsed, err := parseSyntheticMacro(input.Bytes)
	if err != nil {
		return NormalizationCandidate{}, err
	}
	if input.RawRef.Source == nil || input.RawRef.Revision == nil {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorIdentityResolution, "raw source and revision are required; source identity was not inferred", nil)
	}
	subject, source, err := normalizer.resolver.ResolveObservation(parsed.Series)
	if err != nil {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorIdentityResolution, "canonical subject/source identity resolution failed", err)
	}
	observedAt, err := time.Parse(time.RFC3339Nano, parsed.ObservedAt)
	if err != nil || observedAt.Year() < 0 || observedAt.Year() > 9999 {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorInvalidProviderValue, "observed_at is not an RFC3339 timestamp", err)
	}
	observedAt = observedAt.UTC()
	if observedAt.After(input.RawRef.ReceivedAt) {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorInvalidProviderValue, "observed_at must not follow raw receipt time", nil)
	}
	value, err := strconv.ParseFloat(string(parsed.Value), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorInvalidProviderValue, "value must be a finite number", err)
	}
	if parsed.ValueKind == "range" {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorAmbiguousProviderValue, "value_kind range cannot support a canonical point observation", nil)
	}
	if parsed.ValueKind != "point" {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorUnmappableProviderValue, "value_kind has no deterministic canonical mapping", nil)
	}
	unit, ok := mapSyntheticUnit(parsed.Unit)
	if !ok {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorUnmappableProviderValue, "unit has no deterministic canonical mapping", nil)
	}

	identitySeed := strings.Join([]string{string(input.RawRef.ID), input.RawRef.Content.Digest.Value, normalizer.descriptor.Component.ID, normalizer.descriptor.Component.Version.Namespace, normalizer.descriptor.Component.Version.Value, parsed.Series, observedAt.Format(time.RFC3339Nano)}, "\x00")
	digest := canonical.DigestBytes([]byte(identitySeed)).Value
	observationID := canonical.ObservationID("obs_" + digest[:24])
	evidenceID := canonical.EvidenceID("evd_" + digest[24:48])
	evidenceRef, err := input.RawRef.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorIdentityResolution, "exact raw evidence identity could not be constructed", err)
	}
	evidenceRef.ObservedAt = &observedAt
	lineage := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}
	fingerprint, err := canonical.ComputeInputFingerprint([]canonical.LineageInput{lineage})
	if err != nil {
		return NormalizationCandidate{}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorProvenanceValidation, "raw lineage fingerprint could not be constructed", err)
	}
	provenance := canonical.Provenance{
		ContractVersion: canonical.ProvenanceContractV1,
		ID:              "pvn_" + digest[:24],
		Inputs:          []canonical.LineageInput{lineage}, InputFingerprint: fingerprint,
		Producer: cloneComponentIdentity(normalizer.descriptor.Component),
	}
	observation := canonical.Observation{
		ContractVersion: canonical.ObservationContractV2,
		ID:              observationID, Type: canonical.ObservationTypeMacroValue, Subject: subject,
		Metric: "gross_domestic_product_growth", Value: canonical.ObservedValue{Type: canonical.ObservedValueTypeNumber, Number: &value, Unit: unit},
		Source: source, EvidenceIDs: []canonical.EvidenceID{evidenceID}, ObservedAt: observedAt,
		CollectedAt: input.RawRef.ReceivedAt, CreatedAt: input.RawRef.ReceivedAt, Provenance: &provenance,
	}
	dispositions := []FieldDisposition{
		{ProviderField: "series", Status: FieldDispositionRepresented, CanonicalField: "subject"},
		{ProviderField: "observed_at", Status: FieldDispositionRepresented, CanonicalField: "observed_at"},
		{ProviderField: "value", Status: FieldDispositionRepresented, CanonicalField: "value.number"},
		{ProviderField: "unit", Status: FieldDispositionRepresented, CanonicalField: "value.unit"},
		{ProviderField: "value_kind", Status: FieldDispositionRepresented, CanonicalField: "value.type"},
	}
	if parsed.ProviderStatus != "" {
		dispositions = append(dispositions, FieldDisposition{ProviderField: "provider_status", Status: FieldDispositionIntentionallyOmitted, ReasonCode: "provider_only_release_state"})
	}
	if parsed.DisplayPrecision != nil {
		dispositions = append(dispositions, FieldDisposition{ProviderField: "display_precision", Status: FieldDispositionUnsupportedUnmappable, ReasonCode: "canonical_value_preserves_numeric_meaning"})
	}
	return NormalizationCandidate{
		Record:       observation,
		Revision:     canonical.RevisionIdentity{Namespace: "jax.normalized_record", Value: "v1/" + digest},
		Dispositions: dispositions,
	}, nil
}

func parseSyntheticMacro(raw []byte) (syntheticMacroRecord, error) {
	if err := rejectDuplicateObjectProperties(raw); err != nil {
		return syntheticMacroRecord{}, adapterNormalizationError(NormalizationStageParsing, NormalizationErrorParserFailure, "provider JSON is structurally invalid", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var parsed syntheticMacroRecord
	if err := decoder.Decode(&parsed); err != nil {
		return syntheticMacroRecord{}, adapterNormalizationError(NormalizationStageParsing, NormalizationErrorParserFailure, "provider JSON could not be decoded", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return syntheticMacroRecord{}, adapterNormalizationError(NormalizationStageParsing, NormalizationErrorParserFailure, "provider JSON contains trailing data", err)
	}
	for field, value := range map[string]string{
		"series": parsed.Series, "observed_at": parsed.ObservedAt, "value": string(parsed.Value), "unit": parsed.Unit, "value_kind": parsed.ValueKind,
	} {
		if strings.TrimSpace(value) == "" {
			return syntheticMacroRecord{}, adapterNormalizationError(NormalizationStageParsing, NormalizationErrorRequiredFieldMissing, field+" is required", nil)
		}
	}
	return parsed, nil
}

func mapSyntheticUnit(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "percent":
		return "percent", true
	case "index_points":
		return "index_points", true
	default:
		return "", false
	}
}

func adapterNormalizationError(stage NormalizationStage, code NormalizationErrorCode, detail string, cause error) error {
	return &NormalizationError{Stage: stage, Code: code, Detail: detail, Cause: cause}
}

type testNormalizer struct {
	descriptor NormalizerDescriptor
	fn         func(context.Context, NormalizationInput) (NormalizationCandidate, error)
}

func (normalizer *testNormalizer) Descriptor() NormalizerDescriptor {
	return cloneNormalizerDescriptor(normalizer.descriptor)
}
func (normalizer *testNormalizer) Normalize(ctx context.Context, input NormalizationInput) (NormalizationCandidate, error) {
	return normalizer.fn(ctx, input)
}

type normalizationFixture struct {
	providers   *Registry
	normalizers *NormalizerRegistry
	pipeline    *NormalizationPipeline
	store       *MemoryRawPayloadStore
	definition  ProviderDefinition
	descriptor  NormalizerDescriptor
	normalizer  *syntheticMacroNormalizer
	request     NormalizationRequest
}

func newNormalizationFixture(t *testing.T, payload []byte) normalizationFixture {
	t.Helper()
	providers, err := NewRegistry(RegistryContractV1)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	definition := validDefinition(CapabilityMacroObservation)
	definition.Identity = canonical.ProviderIdentity{ID: "pvd_synthetic_macro", Namespace: "synthetic.macro"}
	definition.DisplayName = "Synthetic Macro Fixture"
	definition.AdapterVersion = canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}
	definition.Capabilities[0].Raw.Schema = canonical.VersionIdentity{Namespace: "synthetic.macro.schema", Value: "observation/v1"}
	definition.Capabilities[0].Authentication.Class = AuthenticationNone
	if err := providers.Register(definition); err != nil {
		t.Fatalf("Register(provider) error = %v", err)
	}
	componentProvider := definition.Identity
	mappingContent := canonical.RawContentIdentity([]byte("synthetic-macro-observation-normalizer/v1"))
	descriptor := NormalizerDescriptor{
		ContractVersion: NormalizerDescriptorV1, Provider: definition.Identity, CapabilityID: CapabilityMacroObservation,
		Raw:       definition.Capabilities[0].Raw,
		Component: canonical.ComponentIdentity{ID: "cmp_synthetic_macro_observation", Kind: canonical.ComponentKindNormalizer, Name: "synthetic macro observation normalizer", Version: canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}, Provider: &componentProvider, Content: &mappingContent},
		Target:    canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2},
	}
	normalizer := &syntheticMacroNormalizer{descriptor: descriptor, resolver: fixedSyntheticResolver{}}
	normalizers, err := NewNormalizerRegistry(providers)
	if err != nil {
		t.Fatalf("NewNormalizerRegistry() error = %v", err)
	}
	if err := normalizers.Register(normalizer); err != nil {
		t.Fatalf("Register(normalizer) error = %v", err)
	}
	pipeline, err := NewNormalizationPipeline(providers, normalizers)
	if err != nil {
		t.Fatalf("NewNormalizationPipeline() error = %v", err)
	}
	store := NewMemoryRawPayloadStore()
	source := canonical.SourceIdentity{ID: "src_synthetic_macro", Kind: canonical.SourceKindDataset}
	revision := canonical.RevisionIdentity{Namespace: "synthetic.release", Value: "2026-08-24T11:30:00Z"}
	descriptorRaw, err := PersistRawPayload(context.Background(), providers, store, RawPayloadPersistenceRequest{
		ID: "rpa_synthetic_macro_20260824", Provider: definition.Identity, Capability: CapabilityMacroObservation,
		Raw: definition.Capabilities[0].Raw, Capture: RawPayloadCapture{ByteForm: RawPayloadByteFormEntityBody, ContentCodingState: ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source: &source, Revision: &revision, ReceivedAt: syntheticReceivedAt,
		Retention: RawPayloadRetentionPolicy{Class: RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: RawPayloadRedistributionNotAuthorized}, Complete: true,
	}, payload)
	if err != nil {
		t.Fatalf("PersistRawPayload() error = %v", err)
	}
	verified, err := RetrieveRawPayload(context.Background(), store, descriptorRaw.Ref)
	if err != nil {
		t.Fatalf("RetrieveRawPayload() error = %v", err)
	}
	return normalizationFixture{providers: providers, normalizers: normalizers, pipeline: pipeline, store: store, definition: definition, descriptor: descriptor, normalizer: normalizer, request: NormalizationRequest{RawRef: descriptorRaw.Ref, Bytes: verified, Target: descriptor.Target, Normalizer: descriptor.Component}}
}

func TestNormalizationPipelineRepresentativeEndToEnd(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	stored, err := fixture.pipeline.NormalizeStored(context.Background(), fixture.store, StoredNormalizationRequest{RawRef: fixture.request.RawRef, Target: fixture.request.Target, Normalizer: fixture.request.Normalizer})
	if err != nil {
		t.Fatalf("NormalizeStored() error = %v", err)
	}
	if stored.Status != NormalizationStatusAccepted {
		t.Fatalf("NormalizeStored() status = %q", stored.Status)
	}
	result, err := VerifyDeterministicNormalization(context.Background(), fixture.pipeline, fixture.request)
	if err != nil {
		t.Fatalf("VerifyDeterministicNormalization() error = %v", err)
	}
	if result.Status != NormalizationStatusAccepted || result.Quality != NormalizationQualityValidated {
		t.Fatalf("result status/quality = %q/%q", result.Status, result.Quality)
	}
	if result.RawRef.ID != fixture.request.RawRef.ID || result.RawRef.Content != fixture.request.RawRef.Content {
		t.Fatal("accepted result does not retain exact raw input identity")
	}
	if !sameComponentIdentity(result.Normalizer, fixture.descriptor.Component) || result.Target != fixture.descriptor.Target {
		t.Fatal("accepted result does not retain exact normalizer and target schema")
	}
	if result.Validation != (NormalizationValidation{RawVerified: true, Parsed: true, Mapped: true, CanonicalValidated: true, ProvenanceValidated: true}) {
		t.Fatalf("validation stages = %+v", result.Validation)
	}
	observation, ok := result.Record.(canonical.Observation)
	if !ok {
		t.Fatalf("record type = %T", result.Record)
	}
	if observation.ObservedAt != time.Date(2026, 8, 24, 11, 30, 0, 0, time.UTC) {
		t.Fatalf("observed_at = %s; timestamp offset was not explicitly converted to UTC", observation.ObservedAt)
	}
	if observation.Provenance == nil || !sameComponentIdentity(observation.Provenance.Producer, fixture.descriptor.Component) {
		t.Fatal("normalizer component is absent from canonical provenance")
	}
	if err := result.Output.Content.VerifyCanonicalContract(result.Record); err != nil {
		t.Fatalf("output content identity verification = %v", err)
	}
	if err := VerifyRawPayload(context.Background(), fixture.store, result.RawRef); err != nil {
		t.Fatalf("raw payload is not retrievable/verifiable = %v", err)
	}
	if len(result.Dispositions) != 7 || result.Dispositions[5].Status != FieldDispositionIntentionallyOmitted || result.Dispositions[6].Status != FieldDispositionUnsupportedUnmappable {
		t.Fatalf("loss/omission dispositions = %+v", result.Dispositions)
	}
}

func TestNormalizeStoredRejectsMissingRawPayload(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	result, err := fixture.pipeline.NormalizeStored(context.Background(), NewMemoryRawPayloadStore(), StoredNormalizationRequest{RawRef: fixture.request.RawRef, Target: fixture.request.Target, Normalizer: fixture.request.Normalizer})
	assertNormalizationFailure(t, result, err, NormalizationErrorRawContentVerification)
}

func TestNormalizerDescriptorStrictStableJSON(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	first, err := EncodeNormalizerDescriptorJSON(fixture.descriptor)
	if err != nil {
		t.Fatalf("EncodeNormalizerDescriptorJSON() error = %v", err)
	}
	second, err := EncodeNormalizerDescriptorJSON(fixture.descriptor)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("descriptor JSON is not stable: %v", err)
	}
	var decoded NormalizerDescriptor
	if err := DecodeNormalizerDescriptorJSON(first, &decoded); err != nil {
		t.Fatalf("DecodeNormalizerDescriptorJSON() error = %v", err)
	}
	if !normalizerDescriptorsEqual(decoded, fixture.descriptor) {
		t.Fatalf("descriptor round trip changed meaning: %+v", decoded)
	}
	unknown := bytes.Replace(first, []byte(`"target":`), []byte(`"credential_value":"secret","target":`), 1)
	if err := DecodeNormalizerDescriptorJSON(unknown, &decoded); err == nil {
		t.Fatal("descriptor JSON accepted secret-bearing unknown field")
	}
	duplicate := bytes.Replace(first, []byte(`{"contract_version":`), []byte(`{"contract_version":"jax.provider_normalizer/v1","contract_version":`), 1)
	if err := DecodeNormalizerDescriptorJSON(duplicate, &decoded); err == nil {
		t.Fatal("descriptor JSON accepted duplicate field")
	}
	unsupported := bytes.Replace(first, []byte(string(NormalizerDescriptorV1)), []byte("jax.provider_normalizer/v99"), 1)
	for name, invalid := range map[string][]byte{
		"null":                []byte("null"),
		"invalid_utf8":        {0xff, 0xfe},
		"trailing_value":      append(append([]byte(nil), first...), []byte(` {}`)...),
		"unsupported_version": unsupported,
	} {
		t.Run(name, func(t *testing.T) {
			if err := DecodeNormalizerDescriptorJSON(invalid, &decoded); err == nil {
				t.Fatal("descriptor JSON accepted invalid input")
			}
		})
	}
}

func TestNormalizationPipelineFailsClosedForRawAndRoutingMismatch(t *testing.T) {
	base := newNormalizationFixture(t, syntheticPayload)
	tests := []struct {
		name   string
		mutate func(*NormalizationRequest, *normalizationFixture)
		code   NormalizationErrorCode
	}{
		{"tampered_raw_bytes", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.Bytes = append([]byte(nil), request.Bytes...)
			request.Bytes[0] ^= 1
		}, NormalizationErrorRawContentVerification},
		{"missing_raw_bytes", func(request *NormalizationRequest, _ *normalizationFixture) { request.Bytes = nil }, NormalizationErrorRawContentVerification},
		{"wrong_provider_identity", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.RawRef.Provider.Namespace = "synthetic.other"
		}, NormalizationErrorUnsupportedProvider},
		{"wrong_capability", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.RawRef.CapabilityID = CapabilityMarketQuote
		}, NormalizationErrorUnsupportedCapability},
		{"wrong_raw_schema", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.RawRef.Raw.Schema.Value = "observation/v2"
		}, NormalizationErrorUnsupportedRawSchema},
		{"wrong_media_representation", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.RawRef.Raw.MediaType = "application/xml"
		}, NormalizationErrorUnsupportedRepresentation},
		{"target_contract_version", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.Target.Version = canonical.ObservationContractV1
		}, NormalizationErrorUnsupportedTargetVersion},
		{"normalizer_version", func(request *NormalizationRequest, _ *normalizationFixture) {
			request.Normalizer.Version.Value = "2.0.0"
		}, NormalizationErrorNormalizerVersionMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := base
			request := fixture.request
			request.RawRef = cloneRawPayloadRef(fixture.request.RawRef)
			request.Normalizer = cloneComponentIdentity(fixture.request.Normalizer)
			test.mutate(&request, &fixture)
			result, err := fixture.pipeline.Normalize(context.Background(), request)
			assertNormalizationFailure(t, result, err, test.code)
		})
	}

	emptyRegistry, err := NewNormalizerRegistry(base.providers)
	if err != nil {
		t.Fatal(err)
	}
	emptyPipeline, err := NewNormalizationPipeline(base.providers, emptyRegistry)
	if err != nil {
		t.Fatal(err)
	}
	result, err := emptyPipeline.Normalize(context.Background(), base.request)
	assertNormalizationFailure(t, result, err, NormalizationErrorUnknownNormalizer)
}

func TestNormalizerRegistryRejectsDuplicateOrAmbiguousRoute(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	duplicate := &syntheticMacroNormalizer{descriptor: fixture.descriptor, resolver: fixedSyntheticResolver{}}
	err := fixture.normalizers.Register(duplicate)
	assertNormalizationErrorCode(t, err, NormalizationErrorAmbiguousNormalizer)

	changed := fixture.descriptor
	changed.Component.Version.Value = "2.0.0"
	changedNormalizer := &syntheticMacroNormalizer{descriptor: changed, resolver: fixedSyntheticResolver{}}
	err = fixture.normalizers.Register(changedNormalizer)
	assertNormalizationErrorCode(t, err, NormalizationErrorAmbiguousNormalizer)
}

func TestNormalizationPipelineDistinguishesParserAndMappingFailures(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		resolver syntheticIdentityResolver
		code     NormalizationErrorCode
	}{
		{"malformed", []byte(`{"series":`), fixedSyntheticResolver{}, NormalizationErrorParserFailure},
		{"missing_required", []byte(`{"series":"GDP","observed_at":"2026-08-24T11:30:00Z","unit":"percent","value_kind":"point"}`), fixedSyntheticResolver{}, NormalizationErrorRequiredFieldMissing},
		{"invalid_value", []byte(`{"series":"GDP","observed_at":"not-a-time","value":2.0,"unit":"percent","value_kind":"point"}`), fixedSyntheticResolver{}, NormalizationErrorInvalidProviderValue},
		{"ambiguous_value", []byte(`{"series":"GDP","observed_at":"2026-08-24T11:30:00Z","value":2.0,"unit":"percent","value_kind":"range"}`), fixedSyntheticResolver{}, NormalizationErrorAmbiguousProviderValue},
		{"unmappable_enum", []byte(`{"series":"GDP","observed_at":"2026-08-24T11:30:00Z","value":2.0,"unit":"percent","value_kind":"categorical"}`), fixedSyntheticResolver{}, NormalizationErrorUnmappableProviderValue},
		{"unmappable_unit", []byte(`{"series":"GDP","observed_at":"2026-08-24T11:30:00Z","value":2.0,"unit":"mystery","value_kind":"point"}`), fixedSyntheticResolver{}, NormalizationErrorUnmappableProviderValue},
		{"identity_resolution", syntheticPayload, fixedSyntheticResolver{fail: true}, NormalizationErrorIdentityResolution},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newNormalizationFixture(t, test.payload)
			fixture.normalizer.resolver = test.resolver
			result, err := fixture.pipeline.Normalize(context.Background(), fixture.request)
			assertNormalizationFailure(t, result, err, test.code)
		})
	}
}

func TestNormalizationPipelineRejectsCanonicalProvenanceLossAndPartialFailures(t *testing.T) {
	tests := []struct {
		name          string
		mutate        func(*NormalizationCandidate)
		returnedError error
		code          NormalizationErrorCode
	}{
		{"canonical_validation", func(candidate *NormalizationCandidate) {
			observation := candidate.Record.(canonical.Observation)
			observation.Metric = ""
			candidate.Record = observation
		}, nil, NormalizationErrorCanonicalValidation},
		{"provenance_mismatch", func(candidate *NormalizationCandidate) {
			observation := candidate.Record.(canonical.Observation)
			observation.Provenance.Producer.Version.Value = "wrong"
			candidate.Record = observation
		}, nil, NormalizationErrorProvenanceValidation},
		{"loss_inconsistency", func(candidate *NormalizationCandidate) {
			candidate.Dispositions = append(candidate.Dispositions, candidate.Dispositions[0])
		}, nil, NormalizationErrorLossMetadata},
		{"partial_failure_cannot_succeed", func(_ *NormalizationCandidate) {}, adapterNormalizationError(NormalizationStageMapping, NormalizationErrorCanonicalConstruction, "forced failure after candidate construction", nil), NormalizationErrorCanonicalConstruction},
		{"unsafe_error_detail_is_suppressed", func(_ *NormalizationCandidate) {}, adapterNormalizationError(NormalizationStageParsing, NormalizationErrorParserFailure, string(syntheticPayload), nil), NormalizationErrorParserFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newNormalizationFixture(t, syntheticPayload)
			base := fixture.normalizer
			wrapped := &testNormalizer{descriptor: fixture.descriptor, fn: func(ctx context.Context, input NormalizationInput) (NormalizationCandidate, error) {
				candidate, err := base.Normalize(ctx, input)
				if err != nil {
					return NormalizationCandidate{}, err
				}
				test.mutate(&candidate)
				return candidate, test.returnedError
			}}
			normalizers, err := NewNormalizerRegistry(fixture.providers)
			if err != nil {
				t.Fatal(err)
			}
			if err := normalizers.Register(wrapped); err != nil {
				t.Fatal(err)
			}
			pipeline, err := NewNormalizationPipeline(fixture.providers, normalizers)
			if err != nil {
				t.Fatal(err)
			}
			result, err := pipeline.Normalize(context.Background(), fixture.request)
			assertNormalizationFailure(t, result, err, test.code)
		})
	}
}

func TestNormalizationPipelineRejectsProvenanceSourceFabrication(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	request := fixture.request
	request.RawRef = cloneRawPayloadRef(request.RawRef)
	request.RawRef.Source = nil
	result, err := fixture.pipeline.Normalize(context.Background(), request)
	assertNormalizationFailure(t, result, err, NormalizationErrorIdentityResolution)
}

func TestVerifyDeterministicNormalizationRejectsChangingOutput(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	base := fixture.normalizer
	calls := 0
	toggling := &testNormalizer{descriptor: fixture.descriptor, fn: func(ctx context.Context, input NormalizationInput) (NormalizationCandidate, error) {
		candidate, err := base.Normalize(ctx, input)
		if err != nil {
			return NormalizationCandidate{}, err
		}
		calls++
		observation := candidate.Record.(canonical.Observation)
		if calls%2 == 1 {
			observation.ID = "obs_determinism_a"
			observation.Provenance.ID = "pvn_determinism_a"
			candidate.Revision.Value = "v1/determinism-a"
		} else {
			observation.ID = "obs_determinism_b"
			observation.Provenance.ID = "pvn_determinism_b"
			candidate.Revision.Value = "v1/determinism-b"
		}
		candidate.Record = observation
		return candidate, nil
	}}
	normalizers, err := NewNormalizerRegistry(fixture.providers)
	if err != nil {
		t.Fatal(err)
	}
	if err := normalizers.Register(toggling); err != nil {
		t.Fatal(err)
	}
	pipeline, err := NewNormalizationPipeline(fixture.providers, normalizers)
	if err != nil {
		t.Fatal(err)
	}
	result, err := VerifyDeterministicNormalization(context.Background(), pipeline, fixture.request)
	assertNormalizationFailure(t, result, err, NormalizationErrorNonDeterministic)
}

func assertNormalizationFailure(t *testing.T, result NormalizationResult, err error, code NormalizationErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("Normalize() succeeded; want %s", code)
	}
	if result.Status != "" || result.Record != nil || result.Output.Contract.ID != "" {
		t.Fatalf("failure returned partially accepted result: %+v", result)
	}
	assertNormalizationErrorCode(t, err, code)
}

func assertNormalizationErrorCode(t *testing.T, err error, code NormalizationErrorCode) {
	t.Helper()
	var normalizationErr *NormalizationError
	if !errors.As(err, &normalizationErr) {
		t.Fatalf("error type = %T, want *NormalizationError: %v", err, err)
	}
	if normalizationErr.Code != code {
		t.Fatalf("error code = %q, want %q: %v", normalizationErr.Code, code, err)
	}
	if strings.Contains(normalizationErr.Error(), string(syntheticPayload)) {
		t.Fatal("operator-visible error leaked raw payload")
	}
}
