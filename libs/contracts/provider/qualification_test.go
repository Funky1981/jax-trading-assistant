package provider

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

var qualificationBaseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func syntheticQualificationPolicy() QualificationPolicy {
	return QualificationPolicy{
		ContractVersion: QualificationPolicyContractV1,
		Identity: canonical.ComponentIdentity{
			ID: "cmp_source_qualification_policy", Kind: canonical.ComponentKindPolicy, Name: "synthetic source qualification policy",
			Version: canonical.VersionIdentity{Namespace: "jax.policy.qualification", Value: "synthetic/v1"},
		},
		DecisionVersion: QualificationDecisionVersionV1(),
		RequiredEvidence: []QualificationEvidenceKind{
			EvidenceSourceIdentity, EvidenceAuthorityClassification, EvidenceLicensingReview, EvidenceApprovedInternalReview,
		},
		MinimumIdentity: IdentityCertaintyHigh, MinimumReliability: ReliabilityAcceptable,
		RequireKnownCoverage: true, RequireKnownCost: true, RequireKnownAvailability: true, RequireKnownUpdateLatency: true,
		RequireStableAccess: true, RequireManagedSchema: true,
		RequirePersistentStore: true, RequireDerivedData: true, RequireResearchUse: true,
		MaximumReviewInterval: 365 * 24 * time.Hour,
	}
}

func syntheticQualificationInput(id QualificationID, scope QualificationScope, authority SourceAuthorityClass) QualificationAssessmentInput {
	windowFrom := qualificationBaseTime.AddDate(-1, 0, 0)
	windowTo := qualificationBaseTime.Add(-24 * time.Hour)
	success, missing, correction := 9950, 25, 10
	review := qualificationBaseTime.Add(180 * 24 * time.Hour)
	permissions := QualificationPermissions{
		PrimaryEvidence: PermissionProhibited, Corroboration: PermissionProhibited,
		InvestigationTrigger: PermissionProhibited, ResearchDisplay: PermissionProhibited,
		FallbackInput: PermissionProhibited, CanonicalFactualAssertion: PermissionProhibited,
	}
	setQualificationPermission(&permissions, scope.IntendedUse, PermissionAllowed)
	return QualificationAssessmentInput{
		ID: id, Scope: scope, AuthorityClass: authority,
		Dimensions: QualificationDimensions{
			IdentityCertainty: IdentityCertaintyVerified,
			Reliability: HistoricalReliability{
				Class: ReliabilityStrong, WindowFrom: &windowFrom, WindowTo: &windowTo,
				DeliverySuccessBasisPoints: &success, MissingDataBasisPoints: &missing, CorrectionBasisPoints: &correction,
				Outages: OutageRareBounded, SchemaChanges: SchemaChangeRare,
			},
			Availability: AvailabilityScheduled, UpdateLatency: UpdateLatencyPeriodic,
			ProvenanceQuality: ProvenanceQualityImmutableRawAndLineage,
			Rights: UsageRights{
				InternalProcessing: UseRightAllowed, PersistentStorage: UseRightAllowed, DerivedData: UseRightAllowed,
				ModelResearchUse: UseRightAllowed, Redistribution: UseRightProhibited, CommercialUse: UseRightRestricted,
				Retention: RetentionRightIndefinite,
			},
			Cost:            CostCharacteristics{Model: CostModelFixedSubscription, Class: CostClassLow},
			AccessStability: AccessStabilityStable, SchemaStability: SchemaStabilityManaged, Coverage: cloneCoverage(scope.Coverage),
		},
		Permissions: permissions,
		Evidence:    qualificationEvidenceFixture(scope.Source),
		Assessor: canonical.ComponentIdentity{
			ID: "cmp_source_qualification_assessor", Kind: canonical.ComponentKindValidator, Name: "synthetic source qualification assessor",
			Version: canonical.VersionIdentity{Namespace: "jax.assessment.qualification", Value: "synthetic/v1"},
		},
		AssessedAt: qualificationBaseTime, EffectiveFrom: qualificationBaseTime, ReviewDueAt: &review,
	}
}

func qualificationEvidenceFixture(source canonical.SourceIdentity) []QualificationEvidence {
	kinds := []QualificationEvidenceKind{
		EvidenceSourceIdentity, EvidenceAuthorityClassification, EvidenceLicensingReview,
		EvidenceHistoricalReliability, EvidenceCoverageReview, EvidenceApprovedInternalReview,
	}
	values := make([]QualificationEvidence, 0, len(kinds))
	for i, kind := range kinds {
		evidenceSource := canonical.SourceIdentity{ID: "src_internal_qualification_review", Kind: canonical.SourceKindInternal}
		if kind == EvidenceSourceIdentity {
			evidenceSource = source
		}
		values = append(values, QualificationEvidence{Kind: kind, Ref: canonical.EvidenceRef{
			ContractVersion: canonical.EvidenceRefContractV1,
			Evidence:        canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: "evd_qualification_" + string(rune('a'+i)), ContractVersion: canonical.EvidenceContractV2},
			Content:         canonical.RawContentIdentity([]byte("synthetic immutable qualification evidence " + string(kind))),
			Source:          evidenceSource, Revision: canonical.RevisionIdentity{Namespace: "synthetic.qualification_evidence", Value: string(kind) + "/v1"},
			CollectedAt: qualificationBaseTime.Add(-time.Duration(len(kinds)-i) * time.Hour),
		}})
	}
	return values
}

