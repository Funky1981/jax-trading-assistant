package aishadow

import (
	"reflect"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func TestAccumulateResolutionAttributesModelAndResolverFailuresSeparately(t *testing.T) {
	event := DiagnosticEvent{Label: DiagnosticLabel{MappingStatus: "PROXY", ProxyExposure: "OIL_CATEGORY", ExpectedResolutionStatus: assetresolution.StatusResolved}}
	modelMismatch := DiagnosticCaseEvaluation{
		ModelClassification:     &StructuredResult{MappingStatus: "PROXY", ProxyExposure: "GOLD_NAMED_MARKET"},
		DeterministicResolution: &PolicyResolution{Status: assetresolution.StatusResolved, ResolvedTicker: "GLD"},
		SemanticCorrect:         false,
		ResolutionCorrect:       false,
	}
	resolverMismatch := DiagnosticCaseEvaluation{
		ModelClassification:     &StructuredResult{MappingStatus: "PROXY", ProxyExposure: "OIL_CATEGORY"},
		DeterministicResolution: &PolicyResolution{Status: assetresolution.StatusResolved, ResolvedTicker: "GLD"},
		SemanticCorrect:         true,
		ResolutionCorrect:       false,
	}

	var metrics DiagnosticResolutionMetrics
	accumulateResolution(&metrics, event, modelMismatch)
	accumulateResolution(&metrics, event, resolverMismatch)

	if metrics.ModelInducedResolutionMismatches != 1 || metrics.ResolverFailuresAfterSemanticMatch != 1 {
		t.Fatalf("resolution attribution=%+v", metrics)
	}
}

func TestRepeatabilityReportsExposureAndDeterministicResolutionVariation(t *testing.T) {
	event := DiagnosticEvent{ID: "case-1"}
	reports := make([]DiagnosticRepetitionReport, diagnosticRepetitionCount)
	allRuns := make([][]DiagnosticCaseRun, diagnosticRepetitionCount)
	exposures := []string{"SP500_NAMED_INDEX", "SEMICONDUCTOR_GROUP", "SEMICONDUCTOR_GROUP"}
	tickers := []string{"SPY", "SOXX", "SOXX"}
	for index := 0; index < diagnosticRepetitionCount; index++ {
		output := &StructuredResult{MappingStatus: "PROXY", ProxyExposure: exposures[index], MappingConfidence: "HIGH"}
		resolution := &PolicyResolution{Status: assetresolution.StatusResolved, ResolvedTicker: tickers[index]}
		reports[index].Cases = []DiagnosticCaseEvaluation{{ModelClassification: output, DeterministicResolution: resolution}}
		allRuns[index] = []DiagnosticCaseRun{{Result: EventResult{Attempt: Attempt{ValidationStatus: "accepted"}, Parsed: output, Resolution: resolution}}}
	}

	result := evaluateRepeatability(DiagnosticManifest{Events: []DiagnosticEvent{event}}, reports, allRuns)
	if len(result.Variations) != 1 {
		t.Fatalf("variations=%+v", result.Variations)
	}
	want := []string{"proxy exposure variation", "deterministic resolution variation"}
	if !reflect.DeepEqual(result.Variations[0].Classification, want) {
		t.Fatalf("classification=%v, want %v", result.Variations[0].Classification, want)
	}
}

func TestTickerEmissionIncludesRejectedAttemptFreeText(t *testing.T) {
	resolver := testAssetResolver(t)
	traces := []ProviderTrace{
		{AttemptNumber: 1, Content: `{"market_relevance":7,"mapping_status":"PROXY","reason":"The event names AMD but does not select one issuer."}`},
		{AttemptNumber: 2, Content: unresolvedJSON()},
	}

	got := detectTickerEmission(traces, nil, resolver)
	want := []string{"ticker token in free text: AMD"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ticker violations=%v, want %v", got, want)
	}
}

func TestDiagnosticReportV2ShowsResolutionAttribution(t *testing.T) {
	report := DiagnosticRunReport{
		Version: DiagnosticReportVersion,
		Repetitions: []DiagnosticRepetitionReport{{
			Repetition: 1,
			Resolution: DiagnosticResolutionMetrics{
				ResolverFailuresAfterSemanticMatch: 2,
				ModelInducedResolutionMismatches:   3,
			},
		}},
	}
	markdown := DiagnosticReportMarkdown(report)
	for _, want := range []string{
		"Resolver failures after semantically correct classification: 2",
		"Model-induced resolution mismatches: 3",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("report markdown does not contain %q: %s", want, markdown)
		}
	}
}
