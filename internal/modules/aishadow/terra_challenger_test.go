package aishadow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func terraChallengerTestProfile(t *testing.T) DiagnosticEvaluationProfile {
	t.Helper()
	profile, err := LoadDiagnosticExecutionProfile(C1F3TerraChallengerProfileIdentity)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func terraChallengerTestConfig(t *testing.T, profile DiagnosticEvaluationProfile, hosted, optIn, credential bool) OpenAIDiagnosticConfig {
	t.Helper()
	values := c1e2bConfigValues(profile)
	values["JAX_AI_MODEL"] = OpenAIDiagnosticTerraModel
	values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "2.00"
	values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
	values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "2.50"
	values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "12.00"
	if hosted {
		values[OpenAIDiagnosticInferenceAuthEnv] = "true"
	}
	if credential {
		values[OpenAIDiagnosticAPIKeyEnv] = "offline-test-presence-only"
	}
	config, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, credential)
	if err != nil {
		t.Fatal(err)
	}
	config.C1F3TerraChallengerExecutionAuthorization = NewC1F3TerraChallengerExecutionAuthorization(optIn)
	return config
}

func terraChallengerTestPaths(t *testing.T, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	t.Helper()
	return c1e2bProfilePaths(filepath.Join("..", "..", ".."), t.TempDir(), profile)
}

func terraChallengerPreparedAuthorized(t *testing.T) (PreparedDiagnostic, OpenAIDiagnosticConfig) {
	t.Helper()
	profile := terraChallengerTestProfile(t)
	config := terraChallengerTestConfig(t, profile, true, true, true)
	prepared, err := PrepareHostedDiagnostic(terraChallengerTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	return prepared, config
}

func TestC1F3TerraChallengerFrozenIdentityAndRubric(t *testing.T) {
	profile, err := FrozenC1F3TerraChallengerProfile()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := profile.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != C1F3TerraChallengerProfileFingerprint {
		t.Fatalf("Terra profile fingerprint changed: %s", fingerprint)
	}
	if got := C1F3TerraChallengerExecutionAuthorizationFingerprint(); got != C1F3TerraChallengerAuthorizationFingerprint {
		t.Fatalf("Terra authorization fingerprint changed: %s", got)
	}
	rubric, err := LoadFrozenC1F3TerraChallengerRubric(filepath.Join("..", "..", "..", C1F3TerraChallengerRubricPath), C1F3TerraChallengerRubricFileSHA256, C1F3TerraChallengerRubricFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if rubric.Baseline.SemanticMappingCorrect != 45 || rubric.Baseline.ProxyRecallCorrect != 3 || rubric.MaterialImprovementGates.SemanticMappingCorrectMinimum != 47 || rubric.MaterialImprovementGates.ProxyRecallCorrectMinimum != 5 {
		t.Fatalf("Terra rubric thresholds changed: %+v", rubric)
	}
}

func TestC1F3TerraChallengerZeroNetworkPlan(t *testing.T) {
	profile := terraChallengerTestProfile(t)
	config := terraChallengerTestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(terraChallengerTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	binding := prepared.Plan.C1F3TerraChallengerFrozenBindings
	authorization := prepared.Plan.C1F3TerraChallengerExecutionAuthorization
	t.Logf("estimated_initial=%s maximum_retry_envelope=%s", prepared.Plan.HostedExperiment.EstimatedInitialRunUSD, prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD)
	if binding == nil || authorization == nil || binding.ProfileIdentity != C1F3TerraChallengerProfileIdentity ||
		binding.AcceptedLuna.RunID != C1F3TerraAcceptedLunaRunID || binding.AcceptedLuna.ArtifactIndexSHA256 != C1F3TerraAcceptedLunaArtifactIndexSHA256 ||
		binding.AcceptedLuna.RawResponseCount != 48 || !binding.AcceptedLuna.EvidenceUntouched || binding.CaseCount != 48 || binding.Repetitions != 1 || !binding.BoundaryExcluded {
		t.Fatalf("Terra frozen binding is incomplete: %+v", binding)
	}
	if prepared.Plan.ExecutionRoute != diagnosticRouteC1FTerraChallenger || prepared.Plan.ValidatorVersion != C1FValidatorVersion ||
		prepared.Plan.PromptVersion != V6PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion ||
		prepared.Plan.ModelConfiguration.Model != OpenAIDiagnosticTerraModel || prepared.Plan.ModelConfiguration.ReasoningEffort != "none" ||
		prepared.Plan.CasesPerRepetition != 48 || prepared.Plan.Repetitions != 1 {
		t.Fatalf("Terra route changed: %+v", prepared.Plan)
	}
	if authorization.OperatorOptIn || authorization.HostedInferenceAuthorized || authorization.CredentialPresent || authorization.ExecutionAuthorized ||
		!authorization.FrozenBindingsValid || !authorization.LunaPreservationValid || !authorization.DecisionRubricValid || !authorization.BudgetValid ||
		!authorization.EvidenceNamespaceCollisionFree || !authorization.ProviderInputIsolated || !authorization.OnlyModelVariableChanged ||
		!authorization.BoundaryExcluded || !authorization.RuntimeSafetyValid {
		t.Fatalf("Terra zero-network authorization state is unsafe: %+v", authorization)
	}
	if prepared.Plan.HostedExperiment.BudgetCeilingUSD != "0.30" || prepared.Plan.HostedExperiment.EstimatedInitialRunUSD == "" ||
		prepared.Plan.HostedExperiment.DatabaseMutationAllowed || prepared.Plan.HostedExperiment.TradingStateMutationAllowed {
		t.Fatalf("Terra budget or persistence boundary changed: %+v", prepared.Plan.HostedExperiment)
	}
}

func TestC1F3TerraChallengerAuthorizationMatrixAndIsolation(t *testing.T) {
	profile := terraChallengerTestProfile(t)
	for _, tt := range []struct {
		name       string
		hosted     bool
		optIn      bool
		credential bool
		want       string
	}{
		{name: "default deny", credential: true, want: "--authorize-c1f3-terra-challenger-t1"},
		{name: "hosted authorization alone", hosted: true, credential: true, want: "--authorize-c1f3-terra-challenger-t1"},
		{name: "Terra opt-in alone", optIn: true, credential: true, want: OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "missing credential", hosted: true, optIn: true, want: OpenAIDiagnosticAPIKeyEnv},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := terraChallengerTestConfig(t, profile, tt.hosted, tt.optIn, tt.credential)
			_, err := PrepareHostedDiagnostic(terraChallengerTestPaths(t, profile), config, diagnosticTestSafety())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Terra authorization state did not fail closed: %v", err)
			}
		})
	}
	prepared, config := terraChallengerPreparedAuthorized(t)
	if err := RevalidateOpenAIProviderConstruction(prepared, config); err != nil {
		t.Fatal(err)
	}
	if err := validateC1F3TerraProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures); err != nil {
		t.Fatal(err)
	}
}

