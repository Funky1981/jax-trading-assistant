package aishadow

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func v5TestInput() EventInput {
	publication := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
	return EventInput{Title: "Synthetic event", Summary: "Synthetic development fixture", Source: "offline", PublicationTimestamp: publication, ReceiptTimestamp: publication.Add(time.Minute)}
}

func v5BaseResult() V5StructuredResult {
	return V5StructuredResult{
		MarketRelevance: "MEDIUM", MappingStatus: "UNRESOLVED", DirectIssuer: "", ProxyExposure: NoProxyExposure,
		MappingConfidence: "LOW", ExpectedHorizon: "MULTI_DAY", LikelyDirection: "UNCLEAR",
		CatalystType: "synthetic fixture", Reason: "Synthetic offline policy fixture with no provider inference.",
		MissingEvidence: []string{}, IssuerAttributions: []IssuerAttribution{}, PrincipalProxyCandidates: []string{},
	}
}

func marshalV5(t *testing.T, result V5StructuredResult) string {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func firstProxyExposure(t *testing.T) string {
	t.Helper()
	exposures, err := testAssetResolver(t).ProxyExposures()
	if err != nil || len(exposures) < 2 {
		t.Fatalf("test resolver requires two proxy exposures: %v %#v", err, exposures)
	}
	return exposures[0]
}

func TestV5TypedPolicyAcceptedStates(t *testing.T) {
	resolver := testAssetResolver(t)
	exposures, err := resolver.ProxyExposures()
	if err != nil || len(exposures) < 2 {
		t.Fatalf("proxy exposures: %v %#v", err, exposures)
	}
	tests := []struct {
		name, status, reason string
		mutate               func(*V5StructuredResult)
	}{
		{"one principal despite low relevance", "DIRECT", ReasonUniquePrincipalIssuerAccepted, func(r *V5StructuredResult) {
			r.MarketRelevance, r.MappingStatus, r.DirectIssuer, r.MappingConfidence = "LOW", "DIRECT", "Unknown Example plc", "HIGH"
			r.IssuerAttributions = []IssuerAttribution{{"Unknown Example plc", CausalRolePrincipal}, {"Context Corp", CausalRoleContextOnly}}
		}},
		{"equal principals", "UNRESOLVED", ReasonMultipleEqualPrincipals, func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"Alpha Corp", CausalRoleEqualPrincipal}, {"Beta Corp", CausalRoleEqualPrincipal}}
		}},
		{"possible principal", "UNRESOLVED", ReasonPossiblePrincipal, func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"Alpha Corp", CausalRolePossiblePrincipal}, {"Context Corp", CausalRoleContextOnly}}
		}},
		{"unique proxy", "PROXY", ReasonUniquePrincipalProxyAccepted, func(r *V5StructuredResult) {
			r.MappingStatus, r.ProxyExposure, r.MappingConfidence = "PROXY", exposures[0], "LOW"
			r.IssuerAttributions = []IssuerAttribution{{"Commentator Corp", CausalRoleContextOnly}}
			r.PrincipalProxyCandidates = []string{exposures[0]}
		}},
		{"multiple proxies", "UNRESOLVED", ReasonMultiplePrincipalProxies, func(r *V5StructuredResult) {
			r.PrincipalProxyCandidates = []string{exposures[0], exposures[1]}
		}},
		{"secondary only", "UNRESOLVED", ReasonNoPrincipalMapping, func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"Affected Corp", CausalRoleSecondaryAffected}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := v5BaseResult()
			test.mutate(&result)
			parsed, decision, resolution, validationErrors := ParseValidateAndApplyV5(marshalV5(t, result), v5TestInput(), resolver)
			if len(validationErrors) != 0 || parsed == nil || decision == nil || resolution == nil {
				t.Fatalf("unexpected validation result: parsed=%v decision=%v resolution=%v errors=%v", parsed, decision, resolution, validationErrors)
			}
			if decision.EffectiveMapping.MappingStatus != test.status || decision.ReasonCode != test.reason {
				t.Fatalf("decision=%+v want status=%s reason=%s", decision, test.status, test.reason)
			}
			if result.MappingStatus == "DIRECT" && resolution.Status != "unresolved" {
				t.Fatalf("unknown principal must remain semantic DIRECT while resolver may be unresolved: %+v", resolution)
			}
		})
	}
}

