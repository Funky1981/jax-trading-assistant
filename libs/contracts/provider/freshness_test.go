package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type freshnessProof struct {
	fixture   normalizationFixture
	policies  *FreshnessPolicyRegistry
	evaluator *FreshnessEvaluator
	policy    FreshnessPolicy
}

func newFreshnessProof(t *testing.T) freshnessProof {
	t.Helper()
	fixture := newNormalizationFixture(t, syntheticPayload)
	policies := NewFreshnessPolicyRegistry()
	policy := syntheticFreshnessPolicy("cmp_macro_freshness_policy", "1.0.0", "cmp_macro_lkg_policy", "1.0.0", FallbackFreshOrStale, 2*time.Hour)
	if err := policies.Register(policy); err != nil {
		t.Fatalf("Register(freshness policy) error = %v", err)
	}
	evaluator, err := NewFreshnessEvaluator(fixture.providers, policies)
	if err != nil {
		t.Fatalf("NewFreshnessEvaluator() error = %v", err)
	}
	return freshnessProof{fixture: fixture, policies: policies, evaluator: evaluator, policy: policy}
}

func syntheticFreshnessPolicy(policyID, policyVersion, lkgID, lkgVersion string, mode FallbackMode, maximumAge time.Duration) FreshnessPolicy {
	return FreshnessPolicy{
		ContractVersion: FreshnessPolicyContractV1,
		Identity:        canonical.ComponentIdentity{ID: policyID, Kind: canonical.ComponentKindPolicy, Name: "synthetic macro freshness policy", Version: canonical.VersionIdentity{Namespace: "semver", Value: policyVersion}},
		CapabilityID:    CapabilityMacroObservation,
		Target:          canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2},
		UseClass:        DataUseResearch, ValidityMode: FreshnessValidityAgeBounded, TimestampRole: TimestampRoleObservedAt,
		FreshFor: 30 * time.Minute, ExpireAfter: 4 * time.Hour, AllowedFutureSkew: 5 * time.Second, MissingTimestamp: MissingTimestampFail,
		LastKnownGood: LastKnownGoodPolicy{
			ContractVersion: LastKnownGoodPolicyContractV1,
			Identity:        canonical.ComponentIdentity{ID: lkgID, Kind: canonical.ComponentKindPolicy, Name: "synthetic macro last-known-good policy", Version: canonical.VersionIdentity{Namespace: "semver", Value: lkgVersion}},
			Mode:            mode, MaximumAge: maximumAge,
		},
	}
}

