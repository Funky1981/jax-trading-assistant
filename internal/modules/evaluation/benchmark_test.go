package evaluation

import (
	"strings"
	"testing"
	"time"
)

func TestFrozenBenchmarkRegistryFreezesConfigurationAndContent(t *testing.T) {
	benchmark := benchmarkFixture(t)
	registry := NewBenchmarkRegistry()
	if err := registry.Register(benchmark); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(benchmark); err != nil {
		t.Fatal(err)
	}
	got, err := registry.Get(benchmark.ID)
	if err != nil || got.ID != benchmark.ID || got.Configuration.Model != "fixture-model" {
		t.Fatalf("registered benchmark mismatch: %+v err=%v", got, err)
	}
	benchmark.Configuration.Model = "changed-model"
	if err := registry.Register(benchmark); err == nil {
		t.Fatalf("mutated frozen benchmark was accepted: %v", err)
	}
}

func TestFrozenBenchmarkRequiresNewIdentityForChangedDataset(t *testing.T) {
	benchmark := benchmarkFixture(t)
	changed := benchmark
	changed.Version = "v2"
	changed.DatasetContentSHA256 = strings.Repeat("b", 64)
	changed.ID = deriveFrozenBenchmarkID(changed)
	if err := changed.Validate(); err != nil {
		t.Fatal(err)
	}
	if changed.ID == benchmark.ID {
		t.Fatal("changed benchmark reused frozen identity")
	}
}

func TestFinalHoldoutRejectsTuningAndUnknownCases(t *testing.T) {
	benchmark := benchmarkFixture(t)
	benchmark.Split = BenchmarkFinalHoldout
	benchmark.OutcomesUsedForTuning = true
	if err := benchmark.Validate(); err == nil {
		t.Fatal("final holdout accepted outcome tuning")
	}
	benchmark = benchmarkFixture(t)
	benchmark.CaseIDs = append(benchmark.CaseIDs, "hcase_"+strings.Repeat("f", 64))
	if err := benchmark.Validate(); err == nil {
		t.Fatal("benchmark accepted an undeclared decision timestamp")
	}
}

func benchmarkFixture(t *testing.T) FrozenBenchmark {
	t.Helper()
	caseID := "hcase_" + strings.Repeat("a", 64)
	decisionAt := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	benchmark, err := NewFrozenBenchmark(FrozenBenchmark{
		Version: "v1", DatasetContentSHA256: strings.Repeat("a", 64), Split: BenchmarkOutOfSample,
		Instruments: []string{"AAPL", "MSFT"}, StartAt: decisionAt.Add(-24 * time.Hour), EndAt: decisionAt.Add(24 * time.Hour), CaseIDs: []string{caseID}, DecisionTimes: map[string]time.Time{caseID: decisionAt}, AllowedEvidenceVintages: map[string]string{"epi_fixture": "2026-01-05T11:00:00Z"}, ExpectedProtocol: "historical-artifact-replay-v1", Configuration: ConfigurationFreeze{EvidenceSelectionVersion: "jax.evidence-selection/v1", ResearchContractVersion: "jax.structured_research_output/v1", PromptVersion: "prompt-v1", SystemVersion: "system-v1", Provider: "fixture", Model: "fixture-model", RecommendationContractVersion: "jax.research_recommendation/v1", EligibilityRulesVersion: "eligibility-v1", QuantAlgorithmVersions: []string{"quant-v1"}, Thresholds: map[string]float64{"minimum_sources": 2}, CostPolicyVersion: "cost-v1", Universe: []string{"AAPL", "MSFT"}, BenchmarkReference: "SPY"}, FrozenAt: decisionAt, Provenance: BenchmarkProvenance{Source: "fixture-manifest", ManifestRef: "fixture/benchmark-v1", RecordedAt: decisionAt}, VariantsTried: 1})
	if err != nil {
		t.Fatal(err)
	}
	return benchmark
}
