package aishadow

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func repeatabilityPerfectFixture(t *testing.T) ([]TypedExpectedCase, DiagnosticManifest, []DiagnosticAttemptAudit, IssuerSemanticIdentity, assetresolution.Resolver) {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	frozen, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	sidecar, err := LoadFrozenC1F3TypedLabelSidecar(frozen, filepath.Join(root, filepath.FromSlash(frozen.TypedSidecarPath)))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadFrozenC1F3Manifest(frozen, filepath.Join(root, filepath.FromSlash(frozen.Dataset.ManifestPath)), exposures)
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]EventInput{}
	for _, event := range manifest.Events {
		inputs[event.ID] = event.Input
	}
	audits := make([]DiagnosticAttemptAudit, 0, len(sidecar.Cases))
	for _, expected := range sidecar.Cases {
		mapping := &AssetMapping{MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure}
		resolution := &PolicyResolution{Status: expected.ExpectedDeterministicResolutionStatus}
		if expected.ExpectedMappingStatus != "UNRESOLVED" {
			resolved, err := c1f3ExpectedResolution(expected, inputs[expected.CaseID], resolver)
			if err != nil {
				t.Fatal(err)
			}
			resolution = resolved
		}
		audits = append(audits, DiagnosticAttemptAudit{
			CaseID: expected.CaseID, AttemptNumber: 1, ValidationStatus: "accepted", EffectiveSemanticMapping: mapping,
			TypedAttribution:        &TypedCausalAttribution{IssuerAttributions: expected.ExpectedIssuerAttributions, PrincipalProxyCandidates: expected.ExpectedPrincipalProxyCandidates},
			DeterministicResolution: resolution,
		})
	}
	return sidecar.Cases, manifest, audits, NewIssuerSemanticIdentity(rules), resolver
}

func TestC1F3RepeatabilityScoringExactAgreementAndAccuracyRetention(t *testing.T) {
	labels, manifest, original, identity, resolver := repeatabilityPerfectFixture(t)
	score := ScoreC1F3Repeatability(labels, manifest, original, original, identity, resolver, 0)
	if score.Denominator != 48 || score.Metrics.SemanticEffectiveMapping.Numerator != 48 || score.Metrics.SemanticEffectiveMapping.Percentage != 100 ||
		len(score.ChangedCases) != 0 || score.AccuracyContextCounts["both correct"] != 48 || !score.Passed || score.IncorrectDeterministicResolutions != 0 || score.SafetyPersistenceViolations != 0 {
		t.Fatalf("perfect repeatability score failed: %+v", score)
	}
	for _, metric := range []C1F3Rate{score.Metrics.StrictMapping, score.Metrics.WholeTypedAttribution, score.Metrics.Principal, score.Metrics.EqualPrincipal,
		score.Metrics.SecondaryAffected, score.Metrics.ContextOnly, score.Metrics.PossiblePrincipal, score.Metrics.PrincipalProxyCandidate,
		score.Metrics.AttributionCompleteness, score.Metrics.PolicyDecision, score.Metrics.DeterministicResolution} {
		if metric.Numerator != 48 || metric.Denominator != 48 {
			t.Fatalf("report-only metric did not use frozen denominator: %+v", metric)
		}
	}
}

func TestC1F3RepeatabilityChangedCaseReportsAccuracyContextAndLayers(t *testing.T) {
	labels, manifest, repeat, identity, resolver := repeatabilityPerfectFixture(t)
	original := append([]DiagnosticAttemptAudit(nil), repeat...)
	original[0].EffectiveSemanticMapping = &AssetMapping{MappingStatus: "UNRESOLVED"}
	original[0].CausalAttributionPolicy = &CausalAttributionDecision{Abstained: true}
	repeat[0].CausalAttributionPolicy = &CausalAttributionDecision{Accepted: true}
	score := ScoreC1F3Repeatability(labels, manifest, original, repeat, identity, resolver, 0)
	if score.Metrics.SemanticEffectiveMapping.Numerator != 47 || score.Metrics.SemanticEffectiveMapping.Denominator != 48 || len(score.ChangedCases) != 1 {
		t.Fatalf("changed-case score mismatch: %+v", score.Metrics)
	}
	changed := score.ChangedCases[0]
	if changed.CaseID != labels[0].CaseID || changed.AccuracyContext != "original wrong / repeat correct" || !changed.PolicyDiffers || changed.IdentityOutcome != "N/A" {
		t.Fatalf("changed-case reporting incomplete: %+v", changed)
	}
	if score.AccuracyContextCounts["original wrong / repeat correct"] != 1 || score.AccuracyContextCounts["both correct"] != 47 {
		t.Fatalf("accuracy-context reporting incomplete: %+v", score.AccuracyContextCounts)
	}
}

func TestC1F3RepeatabilityAccuracyRetentionCanFailDespiteStableMapping(t *testing.T) {
	labels, manifest, original, identity, resolver := repeatabilityPerfectFixture(t)
	degraded := append([]DiagnosticAttemptAudit(nil), original...)
	for index := range degraded {
		degraded[index].ValidationStatus = "rejected"
	}
	score := ScoreC1F3Repeatability(labels, manifest, degraded, degraded, identity, resolver, 0)
	if score.Metrics.SemanticEffectiveMapping.Percentage != 100 || score.Passed {
		t.Fatalf("stable but invalid repeat run passed: %+v", score)
	}
}