func qualificationScope(sourceID string, sourceKind canonical.SourceKind, capability CapabilityID, role QualificationRole, use QualificationUse) QualificationScope {
	coverage := QualificationCoverage{Status: CoverageSupported, Scope: CoverageScopeCapabilityWide}
	return QualificationScope{Source: canonical.SourceIdentity{ID: sourceID, Kind: sourceKind}, CapabilityID: capability, Role: role, IntendedUse: use, Coverage: coverage}
}

func setQualificationPermission(permissions *QualificationPermissions, use QualificationUse, state PermissionState) {
	switch use {
	case QualificationUsePrimaryEvidence:
		permissions.PrimaryEvidence = state
	case QualificationUseCorroboration:
		permissions.Corroboration = state
	case QualificationUseInvestigationTrigger:
		permissions.InvestigationTrigger = state
	case QualificationUseResearchDisplay:
		permissions.ResearchDisplay = state
	case QualificationUseFallbackInput:
		permissions.FallbackInput = state
	case QualificationUseCanonicalFactualAssertion:
		permissions.CanonicalFactualAssertion = state
	}
}

func cloneCoverage(value QualificationCoverage) QualificationCoverage {
	value.Bounds = append([]CoverageBound(nil), value.Bounds...)
	return value
}

func qualificationProviderRegistry(t *testing.T) (*Registry, ProviderDefinition, ProviderDefinition) {
	t.Helper()
	registry, err := NewRegistry(RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	direct := validDefinition(CapabilityMacroObservation)
	direct.Identity = canonical.ProviderIdentity{ID: "pvd_synthetic_direct", Namespace: "synthetic.direct"}
	direct.DisplayName = "Synthetic Direct Transport"
	direct.Capabilities[0].Raw.Schema = canonical.VersionIdentity{Namespace: "synthetic.direct.macro", Value: "v1"}
	aggregator := validDefinition(CapabilityMacroObservation)
	aggregator.Identity = canonical.ProviderIdentity{ID: "pvd_synthetic_aggregator", Namespace: "synthetic.aggregator"}
	aggregator.DisplayName = "Synthetic Aggregator Transport"
	aggregator.Capabilities[0].Raw.Schema = canonical.VersionIdentity{Namespace: "synthetic.aggregator.macro", Value: "v1"}
	for _, definition := range []ProviderDefinition{direct, aggregator} {
		if err := registry.Register(definition); err != nil {
			t.Fatalf("Register(provider) error = %v", err)
		}
	}
	return registry, direct, aggregator
}

func qualificationRegistryFixture(t *testing.T) (*QualificationRegistry, QualificationPolicy, ProviderDefinition, ProviderDefinition) {
	t.Helper()
	providers, direct, aggregator := qualificationProviderRegistry(t)
	registry, err := NewQualificationRegistry(QualificationRegistryContractV1, providers)
	if err != nil {
		t.Fatal(err)
	}
	policy := syntheticQualificationPolicy()
	if err := registry.RegisterPolicy(policy); err != nil {
		t.Fatal(err)
	}
	return registry, policy, direct, aggregator
}

func requireQualificationError(t *testing.T, err error, want QualificationRegistryErrorCode) {
	t.Helper()
	var target *QualificationRegistryError
	if !errors.As(err, &target) || target.Code != want {
		t.Fatalf("error = %T %v, want qualification registry code %q", err, err, want)
	}
}

func TestQualificationStatesAuthorityAndCorroborationAreRoleSpecific(t *testing.T) {
	policy := syntheticQualificationPolicy()
	officialScope := qualificationScope("src_synthetic_official_macro", canonical.SourceKindRegulator, CapabilityMacroObservation, QualificationRoleMacroeconomicAuthority, QualificationUseCanonicalFactualAssertion)
	official, err := AssessQualification(policy, syntheticQualificationInput("qlf_official_macro_authority", officialScope, SourceAuthorityOfficialPrimary))
	if err != nil || official.State != QualificationQualified || official.AuthorityClass != SourceAuthorityOfficialPrimary {
		t.Fatalf("official qualification = %+v, %v", official, err)
	}

	secondaryScope := qualificationScope("src_synthetic_professional_news", canonical.SourceKindPublisher, CapabilityNewsArticle, QualificationRoleSecondaryCorroboration, QualificationUseCorroboration)
	secondaryInput := syntheticQualificationInput("qlf_secondary_corroboration", secondaryScope, SourceAuthorityProfessionalSecondary)
	secondaryInput.Permissions.InvestigationTrigger = PermissionAllowed
	secondaryInput.Permissions.ResearchDisplay = PermissionAllowed
	secondary, err := AssessQualification(policy, secondaryInput)
	if err != nil || secondary.State != QualificationQualified {
		t.Fatalf("secondary qualification = %+v, %v", secondary, err)
	}
	if secondary.Permissions.PrimaryEvidence != PermissionProhibited || secondary.Permissions.CanonicalFactualAssertion != PermissionProhibited || secondary.Permissions.Corroboration != PermissionAllowed {
		t.Fatalf("secondary permissions do not preserve corroborating-only boundary: %+v", secondary.Permissions)
	}
}

func TestConditionalQualificationUsesTypedConditions(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_conditional_news", canonical.SourceKindPublisher, CapabilityNewsArticle, QualificationRoleSecondaryCorroboration, QualificationUseCorroboration)
	input := syntheticQualificationInput("qlf_conditional_corroboration", scope, SourceAuthorityProfessionalSecondary)
	input.Permissions.Corroboration = PermissionConditional
	input.Conditions = []QualificationCondition{{Type: ConditionIndependentPrimaryRequired, AppliesTo: QualificationUseCorroboration}}
	decision, err := AssessQualification(policy, input)
	if err != nil || decision.State != QualificationConditionallyQualified || len(decision.Conditions) != 1 {
		t.Fatalf("conditional qualification = %+v, %v", decision, err)
	}
}

func TestRestrictedRequiredRightsDemandTypedConditionalQualification(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_restricted_right", canonical.SourceKindPublisher, CapabilityNewsArticle, QualificationRoleSecondaryCorroboration, QualificationUseCorroboration)
	input := syntheticQualificationInput("qlf_restricted_right", scope, SourceAuthorityProfessionalSecondary)
	input.Dimensions.Rights.PersistentStorage = UseRightRestricted
	input.Permissions.Corroboration = PermissionConditional
	input.Conditions = []QualificationCondition{{Type: ConditionUsageRestrictionsApply, AppliesTo: QualificationUseCorroboration}}
	decision, err := AssessQualification(policy, input)
	if err != nil || decision.State != QualificationConditionallyQualified {
		t.Fatalf("restricted-right qualification = %+v, %v", decision, err)
	}

	missingCondition := input
	missingCondition.ID = "qlf_restricted_right_missing_condition"
	missingCondition.Conditions = nil
	decision, err = AssessQualification(policy, missingCondition)
	if err != nil || decision.State != QualificationUnqualified || !containsDisqualifier(decision.Disqualifiers, DisqualifierConflictingConditions) {
		t.Fatalf("restricted right without condition = %+v, %v", decision, err)
	}
}

func TestHardLicensingDisqualifierCannotBeOverriddenByReliabilityOrCost(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_restricted_macro", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	input := syntheticQualificationInput("qlf_licensing_prohibited", scope, SourceAuthorityOther)
	input.Dimensions.Rights.PersistentStorage = UseRightProhibited
	input.Dimensions.Reliability.Class = ReliabilityStrong
	input.Dimensions.Cost = CostCharacteristics{Model: CostModelFree, Class: CostClassNegligible}
	decision, err := AssessQualification(policy, input)
	if err != nil {
		t.Fatal(err)
	}
	if decision.State != QualificationUnqualified || !containsDisqualifier(decision.Disqualifiers, DisqualifierLicensingProhibitsUse) {
		t.Fatalf("licensing disqualification was overridden: %+v", decision)
	}
}

func TestMissingEvidenceFailsClosedAsNotAssessed(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_missing_evidence", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	input := syntheticQualificationInput("qlf_missing_evidence", scope, SourceAuthorityOther)
	filtered := []QualificationEvidence{}
	for _, evidence := range input.Evidence {
		if evidence.Kind != EvidenceLicensingReview {
			filtered = append(filtered, evidence)
		}
	}
	input.Evidence = filtered
	decision, err := AssessQualification(policy, input)
	if err != nil || decision.State != QualificationNotAssessed || !containsDisqualifier(decision.Disqualifiers, DisqualifierRequiredEvidenceMissing) {
		t.Fatalf("missing evidence decision = %+v, %v", decision, err)
	}
}

func TestModelDerivedSourceCannotSelfQualifyAsAuthority(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_model_archive", canonical.SourceKindInternal, CapabilityNewsArticle, QualificationRoleAuthoritativeSource, QualificationUseCanonicalFactualAssertion)
	input := syntheticQualificationInput("qlf_model_claimed_authority", scope, SourceAuthorityModelDerived)
	decision, err := AssessQualification(policy, input)
	if err != nil || decision.State != QualificationUnqualified || !containsDisqualifier(decision.Disqualifiers, DisqualifierModelDerivedAuthority) {
		t.Fatalf("model-derived authority decision = %+v, %v", decision, err)
	}
}

func TestQualificationRegistryExactLookupAndSameSourceDifferentProviderPaths(t *testing.T) {
	registry, policy, direct, aggregator := qualificationRegistryFixture(t)
	logicalSource := canonical.SourceIdentity{ID: "src_synthetic_macro_authority", Kind: canonical.SourceKindRegulator}
	directScope := qualificationScope(logicalSource.ID, logicalSource.Kind, CapabilityMacroObservation, QualificationRoleMacroeconomicAuthority, QualificationUseCanonicalFactualAssertion)
	directScope.ProviderPath = &direct.Identity
	directInput := syntheticQualificationInput("qlf_macro_direct_path", directScope, SourceAuthorityOfficialPrimary)
	directDecision, err := AssessQualification(policy, directInput)
	if err != nil || registry.Register(directDecision) != nil {
		t.Fatalf("register direct path = %v", err)
	}
	aggregatorScope := cloneQualificationScope(directScope)
	aggregatorScope.ProviderPath = &aggregator.Identity
	aggregatorInput := syntheticQualificationInput("qlf_macro_aggregator_path", aggregatorScope, SourceAuthorityOfficialPrimary)
	aggregatorInput.Dimensions.Rights.PersistentStorage = UseRightProhibited
	aggregatorDecision, err := AssessQualification(policy, aggregatorInput)
	if err != nil || registry.Register(aggregatorDecision) != nil {
		t.Fatalf("register aggregator path = %v", err)
	}
	at := qualificationBaseTime.Add(time.Hour)
	directEvaluation, err := registry.LookupAt(directScope, at)
	if err != nil || directEvaluation.EffectiveState != QualificationQualified {
		t.Fatalf("direct path lookup = %+v, %v", directEvaluation, err)
	}
	aggregatorEvaluation, err := registry.LookupAt(aggregatorScope, at)
	if err != nil || aggregatorEvaluation.EffectiveState != QualificationUnqualified {
		t.Fatalf("aggregator path lookup = %+v, %v", aggregatorEvaluation, err)
	}
	if directEvaluation.Decision.Scope.Source != aggregatorEvaluation.Decision.Scope.Source || sameProviderIdentity(*directEvaluation.Decision.Scope.ProviderPath, *aggregatorEvaluation.Decision.Scope.ProviderPath) {
		t.Fatal("source identity did not remain stable across distinct provider paths")
	}
	sourceOnly := cloneQualificationScope(directScope)
	sourceOnly.ProviderPath = nil
	_, err = registry.LookupAt(sourceOnly, at)
	requireQualificationError(t, err, QualificationErrorUnknownQualification)
}

func TestQualificationRegistryRejectsDuplicateAndAmbiguousDeclarations(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_duplicate", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	decision, err := AssessQualification(policy, syntheticQualificationInput("qlf_duplicate_scope_a", scope, SourceAuthorityOther))
	if err != nil || registry.Register(decision) != nil {
		t.Fatal(err)
	}
	requireQualificationError(t, registry.Register(decision), QualificationErrorDuplicateDecision)
	ambiguousInput := syntheticQualificationInput("qlf_duplicate_scope_b", scope, SourceAuthorityOther)
	ambiguousInput.EffectiveFrom = qualificationBaseTime.Add(time.Hour)
	ambiguous, err := AssessQualification(policy, ambiguousInput)
	if err != nil {
		t.Fatal(err)
	}
	requireQualificationError(t, registry.Register(ambiguous), QualificationErrorAmbiguousDecision)
}

func TestQualificationLookupFailsIfStoredHistoryBecomesAmbiguous(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_ambiguous_lookup", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	first, _ := AssessQualification(policy, syntheticQualificationInput("qlf_ambiguous_lookup_a", scope, SourceAuthorityOther))
	if err := registry.Register(first); err != nil {
		t.Fatal(err)
	}
	secondInput := syntheticQualificationInput("qlf_ambiguous_lookup_b", scope, SourceAuthorityOther)
	secondInput.EffectiveFrom = qualificationBaseTime.Add(time.Hour)
	second, _ := AssessQualification(policy, secondInput)
	key, _ := qualificationScopeKey(scope)
	registry.decisions[second.ID] = second
	registry.byScope[key] = append(registry.byScope[key], second.ID)
	_, err := registry.LookupAt(scope, qualificationBaseTime.Add(2*time.Hour))
	requireQualificationError(t, err, QualificationErrorAmbiguousDecision)
}

func TestQualificationHistorySupersessionAndReviewAreReconstructable(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_history", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	firstInput := syntheticQualificationInput("qlf_history_v1", scope, SourceAuthorityOther)
	firstReview := qualificationBaseTime.Add(300 * 24 * time.Hour)
	firstInput.ReviewDueAt = &firstReview
	first, err := AssessQualification(policy, firstInput)
	if err != nil || registry.Register(first) != nil {
		t.Fatal(err)
	}
	policyV2 := policy
	policyV2.Identity.Version.Value = "synthetic/v2"
	if err := registry.RegisterPolicy(policyV2); err != nil {
		t.Fatal(err)
	}
	secondInput := syntheticQualificationInput("qlf_history_v2", scope, SourceAuthorityOther)
	secondInput.AssessedAt = qualificationBaseTime.Add(90 * 24 * time.Hour)
	secondInput.EffectiveFrom = secondInput.AssessedAt
	secondReview := secondInput.AssessedAt.Add(180 * 24 * time.Hour)
	secondInput.ReviewDueAt = &secondReview
	secondInput.Supersedes = &first.ID
	secondInput.Dimensions.Rights.DerivedData = UseRightProhibited
	second, err := AssessQualification(policyV2, secondInput)
	if err != nil || registry.Register(second) != nil {
		t.Fatalf("register successor = %v", err)
	}
	before, err := registry.LookupAt(scope, qualificationBaseTime.Add(30*24*time.Hour))
	if err != nil || before.Decision.ID != first.ID || before.EffectiveState != QualificationQualified {
		t.Fatalf("historical v1 lookup = %+v, %v", before, err)
	}
	after, err := registry.LookupAt(scope, secondInput.EffectiveFrom.Add(time.Hour))
	if err != nil || after.Decision.ID != second.ID || after.EffectiveState != QualificationUnqualified {
		t.Fatalf("historical v2 lookup = %+v, %v", after, err)
	}
	history, err := registry.History(scope)
	if err != nil || len(history) != 2 || history[0].ID != first.ID || history[1].ID != second.ID {
		t.Fatalf("history = %+v, %v", history, err)
	}
	if history[0].Policy.Version.Value == history[1].Policy.Version.Value {
		t.Fatal("policy evolution was not preserved in immutable qualification history")
	}

	reviewScope := qualificationScope("src_synthetic_review_due", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	reviewInput := syntheticQualificationInput("qlf_review_due", reviewScope, SourceAuthorityOther)
	reviewDue := qualificationBaseTime.Add(10 * 24 * time.Hour)
	reviewInput.ReviewDueAt = &reviewDue
	reviewDecision, _ := AssessQualification(policy, reviewInput)
	if err := registry.Register(reviewDecision); err != nil {
		t.Fatal(err)
	}
	overdue, err := registry.LookupAt(reviewScope, reviewDue)
	if err != nil || overdue.EffectiveState != QualificationUnqualified || !containsDisqualifier(overdue.ActiveDisqualifiers, DisqualifierReviewOverdue) {
		t.Fatalf("review-overdue evaluation = %+v, %v", overdue, err)
	}
}

func TestQualificationExpiryAndExplicitUnqualifiedDifferFromUnknown(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_expiry", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	input := syntheticQualificationInput("qlf_expiring", scope, SourceAuthorityOther)
	expires := qualificationBaseTime.Add(24 * time.Hour)
	input.EffectiveTo = &expires
	decision, _ := AssessQualification(policy, input)
	if err := registry.Register(decision); err != nil {
		t.Fatal(err)
	}
	expired, err := registry.LookupAt(scope, expires)
	if err != nil || expired.EffectiveState != QualificationUnqualified || !containsDisqualifier(expired.ActiveDisqualifiers, DisqualifierQualificationExpired) {
		t.Fatalf("expired qualification = %+v, %v", expired, err)
	}
	unknownScope := cloneQualificationScope(scope)
	unknownScope.Source.ID = "src_synthetic_never_assessed"
	_, err = registry.LookupAt(unknownScope, expires)
	requireQualificationError(t, err, QualificationErrorUnknownQualification)
}

func TestHealthAndFreshnessCannotChangeQualification(t *testing.T) {
	registry, policy, direct, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_independent_dimensions", canonical.SourceKindRegulator, CapabilityMacroObservation, QualificationRoleMacroeconomicAuthority, QualificationUseCanonicalFactualAssertion)
	scope.ProviderPath = &direct.Identity
	decision, _ := AssessQualification(policy, syntheticQualificationInput("qlf_independent_dimensions", scope, SourceAuthorityOfficialPrimary))
	if err := registry.Register(decision); err != nil {
		t.Fatal(err)
	}
	at := qualificationBaseTime.Add(time.Hour)
	want, err := registry.LookupAt(scope, at)
	if err != nil {
		t.Fatal(err)
	}
	combinations := []struct {
		health    RuntimeStatus
		freshness TemporalQualityState
	}{
		{RuntimeHealthy, TemporalFresh},
		{RuntimeHealthy, TemporalStale},
		{RuntimeUnavailable, TemporalFresh},
		{RuntimeDegraded, TemporalFresh},
	}
	for _, combination := range combinations {
		_ = HealthAssessment{Provider: direct.Identity, CapabilityID: scope.CapabilityID, Status: combination.health}
		_ = FreshnessEvaluation{CapabilityID: scope.CapabilityID, State: combination.freshness}
		got, lookupErr := registry.LookupAt(scope, at)
		if lookupErr != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("health=%q freshness=%q changed qualification: %+v, %v", combination.health, combination.freshness, got, lookupErr)
		}
	}
}

func TestQualificationValidationNegativeCases(t *testing.T) {
	policy := syntheticQualificationPolicy()
	baseScope := qualificationScope("src_synthetic_negative", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	cases := map[string]func(*QualificationAssessmentInput){
		"invalid_source_identity": func(input *QualificationAssessmentInput) { input.Scope.Source.ID = "bad" },
		"invalid_provider_identity": func(input *QualificationAssessmentInput) {
			input.Scope.ProviderPath = &canonical.ProviderIdentity{ID: "bad", Namespace: "synthetic.bad"}
		},
		"invalid_historical_interval": func(input *QualificationAssessmentInput) {
			before := input.EffectiveFrom.Add(-time.Hour)
			input.EffectiveTo = &before
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := syntheticQualificationInput("qlf_negative_"+QualificationID(name), baseScope, SourceAuthorityOther)
			mutate(&input)
			if _, err := AssessQualification(policy, input); err == nil {
				t.Fatal("invalid qualification was accepted")
			}
		})
	}

	mismatchScope := qualificationScope("src_synthetic_capability_mismatch", canonical.SourceKindDataset, CapabilityCorporateFiling, QualificationRoleMacroeconomicAuthority, QualificationUseCanonicalFactualAssertion)
	mismatch, err := AssessQualification(policy, syntheticQualificationInput("qlf_capability_mismatch", mismatchScope, SourceAuthorityOfficialPrimary))
	if err != nil || mismatch.State != QualificationUnqualified || !containsDisqualifier(mismatch.Disqualifiers, DisqualifierCapabilityMismatch) {
		t.Fatalf("capability mismatch = %+v, %v", mismatch, err)
	}

	prohibitedScope := qualificationScope("src_synthetic_use_prohibited", canonical.SourceKindPublisher, CapabilityNewsArticle, QualificationRoleNewsEvidence, QualificationUseResearchDisplay)
	prohibitedInput := syntheticQualificationInput("qlf_use_prohibited", prohibitedScope, SourceAuthorityProfessionalSecondary)
	prohibitedInput.Permissions.ResearchDisplay = PermissionProhibited
	prohibited, err := AssessQualification(policy, prohibitedInput)
	if err != nil || prohibited.State != QualificationUnqualified || !containsDisqualifier(prohibited.Disqualifiers, DisqualifierIntendedUseProhibited) {
		t.Fatalf("prohibited intended use = %+v, %v", prohibited, err)
	}

	coverageScope := qualificationScope("src_synthetic_coverage_mismatch", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	coverageInput := syntheticQualificationInput("qlf_coverage_mismatch", coverageScope, SourceAuthorityOther)
	coverageInput.Dimensions.Coverage = QualificationCoverage{Status: CoverageSupported, Scope: CoverageScopeBounded, Bounds: []CoverageBound{{Dimension: CoverageDimensionGeography, Value: "us"}}}
	coverage, err := AssessQualification(policy, coverageInput)
	if err != nil || coverage.State != QualificationUnqualified || !containsDisqualifier(coverage.Disqualifiers, DisqualifierUnsupportedCoverage) {
		t.Fatalf("coverage mismatch = %+v, %v", coverage, err)
	}

	conflictScope := qualificationScope("src_synthetic_condition_conflict", canonical.SourceKindPublisher, CapabilityNewsArticle, QualificationRoleSecondaryCorroboration, QualificationUseCorroboration)
	conflictInput := syntheticQualificationInput("qlf_condition_conflict", conflictScope, SourceAuthorityProfessionalSecondary)
	conflictInput.Permissions.Corroboration = PermissionConditional
	conflictInput.Conditions = []QualificationCondition{
		{Type: ConditionResearchDisplayOnly, AppliesTo: QualificationUseCorroboration},
		{Type: ConditionIndependentPrimaryRequired, AppliesTo: QualificationUseCorroboration},
	}
	conflict, err := AssessQualification(policy, conflictInput)
	if err != nil || conflict.State != QualificationUnqualified || !containsDisqualifier(conflict.Disqualifiers, DisqualifierConflictingConditions) {
		t.Fatalf("conflicting conditions = %+v, %v", conflict, err)
	}

	valid, _ := AssessQualification(policy, syntheticQualificationInput("qlf_unknown_state", baseScope, SourceAuthorityOther))
	valid.State = "MAGICAL"
	if err := valid.Validate(); err == nil {
		t.Fatal("unknown qualification state was accepted")
	}
	valid, _ = AssessQualification(policy, syntheticQualificationInput("qlf_unknown_version", baseScope, SourceAuthorityOther))
	valid.DecisionVersion = canonical.VersionIdentity{Namespace: "jax.qualification_decision", Value: "v99"}
	providers, _, _ := qualificationProviderRegistry(t)
	registry, _ := NewQualificationRegistry(QualificationRegistryContractV1, providers)
	_ = registry.RegisterPolicy(policy)
	requireQualificationError(t, registry.Register(valid), QualificationErrorUnsupportedDecisionVersion)

	unknownProviderScope := cloneQualificationScope(baseScope)
	unknownProviderScope.ProviderPath = &canonical.ProviderIdentity{ID: "pvd_synthetic_unknown", Namespace: "synthetic.unknown"}
	unknownProvider, _ := AssessQualification(policy, syntheticQualificationInput("qlf_unknown_provider", unknownProviderScope, SourceAuthorityOther))
	requireQualificationError(t, registry.Register(unknownProvider), QualificationErrorInvalidProviderIdentity)

	badPolicy := policy
	badPolicy.ContractVersion = "jax.source_qualification_policy/v99"
	if err := badPolicy.Validate(); err == nil {
		t.Fatal("unsupported qualification policy version was accepted")
	}
	requireQualificationError(t, registry.RegisterPolicy(policy), QualificationErrorDuplicatePolicy)
}

func TestQualificationDecisionStrictStableJSON(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_json", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	decision, err := AssessQualification(policy, syntheticQualificationInput("qlf_json_round_trip", scope, SourceAuthorityOther))
	if err != nil {
		t.Fatal(err)
	}
	first, err := EncodeQualificationDecisionJSON(decision)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeQualificationDecisionJSON(decision)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("qualification JSON is not stable: %v", err)
	}
	var decoded QualificationDecision
	if err := DecodeQualificationDecisionJSON(first, &decoded); err != nil || !reflect.DeepEqual(decoded, decision) {
		t.Fatalf("qualification JSON round trip changed meaning: %v", err)
	}
	unknown := bytes.Replace(first, []byte(`"assessed_at":`), []byte(`"credential_value":"secret","assessed_at":`), 1)
	duplicate := bytes.Replace(first, []byte(`{"contract_version":`), []byte(`{"contract_version":"jax.source_qualification_decision/v1","contract_version":`), 1)
	unsupported := bytes.Replace(first, []byte(string(QualificationDecisionContractV1)), []byte("jax.source_qualification_decision/v99"), 1)
	for name, raw := range map[string][]byte{
		"unknown": unknown, "duplicate": duplicate, "unsupported": unsupported,
		"trailing": append(append([]byte(nil), first...), []byte(` {}`)...), "null": []byte("null"), "invalid_utf8": []byte{0xff},
	} {
		t.Run(name, func(t *testing.T) {
			if err := DecodeQualificationDecisionJSON(raw, &decoded); err == nil {
				t.Fatal("invalid qualification JSON was accepted")
			}
		})
	}
}

func TestDisabledQualificationRemainsDistinctWhenReviewIsOverdue(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_disabled", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	input := syntheticQualificationInput("qlf_disabled", scope, SourceAuthorityOther)
	input.Disabled = true
	decision, err := AssessQualification(policy, input)
	if err != nil || decision.State != QualificationDisabled || !containsDisqualifier(decision.Disqualifiers, DisqualifierSourceDisabled) {
		t.Fatalf("disabled qualification = %+v, %v", decision, err)
	}
	if err := registry.Register(decision); err != nil {
		t.Fatal(err)
	}
	evaluation, err := registry.LookupAt(scope, decision.ReviewDueAt.Add(time.Hour))
	if err != nil || evaluation.EffectiveState != QualificationDisabled || !containsDisqualifier(evaluation.ActiveDisqualifiers, DisqualifierReviewOverdue) {
		t.Fatalf("overdue disabled qualification = %+v, %v", evaluation, err)
	}
}

func TestQualificationPolicyReliabilityAndStabilityRequirementsFailClosed(t *testing.T) {
	policy := syntheticQualificationPolicy()
	scope := qualificationScope("src_synthetic_policy_dimensions", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	tests := []struct {
		name string
		want HardDisqualifier
		edit func(*QualificationAssessmentInput)
	}{
		{name: "insufficient_reliability", want: DisqualifierInsufficientReliability, edit: func(input *QualificationAssessmentInput) {
			input.Dimensions.Reliability.Class = ReliabilityLimited
		}},
		{name: "unstable_access", want: DisqualifierUnstableAccess, edit: func(input *QualificationAssessmentInput) {
			input.Dimensions.AccessStability = AccessStabilityUnstable
		}},
		{name: "unstable_schema", want: DisqualifierUnstableSchema, edit: func(input *QualificationAssessmentInput) {
			input.Dimensions.SchemaStability = SchemaStabilityUnstable
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := syntheticQualificationInput(QualificationID("qlf_"+test.name), scope, SourceAuthorityOther)
			test.edit(&input)
			decision, err := AssessQualification(policy, input)
			if err != nil || decision.State != QualificationUnqualified || !containsDisqualifier(decision.Disqualifiers, test.want) {
				t.Fatalf("policy dimension decision = %+v, %v", decision, err)
			}
		})
	}
}

func TestQualificationRegistryRejectsInvalidVersionAndSupersession(t *testing.T) {
	providers, _, _ := qualificationProviderRegistry(t)
	_, err := NewQualificationRegistry("jax.source_qualification_registry/v99", providers)
	requireQualificationError(t, err, QualificationErrorUnsupportedRegistryVersion)

	registry, policy, _, _ := qualificationRegistryFixture(t)
	firstScope := qualificationScope("src_synthetic_supersession", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	first, err := AssessQualification(policy, syntheticQualificationInput("qlf_supersession_first", firstScope, SourceAuthorityOther))
	if err != nil || registry.Register(first) != nil {
		t.Fatal(err)
	}
	changedScope := cloneQualificationScope(firstScope)
	changedScope.Source.ID = "src_synthetic_supersession_other"
	secondInput := syntheticQualificationInput("qlf_supersession_invalid", changedScope, SourceAuthorityOther)
	secondInput.EffectiveFrom = first.EffectiveFrom.Add(time.Hour)
	secondInput.AssessedAt = secondInput.EffectiveFrom
	secondReview := secondInput.AssessedAt.Add(180 * 24 * time.Hour)
	secondInput.ReviewDueAt = &secondReview
	secondInput.Supersedes = &first.ID
	second, err := AssessQualification(policy, secondInput)
	if err != nil {
		t.Fatal(err)
	}
	requireQualificationError(t, registry.Register(second), QualificationErrorInvalidHistoricalInterval)
}

func TestQualificationRegistryReturnsDefensiveCopies(t *testing.T) {
	registry, policy, _, _ := qualificationRegistryFixture(t)
	scope := qualificationScope("src_synthetic_defensive_copy", canonical.SourceKindDataset, CapabilityMacroObservation, QualificationRoleResearchSupplementary, QualificationUseResearchDisplay)
	scope.Coverage = QualificationCoverage{Status: CoverageSupported, Scope: CoverageScopeBounded, Bounds: []CoverageBound{{Dimension: CoverageDimensionGeography, Value: "gb"}}}
	decision, err := AssessQualification(policy, syntheticQualificationInput("qlf_defensive_copy", scope, SourceAuthorityOther))
	if err != nil || registry.Register(decision) != nil {
		t.Fatal(err)
	}
	at := qualificationBaseTime.Add(time.Hour)
	first, err := registry.LookupAt(scope, at)
	if err != nil {
		t.Fatal(err)
	}
	first.Decision.State = QualificationDisabled
	first.Decision.Evidence[0].Kind = EvidenceCoverageReview
	first.Decision.Scope.Coverage.Bounds[0].Value = "mutated"
	first.Decision.Policy.Version.Value = "mutated"
	second, err := registry.LookupAt(scope, at)
	if err != nil {
		t.Fatal(err)
	}
	if second.Decision.State != decision.State || !reflect.DeepEqual(second.Decision.Evidence, decision.Evidence) || !reflect.DeepEqual(second.Decision.Scope.Coverage, decision.Scope.Coverage) || !sameComponentIdentity(second.Decision.Policy, decision.Policy) {
		t.Fatalf("lookup mutation escaped defensive copy: %+v", second.Decision)
	}

	history, err := registry.History(scope)
	if err != nil {
		t.Fatal(err)
	}
	history[0].Evidence[0].Kind = EvidenceCoverageReview
	again, err := registry.History(scope)
	if err != nil || !reflect.DeepEqual(again[0].Evidence, decision.Evidence) {
		t.Fatalf("history mutation escaped defensive copy: %+v, %v", again, err)
	}
}
