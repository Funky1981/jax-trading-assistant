package quant

import (
	"math"
	"testing"
	"time"
)

func TestCorrelationBetaCovarianceKnownValues(t *testing.T) {
	asset := returnsDataset(t, []float64{100, 110, 99, 118.8})
	benchmark := returnsDataset(t, []float64{100, 105, 105, 115.5})
	benchmark.ID = "benchmark-correlation"
	benchmark.ContentSHA256 = benchmark.contentDigest()
	result, err := CalculateCorrelationBetaCovariance(asset, benchmark, 3, false)
	if err != nil {
		t.Fatal(err)
	}
	// Independent hand calculation for returns [0.10,-0.10,0.20] and
	// [0.05,0,0.10]: sample covariance=.0075, benchmark variance=.0025, beta=3.
	if math.Abs(result.Values[0].Value-0.0075) > 1e-12 || math.Abs(result.Values[2].Value-3) > 1e-12 || result.AlgorithmVersion != CorrelationAlgorithmV1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCorrelationRejectsMisalignmentAndZeroBenchmarkVariance(t *testing.T) {
	asset := returnsDataset(t, []float64{100, 110, 99})
	benchmark := returnsDataset(t, []float64{100, 105, 105})
	benchmark.ID = "benchmark-correlation-invalid"
	benchmark.Bars[1].At = benchmark.Bars[1].At.Add(time.Hour)
	benchmark.ContentSHA256 = benchmark.contentDigest()
	if _, err := CalculateCorrelationBetaCovariance(asset, benchmark, 2, false); err == nil {
		t.Fatal("misaligned pair accepted")
	}
	benchmark = returnsDataset(t, []float64{100, 100, 100})
	benchmark.ID = "benchmark-zero-variance"
	benchmark.ContentSHA256 = benchmark.contentDigest()
	asset = returnsDataset(t, []float64{100, 110, 90})
	if _, err := CalculateCorrelationBetaCovariance(asset, benchmark, 2, false); err == nil {
		t.Fatal("zero benchmark variance accepted")
	}
}
