package quant

import (
	"math"
	"testing"
)

func TestRiskAdjustedMetricsUsesExplicitKnownValues(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 110, 99, 118.8})
	result, err := CalculateRiskAdjustedMetrics(dataset, 3, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	returns := []float64{0.1, -0.1, 0.2}
	average := (returns[0] + returns[1] + returns[2]) / 3
	variance := ((returns[0]-average)*(returns[0]-average) + (returns[1]-average)*(returns[1]-average) + (returns[2]-average)*(returns[2]-average)) / 2
	expectedSharpe := average / math.Sqrt(variance)
	expectedSortino := average / math.Sqrt((0.1*0.1)/3)
	maxDrawdown := 99.0/110.0 - 1
	expectedCalmar := math.Pow(118.8/100, 1.0/3.0) - 1
	expectedCalmar /= math.Abs(maxDrawdown)
	if math.Abs(result.Values[0].Value-expectedSharpe) > 1e-12 || math.Abs(result.Values[1].Value-expectedSortino) > 1e-12 || math.Abs(result.Values[2].Value-expectedCalmar) > 1e-12 {
		t.Fatalf("risk-adjusted=%+v expected sharpe=%v sortino=%v calmar=%v", result, expectedSharpe, expectedSortino, expectedCalmar)
	}
}

func TestRiskAdjustedMetricsRejectsDegenerateDenominators(t *testing.T) {
	flat := returnsDataset(t, []float64{100, 100, 100})
	if _, err := CalculateRiskAdjustedMetrics(flat, 2, 252, 0, 0); err == nil {
		t.Fatal("zero-variance metrics accepted")
	}
	noDrawdown := returnsDataset(t, []float64{100, 101, 102})
	if _, err := CalculateRiskAdjustedMetrics(noDrawdown, 2, 252, 0, 0); err == nil {
		t.Fatal("zero-drawdown Calmar accepted")
	}
}

func TestRiskAdjustedSortinoUsesDownsideTargetNumerator(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 110, 99, 118.8})
	result, err := CalculateRiskAdjustedMetrics(dataset, 3, 1, 0.1, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	returns := []float64{0.1, -0.1, 0.2}
	periodicTarget := 0.1
	sum := 0.0
	downsideSquares := 0.0
	for _, value := range returns {
		sum += value
		shortfall := value - periodicTarget
		if shortfall < 0 {
			downsideSquares += shortfall * shortfall
		}
	}
	expected := (sum/3 - periodicTarget) / math.Sqrt(downsideSquares/3)
	if math.Abs(result.Values[1].Value-expected) > 1e-12 {
		t.Fatalf("sortino=%v expected %v", result.Values[1].Value, expected)
	}
}
