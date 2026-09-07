package quant

import "testing"

func TestPositionSizingUsesExplicitRiskBudgetAndCap(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	result, err := CalculatePositionSizing(dataset, 100, 95, 10000, 0.01, 25)
	if err != nil {
		t.Fatal(err)
	}
	expected := []float64{100, 5, 20, 20}
	for index, value := range result.Values {
		if value.Value != expected[index] {
			t.Fatalf("metric %q=%v, expected %v", value.Metric, value.Value, expected[index])
		}
	}
	if result.AlgorithmVersion != PositionSizingAlgorithmV1 || result.InputSHA256 != dataset.ContentSHA256 {
		t.Fatalf("position sizing provenance missing: %+v", result)
	}
}

func TestPositionSizingAppliesQuantityCap(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	result, err := CalculatePositionSizing(dataset, 100, 99, 10000, 0.02, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Values[3].Value != 10 {
		t.Fatalf("quantity=%v, expected cap 10", result.Values[3].Value)
	}
}

func TestPositionSizingRejectsUnsafeInputs(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	inputs := [][5]float64{
		{100, 100, 10000, 0.01, 25},
		{100, 95, 10000, 0, 25},
		{100, 95, 10000, 1.1, 25},
		{100, 95, 10000, 0.01, 0},
	}
	for _, input := range inputs {
		if _, err := CalculatePositionSizing(dataset, input[0], input[1], input[2], input[3], input[4]); err == nil {
			t.Fatalf("unsafe sizing input accepted: %v", input)
		}
	}
}
