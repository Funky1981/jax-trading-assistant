package advancedquant

import (
	"testing"
	"time"
)

func testProtocol(t *testing.T) ResearchProtocol {
	t.Helper()
	d := func(y, m, day int) time.Time { return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC) }
	p, err := NewResearchProtocol(ResearchProtocol{
		HypothesisID: "HYP-EVENT-001A", DatasetID: "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1", DatasetManifestHash: "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d", EventFamily: "FORM 8-K", EntryRule: "NEXT_REGULAR_SESSION_OPEN_AFTER_PUBLIC_AVAILABILITY", Benchmark: "SPY", PrimaryHorizonDays: 5, SecondaryHorizonDays: []int{1, 3}, PrimaryMetric: "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN", MetricVersion: "metric_v1", CostModelID: "cost_phase11_v1", FinalHoldoutSealed: true,
		Windows: []DateWindow{{PartitionDevelopment, d(2016, 1, 1), d(2021, 12, 31)}, {PartitionValidation, d(2022, 1, 1), d(2023, 12, 31)}, {PartitionOOS, d(2024, 1, 1), d(2024, 12, 31)}, {PartitionFinalHoldout, d(2025, 1, 1), d(2025, 12, 31)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func testObservations(t *testing.T, p ResearchProtocol) []ResearchObservation {
	t.Helper()
	event := time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC)
	entry := event.Add(24 * time.Hour)
	return []ResearchObservation{{EventID: "evt_1", IssuerID: "issuer_1", Partition: PartitionOOS, EventAt: event, EntryAt: entry, ExitAt: entry.Add(5 * 24 * time.Hour), PredictedDirection: 1, EvidenceQuality: .9, BenchmarkRelativeReturn: .04, CostAdjustedReturn: .03}, {EventID: "evt_2", IssuerID: "issuer_2", Partition: PartitionOOS, EventAt: event.Add(24 * time.Hour), EntryAt: entry.Add(24 * time.Hour), ExitAt: entry.Add(10 * 24 * time.Hour), PredictedDirection: -1, EvidenceQuality: .2, BenchmarkRelativeReturn: .01, CostAdjustedReturn: 0}, {EventID: "evt_3", IssuerID: "issuer_3", Partition: PartitionValidation, EventAt: time.Date(2022, 2, 1, 15, 0, 0, 0, time.UTC), EntryAt: time.Date(2022, 2, 2, 14, 0, 0, 0, time.UTC), ExitAt: time.Date(2022, 2, 9, 14, 0, 0, 0, time.UTC), PredictedDirection: 1, EvidenceQuality: .8, BenchmarkRelativeReturn: .02, CostAdjustedReturn: .01}}
}

func TestProtocolEnforcesFrozenPartitionsAndOOSOnce(t *testing.T) {
	p := testProtocol(t)
	observations := testObservations(t, p)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "5-day SPY-relative return", Baseline: "DIRECTION_ONLY", Algorithm: "evidence-quality-threshold-v1", Parameters: map[string]string{"minimum_evidence_quality": "0.8"}, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	run := &FrozenOOSRun{}
	if _, err := run.ScoreOOS(observations, p); err == nil {
		t.Fatal("unfrozen OOS was scored")
	}
	if err := run.Freeze(config); err != nil {
		t.Fatal(err)
	}
	result, err := run.ScoreOOS(observations, p)
	if err != nil {
		t.Fatal(err)
	}
	if result.ObservationCount != 1 || result.MeanCostAdjustedReturn != .03 {
		t.Fatalf("unexpected frozen OOS result: %+v", result)
	}
	if _, err := run.ScoreOOS(observations, p); err == nil {
		t.Fatal("OOS was scored twice")
	}
	if err := run.Freeze(config); err == nil {
		t.Fatal("frozen config changed")
	}
}

func TestProtocolRejectsHoldoutAndFutureDerivedObservation(t *testing.T) {
	p := testProtocol(t)
	o := testObservations(t, p)[0]
	o.Partition = PartitionFinalHoldout
	if err := o.Validate(p); err == nil {
		t.Fatal("final holdout observation was accepted")
	}
	o = testObservations(t, p)[0]
	o.EntryAt = o.EventAt
	if err := o.Validate(p); err == nil {
		t.Fatal("same-event entry was accepted")
	}
	o = testObservations(t, p)[0]
	o.ExitAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := o.Validate(p); err == nil {
		t.Fatal("outcome crossing its partition boundary was accepted")
	}
	if _, err := ScoreDirectionOnly("exp_test", []ResearchObservation{o}, PartitionFinalHoldout, p); err == nil {
		t.Fatal("holdout scoring was accepted")
	}
}

func TestMetricBaselineIsDeterministic(t *testing.T) {
	p := testProtocol(t)
	a := testObservations(t, p)[:2]
	left, err := ScoreDirectionOnly("exp_test", a, PartitionOOS, p)
	if err != nil {
		t.Fatal(err)
	}
	right, err := ScoreDirectionOnly("exp_test", a, PartitionOOS, p)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("baseline changed between identical runs: %+v != %+v", left, right)
	}
}
