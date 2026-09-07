package quant

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestPhase05ExitConditionFrozenDatasetIsDeterministicAndReproducible(t *testing.T) {
	dataset := phase05FixtureDataset(t, "phase05-equity", "JAX", []float64{100, 102, 99, 104, 101, 106, 103}, []float64{10, 11, 10, 10, 10, 20, 10}, "SIP_CONSOLIDATED")
	benchmark := phase05FixtureDataset(t, "phase05-benchmark", "BENCH", []float64{100, 101, 98, 102, 100, 103, 101}, []float64{10, 10, 10, 10, 10, 10, 10}, "SIP_CONSOLIDATED")
	runStable := func(name string, calculate func() (any, error)) any {
		first, err := calculate()
		if err != nil {
			t.Fatalf("%s failed: %v", name, err)
		}
		second, err := calculate()
		if err != nil {
			t.Fatalf("%s second run failed: %v", name, err)
		}
		firstJSON, err := json.Marshal(first)
		if err != nil {
			t.Fatalf("%s first marshal failed: %v", name, err)
		}
		secondJSON, err := json.Marshal(second)
		if err != nil {
			t.Fatalf("%s second marshal failed: %v", name, err)
		}
		if !bytes.Equal(firstJSON, secondJSON) {
			t.Fatalf("%s changed between identical frozen runs:\n%s\n%s", name, firstJSON, secondJSON)
		}
		return first
	}

	simple := runStable("simple returns", func() (any, error) { return CalculateSimpleReturns(dataset) }).(Response)
	logReturns := runStable("log returns", func() (any, error) { return CalculateLogReturns(dataset) }).(Response)
	benchmarkRelative := runStable("benchmark relative returns", func() (any, error) { return CalculateBenchmarkRelativeReturns(dataset, benchmark) }).(Response)
	volatility := runStable("volatility", func() (any, error) { return CalculateVolatility(dataset, 3, 252) }).(Response)
	atr := runStable("ATR", func() (any, error) { return CalculateATR(dataset, 3) }).(Response)
	drawdown := runStable("drawdown", func() (any, error) { return CalculateDrawdown(dataset) }).(Response)
	liquidity := runStable("liquidity", func() (any, error) { return CalculateLiquidityVolumeAnomaly(dataset, 2, 2) }).(Response)
	riskAdjusted := runStable("risk-adjusted", func() (any, error) { return CalculateRiskAdjustedMetrics(dataset, 3, 1, 0, 0) }).(Response)
	positionSizing := runStable("position sizing", func() (any, error) { return CalculatePositionSizing(dataset, 100, 95, 10000, 0.01, 25) }).(Response)
	exposure := runStable("portfolio exposure", func() (any, error) {
		return CalculatePortfolioExposure(dataset, []Position{{Instrument: "BBB", Quantity: -2, Price: 50}, {Instrument: "AAA", Quantity: 10, Price: 100}})
	}).(PortfolioExposureResult)

	for _, result := range []Response{simple, logReturns, benchmarkRelative, volatility, atr, drawdown, liquidity, riskAdjusted, positionSizing, exposure.Response} {
		if result.ContractVersion != ResultContractV1 || result.DatasetID != dataset.ID || result.InputSHA256 != dataset.ContentSHA256 || result.AlgorithmVersion == "" {
			t.Fatalf("result lacks frozen-input/version identity: %+v", result)
		}
	}
	if math.Abs(simple.Values[0].Value-0.02) > 1e-15 || math.Abs(logReturns.Values[0].Value-math.Log(1.02)) > 1e-15 {
		t.Fatalf("independent return fixture mismatch: simple=%v log=%v", simple.Values[0].Value, logReturns.Values[0].Value)
	}
	benchmarkFirst := (102.0/100 - 1) - (101.0/100 - 1)
	if math.Abs(benchmarkRelative.Values[0].Value-benchmarkFirst) > 1e-15 {
		t.Fatalf("benchmark-relative=%v expected %v", benchmarkRelative.Values[0].Value, benchmarkFirst)
	}
	if math.Abs(atr.Values[0].Value-11.0/3.0) > 1e-15 {
		t.Fatalf("ATR=%v expected %v", atr.Values[0].Value, 11.0/3.0)
	}
	if math.Abs(drawdown.Values[0].Value-(103.0/106-1)) > 1e-15 || math.Abs(drawdown.Values[1].Value-(99.0/102-1)) > 1e-15 {
		t.Fatalf("drawdown=%+v", drawdown.Values)
	}
	if math.Abs(liquidity.Values[0].Value-1575) > 1e-12 || math.Abs(liquidity.Values[1].Value-1575.0/1025.0) > 1e-12 {
		t.Fatalf("liquidity=%+v", liquidity.Values)
	}
	if positionSizing.Values[3].Value != 20 || exposure.Response.Values[0].Value != 1100 {
		t.Fatalf("bounded risk primitives mismatch: sizing=%+v exposure=%+v", positionSizing.Values, exposure.Response.Values)
	}
	if len(exposure.Instruments) != 2 || exposure.Instruments[0].Instrument != "AAA" || exposure.Instruments[1].Instrument != "BBB" {
		t.Fatalf("exposure breakdown is not canonical: %+v", exposure.Instruments)
	}

	// Recompute the volatility value independently from the documented latest
	// three natural-log returns and sample deviation convention.
	latest := []float64{math.Log(101.0 / 104), math.Log(106.0 / 101), math.Log(103.0 / 106)}
	average := (latest[0] + latest[1] + latest[2]) / 3
	variance := ((latest[0]-average)*(latest[0]-average) + (latest[1]-average)*(latest[1]-average) + (latest[2]-average)*(latest[2]-average)) / 2
	expectedVolatility := math.Sqrt(variance) * math.Sqrt(252)
	if math.Abs(volatility.Values[0].Value-expectedVolatility) > 1e-15 {
		t.Fatalf("volatility=%v expected %v", volatility.Values[0].Value, expectedVolatility)
	}
}

func phase05FixtureDataset(t *testing.T, id, instrument string, closes, volumes []float64, volumeSource string) FrozenDataset {
	t.Helper()
	if len(closes) != len(volumes) {
		t.Fatalf("fixture closes and volumes differ")
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]Bar, len(closes))
	for index := range closes {
		bars[index] = Bar{At: start.AddDate(0, 0, index), Open: closes[index], High: closes[index], Low: closes[index], Close: closes[index], Volume: volumes[index]}
	}
	dataset, err := NewFrozenDatasetWithVolumeSource(id, instrument, "1d", "AS_PROVIDED", "phase05/frozen-fixture", volumeSource, start.AddDate(0, 0, len(closes)), bars)
	if err != nil {
		t.Fatal(err)
	}
	return dataset
}
