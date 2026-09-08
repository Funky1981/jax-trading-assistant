package advancedquant

import "testing"

func TestModelSelectionKeepsBaselineWhenAdvancedValueIsNotProven(t *testing.T) {
	candidate, err := NewModelCandidate(ModelCandidate{Name: "deterministic_candidate", Algorithm: "threshold-v1", ValidationMetric: .11, BaselineMetric: .10, CostAdjusted: true, SelectionPartition: PartitionValidation, ComplexityRank: 2})
	if err != nil {
		t.Fatal(err)
	}
	result, err := SelectModel("direction-only", .10, []ModelCandidate{candidate}, .02, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != SelectionBaselineWins || result.SelectedModelID != "" {
		t.Fatalf("baseline was not retained: %+v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestModelSelectionSelectsOnlyFrozenValidationCandidate(t *testing.T) {
	candidate, err := NewModelCandidate(ModelCandidate{Name: "simple_quality", Algorithm: "threshold-v1", ValidationMetric: .20, BaselineMetric: .10, CostAdjusted: true, SelectionPartition: PartitionValidation, ComplexityRank: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := SelectModel("direction-only", .10, []ModelCandidate{candidate}, .02, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != SelectionCandidateSelected || result.SelectedModelID != candidate.ID {
		t.Fatalf("candidate was not selected: %+v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := SelectModel("direction-only", .10, []ModelCandidate{candidate}, .02, true, false); err == nil {
		t.Fatal("unvalidated ensemble was accepted")
	}
}

func TestModelSelectionRejectsOOSOrUncostedEvidence(t *testing.T) {
	_, err := NewModelCandidate(ModelCandidate{Name: "oos_tuned", Algorithm: "ml", ValidationMetric: .2, BaselineMetric: .1, CostAdjusted: true, SelectionPartition: PartitionOOS, ComplexityRank: 1})
	if err == nil {
		t.Fatal("OOS candidate was allowed to drive selection")
	}
	_, err = NewModelCandidate(ModelCandidate{Name: "free_result", Algorithm: "threshold-v1", ValidationMetric: .2, BaselineMetric: .1, CostAdjusted: false, SelectionPartition: PartitionValidation, ComplexityRank: 1})
	if err == nil {
		t.Fatal("uncosted candidate was allowed")
	}
}
