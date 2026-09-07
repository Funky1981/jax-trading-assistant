package quant

import (
	"fmt"
	"math"
)

const VolatilityAlgorithmV1 = "jax.quant.volatility_atr_drawdown/v1"

// CalculateVolatility uses natural-log close-to-close returns, the most recent
// window of return observations, sample standard deviation (n-1), and the
// caller-supplied annualization factor. No return is annualized implicitly.
func CalculateVolatility(dataset FrozenDataset, window int, annualizationFactor float64) (Response, error) {
	if window < 2 || !finitePositive(annualizationFactor) {
		return Response{}, fmt.Errorf("volatility requires window >= 2 and a finite positive annualization factor")
	}
	request, err := newCalculationRequest(dataset, nil, "volatility", VolatilityAlgorithmV1, []Parameter{{Name: "annualization_factor", Value: formatFloat(annualizationFactor)}, {Name: "return_type", Value: "natural_log"}, {Name: "sample_convention", Value: "n_minus_1"}, {Name: "window", Value: fmt.Sprintf("%d", window)}})
	if err != nil {
		return Response{}, err
	}
	if len(dataset.Bars)-1 < window {
		return Response{}, fmt.Errorf("volatility requires %d return observations", window)
	}
	start := len(dataset.Bars) - window
	returns := make([]float64, 0, window)
	for index := start; index < len(dataset.Bars); index++ {
		returns = append(returns, math.Log(dataset.Bars[index].Close/dataset.Bars[index-1].Close))
	}
	mean := 0.0
	for _, value := range returns {
		mean += value
	}
	mean /= float64(len(returns))
	variance := 0.0
	for _, value := range returns {
		delta := value - mean
		variance += delta * delta
	}
	volatility := math.Sqrt(variance/float64(len(returns)-1)) * math.Sqrt(annualizationFactor)
	if !finite(volatility) {
		return Response{}, fmt.Errorf("volatility produced a non-finite result")
	}
	return newCalculationResponse(request, []Value{{Metric: "annualized_log_return_volatility", Value: volatility, Unit: "decimal"}}, nil)
}

// CalculateATR uses simple average true range over the most recent period.
// Each true range is max(high-low, abs(high-prevClose), abs(low-prevClose));
// the first bar is excluded because it has no prior close.
func CalculateATR(dataset FrozenDataset, period int) (Response, error) {
	if period < 1 {
		return Response{}, fmt.Errorf("ATR period must be positive")
	}
	request, err := newCalculationRequest(dataset, nil, "atr", VolatilityAlgorithmV1, []Parameter{{Name: "period", Value: fmt.Sprintf("%d", period)}, {Name: "true_range", Value: "max_high_low_prev_close"}, {Name: "smoothing", Value: "simple_average"}})
	if err != nil {
		return Response{}, err
	}
	if len(dataset.Bars)-1 < period {
		return Response{}, fmt.Errorf("ATR requires %d prior-close true-range observations", period)
	}
	start := len(dataset.Bars) - period
	sum := 0.0
	for index := start; index < len(dataset.Bars); index++ {
		bar, previous := dataset.Bars[index], dataset.Bars[index-1].Close
		trueRange := math.Max(bar.High-bar.Low, math.Max(math.Abs(bar.High-previous), math.Abs(bar.Low-previous)))
		sum += trueRange
	}
	atr := sum / float64(period)
	if !finite(atr) {
		return Response{}, fmt.Errorf("ATR produced a non-finite result")
	}
	return newCalculationResponse(request, []Value{{Metric: "atr", Value: atr, Unit: "price_units"}}, nil)
}

// CalculateDrawdown defines drawdown as close/running-peak - 1. Maximum
// drawdown is the most negative observed value and current drawdown is the
// final bar's drawdown. Peaks start at the first supplied close.
func CalculateDrawdown(dataset FrozenDataset) (Response, error) {
	request, err := newCalculationRequest(dataset, nil, "drawdown", VolatilityAlgorithmV1, []Parameter{{Name: "price_field", Value: "close"}, {Name: "definition", Value: "close_divided_by_running_peak_minus_one"}})
	if err != nil {
		return Response{}, err
	}
	peak := dataset.Bars[0].Close
	maximum := 0.0
	current := 0.0
	for _, bar := range dataset.Bars {
		if bar.Close > peak {
			peak = bar.Close
		}
		current = bar.Close/peak - 1
		if current < maximum {
			maximum = current
		}
	}
	values := []Value{{Metric: "current_drawdown", Value: current, Unit: "decimal"}, {Metric: "maximum_drawdown", Value: maximum, Unit: "decimal"}}
	return newCalculationResponse(request, values, nil)
}
