package aishadow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type repeatabilityRoutingProvider struct {
	content  string
	snapshot HostedExperimentSnapshot
	calls    int
}

func (p *repeatabilityRoutingProvider) Complete(ProviderRequest) (ProviderResponse, error) {
	p.calls++
	return ProviderResponse{
		Content: p.content, ModelIdentifier: OpenAIDiagnosticLunaModel,
		RequestID: fmt.Sprintf("req-offline-%02d", p.calls), ResponseID: fmt.Sprintf("resp-offline-%02d", p.calls), Status: "completed",
	}, nil
}

func (p *repeatabilityRoutingProvider) ExperimentSnapshot() HostedExperimentSnapshot {
	snapshot := p.snapshot
	snapshot.RequestCount = p.calls
	return snapshot
}

func repeatabilityRoutingSnapshot(prepared PreparedDiagnostic) HostedExperimentSnapshot {
	plan := prepared.Plan.HostedExperiment
	return HostedExperimentSnapshot{
		ExperimentID: plan.ExperimentID, Provider: prepared.Config.Provider, RequestedModel: prepared.Config.Model,
		ReasoningEffort: prepared.Plan.ModelConfiguration.ReasoningEffort, ServiceTier: prepared.Plan.ModelConfiguration.ServiceTier,
		ThinkingMode: prepared.Plan.ModelConfiguration.ThinkingMode, StructuredOutputMode: prepared.Plan.ModelConfiguration.StructuredOutputMode,
		MaxOutputTokensPerRequest: prepared.Plan.ModelConfiguration.MaxOutputTokens, Pricing: plan.Pricing, BudgetCeilingUSD: plan.BudgetCeilingUSD,
	}
}

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

