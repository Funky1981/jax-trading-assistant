package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repeatabilityTestProfile(t *testing.T) DiagnosticEvaluationProfile {
	t.Helper()
	profile, err := LoadDiagnosticExecutionProfile(C1F3RepeatabilityProfileIdentity)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func repeatabilityTestConfig(t *testing.T, profile DiagnosticEvaluationProfile, hosted, optIn, credential bool) OpenAIDiagnosticConfig {
	t.Helper()
	values := c1e2bConfigValues(profile)
	values[OpenAIDiagnosticInferenceAuthEnv] = "false"
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
	config.C1F3RepeatabilityExecutionAuthorization = NewC1F3RepeatabilityExecutionAuthorization(optIn)
	return config
}

func repeatabilityTestPaths(t *testing.T, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	return DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        filepath.Join(root, filepath.FromSlash(profile.ManifestPath)), FingerprintLockPath: filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		FreezePath: filepath.Join(root, filepath.FromSlash(profile.FreezePath)), AssetRulesetPath: filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		TypedLabelPath: filepath.Join(root, filepath.FromSlash(profile.TypedLabelPath)), ScoringRubricPath: filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath)),
		OutputRoot: filepath.Join(t.TempDir(), profile.EvidenceNamespace, profile.RequiredExperimentID),
	}
}

func repeatabilityPreparedAuthorized(t *testing.T) (PreparedDiagnostic, OpenAIDiagnosticConfig) {
	t.Helper()
	profile := repeatabilityTestProfile(t)
	config := repeatabilityTestConfig(t, profile, true, true, true)
	prepared, err := PrepareHostedDiagnostic(repeatabilityTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	return prepared, config
}

func TestC1F3RepeatabilityFrozenIdentityMaterial(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(C1F3RepeatabilityScoringPath)))
	if err != nil {
		t.Fatal(err)
	}
	var freeze C1F3RepeatabilityScoringFreeze
	if err := json.Unmarshal(raw, &freeze); err != nil {
		t.Fatal(err)
	}
	scoringFingerprint, err := c1f3RepeatabilityScoringFingerprint(freeze)
	if err != nil {
		t.Fatal(err)
	}
	fileSHA, err := hashOpaqueFile(filepath.Join(root, filepath.FromSlash(C1F3RepeatabilityScoringPath)))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := FrozenC1F3RepeatabilityProfile()
	if err != nil {
		t.Fatal(err)
	}
	profileFingerprint, err := profile.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("scoring_fingerprint=%s scoring_file_sha=%s profile_fingerprint=%s authorization_fingerprint=%s", scoringFingerprint, fileSHA, profileFingerprint, C1F3RepeatabilityExecutionAuthorizationFingerprint())
	if C1F3RepeatabilityScoringFingerprint != "PENDING" && scoringFingerprint != C1F3RepeatabilityScoringFingerprint {
		t.Fatalf("repeatability scoring fingerprint changed: %s", scoringFingerprint)
	}
	if C1F3RepeatabilityScoringFileSHA256 != "PENDING" && fileSHA != C1F3RepeatabilityScoringFileSHA256 {
		t.Fatalf("repeatability scoring file hash changed: %s", fileSHA)
	}
	if C1F3RepeatabilityProfileFingerprint != "PENDING" && profileFingerprint != C1F3RepeatabilityProfileFingerprint {
		t.Fatalf("repeatability profile fingerprint changed: %s", profileFingerprint)
	}
	if C1F3RepeatabilityExecutionAuthorizationFingerprint() != C1F3RepeatabilityAuthorizationFingerprint {
		t.Fatalf("repeatability authorization fingerprint changed: %s", C1F3RepeatabilityExecutionAuthorizationFingerprint())
	}
}

func TestC1F3RepeatabilityProfileBindsAcceptedEvidenceAndSemanticStack(t *testing.T) {
	profile := repeatabilityTestProfile(t)
	config := repeatabilityTestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(repeatabilityTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	binding := prepared.Plan.C1F3RepeatabilityFrozenBindings
	authorization := prepared.Plan.C1F3RepeatabilityExecutionAuthorization
	if binding == nil || authorization == nil || binding.ProfileIdentity != C1F3RepeatabilityProfileIdentity ||
		binding.Baseline != frozenC1F3RepeatabilityBaseline() || binding.SemanticStack.ProfileIdentity != C1F3ProfileGeneralization ||
		binding.CaseCount != 48 || binding.Repetitions != 1 || binding.ComparisonScoring.Identity != C1F3RepeatabilityScoringVersion {
		t.Fatalf("repeatability binding is incomplete: %+v", binding)
	}
	if authorization.OperatorOptIn || authorization.HostedInferenceAuthorized || authorization.CredentialPresent || authorization.ExecutionAuthorized ||
		!authorization.FrozenBindingsValid || !authorization.BaselineBindingValid || !authorization.RepeatabilityScoringValid || !authorization.BudgetValid ||
		!authorization.EvidenceNamespaceCollisionFree || !authorization.ProviderInputIsolated || !authorization.ProviderInputMatchesC1F3 || !authorization.RuntimeSafetyValid {
		t.Fatalf("repeatability zero-network preflight state is unsafe: %+v", authorization)
	}
	if prepared.Plan.PromptVersion != V6PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion || prepared.Plan.CausalAttributionPolicy != CausalAttributionPolicyVersion ||
		prepared.Plan.ScoringVersion != C1FScoringVersion || prepared.Plan.ModelConfiguration.Model != OpenAIDiagnosticLunaModel || prepared.Plan.ModelConfiguration.ReasoningEffort != "none" {
		t.Fatalf("repeatability semantic/model stack changed: %+v", prepared.Plan)
	}
	if prepared.Plan.HostedExperiment.BudgetCeilingUSD != "0.30" || prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD == "" {
		t.Fatalf("repeatability budget evidence missing: %+v", prepared.Plan.HostedExperiment)
	}
}

func TestC1F3RepeatabilityAuthorizationMatrix(t *testing.T) {
	profile := repeatabilityTestProfile(t)
	for _, tt := range []struct {
		name       string
		hosted     bool
		optIn      bool
		credential bool
		want       string
	}{
		{name: "default deny", credential: true, want: "--authorize-c1f3-repeatability"},
		{name: "hosted authorization alone", hosted: true, credential: true, want: "--authorize-c1f3-repeatability"},
		{name: "repeatability opt-in alone", optIn: true, credential: true, want: OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "missing credential", hosted: true, optIn: true, want: OpenAIDiagnosticAPIKeyEnv},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := repeatabilityTestConfig(t, profile, tt.hosted, tt.optIn, tt.credential)
			_, err := PrepareHostedDiagnostic(repeatabilityTestPaths(t, profile), config, diagnosticTestSafety())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("repeatability authorization state did not fail closed: %v", err)
			}
		})
	}
}

