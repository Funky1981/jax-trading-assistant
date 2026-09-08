package advancedquant

import (
	"testing"
	"time"
)

func testDriftPolicy(t *testing.T) DriftPolicy {
	t.Helper()
	p, err := NewDriftPolicy(DriftPolicy{Version: "v1", MaxFeatureDrift: .2, MaxPredictionDrift: .2, MinimumPerformance: 0, RetireAfterDegraded: 2})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDriftStatesAreDeterministicAndFailClosed(t *testing.T) {
	policy := testDriftPolicy(t)
	model, err := NewModelCandidate(ModelCandidate{Name: "model", Algorithm: "threshold", ValidationMetric: .2, BaselineMetric: .1, CostAdjusted: true, SelectionPartition: PartitionValidation, ComplexityRank: 0})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	report, err := EvaluateDrift(model.ID, policy, at, .1, .1, .05, 20, 0)
	if err != nil || report.State != DriftValid {
		t.Fatalf("valid drift report = %+v err=%v", report, err)
	}
	report, err = EvaluateDrift(model.ID, policy, at, .3, .1, .05, 20, 0)
	if err != nil || report.State != DriftDegraded {
		t.Fatalf("breach was not degraded: %+v err=%v", report, err)
	}
	report, err = EvaluateDrift(model.ID, policy, at, .3, .1, .05, 20, 1)
	if err != nil || report.State != DriftRejected {
		t.Fatalf("repeated breach was not rejected: %+v err=%v", report, err)
	}
	if _, err := EvaluateDrift(model.ID, policy, at, .1, .1, .05, 0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := EvaluateDrift("model_bad", policy, at, .1, .1, .05, 20, 0); err == nil {
		t.Fatal("invalid model identity was accepted")
	}
}

func TestRollingRetrainingSeparatesTrainingAndEvaluationWindows(t *testing.T) {
	model, err := NewModelCandidate(ModelCandidate{Name: "model", Algorithm: "threshold", ValidationMetric: .2, BaselineMetric: .1, CostAdjusted: true, SelectionPartition: PartitionValidation, ComplexityRank: 0})
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	window, err := NewRollingRetrainingWindow(RollingRetrainingWindow{ModelID: model.ID, DatasetID: "data", FeatureVersion: "features-v1", TrainStart: base, TrainEnd: base.AddDate(1, 0, 0), EvaluateStart: base.AddDate(1, 0, 1), EvaluateEnd: base.AddDate(1, 6, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if err := window.Validate(); err != nil {
		t.Fatal(err)
	}
	window.EvaluateStart = window.TrainEnd
	if err := window.Validate(); err == nil {
		t.Fatal("overlapping retraining/evaluation window was accepted")
	}
}
