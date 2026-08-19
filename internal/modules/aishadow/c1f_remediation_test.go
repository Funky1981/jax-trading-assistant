package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func c1fTestRules(t *testing.T) assetresolution.Ruleset {
	t.Helper()
	rules, err := assetresolution.LoadRuleset(filepath.Join("..", "..", "..", "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func TestV6PromptUsesUnchangedV5ContractAndGeneralBoundaries(t *testing.T) {
	request, err := V6InitialRequest(v5TestInput(), mustProxyExposures(t))
	if err != nil {
		t.Fatal(err)
	}
	if request.SchemaContract != V5SchemaVersion {
		t.Fatalf("contract=%s", request.SchemaContract)
	}
	if err := ValidateV5ProviderRequestSchema(request); err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"Do not invent a legal suffix", "Issuer recognition and asset support are separate", "Never choose the closest enum as fallback", "Do not omit another causal issuer", "supplier or customer is not automatically the principal"} {
		if !strings.Contains(request.System, rule) {
			t.Errorf("missing general rule %q", rule)
		}
	}
	if err := ValidateC1FContractRoute(V6PromptVersion, V5SchemaVersion, C1FValidatorVersion, CausalAttributionPolicyVersion, IssuerSemanticIdentityVersion, C1FScoringVersion); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContractRoute(V5PromptVersion, V5SchemaVersion, CausalAttributionPolicyVersion); err != nil {
		t.Fatalf("historical route changed: %v", err)
	}
}

func TestV6CorrectiveRetryForbidsSemanticInvention(t *testing.T) {
	request, err := V6CorrectiveRequest([]string{"DIRECT mapping requires a non-empty direct_issuer"}, `{"mapping_status":"DIRECT"}`, mustProxyExposures(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"Do not invent or substitute any issuer", "proxy candidate", "preserve that uncertainty", "UNRESOLVED"} {
		if !strings.Contains(request.System, rule) {
			t.Errorf("missing corrective safety rule %q", rule)
		}
	}
	if request.RequestKind != "corrective" || request.SchemaContract != V5SchemaVersion {
		t.Fatalf("unexpected request route: %+v", request)
	}
}

func TestC1FCorrectiveRetryIsBoundedAndCanPreserveUncertainty(t *testing.T) {
	invalid := v5BaseResult()
	invalid.MappingStatus = "DIRECT"
	invalid.MappingConfidence = "HIGH"
	invalid.IssuerAttributions = []IssuerAttribution{{Issuer: "semiconductor stocks", CausalRole: CausalRolePrincipal}}
	corrected := v5BaseResult()
	corrected.MappingConfidence = "HIGH"
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: marshalV5(t, invalid)}, {Content: marshalV5(t, corrected)}, {Content: marshalV5(t, corrected)}}}
	result, attempts, _, err := analyseC1FEvent(Config{Provider: "mock", Model: "offline"}, provider, testAssetResolver(t), "run", "fixture", "case", "fingerprint", v5TestInput(), mustProxyExposures(t))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || len(attempts) != 2 || result.RetryCount != 1 || result.ValidationStatus != "accepted" || result.V5Parsed.MappingStatus != "UNRESOLVED" {
		t.Fatalf("C1F bounded semantic-safe correction failed: calls=%d attempts=%+v result=%+v", provider.calls, attempts, result)
	}
	if attempts[1].PromptVersion != V6PromptVersion || provider.requests[1].RequestKind != "corrective" || !strings.Contains(provider.requests[1].System, "Do not invent") {
		t.Fatalf("C1F corrective route lost safety instructions")
	}
}

