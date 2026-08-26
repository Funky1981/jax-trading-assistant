package provider

import (
	"context"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type syntheticResearchObservation struct {
	Subject canonical.ContractRef
	Metric  string
	Value   float64
	Unit    string
	At      time.Time
}

// projectSyntheticResearchObservation deliberately accepts only the canonical
// contract. No provider DTO, endpoint, method, or provider identity is part of
// the downstream research-facing representation.
func projectSyntheticResearchObservation(record canonical.Observation) syntheticResearchObservation {
	return syntheticResearchObservation{
		Subject: record.Subject, Metric: record.Metric, Value: *record.Value.Number,
		Unit: record.Value.Unit, At: record.ObservedAt,
	}
}

func TestPhase02ExitChainUsesCanonicalProviderNeutralResearchRepresentation(t *testing.T) {
	fixture := newNormalizationFixture(t, syntheticPayload)
	clock := &fakeTimeSource{now: syntheticReceivedAt}
	executor, err := NewOperationalExecutor(fixture.providers, syntheticOperationalPolicy(), clock, nil, NewMemoryInstrumentation())
	if err != nil {
		t.Fatal(err)
	}
	operation := syntheticOperation(fixture.definition)
	acquired, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
		return ProviderAttemptResult{RawBytes: append([]byte(nil), syntheticPayload...)}
	})
	if err != nil || acquired.Status != ExecutionSucceeded || acquired.Health.Status != RuntimeHealthy {
		t.Fatalf("WP-02.05 acquisition/health = %+v, %v", acquired, err)
	}

	source := canonical.SourceIdentity{ID: "src_synthetic_macro", Kind: canonical.SourceKindDataset}
	revision := canonical.RevisionIdentity{Namespace: "synthetic.release", Value: "phase-02-exit/v1"}
	raw, err := PersistRawPayload(context.Background(), fixture.providers, fixture.store, RawPayloadPersistenceRequest{
		ID: "rpa_phase_02_exit_chain", Provider: fixture.definition.Identity, Capability: CapabilityMacroObservation,
		Raw:     fixture.definition.Capabilities[0].Raw,
		Capture: RawPayloadCapture{ByteForm: RawPayloadByteFormEntityBody, ContentCodingState: ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source:  &source, Revision: &revision, ReceivedAt: acquired.CompletedAt,
		Retention: RawPayloadRetentionPolicy{
			Class:          RawPayloadRetentionReplayAudit,
			Policy:         canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"},
			Redistribution: RawPayloadRedistributionNotAuthorized,
		},
		Complete: true,
	}, acquired.RawBytes)
	if err != nil {
		t.Fatalf("WP-02.02 exact-byte persistence = %v", err)
	}
	verified, err := RetrieveRawPayload(context.Background(), fixture.store, raw.Ref)
	if err != nil || !reflect.DeepEqual(verified, syntheticPayload) || raw.Ref.Content.Digest.VerifyBytes(syntheticPayload) != nil {
		t.Fatalf("WP-02.02 exact-byte verification = %q, %v", verified, err)
	}

	normalized, err := fixture.pipeline.NormalizeStored(context.Background(), fixture.store, StoredNormalizationRequest{
		RawRef: raw.Ref, Target: fixture.descriptor.Target, Normalizer: fixture.descriptor.Component,
	})
	if err != nil || normalized.Status != NormalizationStatusAccepted || normalized.Quality != NormalizationQualityValidated {
		t.Fatalf("WP-02.03 normalization = %+v, %v", normalized, err)
	}
	observation, ok := normalized.Record.(canonical.Observation)
	if !ok || observation.ContractVersion != canonical.ObservationContractV2 || observation.Provenance == nil {
		t.Fatalf("canonical observation/provenance = %T %+v", normalized.Record, observation.Provenance)
	}

	freshnessPolicy := syntheticFreshnessPolicy("cmp_phase02_exit_freshness", "1.0.0", "cmp_phase02_exit_lkg", "1.0.0", FallbackFreshOrStale, 2*time.Hour)
	freshnessPolicy.FreshFor = time.Hour
	freshnessPolicies := NewFreshnessPolicyRegistry()
	if err := freshnessPolicies.Register(freshnessPolicy); err != nil {
		t.Fatal(err)
	}
	freshnessEvaluator, err := NewFreshnessEvaluator(fixture.providers, freshnessPolicies)
	if err != nil {
		t.Fatal(err)
	}
	temporal := TemporalRecord{
		Normalized: normalized,
		Key:        FreshnessKey{CapabilityID: CapabilityMacroObservation, Target: normalized.Target, Subject: observation.Subject, Qualifier: observation.Metric},
		Lifecycle:  TemporalRecordLifecycle{State: TemporalRecordActive},
	}
	evaluatedAt := acquired.CompletedAt.Add(10 * time.Minute)
	freshness, err := freshnessEvaluator.Evaluate(FreshnessEvaluationRequest{
		Policy: freshnessPolicy.Identity, UseClass: DataUseResearch, EvaluationTime: evaluatedAt,
		Context: FreshnessContextCurrentState, Record: temporal,
	})
	if err != nil || freshness.State != TemporalFresh {
		t.Fatalf("WP-02.04 freshness = %+v, %v", freshness, err)
	}

	qualificationRegistry, err := NewQualificationRegistry(QualificationRegistryContractV1, fixture.providers)
	if err != nil {
		t.Fatal(err)
	}
	qualificationPolicy := syntheticQualificationPolicy()
	if err := qualificationRegistry.RegisterPolicy(qualificationPolicy); err != nil {
		t.Fatal(err)
	}
	qualificationScope := qualificationScope(source.ID, source.Kind, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	qualificationScope.ProviderPath = &fixture.definition.Identity
	qualificationInput := syntheticQualificationInput("qlf_phase_02_exit_source", qualificationScope, SourceAuthorityOther)
	qualificationInput.AssessedAt = acquired.CompletedAt
	qualificationInput.EffectiveFrom = acquired.CompletedAt
	reviewDue := acquired.CompletedAt.Add(180 * 24 * time.Hour)
	qualificationInput.ReviewDueAt = &reviewDue
	qualification, err := AssessQualification(qualificationPolicy, qualificationInput)
	if err != nil || qualificationRegistry.Register(qualification) != nil {
		t.Fatalf("WP-02.06 qualification = %+v, %v", qualification, err)
	}
	eligibility, err := qualificationRegistry.LookupAt(qualificationScope, evaluatedAt)
	if err != nil || eligibility.EffectiveState != QualificationQualified {
		t.Fatalf("WP-02.06 lookup = %+v, %v", eligibility, err)
	}

	researchView := projectSyntheticResearchObservation(observation)
	if researchView.Metric != "gross_domestic_product_growth" || researchView.Value != 2.75 || researchView.Unit != "percent" {
		t.Fatalf("research-facing canonical projection = %+v", researchView)
	}
	if normalized.RawRef.Provider.ID == "" || reflect.TypeOf(researchView).NumField() != 5 {
		t.Fatal("proof did not retain provider provenance while keeping provider identity out of the research projection")
	}

	alternate := validDefinition(CapabilityMacroObservation)
	alternate.Identity = canonical.ProviderIdentity{ID: "pvd_synthetic_swap", Namespace: "synthetic.swap"}
	alternate.DisplayName = "Synthetic Swappable Adapter"
	alternate.Capabilities[0].Raw.Schema = canonical.VersionIdentity{Namespace: "synthetic.swap.macro", Value: "v1"}
	if err := fixture.providers.Register(alternate); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(alternate.Capabilities[0].CanonicalOutputs, fixture.definition.Capabilities[0].CanonicalOutputs) || alternate.Capabilities[0].CanonicalOutputs[0] != normalized.Target {
		t.Fatal("alternate adapter does not terminate in the same canonical contract")
	}

	// These independent dimensions are all visible at the gate and are not
	// collapsed into one GOOD/BAD value.
	if acquired.Health.Status != RuntimeHealthy || freshness.State != TemporalFresh || normalized.Quality != NormalizationQualityValidated || eligibility.EffectiveState != QualificationQualified {
		t.Fatal("phase 02 dimensions were conflated or lost")
	}
}
