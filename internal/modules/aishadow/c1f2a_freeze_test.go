package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func TestC1F2AFrozenArtifactFingerprints(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, path := range []string{
		"config/ai-shadow-causal-attribution-labels-generalization-v3-v1.json",
		"config/ai-shadow-causal-attribution-labels-boundary-v3-v1.json",
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		sidecar, err := loadStrictC1F3TypedLabels(raw)
		if err != nil {
			t.Fatal(err)
		}
		computed, err := c1f3TypedLabelFingerprint(sidecar)
		if err != nil {
			t.Fatal(err)
		}
		if sidecar.Fingerprint != computed {
			t.Fatalf("%s fingerprint got %q want %s", path, sidecar.Fingerprint, computed)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "config", "ai-shadow-causal-attribution-scoring-c1f3-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var scoring C1F3ScoringFreeze
	if err = json.Unmarshal(raw, &scoring); err != nil {
		t.Fatal(err)
	}
	computed, err := c1f3ScoringFreezeFingerprint(scoring)
	if err != nil {
		t.Fatal(err)
	}
	if scoring.Fingerprint != computed {
		t.Fatalf("scoring fingerprint got %q want %s", scoring.Fingerprint, computed)
	}
}

func TestC1F2ATypedSidecarsProjectFrozenV3(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		if _, err := LoadDiagnosticEvaluationProfile(identity); err == nil {
			t.Fatalf("C1F3 profile %s must not be registered with the hosted executor", identity)
		}
		profile, err := LoadC1F3EvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		sidecar, err := LoadFrozenC1F3TypedLabelSidecar(profile, filepath.Join(root, filepath.FromSlash(profile.TypedSidecarPath)))
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := LoadFrozenC1F3Manifest(profile, filepath.Join(root, filepath.FromSlash(profile.Dataset.ManifestPath)), exposures)
		if err != nil {
			t.Fatal(err)
		}
		if err = ValidateC1F3TypedLabelSidecar(profile, sidecar, manifest, resolver); err != nil {
			t.Fatal(err)
		}
		if err = ValidateFrozenC1F3InputLock(profile, filepath.Join(root, filepath.FromSlash(profile.Dataset.InputLockPath)), manifest); err != nil {
			t.Fatal(err)
		}
		for index, expected := range sidecar.Cases {
			result := V5StructuredResult{MarketRelevance: "MEDIUM", MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure, MappingConfidence: "HIGH", ExpectedHorizon: "UNCLEAR", LikelyDirection: "UNCLEAR", CatalystType: "offline label compatibility", Reason: "Offline compatibility validation for the frozen expected typed attribution.", MissingEvidence: []string{}, IssuerAttributions: expected.ExpectedIssuerAttributions, PrincipalProxyCandidates: expected.ExpectedPrincipalProxyCandidates}
			raw, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, _, _, validationErrors := ParseValidateAndApplyC1F(string(raw), manifest.Events[index].Input, resolver); len(validationErrors) > 0 {
				t.Fatalf("%s C1F compatibility: %v", expected.CaseID, validationErrors)
			}
		}
	}
}

func TestC1F2ARoleAndProxyCardinalities(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	tests := []struct {
		path                       string
		roles                      map[CausalRole]int
		empty, singleton, multiple int
		clean, reconsidered        int
	}{
		{"config/ai-shadow-causal-attribution-labels-generalization-v3-v1.json", map[CausalRole]int{CausalRolePrincipal: 25, CausalRoleEqualPrincipal: 6, CausalRoleSecondaryAffected: 0, CausalRoleContextOnly: 13, CausalRolePossiblePrincipal: 2}, 41, 6, 1, 38, 10},
		{"config/ai-shadow-causal-attribution-labels-boundary-v3-v1.json", map[CausalRole]int{CausalRolePrincipal: 11, CausalRoleEqualPrincipal: 8, CausalRoleSecondaryAffected: 3, CausalRoleContextOnly: 12, CausalRolePossiblePrincipal: 4}, 25, 5, 2, 20, 12},
	}
	for _, test := range tests {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(test.path)))
		if err != nil {
			t.Fatal(err)
		}
		sidecar, err := loadStrictC1F3TypedLabels(raw)
		if err != nil {
			t.Fatal(err)
		}
		roles := map[CausalRole]int{}
		empty, singleton, multiple := 0, 0, 0
		for _, expected := range sidecar.Cases {
			for _, attribution := range expected.ExpectedIssuerAttributions {
				roles[attribution.CausalRole]++
			}
			switch len(expected.ExpectedPrincipalProxyCandidates) {
			case 0:
				empty++
			case 1:
				singleton++
			default:
				multiple++
			}
		}
		for role, want := range test.roles {
			if roles[role] != want {
				t.Fatalf("%s role %s got %d want %d", test.path, role, roles[role], want)
			}
		}
		if empty != test.empty || singleton != test.singleton || multiple != test.multiple {
			t.Fatalf("%s proxy cardinalities got %d/%d/%d want %d/%d/%d", test.path, empty, singleton, multiple, test.empty, test.singleton, test.multiple)
		}
		if sidecar.QualityControl.FirstPassClean != test.clean || sidecar.QualityControl.RequiredReconsideration != test.reconsidered || sidecar.QualityControl.RemainingAmbiguous != 0 || sidecar.QualityControl.ContractConflicts != 0 {
			t.Fatalf("%s quality control changed: %+v", test.path, sidecar.QualityControl)
		}
	}
}

