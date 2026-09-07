package quant

import "fmt"

const ReturnsAlgorithmV1 = "jax.quant.returns/v1"

func CalculateSimpleReturns(dataset FrozenDataset) (Response, error) {
	request, err := newCalculationRequest(dataset, nil, "simple_returns", ReturnsAlgorithmV1, []Parameter{{Name: "price_field", Value: "close"}, {Name: "return_type", Value: "simple"}})
	if err != nil {
		return Response{}, err
	}
	values, err := valuesForReturns(dataset, "simple_return", false)
	if err != nil {
		return Response{}, err
	}
	if err := ensureFiniteValues(values); err != nil {
		return Response{}, err
	}
	return newCalculationResponse(request, values, nil)
}

func CalculateLogReturns(dataset FrozenDataset) (Response, error) {
	request, err := newCalculationRequest(dataset, nil, "log_returns", ReturnsAlgorithmV1, []Parameter{{Name: "price_field", Value: "close"}, {Name: "return_type", Value: "natural_log"}})
	if err != nil {
		return Response{}, err
	}
	values, err := valuesForReturns(dataset, "log_return", true)
	if err != nil {
		return Response{}, err
	}
	if err := ensureFiniteValues(values); err != nil {
		return Response{}, err
	}
	return newCalculationResponse(request, values, nil)
}

func CalculateBenchmarkRelativeReturns(asset, benchmark FrozenDataset) (Response, error) {
	if asset.Frequency != benchmark.Frequency {
		return Response{}, fmt.Errorf("asset and benchmark frequencies must match")
	}
	input := []InputReference{{ID: benchmark.ID, ContentSHA256: benchmark.ContentSHA256}}
	request, err := newCalculationRequest(asset, input, "benchmark_relative_returns", ReturnsAlgorithmV1, []Parameter{{Name: "benchmark_return_definition", Value: "asset_simple_minus_benchmark_simple"}, {Name: "price_field", Value: "close"}})
	if err != nil {
		return Response{}, err
	}
	if err := benchmark.Validate(); err != nil {
		return Response{}, fmt.Errorf("benchmark dataset: %w", err)
	}
	if len(asset.Bars) < 2 || len(benchmark.Bars) < 2 {
		return Response{}, fmt.Errorf("benchmark-relative returns require at least two bars per dataset")
	}
	if len(asset.Bars) != len(benchmark.Bars) {
		return Response{}, fmt.Errorf("asset and benchmark bars must align exactly")
	}
	values := make([]Value, 0, len(asset.Bars)-1)
	for index := 1; index < len(asset.Bars); index++ {
		if !asset.Bars[index].At.Equal(benchmark.Bars[index].At) || !asset.Bars[index-1].At.Equal(benchmark.Bars[index-1].At) {
			return Response{}, fmt.Errorf("asset and benchmark timestamps do not align at index %d", index)
		}
		assetReturn := asset.Bars[index].Close/asset.Bars[index-1].Close - 1
		benchmarkReturn := benchmark.Bars[index].Close/benchmark.Bars[index-1].Close - 1
		at := asset.Bars[index].At
		values = append(values, Value{Metric: "benchmark_relative_simple_return", Value: assetReturn - benchmarkReturn, Unit: "decimal_return", ObservedAt: &at})
	}
	if err := ensureFiniteValues(values); err != nil {
		return Response{}, err
	}
	return newCalculationResponse(request, values, nil)
}
