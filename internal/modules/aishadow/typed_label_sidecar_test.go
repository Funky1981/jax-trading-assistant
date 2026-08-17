package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func c1e2aRepoRoot() string {
	return filepath.Join("..", "..", "..")
}

func c1e2aProfilePaths(profile DiagnosticEvaluationProfile) DiagnosticPaths {
	root := c1e2aRepoRoot()
	return DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        filepath.Join(root, filepath.FromSlash(profile.ManifestPath)),
		FingerprintLockPath: filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		FreezePath:          filepath.Join(root, filepath.FromSlash(profile.FreezePath)),
		AssetRulesetPath:    filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		TypedLabelPath:      filepath.Join(root, filepath.FromSlash(profile.TypedLabelPath)),
		ScoringRubricPath:   filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath)),
	}
}

func TestC1E2ATypedLabelSidecarsMatchFrozenSourcesAndV5Invariants(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, err := LoadDiagnosticEvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		paths := c1e2aProfilePaths(profile)
		raw, err := os.ReadFile(paths.TypedLabelPath)
		if err != nil {
			t.Fatal(err)
		}
		unfrozen, err := loadStrictTypedLabels(raw)
		if err != nil {
			t.Fatal(err)
		}
		computed, err := typedLabelSidecarFingerprint(unfrozen)
		if err != nil {
			t.Fatal(err)
		}
		if unfrozen.Fingerprint != computed || profile.TypedLabelFingerprint != computed {
			t.Fatalf("%s typed-label fingerprint: embedded=%s computed=%s profile=%s", identity, unfrozen.Fingerprint, computed, profile.TypedLabelFingerprint)
		}
		sidecar, err := LoadFrozenTypedLabelSidecarForProfile(profile, paths.TypedLabelPath)
		if err != nil {
			t.Fatal(err)
		}
		rules, err := assetresolution.LoadRuleset(paths.AssetRulesetPath)
		if err != nil {
			t.Fatal(err)
		}
		resolver := assetresolution.Resolver{Rules: rules}
		exposures, err := resolver.ProxyExposures()
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := LoadFrozenDiagnosticManifestForProfile(profile, paths.ManifestPath, exposures)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateTypedLabelSidecarAgainstSource(profile, sidecar, manifest, resolver); err != nil {
			t.Fatal(err)
		}
	}
}

func TestC1E2AScoringRubricFreezesEveryDenominator(t *testing.T) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	path := filepath.Join(c1e2aRepoRoot(), filepath.FromSlash(profile.ScoringRubricPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rubric CausalAttributionScoringRubric
	if err := json.Unmarshal(raw, &rubric); err != nil {
		t.Fatal(err)
	}
	computed, err := scoringRubricFingerprint(rubric)
	if err != nil {
		t.Fatal(err)
	}
	if rubric.Fingerprint != computed || profile.ScoringRubricFingerprint != computed {
		t.Fatalf("scoring fingerprint: embedded=%s computed=%s profile=%s", rubric.Fingerprint, computed, profile.ScoringRubricFingerprint)
	}
	if _, err := LoadFrozenC1E3ScoringRubric(profile, path); err != nil {
		t.Fatal(err)
	}
}

func TestC1E2AExpectedAnswersCannotEnterProviderRequest(t *testing.T) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	paths := c1e2aProfilePaths(profile)
	sidecar, err := LoadFrozenTypedLabelSidecarForProfile(profile, paths.TypedLabelPath)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := assetresolution.LoadRuleset(paths.AssetRulesetPath)
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, _ := resolver.ProxyExposures()
	manifest, err := LoadFrozenDiagnosticManifestForProfile(profile, paths.ManifestPath, exposures)
	if err != nil {
		t.Fatal(err)
	}
	request, err := V5InitialRequest(manifest.Events[0].Input, exposures)
	if err != nil {
		t.Fatal(err)
	}
	config := OpenAIDiagnosticConfig{
		Runtime: Config{Model: OpenAIDiagnosticLunaModel}, ExperimentID: profile.RequiredExperimentID,
		ReasoningEffort: OpenAIDiagnosticReasoningEffort, MaxOutputTokens: 256,
		OutputContractMode: OpenAIOutputContractStrictJSONSchema,
		PromptVersion:      V5PromptVersion, OutputContract: V5SchemaVersion, CausalPolicy: CausalAttributionPolicyVersion,
	}
	wire, err := marshalOpenAIDiagnosticRequest(config, request)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"expected_mapping_status", "expected_direct_issuer", "expected_proxy_exposure", "expected_issuer_attributions",
		"expected_principal_proxy_candidates", "typed_attribution_rationale", "adjudication_status", "scoring_rubric",
		"numerator", "denominator", sidecar.Cases[0].TypedAttributionRationale,
	} {
		if strings.Contains(string(wire), forbidden) {
			t.Fatalf("provider-visible request leaked frozen expected-answer material %q", forbidden)
		}
	}
	if !strings.Contains(request.User, manifest.Events[0].Input.Title) || len(request.User) == 0 {
		t.Fatal("provider request did not retain the permitted frozen EventInput")
	}
}