func TestC1F3TerraChallengerRejectsWrongCellAndCrossAuthorization(t *testing.T) {
	profile := terraChallengerTestProfile(t)
	for name, mutate := range map[string]func(map[string]string){
		"wrong model":      func(values map[string]string) { values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel },
		"wrong reasoning":  func(values map[string]string) { values["JAX_AI_REASONING_EFFORT"] = "low" },
		"wrong case count": func(values map[string]string) { values["JAX_AI_MAX_EVENTS"] = "47" },
		"raised budget":    func(values map[string]string) { values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.31" },
	} {
		t.Run(name, func(t *testing.T) {
			values := c1e2bConfigValues(profile)
			values["JAX_AI_MODEL"] = OpenAIDiagnosticTerraModel
			mutate(values)
			if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
				t.Fatalf("Terra %s accepted", name)
			}
		})
	}
	if _, err := LoadDiagnosticRepetitionSelectionForProfile(func(string) (string, bool) { return "2", true }, profile); err == nil {
		t.Fatal("Terra second repetition accepted")
	}
	config := terraChallengerTestConfig(t, profile, true, false, true)
	config.C1F3RepeatabilityR3ExecutionAuthorization = NewC1F3RepeatabilityR3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(terraChallengerTestPaths(t, profile), config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("Luna r3 authorization reached Terra: %v", err)
	}
	luna := repeatabilityR3TestProfile(t)
	lunaConfig := repeatabilityR3TestConfig(t, luna, true, false, true)
	lunaConfig.C1F3TerraChallengerExecutionAuthorization = NewC1F3TerraChallengerExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(repeatabilityR3TestPaths(t, luna), lunaConfig, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("Terra authorization reached Luna r3: %v", err)
	}
	paths := terraChallengerTestPaths(t, profile)
	if err := os.MkdirAll(filepath.Join(paths.OutputRoot, "existing-run"), 0o750); err != nil {
		t.Fatal(err)
	}
	fullyAuthorized := terraChallengerTestConfig(t, profile, true, true, true)
	if _, err := PrepareHostedDiagnostic(paths, fullyAuthorized, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "already contains execution evidence") {
		t.Fatalf("colliding Terra namespace accepted: %v", err)
	}
}

func TestC1F3TerraChallengerDispositionRubric(t *testing.T) {
	rubric, err := LoadFrozenC1F3TerraChallengerRubric(filepath.Join("..", "..", "..", C1F3TerraChallengerRubricPath), C1F3TerraChallengerRubricFileSHA256, C1F3TerraChallengerRubricFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	base := C1F3RepeatabilityScore{AccuracyRetention: C1FDualScore{Semantic: C1FMetricLens{
		FinalValidity:   C1FMetric{Numerator: 48, Denominator: 48, Percentage: 100},
		WholeMapping:    C1FMetric{Numerator: 47, Denominator: 48, Percentage: 97.9167},
		DirectPrecision: C1FMetric{Numerator: 25, Denominator: 25, Percentage: 100},
		DirectRecall:    C1FMetric{Numerator: 25, Denominator: 25, Percentage: 100},
		ProxyRecall:     C1FMetric{Numerator: 5, Denominator: 6, Percentage: 83.3333},
		FalseDirect:     C1FMetric{Numerator: 0, Denominator: 48, Percentage: 0},
	}}}
	if got := C1F3TerraChallengerDisposition(rubric, base); got != C1F3TerraMateriallyBetter {
		t.Fatalf("material disposition=%s", got)
	}
	base.AccuracyRetention.Semantic.WholeMapping.Numerator = 46
	if got := C1F3TerraChallengerDisposition(rubric, base); got != C1F3TerraBetterButNotMaterial {
		t.Fatalf("better-not-material disposition=%s", got)
	}
	base.AccuracyRetention.Semantic.WholeMapping.Numerator = 45
	base.AccuracyRetention.Semantic.ProxyRecall.Numerator = 3
	if got := C1F3TerraChallengerDisposition(rubric, base); got != C1F3TerraEquivalent {
		t.Fatalf("equivalent disposition=%s", got)
	}
	base.AccuracyRetention.Semantic.DirectRecall.Percentage = 80
	if got := C1F3TerraChallengerDisposition(rubric, base); got != C1F3TerraWorse {
		t.Fatalf("worse disposition=%s", got)
	}
}
