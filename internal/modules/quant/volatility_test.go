package quant

import (
	"math"
	"testing"
)

func TestVolatilityKnownSampleAndAnnualization(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 110, 99})
	result, err := CalculateVolatility(dataset, 2, 1)
	first, second := math.Log(1.1), math.Log(0.9)
	mean := (first + second) / 2
	expected := math.Sqrt(((first-mean)*(first-mean) + (second-mean)*(second-mean)) / 1)
	if err != nil || len(result.Values) != 1 || math.Abs(result.Values[0].Value-expected) > 1e-12 || result.AlgorithmVersion != VolatilityAlgorithmV1 {
		t.Fatalf("volatility=%+v expected=%.15f err=%v", result, expected, err)
	}
}

func TestATRUsesPriorCloseTrueRangeAndDrawdownUsesRunningPeak(t *testing.T) {
	bars := []Bar{
		{At: returnsDataset(t, []float64{1, 1, 1, 1}).Bars[0].At, Open: 100, High: 105, Low: 95, Close: 100, Volume: 1},
		{At: returnsDataset(t, []float64{1, 1, 1, 1}).Bars[1].At, Open: 100, High: 110, Low: 99, Close: 105, Volume: 1},
		{At: returnsDataset(t, []float64{1, 1, 1, 1}).Bars[2].At, Open: 105, High: 106, Low: 90, Close: 95, Volume: 1},
	}
	dataset, err := NewFrozenDataset("atr_dd", "TEST", "1D", "AS_PROVIDED", "frozen-volatility-fixture-v1", bars[0].At, bars)
	if err != nil {
		t.Fatal(err)
	}
	atr, err := CalculateATR(dataset, 2)
	if err != nil || math.Abs(atr.Values[0].Value-13.5) > 1e-12 {
		t.Fatalf("ATR=%+v err=%v", atr, err)
	}
	drawdown, err := CalculateDrawdown(dataset)
	if err != nil || math.Abs(drawdown.Values[0].Value-(95.0/105.0-1)) > 1e-12 || math.Abs(drawdown.Values[1].Value-(95.0/105.0-1)) > 1e-12 {
		t.Fatalf("drawdown=%+v err=%v", drawdown, err)
	}
}

func TestVolatilityAndATRRejectInsufficientHistory(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101})
	if _, err := CalculateVolatility(dataset, 2, 252); err == nil {
		t.Fatal("insufficient volatility history accepted")
	}
	if _, err := CalculateATR(dataset, 2); err == nil {
		t.Fatal("insufficient ATR history accepted")
	}
}
