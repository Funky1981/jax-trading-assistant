package aishadow

import (
	"strings"
	"testing"
)

func TestC1FAuditProjectionPreservesAcceptedRelaxedUnresolvedState(t *testing.T) {
	model := v5BaseResult()
	model.MappingConfidence = "HIGH"
	model.IssuerAttributions = []IssuerAttribution{{Issuer: "NVIDIA", CausalRole: CausalRoleContextOnly}}
	raw := marshalV5(t, model)
	resolver := testAssetResolver(t)
	if _, _, _, validationErrors := ParseValidateAndApplyV5(raw, v5TestInput(), resolver); !contains(validationErrors, "UNRESOLVED mapping requires LOW mapping_confidence") {
		t.Fatalf("fixture no longer reproduces the historical projection defect: %v", validationErrors)
	}

	provider := &sequenceProvider{responses: []ProviderResponse{{
		Content: raw, ModelIdentifier: OpenAIDiagnosticLunaModel,
		RequestID: "req-offline", ResponseID: "resp-offline", Status: "completed",
	}}}
	result, attempts, traces, err := analyseC1FEvent(
		Config{Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel}, provider, resolver,
		"offline-run", "offline-manifest", "offline-case", "offline-input", v5TestInput(), mustProxyExposures(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].ValidationStatus != "accepted" || result.V5Parsed == nil {
		t.Fatalf("live C1F route did not accept the relaxed state: %+v", attempts)
	}
	profile, err := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	if err = attachDiagnosticResultProjection(profile, result, attempts); err != nil {
		t.Fatal(err)
	}
	audit, err := buildDiagnosticAttemptAudit(profile, "offline-run", 1, DiagnosticEvent{ID: "offline-case", Category: "offline", Input: v5TestInput()}, attempts[0], traces[0], resolver)
	if err != nil {
		t.Fatal(err)
	}
	if audit.V5RawModelOutput == nil || audit.TypedAttribution == nil || audit.CausalAttributionPolicy == nil || audit.EffectiveSemanticMapping == nil || audit.DeterministicResolution == nil {
		t.Fatalf("C1F evidence projection omitted a persisted layer: %+v", audit)
	}
	if audit.V5RawModelOutput.MappingConfidence != "HIGH" || !audit.CausalAttributionPolicy.Abstained || audit.EffectiveSemanticMapping.MappingStatus != "UNRESOLVED" || audit.DeterministicResolution.Status != "unresolved" {
		t.Fatalf("C1F evidence projection changed the accepted policy/resolver result: %+v", audit)
	}
	if provider.calls != 1 {
		t.Fatalf("offline regression unexpectedly retried: %d", provider.calls)
	}
}

func TestDiagnosticProjectionRoutingIsVersionAwareAndFailsClosed(t *testing.T) {
	v5Profile, err := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	if err != nil {
		t.Fatal(err)
	}
	if route, routeErr := diagnosticProjectionRouteForProfile(v5Profile); routeErr != nil || route != diagnosticProjectionV5 {
		t.Fatalf("historical v5 route changed: route=%q err=%v", route, routeErr)
	}
	c1fProfile, err := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	if route, routeErr := diagnosticProjectionRouteForProfile(c1fProfile); routeErr != nil || route != diagnosticProjectionC1F {
		t.Fatalf("C1F route was not selected explicitly: route=%q err=%v", route, routeErr)
	}

	unknown := c1fProfile
	unknown.ExecutionPromptVersion = "unknown-prompt"
	if _, routeErr := diagnosticProjectionRouteForProfile(unknown); routeErr == nil || !strings.Contains(routeErr.Error(), "C1F route requires prompt") {
		t.Fatalf("unknown route did not fail closed: %v", routeErr)
	}
	mutatedHistorical := v5Profile
	mutatedHistorical.ExecutionCausalPolicy = "unknown-policy"
	if _, routeErr := diagnosticProjectionRouteForProfile(mutatedHistorical); routeErr == nil {
		t.Fatal("unknown historical validator/policy route did not fail closed")
	}
}

func TestAcceptedAuditRejectsMissingOrWrongProjectionRoute(t *testing.T) {
	profile, err := LoadDiagnosticExecutionProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	raw := marshalV5(t, v5BaseResult())
	attempt := Attempt{AttemptNumber: 1, PromptVersion: V6PromptVersion, SchemaVersion: V5SchemaVersion, ValidationStatus: "accepted", RawResponseHash: rawHash(raw)}
	event := DiagnosticEvent{ID: "offline-case", Category: "offline", Input: v5TestInput()}
	if _, err = buildDiagnosticAttemptAudit(profile, "offline-run", 1, event, attempt, ProviderTrace{Content: raw}, testAssetResolver(t)); err == nil || !strings.Contains(err.Error(), "no projection") {
		t.Fatalf("missing accepted projection did not fail closed: %v", err)
	}
	parsed, decision, resolution, validationErrors := ParseValidateAndApplyC1F(raw, event.Input, testAssetResolver(t))
	if len(validationErrors) > 0 {
		t.Fatal(validationErrors)
	}
	attempt.projection = newV5DiagnosticAttemptProjection(parsed, decision, resolution)
	if _, err = buildDiagnosticAttemptAudit(profile, "offline-run", 1, event, attempt, ProviderTrace{Content: raw}, testAssetResolver(t)); err == nil || !strings.Contains(err.Error(), "no projection") {
		t.Fatalf("cross-routed historical projection did not fail closed: %v", err)
	}
}
