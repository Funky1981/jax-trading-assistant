package quant

import (
	"math"
	"testing"
	"time"
)

func returnsDataset(t *testing.T, closes []float64) FrozenDataset {
	t.Helper()
	bars := make([]Bar, len(closes))
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for index, close := range closes {
		bars[index] = Bar{At: start.AddDate(0, 0, index), Open: close, High: close, Low: close, Close: close, Volume: 1}
	}
	dataset, err := NewFrozenDataset("returns_"+string(rune('a'+len(closes))), "TEST", "1D", "AS_PROVIDED", "frozen-returns-fixture-v1", start, bars)
	if err != nil {
		t.Fatal(err)
	}
	return dataset
}

func TestReturnsKnownValuesAndVersions(t *testing.T) {
	dataset := returnsDataset(t, []float64{100, 110, 99})
	simple, err := CalculateSimpleReturns(dataset)
	if err != nil || len(simple.Values) != 2 || math.Abs(simple.Values[0].Value-0.1) > 1e-12 || math.Abs(simple.Values[1].Value-(-0.1)) > 1e-12 {
		t.Fatalf("simple=%+v err=%v", simple, err)
	}
	logReturns, err := CalculateLogReturns(dataset)
	if err != nil || math.Abs(logReturns.Values[0].Value-math.Log(1.1)) > 1e-12 || math.Abs(logReturns.Values[1].Value-math.Log(0.9)) > 1e-12 || logReturns.AlgorithmVersion != ReturnsAlgorithmV1 {
		t.Fatalf("log=%+v err=%v", logReturns, err)
	}
}

func TestBenchmarkRelativeReturnsUsesExactTimestampAlignment(t *testing.T) {
	asset := returnsDataset(t, []float64{100, 110, 99})
	benchmark := returnsDataset(t, []float64{100, 105, 105})
	benchmark.ID = "benchmark"
	benchmark.ContentSHA256 = benchmark.contentDigest()
	result, err := CalculateBenchmarkRelativeReturns(asset, benchmark)
	if err != nil || len(result.Values) != 2 || math.Abs(result.Values[0].Value-0.05) > 1e-12 || math.Abs(result.Values[1].Value-(-0.1)) > 1e-12 || len(result.InputReferences) != 1 {
		t.Fatalf("relative=%+v err=%v", result, err)
	}
	benchmark.Bars[1].At = benchmark.Bars[1].At.Add(time.Hour)
	benchmark.ContentSHA256 = benchmark.contentDigest()
	if _, err := CalculateBenchmarkRelativeReturns(asset, benchmark); err == nil {
		t.Fatal("misaligned benchmark accepted")
	}
}

func TestReturnsRejectInsufficientHistory(t *testing.T) {
	if _, err := CalculateSimpleReturns(returnsDataset(t, []float64{100})); err == nil {
		t.Fatal("one-bar simple return accepted")
	}
}