func (proof freshnessProof) temporalObservation(t *testing.T, id string, observedAt, receivedAt time.Time, value float64) TemporalRecord {
	t.Helper()
	payload := []byte(fmt.Sprintf(`{"series":"GDP","observed_at":%q,"value":%g,"unit":"percent","value_kind":"point"}`, observedAt.Format(time.RFC3339Nano), value))
	source := canonical.SourceIdentity{ID: "src_synthetic_macro", Kind: canonical.SourceKindDataset}
	revision := canonical.RevisionIdentity{Namespace: "synthetic.release", Value: id}
	descriptor, err := PersistRawPayload(context.Background(), proof.fixture.providers, proof.fixture.store, RawPayloadPersistenceRequest{
		ID: RawPayloadID("rpa_" + id), Provider: proof.fixture.definition.Identity, Capability: CapabilityMacroObservation,
		Raw:     proof.fixture.definition.Capabilities[0].Raw,
		Capture: RawPayloadCapture{ByteForm: RawPayloadByteFormEntityBody, ContentCodingState: ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source:  &source, Revision: &revision, ReceivedAt: receivedAt,
		Retention: RawPayloadRetentionPolicy{Class: RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: RawPayloadRedistributionNotAuthorized},
		Complete:  true,
	}, payload)
	if err != nil {
		t.Fatalf("PersistRawPayload(%s) error = %v", id, err)
	}
	result, err := proof.fixture.pipeline.NormalizeStored(context.Background(), proof.fixture.store, StoredNormalizationRequest{
		RawRef: descriptor.Ref, Target: proof.fixture.descriptor.Target, Normalizer: proof.fixture.descriptor.Component,
	})
	if err != nil {
		t.Fatalf("NormalizeStored(%s) error = %v", id, err)
	}
	observation := result.Record.(canonical.Observation)
	return TemporalRecord{
		Normalized: result,
		Key:        FreshnessKey{CapabilityID: CapabilityMacroObservation, Target: result.Target, Subject: observation.Subject, Qualifier: observation.Metric},
		Lifecycle:  TemporalRecordLifecycle{State: TemporalRecordActive},
	}
}

func (proof freshnessProof) evaluate(t *testing.T, record TemporalRecord, at time.Time) FreshnessEvaluation {
	t.Helper()
	evaluation, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{
		Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at,
		Context: FreshnessContextCurrentState, Record: record,
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	return evaluation
}

func (proof freshnessProof) withPriorFresh(t *testing.T, record TemporalRecord, at time.Time) TemporalRecord {
	t.Helper()
	prior := proof.evaluate(t, record, at)
	if prior.State != TemporalFresh {
		t.Fatalf("prior state = %q, want FRESH", prior.State)
	}
	record.PriorFreshEvaluation = &prior
	return record
}

func TestFreshnessEvaluatorRepresentativeObservationV2Proof(t *testing.T) {
	proof := newFreshnessProof(t)
	observedAt := time.Date(2026, 8, 24, 11, 30, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "freshness_boundary", observedAt, observedAt.Add(5*time.Minute), 2.75)
	if observation, ok := record.Normalized.Record.(canonical.Observation); !ok || observation.ContractVersion != canonical.ObservationContractV2 || observation.Provenance == nil {
		t.Fatal("proof does not begin with validated provenance-bearing Observation V2")
	}

	freshAt := observedAt.Add(30*time.Minute - time.Nanosecond)
	fresh := proof.evaluate(t, record, freshAt)
	if fresh.State != TemporalFresh || fresh.Age == nil || *fresh.Age != 30*time.Minute-time.Nanosecond || fresh.TimestampRole != TimestampRoleObservedAt {
		t.Fatalf("fresh evaluation = %+v", fresh)
	}
	boundary := proof.evaluate(t, record, observedAt.Add(30*time.Minute))
	if boundary.State != TemporalStale || boundary.ReasonCode != FreshnessReasonAtOrBeyondTTL {
		t.Fatalf("TTL boundary = %+v; age == TTL must be STALE", boundary)
	}
	expiry := proof.evaluate(t, record, observedAt.Add(4*time.Hour))
	if expiry.State != TemporalExpired || expiry.ReasonCode != FreshnessReasonAtOrBeyondExpiry {
		t.Fatalf("expiry boundary = %+v; age == expiry must be EXPIRED", expiry)
	}

	first := proof.evaluate(t, record, observedAt.Add(time.Hour))
	second := proof.evaluate(t, record, observedAt.Add(time.Hour))
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same immutable fixture, policy, and evaluation time produced a different result")
	}
	if first.Record != record.Normalized.Output || first.RawPayloadID != record.Normalized.RawRef.ID || !samePolicyIdentity(first.Policy, proof.policy.Identity) {
		t.Fatal("evaluation omitted immutable record/raw/policy identity")
	}
}

func TestFreshnessUsesCanonicalObservedTimeNotRecentRawReceipt(t *testing.T) {
	proof := newFreshnessProof(t)
	evaluationTime := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "recent_receipt_old_observation", evaluationTime.Add(-2*time.Hour), evaluationTime.Add(-time.Minute), 2.75)
	evaluation := proof.evaluate(t, record, evaluationTime)
	if evaluation.State != TemporalStale || evaluation.Age == nil || *evaluation.Age != 2*time.Hour {
		t.Fatalf("evaluation = %+v; recent raw receipt must not refresh old canonical observation", evaluation)
	}
	if record.Normalized.RawRef.ReceivedAt.Sub(*evaluation.AuthoritativeTimestamp) != 119*time.Minute {
		t.Fatal("fixture did not prove distinct raw receipt and canonical observation clocks")
	}
}

