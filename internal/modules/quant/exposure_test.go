package quant

import "testing"

func TestPortfolioExposureUsesSignedValuesAndStableBreakdown(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	positions := []Position{
		{Instrument: "BBB", Quantity: -2, Price: 50},
		{Instrument: "AAA", Quantity: 10, Price: 100},
	}
	result, err := CalculatePortfolioExposure(dataset, positions)
	if err != nil {
		t.Fatal(err)
	}
	// Independent arithmetic: +1000 market value and -100 market value.
	expected := []float64{1100, 900, 1000, 100, 1000.0 / 1100.0}
	for index, value := range result.Response.Values[:5] {
		if value.Value != expected[index] {
			t.Fatalf("metric %q=%v, expected %v", value.Metric, value.Value, expected[index])
		}
	}
	if len(result.Instruments) != 2 || result.Instruments[0].Instrument != "AAA" || result.Instruments[1].Instrument != "BBB" {
		t.Fatalf("instrument order=%+v", result.Instruments)
	}
	if result.Response.InputSHA256 != dataset.ContentSHA256 || len(result.Response.InputReferences) != 1 {
		t.Fatalf("portfolio provenance missing: %+v", result.Response)
	}
}

func TestPortfolioExposureIsInvariantToInputOrder(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	left, err := CalculatePortfolioExposure(dataset, []Position{{Instrument: "AAA", Quantity: 2, Price: 10}, {Instrument: "BBB", Quantity: -1, Price: 5}})
	if err != nil {
		t.Fatal(err)
	}
	right, err := CalculatePortfolioExposure(dataset, []Position{{Instrument: "BBB", Quantity: -1, Price: 5}, {Instrument: "AAA", Quantity: 2, Price: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if left.Response.RequestID != right.Response.RequestID || left.Response.InputReferences[0] != right.Response.InputReferences[0] {
		t.Fatalf("input order changed deterministic identity: %q vs %q", left.Response.RequestID, right.Response.RequestID)
	}
}

func TestPortfolioExposureRejectsInvalidPositions(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	invalid := []Position{{Instrument: "", Quantity: 1, Price: 10}, {Instrument: "AAA", Quantity: 0, Price: 10}, {Instrument: "AAA", Quantity: 1, Price: 0}}
	for _, position := range invalid {
		if _, err := CalculatePortfolioExposure(dataset, []Position{position}); err == nil {
			t.Fatalf("invalid position accepted: %+v", position)
		}
	}
}
