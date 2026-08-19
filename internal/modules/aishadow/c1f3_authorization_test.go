package aishadow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func c1f3AuthorizationTestConfig(t *testing.T, profile DiagnosticEvaluationProfile, hostedAuthorized, operatorOptIn, credential bool) OpenAIDiagnosticConfig {
	t.Helper()
	values := c1e2bConfigValues(profile)
	values[OpenAIDiagnosticInferenceAuthEnv] = "false"
	if hostedAuthorized {
		values[OpenAIDiagnosticInferenceAuthEnv] = "true"
	}
	if credential {
		values[OpenAIDiagnosticAPIKeyEnv] = "offline-test-presence-only"
	}
	config, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, credential)
	if err != nil {
		t.Fatal(err)
	}
	config.C1F3ExecutionAuthorization = NewC1F3ExecutionAuthorization(operatorOptIn)
	return config
}

func c1f3AuthorizationTestPaths(t *testing.T, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	return DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        filepath.Join(root, filepath.FromSlash(profile.ManifestPath)),
		FingerprintLockPath: filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		FreezePath:          filepath.Join(root, filepath.FromSlash(profile.FreezePath)),
		AssetRulesetPath:    filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		TypedLabelPath:      filepath.Join(root, filepath.FromSlash(profile.TypedLabelPath)),
		ScoringRubricPath:   filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath)),
		OutputRoot:          filepath.Join(t.TempDir(), profile.EvidenceNamespace, profile.RequiredExperimentID),
	}
}

func c1f3PreparedAuthorized(t *testing.T, identity string) (PreparedDiagnostic, OpenAIDiagnosticConfig) {
	t.Helper()
	profile, err := LoadDiagnosticExecutionProfile(identity)
	if err != nil {
		t.Fatal(err)
	}
	config := c1f3AuthorizationTestConfig(t, profile, true, true, true)
	prepared, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	return prepared, config
}

func TestC1F3ExecutionAuthorizationMatrixDefaultsToDeny(t *testing.T) {
	profile, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	tests := []struct {
		name             string
		hostedAuthorized bool
		operatorOptIn    bool
		want             string
	}{
		{name: "both absent", want: "explicit --authorize-c1f3-execution"},
		{name: "global only", hostedAuthorized: true, want: "explicit --authorize-c1f3-execution"},
		{name: "C1F3 opt-in only", operatorOptIn: true, want: OpenAIDiagnosticInferenceAuthEnv + "=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := c1f3AuthorizationTestConfig(t, profile, tt.hostedAuthorized, tt.operatorOptIn, true)
			_, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("C1F3 authorization combination did not fail closed: %v", err)
			}
		})
	}
}

func TestC1F3CredentiallessZeroNetworkPreflight(t *testing.T) {
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, _ := LoadDiagnosticExecutionProfile(identity)
		config := c1f3AuthorizationTestConfig(t, profile, false, false, false)
		prepared, err := PrepareHostedDiagnosticPreflight(c1f3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
		if err != nil {
			t.Fatal(err)
		}
		prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
		if err != nil {
			t.Fatal(err)
		}
		plan := prepared.Plan.C1F3ExecutionAuthorization
		if plan == nil || plan.Version != C1F3ExecutionAuthorizationVersion || plan.OperatorOptIn || plan.HostedInferenceAuthorized ||
			plan.CredentialPresent || !plan.FrozenBindingsValid || !plan.BudgetValid || !plan.EvidenceNamespaceCollisionFree ||
			!plan.ProviderInputIsolated || plan.ExecutionAuthorized || prepared.Plan.C1F3FrozenBindings == nil {
			t.Fatalf("credentialless C1F3 preflight state is wrong: %+v", plan)
		}
		if prepared.Plan.PromptVersion != V6PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion ||
			prepared.Plan.HostedExperiment.SchemaSHA256 != frozenV5SchemaSHA256 {
			t.Fatalf("credentialless C1F3 preflight lost frozen execution bindings: %+v", prepared.Plan)
		}
		if _, _, err := WriteDiagnosticPreflight(prepared); err != nil {
			t.Fatal(err)
		}
	}
}