func TestV5TypedPolicyRejectsImpossibleProjections(t *testing.T) {
	resolver := testAssetResolver(t)
	proxy := firstProxyExposure(t)
	tests := []struct {
		name, want string
		mutate     func(*V5StructuredResult)
	}{
		{"duplicate normalized issuer", "unique after deterministic normalization", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"Example Corp.", CausalRoleContextOnly}, {" example corp ", CausalRoleSecondaryAffected}}
		}},
		{"multiple principal", "at most one PRINCIPAL", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}, {"B", CausalRolePrincipal}}
		}},
		{"single equal", "at least two", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRoleEqualPrincipal}}
		}},
		{"principal and possible", "mutually exclusive", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}, {"B", CausalRolePossiblePrincipal}}
		}},
		{"principal with proxy candidate", "require empty", func(r *V5StructuredResult) {
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}}
			r.PrincipalProxyCandidates = []string{proxy}
		}},
		{"direct without principal", "DIRECT requires exactly one", func(r *V5StructuredResult) {
			r.MappingStatus, r.DirectIssuer, r.MappingConfidence = "DIRECT", "A", "HIGH"
		}},
		{"direct issuer mismatch", "exactly match", func(r *V5StructuredResult) {
			r.MappingStatus, r.DirectIssuer, r.MappingConfidence = "DIRECT", "B", "HIGH"
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}}
		}},
		{"proxy with principal", "PROXY requires no principal", func(r *V5StructuredResult) {
			r.MappingStatus, r.ProxyExposure = "PROXY", proxy
			r.PrincipalProxyCandidates = []string{proxy}
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}}
		}},
		{"proxy with equal principals", "PROXY requires no principal", func(r *V5StructuredResult) {
			r.MappingStatus, r.ProxyExposure = "PROXY", proxy
			r.PrincipalProxyCandidates = []string{proxy}
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRoleEqualPrincipal}, {"B", CausalRoleEqualPrincipal}}
		}},
		{"proxy with possible principal", "PROXY requires no principal", func(r *V5StructuredResult) {
			r.MappingStatus, r.ProxyExposure = "PROXY", proxy
			r.PrincipalProxyCandidates = []string{proxy}
			r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePossiblePrincipal}}
		}},
		{"proxy without candidate", "exactly one", func(r *V5StructuredResult) { r.MappingStatus, r.ProxyExposure = "PROXY", proxy }},
		{"proxy with multiple candidates", "exactly one", func(r *V5StructuredResult) {
			exposures := mustProxyExposures(t)
			r.MappingStatus, r.ProxyExposure = "PROXY", exposures[0]
			r.PrincipalProxyCandidates = []string{exposures[0], exposures[1]}
		}},
		{"proxy candidate mismatch", "exactly match", func(r *V5StructuredResult) {
			r.MappingStatus, r.ProxyExposure, r.MappingConfidence = "PROXY", proxy, "LOW"
			r.PrincipalProxyCandidates = []string{proxy + "_wrong"}
		}},
		{"unresolved unique principal", "UNRESOLVED requires", func(r *V5StructuredResult) { r.IssuerAttributions = []IssuerAttribution{{"A", CausalRolePrincipal}} }},
		{"unresolved unique proxy", "UNRESOLVED requires", func(r *V5StructuredResult) { r.PrincipalProxyCandidates = []string{proxy} }},
		{"candidate none", "not an active bounded exposure", func(r *V5StructuredResult) { r.PrincipalProxyCandidates = []string{NoProxyExposure, proxy} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := v5BaseResult()
			test.mutate(&result)
			parsed, decision, resolution, errors := ParseValidateAndApplyV5(marshalV5(t, result), v5TestInput(), resolver)
			if parsed != nil || decision != nil || resolution != nil || !hasError(errors, test.want) {
				t.Fatalf("invalid projection was not rejected before policy/resolution: parsed=%v decision=%v resolution=%v errors=%v", parsed, decision, resolution, errors)
			}
		})
	}
}

func TestV5MissingAttributionStructureFailsValidation(t *testing.T) {
	v4 := StructuredResult{
		MarketRelevance: "HIGH", MappingStatus: "DIRECT", DirectIssuer: "Example plc", ProxyExposure: NoProxyExposure,
		MappingConfidence: "HIGH", ExpectedHorizon: "ONE_DAY", LikelyDirection: "POSITIVE",
		CatalystType: "synthetic", Reason: "Synthetic contract-routing fixture with no provider inference.", MissingEvidence: []string{},
	}
	raw, err := json.Marshal(v4)
	if err != nil {
		t.Fatal(err)
	}
	parsed, decision, resolution, validationErrors := ParseValidateAndApplyV5(string(raw), v5TestInput(), testAssetResolver(t))
	if parsed != nil || decision != nil || resolution != nil || !hasError(validationErrors, "missing required field: issuer_attributions") || !hasError(validationErrors, "missing required field: principal_proxy_candidates") {
		t.Fatalf("v4-shaped output did not fail closed under v5: parsed=%v decision=%v resolution=%v errors=%v", parsed, decision, resolution, validationErrors)
	}
}

func TestV5PolicyDoesNotUseProseOrRelevance(t *testing.T) {
	result := v5BaseResult()
	result.MappingStatus, result.DirectIssuer, result.MappingConfidence = "DIRECT", "Example plc", "HIGH"
	result.IssuerAttributions = []IssuerAttribution{{"Example plc", CausalRolePrincipal}}
	first, err := ApplyCausalAttributionPolicy(result)
	if err != nil {
		t.Fatal(err)
	}
	result.MarketRelevance = "UNCERTAIN"
	result.Reason = strings.Repeat("contradictory audit prose ", 2)
	result.MissingEvidence = []string{"different audit prose"}
	second, err := ApplyCausalAttributionPolicy(result)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("typed policy coupled to prose or relevance: first=%+v second=%+v", first, second)
	}
}