func TestC1F2ASemanticIdentityCompatibility(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	identity := NewIssuerSemanticIdentity(rules)
	unavailable := map[string]bool{}
	for _, path := range []string{"config/ai-shadow-causal-attribution-labels-generalization-v3-v1.json", "config/ai-shadow-causal-attribution-labels-boundary-v3-v1.json"} {
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sidecar, decodeErr := loadStrictC1F3TypedLabels(raw)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		for _, expected := range sidecar.Cases {
			for _, attribution := range expected.ExpectedIssuerAttributions {
				key := assetresolution.CanonicalizeIssuerName(attribution.Issuer)
				candidates := identity.candidates(key)
				if len(candidates) > 1 {
					t.Fatalf("unexpected semantic identity collision for %s/%s: %v", expected.CaseID, attribution.Issuer, candidates)
				}
				if len(candidates) == 0 {
					unavailable[attribution.Issuer] = true
				}
				if comparison := identity.Compare(attribution.Issuer, attribution.Issuer); comparison.Outcome != IssuerIdentityExact {
					t.Fatalf("truth identity is not self-exact for %s: %+v", attribution.Issuer, comparison)
				}
			}
		}
	}
	if len(unavailable) == 0 {
		t.Fatal("expected explicit record of identities outside the bounded comparator catalog")
	}
	t.Logf("deterministic equivalence metadata unavailable for %d distinct expected surfaces: %v", len(unavailable), unavailable)
}

func TestC1F2ADirectIssuerSurfaceCompatibilityAccounting(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	identity := NewIssuerSemanticIdentity(rules)
	counts := map[IssuerIdentityOutcome]int{}
	for _, profileIdentity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, loadErr := LoadC1F3EvaluationProfile(profileIdentity)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		manifest, loadErr := LoadFrozenC1F3Manifest(profile, filepath.Join(root, filepath.FromSlash(profile.Dataset.ManifestPath)), exposures)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		for _, event := range manifest.Events {
			if event.Label.MappingStatus != "DIRECT" {
				continue
			}
			best := IssuerIdentityDistinct
			for _, entity := range event.Input.Entities {
				outcome := identity.Compare(event.Label.DirectIssuer, entity).Outcome
				if outcome == IssuerIdentityExact || outcome == IssuerIdentityEquivalent && best != IssuerIdentityExact || outcome == IssuerIdentityAmbiguous && best == IssuerIdentityDistinct {
					best = outcome
				}
			}
			counts[best]++
		}
	}
	if counts[IssuerIdentityExact] != 8 || counts[IssuerIdentityEquivalent] != 27 || counts[IssuerIdentityDistinct] != 1 || counts[IssuerIdentityAmbiguous] != 0 {
		t.Fatalf("direct expected-to-event surface accounting changed: %+v", counts)
	}
}

func TestC1F2AScoringFreezeAndIdentityOutcomes(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, err := LoadC1F3EvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = LoadFrozenC1F3ScoringFreeze(profile, filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath))); err != nil {
			t.Fatal(err)
		}
	}
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	identity := NewIssuerSemanticIdentity(rules)
	tests := []struct {
		left, right string
		want        IssuerIdentityOutcome
	}{
		{"Apple Inc.", "Apple Inc.", IssuerIdentityExact},
		{"Apple Inc.", "Apple", IssuerIdentityEquivalent},
		{"Apple Inc.", "Microsoft Corporation", IssuerIdentityDistinct},
	}
	for _, test := range tests {
		if got := identity.Compare(test.left, test.right).Outcome; got != test.want {
			t.Fatalf("identity outcome %q/%q got %s want %s", test.left, test.right, got, test.want)
		}
	}
}