func TestFreshnessRejectsInvalidCanonicalProvenanceAndSemanticKey(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "invalid_dimensions", at.Add(-time.Hour), at.Add(-55*time.Minute), 2.75)

	invalidCanonical := record
	observation := invalidCanonical.Normalized.Record.(canonical.Observation)
	observation.Metric = ""
	invalidCanonical.Normalized.Record = observation
	_, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: invalidCanonical})
	requireFreshnessCode(t, err, FreshnessErrorInvalidCanonicalInput)

	invalidProvenance := record
	invalidProvenance.Normalized.RawRef.Source = &canonical.SourceIdentity{ID: "src_other_macro", Kind: canonical.SourceKindDataset}
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: invalidProvenance})
	requireFreshnessCode(t, err, FreshnessErrorInvalidProvenanceInput)

	wrongKey := record
	wrongKey.Key.Qualifier = "consumer_price_index"
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: wrongKey})
	requireFreshnessCode(t, err, FreshnessErrorSemanticKeyMismatch)
}

func TestFreshnessCanUseExplicitDatasetVintageAuthority(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "dataset_vintage", at.Add(-time.Hour), at.Add(-50*time.Minute), 2.75)
	asOf := at.Add(-24 * time.Hour)
	dataset := canonical.DatasetSnapshotRef{
		ContractVersion: canonical.DatasetSnapshotContractV1,
		Dataset:         canonical.DatasetIdentity{ID: "dset_synthetic_gdp", ExternalID: canonical.ExternalID{Namespace: "synthetic.dataset", Value: "gdp"}, SchemaVersion: canonical.VersionIdentity{Namespace: "synthetic.dataset.schema", Value: "v1"}},
		SnapshotID:      "dss_synthetic_gdp_20260823", Revision: canonical.RevisionIdentity{Namespace: "synthetic.vintage", Value: "2026-08-23"},
		Content: canonical.RawContentIdentity([]byte("synthetic-gdp-vintage-2026-08-23")), AsOf: &asOf, CollectedAt: at.Add(-23 * time.Hour),
	}
	observation := record.Normalized.Record.(canonical.Observation)
	observation.Provenance.Inputs = append(observation.Provenance.Inputs, canonical.LineageInput{Kind: canonical.LineageInputKindDataset, Dataset: &dataset})
	fingerprint, err := canonical.ComputeInputFingerprint(observation.Provenance.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	observation.Provenance.InputFingerprint = fingerprint
	record.Normalized.Record = observation
	record.Normalized.Output.Content, err = canonical.CanonicalContractContentIdentity(observation)
	if err != nil {
		t.Fatal(err)
	}

	policy := syntheticFreshnessPolicy("cmp_dataset_vintage", "1.1.0", "cmp_dataset_vintage_lkg", "1.1.0", FallbackFreshOrStale, 72*time.Hour)
	policy.TimestampRole = TimestampRoleDatasetAsOf
	policy.FreshFor = 48 * time.Hour
	policy.ExpireAfter = 96 * time.Hour
	if err := proof.policies.Register(policy); err != nil {
		t.Fatal(err)
	}
	evaluation, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	if err != nil || evaluation.State != TemporalFresh || evaluation.AuthoritativeTimestamp == nil || !evaluation.AuthoritativeTimestamp.Equal(asOf) || evaluation.Age == nil || *evaluation.Age != 24*time.Hour {
		t.Fatalf("dataset-vintage evaluation = %+v, %v", evaluation, err)
	}

	otherAsOf := asOf.Add(-24 * time.Hour)
	other := dataset
	other.SnapshotID = "dss_synthetic_gdp_20260822"
	other.Revision.Value = "2026-08-22"
	other.Content = canonical.RawContentIdentity([]byte("synthetic-gdp-vintage-2026-08-22"))
	other.AsOf = &otherAsOf
	other.CollectedAt = at.Add(-47 * time.Hour)
	observation.Provenance.Inputs = append(observation.Provenance.Inputs, canonical.LineageInput{Kind: canonical.LineageInputKindDataset, Dataset: &other})
	fingerprint, err = canonical.ComputeInputFingerprint(observation.Provenance.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	observation.Provenance.InputFingerprint = fingerprint
	record.Normalized.Record = observation
	record.Normalized.Output.Content, err = canonical.CanonicalContractContentIdentity(observation)
	if err != nil {
		t.Fatal(err)
	}
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorAmbiguousAuthoritativeTimestamp)
}