func TestIssuerSemanticIdentityDeterministicEquivalenceAndNegatives(t *testing.T) {
	id := NewIssuerSemanticIdentity(c1fTestRules(t))
	tests := []struct {
		name, left, right string
		want              IssuerIdentityOutcome
	}{
		{"exact", "Tesla", "tesla", IssuerIdentityExact},
		{"approved alias", "IBM", "International Business Machines Corporation", IssuerIdentityEquivalent},
		{"legal suffix unsupported", "DoorDash", "DoorDash, Inc.", IssuerIdentityEquivalent},
		{"parent subsidiary negative", "Instagram", "Meta Platforms, Inc.", IssuerIdentityDistinct},
		{"brand parent negative", "YouTube", "Alphabet Inc.", IssuerIdentityDistinct},
		{"product company negative", "iPhone", "Apple Inc.", IssuerIdentityDistinct},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := id.Compare(test.left, test.right)
			if got.Outcome != test.want {
				t.Fatalf("%s vs %s: got %+v", test.left, test.right, got)
			}
		})
	}
}

func TestIssuerSemanticIdentityCollisionFailsClosed(t *testing.T) {
	rules := assetresolution.Ruleset{Aliases: []assetresolution.AliasRule{
		{CanonicalEntity: "Mercury Holdings Inc.", Aliases: []string{"Mercury"}, Symbol: "AAA"},
		{CanonicalEntity: "Mercury Systems Inc.", Aliases: []string{"Mercury"}, Symbol: "BBB"},
	}}
	got := NewIssuerSemanticIdentity(rules).Compare("Mercury", "Mercury")
	if got.Outcome != IssuerIdentityAmbiguous {
		t.Fatalf("collision must be ambiguous: %+v", got)
	}
}

func TestC1FValidatorRemovesOnlyUnresolvedConfidenceCoupling(t *testing.T) {
	resolver := testAssetResolver(t)
	result := v5BaseResult()
	result.MappingStatus = "UNRESOLVED"
	result.DirectIssuer = ""
	result.ProxyExposure = NoProxyExposure
	result.MappingConfidence = "HIGH"
	result.IssuerAttributions = []IssuerAttribution{}
	result.PrincipalProxyCandidates = []string{}
	raw := marshalV5(t, result)
	if _, _, _, errs := ParseValidateAndApplyV5(raw, v5TestInput(), resolver); !contains(errs, "UNRESOLVED mapping requires LOW mapping_confidence") {
		t.Fatalf("historical validator no longer reproduces: %v", errs)
	}
	if _, _, _, errs := ParseValidateAndApplyC1F(raw, v5TestInput(), resolver); len(errs) > 0 {
		t.Fatalf("C1F rejected valid high-confidence abstention: %v", errs)
	}
	result.MappingStatus = "DIRECT"
	result.DirectIssuer = ""
	result.IssuerAttributions = []IssuerAttribution{{Issuer: "Apple", CausalRole: CausalRolePrincipal}}
	if _, _, _, errs := ParseValidateAndApplyC1F(marshalV5(t, result), v5TestInput(), resolver); len(errs) == 0 {
		t.Fatal("C1F weakened direct projection invariant")
	}
}

func TestC1FProxyAndRoleRegressionRules(t *testing.T) {
	for _, phrase := range []string{"bounded exposure itself is the principal subject", "secondary effect", "speculative beneficiary", "no uniquely principal supported exposure", "EQUAL_PRINCIPAL", "POSSIBLE_PRINCIPAL", "genuine evidenced downstream effect", "context, comparison, commentary"} {
		if !strings.Contains(v6SystemPrompt, phrase) {
			t.Errorf("missing proxy/role regression rule %q", phrase)
		}
	}
}

