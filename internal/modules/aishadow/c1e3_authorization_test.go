package aishadow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func c1e3AuthorizationTestConfig(t *testing.T, profile DiagnosticEvaluationProfile, hostedAuthorized, operatorOptIn, credential bool) OpenAIDiagnosticConfig {
	t.Helper()
	values := hostedConfigValues()
	values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
	values["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
	values["JAX_AI_MAX_EVENTS"] = itoa(profile.CaseCount)
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = formatUSDMicros(profile.MaximumBudgetMicros)
	values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
	values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.02"
	values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "0.25"
	values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "1.20"
	values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
	values[OpenAIDiagnosticInferenceAuthEnv] = "false"
	if hostedAuthorized {
		values[OpenAIDiagnosticInferenceAuthEnv] = "true"
	}
	if !credential {
		delete(values, OpenAIDiagnosticAPIKeyEnv)
	}
	config, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, credential)
	if err != nil {
		t.Fatal(err)
	}
	config.C1E3ExecutionAuthorization = NewC1E3ExecutionAuthorization(operatorOptIn)
	return config
}

func c1e3AuthorizationTestPaths(t *testing.T, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	t.Helper()
	paths := c1e2aProfilePaths(profile)
	paths.OutputRoot = filepath.Join(t.TempDir(), profile.EvidenceNamespace, profile.RequiredExperimentID)
	return paths
}

func TestC1E3ExecutionAuthorizationMatrixDefaultsToDeny(t *testing.T) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	tests := []struct {
		name             string
		hostedAuthorized bool
		operatorOptIn    bool
		want             string
	}{
		{name: "both absent", want: "explicit --authorize-c1e3-execution"},
		{name: "global only", hostedAuthorized: true, want: "explicit --authorize-c1e3-execution"},
		{name: "experiment opt-in only", operatorOptIn: true, want: OpenAIDiagnosticInferenceAuthEnv + "=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := c1e3AuthorizationTestConfig(t, profile, tt.hostedAuthorized, tt.operatorOptIn, true)
			_, err := PrepareHostedDiagnostic(c1e3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("authorization combination did not fail closed: %v", err)
			}
		})
	}
}

func TestC1E3CredentiallessPreflightIsAllowedAndUnauthorised(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, _ := LoadDiagnosticEvaluationProfile(identity)
		config := c1e3AuthorizationTestConfig(t, profile, false, false, false)
		prepared, err := PrepareHostedDiagnosticPreflight(c1e3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
		if err != nil {
			t.Fatal(err)
		}
		prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
		if err != nil {
			t.Fatal(err)
		}
		plan := prepared.Plan.C1E3ExecutionAuthorization
		if plan == nil || plan.Version != C1E3ExecutionAuthorizationVersion || plan.OperatorOptIn || plan.HostedInferenceAuthorized ||
			plan.CredentialPresent || !plan.FrozenInputsValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree || plan.ExecutionAuthorized {
			t.Fatalf("credentialless C1E3 preflight authorization state is wrong: %+v", plan)
		}
		if prepared.Plan.PromptVersion != V5PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion ||
			prepared.Plan.ModelConfiguration.SchemaContract != V5SchemaVersion || prepared.Plan.HostedExperiment.SchemaSHA256 != frozenV5SchemaSHA256 {
			t.Fatalf("credentialless preflight lost frozen v5 bindings: %+v", prepared.Plan)
		}
		paths, _, err := WriteDiagnosticPreflight(prepared)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(paths.Preflight)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"provider_contact": false`, `"inference": false`, `"execution_authorized": false`} {
			if !strings.Contains(string(raw), want) {
				t.Fatalf("preflight evidence missing %s: %s", want, raw)
			}
		}
	}
}

func TestC1E3ExactProfilesReachProviderConstructionGateOnlyWhenFullyAuthorized(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, _ := LoadDiagnosticEvaluationProfile(identity)
		config := c1e3AuthorizationTestConfig(t, profile, true, true, true)
		prepared, err := PrepareHostedDiagnostic(c1e3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
		if err != nil {
			t.Fatal(err)
		}
		prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
		if err != nil {
			t.Fatal(err)
		}
		if err := RevalidateC1E3ProviderConstruction(prepared, config); err != nil {
			t.Fatal(err)
		}
		if plan := prepared.Plan.C1E3ExecutionAuthorization; plan == nil || !plan.ExecutionAuthorized {
			t.Fatalf("fully authorized exact profile did not reach provider-construction gate: %+v", plan)
		}
	}
}

func TestC1E3AuthorizationRejectsWrongScopeShapeEvidenceAndCredential(t *testing.T) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)

	t.Run("wrong profile", func(t *testing.T) {
		other, paths := registeredProfilePaths(t, DiagnosticProfileGeneralization)
		config := c1e3AuthorizationTestConfig(t, other, true, true, true)
		paths.OutputRoot = filepath.Join(t.TempDir(), other.EvidenceNamespace, other.RequiredExperimentID)
		if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
			t.Fatalf("C1E3 opt-in authorized another profile: %v", err)
		}
	})

	t.Run("wrong experiment", func(t *testing.T) {
		values := hostedConfigValues()
		values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
		values["JAX_AI_EXPERIMENT_ID"] = OpenAIC1E3BoundaryExperimentID
		values["JAX_AI_MAX_EVENTS"] = "48"
		values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, true); err == nil || !strings.Contains(err.Error(), "frozen profile") {
			t.Fatalf("wrong experiment was accepted: %v", err)
		}
	})

	t.Run("wrong model and provider", func(t *testing.T) {
		for _, mutate := range []func(map[string]string){
			func(values map[string]string) { values["JAX_AI_MODEL"] = OpenAIDiagnosticSolModel },
			func(values map[string]string) { values["JAX_AI_PROVIDER"] = "deepseek" },
		} {
			values := hostedConfigValues()
			values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
			values["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
			values["JAX_AI_MAX_EVENTS"] = "48"
			values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
			mutate(values)
			if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, true); err == nil {
				t.Fatal("wrong model/provider was accepted")
			}
		}
	})

	t.Run("wrong case count", func(t *testing.T) {
		values := hostedConfigValues()
		values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
		values["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
		values["JAX_AI_MAX_EVENTS"] = "47"
		values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, true); err == nil || !strings.Contains(err.Error(), "JAX_AI_MAX_EVENTS=48") {
			t.Fatalf("wrong case count was accepted: %v", err)
		}
	})

	t.Run("wrong repetition", func(t *testing.T) {
		if _, err := LoadDiagnosticRepetitionSelectionForProfile(func(string) (string, bool) { return "2", true }, profile); err == nil {
			t.Fatal("wrong repetition was accepted")
		}
	})

	t.Run("missing credential", func(t *testing.T) {
		values := hostedConfigValues()
		delete(values, OpenAIDiagnosticAPIKeyEnv)
		values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
		values["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
		values["JAX_AI_MAX_EVENTS"] = "48"
		values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, true); err == nil || !strings.Contains(err.Error(), OpenAIDiagnosticAPIKeyEnv) {
			t.Fatalf("missing credential was accepted: %v", err)
		}
	})

	t.Run("execution evidence collision", func(t *testing.T) {
		config := c1e3AuthorizationTestConfig(t, profile, true, true, true)
		paths := c1e3AuthorizationTestPaths(t, profile)
		if err := os.MkdirAll(filepath.Join(paths.OutputRoot, "existing-run"), 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "already contains execution evidence") {
			t.Fatalf("execution evidence collision was accepted: %v", err)
		}
	})
}
