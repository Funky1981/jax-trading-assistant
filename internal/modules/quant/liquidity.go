package quant

import "fmt"

const LiquidityAlgorithmV1 = "jax.quant.liquidity_volume_anomaly/v1"

// CalculateLiquidityVolumeAnomaly compares the current window's average
// dollar volume with the immediately preceding baseline window. Volume source
// is part of the frozen dataset and UNKNOWN/UNSPECIFIED sources fail closed.
func CalculateLiquidityVolumeAnomaly(dataset FrozenDataset, window, baselineWindow int) (Response, error) {
	if window < 1 || baselineWindow < 1 {
		return Response{}, fmt.Errorf("liquidity windows must be positive")
	}
	if dataset.VolumeSource == "UNSPECIFIED" || dataset.VolumeSource == "UNKNOWN" {
		return Response{}, fmt.Errorf("volume source provenance is required for liquidity metrics")
	}
	request, err := newCalculationRequest(dataset, nil, "liquidity_volume_anomaly", LiquidityAlgorithmV1, []Parameter{{Name: "baseline_window", Value: fmt.Sprintf("%d", baselineWindow)}, {Name: "current_window", Value: fmt.Sprintf("%d", window)}, {Name: "dollar_volume_definition", Value: "close_times_volume"}, {Name: "volume_source", Value: dataset.VolumeSource}})
	if err != nil {
		return Response{}, err
	}
	if len(dataset.Bars) < window+baselineWindow {
		return Response{}, fmt.Errorf("liquidity requires %d bars", window+baselineWindow)
	}
	currentStart := len(dataset.Bars) - window
	baselineStart := currentStart - baselineWindow
	currentAverage := averageDollarVolume(dataset.Bars[currentStart:])
	baselineAverage := averageDollarVolume(dataset.Bars[baselineStart:currentStart])
	if baselineAverage <= 0 || !finite(baselineAverage) {
		return Response{}, fmt.Errorf("liquidity baseline dollar volume is zero or non-finite")
	}
	ratio := currentAverage / baselineAverage
	if !finite(ratio) {
		return Response{}, fmt.Errorf("liquidity anomaly ratio is non-finite")
	}
	values := []Value{{Metric: "average_dollar_volume", Value: currentAverage, Unit: "currency_volume"}, {Metric: "volume_anomaly_ratio", Value: ratio, Unit: "ratio"}}
	return newCalculationResponse(request, values, nil)
}

func averageDollarVolume(bars []Bar) float64 {
	total := 0.0
	for _, bar := range bars {
		total += bar.Close * bar.Volume
	}
	return total / float64(len(bars))
}
