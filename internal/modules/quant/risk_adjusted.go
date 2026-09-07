package quant

import (
	"fmt"
	"math"
)

const RiskAdjustedAlgorithmV1 = "jax.quant.risk_adjusted_metrics/v1"

// CalculateRiskAdjustedMetrics uses simple close returns from the most recent
// window. Annual risk-free and downside-target rates are converted to periodic
// rates by compounding with the explicit annualization factor. Sharpe uses
// sample excess-return deviation; Sortino uses sqrt(mean squared negative
// target shortfall); Calmar uses annualized compounded return over the full
// dataset divided by absolute maximum drawdown.
func CalculateRiskAdjustedMetrics(dataset FrozenDataset, window int, annualizationFactor, annualRiskFree, annualDownsideTarget float64) (Response, error) {
	if window < 2 || !finitePositive(annualizationFactor) || !finite(annualRiskFree) || !finite(annualDownsideTarget) || annualRiskFree <= -1 || annualDownsideTarget <= -1 {
		return Response{}, fmt.Errorf("risk-adjusted metrics require valid window, annualization, risk-free, and downside-target inputs")
	}
	request, err := newCalculationRequest(dataset, nil, "risk_adjusted_metrics", RiskAdjustedAlgorithmV1, []Parameter{{Name: "annual_downside_target", Value: formatFloat(annualDownsideTarget)}, {Name: "annual_risk_free", Value: formatFloat(annualRiskFree)}, {Name: "annualization_factor", Value: formatFloat(annualizationFactor)}, {Name: "return_type", Value: "simple"}, {Name: "sharpe_deviation", Value: "sample_n_minus_1"}, {Name: "sortino_downside_deviation", Value: "sqrt_mean_squared_negative_shortfall"}, {Name: "window", Value: fmt.Sprintf("%d", window)}})
	if err != nil {
		return Response{}, err
	}
	if len(dataset.Bars)-1 < window {
		return Response{}, fmt.Errorf("risk-adjusted metrics require %d return observations", window)
	}
	returns := make([]float64, window)
	start := len(dataset.Bars) - 1 - window
	for index := 0; index < window; index++ {
		returns[index] = dataset.Bars[start+index+1].Close/dataset.Bars[start+index].Close - 1
	}
	periodicRiskFree := math.Pow(1+annualRiskFree, 1/annualizationFactor) - 1
	periodicTarget := math.Pow(1+annualDownsideTarget, 1/annualizationFactor) - 1
	excess := make([]float64, window)
	for index, value := range returns {
		excess[index] = value - periodicRiskFree
	}
	excessMean := mean(excess)
	variance := 0.0
	downsideSquares := 0.0
	for index, value := range excess {
		delta := value - excessMean
		variance += delta * delta
		shortfall := returns[index] - periodicTarget
		if shortfall < 0 {
			downsideSquares += shortfall * shortfall
		}
	}
	if variance == 0 || !finite(variance) {
		return Response{}, fmt.Errorf("Sharpe is undefined for zero or non-finite excess-return variance")
	}
	standardDeviation := math.Sqrt(variance / float64(window-1))
	downsideDeviation := math.Sqrt(downsideSquares / float64(window))
	sharpe := excessMean / standardDeviation * math.Sqrt(annualizationFactor)
	if downsideDeviation == 0 || !finite(downsideDeviation) {
		return Response{}, fmt.Errorf("Sortino is undefined for zero downside deviation")
	}
	maximumDrawdown := maximumDrawdown(dataset.Bars)
	if maximumDrawdown == 0 || !finite(maximumDrawdown) {
		return Response{}, fmt.Errorf("Calmar is undefined without non-zero maximum drawdown")
	}
	periods := len(dataset.Bars) - 1
	annualizedReturn := math.Pow(dataset.Bars[len(dataset.Bars)-1].Close/dataset.Bars[0].Close, annualizationFactor/float64(periods)) - 1
	sortinoNumerator := mean(returns) - periodicTarget
	sortino := sortinoNumerator / downsideDeviation * math.Sqrt(annualizationFactor)
	calmar := annualizedReturn / math.Abs(maximumDrawdown)
	values := []Value{{Metric: "sharpe", Value: sharpe, Unit: "ratio"}, {Metric: "sortino", Value: sortino, Unit: "ratio"}, {Metric: "calmar", Value: calmar, Unit: "ratio"}}
	if !finite(sharpe) || !finite(sortino) || !finite(calmar) {
		return Response{}, fmt.Errorf("risk-adjusted metric produced a non-finite result")
	}
	return newCalculationResponse(request, values, nil)
}

func maximumDrawdown(bars []Bar) float64 {
	peak, result := bars[0].Close, 0.0
	for _, bar := range bars {
		if bar.Close > peak {
			peak = bar.Close
		}
		value := bar.Close/peak - 1
		if value < result {
			result = value
		}
	}
	return result
}