func TestFreshnessMissingTimestampEvaluationTimeAndPolicyVersionFailures(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "missing_published", at.Add(-time.Hour), at.Add(-50*time.Minute), 2.75)

	failPolicy := syntheticFreshnessPolicy("cmp_macro_published_fail", "1.0.1", "cmp_macro_published_fail_lkg", "1.0.1", FallbackFreshOrStale, 2*time.Hour)
	failPolicy.TimestampRole = TimestampRolePublishedAt
	if err := proof.policies.Register(failPolicy); err != nil {
		t.Fatal(err)
	}
	_, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: failPolicy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorMissingAuthoritativeTimestamp)

	unknownPolicy := syntheticFreshnessPolicy("cmp_macro_published_unknown", "1.0.2", "cmp_macro_published_unknown_lkg", "1.0.2", FallbackFreshOrStale, 2*time.Hour)
	unknownPolicy.TimestampRole = TimestampRolePublishedAt
	unknownPolicy.MissingTimestamp = MissingTimestampUnknown
	if err := proof.policies.Register(unknownPolicy); err != nil {
		t.Fatal(err)
	}
	unknown, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: unknownPolicy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	if err != nil || unknown.State != TemporalUnknown || unknown.Age != nil {
		t.Fatalf("missing timestamp UNKNOWN evaluation = %+v, %v", unknown, err)
	}

	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorInvalidEvaluationTime)
	nonUTC := time.Date(2026, 8, 24, 13, 0, 0, 0, time.FixedZone("BST", 3600))
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: nonUTC, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorInvalidEvaluationTime)

	wrongVersion := cloneComponentIdentity(proof.policy.Identity)
	wrongVersion.Version.Value = "2.0.0"
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: wrongVersion, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorPolicyVersionMismatch)
}

func TestFreshnessPolicyValidationAndRegistryRejectInvalidOrAmbiguousDeclarations(t *testing.T) {
	registry := NewFreshnessPolicyRegistry()
	zeroTTL := syntheticFreshnessPolicy("cmp_zero_ttl", "3.0.0", "cmp_zero_ttl_lkg", "3.0.0", FallbackFreshOrStale, time.Hour)
	zeroTTL.FreshFor = 0
	requireFreshnessCode(t, registry.Register(zeroTTL), FreshnessErrorInvalidTTL)
	negativeTTL := syntheticFreshnessPolicy("cmp_negative_ttl", "3.0.1", "cmp_negative_ttl_lkg", "3.0.1", FallbackFreshOrStale, time.Hour)
	negativeTTL.FreshFor = -time.Second
	requireFreshnessCode(t, registry.Register(negativeTTL), FreshnessErrorInvalidTTL)
	unsupported := syntheticFreshnessPolicy("cmp_bad_contract", "3.0.2", "cmp_bad_contract_lkg", "3.0.2", FallbackProhibited, 0)
	unsupported.ContractVersion = "jax.provider_freshness_policy/v99"
	requireFreshnessCode(t, registry.Register(unsupported), FreshnessErrorUnsupportedPolicyVersion)

	valid := syntheticFreshnessPolicy("cmp_registry_policy", "3.1.0", "cmp_registry_lkg", "3.1.0", FallbackFreshOrStale, 2*time.Hour)
	if err := registry.Register(valid); err != nil {
		t.Fatal(err)
	}
	requireFreshnessCode(t, registry.Register(valid), FreshnessErrorDuplicatePolicy)
	ambiguous := valid
	ambiguous.Identity.ID = "cmp_registry_other"
	ambiguous.LastKnownGood.Identity.ID = "cmp_registry_other_lkg"
	requireFreshnessCode(t, registry.Register(ambiguous), FreshnessErrorAmbiguousPolicy)

	mismatch := valid
	mismatch.Identity.ID = "cmp_registry_capability_mismatch"
	mismatch.Identity.Version.Value = "3.2.0"
	mismatch.LastKnownGood.Identity.ID = "cmp_registry_capability_mismatch_lkg"
	mismatch.LastKnownGood.Identity.Version.Value = "3.2.0"
	mismatch.CapabilityID = CapabilityInstrumentReference
	requireFreshnessCode(t, registry.Register(mismatch), FreshnessErrorPolicyCapabilityMismatch)
}

