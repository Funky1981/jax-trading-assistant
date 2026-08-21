package aishadow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repeatabilityR3TestProfile(t *testing.T) DiagnosticEvaluationProfile {
	t.Helper()
	profile, err := LoadDiagnosticExecutionProfile(C1F3RepeatabilityR3ProfileIdentity)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func repeatabilityR3TestConfig(t *testing.T, profile DiagnosticEvaluationProfile, hosted, optIn, credential bool) OpenAIDiagnosticConfig {
	t.Helper()
	values := c1e2bConfigValues(profile)
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
	config.C1F3RepeatabilityR3ExecutionAuthorization = NewC1F3RepeatabilityR3ExecutionAuthorization(optIn)
	return config
}

func repeatabilityR3TestPaths(t *testing.T, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	return c1e2bProfilePaths(root, t.TempDir(), profile)
}

func repeatabilityR3PreparedAuthorized(t *testing.T) (PreparedDiagnostic, OpenAIDiagnosticConfig) {
	t.Helper()
	profile := repeatabilityR3TestProfile(t)
	config := repeatabilityR3TestConfig(t, profile, true, true, true)
	prepared, err := PrepareHostedDiagnostic(repeatabilityR3TestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	return prepared, config
}

func TestC1F3RepeatabilityR3FrozenIdentityMaterial(t *testing.T) {
	profile, err := FrozenC1F3RepeatabilityR3Profile()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := profile.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	authorizationFingerprint := C1F3RepeatabilityR3ExecutionAuthorizationFingerprint()
	t.Logf("r3_profile_fingerprint=%s r3_authorization_fingerprint=%s", fingerprint, authorizationFingerprint)
	if fingerprint != C1F3RepeatabilityR3ProfileFingerprint {
		t.Fatalf("r3 profile fingerprint changed: %s", fingerprint)
	}
	if authorizationFingerprint != C1F3RepeatabilityR3AuthorizationFingerprint {
		t.Fatalf("r3 authorization fingerprint changed: %s", authorizationFingerprint)
	}
}

func TestC1F3RepeatabilityR3ZeroNetworkPlanBindsC1FRouteAndAcceptedBaseline(t *testing.T) {
	profile := repeatabilityR3TestProfile(t)
	config := repeatabilityR3TestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(repeatabilityR3TestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	binding := prepared.Plan.C1F3RepeatabilityFrozenBindings
	authorization := prepared.Plan.C1F3RepeatabilityR3ExecutionAuthorization
	if binding == nil || authorization == nil || binding.ProfileIdentity != C1F3RepeatabilityR3ProfileIdentity ||
		binding.Baseline != frozenC1F3RepeatabilityBaseline() || binding.ComparisonScoring.Identity != C1F3RepeatabilityScoringVersion ||
		binding.ComparisonScoring.FileSHA256 != C1F3RepeatabilityScoringFileSHA256 || binding.CaseCount != 48 || binding.Repetitions != 1 {
		t.Fatalf("r3 frozen binding is incomplete: %+v", binding)
	}
	if prepared.Plan.ExecutionRoute != diagnosticRouteC1FRepeatabilityR3 || prepared.Plan.ValidatorVersion != C1FValidatorVersion ||
		prepared.Plan.PromptVersion != V6PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion {
		t.Fatalf("r3 did not bind the explicit C1F route: %+v", prepared.Plan)
	}
	if authorization.OperatorOptIn || authorization.HostedInferenceAuthorized || authorization.CredentialPresent || authorization.ExecutionAuthorized ||
		!authorization.FrozenBindingsValid || !authorization.BaselineBindingValid || !authorization.RepeatabilityScoringValid || !authorization.BudgetValid ||
		!authorization.EvidenceNamespaceCollisionFree || !authorization.ProviderInputIsolated || !authorization.ProviderInputMatchesC1F3 ||
		!authorization.R2ResponseIsolated || !authorization.RuntimeSafetyValid {
		t.Fatalf("r3 zero-network authorization state is unsafe: %+v", authorization)
	}
	if prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD != "0.186754" || prepared.Plan.HostedExperiment.BudgetCeilingUSD != "0.30" {
		t.Fatalf("r3 cost freeze changed: %+v", prepared.Plan.HostedExperiment)
	}
}

func TestC1F3RepeatabilityR3AuthorizationMatrix(t *testing.T) {
	profile := repeatabilityR3TestProfile(t)
	for _, tt := range []struct {
		name       string
		hosted     bool
		optIn      bool
		credential bool
		want       string
	}{
		{name: "default deny", credential: true, want: "--authorize-c1f3-repeatability-r3"},
		{name: "hosted authorization alone", hosted: true, credential: true, want: "--authorize-c1f3-repeatability-r3"},
		{name: "r3 opt-in alone", optIn: true, credential: true, want: OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "missing credential", hosted: true, optIn: true, want: OpenAIDiagnosticAPIKeyEnv},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := repeatabilityR3TestConfig(t, profile, tt.hosted, tt.optIn, tt.credential)
			_, err := PrepareHostedDiagnostic(repeatabilityR3TestPaths(t, profile), config, diagnosticTestSafety())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("r3 authorization state did not fail closed: %v", err)
			}
		})
	}
}

