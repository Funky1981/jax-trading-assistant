package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const (
	C1F3TerraChallengerExecutionAuthorizationVersion = "ai-shadow-c1f3-terra-challenger-t1-execution-authorization-v1"
	C1F3TerraChallengerAuthorizationFingerprint      = "d663eedb1416234b36924a7526366f9b90c8742dc7aa0dad0b6607dd9d678770"
)

type C1F3TerraChallengerExecutionAuthorization struct {
	Version       string
	OperatorOptIn bool
}

type C1F3TerraChallengerExecutionAuthorizationPlan struct {
	Version                        string `json:"version"`
	AuthorizationFingerprint       string `json:"authorization_fingerprint"`
	OperatorOptIn                  bool   `json:"operator_opt_in"`
	HostedInferenceAuthorized      bool   `json:"hosted_inference_authorized"`
	CredentialPresent              bool   `json:"credential_present"`
	FrozenBindingsValid            bool   `json:"frozen_bindings_valid"`
	LunaPreservationValid          bool   `json:"luna_preservation_valid"`
	DecisionRubricValid            bool   `json:"decision_rubric_valid"`
	BudgetValid                    bool   `json:"budget_valid"`
	EvidenceNamespaceCollisionFree bool   `json:"evidence_namespace_collision_free"`
	ProviderInputIsolated          bool   `json:"provider_input_isolated"`
	OnlyModelVariableChanged       bool   `json:"only_model_variable_changed"`
	BoundaryExcluded               bool   `json:"boundary_excluded"`
	RuntimeSafetyValid             bool   `json:"runtime_safety_valid"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
}

type c1f3TerraChallengerAuthorizationDefinition struct {
	Identity                    string `json:"identity"`
	ApplicableProfile           string `json:"applicable_profile"`
	ProfileFingerprint          string `json:"profile_fingerprint"`
	AcceptedLunaRunID           string `json:"accepted_luna_run_id"`
	AcceptedLunaArtifactIndex   string `json:"accepted_luna_artifact_index_sha256"`
	RubricFingerprint           string `json:"rubric_fingerprint"`
	Provider                    string `json:"provider"`
	Model                       string `json:"model"`
	ReasoningEffort             string `json:"reasoning_effort"`
	OperatorOptInMechanism      string `json:"operator_opt_in_mechanism"`
	HostedAuthorizationRequired string `json:"hosted_authorization_required"`
	CaseCount                   int    `json:"case_count"`
	Repetitions                 int    `json:"repetitions"`
	MaximumBudgetMicros         int64  `json:"maximum_budget_micros"`
	EvidenceNamespace           string `json:"evidence_namespace"`
	BoundaryExcluded            bool   `json:"boundary_excluded"`
	DefaultDeny                 bool   `json:"default_deny"`
}

func NewC1F3TerraChallengerExecutionAuthorization(operatorOptIn bool) C1F3TerraChallengerExecutionAuthorization {
	return C1F3TerraChallengerExecutionAuthorization{Version: C1F3TerraChallengerExecutionAuthorizationVersion, OperatorOptIn: operatorOptIn}
}

func C1F3TerraChallengerExecutionAuthorizationFingerprint() string {
	definition := c1f3TerraChallengerAuthorizationDefinition{
		Identity: C1F3TerraChallengerExecutionAuthorizationVersion, ApplicableProfile: C1F3TerraChallengerProfileIdentity,
		ProfileFingerprint: C1F3TerraChallengerProfileFingerprint, AcceptedLunaRunID: C1F3TerraAcceptedLunaRunID,
		AcceptedLunaArtifactIndex: C1F3TerraAcceptedLunaArtifactIndexSHA256, RubricFingerprint: C1F3TerraChallengerRubricFingerprint,
		Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticTerraModel, ReasoningEffort: OpenAIDiagnosticReasoningEffort,
		OperatorOptInMechanism: "--authorize-c1f3-terra-challenger-t1", HostedAuthorizationRequired: OpenAIDiagnosticInferenceAuthEnv + "=true",
		CaseCount: 48, Repetitions: 1, MaximumBudgetMicros: 300_000, EvidenceNamespace: C1F3TerraChallengerEvidenceNamespace,
		BoundaryExcluded: true, DefaultDeny: true,
	}
	raw, _ := json.Marshal(definition)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func validateC1F3TerraChallengerAuthorizationScope(profile DiagnosticEvaluationProfile, config OpenAIDiagnosticConfig) error {
	if !isC1F3TerraChallengerProfile(profile) {
		if config.C1F3TerraChallengerExecutionAuthorization.OperatorOptIn {
			return fmt.Errorf("Terra challenger t1 execution authorization is scoped only to %s", C1F3TerraChallengerProfileIdentity)
		}
		return nil
	}
	if config.C1F3TerraChallengerExecutionAuthorization.Version != C1F3TerraChallengerExecutionAuthorizationVersion {
		return fmt.Errorf("Terra challenger t1 execution authorization identity is missing or incompatible")
	}
	expected, ok := loadC1F3TerraChallengerDiagnosticProfile(profile.Identity)
	if !ok || !reflect.DeepEqual(profile, expected) {
		return fmt.Errorf("Terra challenger t1 authorization scope does not match the registered frozen profile")
	}
	if profile.RequiredProvider != config.Runtime.Provider || profile.RequiredModel != config.Runtime.Model ||
		profile.RequiredExperimentID != config.ExperimentID || profile.RequiredOutputContractMode != config.OutputContractMode ||
		profile.EvidenceNamespace != config.EvidenceNamespace() || profile.CaseCount != 48 || profile.DefaultRepetitions != 1 ||
		len(profile.AllowedRepetitions) != 1 || profile.AllowedRepetitions[0] != 1 || !profile.RequiresTypedAttributionLabels ||
		config.ReasoningEffort != OpenAIDiagnosticReasoningEffort || config.BudgetCeilingMicros > 300_000 {
		return fmt.Errorf("Terra challenger t1 authorization scope does not match the exact frozen 48 x 1 cell")
	}
	return nil
}

func validateC1F3TerraProviderInputIsolation(manifest DiagnosticManifest, config OpenAIDiagnosticConfig, exposures []string) error {
	lunaConfig := config
	lunaConfig.Runtime.Model = OpenAIDiagnosticLunaModel
	lunaConfig.ExperimentID = C1F3RepeatabilityR3ExperimentID
	forbidden := []string{
		C1F3TerraAcceptedLunaRunID, C1F3TerraAcceptedLunaArtifactIndexSHA256, C1F3TerraChallengerRubricVersion,
		"expected_mapping_status", "expected_direct_issuer", "expected_proxy_exposure", "expected_issuer_attributions",
		"accepted_luna_baseline", "challenger", "materially better", "frozen_luna_failure_cases",
	}
	for _, event := range manifest.Events {
		terraRequest, err := diagnosticInitialRequest(config, event.Input, exposures)
		if err != nil {
			return err
		}
		lunaRequest, err := diagnosticInitialRequest(lunaConfig, event.Input, exposures)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(terraRequest, lunaRequest) {
			return fmt.Errorf("Terra challenger provider semantic request differs from accepted Luna request")
		}
		terraWire, err := marshalOpenAIDiagnosticRequest(config, terraRequest)
		if err != nil {
			return err
		}
		lunaWire, err := marshalOpenAIDiagnosticRequest(lunaConfig, lunaRequest)
		if err != nil {
			return err
		}
		var terraJSON, lunaJSON map[string]any
		if json.Unmarshal(terraWire, &terraJSON) != nil || json.Unmarshal(lunaWire, &lunaJSON) != nil {
			return fmt.Errorf("decode provider request for Terra/Luna isolation comparison")
		}
		terraModel, terraOK := terraJSON["model"].(string)
		lunaModel, lunaOK := lunaJSON["model"].(string)
		delete(terraJSON, "model")
		delete(lunaJSON, "model")
		if !terraOK || !lunaOK || terraModel != OpenAIDiagnosticTerraModel || lunaModel != OpenAIDiagnosticLunaModel || !reflect.DeepEqual(terraJSON, lunaJSON) {
			return fmt.Errorf("Terra challenger wire request changes a variable other than model")
		}
		visible := strings.ToLower(string(terraWire))
		for _, blocked := range forbidden {
			if strings.Contains(visible, strings.ToLower(blocked)) {
				return fmt.Errorf("Terra challenger provider-visible request contains control-plane metadata %q", blocked)
			}
		}
	}
	return nil
}

func validateC1F3TerraChallengerExecutionAuthorization(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1F3TerraChallengerAuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if !isC1F3TerraChallengerProfile(prepared.Profile) {
		return nil
	}
	plan := prepared.Plan.C1F3TerraChallengerExecutionAuthorization
	bindings := prepared.Plan.C1F3TerraChallengerFrozenBindings
	frozen, frozenErr := FrozenC1F3TerraChallengerProfile()
	if plan == nil || bindings == nil || plan.Version != C1F3TerraChallengerExecutionAuthorizationVersion ||
		plan.AuthorizationFingerprint != C1F3TerraChallengerExecutionAuthorizationFingerprint() || !plan.FrozenBindingsValid ||
		!plan.LunaPreservationValid || !plan.DecisionRubricValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree ||
		!plan.ProviderInputIsolated || !plan.OnlyModelVariableChanged || !plan.BoundaryExcluded || !plan.RuntimeSafetyValid ||
		frozenErr != nil || prepared.Plan.C1F3FrozenBindings == nil || bindings.SemanticStack != *prepared.Plan.C1F3FrozenBindings ||
		bindings.ProfileIdentity != C1F3TerraChallengerProfileIdentity || bindings.ProfileFingerprint != C1F3TerraChallengerProfileFingerprint ||
		bindings.AcceptedLuna.RunID != C1F3TerraAcceptedLunaRunID || bindings.AcceptedLuna.ArtifactIndexSHA256 != C1F3TerraAcceptedLunaArtifactIndexSHA256 ||
		bindings.DecisionRubric != frozen.DecisionRubric || bindings.ComparisonScoring != frozen.ComparisonScoring || bindings.EvidenceScoring != frozen.EvidenceScoring ||
		bindings.CaseCount != 48 || bindings.Repetitions != 1 || !bindings.BoundaryExcluded {
		return fmt.Errorf("Terra challenger t1 execution authorization plan is incomplete or invalid")
	}
	if !config.C1F3TerraChallengerExecutionAuthorization.OperatorOptIn {
		return fmt.Errorf("Terra challenger t1 execution requires the explicit --authorize-c1f3-terra-challenger-t1 operator opt-in")
	}
	if !config.InferenceExplicitlyAuthorized {
		return fmt.Errorf("Terra challenger t1 execution requires %s=true in addition to the Terra opt-in", OpenAIDiagnosticInferenceAuthEnv)
	}
	if !config.APIKey.present() {
		return fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !plan.OperatorOptIn || !plan.HostedInferenceAuthorized || !plan.CredentialPresent || !plan.ExecutionAuthorized {
		return fmt.Errorf("Terra challenger t1 execution authorization conditions are not all satisfied")
	}
	return ValidateDiagnosticExecutionShape(prepared)
}

func RevalidateC1F3TerraChallengerProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if !isC1F3TerraChallengerProfile(prepared.Profile) {
		return validateC1F3TerraChallengerAuthorizationScope(prepared.Profile, config)
	}
	revalidated, err := prepareHostedDiagnostic(prepared.Paths, config, prepared.Plan.Safety, true)
	if err != nil {
		return err
	}
	revalidated, err = ApplyDiagnosticExecutionShape(revalidated, prepared.ExecutionShape)
	if err != nil {
		return err
	}
	if err := validateC1F3TerraChallengerExecutionAuthorization(revalidated, config); err != nil {
		return err
	}
	if !reflect.DeepEqual(revalidated.Plan, prepared.Plan) || !reflect.DeepEqual(revalidated.ExecutionShape, prepared.ExecutionShape) {
		return fmt.Errorf("Terra challenger t1 provider-construction revalidation diverged from the prepared frozen plan")
	}
	return nil
}
