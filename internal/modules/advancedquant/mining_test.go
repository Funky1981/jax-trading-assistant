package advancedquant

import "testing"

func testTrial(t *testing.T) FactorTrial {
	t.Helper()
	f := testFeature(t, FeatureKnown, ptr(1))
	factor, err := NewFactorDefinition(FactorDefinition{Name: "event_quality", Version: "v1", Rationale: "registered quality test", InputFeatureIDs: []string{f.ID}, Algorithm: "threshold-v1"})
	if err != nil {
		t.Fatal(err)
	}
	trial, err := NewFactorTrial(FactorTrial{ExperimentID: "exp_test", FactorID: factor.ID, Name: "quality-threshold", HorizonDays: 5, FeatureIDs: []string{f.ID}, Parameters: map[string]string{"threshold": "0.8"}, CostModelID: "cost_phase11_v1"})
	if err != nil {
		t.Fatal(err)
	}
	return trial
}

func ptr(value float64) *float64 { return &value }

func completeFalsifications() []FalsificationResult {
	results := make([]FalsificationResult, 0, len(requiredFalsifications))
	for _, name := range requiredFalsifications {
		results = append(results, FalsificationResult{ContractVersion: FalsificationContractV1, Name: name, Comparable: true, Passed: true, Reason: "registered fixture result"})
	}
	return results
}

func TestTrialLedgerAndFalsificationSuiteAreComplete(t *testing.T) {
	trial := testTrial(t)
	ledger := NewTrialLedger()
	if err := ledger.Add(trial); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Add(trial); err != nil {
		t.Fatal(err)
	}
	if len(ledger.List()) != 1 {
		t.Fatal("duplicate trial was not idempotent")
	}
	if err := ValidateFalsificationSuite(completeFalsifications()); err != nil {
		t.Fatal(err)
	}
	incomplete := completeFalsifications()[:len(requiredFalsifications)-1]
	if err := ValidateFalsificationSuite(incomplete); err == nil {
		t.Fatal("incomplete falsification suite was accepted")
	}
}

func TestStabilityRejectsHoldoutAndFailedPlacebo(t *testing.T) {
	trial := testTrial(t)
	metrics := []MetricResult{{Partition: PartitionDevelopment, ObservationCount: 2, MeanCostAdjustedReturn: .01}, {Partition: PartitionValidation, ObservationCount: 2, MeanCostAdjustedReturn: .01}, {Partition: PartitionOOS, ObservationCount: 2, MeanCostAdjustedReturn: .01}}
	falsifications := completeFalsifications()
	falsifications[2].Passed = false
	report, err := EvaluateStability(trial, StabilityCriteria{MinimumObservationsPerPartition: 2, MaximumIssuerShare: .05, RequireCostAdjusted: true}, metrics, falsifications)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StabilityUnstable {
		t.Fatalf("failed placebo did not reject stability: %+v", report)
	}
	metrics[0].Partition = PartitionFinalHoldout
	if _, err := EvaluateStability(trial, StabilityCriteria{MinimumObservationsPerPartition: 2, MaximumIssuerShare: .05, RequireCostAdjusted: true}, metrics, completeFalsifications()); err == nil {
		t.Fatal("holdout was used for stability")
	}
}