func TestC1F3ExactProfilesReachMockableProviderConstructionGate(t *testing.T) {
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		prepared, config := c1f3PreparedAuthorized(t, identity)
		if err := RevalidateOpenAIProviderConstruction(prepared, config); err != nil {
			t.Fatal(err)
		}
		if plan := prepared.Plan.C1F3ExecutionAuthorization; plan == nil || !plan.ExecutionAuthorized {
			t.Fatalf("fully authorized exact C1F3 profile did not reach construction gate: %+v", plan)
		}
	}
}

func TestC1F3RejectsWrongCellShapeCredentialAndCollision(t *testing.T) {
	profile, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)

	t.Run("wrong experiment", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_EXPERIMENT_ID"] = OpenAIC1F3BoundaryExperimentID
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil || !strings.Contains(err.Error(), "frozen profile") {
			t.Fatalf("wrong C1F3 experiment was accepted: %v", err)
		}
	})

	t.Run("wrong model and provider", func(t *testing.T) {
		for _, mutate := range []func(map[string]string){
			func(values map[string]string) { values["JAX_AI_MODEL"] = OpenAIDiagnosticSolModel },
			func(values map[string]string) { values["JAX_AI_PROVIDER"] = "deepseek" },
		} {
			values := c1e2bConfigValues(profile)
			mutate(values)
			if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
				t.Fatal("wrong C1F3 model/provider was accepted")
			}
		}
	})

	t.Run("wrong case count", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_MAX_EVENTS"] = "47"
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil || !strings.Contains(err.Error(), "JAX_AI_MAX_EVENTS=48") {
			t.Fatalf("wrong C1F3 case count was accepted: %v", err)
		}
	})

	t.Run("wrong repetition", func(t *testing.T) {
		if _, err := LoadDiagnosticRepetitionSelectionForProfile(func(string) (string, bool) { return "2", true }, profile); err == nil {
			t.Fatal("wrong C1F3 repetition was accepted")
		}
	})

	t.Run("wrong dataset input lock freeze sidecar and rubric paths", func(t *testing.T) {
		config := c1f3AuthorizationTestConfig(t, profile, true, true, true)
		base := c1f3AuthorizationTestPaths(t, profile)
		for _, mutate := range []func(*DiagnosticPaths){
			func(paths *DiagnosticPaths) { paths.ManifestPath = paths.FingerprintLockPath },
			func(paths *DiagnosticPaths) { paths.FingerprintLockPath = paths.FreezePath },
			func(paths *DiagnosticPaths) { paths.FreezePath = paths.ManifestPath },
			func(paths *DiagnosticPaths) { paths.TypedLabelPath = paths.ScoringRubricPath },
			func(paths *DiagnosticPaths) { paths.ScoringRubricPath = paths.TypedLabelPath },
		} {
			paths := base
			mutate(&paths)
			if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil {
				t.Fatal("wrong frozen C1F3 path was accepted")
			}
		}
	})

	t.Run("missing credential", func(t *testing.T) {
		config := c1f3AuthorizationTestConfig(t, profile, true, true, false)
		if _, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), OpenAIDiagnosticAPIKeyEnv) {
			t.Fatalf("missing C1F3 credential was accepted: %v", err)
		}
	})

	t.Run("colliding evidence namespace", func(t *testing.T) {
		config := c1f3AuthorizationTestConfig(t, profile, true, true, true)
		paths := c1f3AuthorizationTestPaths(t, profile)
		if err := os.MkdirAll(filepath.Join(paths.OutputRoot, "existing-run"), 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "already contains execution evidence") {
			t.Fatalf("colliding C1F3 evidence namespace was accepted: %v", err)
		}
	})
}

