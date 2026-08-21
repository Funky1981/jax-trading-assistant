package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
)

const (
	C1F3RepeatabilityR3ExecutionAuthorizationVersion = "ai-shadow-c1f3-repeatability-r3-execution-authorization-v1"
	C1F3RepeatabilityR3AuthorizationFingerprint      = "ab3651e951a0c059390c3e5ad29cf425dfb195f7568f705d28072ec6cc7a5c87"
)

type C1F3RepeatabilityR3ExecutionAuthorization struct {
	Version       string
	OperatorOptIn bool
}

type C1F3RepeatabilityR3ExecutionAuthorizationPlan struct {
	Version                        string `json:"version"`
	AuthorizationFingerprint       string `json:"authorization_fingerprint"`
	OperatorOptIn                  bool   `json:"operator_opt_in"`
	HostedInferenceAuthorized      bool   `json:"hosted_inference_authorized"`
	CredentialPresent              bool   `json:"credential_present"`
	FrozenBindingsValid            bool   `json:"frozen_bindings_valid"`
	BaselineBindingValid           bool   `json:"baseline_binding_valid"`
	RepeatabilityScoringValid      bool   `json:"repeatability_scoring_valid"`
	BudgetValid                    bool   `json:"budget_valid"`
	EvidenceNamespaceCollisionFree bool   `json:"evidence_namespace_collision_free"`
	ProviderInputIsolated          bool   `json:"provider_input_isolated"`
	ProviderInputMatchesC1F3       bool   `json:"provider_input_matches_original_c1f3"`
	R2ResponseIsolated             bool   `json:"r2_response_isolated"`
	RuntimeSafetyValid             bool   `json:"runtime_safety_valid"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
}

type c1f3RepeatabilityR3AuthorizationDefinition struct {
	Identity                    string `json:"identity"`
	ApplicableProfile           string `json:"applicable_profile"`
	ProfileFingerprint          string `json:"profile_fingerprint"`
	BaselineRunID               string `json:"baseline_run_id"`
	BaselineArtifactIndexSHA256 string `json:"baseline_artifact_index_sha256"`
	SemanticProfileFingerprint  string `json:"semantic_profile_fingerprint"`
	ScorerIdentity              string `json:"scorer_identity"`
	ScorerFileSHA256            string `json:"scorer_file_sha256"`
	Provider                    string `json:"provider"`
	Model                       string `json:"model"`
	ReasoningEffort             string `json:"reasoning_effort"`
	OperatorOptInMechanism      string `json:"operator_opt_in_mechanism"`
	HostedAuthorizationRequired string `json:"hosted_authorization_required"`
	CaseCount                   int    `json:"case_count"`
	Repetitions                 int    `json:"repetitions"`
	MaximumBudgetMicros         int64  `json:"maximum_budget_micros"`
	EvidenceNamespace           string `json:"evidence_namespace"`
	DefaultDeny                 bool   `json:"default_deny"`
}

func NewC1F3RepeatabilityR3ExecutionAuthorization(operatorOptIn bool) C1F3RepeatabilityR3ExecutionAuthorization {
	return C1F3RepeatabilityR3ExecutionAuthorization{Version: C1F3RepeatabilityR3ExecutionAuthorizationVersion, OperatorOptIn: operatorOptIn}
}

func C1F3RepeatabilityR3ExecutionAuthorizationFingerprint() string {
	definition := c1f3RepeatabilityR3AuthorizationDefinition{
		Identity: C1F3RepeatabilityR3ExecutionAuthorizationVersion, ApplicableProfile: C1F3RepeatabilityR3ProfileIdentity,
		ProfileFingerprint: C1F3RepeatabilityR3ProfileFingerprint, BaselineRunID: C1F3GeneralizationSourceRunID,
		BaselineArtifactIndexSHA256: C1F3GeneralizationArtifactIndex, SemanticProfileFingerprint: "dc38761583c8856db7b79d515d0f799e416581e84f369649a32aab3053dadd9d",
		ScorerIdentity: C1F3RepeatabilityScoringVersion, ScorerFileSHA256: C1F3RepeatabilityScoringFileSHA256,
		Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel, ReasoningEffort: OpenAIDiagnosticReasoningEffort,
		OperatorOptInMechanism: "--authorize-c1f3-repeatability-r3", HostedAuthorizationRequired: OpenAIDiagnosticInferenceAuthEnv + "=true",
		CaseCount: 48, Repetitions: 1, MaximumBudgetMicros: 300_000, EvidenceNamespace: C1F3RepeatabilityR3EvidenceNamespace, DefaultDeny: true,
	}
	raw, _ := json.Marshal(definition)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func validateC1F3RepeatabilityR3AuthorizationScope(profile DiagnosticEvaluationProfile, config OpenAIDiagnosticConfig) error {
	if !isC1F3RepeatabilityR3Profile(profile) {
		if config.C1F3RepeatabilityR3ExecutionAuthorization.OperatorOptIn {
			return fmt.Errorf("C1F3 repeatability r3 execution authorization is scoped only to %s", C1F3RepeatabilityR3ProfileIdentity)
		}
		return nil
	}
	if config.C1F3RepeatabilityR3ExecutionAuthorization.Version != C1F3RepeatabilityR3ExecutionAuthorizationVersion {
		return fmt.Errorf("C1F3 repeatability r3 execution authorization identity is missing or incompatible")
	}
	expected, ok := loadC1F3RepeatabilityDiagnosticProfile(profile.Identity)
	if !ok || !reflect.DeepEqual(profile, expected) {
		return fmt.Errorf("C1F3 repeatability r3 authorization scope does not match the registered frozen profile")
	}
	if profile.RequiredProvider != config.Runtime.Provider || profile.RequiredModel != config.Runtime.Model ||
		profile.RequiredExperimentID != config.ExperimentID || profile.RequiredOutputContractMode != config.OutputContractMode ||
		profile.EvidenceNamespace != config.EvidenceNamespace() || profile.CaseCount != 48 || profile.DefaultRepetitions != 1 ||
		len(profile.AllowedRepetitions) != 1 || profile.AllowedRepetitions[0] != 1 || !profile.RequiresTypedAttributionLabels ||
		config.ReasoningEffort != OpenAIDiagnosticReasoningEffort || config.BudgetCeilingMicros > 300_000 {
		return fmt.Errorf("C1F3 repeatability r3 authorization scope does not match the exact frozen 48 x 1 cell")
	}
	return nil
}

func validateC1F3RepeatabilityR3ExecutionAuthorization(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1F3RepeatabilityR3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if !isC1F3RepeatabilityR3Profile(prepared.Profile) {
		return nil
	}
	plan := prepared.Plan.C1F3RepeatabilityR3ExecutionAuthorization
	bindings := prepared.Plan.C1F3RepeatabilityFrozenBindings
	frozenProfile, frozenProfileErr := FrozenC1F3RepeatabilityR3Profile()
	if plan == nil || bindings == nil || plan.Version != C1F3RepeatabilityR3ExecutionAuthorizationVersion ||
		plan.AuthorizationFingerprint != C1F3RepeatabilityR3ExecutionAuthorizationFingerprint() || !plan.FrozenBindingsValid ||
		!plan.BaselineBindingValid || !plan.RepeatabilityScoringValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree ||
		!plan.ProviderInputIsolated || !plan.ProviderInputMatchesC1F3 || !plan.R2ResponseIsolated || !plan.RuntimeSafetyValid ||
		frozenProfileErr != nil || prepared.Plan.C1F3FrozenBindings == nil || bindings.SemanticStack != *prepared.Plan.C1F3FrozenBindings ||
		bindings.ComparisonScoring != frozenProfile.ComparisonScoring || bindings.ProfileIdentity != C1F3RepeatabilityR3ProfileIdentity ||
		bindings.ProfileFingerprint != C1F3RepeatabilityR3ProfileFingerprint || bindings.Baseline != frozenC1F3RepeatabilityBaseline() ||
		bindings.CaseCount != 48 || bindings.Repetitions != 1 {
		return fmt.Errorf("C1F3 repeatability r3 execution authorization plan is incomplete or invalid")
	}
	if !config.C1F3RepeatabilityR3ExecutionAuthorization.OperatorOptIn {
		return fmt.Errorf("C1F3 repeatability r3 execution requires the explicit --authorize-c1f3-repeatability-r3 operator opt-in")
	}
	if !config.InferenceExplicitlyAuthorized {
		return fmt.Errorf("C1F3 repeatability r3 execution requires %s=true in addition to the r3 opt-in", OpenAIDiagnosticInferenceAuthEnv)
	}
	if !config.APIKey.present() {
		return fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !plan.OperatorOptIn || !plan.HostedInferenceAuthorized || !plan.CredentialPresent || !plan.ExecutionAuthorized {
		return fmt.Errorf("C1F3 repeatability r3 execution authorization conditions are not all satisfied")
	}
	return ValidateDiagnosticExecutionShape(prepared)
}

func RevalidateC1F3RepeatabilityR3ProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if !isC1F3RepeatabilityR3Profile(prepared.Profile) {
		return validateC1F3RepeatabilityR3AuthorizationScope(prepared.Profile, config)
	}
	revalidated, err := prepareHostedDiagnostic(prepared.Paths, config, prepared.Plan.Safety, true)
	if err != nil {
		return err
	}
	revalidated, err = ApplyDiagnosticExecutionShape(revalidated, prepared.ExecutionShape)
	if err != nil {
		return err
	}
	if err := validateC1F3RepeatabilityR3ExecutionAuthorization(revalidated, config); err != nil {
		return err
	}
	if !reflect.DeepEqual(revalidated.Plan, prepared.Plan) || !reflect.DeepEqual(revalidated.ExecutionShape, prepared.ExecutionShape) {
		return fmt.Errorf("C1F3 repeatability r3 provider-construction revalidation diverged from the prepared frozen plan")
	}
	return nil
}