func TestC1F3RepeatabilityExactCombinationReachesProviderConstructionGate(t *testing.T) {
	prepared, config := repeatabilityPreparedAuthorized(t)
	if err := RevalidateOpenAIProviderConstruction(prepared, config); err != nil {
		t.Fatal(err)
	}
	if !prepared.Plan.C1F3RepeatabilityExecutionAuthorization.ExecutionAuthorized {
		t.Fatal("exact repeatability combination did not reach provider construction gate")
	}
}

func TestC1F3RepeatabilityRejectsWrongProfileModelShapeBindingsAndNamespace(t *testing.T) {
	profile := repeatabilityTestProfile(t)
	t.Run("wrong profile", func(t *testing.T) {
		original, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
		config := c1f3AuthorizationTestConfig(t, original, true, false, true)
		config.C1F3RepeatabilityExecutionAuthorization = NewC1F3RepeatabilityExecutionAuthorization(true)
		if _, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, original), config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoped only") {
			t.Fatalf("repeatability authorization accepted original C1F3: %v", err)
		}
	})
	t.Run("wrong model", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_MODEL"] = OpenAIDiagnosticSolModel
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil {
			t.Fatal("wrong repeatability model accepted")
		}
	})
	t.Run("wrong case count", func(t *testing.T) {
		values := c1e2bConfigValues(profile)
		values["JAX_AI_MAX_EVENTS"] = "47"
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false); err == nil || !strings.Contains(err.Error(), "JAX_AI_MAX_EVENTS=48") {
			t.Fatalf("wrong repeatability case count accepted: %v", err)
		}
	})
	t.Run("wrong repetition", func(t *testing.T) {
		if _, err := LoadDiagnosticRepetitionSelectionForProfile(func(string) (string, bool) { return "2", true }, profile); err == nil {
			t.Fatal("wrong repeatability repetition accepted")
		}
	})
	t.Run("colliding namespace", func(t *testing.T) {
		config := repeatabilityTestConfig(t, profile, true, true, true)
		paths := repeatabilityTestPaths(t, profile)
		if err := os.MkdirAll(filepath.Join(paths.OutputRoot, "existing-run"), 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "already contains execution evidence") {
			t.Fatalf("colliding repeatability namespace accepted: %v", err)
		}
	})
	prepared, config := repeatabilityPreparedAuthorized(t)
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
			if err := validateC1F3RepeatabilityExecutionAuthorization(changed, config); err == nil {
				t.Fatal("mutated repeatability binding accepted")
			}
		})
	}
}

func TestC1F3AndRepeatabilityAuthorizationsCannotCrossAuthorize(t *testing.T) {
	repeatProfile := repeatabilityTestProfile(t)
	repeatConfig := repeatabilityTestConfig(t, repeatProfile, true, false, true)
	repeatConfig.C1F3ExecutionAuthorization = NewC1F3ExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(repeatabilityTestPaths(t, repeatProfile), repeatConfig, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "C1F3 execution authorization is scoped only") {
		t.Fatalf("original C1F3 authorization cross-authorized repeatability: %v", err)
	}
	original, _ := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	originalConfig := c1f3AuthorizationTestConfig(t, original, true, false, true)
	originalConfig.C1F3RepeatabilityExecutionAuthorization = NewC1F3RepeatabilityExecutionAuthorization(true)
	if _, err := PrepareHostedDiagnostic(c1f3AuthorizationTestPaths(t, original), originalConfig, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "repeatability execution authorization is scoped only") {
		t.Fatalf("repeatability authorization cross-authorized original C1F3: %v", err)
	}
}

func TestC1F3RepeatabilityProviderAnswerIsolationMatchesOriginalWireRequests(t *testing.T) {
	profile := repeatabilityTestProfile(t)
	config := repeatabilityTestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(repeatabilityTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateC1F3RepeatabilityProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures); err != nil {
		t.Fatal(err)
	}
}