func TestLastKnownGoodSelectsValidHistoricalRecordWithoutQualityUpgrade(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	valid := proof.temporalObservation(t, "valid_historical", at.Add(-time.Hour), at.Add(-55*time.Minute), 2.70)
	valid = proof.withPriorFresh(t, valid, at.Add(-50*time.Minute))
	older := proof.temporalObservation(t, "older_historical", at.Add(-90*time.Minute), at.Add(-85*time.Minute), 2.65)
	older = proof.withPriorFresh(t, older, at.Add(-80*time.Minute))
	invalidCurrent := proof.temporalObservation(t, "newer_invalid", at.Add(-40*time.Minute), at.Add(-35*time.Minute), 2.80)
	invalidObservation := invalidCurrent.Normalized.Record.(canonical.Observation)
	invalidObservation.Metric = ""
	invalidCurrent.Normalized.Record = invalidObservation

	request := FreshnessResolutionRequest{
		Policy: proof.policy.Identity, ExpectedFallbackPolicy: proof.policy.LastKnownGood.Identity, UseClass: DataUseResearch,
		EvaluationTime: at, Context: FreshnessContextCurrentState, Key: valid.Key,
		Current: &invalidCurrent, Historical: []TemporalRecord{older, valid},
	}
	resolution, err := proof.evaluator.Resolve(request)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.FallbackStatus != FallbackUsed || resolution.Selected.State != TemporalStale || resolution.Selected.Record != valid.Normalized.Output {
		t.Fatalf("resolution = %+v", resolution)
	}
	if resolution.FallbackAge == nil || *resolution.FallbackAge != time.Hour || resolution.Qualification != LKGQualificationPriorFresh || resolution.PriorFreshEvaluation == nil {
		t.Fatalf("fallback qualification/age = %+v", resolution)
	}
	if resolution.ReasonCode != ResolutionCurrentInvalid || resolution.CurrentRecord == nil || *resolution.CurrentRecord != invalidCurrent.Normalized.Output {
		t.Fatalf("fallback current identity/reason = %+v", resolution)
	}
	if resolution.Selected.CanonicalQuality != NormalizationQualityValidated || resolution.Selected.State == TemporalFresh {
		t.Fatal("LKG selection collapsed validation/freshness/fallback dimensions or upgraded stale data")
	}

	request.Historical = []TemporalRecord{valid, older}
	repeated, err := proof.evaluator.Resolve(request)
	if err != nil || repeated.Selected.Record != resolution.Selected.Record || repeated.FallbackStatus != resolution.FallbackStatus {
		t.Fatalf("candidate retrieval order changed deterministic selection: %+v, %v", repeated, err)
	}
}

func TestFreshCurrentRecordResolvesWithoutFallback(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	current := proof.temporalObservation(t, "fresh_current", at.Add(-10*time.Minute), at.Add(-5*time.Minute), 2.80)
	resolution, err := proof.evaluator.Resolve(FreshnessResolutionRequest{
		Policy: proof.policy.Identity, ExpectedFallbackPolicy: proof.policy.LastKnownGood.Identity, UseClass: DataUseResearch,
		EvaluationTime: at, Context: FreshnessContextCurrentState, Key: current.Key, Current: &current,
	})
	if err != nil || resolution.FallbackStatus != FallbackNotUsed || resolution.Selected.State != TemporalFresh || resolution.Selected.Record != current.Normalized.Output {
		t.Fatalf("fresh current resolution = %+v, %v", resolution, err)
	}
	wrongRequestedKey := current.Key
	wrongRequestedKey.Qualifier = "consumer_price_index"
	_, err = proof.evaluator.Resolve(FreshnessResolutionRequest{
		Policy: proof.policy.Identity, ExpectedFallbackPolicy: proof.policy.LastKnownGood.Identity, UseClass: DataUseResearch,
		EvaluationTime: at, Context: FreshnessContextCurrentState, Key: wrongRequestedKey, Current: &current,
	})
	requireFreshnessCode(t, err, FreshnessErrorNoAcceptableLKG)
}