func TestC1E3ProfilesBindExactLabelsForAuthorizedPlanning(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, _ := LoadDiagnosticEvaluationProfile(identity)
		paths := c1e2aProfilePaths(profile)
		prepared, err := PrepareDiagnostic(paths, Config{Provider: OpenAIDiagnosticProvider, MaxEvents: profile.CaseCount}, diagnosticTestSafety())
		if err != nil {
			t.Fatal(err)
		}
		if prepared.Plan.PromptVersion != V5PromptVersion || prepared.Plan.OutputContract != V5SchemaVersion ||
			prepared.Plan.CausalAttributionPolicy != CausalAttributionPolicyVersion || prepared.Plan.TypedLabelVersion != profile.TypedLabelVersion ||
			prepared.Plan.ScoringRubricVersion != profile.ScoringRubricVersion {
			t.Fatalf("C1E3 planning lost frozen v5/sidecar/scoring bindings: %+v", prepared.Plan)
		}
		missing := paths
		missing.TypedLabelPath = filepath.Join(t.TempDir(), "missing.json")
		if _, err := PrepareDiagnostic(missing, Config{Provider: OpenAIDiagnosticProvider, MaxEvents: profile.CaseCount}, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "typed-attribution label sidecar") {
			t.Fatalf("missing typed labels did not fail closed: %v", err)
		}
	}
	generalization, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	boundary, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileBoundaryV2)
	swapped := c1e2aProfilePaths(generalization)
	swapped.TypedLabelPath = filepath.Join(c1e2aRepoRoot(), filepath.FromSlash(boundary.TypedLabelPath))
	if _, err := PrepareDiagnostic(swapped, Config{Provider: OpenAIDiagnosticProvider, MaxEvents: generalization.CaseCount}, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "file hash changed") {
		t.Fatalf("swapped typed-label sidecar did not fail closed: %v", err)
	}
	rawRubric, err := os.ReadFile(swapped.ScoringRubricPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedRubric := filepath.Join(t.TempDir(), "tampered-scoring.json")
	if err := os.WriteFile(tamperedRubric, append(rawRubric, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	exact := c1e2aProfilePaths(generalization)
	exact.ScoringRubricPath = tamperedRubric
	if _, err := PrepareDiagnostic(exact, Config{Provider: OpenAIDiagnosticProvider, MaxEvents: generalization.CaseCount}, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "scoring rubric file hash changed") {
		t.Fatalf("tampered scoring rubric did not fail closed: %v", err)
	}
}

func TestC1E2AImmutableInputsAndImplementationRemainFrozen(t *testing.T) {
	root := c1e2aRepoRoot()
	expected := map[string]string{
		"config/ai-shadow-issuer-generalization-holdout-v2.json":                    "7b22c4c6d72d53d9976df17463bd6116a50ac305008c6d71c5a36f6971091c04",
		"config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v2.json": "c3dc9715e5c7bcc1f8e0cb1020d95e7979a79f4c5748abf0124df5fa76e1cf88",
		"config/ai-shadow-issuer-generalization-holdout-freeze-v2.json":             "e32eb3ef76a234b5b53db2cea9c011a4e1d85571c6d4b865e1498cd48761d878",
		"config/ai-shadow-issuer-boundary-challenge-v2.json":                        "ae2e15a18e28094c44663bd94bc8f40145e3fd1358ae46e525fca85166ce7578",
		"config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v2.json":     "3cced77fbc0d2d229143f379d22365981a668dce6e93174027b0dcfe7a137112",
		"config/ai-shadow-issuer-boundary-challenge-freeze-v2.json":                 "0123286e7e0862961368e85ebea81d3474b49008bd70f85b5c21f7ad75f80dc2",
		"config/event-asset-resolution-v1.json":                                     expectedAssetRulesetFileSHA256,
		"internal/modules/aishadow/causal_attribution.go":                           frozenC1EPolicySHA256,
	}
	for relative, want := range expected {
		got, err := diagnosticFileSHA256(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || got != want {
			t.Fatalf("immutable file %s changed: got %s want %s err=%v", relative, got, want, err)
		}
	}
	if got := rawHash(v5SystemPrompt); got != frozenV5PromptSHA256 {
		t.Fatalf("frozen v5 prompt changed: got %s want %s", got, frozenV5PromptSHA256)
	}
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	exposures, _ := (assetresolution.Resolver{Rules: rules}).ProxyExposures()
	gotSchema, err := fingerprint(V5OutputSchema(exposures))
	if err != nil || gotSchema != frozenV5SchemaSHA256 {
		t.Fatalf("frozen v5 schema changed: got %s want %s err=%v", gotSchema, frozenV5SchemaSHA256, err)
	}
}