func TestC1F2AProfilesBoundButDefaultDeny(t *testing.T) {
	wantFingerprints := map[string]string{
		C1F3ProfileGeneralization: "dc38761583c8856db7b79d515d0f799e416581e84f369649a32aab3053dadd9d",
		C1F3ProfileBoundary:       "3fadbd207c5b340430e81b5a9663f2d7e791e0291fb492d74dd982b358387b94",
	}
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, err := LoadC1F3EvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		if profile.Executable || profile.TypedSidecarIdentity == "" || profile.TypedSidecarSHA256 == "" || profile.TypedSidecarFingerprint == "" || profile.ScoringRubricSHA256 == "" || profile.ScoringRubricFingerprint == "" || profile.DefaultRepetitions != 1 {
			t.Fatalf("C1F3 profile binding incomplete or executable: %+v", profile)
		}
		fingerprint, fingerprintErr := profile.Fingerprint()
		if fingerprintErr != nil {
			t.Fatal(fingerprintErr)
		}
		if fingerprint != wantFingerprints[identity] {
			t.Fatalf("%s fingerprint got %s want %s", identity, fingerprint, wantFingerprints[identity])
		}
		if err = ValidateC1F3ExecutionReadiness(profile); err == nil || !strings.Contains(err.Error(), "non-executable") {
			t.Fatalf("C1F3 profile did not fail closed: %v", err)
		}
	}
}

func TestC1F2AExpectedAnswersCannotEnterProviderRequest(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadFrozenC1F3Manifest(profile, filepath.Join(root, filepath.FromSlash(profile.Dataset.ManifestPath)), exposures)
	if err != nil {
		t.Fatal(err)
	}
	request, err := V6InitialRequest(manifest.Events[19].Input, exposures)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	providerVisible := string(raw)
	for _, forbidden := range []string{C1F3GeneralizationTypedLabelsVersion, C1F3ScoringRubricVersion, C1F2AAdjudicationRubricVersion, "typed_attribution_rationale", "expected_mapping_status", "expected_issuer_attributions", "expected_principal_proxy_candidates", "Apple owns the recall; Belkin supplies evidence"} {
		if strings.Contains(providerVisible, forbidden) {
			t.Fatalf("provider request contains expected-answer material %q", forbidden)
		}
	}
	if request.User == "" || request.System != v6SystemPrompt || request.SchemaContract != V5SchemaVersion || request.SchemaSHA256 != frozenV5SchemaSHA256 {
		t.Fatalf("provider request permitted material changed: %+v", request)
	}
}

func TestC1F2AFrozenSourceHashesPreserved(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	expected := map[string]string{
		"config/ai-shadow-issuer-generalization-holdout-v3.json":                    "6abd7767e0031945e71f2a1d3ef49536adc2d9e9b7d4a78bd938f1f469d27502",
		"config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v3.json": "0b6ab35b963c99dae44d1c43fa657dc23c202a65f6b78f17dcc73af0a145ee8a",
		"config/ai-shadow-issuer-generalization-holdout-freeze-v3.json":             "747beb6e52bbd8d3710674bd79014eee5bfb9d801e902e5a4e2e9e385d9cee6b",
		"config/ai-shadow-issuer-boundary-challenge-v3.json":                        "cb2c93afa18cd889790664e3642fadd57015506af23873942115c29d8be27a56",
		"config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v3.json":     "dcf485e1a56ecc88e66bba0a1b91ca86530b1bb815d39407380ada92f8641d37",
		"config/ai-shadow-issuer-boundary-challenge-freeze-v3.json":                 "46a915f41536872da4e153386eb166612a356b65a6e3aebc197b533e2ae5131c",
		"internal/modules/aishadow/validation_c1f.go":                               frozenC1FValidatorSHA256,
		"internal/modules/aishadow/causal_attribution.go":                           frozenC1EPolicySHA256,
		"internal/modules/aishadow/semantic_identity.go":                            frozenSemanticIdentitySHA256,
		"internal/modules/aishadow/scoring_c1f.go":                                  frozenC1FScoringSourceSHA256,
		"config/event-asset-resolution-v1.json":                                     expectedAssetRulesetFileSHA256,
	}
	for path, want := range expected {
		got, err := diagnosticFileSHA256(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || got != want {
			t.Fatalf("frozen hash %s got %s want %s err=%v", path, got, want, err)
		}
	}
	if V6PromptSHA256() != frozenV6PromptSHA256 {
		t.Fatalf("v6 prompt changed: %s", V6PromptSHA256())
	}
}