func TestLastKnownGoodFallbackPolicyAndMaximumAgeBoundaries(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	atMaximum := proof.temporalObservation(t, "at_maximum_age", at.Add(-2*time.Hour), at.Add(-115*time.Minute), 2.60)
	atMaximum = proof.withPriorFresh(t, atMaximum, at.Add(-110*time.Minute))
	request := FreshnessResolutionRequest{
		Policy: proof.policy.Identity, ExpectedFallbackPolicy: proof.policy.LastKnownGood.Identity, UseClass: DataUseResearch,
		EvaluationTime: at, Context: FreshnessContextCurrentState, Key: atMaximum.Key, Historical: []TemporalRecord{atMaximum},
	}
	allowed, err := proof.evaluator.Resolve(request)
	if err != nil || allowed.FallbackAge == nil || *allowed.FallbackAge != 2*time.Hour {
		t.Fatalf("age == maximum fallback age must be allowed: %+v, %v", allowed, err)
	}

	tooOld := proof.temporalObservation(t, "beyond_maximum_age", at.Add(-2*time.Hour-time.Second), at.Add(-115*time.Minute), 2.55)
	tooOld = proof.withPriorFresh(t, tooOld, at.Add(-110*time.Minute))
	request.Key = tooOld.Key
	request.Historical = []TemporalRecord{tooOld}
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorFallbackAgeExceeded)

	prohibited := syntheticFreshnessPolicy("cmp_fallback_prohibited", "4.0.0", "cmp_fallback_prohibited_lkg", "4.0.0", FallbackProhibited, 0)
	if err := proof.policies.Register(prohibited); err != nil {
		t.Fatal(err)
	}
	request.Policy = prohibited.Identity
	request.ExpectedFallbackPolicy = prohibited.LastKnownGood.Identity
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorFallbackProhibited)

	freshOnly := syntheticFreshnessPolicy("cmp_fresh_only", "4.0.1", "cmp_fresh_only_lkg", "4.0.1", FallbackFreshOnly, 2*time.Hour)
	if err := proof.policies.Register(freshOnly); err != nil {
		t.Fatal(err)
	}
	request.Policy = freshOnly.Identity
	request.ExpectedFallbackPolicy = freshOnly.LastKnownGood.Identity
	request.Key = atMaximum.Key
	request.Historical = []TemporalRecord{atMaximum}
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorNoAcceptableLKG)

	wrongFallbackVersion := cloneComponentIdentity(freshOnly.LastKnownGood.Identity)
	wrongFallbackVersion.Version.Value = "9.9.9"
	request.ExpectedFallbackPolicy = wrongFallbackVersion
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorPolicyVersionMismatch)
}

func TestLastKnownGoodRejectsAmbiguityMissingPriorAndInactiveCandidates(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	left := proof.temporalObservation(t, "equal_time_left", at.Add(-time.Hour), at.Add(-55*time.Minute), 2.70)
	right := proof.temporalObservation(t, "equal_time_right", at.Add(-time.Hour), at.Add(-54*time.Minute), 2.71)
	left = proof.withPriorFresh(t, left, at.Add(-50*time.Minute))
	right = proof.withPriorFresh(t, right, at.Add(-49*time.Minute))
	request := FreshnessResolutionRequest{
		Policy: proof.policy.Identity, ExpectedFallbackPolicy: proof.policy.LastKnownGood.Identity, UseClass: DataUseResearch,
		EvaluationTime: at, Context: FreshnessContextCurrentState, Key: left.Key, Historical: []TemporalRecord{right, left},
	}
	_, err := proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorAmbiguousLKG)

	noPrior := proof.temporalObservation(t, "no_prior", at.Add(-time.Hour), at.Add(-55*time.Minute), 2.70)
	request.Historical = []TemporalRecord{noPrior}
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorNoAcceptableLKG)

	retracted := proof.withPriorFresh(t, proof.temporalObservation(t, "retracted", at.Add(-time.Hour), at.Add(-55*time.Minute), 2.70), at.Add(-50*time.Minute))
	changedAt := at.Add(-10 * time.Minute)
	retracted.Lifecycle = TemporalRecordLifecycle{State: TemporalRecordRetracted, ChangedAt: &changedAt}
	request.Historical = []TemporalRecord{retracted}
	_, err = proof.evaluator.Resolve(request)
	requireFreshnessCode(t, err, FreshnessErrorNoAcceptableLKG)
}