func TestV5CorrectiveRetryIsBoundedAndPreservesV5Schema(t *testing.T) {
	valid := v5BaseResult()
	valid.MappingStatus, valid.DirectIssuer, valid.MappingConfidence = "DIRECT", "Unknown Example plc", "HIGH"
	valid.IssuerAttributions = []IssuerAttribution{{"Unknown Example plc", CausalRolePrincipal}}
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: `{}`}, {Content: marshalV5(t, valid)}}}
	result, attempts, _, err := analyseV5Event(Config{Provider: "mock", Model: "offline"}, provider, testAssetResolver(t), "run", "fixture", "case", "fingerprint", v5TestInput(), mustProxyExposures(t))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || len(attempts) != 2 || result.RetryCount != 1 || result.ValidationStatus != "accepted" {
		t.Fatalf("bounded corrective retry failed: calls=%d attempts=%d result=%+v", provider.calls, len(attempts), result)
	}
	if attempts[0].SchemaVersion != V5SchemaVersion || attempts[1].PromptVersion != V5PromptVersion || provider.requests[1].RequestKind != "corrective" {
		t.Fatalf("retry lost v5 routing: attempts=%+v request=%+v", attempts, provider.requests[1])
	}
	if len(provider.requests[1].Schema["required"].([]string)) != len(requiredV5ResultFields) {
		t.Fatalf("corrective request did not preserve canonical v5 schema")
	}
}

func mustProxyExposures(t *testing.T) []string {
	t.Helper()
	exposures, err := testAssetResolver(t).ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	return exposures
}

func TestV5PersistenceSeparatesRawAttributionEffectiveAndResolver(t *testing.T) {
	model := v5BaseResult()
	model.MappingStatus, model.DirectIssuer, model.MappingConfidence = "DIRECT", "Unknown Example plc", "HIGH"
	model.IssuerAttributions = []IssuerAttribution{{"Unknown Example plc", CausalRolePrincipal}, {"Context Corp", CausalRoleContextOnly}}
	_, decision, resolution, errors := ParseValidateAndApplyV5(marshalV5(t, model), v5TestInput(), testAssetResolver(t))
	if len(errors) != 0 {
		t.Fatal(errors)
	}
	raw, err := persistedResultJSON(EventResult{Attempt: Attempt{SchemaVersion: V5SchemaVersion}, V5Parsed: &model, CausalAttribution: decision, Resolution: resolution})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePersistedResult(V5SchemaVersion, []byte(raw.(string)))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.V5 == nil || decoded.V5.RawModelOutput.DirectIssuer != "Unknown Example plc" || len(decoded.V5.TypedAttribution.IssuerAttributions) != 2 || decoded.V5.EffectiveSemanticMapping.MappingStatus != "DIRECT" || decoded.V5.DeterministicResolution.Status != "unresolved" {
		t.Fatalf("v5 evidence layers not preserved: %+v", decoded.V5)
	}
}

func TestV5DiagnosticAuditSeparatesEveryEvidenceLayerAndExcludesC1D(t *testing.T) {
	model := v5BaseResult()
	model.MappingStatus, model.DirectIssuer, model.MappingConfidence = "DIRECT", "Unknown Example plc", "HIGH"
	model.IssuerAttributions = []IssuerAttribution{{"Unknown Example plc", CausalRolePrincipal}, {"Context Corp", CausalRoleContextOnly}}
	raw := marshalV5(t, model)
	parsed, decision, resolution, validationErrors := ParseValidateAndApplyV5(raw, v5TestInput(), testAssetResolver(t))
	if len(validationErrors) > 0 {
		t.Fatal(validationErrors)
	}
	profile, err := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := buildDiagnosticAttemptAudit(profile, "run", 1, DiagnosticEvent{ID: "synthetic", Category: "offline", Input: v5TestInput()}, Attempt{
		AttemptNumber: 1, SchemaVersion: V5SchemaVersion, PromptVersion: V5PromptVersion,
		ValidationStatus: "accepted", RawResponseHash: rawHash(raw), projection: newV5DiagnosticAttemptProjection(parsed, decision, resolution),
	}, ProviderTrace{Content: raw}, testAssetResolver(t))
	if err != nil {
		t.Fatal(err)
	}
	if audit.V5RawModelOutput == nil || audit.TypedAttribution == nil || audit.CausalAttributionPolicy == nil || audit.EffectiveSemanticMapping == nil || audit.DeterministicResolution == nil {
		t.Fatalf("v5 audit omitted an evidence layer: %+v", audit)
	}
	if audit.ModelClassification != nil || audit.CausalConsistencyGuard != nil || audit.CausalAttributionPolicy.PolicyVersion != CausalAttributionPolicyVersion {
		t.Fatalf("v5 audit activated or conflated the historical C1D path: %+v", audit)
	}
}
