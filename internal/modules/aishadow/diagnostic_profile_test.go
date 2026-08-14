package aishadow

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func registeredProfilePaths(t *testing.T, identity string) (DiagnosticEvaluationProfile, DiagnosticPaths) {
	t.Helper()
	profile, err := LoadDiagnosticEvaluationProfile(identity)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "..")
	paths := DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        filepath.Join(root, filepath.FromSlash(profile.ManifestPath)),
		FingerprintLockPath: filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		AssetRulesetPath:    filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		OutputRoot:          t.TempDir(),
	}
	if profile.FreezePath != "" {
		paths.FreezePath = filepath.Join(root, filepath.FromSlash(profile.FreezePath))
	}
	return profile, paths
}

func profileRuntimeConfig(profile DiagnosticEvaluationProfile) Config {
	provider := profile.RequiredProvider
	model := profile.RequiredModel
	if provider == "" {
		provider, model = "ollama", "test-model"
	}
	return Config{
		Enabled: true, Provider: provider, Model: model, BaseURL: "https://example.invalid",
		Timeout: 120 * time.Second, MaxEvents: profile.CaseCount,
	}
}

func TestRegisteredDiagnosticProfilesAcceptOnlyTheirFrozenEvidenceAndCount(t *testing.T) {
	identities := []string{DiagnosticProfileOriginal, DiagnosticProfileGeneralization, DiagnosticProfileBoundary}
	for _, identity := range identities {
		identity := identity
		t.Run(identity, func(t *testing.T) {
			profile, paths := registeredProfilePaths(t, identity)
			prepared, err := PrepareDiagnostic(paths, profileRuntimeConfig(profile), diagnosticTestSafety())
			if err != nil {
				t.Fatal(err)
			}
			if prepared.Profile.Identity != profile.Identity || prepared.Plan.EvaluationProfile != profile.Identity ||
				prepared.Plan.DatasetIdentity != profile.ManifestVersion || prepared.Plan.ManifestFileSHA256 != profile.ManifestFileSHA256 ||
				prepared.Plan.FingerprintLockFileSHA256 != profile.FingerprintLockFileSHA256 || prepared.Plan.CasesPerRepetition != profile.CaseCount ||
				prepared.Plan.CausalConsistencyPolicy != CausalConsistencyPolicyVersion || len(prepared.Manifest.Events) != profile.CaseCount {
				t.Fatalf("registered profile plan is incomplete: %+v", prepared.Plan)
			}
			if profile.isHoldout() && (prepared.Freeze == nil || prepared.Plan.FreezeVersion != profile.FreezeVersion || prepared.Plan.FreezeFileSHA256 != profile.FreezeFileSHA256) {
				t.Fatalf("holdout freeze identity is missing: %+v", prepared.Plan)
			}

			wrongCount := profileRuntimeConfig(profile)
			if profile.CaseCount == 24 {
				wrongCount.MaxEvents = 48
			} else {
				wrongCount.MaxEvents = 24
			}
			if _, err := PrepareDiagnostic(paths, wrongCount, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "requires JAX_AI_MAX_EVENTS") {
				t.Fatalf("wrong frozen case count did not fail closed: %v", err)
			}
		})
	}

	if _, err := LoadDiagnosticEvaluationProfile("unregistered-evaluation"); err == nil || !strings.Contains(err.Error(), "unknown frozen") {
		t.Fatalf("unknown profile did not fail closed: %v", err)
	}
}