func TestUntilSupersededAndHistoricalReplayLifecycleSemantics(t *testing.T) {
	proof := newFreshnessProof(t)
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	record := proof.temporalObservation(t, "until_superseded", at.Add(-24*time.Hour), at.Add(-23*time.Hour), 2.70)
	policy := syntheticFreshnessPolicy("cmp_until_superseded", "5.0.0", "cmp_until_superseded_lkg", "5.0.0", FallbackProhibited, 0)
	policy.ValidityMode = FreshnessValidityUntilSuperseded
	policy.TimestampRole = TimestampRoleNone
	policy.FreshFor = 0
	policy.ExpireAfter = 0
	policy.AllowedFutureSkew = 0
	if err := proof.policies.Register(policy); err != nil {
		t.Fatal(err)
	}
	evaluation, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	if err != nil || evaluation.State != TemporalFresh || evaluation.Age != nil || evaluation.ReasonCode != FreshnessReasonUntilSuperseded {
		t.Fatalf("until-superseded evaluation = %+v, %v", evaluation, err)
	}

	changedAt := at.Add(time.Hour)
	record.Lifecycle = TemporalRecordLifecycle{State: TemporalRecordSuperseded, ChangedAt: &changedAt}
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorRecordSuperseded)
	historical, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextHistoricalReplay, Record: record})
	if err != nil || historical.State != TemporalFresh {
		t.Fatalf("historical pre-supersession evaluation = %+v, %v", historical, err)
	}
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: DataUseResearch, EvaluationTime: changedAt, Context: FreshnessContextHistoricalReplay, Record: record})
	requireFreshnessCode(t, err, FreshnessErrorRecordSuperseded)
}

func TestFutureTimestampSkewAndKnowledgeCutoff(t *testing.T) {
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	proof, within := newEventFreshnessProof(t, "future_within_skew", at.Add(3*time.Second), at.Add(-time.Minute))
	evaluation, err := proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: within})
	if err != nil || evaluation.State != TemporalFresh || !evaluation.WithinFutureSkew || evaluation.Age == nil || *evaluation.Age != -3*time.Second {
		t.Fatalf("within-skew evaluation = %+v, %v", evaluation, err)
	}

	_, beyond := newEventFreshnessProof(t, "future_beyond_skew", at.Add(6*time.Second), at.Add(-time.Minute))
	_, err = proof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: proof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextCurrentState, Record: beyond})
	requireFreshnessCode(t, err, FreshnessErrorFutureTimestampBeyondSkew)

	observationProof := newFreshnessProof(t)
	notKnown := observationProof.temporalObservation(t, "not_yet_collected", at.Add(-time.Minute), at.Add(time.Minute), 2.75)
	_, err = observationProof.evaluator.Evaluate(FreshnessEvaluationRequest{Policy: observationProof.policy.Identity, UseClass: DataUseResearch, EvaluationTime: at, Context: FreshnessContextHistoricalReplay, Record: notKnown})
	requireFreshnessCode(t, err, FreshnessErrorRecordNotYetAvailable)
}

