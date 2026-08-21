package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F3ExecutionAuthorizationVersion = "ai-shadow-c1f3-execution-authorization-v1"

	OpenAIC1F3GeneralizationExperimentID      = "WP-00.03C1F3-GENERALIZATION"
	OpenAIC1F3BoundaryExperimentID            = "WP-00.03C1F3-BOUNDARY"
	OpenAIC1F3GeneralizationEvidenceNamespace = C1F3ProfileGeneralization
	OpenAIC1F3BoundaryEvidenceNamespace       = C1F3ProfileBoundary
)

// C1F3ExecutionAuthorization is process-local control-plane state supplied
// only by the diagnostic CLI. It is deliberately non-secret and non-persistent.
type C1F3ExecutionAuthorization struct {
	Version       string
	OperatorOptIn bool
}

type C1F3ExecutionAuthorizationPlan struct {
	Version                        string `json:"version"`
	AuthorizationFingerprint       string `json:"authorization_fingerprint"`
	OperatorOptIn                  bool   `json:"operator_opt_in"`
	HostedInferenceAuthorized      bool   `json:"hosted_inference_authorized"`
	CredentialPresent              bool   `json:"credential_present"`
	FrozenBindingsValid            bool   `json:"frozen_bindings_valid"`
	BudgetValid                    bool   `json:"budget_valid"`
	EvidenceNamespaceCollisionFree bool   `json:"evidence_namespace_collision_free"`
	ProviderInputIsolated          bool   `json:"provider_input_isolated"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
}

type C1F3FrozenBindingPlan struct {
	ProfileIdentity            string `json:"profile_identity"`
	ProfileFingerprint         string `json:"profile_fingerprint"`
	DatasetIdentity            string `json:"dataset_identity"`
	ManifestSHA256             string `json:"manifest_sha256"`
	ManifestFingerprint        string `json:"manifest_fingerprint"`
	InputLockIdentity          string `json:"input_lock_identity"`
	InputLockSHA256            string `json:"input_lock_sha256"`
	InputLockFingerprint       string `json:"input_lock_fingerprint"`
	FreezeIdentity             string `json:"freeze_identity"`
	FreezeSHA256               string `json:"freeze_sha256"`
	TypedSidecarIdentity       string `json:"typed_sidecar_identity"`
	TypedSidecarSHA256         string `json:"typed_sidecar_sha256"`
	TypedSidecarFingerprint    string `json:"typed_sidecar_fingerprint"`
	AdjudicationRubricIdentity string `json:"adjudication_rubric_identity"`
	AdjudicationRubricSHA256   string `json:"adjudication_rubric_sha256"`
	ScoringIdentity            string `json:"scoring_identity"`
	ScoringSHA256              string `json:"scoring_sha256"`
	ScoringRubricIdentity      string `json:"scoring_rubric_identity"`
	ScoringRubricSHA256        string `json:"scoring_rubric_sha256"`
	ScoringRubricFingerprint   string `json:"scoring_rubric_fingerprint"`
	PromptIdentity             string `json:"prompt_identity"`
	PromptSHA256               string `json:"prompt_sha256"`
	OutputContractIdentity     string `json:"output_contract_identity"`
	OutputContractSHA256       string `json:"output_contract_sha256"`
	ValidatorIdentity          string `json:"validator_identity"`
	ValidatorSHA256            string `json:"validator_sha256"`
	AttributionPolicyIdentity  string `json:"attribution_policy_identity"`
	AttributionPolicySHA256    string `json:"attribution_policy_sha256"`
	SemanticComparatorIdentity string `json:"semantic_comparator_identity"`
	SemanticComparatorSHA256   string `json:"semantic_comparator_sha256"`
	ResolverIdentity           string `json:"resolver_identity"`
	ResolverSHA256             string `json:"resolver_sha256"`
	Provider                   string `json:"provider"`
	Model                      string `json:"model"`
	ExperimentIdentity         string `json:"experiment_identity"`
	CaseCount                  int    `json:"case_count"`
	Repetitions                int    `json:"repetitions"`
}

type c1f3AuthorizationDefinition struct {
	Identity                    string   `json:"identity"`
	ApplicableProfiles          []string `json:"applicable_profiles"`
	OperatorOptInMechanism      string   `json:"operator_opt_in_mechanism"`
	HostedAuthorizationRequired string   `json:"hosted_authorization_required"`
	DefaultDeny                 bool     `json:"default_deny"`
}

func NewC1F3ExecutionAuthorization(operatorOptIn bool) C1F3ExecutionAuthorization {
	return C1F3ExecutionAuthorization{Version: C1F3ExecutionAuthorizationVersion, OperatorOptIn: operatorOptIn}
}

func C1F3ExecutionAuthorizationFingerprint() string {
	definition := c1f3AuthorizationDefinition{
		Identity:                    C1F3ExecutionAuthorizationVersion,
		ApplicableProfiles:          []string{C1F3ProfileGeneralization, C1F3ProfileBoundary},
		OperatorOptInMechanism:      "--authorize-c1f3-execution",
		HostedAuthorizationRequired: OpenAIDiagnosticInferenceAuthEnv + "=true",
		DefaultDeny:                 true,
	}
	raw, _ := json.Marshal(definition)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func isC1F3Profile(profile DiagnosticEvaluationProfile) bool {
	return profile.Identity == C1F3ProfileGeneralization || profile.Identity == C1F3ProfileBoundary
}

func loadC1F3AuthorizedDiagnosticProfile(identity string) (DiagnosticEvaluationProfile, bool) {
	frozen, err := LoadC1F3EvaluationProfile(identity)
	if err != nil {
		return DiagnosticEvaluationProfile{}, false
	}
	experimentID := OpenAIC1F3GeneralizationExperimentID
	namespace := OpenAIC1F3GeneralizationEvidenceNamespace
	budget := int64(300_000)
	freezeVersion := "ai-shadow-issuer-generalization-holdout-freeze-v3"
	if identity == C1F3ProfileBoundary {
		experimentID = OpenAIC1F3BoundaryExperimentID
		namespace = OpenAIC1F3BoundaryEvidenceNamespace
		budget = 200_000
		freezeVersion = "ai-shadow-issuer-boundary-challenge-freeze-v3"
	}
	return DiagnosticEvaluationProfile{
		Identity:     identity,
		ManifestPath: frozen.Dataset.ManifestPath, ManifestVersion: frozen.Dataset.Identity,
		ManifestFileSHA256: frozen.Dataset.ManifestSHA256, ManifestFingerprint: frozen.Dataset.SemanticFingerprint,
		FingerprintLockPath: frozen.Dataset.InputLockPath, FingerprintLockVersion: frozen.SourceInputLockIdentity(),
		FingerprintLockFileSHA256: frozen.Dataset.InputLockSHA256, FingerprintLockFingerprint: frozen.Dataset.InputFingerprint,
		FreezePath: frozen.Dataset.FreezePath, FreezeVersion: freezeVersion, FreezeFileSHA256: frozen.Dataset.FreezeSHA256,
		CaseCount: frozen.Dataset.CaseCount, DefaultRepetitions: 1, AllowedRepetitions: []int{1},
		RequiredProvider: OpenAIDiagnosticProvider, RequiredModel: OpenAIDiagnosticLunaModel,
		RequiredExperimentID: experimentID, RequiredOutputContractMode: OpenAIOutputContractStrictJSONSchema,
		EvidenceNamespace: namespace, CredentiallessPreflightAllowed: true, MaximumBudgetMicros: budget,
		ExecutionPromptVersion: V6PromptVersion, ExecutionOutputContract: V5SchemaVersion,
		ExecutionValidatorVersion: C1FValidatorVersion,
		ExecutionCausalPolicy:     CausalAttributionPolicyVersion, ScoringVersion: C1FScoringVersion,
		ScoringRubricVersion: C1F3ScoringRubricVersion, RequiresTypedAttributionLabels: true,
		TypedLabelPath: frozen.TypedSidecarPath, TypedLabelVersion: frozen.TypedSidecarIdentity,
		TypedLabelFileSHA256: frozen.TypedSidecarSHA256, TypedLabelFingerprint: frozen.TypedSidecarFingerprint,
		ScoringRubricPath: frozen.ScoringRubricPath, ScoringRubricFileSHA256: frozen.ScoringRubricSHA256,
		ScoringRubricFingerprint: frozen.ScoringRubricFingerprint,
	}, true
}

// LoadDiagnosticExecutionProfile retains the historical hosted registry and
// adds only the frozen C1F3 and versioned repeatability profiles through this
// explicit control-plane route.
func LoadDiagnosticExecutionProfile(identity string) (DiagnosticEvaluationProfile, error) {
	if profile, err := LoadDiagnosticEvaluationProfile(identity); err == nil {
		return profile, nil
	}
	if profile, ok := loadC1F3AuthorizedDiagnosticProfile(identity); ok {
		return profile, nil
	}
	if profile, ok := loadC1F3RepeatabilityDiagnosticProfile(identity); ok {
		return profile, nil
	}
	return DiagnosticEvaluationProfile{}, fmt.Errorf("unknown frozen issuer diagnostic execution profile %q", identity)
}

func c1f3FrozenBindingPlan(profile DiagnosticEvaluationProfile) (C1F3FrozenBindingPlan, error) {
	frozen, err := LoadC1F3EvaluationProfile(profile.Identity)
	if err != nil {
		return C1F3FrozenBindingPlan{}, err
	}
	fingerprint, err := frozen.Fingerprint()
	if err != nil {
		return C1F3FrozenBindingPlan{}, err
	}
	return C1F3FrozenBindingPlan{
		ProfileIdentity: frozen.Identity, ProfileFingerprint: fingerprint,
		DatasetIdentity: frozen.Dataset.Identity, ManifestSHA256: frozen.Dataset.ManifestSHA256, ManifestFingerprint: frozen.Dataset.SemanticFingerprint,
		InputLockIdentity: frozen.SourceInputLockIdentity(), InputLockSHA256: frozen.Dataset.InputLockSHA256, InputLockFingerprint: frozen.Dataset.InputFingerprint,
		FreezeIdentity: profile.FreezeVersion, FreezeSHA256: frozen.Dataset.FreezeSHA256,
		TypedSidecarIdentity: frozen.TypedSidecarIdentity, TypedSidecarSHA256: frozen.TypedSidecarSHA256, TypedSidecarFingerprint: frozen.TypedSidecarFingerprint,
		AdjudicationRubricIdentity: frozen.AdjudicationRubricIdentity, AdjudicationRubricSHA256: frozen.AdjudicationRubricSHA256,
		ScoringIdentity: frozen.Scoring, ScoringSHA256: frozen.ScoringSHA256,
		ScoringRubricIdentity: C1F3ScoringRubricVersion, ScoringRubricSHA256: frozen.ScoringRubricSHA256, ScoringRubricFingerprint: frozen.ScoringRubricFingerprint,
		PromptIdentity: frozen.Prompt, PromptSHA256: frozen.PromptSHA256,
		OutputContractIdentity: frozen.OutputContract, OutputContractSHA256: frozen.OutputContractSHA256,
		ValidatorIdentity: frozen.Validator, ValidatorSHA256: frozen.ValidatorSHA256,
		AttributionPolicyIdentity: frozen.AttributionPolicy, AttributionPolicySHA256: frozen.AttributionPolicySHA256,
		SemanticComparatorIdentity: frozen.SemanticIdentity, SemanticComparatorSHA256: frozen.SemanticIdentitySHA256,
		ResolverIdentity: frozen.Resolver, ResolverSHA256: frozen.ResolverSHA256,
		Provider: profile.RequiredProvider, Model: profile.RequiredModel, ExperimentIdentity: profile.RequiredExperimentID,
		CaseCount: profile.CaseCount, Repetitions: profile.DefaultRepetitions,
	}, nil
}

func validateC1F3AuthorizationScope(profile DiagnosticEvaluationProfile, config OpenAIDiagnosticConfig) error {
	if !isC1F3Profile(profile) {
		if config.C1F3ExecutionAuthorization.OperatorOptIn {
			return fmt.Errorf("C1F3 execution authorization is scoped only to the two registered frozen C1F3 profiles")
		}
		return nil
	}
	if config.C1F3ExecutionAuthorization.Version != C1F3ExecutionAuthorizationVersion {
		return fmt.Errorf("C1F3 execution authorization identity is missing or incompatible")
	}
	expected, ok := loadC1F3AuthorizedDiagnosticProfile(profile.Identity)
	if !ok || !reflect.DeepEqual(profile, expected) {
		return fmt.Errorf("C1F3 execution authorization scope does not match the registered frozen profile")
	}
	if profile.RequiredProvider != config.Runtime.Provider || profile.RequiredModel != config.Runtime.Model ||
		profile.RequiredExperimentID != config.ExperimentID || profile.RequiredOutputContractMode != config.OutputContractMode ||
		profile.EvidenceNamespace != config.EvidenceNamespace() || profile.DefaultRepetitions != 1 ||
		len(profile.AllowedRepetitions) != 1 || profile.AllowedRepetitions[0] != 1 || !profile.RequiresTypedAttributionLabels {
		return fmt.Errorf("C1F3 execution authorization scope does not match the exact execution cell")
	}
	return nil
}

func c1f3EvidenceNamespaceCollisionFree(root string) (bool, error) {
	return c1e3EvidenceNamespaceCollisionFree(root)
}

func validateC1F3FrozenSemanticSources(paths DiagnosticPaths, profile DiagnosticEvaluationProfile, resolver assetresolution.Resolver, exposures []string) (C1F3FrozenBindingPlan, error) {
	frozen, err := LoadC1F3EvaluationProfile(profile.Identity)
	if err != nil {
		return C1F3FrozenBindingPlan{}, err
	}
	wantProfileFingerprint := map[string]string{
		C1F3ProfileGeneralization: "dc38761583c8856db7b79d515d0f799e416581e84f369649a32aab3053dadd9d",
		C1F3ProfileBoundary:       "3fadbd207c5b340430e81b5a9663f2d7e791e0291fb492d74dd982b358387b94",
	}[profile.Identity]
	profileFingerprint, err := frozen.Fingerprint()
	if err != nil || frozen.Executable || profileFingerprint != wantProfileFingerprint {
		return C1F3FrozenBindingPlan{}, fmt.Errorf("frozen C1F3 metadata identity changed")
	}
	root := c1e3RepositoryRoot(paths.AssetRulesetPath)
	checks := []struct{ path, want string }{
		{filepath.Join(root, "internal", "modules", "aishadow", "validation_c1f.go"), frozen.ValidatorSHA256},
		{filepath.Join(root, "internal", "modules", "aishadow", "causal_attribution.go"), frozen.AttributionPolicySHA256},
		{filepath.Join(root, "internal", "modules", "aishadow", "semantic_identity.go"), frozen.SemanticIdentitySHA256},
		{filepath.Join(root, "internal", "modules", "aishadow", "scoring_c1f.go"), frozen.ScoringSHA256},
		{paths.AssetRulesetPath, frozen.ResolverSHA256},
		{filepath.Join(root, filepath.FromSlash(frozen.AdjudicationRubricPath)), frozen.AdjudicationRubricSHA256},
	}
	for _, check := range checks {
		got, hashErr := diagnosticFileSHA256(check.path)
		if hashErr != nil || got != check.want {
			return C1F3FrozenBindingPlan{}, fmt.Errorf("frozen C1F3 source hash changed for %s", filepath.ToSlash(check.path))
		}
	}
	if V6PromptSHA256() != frozen.PromptSHA256 || resolver.Rules.Version != frozen.Resolver {
		return C1F3FrozenBindingPlan{}, fmt.Errorf("frozen C1F3 prompt or resolver identity changed")
	}
	schemaSHA, err := fingerprint(V5OutputSchema(exposures))
	if err != nil || schemaSHA != frozen.OutputContractSHA256 {
		return C1F3FrozenBindingPlan{}, fmt.Errorf("frozen C1F3 output schema hash changed")
	}
	return c1f3FrozenBindingPlan(profile)
}

func loadC1F3ExecutionInputs(paths DiagnosticPaths, profile DiagnosticEvaluationProfile, resolver assetresolution.Resolver, exposures []string) (DiagnosticManifest, DiagnosticFingerprintLock, DiagnosticFreezeRecord, C1F3FrozenBindingPlan, error) {
	frozen, err := LoadC1F3EvaluationProfile(profile.Identity)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	bindings, err := validateC1F3FrozenSemanticSources(paths, profile, resolver, exposures)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	manifest, err := LoadFrozenC1F3Manifest(frozen, paths.ManifestPath, exposures)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	if err := ValidateFrozenC1F3InputLock(frozen, paths.FingerprintLockPath, manifest); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	lockRaw, err := os.ReadFile(paths.FingerprintLockPath)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	var lock DiagnosticFingerprintLock
	if err := decodeStrictC1F3ControlPlane(lockRaw, &lock); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	freezeRaw, err := os.ReadFile(paths.FreezePath)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	digest := sha256.Sum256(freezeRaw)
	if hex.EncodeToString(digest[:]) != frozen.Dataset.FreezeSHA256 {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, fmt.Errorf("C1F3 freeze file hash changed")
	}
	var freeze DiagnosticFreezeRecord
	if err := decodeStrictC1F3ControlPlane(freezeRaw, &freeze); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	if freeze.Version != profile.FreezeVersion || freeze.DatasetVersion != frozen.Dataset.Identity || freeze.CaseCount != frozen.Dataset.CaseCount ||
		freeze.Manifest.FileSHA256 != frozen.Dataset.ManifestSHA256 || freeze.Manifest.SemanticFingerprint != frozen.Dataset.SemanticFingerprint ||
		freeze.InputFingerprintLock.FileSHA256 != frozen.Dataset.InputLockSHA256 || freeze.InputFingerprintLock.SemanticFingerprint != frozen.Dataset.InputFingerprint ||
		freeze.Policy.Identity != frozen.Resolver || freeze.Policy.FileSHA256 != frozen.ResolverSHA256 ||
		freeze.PromptVersion != V5PromptVersion || freeze.OutputContract != V5SchemaVersion || freeze.LabelVersion != DiagnosticLabelVersion || freeze.CreatedAt.IsZero() {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, fmt.Errorf("C1F3 freeze metadata changed")
	}
	sidecar, err := LoadFrozenC1F3TypedLabelSidecar(frozen, paths.TypedLabelPath)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	if err := ValidateC1F3TypedLabelSidecar(frozen, sidecar, manifest, resolver); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	if _, err := LoadFrozenC1F3ScoringFreeze(frozen, paths.ScoringRubricPath); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, err
	}
	return manifest, lock, freeze, bindings, nil
}

func decodeStrictC1F3ControlPlane(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func validateC1F3ProviderInputIsolation(manifest DiagnosticManifest, config OpenAIDiagnosticConfig, exposures []string) error {
	forbidden := []string{
		"expected_mapping_status", "expected_direct_issuer", "expected_proxy_exposure", "expected_issuer_attributions",
		"expected_principal_proxy_candidates", "typed_attribution_rationale", "adjudication_status", "scoring_rubric",
	}
	for _, event := range manifest.Events {
		request, err := diagnosticInitialRequest(config, event.Input, exposures)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(request)
		if err != nil {
			return err
		}
		visible := strings.ToLower(string(raw))
		for _, blocked := range forbidden {
			if strings.Contains(visible, blocked) {
				return fmt.Errorf("C1F3 provider-visible request contains expected-answer field %q", blocked)
			}
		}
	}
	return nil
}

func validateC1F3ExecutionAuthorization(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1F3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if !isC1F3Profile(prepared.Profile) {
		return nil
	}
	plan := prepared.Plan.C1F3ExecutionAuthorization
	wantBindings, err := c1f3FrozenBindingPlan(prepared.Profile)
	if err != nil {
		return err
	}
	if plan == nil || plan.Version != C1F3ExecutionAuthorizationVersion || plan.AuthorizationFingerprint != C1F3ExecutionAuthorizationFingerprint() ||
		!plan.FrozenBindingsValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree || !plan.ProviderInputIsolated ||
		prepared.Plan.C1F3FrozenBindings == nil || *prepared.Plan.C1F3FrozenBindings != wantBindings {
		return fmt.Errorf("C1F3 execution authorization plan is incomplete or invalid")
	}
	if !config.C1F3ExecutionAuthorization.OperatorOptIn {
		return fmt.Errorf("C1F3 execution requires the explicit --authorize-c1f3-execution operator opt-in")
	}
	if !config.InferenceExplicitlyAuthorized {
		return fmt.Errorf("C1F3 execution requires %s=true in addition to the experiment-specific opt-in", OpenAIDiagnosticInferenceAuthEnv)
	}
	if !config.APIKey.present() {
		return fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !plan.OperatorOptIn || !plan.HostedInferenceAuthorized || !plan.CredentialPresent || !plan.ExecutionAuthorized {
		return fmt.Errorf("C1F3 execution authorization conditions are not all satisfied")
	}
	return ValidateDiagnosticExecutionShape(prepared)
}

// RevalidateC1F3ProviderConstruction repeats every frozen input, isolation,
// budget, namespace, credential, and authorization check immediately before
// provider construction becomes reachable.
func RevalidateC1F3ProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if !isC1F3Profile(prepared.Profile) {
		return validateC1F3AuthorizationScope(prepared.Profile, config)
	}
	revalidated, err := prepareHostedDiagnostic(prepared.Paths, config, prepared.Plan.Safety, true)
	if err != nil {
		return err
	}
	revalidated, err = ApplyDiagnosticExecutionShape(revalidated, prepared.ExecutionShape)
	if err != nil {
		return err
	}
	if err := validateC1F3ExecutionAuthorization(revalidated, config); err != nil {
		return err
	}
	if !reflect.DeepEqual(revalidated.Plan, prepared.Plan) || !reflect.DeepEqual(revalidated.ExecutionShape, prepared.ExecutionShape) {
		return fmt.Errorf("C1F3 provider-construction revalidation diverged from the prepared frozen plan")
	}
	return nil
}

func RevalidateOpenAIProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1E3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if err := validateC1F3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if err := validateC1F3RepeatabilityAuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if err := validateC1F3RepeatabilityR3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if isC1F3RepeatabilityR3Profile(prepared.Profile) {
		return RevalidateC1F3RepeatabilityR3ProviderConstruction(prepared, config)
	}
	if isC1F3RepeatabilityR2Profile(prepared.Profile) {
		return RevalidateC1F3RepeatabilityProviderConstruction(prepared, config)
	}
	if isC1F3Profile(prepared.Profile) {
		return RevalidateC1F3ProviderConstruction(prepared, config)
	}
	return RevalidateC1E3ProviderConstruction(prepared, config)
}