func TestC1F3RepeatabilityR3ExactCombinationReachesProviderConstructionGate(t *testing.T) {
	prepared, config := repeatabilityR3PreparedAuthorized(t)
	if err := RevalidateOpenAIProviderConstruction(prepared, config); err != nil {
		t.Fatal(err)
	}
	if !prepared.Plan.C1F3RepeatabilityR3ExecutionAuthorization.ExecutionAuthorized {
		t.Fatal("exact r3 combination did not reach provider construction gate")
	}
}

func TestC1F3RepeatabilityR3RejectsMutatedBindingsAndNamespaceCollision(t *testing.T) {
	profile := repeatabilityR3TestProfile(t)
	t.Run("wrong model", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_MODEL"] = OpenAIDiagnosticSolModel
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
			t.Fatal("wrong r3 model accepted")
		}
	})
	t.Run("wrong reasoning", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_REASONING_EFFORT"] = "low"
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
			t.Fatal("wrong r3 reasoning effort accepted")
		}
	})
	t.Run("wrong case count", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_MAX_EVENTS"] = "47"
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil || !strings.Contains(err.Error(), "JAX_AI_MAX_EVENTS=48") {
			t.Fatalf("wrong r3 case count accepted: %v", err)
		}
	})
	t.Run("wrong repetition", func(t *testing.T) {
		if _, err := LoadDiagnosticRepetitionSelectionForProfile(func(string) (string, bool) { return "2", true }, profile); err == nil {
			t.Fatal("wrong r3 repetition accepted")
		}
	})
	t.Run("raised budget", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.31"
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
			t.Fatal("raised r3 budget ceiling accepted")
		}
	})
	t.Run("colliding namespace", func(t *testing.T) {
		config := repeatabilityR3TestConfig(t, profile, true, true, true)
		paths := repeatabilityR3TestPaths(t, profile)
		if err := os.MkdirAll(filepath.Join(paths.OutputRoot, "existing-run"), 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "already contains execution evidence") {
			t.Fatalf("colliding r3 namespace accepted: %v", err)
		}
	})
	prepared, config := repeatabilityR3PreparedAuthorized(t)
	mutations := map[string]func(*C1F3RepeatabilityFrozenBindingPlan){
		"wrong baseline":       func(binding *C1F3RepeatabilityFrozenBindingPlan) { binding.Baseline.RunID = "wrong" },
		"wrong baseline index": func(binding *C1F3RepeatabilityFrozenBindingPlan) { binding.Baseline.ArtifactIndexSHA256 = "wrong" },
		"wrong semantic hash":  func(binding *C1F3RepeatabilityFrozenBindingPlan) { binding.SemanticStack.PromptSHA256 = "wrong" },
		"wrong scoring":        func(binding *C1F3RepeatabilityFrozenBindingPlan) { binding.ComparisonScoring.FileSHA256 = "wrong" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := prepared
			binding := *prepared.Plan.C1F3RepeatabilityFrozenBindings
			mutate(&binding)
			changed.Plan.C1F3RepeatabilityFrozenBindings = &binding
			if err := validateC1F3RepeatabilityR3ExecutionAuthorization(changed, config); err == nil {
				t.Fatal("mutated r3 binding accepted")
			}
		})
	}
}

func TestC1F3RepeatabilityR3AuthorizationsRemainFamilyIsolated(t *testing.T) {
	r3 := repeatabilityR3TestProfile(t)
	for name, mutate := range map[string]func(*OpenAIDiagnosticConfig){
		"original C1F3": func(config *OpenAIDiagnosticConfig) {
			config.C1F3ExecutionAuthorization = NewC1F3ExecutionAuthorization(true)
		},
		"consumed r2": func(config *OpenAIDiagnosticConfig) {
			config.C1F3RepeatabilityExecutionAuthorization = NewC1F3RepeatabilityExecutionAuthorization(true)
		},
		"historical C1E": func(config *OpenAIDiagnosticConfig) {
			config.C1E3ExecutionAuthorization = NewC1E3ExecutionAuthorization(true)
		},
	} {
		t.Run(name+" cannot authorize r3", func(t *testing.T) {
			config := repeatabilityR3TestConfig(t, r3, true, false, true)
			mutate(&config)
			if _, err := PrepareHostedDiagnostic(repeatabilityR3TestPaths(t, r3), config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
				t.Fatalf("%s authorization reached r3: %v", name, err)
			}
		})
	}
	original, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	originalConfig := c1f3AuthorizationTestConfig(t, original, true, false, true)
	originalConfig.C1F3RepeatabilityR3ExecutionAuthorization = NewC1F3RepeatabilityR3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, original), originalConfig, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("r3 authorization reached original C1F3: %v", err)
	}
	r2 := repeatabilityTestProfile(t)
	r2Config := repeatabilityTestConfig(t, r2, true, false, true)
	r2Config.C1F3RepeatabilityR3ExecutionAuthorization = NewC1F3RepeatabilityR3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(repeatabilityTestPaths(t, r2), r2Config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("r3 authorization reached consumed r2: %v", err)
	}
}

func TestC1F3RepeatabilityR3ProviderInputsMatchOriginalAndExcludeR2Evidence(t *testing.T) {
	profile := repeatabilityR3TestProfile(t)
	config := repeatabilityR3TestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(repeatabilityR3TestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateC1F3RepeatabilityProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures); err != nil {
		t.Fatal(err)
	}
}