func newEventFreshnessProof(t *testing.T, id string, effectiveAt, receivedAt time.Time) (freshnessProof, TemporalRecord) {
	t.Helper()
	providers, err := NewRegistry(RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	definition := validDefinition(CapabilityEconomicCalendar)
	definition.Identity = canonical.ProviderIdentity{ID: "pvd_synthetic_calendar", Namespace: "synthetic.calendar"}
	definition.DisplayName = "Synthetic Calendar"
	definition.AdapterVersion = canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}
	definition.Capabilities[0].Raw.Schema = canonical.VersionIdentity{Namespace: "synthetic.calendar.schema", Value: "event/v1"}
	definition.Capabilities[0].Authentication.Class = AuthenticationNone
	if err := providers.Register(definition); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryRawPayloadStore()
	source := canonical.SourceIdentity{ID: "src_synthetic_calendar", Kind: canonical.SourceKindDataset}
	revision := canonical.RevisionIdentity{Namespace: "synthetic.calendar.release", Value: id}
	raw, err := PersistRawPayload(context.Background(), providers, store, RawPayloadPersistenceRequest{
		ID: RawPayloadID("rpa_" + id), Provider: definition.Identity, Capability: CapabilityEconomicCalendar, Raw: definition.Capabilities[0].Raw,
		Capture: RawPayloadCapture{ByteForm: RawPayloadByteFormEntityBody, ContentCodingState: ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source:  &source, Revision: &revision, ReceivedAt: receivedAt,
		Retention: RawPayloadRetentionPolicy{Class: RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: RawPayloadRedistributionNotAuthorized}, Complete: true,
	}, []byte(fmt.Sprintf(`{"effective_at":%q}`, effectiveAt.Format(time.RFC3339Nano))))
	if err != nil {
		t.Fatal(err)
	}
	providerIdentity := definition.Identity
	normalizer := canonical.ComponentIdentity{ID: "cmp_synthetic_calendar_normalizer", Kind: canonical.ComponentKindNormalizer, Name: "synthetic calendar normalizer", Version: canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}, Provider: &providerIdentity}
	subject := canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_us_policy_rate", ContractVersion: canonical.InstrumentContractV1}
	event := canonical.Event{ContractVersion: canonical.EventContractV1, ID: canonical.EventID("evt_" + id), Type: canonical.EventTypeMacroRelease, Assertion: canonical.EventAssertionAsserted, Title: "Scheduled release", Subjects: []canonical.ContractRef{subject}, EffectiveAt: &effectiveAt, CreatedAt: receivedAt}
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}
	content, err := canonical.CanonicalContractContentIdentity(event)
	if err != nil {
		t.Fatal(err)
	}
	target := canonical.ContractSchemaRef{Kind: canonical.ContractKindEvent, Version: canonical.EventContractV1}
	output := canonical.ImmutableContractRef{Contract: canonical.ContractRef{Kind: canonical.ContractKindEvent, ID: string(event.ID), ContractVersion: event.ContractVersion}, Revision: revision, Content: content}
	result := NormalizationResult{
		Status: NormalizationStatusAccepted, Quality: NormalizationQualityValidated, RawRef: raw.Ref, Normalizer: normalizer, Target: target, Output: output,
		Validation: NormalizationValidation{RawVerified: true, Parsed: true, Mapped: true, CanonicalValidated: true, ProvenanceValidated: true}, Record: event,
	}
	policy := FreshnessPolicy{
		ContractVersion: FreshnessPolicyContractV1,
		Identity:        canonical.ComponentIdentity{ID: "cmp_calendar_freshness", Kind: canonical.ComponentKindPolicy, Name: "synthetic calendar freshness", Version: canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}},
		CapabilityID:    CapabilityEconomicCalendar, Target: target, UseClass: DataUseResearch,
		ValidityMode: FreshnessValidityAgeBounded, TimestampRole: TimestampRoleEffectiveAt, FreshFor: 30 * time.Minute, ExpireAfter: 4 * time.Hour,
		AllowedFutureSkew: 5 * time.Second, MissingTimestamp: MissingTimestampFail,
		LastKnownGood: LastKnownGoodPolicy{ContractVersion: LastKnownGoodPolicyContractV1, Identity: canonical.ComponentIdentity{ID: "cmp_calendar_lkg", Kind: canonical.ComponentKindPolicy, Name: "synthetic calendar LKG", Version: canonical.VersionIdentity{Namespace: "semver", Value: "1.0.0"}}, Mode: FallbackProhibited},
	}
	policies := NewFreshnessPolicyRegistry()
	if err := policies.Register(policy); err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewFreshnessEvaluator(providers, policies)
	if err != nil {
		t.Fatal(err)
	}
	return freshnessProof{policies: policies, evaluator: evaluator, policy: policy}, TemporalRecord{Normalized: result, Key: FreshnessKey{CapabilityID: CapabilityEconomicCalendar, Target: target, Subject: subject, Qualifier: "scheduled_release"}, Lifecycle: TemporalRecordLifecycle{State: TemporalRecordActive}}
}

func requireFreshnessCode(t *testing.T, err error, want FreshnessErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected freshness error %q", want)
	}
	var typed *FreshnessError
	if !errors.As(err, &typed) || typed.Code != want {
		t.Fatalf("error = %T %v; want freshness code %q", err, err, want)
	}
}
