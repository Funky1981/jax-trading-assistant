package aishadow

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const C1E3ExecutionAuthorizationVersion = "ai-shadow-c1e3-execution-authorization-v1"

// C1E3ExecutionAuthorization is process-local control-plane state. It is
// supplied only by the diagnostic CLI flag and is never read from an
// environment variable or persisted as reusable authorization.
type C1E3ExecutionAuthorization struct {
	Version       string
	OperatorOptIn bool
}

type C1E3ExecutionAuthorizationPlan struct {
	Version                        string `json:"version"`
	OperatorOptIn                  bool   `json:"operator_opt_in"`
	HostedInferenceAuthorized      bool   `json:"hosted_inference_authorized"`
	CredentialPresent              bool   `json:"credential_present"`
	FrozenInputsValid              bool   `json:"frozen_inputs_valid"`
	BudgetValid                    bool   `json:"budget_valid"`
	EvidenceNamespaceCollisionFree bool   `json:"evidence_namespace_collision_free"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
}

func NewC1E3ExecutionAuthorization(operatorOptIn bool) C1E3ExecutionAuthorization {
	return C1E3ExecutionAuthorization{Version: C1E3ExecutionAuthorizationVersion, OperatorOptIn: operatorOptIn}
}

func isC1E3Profile(profile DiagnosticEvaluationProfile) bool {
	return profile.Identity == DiagnosticProfileGeneralizationV2 || profile.Identity == DiagnosticProfileBoundaryV2
}

func validateC1E3AuthorizationScope(profile DiagnosticEvaluationProfile, config OpenAIDiagnosticConfig) error {
	if !isC1E3Profile(profile) {
		if config.C1E3ExecutionAuthorization.OperatorOptIn {
			return fmt.Errorf("C1E3 execution authorization is scoped only to the registered frozen C1E3 profiles")
		}
		return nil
	}
	if config.C1E3ExecutionAuthorization.Version != C1E3ExecutionAuthorizationVersion {
		return fmt.Errorf("C1E3 execution authorization identity is missing or incompatible")
	}
	if profile.RequiredProvider != OpenAIDiagnosticProvider || profile.RequiredModel != OpenAIDiagnosticLunaModel ||
		profile.RequiredExperimentID != config.ExperimentID || profile.EvidenceNamespace != config.EvidenceNamespace() ||
		profile.RequiredOutputContractMode != OpenAIOutputContractStrictJSONSchema || profile.DefaultRepetitions != 1 ||
		len(profile.AllowedRepetitions) != 1 || profile.AllowedRepetitions[0] != 1 || !profile.RequiresTypedAttributionLabels {
		return fmt.Errorf("C1E3 execution authorization scope does not match the registered frozen profile")
	}
	return nil
}

func c1e3EvidenceNamespaceCollisionFree(root string) (bool, error) {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() != "preflight" || !entry.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

func validateC1E3ExecutionAuthorization(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if err := validateC1E3AuthorizationScope(prepared.Profile, config); err != nil {
		return err
	}
	if !isC1E3Profile(prepared.Profile) {
		return nil
	}
	plan := prepared.Plan.C1E3ExecutionAuthorization
	if plan == nil || plan.Version != C1E3ExecutionAuthorizationVersion || !plan.FrozenInputsValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree {
		return fmt.Errorf("C1E3 execution authorization plan is incomplete or invalid")
	}
	if !config.C1E3ExecutionAuthorization.OperatorOptIn {
		return fmt.Errorf("C1E3 execution requires the explicit --authorize-c1e3-execution operator opt-in")
	}
	if !config.InferenceExplicitlyAuthorized {
		return fmt.Errorf("C1E3 execution requires %s=true in addition to the experiment-specific opt-in", OpenAIDiagnosticInferenceAuthEnv)
	}
	if !config.APIKey.present() {
		return fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !plan.OperatorOptIn || !plan.HostedInferenceAuthorized || !plan.CredentialPresent || !plan.ExecutionAuthorized {
		return fmt.Errorf("C1E3 execution authorization conditions are not all satisfied")
	}
	return ValidateDiagnosticExecutionShape(prepared)
}

// RevalidateC1E3ProviderConstruction repeats every frozen preparation and
// authorization check immediately before the CLI constructs a provider.
func RevalidateC1E3ProviderConstruction(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) error {
	if !isC1E3Profile(prepared.Profile) {
		return validateC1E3AuthorizationScope(prepared.Profile, config)
	}
	revalidated, err := prepareHostedDiagnostic(prepared.Paths, config, prepared.Plan.Safety, true)
	if err != nil {
		return err
	}
	revalidated, err = ApplyDiagnosticExecutionShape(revalidated, prepared.ExecutionShape)
	if err != nil {
		return err
	}
	if err := validateC1E3ExecutionAuthorization(revalidated, config); err != nil {
		return err
	}
	if !reflect.DeepEqual(revalidated.Plan, prepared.Plan) || !reflect.DeepEqual(revalidated.ExecutionShape, prepared.ExecutionShape) {
		return fmt.Errorf("C1E3 provider-construction revalidation diverged from the prepared frozen plan")
	}
	return nil
}

func c1e3RepositoryRoot(assetRulesetPath string) string {
	clean := filepath.Clean(assetRulesetPath)
	return filepath.Dir(filepath.Dir(clean))
}

func validateC1E3FrozenSemanticSources(paths DiagnosticPaths, resolverVersion string, exposures []string) error {
	if rawHash(v5SystemPrompt) != frozenV5PromptSHA256 {
		return fmt.Errorf("frozen v5 prompt hash changed")
	}
	schemaSHA, err := fingerprint(V5OutputSchema(exposures))
	if err != nil || schemaSHA != frozenV5SchemaSHA256 {
		return fmt.Errorf("frozen v5 schema hash changed")
	}
	root := c1e3RepositoryRoot(paths.AssetRulesetPath)
	checks := []struct {
		path string
		want string
	}{
		{filepath.Join(root, "internal", "modules", "aishadow", "causal_attribution.go"), frozenC1EPolicySHA256},
		{filepath.Join(root, "internal", "modules", "aishadow", "causal_consistency.go"), "1f07d7854ef733a55a8172419b060ce5806ebf28fad3ab7ab8392e3cc5bdd895"},
	}
	for _, check := range checks {
		got, hashErr := diagnosticFileSHA256(check.path)
		if hashErr != nil || got != check.want {
			return fmt.Errorf("frozen semantic source hash changed for %s", filepath.ToSlash(check.path))
		}
	}
	if strings.TrimSpace(resolverVersion) != "event-asset-resolution-v1" {
		return fmt.Errorf("frozen resolver identity changed")
	}
	return nil
}
