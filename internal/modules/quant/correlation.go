package quant

import (
	"fmt"
	"math"
)

const CorrelationAlgorithmV1 = "jax.quant.correlation_beta_covariance/v1"

// CalculateCorrelationBetaCovariance uses aligned close-to-close returns,
// sample covariance/variance (n-1), Pearson correlation, and beta=cov(asset,
// benchmark)/var(benchmark). The window selects the most recent aligned
// returns; no missing timestamp is filled with zero.
func CalculateCorrelationBetaCovariance(asset, benchmark FrozenDataset, window int, logarithmic bool) (Response, error) {
	return calculateAlignedPair(asset, benchmark, window, logarithmic, "correlation_beta_covariance")
}

func calculateAlignedPair(asset, benchmark FrozenDataset, window int, logarithmic bool, algorithm string) (Response, error) {
	if window < 2 {
		return Response{}, fmt.Errorf("correlation/beta/covariance window must be at least two")
	}
	if asset.Frequency != benchmark.Frequency {
		return Response{}, fmt.Errorf("asset and benchmark frequencies must match")
	}
	returnType := "simple"
	if logarithmic {
		returnType = "natural_log"
	}
	parameters := []Parameter{{Name: "covariance_convention", Value: "sample_n_minus_1"}, {Name: "return_type", Value: returnType}, {Name: "window", Value: fmt.Sprintf("%d", window)}}
	request, err := newCalculationRequest(asset, []InputReference{{ID: benchmark.ID, ContentSHA256: benchmark.ContentSHA256}}, algorithm, CorrelationAlgorithmV1, parameters)
	if err != nil {
		return Response{}, err
	}
	if err := benchmark.Validate(); err != nil {
		return Response{}, fmt.Errorf("benchmark dataset: %w", err)
	}
	assetReturns, benchmarkReturns, err := alignedReturns(asset, benchmark, logarithmic)
	if err != nil {
		return Response{}, err
	}
	if len(assetReturns) < window {
		return Response{}, fmt.Errorf("aligned return history has %d observations; %d required", len(assetReturns), window)
	}
	assetReturns = assetReturns[len(assetReturns)-window:]
	benchmarkReturns = benchmarkReturns[len(benchmarkReturns)-window:]
	assetMean, benchmarkMean := mean(assetReturns), mean(benchmarkReturns)
	covariance, assetVariance, benchmarkVariance := 0.0, 0.0, 0.0
	for index := range assetReturns {
		assetDelta, benchmarkDelta := assetReturns[index]-assetMean, benchmarkReturns[index]-benchmarkMean
		covariance += assetDelta * benchmarkDelta
		assetVariance += assetDelta * assetDelta
		benchmarkVariance += benchmarkDelta * benchmarkDelta
	}
	denominator := float64(window - 1)
	covariance /= denominator
	assetVariance /= denominator
	benchmarkVariance /= denominator
	if benchmarkVariance == 0 || !finite(benchmarkVariance) {
		return Response{}, fmt.Errorf("benchmark variance is zero or non-finite")
	}
	correlationDenominator := math.Sqrt(assetVariance * benchmarkVariance)
	if correlationDenominator == 0 || !finite(correlationDenominator) {
		return Response{}, fmt.Errorf("correlation is undefined for zero or non-finite variance")
	}
	correlation := covariance / correlationDenominator
	beta := covariance / benchmarkVariance
	values := []Value{{Metric: "sample_covariance", Value: covariance, Unit: "decimal_squared"}, {Metric: "pearson_correlation", Value: correlation, Unit: "coefficient"}, {Metric: "beta", Value: beta, Unit: "coefficient"}}
	if !finite(correlation) || !finite(beta) {
		return Response{}, fmt.Errorf("correlation or beta produced a non-finite result")
	}
	return newCalculationResponse(request, values, nil)
}

func alignedReturns(asset, benchmark FrozenDataset, logarithmic bool) ([]float64, []float64, error) {
	if len(asset.Bars) != len(benchmark.Bars) || len(asset.Bars) < 2 {
		return nil, nil, fmt.Errorf("asset and benchmark require equal aligned histories of at least two bars")
	}
	assetReturns, benchmarkReturns := make([]float64, 0, len(asset.Bars)-1), make([]float64, 0, len(asset.Bars)-1)
	for index := 1; index < len(asset.Bars); index++ {
		if !asset.Bars[index].At.Equal(benchmark.Bars[index].At) || !asset.Bars[index-1].At.Equal(benchmark.Bars[index-1].At) {
			return nil, nil, fmt.Errorf("asset and benchmark timestamps do not align at index %d", index)
		}
		assetRatio := asset.Bars[index].Close / asset.Bars[index-1].Close
		benchmarkRatio := benchmark.Bars[index].Close / benchmark.Bars[index-1].Close
		if logarithmic {
			assetReturns = append(assetReturns, math.Log(assetRatio))
			benchmarkReturns = append(benchmarkReturns, math.Log(benchmarkRatio))
		} else {
			assetReturns = append(assetReturns, assetRatio-1)
			benchmarkReturns = append(benchmarkReturns, benchmarkRatio-1)
		}
	}
	return assetReturns, benchmarkReturns, nil
}

func mean(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