func TestC1FProxyTypedStateRegressions(t *testing.T) {
	resolver := testAssetResolver(t)
	exposures := mustProxyExposures(t)
	tests := []struct {
		name, wantStatus string
		mutate           func(*V5StructuredResult)
		wantInvalid      bool
	}{
		{"genuine principal exposure", "PROXY", func(r *V5StructuredResult) {
			r.MappingStatus = "PROXY"
			r.ProxyExposure = exposures[0]
			r.PrincipalProxyCandidates = []string{exposures[0]}
		}, false},
		{"nearest topic is not enough", "UNRESOLVED", func(r *V5StructuredResult) {}, false},
		{"speculative beneficiary is context", "UNRESOLVED", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{Issuer: "Example Co", CausalRole: CausalRoleContextOnly}}
		}, false},
		{"secondary effect is not proxy", "UNRESOLVED", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{Issuer: "Affected Co", CausalRole: CausalRoleSecondaryAffected}}
		}, false},
		{"direct issuer blocks proxy", "", func(r *V5StructuredResult) {
			r.MappingStatus = "DIRECT"
			r.DirectIssuer = "Issuer Co"
			r.MappingConfidence = "HIGH"
			r.IssuerAttributions = []IssuerAttribution{{Issuer: "Issuer Co", CausalRole: CausalRolePrincipal}}
			r.PrincipalProxyCandidates = []string{exposures[0]}
		}, true},
		{"multiple principal exposures abstain", "UNRESOLVED", func(r *V5StructuredResult) { r.PrincipalProxyCandidates = []string{exposures[0], exposures[1]} }, false},
		{"entity is not bounded exposure", "", func(r *V5StructuredResult) {
			r.MappingStatus = "PROXY"
			r.ProxyExposure = "Example Co"
			r.PrincipalProxyCandidates = []string{"Example Co"}
		}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := v5BaseResult()
			test.mutate(&result)
			_, decision, _, errs := ParseValidateAndApplyC1F(marshalV5(t, result), v5TestInput(), resolver)
			if test.wantInvalid {
				if len(errs) == 0 {
					t.Fatal("expected invalid typed state")
				}
				return
			}
			if len(errs) > 0 || decision == nil || decision.EffectiveMapping.MappingStatus != test.wantStatus {
				t.Fatalf("decision=%+v errors=%v", decision, errs)
			}
		})
	}
}

func TestC1FRoleContractRegressions(t *testing.T) {
	resolver := testAssetResolver(t)
	tests := []struct {
		name, status, reason string
		attrs                []IssuerAttribution
	}{
		{"principal and secondary", "DIRECT", ReasonUniquePrincipalIssuerAccepted, []IssuerAttribution{{Issuer: "Affected Customer", CausalRole: CausalRolePrincipal}, {Issuer: "Upstream Supplier", CausalRole: CausalRoleSecondaryAffected}}},
		{"principal and context", "DIRECT", ReasonUniquePrincipalIssuerAccepted, []IssuerAttribution{{Issuer: "Current Subject", CausalRole: CausalRolePrincipal}, {Issuer: "Historical Peer", CausalRole: CausalRoleContextOnly}}},
		{"equal principals", "UNRESOLVED", ReasonMultipleEqualPrincipals, []IssuerAttribution{{Issuer: "Issuer A", CausalRole: CausalRoleEqualPrincipal}, {Issuer: "Issuer B", CausalRole: CausalRoleEqualPrincipal}}},
		{"possible principal ambiguity", "UNRESOLVED", ReasonPossiblePrincipal, []IssuerAttribution{{Issuer: "Issuer A", CausalRole: CausalRolePossiblePrincipal}, {Issuer: "Issuer B", CausalRole: CausalRolePossiblePrincipal}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := v5BaseResult()
			result.IssuerAttributions = test.attrs
			if test.status == "DIRECT" {
				result.MappingStatus = "DIRECT"
				result.DirectIssuer = test.attrs[0].Issuer
				result.MappingConfidence = "HIGH"
			}
			_, decision, _, errs := ParseValidateAndApplyC1F(marshalV5(t, result), v5TestInput(), resolver)
			if len(errs) > 0 || decision == nil || decision.EffectiveMapping.MappingStatus != test.status || decision.ReasonCode != test.reason {
				t.Fatalf("decision=%+v errors=%v", decision, errs)
			}
			if len(decision.RawMapping.DirectIssuer) > 0 && len(test.attrs) < 2 {
				t.Fatal("fixture must retain attribution completeness context")
			}
		})
	}
}

