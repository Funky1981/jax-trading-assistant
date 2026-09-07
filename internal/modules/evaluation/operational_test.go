package evaluation

import (
	"strings"
	"testing"
	"time"
)

func operationalAdapter(t *testing.T) BacktestAdapterAssessment {
	t.Helper()
	candidate, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "custom event replay", Version: "v1", Language: "Go", License: "Jax proprietary", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low", IntegrationBurden: "low", DecisionStatus: AdapterSelected, AssessmentNotes: "bounded fixture"})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := NewBacktestAdapterAssessment(time.Date(2026, 9, 7, 17, 0, 0, 0, time.UTC), "operational fixture", []BacktestAdapterCandidate{candidate}, candidate.ID, "existing deterministic boundary")
	if err != nil {
		t.Fatal(err)
	}
	return assessment
}

func operationalBenchmarks(t *testing.T, caseFile HistoricalCase) (FrozenBenchmark, map[string]FrozenBenchmark, []WalkForwardWindow) {
	t.Helper()
	decision := caseFile.DecisionAt
	makeBenchmark := func(split BenchmarkSplit, caseID, suffix string, start, end, caseDecision time.Time) FrozenBenchmark {
		benchmark := benchmarkFixture(t)
		benchmark.Split = split
		benchmark.Version = "operational-" + suffix
		benchmark.DatasetContentSHA256 = strings.Repeat(suffix, 64)
		benchmark.StartAt = start
		benchmark.EndAt = end
		benchmark.CaseIDs = []string{caseID}
		benchmark.DecisionTimes = map[string]time.Time{caseID: caseDecision}
		benchmark.FrozenAt = start
		benchmark.Provenance.RecordedAt = start
		benchmark.ID = deriveFrozenBenchmarkID(benchmark)
		if err := benchmark.Validate(); err != nil {
			t.Fatal(err)
		}
		return benchmark
	}
	development := makeBenchmark(BenchmarkDevelopment, "hcase_"+strings.Repeat("d", 64), "d", decision.Add(-9*24*time.Hour), decision.Add(-7*24*time.Hour), decision.Add(-8*24*time.Hour))
	validation := makeBenchmark(BenchmarkValidation, "hcase_"+strings.Repeat("e", 64), "e", decision.Add(-6*24*time.Hour), decision.Add(-4*24*time.Hour), decision.Add(-5*24*time.Hour))
	outOfSample := makeBenchmark(BenchmarkOutOfSample, caseFile.ID, "f", decision.Add(-24*time.Hour), decision.Add(24*time.Hour), decision)
	benchmarks := map[string]FrozenBenchmark{development.ID: development, validation.ID: validation, outOfSample.ID: outOfSample}
	windows := []WalkForwardWindow{{WindowNumber: 1, Split: development.Split, BenchmarkID: development.ID, CaseIDs: development.CaseIDs, StartAt: development.StartAt, EndAt: development.EndAt}, {WindowNumber: 2, Split: validation.Split, BenchmarkID: validation.ID, CaseIDs: validation.CaseIDs, StartAt: validation.StartAt, EndAt: validation.EndAt}, {WindowNumber: 3, Split: outOfSample.Split, BenchmarkID: outOfSample.ID, CaseIDs: outOfSample.CaseIDs, StartAt: outOfSample.StartAt, EndAt: outOfSample.EndAt}}
	return outOfSample, benchmarks, windows
}

func operationalReportFixture(t *testing.T) (OperationalReplayReport, HistoricalCase, FrozenBenchmark, map[string]FrozenBenchmark, WalkForwardProtocol, BacktestAdapterAssessment, CostSlippagePolicy, HypotheticalTradingCost) {
	t.Helper()
	caseFile := replayFixture(t)
	benchmark, benchmarks, windows := operationalBenchmarks(t, caseFile)
	protocol, err := NewWalkForwardProtocol([]FrozenBenchmark{benchmarks[windows[0].BenchmarkID], benchmarks[windows[1].BenchmarkID], benchmark}, windows, "v1", "candidate-v1", "cost-v1", "decision-time frozen evidence only", caseFile.DecisionAt.Add(-48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := ReplayHistoricalCase(caseFile)
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]OutcomeObservation, 20)
	for index := range observations {
		observations[index] = OutcomeObservation{At: caseFile.DecisionAt.Add(time.Duration(index+1) * time.Hour), Price: 100 + float64(index)}
	}
	outcome, err := TrackRecommendationOutcome(caseFile, observations, nil, "operational fixture", caseFile.DecisionAt.Add(21*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	adapter := operationalAdapter(t)
	policy := costPolicyFixture(t)
	cost, err := ApplyHypotheticalTradingCost(policy, "BUY", 100, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildOperationalReplayReport(benchmark, caseFile, replay, outcome, protocol, benchmarks, adapter, policy, cost)
	if err != nil {
		t.Fatal(err)
	}
	return report, caseFile, benchmark, benchmarks, protocol, adapter, policy, cost
}

func TestBuildOperationalReplayReportProvesReplayOutcomeAndOOSBindings(t *testing.T) {
	report, caseFile, benchmark, benchmarks, protocol, adapter, policy, cost := operationalReportFixture(t)
	if err := report.Validate(benchmark, caseFile, outcomeForReport(t, caseFile, report), protocol, benchmarks, adapter, policy, cost); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(report.ID, "opreplay_") || report.EvaluationStatus != "EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE" || report.EdgeConclusion != "UNKNOWN_INSUFFICIENT_SAMPLE_NO_EDGE_CLAIM" {
		t.Fatalf("operational report omitted bounded evaluation status: %#v", report)
	}
}

func outcomeForReport(t *testing.T, caseFile HistoricalCase, report OperationalReplayReport) RecommendationOutcome {
	t.Helper()
	observations := make([]OutcomeObservation, 20)
	for index := range observations {
		observations[index] = OutcomeObservation{At: caseFile.DecisionAt.Add(time.Duration(index+1) * time.Hour), Price: 100 + float64(index)}
	}
	outcome, err := TrackRecommendationOutcome(caseFile, observations, nil, "operational fixture", caseFile.DecisionAt.Add(21*time.Hour))
	if err != nil || outcome.ID != report.OutcomeID {
		t.Fatalf("could not reconstruct report outcome: err=%v id=%s report=%s", err, outcome.ID, report.OutcomeID)
	}
	return outcome
}

func TestBuildOperationalReplayReportRejectsTamperedReplayBinding(t *testing.T) {
	report, caseFile, benchmark, benchmarks, protocol, adapter, policy, cost := operationalReportFixture(t)
	report.ContextPlanID = "tampered"
	if err := report.Validate(benchmark, caseFile, outcomeForReport(t, caseFile, report), protocol, benchmarks, adapter, policy, cost); err == nil {
		t.Fatal("expected tampered context binding rejection")
	}
}

func TestBuildOperationalReplayReportRejectsNonOOSBenchmark(t *testing.T) {
	report, caseFile, benchmark, benchmarks, protocol, adapter, policy, cost := operationalReportFixture(t)
	benchmark.Split = BenchmarkValidation
	benchmark.ID = deriveFrozenBenchmarkID(benchmark)
	report.BenchmarkID = benchmark.ID
	if err := report.Validate(benchmark, caseFile, outcomeForReport(t, caseFile, report), protocol, benchmarks, adapter, policy, cost); err == nil {
		t.Fatal("expected non-OOS benchmark rejection")
	}
}