func TestHistoricalV5AuditProjectionRemainsOnHistoricalRoute(t *testing.T) {
	profile, err := LoadDiagnosticEvaluationProfile(DiagnosticProfileGeneralizationV2)
	if err != nil {
		t.Fatal(err)
	}
	model := v5BaseResult()
	raw := marshalV5(t, model)
	provider := &sequenceProvider{responses: []ProviderResponse{{
		Content: raw, ModelIdentifier: OpenAIDiagnosticLunaModel,
		RequestID: "req-v5-offline", ResponseID: "resp-v5-offline", Status: "completed",
	}}}
	resolver := testAssetResolver(t)
	input := v5TestInput()
	result, attempts, traces, err := analyseV5Event(
		Config{Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel}, provider, resolver,
		"offline-v5-run", "offline-v5-manifest", "offline-v5-case", "offline-v5-input", input, mustProxyExposures(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = attachDiagnosticResultProjection(profile, result, attempts); err != nil {
		t.Fatal(err)
	}
	audit, err := buildDiagnosticAttemptAudit(profile, "offline-v5-run", 1, DiagnosticEvent{ID: "offline-v5-case", Category: "offline", Input: input}, attempts[0], traces[0], resolver)
	if err != nil {
		t.Fatal(err)
	}
	if attempts[0].projection == nil || attempts[0].projection.route != diagnosticProjectionV5 || audit.V5RawModelOutput == nil || audit.CausalAttributionPolicy == nil || audit.DeterministicResolution == nil {
		t.Fatalf("historical v5 projection route changed: attempt=%+v audit=%+v", attempts[0], audit)
	}
}

func TestC1F3RepeatabilityR2ExecutionPersistsC1FProjectionOffline(t *testing.T) {
	profile := repeatabilityTestProfile(t)
	config := repeatabilityTestConfig(t, profile, false, false, false)
	prepared, err := PrepareHostedDiagnosticPreflight(repeatabilityTestPaths(t, profile), config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	model := v5BaseResult()
	model.MappingConfidence = "HIGH"
	raw := marshalV5(t, model)
	if _, _, _, historicalErrors := ParseValidateAndApplyV5(raw, v5TestInput(), prepared.Resolver); !contains(historicalErrors, "UNRESOLVED mapping requires LOW mapping_confidence") {
		t.Fatalf("fixture no longer distinguishes C1F from historical v5: %v", historicalErrors)
	}
	provider := &repeatabilityRoutingProvider{content: raw, snapshot: repeatabilityRoutingSnapshot(prepared)}
	firstCaseID := prepared.Manifest.Events[0].ID
	report, paths, err := ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: OpenAIDiagnosticLunaModel})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 48 || len(report.Repetitions) != 1 || len(report.Repetitions[0].Cases) != 48 {
		t.Fatalf("repeatability mock execution did not complete 48 x 1: calls=%d report=%+v", provider.calls, report.Repetitions)
	}
	firstAttempt := filepath.Join(paths.Directory, "repetition-01", firstCaseID+"-attempt-01.json")
	rawAudit, err := os.ReadFile(firstAttempt)
	if err != nil {
		t.Fatal(err)
	}
	var audit DiagnosticAttemptAudit
	if err := json.Unmarshal(rawAudit, &audit); err != nil {
		t.Fatal(err)
	}
	if audit.V5RawModelOutput == nil || audit.TypedAttribution == nil || audit.CausalAttributionPolicy == nil || audit.EffectiveSemanticMapping == nil || audit.DeterministicResolution == nil {
		t.Fatalf("repeatability C1F projection omitted persisted evidence: %+v", audit)
	}
	if audit.V5RawModelOutput.MappingConfidence != "HIGH" || audit.EffectiveSemanticMapping.MappingStatus != "UNRESOLVED" || audit.DeterministicResolution.Status != "unresolved" {
		t.Fatalf("repeatability C1F projection changed accepted synthetic evidence: %+v", audit)
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
	r2Profile := repeatabilityTestProfile(t)
	if route, routeErr := diagnosticProjectionRouteForProfile(r2Profile); routeErr != nil || route != diagnosticProjectionC1FRepeatabilityR2 {
		t.Fatalf("consumed r2 C1F route was not preserved for offline evidence: route=%q err=%v", route, routeErr)
	}
	r3Profile := repeatabilityR3TestProfile(t)
	if route, routeErr := diagnosticProjectionRouteForProfile(r3Profile); routeErr != nil || route != diagnosticProjectionC1FRepeatabilityR3 {
		t.Fatalf("r3 C1F route was not selected explicitly: route=%q err=%v", route, routeErr)
	}

	unknown := c1fProfile
	unknown.ExecutionPromptVersion = "unknown-prompt"
	if _, routeErr := diagnosticProjectionRouteForProfile(unknown); routeErr == nil || !strings.Contains(routeErr.Error(), "requires prompt") {
		t.Fatalf("unknown route did not fail closed: %v", routeErr)
	}
	mutatedHistorical := v5Profile
	mutatedHistorical.ExecutionCausalPolicy = "unknown-policy"
	if _, routeErr := diagnosticProjectionRouteForProfile(mutatedHistorical); routeErr == nil {
		t.Fatal("unknown historical validator/policy route did not fail closed")
	}
	mutatedValidator := r3Profile
	mutatedValidator.ExecutionValidatorVersion = DiagnosticValidatorV5
	if _, routeErr := diagnosticProjectionRouteForProfile(mutatedValidator); routeErr == nil || !strings.Contains(routeErr.Error(), "requires validator") {
		t.Fatalf("mismatched r3 validator did not fail closed: %v", routeErr)
	}
	if _, configErr := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(c1e2bConfigValues(mutatedValidator)), mutatedValidator, false); configErr == nil || !strings.Contains(configErr.Error(), "requires validator") {
		t.Fatalf("mismatched r3 validator reached provider configuration: %v", configErr)
	}
	unknownFuture := r3Profile
	unknownFuture.Identity = "openai-hosted-c1f3-repeatability-generalization-v3-r4"
	if _, routeErr := diagnosticProjectionRouteForProfile(unknownFuture); routeErr == nil || !strings.Contains(routeErr.Error(), "unknown diagnostic execution contract") {
		t.Fatalf("unknown future profile did not fail closed: %v", routeErr)
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