func TestC1F3ProfilesAreMetadataOnlyAndDefaultDeny(t *testing.T) {
	for _, name := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		p, err := LoadC1F3EvaluationProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		if p.Executable || p.TypedSidecarIdentity != "" || p.ScoringRubricSHA256 != "" {
			t.Fatalf("profile must be unbound: %+v", p)
		}
		if _, err = p.Fingerprint(); err != nil {
			t.Fatal(err)
		}
		if err = ValidateC1F3ExecutionReadiness(p); err == nil || !strings.Contains(err.Error(), "C1F2A typed sidecar") {
			t.Fatalf("profile did not fail closed: %v", err)
		}
	}
	g := FrozenC1F3QualityGates()
	if g.FinalValidity != 98 || g.DirectPrecision != 95 || g.DirectRecall != 90 || g.SemanticFalseDirect != 5 || g.MaximumIncorrectTickerResolutions != 0 {
		t.Fatalf("quality gates changed: %+v", g)
	}
}

func TestC1E3OfflineDualScoringReplay(t *testing.T) {
	rules := c1fTestRules(t)
	identity := NewIssuerSemanticIdentity(rules)
	tests := []struct {
		name, labelPath, attemptPattern                               string
		strictMap, semanticMap, semanticAttr, semanticComplete, total int
	}{
		{"Generalization", "config/ai-shadow-causal-attribution-labels-generalization-v2-v1.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-generalization-v2/WP-00.03C1E3-GENERALIZATION/60f4a31f-7363-4801-8724-36a76add70aa/repetition-01/*-attempt-*.json", 19, 44, 39, 45, 48},
		{"Boundary", "config/ai-shadow-causal-attribution-labels-boundary-v2-v1.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-boundary-v2/WP-00.03C1E3-BOUNDARY/e736ac51-485a-44a2-b3e1-8e0812ae3793/repetition-01/*-attempt-*.json", 12, 25, 22, 26, 32},
	}
	root := filepath.Join("..", "..", "..")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(test.labelPath)))
			if err != nil {
				t.Fatal(err)
			}
			var sidecar TypedLabelSidecar
			if err = json.Unmarshal(raw, &sidecar); err != nil {
				t.Fatal(err)
			}
			paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(test.attemptPattern)))
			if err != nil {
				t.Fatal(err)
			}
			audits := []DiagnosticAttemptAudit{}
			for _, path := range paths {
				raw, err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var audit DiagnosticAttemptAudit
				if err = json.Unmarshal(raw, &audit); err != nil {
					t.Fatal(err)
				}
				audits = append(audits, audit)
			}
			score := ScoreC1FDataset(test.name, sidecar.Cases, audits, identity)
			if score.Strict.WholeMapping.Numerator != test.strictMap || score.Strict.WholeMapping.Denominator != test.total {
				t.Fatalf("strict old score not reproduced: %+v", score.Strict.WholeMapping)
			}
			if score.Strict.FinalValidity.Numerator != test.total {
				t.Fatalf("strict final validity not reproduced: %+v", score.Strict.FinalValidity)
			}
			if test.name == "Generalization" && (score.Strict.WholeAttribution.Numerator != 6 || score.Strict.AttributionCompleteness.Numerator != 9 || score.Strict.Principal.Denominator != 25 || score.Strict.EqualPrincipal.Denominator != 6 || score.Strict.SecondaryAffected.Denominator != 2 || score.Strict.ContextOnly.Denominator != 13 || score.Strict.PossiblePrincipal.Numerator != 1 || score.Strict.PossiblePrincipal.Denominator != 2 || score.Strict.PrincipalProxyCandidate.Numerator != 43) {
				t.Fatalf("strict Generalization typed metrics not reproduced: %+v", score.Strict)
			}
			if test.name == "Boundary" && (score.Strict.WholeAttribution.Numerator != 4 || score.Strict.AttributionCompleteness.Numerator != 6 || score.Strict.Principal.Denominator != 17 || score.Strict.EqualPrincipal.Denominator != 3 || score.Strict.SecondaryAffected.Denominator != 2 || score.Strict.ContextOnly.Denominator != 12 || score.Strict.PossiblePrincipal.Numerator != 1 || score.Strict.PossiblePrincipal.Denominator != 2 || score.Strict.PrincipalProxyCandidate.Numerator != 26) {
				t.Fatalf("strict Boundary typed metrics not reproduced: %+v", score.Strict)
			}
			if score.Semantic.WholeMapping.Numerator != test.semanticMap || score.Semantic.WholeAttribution.Numerator != test.semanticAttr || score.Semantic.AttributionCompleteness.Numerator != test.semanticComplete {
				t.Fatalf("normalized forensic replay mismatch: mapping=%+v attr=%+v completeness=%+v", score.Semantic.WholeMapping, score.Semantic.WholeAttribution, score.Semantic.AttributionCompleteness)
			}
		})
	}
}