func TestC1F3RejectsEveryFrozenBindingMutation(t *testing.T) {
	prepared, config := c1f3PreparedAuthorized(t, C1F3ProfileGeneralization)
	mutations := map[string]func(*C1F3FrozenBindingPlan){
		"dataset":             func(plan *C1F3FrozenBindingPlan) { plan.ManifestSHA256 = "wrong" },
		"input lock":          func(plan *C1F3FrozenBindingPlan) { plan.InputLockFingerprint = "wrong" },
		"freeze":              func(plan *C1F3FrozenBindingPlan) { plan.FreezeSHA256 = "wrong" },
		"sidecar":             func(plan *C1F3FrozenBindingPlan) { plan.TypedSidecarFingerprint = "wrong" },
		"adjudication rubric": func(plan *C1F3FrozenBindingPlan) { plan.AdjudicationRubricSHA256 = "wrong" },
		"scoring":             func(plan *C1F3FrozenBindingPlan) { plan.ScoringSHA256 = "wrong" },
		"scoring rubric":      func(plan *C1F3FrozenBindingPlan) { plan.ScoringRubricSHA256 = "wrong" },
		"prompt":              func(plan *C1F3FrozenBindingPlan) { plan.PromptSHA256 = "wrong" },
		"output":              func(plan *C1F3FrozenBindingPlan) { plan.OutputContractSHA256 = "wrong" },
		"validator":           func(plan *C1F3FrozenBindingPlan) { plan.ValidatorSHA256 = "wrong" },
		"policy":              func(plan *C1F3FrozenBindingPlan) { plan.AttributionPolicySHA256 = "wrong" },
		"comparator":          func(plan *C1F3FrozenBindingPlan) { plan.SemanticComparatorSHA256 = "wrong" },
		"resolver":            func(plan *C1F3FrozenBindingPlan) { plan.ResolverSHA256 = "wrong" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := prepared
			binding := *prepared.Plan.C1F3FrozenBindings
			mutate(&binding)
			changed.Plan.C1F3FrozenBindings = &binding
			if err := validateC1F3ExecutionAuthorization(changed, config); err == nil {
				t.Fatal("mutated C1F3 frozen binding was accepted")
			}
		})
	}
}

func TestC1E3AndC1F3OptInsCannotCrossAuthorize(t *testing.T) {
	c1f3, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	c1f3Config := c1f3AuthorizationTestConfig(t, c1f3, true, false, true)
	c1f3Config.C1E3ExecutionAuthorization = NewC1E3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, c1f3), c1f3Config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "C1E3 execution authorization is scoped only") {
		t.Fatalf("C1E3 flag cross-authorized C1F3: %v", err)
	}

	c1e3, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	c1e3Config := c1e3AuthorizationTestConfig(t, c1e3, true, true, true)
	c1e3Config.C1F3ExecutionAuthorization = NewC1F3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(c1e3AuthorizationTestPaths(t, c1e3), c1e3Config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "C1F3 execution authorization is scoped only") {
		t.Fatalf("C1F3 flag cross-authorized C1E3: %v", err)
	}
}

func TestC1F3ControlPlaneRetainsHistoricalRegistryAndProviderInputIsolation(t *testing.T) {
	if _, err := LoadDiagnosticEvaluationProfile(C1F3ProfileGeneralization); err == nil {
		t.Fatal("C1F3 entered the historical hosted executor registry")
	}
	for _, identity := range []string{DiagnosticProfileOriginal, DiagnosticProfileGeneralization, DiagnosticProfileBoundary, DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		if _, err := LoadDiagnosticEvaluationProfile(identity); err != nil {
			t.Fatalf("historical profile %s changed semantics: %v", identity, err)
		}
	}
	profile, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	config := c1f3AuthorizationTestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(c1f3AuthorizationTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateC1F3ProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures); err != nil {
		t.Fatal(err)
	}
}
