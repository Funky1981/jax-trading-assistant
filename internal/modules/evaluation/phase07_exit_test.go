package evaluation

import "testing"

func TestPhase07ExitFrozenCaseReplayOutcomeAndOutOfSampleEvidence(t *testing.T) {
	report, caseFile, benchmark, benchmarks, protocol, adapter, policy, cost := operationalReportFixture(t)
	replay, err := ReplayHistoricalCase(caseFile)
	if err != nil {
		t.Fatal(err)
	}
	outcome := outcomeForReport(t, caseFile, report)
	if !replay.Reconstructed || replay.ContextPlanID != caseFile.ContextPlan.ID || replay.ResearchOutputID != caseFile.Research.ID || replay.RecommendationID != caseFile.Recommendation.ID {
		t.Fatalf("frozen case did not reproduce its research chain: %#v", replay)
	}
	if !protocolContainsCase(protocol, benchmark.ID, BenchmarkOutOfSample, caseFile.ID) || protocol.CandidateLogicVersion == "" || !report.OutOfSampleEvidence || !report.OutcomeLeakageChecked {
		t.Fatal("candidate logic lacks explicit leakage-checked out-of-sample evidence")
	}
	if caseFile.Recommendation.ExecutionAuthority != "NONE" || report.ExecutionAuthority != "NONE" || report.InferenceUsed {
		t.Fatal("Phase 07 evidence crossed the research-only safety boundary")
	}
	if report.EvaluationStatus != "EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE" || report.EdgeConclusion != "UNKNOWN_INSUFFICIENT_SAMPLE_NO_EDGE_CLAIM" {
		t.Fatal("single-case evidence was allowed to claim an edge")
	}
	if err := outcome.Validate(caseFile); err != nil {
		t.Fatal(err)
	}
	repeated, err := BuildOperationalReplayReport(benchmark, caseFile, replay, outcome, protocol, benchmarks, adapter, policy, cost)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ID != report.ID {
		t.Fatalf("same frozen phase evidence did not reproduce report identity: %s != %s", repeated.ID, report.ID)
	}
}