func TestC1FValidatorOfflineRetryProjection(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	resolver := testAssetResolver(t)
	tests := []struct{ manifest, attempts string }{
		{"config/ai-shadow-issuer-generalization-holdout-v2.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-generalization-v2/WP-00.03C1E3-GENERALIZATION/60f4a31f-7363-4801-8724-36a76add70aa/repetition-01/*-attempt-01.json"},
		{"config/ai-shadow-issuer-boundary-challenge-v2.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-boundary-v2/WP-00.03C1E3-BOUNDARY/e736ac51-485a-44a2-b3e1-8e0812ae3793/repetition-01/*-attempt-01.json"},
	}
	oldRejected, confidenceOnly, c1fRejected := 0, 0, 0
	for _, test := range tests {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(test.manifest)))
		if err != nil {
			t.Fatal(err)
		}
		var manifest DiagnosticManifest
		if err = json.Unmarshal(raw, &manifest); err != nil {
			t.Fatal(err)
		}
		inputs := map[string]EventInput{}
		for _, event := range manifest.Events {
			inputs[event.ID] = event.Input
		}
		paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(test.attempts)))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			raw, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var audit DiagnosticAttemptAudit
			if err = json.Unmarshal(raw, &audit); err != nil {
				t.Fatal(err)
			}
			_, _, _, oldErrors := ParseValidateAndApplyV5(audit.RawResponseBody, inputs[audit.CaseID], resolver)
			_, _, _, newErrors := ParseValidateAndApplyC1F(audit.RawResponseBody, inputs[audit.CaseID], resolver)
			if len(oldErrors) > 0 {
				oldRejected++
			}
			if len(oldErrors) == 1 && oldErrors[0] == "UNRESOLVED mapping requires LOW mapping_confidence" {
				confidenceOnly++
			}
			if len(newErrors) > 0 {
				c1fRejected++
			}
		}
	}
	if oldRejected != 21 || confidenceOnly != 19 || c1fRejected != 2 {
		t.Fatalf("retry projection old=%d confidence-only=%d c1f=%d", oldRejected, confidenceOnly, c1fRejected)
	}
}

func TestGenerateC1F2OfflineFreeze(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path, hash, err := GenerateC1F2OfflineFreeze(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || !strings.HasSuffix(filepath.ToSlash(path), "/ai-shadow-c1f2-offline-freeze-v1/WP-00.03C1F2/offline-freeze.json") {
		t.Fatalf("unexpected freeze result path=%s hash=%s", path, hash)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var freeze C1F2OfflineFreeze
	if err = json.Unmarshal(raw, &freeze); err != nil {
		t.Fatal(err)
	}
	if freeze.ProviderContact || freeze.Inference || freeze.HostedAuthorization || freeze.CredentialLoaded || freeze.DatabaseMutation || freeze.TradingMutation || freeze.V3ContentsInspected {
		t.Fatalf("unsafe offline evidence state: %+v", freeze)
	}
	if len(freeze.Profiles) != 2 || len(freeze.V3) != 2 || len(freeze.DevelopmentReplay) != 2 {
		t.Fatalf("incomplete freeze: %+v", freeze)
	}
}
