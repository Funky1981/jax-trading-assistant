package evaluation

import (
	"sort"
	"strings"
	"testing"
	"time"
)

func walkBenchmark(t *testing.T, split BenchmarkSplit, start, end time.Time, suffix string) FrozenBenchmark {
	t.Helper()
	benchmark := benchmarkFixture(t)
	benchmark.Split = split
	benchmark.Version = "v1-" + suffix
	benchmark.StartAt = start
	benchmark.EndAt = end
	for caseID := range benchmark.DecisionTimes {
		benchmark.DecisionTimes[caseID] = start.Add(24 * time.Hour)
	}
	benchmark.FrozenAt = start
	benchmark.Provenance.RecordedAt = start
	benchmark.DatasetContentSHA256 = strings.Repeat(string(suffix[0]), 64)
	benchmark.ID = deriveFrozenBenchmarkID(benchmark)
	if split == BenchmarkFinalHoldout {
		benchmark.OutcomesUsedForTuning = false
		benchmark.HoldoutTouched = false
	}
	if err := benchmark.Validate(); err != nil {
		t.Fatal(err)
	}
	return benchmark
}

func walkProtocolFixture(t *testing.T) (WalkForwardProtocol, map[string]FrozenBenchmark) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	development := walkBenchmark(t, BenchmarkDevelopment, base, base.Add(10*24*time.Hour), "a")
	validation := walkBenchmark(t, BenchmarkValidation, base.Add(11*24*time.Hour), base.Add(20*24*time.Hour), "b")
	outOfSample := walkBenchmark(t, BenchmarkOutOfSample, base.Add(21*24*time.Hour), base.Add(30*24*time.Hour), "c")
	benchmarks := []FrozenBenchmark{development, validation, outOfSample}
	windows := []WalkForwardWindow{{WindowNumber: 1, Split: BenchmarkDevelopment, BenchmarkID: development.ID, CaseIDs: development.CaseIDs, StartAt: development.StartAt, EndAt: development.EndAt}, {WindowNumber: 2, Split: BenchmarkValidation, BenchmarkID: validation.ID, CaseIDs: validation.CaseIDs, StartAt: validation.StartAt, EndAt: validation.EndAt}, {WindowNumber: 3, Split: BenchmarkOutOfSample, BenchmarkID: outOfSample.ID, CaseIDs: outOfSample.CaseIDs, StartAt: outOfSample.StartAt, EndAt: outOfSample.EndAt}}
	protocol, err := NewWalkForwardProtocol(benchmarks, windows, "v1", "candidate-v1", "cost-v1", "Only decision-time and earlier frozen evidence may enter a case.", base.Add(20*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]FrozenBenchmark{}
	for _, benchmark := range benchmarks {
		result[benchmark.ID] = benchmark
	}
	return protocol, result
}

func TestWalkForwardProtocolRequiresChronologicalFrozenOutOfSampleEvidence(t *testing.T) {
	protocol, benchmarks := walkProtocolFixture(t)
	if err := protocol.Validate(benchmarks); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(protocol.ID, "walk_") || protocol.CandidateLogicVersion != "candidate-v1" || protocol.FinalHoldoutUntouched != true {
		t.Fatalf("invalid walk-forward protocol: %#v", protocol)
	}
}

func TestWalkForwardProtocolRejectsOutOfSampleTuningAndOverlap(t *testing.T) {
	protocol, benchmarks := walkProtocolFixture(t)
	protocol.Windows[2].OutcomesUsedForTuning = true
	if err := protocol.Validate(benchmarks); err == nil {
		t.Fatal("expected out-of-sample tuning rejection")
	}
	protocol, benchmarks = walkProtocolFixture(t)
	protocol.Windows[1].StartAt = protocol.Windows[0].EndAt
	if err := protocol.Validate(benchmarks); err == nil {
		t.Fatal("expected overlapping windows rejection")
	}
}

func TestWalkForwardProtocolRejectsConfigurationFrozenAfterOutOfSampleStart(t *testing.T) {
	protocol, benchmarks := walkProtocolFixture(t)
	protocol.ConfigurationFrozenAt = protocol.Windows[2].StartAt
	if err := protocol.Validate(benchmarks); err == nil {
		t.Fatal("expected late configuration freeze rejection")
	}
}

func TestWalkForwardProtocolRejectsFinalHoldoutMutation(t *testing.T) {
	protocol, benchmarks := walkProtocolFixture(t)
	final := walkBenchmark(t, BenchmarkFinalHoldout, protocol.Windows[2].EndAt.Add(24*time.Hour), protocol.Windows[2].EndAt.Add(10*24*time.Hour), "d")
	benchmarks[final.ID] = final
	protocol.BenchmarkIDs = append(protocol.BenchmarkIDs, final.ID)
	sort.Strings(protocol.BenchmarkIDs)
	protocol.Windows = append(protocol.Windows, WalkForwardWindow{WindowNumber: 4, Split: BenchmarkFinalHoldout, BenchmarkID: final.ID, CaseIDs: final.CaseIDs, StartAt: final.StartAt, EndAt: final.EndAt})
	protocol.ID = deriveWalkForwardProtocolID(protocol)
	if err := protocol.Validate(benchmarks); err != nil {
		t.Fatal(err)
	}
	protocol.Windows[3].OutcomesUsedForTuning = true
	if err := protocol.Validate(benchmarks); err == nil {
		t.Fatal("expected final holdout mutation rejection")
	}
}
