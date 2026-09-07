package quant

import (
	"math"
	"testing"
	"time"
)

func TestLiquidityVolumeAnomalyKnownValuePreservesVolumeSource(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bars := []Bar{{At: start, Open: 10, High: 10, Low: 10, Close: 10, Volume: 100}, {At: start.AddDate(0, 0, 1), Open: 10, High: 10, Low: 10, Close: 10, Volume: 110}, {At: start.AddDate(0, 0, 2), Open: 10, High: 10, Low: 10, Close: 10, Volume: 120}, {At: start.AddDate(0, 0, 3), Open: 10, High: 10, Low: 10, Close: 10, Volume: 500}}
	dataset, err := NewFrozenDatasetWithVolumeSource("liquidity_known", "TEST", "1D", "AS_PROVIDED", "frozen-volume-fixture-v1", "SIP_CONSOLIDATED", start, bars)
	if err != nil {
		t.Fatal(err)
	}
	result, err := CalculateLiquidityVolumeAnomaly(dataset, 1, 3)
	if err != nil || len(result.Values) != 2 || math.Abs(result.Values[0].Value-5000) > 1e-12 || math.Abs(result.Values[1].Value-(500.0/110.0)) > 1e-12 {
		t.Fatalf("liquidity=%+v err=%v", result, err)
	}
}

func TestLiquidityRejectsUnknownSourceAndZeroBaseline(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 101, 102})
	if _, err := CalculateLiquidityVolumeAnomaly(dataset, 1, 1); err == nil {
		t.Fatal("unspecified volume source accepted")
	}
	dataset, err := NewFrozenDatasetWithVolumeSource("liquidity_zero", "TEST", "1D", "AS_PROVIDED", "frozen-volume-zero-v1", "VENUE_SPECIFIC", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []Bar{{At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 0}, {At: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CalculateLiquidityVolumeAnomaly(dataset, 1, 1); err == nil {
		t.Fatal("zero baseline accepted")
	}
}
