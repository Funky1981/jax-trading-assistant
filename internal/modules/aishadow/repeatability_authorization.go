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
	C1F3RepeatabilityExecutionAuthorizationVersion = "ai-shadow-c1f3-repeatability-execution-authorization-v1"
	C1F3RepeatabilityAuthorizationFingerprint      = "c0a389adcc6df98f2d8655686e9275fa323752614eed929e11b5f59d2810d9d9"
)

type C1F3RepeatabilityExecutionAuthorization struct {
	Version       string
	OperatorOptIn bool
}

type C1F3RepeatabilityExecutionAuthorizationPlan struct {
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
	RuntimeSafetyValid             bool   `json:"runtime_safety_valid"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
}

type c1f3RepeatabilityAuthorizationDefinition struct {
	Identity                    string `json:"identity"`
	ApplicableProfile           string `json:"applicable_profile"`
	BaselineRunID               string `json:"baseline_run_id"`
	BaselineArtifactIndexSHA256 string `json:"baseline_artifact_index_sha256"`
	OperatorOptInMechanism      string `json:"operator_opt_in_mechanism"`
	HostedAuthorizationRequired string `json:"hosted_authorization_required"`
	CaseCount                   int    `json:"case_count"`
	Repetitions                 int    `json:"repetitions"`
	DefaultDeny                 bool   `json:"default_deny"`
}

func NewC1F3RepeatabilityExecutionAuthorization(operatorOptIn bool) C1F3RepeatabilityExecutionAuthorization {
	return C1F3RepeatabilityExecutionAuthorization{Version: C1F3RepeatabilityExecutionAuthorizationVersion, OperatorOptIn: operatorOptIn}
}

func C1F3RepeatabilityExecutionAuthorizationFingerprint() string {
	definition := c1f3RepeatabilityAuthorizationDefinition{
		Identity: C1F3RepeatabilityExecutionAuthorizationVersion, ApplicableProfile: C1F3RepeatabilityProfileIdentity,
		BaselineRunID: C1F3GeneralizationSourceRunID, BaselineArtifactIndexSHA256: C1F3GeneralizationArtifactIndex,
		OperatorOptInMechanism: "--authorize-c1f3-repeatability", HostedAuthorizationRequired: OpenAIDiagnosticInferenceAuthEnv + "=true",
		CaseCount: 48, Repetitions: 1, DefaultDeny: true,
	}
	raw, _ := json.Marshal(definition)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func validateC1F3RepeatabilityAuthorizationScope(profile DiagnosticEvaluationProfile, config OpenAIDiagnosticConfig) error {
	if !isC1F3RepeatabilityR2Profile(profile) {
		if config.C1F3RepeatabilityExecutionAuthorization.OperatorOptIn {
			return fmt.Errorf("C1F3 repeatability execution authorization is scoped only to %s", C1F3RepeatabilityProfileIdentity)
		}
		return nil
	}
	if config.C1F3RepeatabilityExecutionAuthorization.Version != C1F3RepeatabilityExecutionAuthorizationVersion {
		return fmt.Errorf("C1F3 repeatability execution authorization identity is missing or incompatible")
	}
	expected, ok := loadC1F3RepeatabilityDiagnosticProfile(profile.Identity)
	if !ok || !reflect.DeepEqual(profile, expected) {
		return fmt.Errorf("C1F3 repeatability authorization scope does not match the registered frozen profile")
	}
	if profile.RequiredProvider != config.Runtime.Provider || profile.RequiredModel != config.Runtime.Model ||
		profile.RequiredExperimentID != config.ExperimentID || profile.RequiredOutputContractMode != config.OutputContractMode ||
		profile.EvidenceNamespace != config.EvidenceNamespace() || profile.CaseCount != 48 || profile.DefaultRepetitions != 1 ||
		len(profile.AllowedRepetitions) != 1 || profile.AllowedRepetitions[0] != 1 || !profile.RequiresTypedAttributionLabels {
		return fmt.Errorf("C1F3 repeatability authorization scope does not match the exact 48 x 1 execution cell")
	}
	return nil
}

func validateC1F3RepeatabilityProviderInputIsolation(manifest DiagnosticManifest, config OpenAIDiagnosticConfig, exposures []string) error {
	baselineConfig := config
	baselineConfig.ExperimentID = OpenAIC1F3GeneralizationExperimentID
	forbidden := []string{
		C1F3GeneralizationSourceRunID, C1F3GeneralizationArtifactIndex, C1F3RepeatabilityScoringVersion,
		C1F3GeneralizationTypedLabelsVersion, C1F3AAcceptedReprojectionFingerprint,
		C1F3RepeatabilityProfileIdentity, C1F3RepeatabilityR2FailedRunID, C1F3RepeatabilityR2PreflightRunID,
		C1F3RepeatabilityR2PreflightSHA256, C1F3RepeatabilityR2FailureDispositionVersion,
		"baseline_mapping", "repeatability_comparison", "original_luna_answer", "failure_ids",
	}
	for _, event := range manifest.Events {
		repeatRequest, err := diagnosticInitialRequest(config, event.Input, exposures)
		if err != nil {
			return err
		}
		baselineRequest, err := diagnosticInitialRequest(baselineConfig, event.Input, exposures)
		if err != nil {
			return err
		}
		repeatWire, err := marshalOpenAIDiagnosticRequest(config, repeatRequest)
		if err != nil {
			return err
		}
		baselineWire, err := marshalOpenAIDiagnosticRequest(baselineConfig, baselineRequest)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(repeatRequest, baselineRequest) || string(repeatWire) != string(baselineWire) {
			return fmt.Errorf("C1F3 repeatability provider-visible request differs from original C1F3 Generalization")
		}
		if isR3 := config.ExperimentID == C1F3RepeatabilityR3ExperimentID; isR3 {
			r2Config := config
			r2Config.ExperimentID = C1F3RepeatabilityExperimentID
			r2Request, err := diagnosticInitialRequest(r2Config, event.Input, exposures)
			if err != nil {
				return err
			}
			r2Wire, err := marshalOpenAIDiagnosticRequest(r2Config, r2Request)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(repeatRequest, r2Request) || string(repeatWire) != string(r2Wire) {
				return fmt.Errorf("C1F3 repeatability r3 provider-visible request differs from consumed r2 request")
			}
		}
		visible := strings.ToLower(string(repeatWire))
		for _, blocked := range forbidden {
			if strings.Contains(visible, strings.ToLower(blocked)) {
				return fmt.Errorf("C1F3 repeatability provider-visible request contains control-plane metadata %q", blocked)
			}
		}
	}
	return nil
}

func validateC1F3RepeatabilityExecutionAuthorization(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1F3RepeatabilityAuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if !isC1F3RepeatabilityR2Profile(prepared.Profile) {
		return nil
	}
	return fmt.Errorf("C1F3 repeatability r2 cell is permanently consumed and cannot be executed")
}

func RevalidateC1F3RepeatabilityProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if !isC1F3RepeatabilityR2Profile(prepared.Profile) {
		return validateC1F3RepeatabilityAuthorizationScope(prepared.Profile, config)
	}
	revalidated, err := prepareHostedDiagnostic(prepared.Paths, config, prepared.Plan.Safety, true)
	if err != nil {
		return err
	}
	revalidated, err = ApplyDiagnosticExecutionShape(revalidated, prepared.ExecutionShape)
	if err != nil {
		return err
	}
	if err := validateC1F3RepeatabilityExecutionAuthorization(revalidated, config); err != nil {
		return err
	}
	if !reflect.DeepEqual(revalidated.Plan, prepared.Plan) || !reflect.DeepEqual(revalidated.ExecutionShape, prepared.ExecutionShape) {
		return fmt.Errorf("C1F3 repeatability provider-construction revalidation diverged from the prepared frozen plan")
	}
	return nil
}
