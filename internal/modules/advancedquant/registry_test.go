package advancedquant

import "testing"

func testRecord(t *testing.T) ExperimentRecord {
	t.Helper()
	f := testFeature(t, FeatureKnown, ptr(1))
	r, err := NewExperimentRecord(ExperimentRecord{HypothesisID: "HYP-EVENT-001A", DatasetID: "data", DatasetHash: "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d", FeatureIDs: []string{f.ID}, Target: "benchmark-relative-return", Horizons: []int{5}, Partitions: []Partition{PartitionDevelopment, PartitionValidation, PartitionOOS}, Baseline: "direction-only", Algorithm: "threshold-v1", SoftwareVersion: "jax-phase12-v1", CostModelID: "cost_phase11_v1"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestExperimentRegistryPreservesEveryImmutableVariant(t *testing.T) {
	record := testRecord(t)
	registry := NewExperimentRegistry()
	if err := registry.Register(record); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(record); err != nil {
		t.Fatal(err)
	}
	if len(registry.List()) != 1 {
		t.Fatal("exact retry was not idempotent")
	}
	record.Parameters = map[string]string{"threshold": "0.8"}
	record.ID = registryExperimentID(record)
	if err := registry.Register(record); err != nil {
		t.Fatal(err)
	}
	if len(registry.List()) != 2 {
		t.Fatal("distinct variant was not retained")
	}
}

func TestResearchOutcomeRejectsHoldoutAndDoesNotPromote(t *testing.T) {
	outcome, err := DecideResearchOutcome("exp", OutcomeCriteria{MinimumOOSObservations: 20, BeatBaseline: true, CostAdjusted: true, PlaceboPassed: true, StableAcrossPeriods: true, Reproducible: true, SurvivorshipControlled: false})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != OutcomeCandidate || outcome.Reason == "" {
		t.Fatalf("unexpected candidate outcome: %+v", outcome)
	}
	if err := outcome.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := DecideResearchOutcome("exp", OutcomeCriteria{MinimumOOSObservations: 20, FinalHoldoutUsed: true}); err == nil {
		t.Fatal("holdout contamination was accepted")
	}
}