func TestHoldoutProfilesCannotBeSwappedOrReshaped(t *testing.T) {
	generalization, generalizationPaths := registeredProfilePaths(t, DiagnosticProfileGeneralization)
	boundary, boundaryPaths := registeredProfilePaths(t, DiagnosticProfileBoundary)

	swapped := generalizationPaths
	swapped.ManifestPath = boundaryPaths.ManifestPath
	if _, err := PrepareDiagnostic(swapped, profileRuntimeConfig(generalization), diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "manifest file hash changed") {
		t.Fatalf("Boundary manifest was accepted by Generalization profile: %v", err)
	}
	swapped = boundaryPaths
	swapped.FingerprintLockPath = generalizationPaths.FingerprintLockPath
	if _, err := PrepareDiagnostic(swapped, profileRuntimeConfig(boundary), diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "input lock file hash changed") {
		t.Fatalf("Generalization input lock was accepted by Boundary profile: %v", err)
	}

	prepared, err := PrepareDiagnostic(boundaryPaths, profileRuntimeConfig(boundary), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared.ExecutionShape.CasesPerRepetition = 48
	prepared.ExecutionShape.TotalPlannedCases = 48
	if err := ValidateDiagnosticExecutionShape(prepared); err == nil || !strings.Contains(err.Error(), "not permitted") {
		t.Fatalf("Boundary profile accepted a 48-case execution shape: %v", err)
	}

	prepared, err = PrepareDiagnostic(generalizationPaths, profileRuntimeConfig(generalization), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared.ExecutionShape.CasesPerRepetition = 24
	prepared.ExecutionShape.TotalPlannedCases = 24
	if err := ValidateDiagnosticExecutionShape(prepared); err == nil || !strings.Contains(err.Error(), "not permitted") {
		t.Fatalf("Generalization profile accepted a 24-case execution shape: %v", err)
	}
}

func TestHoldoutProviderRequestsContainOnlyFrozenEventInput(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralization, DiagnosticProfileBoundary} {
		identity := identity
		t.Run(identity, func(t *testing.T) {
			profile, paths := registeredProfilePaths(t, identity)
			prepared, err := PrepareDiagnostic(paths, profileRuntimeConfig(profile), diagnosticTestSafety())
			if err != nil {
				t.Fatal(err)
			}
			config := OpenAIDiagnosticConfig{
				Runtime: profileRuntimeConfig(profile), ExperimentID: profile.RequiredExperimentID,
				ReasoningEffort: OpenAIDiagnosticReasoningEffort, MaxOutputTokens: 256,
				OutputContractMode: OpenAIOutputContractStrictJSONSchema,
			}
			for _, event := range prepared.Manifest.Events {
				request, err := InitialRequest(event.Input, prepared.ProxyExposures)
				if err != nil {
					t.Fatal(err)
				}
				wireJSON, err := marshalOpenAIDiagnosticRequest(config, request)
				if err != nil {
					t.Fatal(err)
				}
				var wire openAIDiagnosticRequest
				if err := json.Unmarshal(wireJSON, &wire); err != nil {
					t.Fatal(err)
				}
				if len(wire.Input) != 2 {
					t.Fatalf("event %s has unexpected provider message count", event.ID)
				}
				var visible EventInput
				if err := json.Unmarshal([]byte(wire.Input[1].Content), &visible); err != nil {
					t.Fatalf("event %s provider input is not EventInput: %v", event.ID, err)
				}
				if !reflect.DeepEqual(visible, event.Input) {
					t.Fatalf("event %s provider-visible input diverged from frozen EventInput", event.ID)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal([]byte(wire.Input[1].Content), &fields); err != nil {
					t.Fatal(err)
				}
				for _, forbidden := range []string{"category", "adjudicated_label", "mapping_status", "direct_issuer", "proxy_exposure", "expected_resolution_status", "rationale"} {
					if _, found := fields[forbidden]; found {
						t.Fatalf("event %s leaked answer-side field %q to provider request", event.ID, forbidden)
					}
				}
			}
		})
	}
}

func TestHoldoutProfilesPinLunaStrictContractExperimentAndBudget(t *testing.T) {
	tests := []struct {
		identity     string
		experimentID string
		cases        string
		budget       string
	}{
		{DiagnosticProfileGeneralization, OpenAIGeneralizationExperimentID, "48", "0.20"},
		{DiagnosticProfileBoundary, OpenAIBoundaryExperimentID, "24", "0.10"},
	}
	for _, tt := range tests {
		t.Run(tt.identity, func(t *testing.T) {
			profile, err := LoadDiagnosticEvaluationProfile(tt.identity)
			if err != nil {
				t.Fatal(err)
			}
			values := hostedConfigValues()
			delete(values, OpenAIDiagnosticAPIKeyEnv)
			values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
			values["JAX_AI_EXPERIMENT_ID"] = tt.experimentID
			values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
			values["JAX_AI_MAX_EVENTS"] = tt.cases
			values["JAX_AI_MAX_OUTPUT_TOKENS"] = "256"
			values["JAX_AI_EXPERIMENT_BUDGET_USD"] = tt.budget
			values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
			values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.02"
			values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "0.25"
			values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "1.20"

			config, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, false)
			if err != nil {
				t.Fatal(err)
			}
			if config.APIKey.present() || config.Runtime.Model != OpenAIDiagnosticLunaModel || config.Runtime.MaxEvents != profile.CaseCount ||
				config.ExperimentID != profile.RequiredExperimentID || config.OutputContractMode != OpenAIOutputContractStrictJSONSchema ||
				config.MaxOutputTokens != 256 || config.ReasoningEffort != "none" || config.ServiceTier() != "default" ||
				config.EvidenceNamespace() != profile.EvidenceNamespace {
				t.Fatalf("holdout provider configuration is not pinned: %+v", config)
			}

			values["JAX_AI_EXPERIMENT_ID"] = OpenAIStructuredOutputsExperimentID
			if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, false); err == nil || !strings.Contains(err.Error(), "frozen profile") {
				t.Fatalf("holdout accepted the C1B experiment identity: %v", err)
			}
		})
	}
}
