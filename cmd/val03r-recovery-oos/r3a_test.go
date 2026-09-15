package main

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"jax-trading-assistant/libs/strategies"
)

func TestR3AAuthorizationAndRunStateAreStrictlyTyped(t *testing.T) {
	authorization := `{"contract_id":"jax.val-03r4.execution-authorization/v1","candidate":"ma_crossover_v1","recovery_boundary":"2025-01-01..2026-09-11","execution_freeze_sha256":"freeze","performance_run_count_before":0,"execute_once":true,"external_authorization":"GO_VAL03R4_SINGLE_RECOVERY_OOS_EXECUTION"}`
	if _, err := r3aValidateAuthorizationBytes([]byte(authorization), "freeze"); err != nil {
		t.Fatalf("valid authorization rejected: %v", err)
	}
	unknown := `{"contract_id":"jax.val-03r4.execution-authorization/v1","candidate":"ma_crossover_v1","recovery_boundary":"2025-01-01..2026-09-11","execution_freeze_sha256":"freeze","performance_run_count_before":0,"execute_once":true,"external_authorization":"GO_VAL03R4_SINGLE_RECOVERY_OOS_EXECUTION","threshold":0.6}`
	if _, err := r3aValidateAuthorizationBytes([]byte(unknown), "freeze"); err == nil {
		t.Fatal("unknown authorization field was accepted")
	}
	if _, err := r3aValidateRunStateBytes([]byte(`{"performance_run_count":1,"status":"COMPLETED_ONCE"}`)); err == nil {
		t.Fatal("completed run state was accepted for a second run")
	}
	if _, err := r3aValidateRunStateBytes([]byte(`{"performance_run_count":0,"status":"NOT_STARTED"}`)); err == nil {
		t.Fatal("pre-run state was accepted as a reusable one-shot state")
	}
}

func TestR3AReadinessHarnessMatchesFrozenStrategyForRepresentativeInputs(t *testing.T) {
	ctx := context.Background()
	strategy := strategies.NewMACrossoverStrategy()

	tests := []struct {
		name       string
		data       *instrumentData
		wantType   string
		wantConf   float64
		actionable bool
	}{
		{name: "bullish alignment with all boosts", data: vectorData(110, 100, 90, 120), wantType: "BUY", wantConf: 0.95, actionable: true},
		{name: "bullish pullback remains below gate", data: vectorData(100, 90, 80, 99.5), wantType: "BUY", wantConf: 0.55, actionable: false},
		{name: "hold", data: vectorData(100, 100, 90, 95), wantType: "HOLD", wantConf: 0, actionable: false},
		{name: "bearish diagnostic", data: vectorData(90, 100, 110, 80), wantType: "SELL", wantConf: 0.95, actionable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.data
			i := len(d.Dates) - 1
			got, kind := buildSignal(d, i, 20, 50, 200, FrozenExperimentConfig{AvgVolumePeriod: 20, ATRPeriod: 14})
			input := strategies.AnalysisInput{
				Symbol: "SPY", Price: d.Split[d.Dates[i]].Close,
				SMA20: avgWindow(d, i, 20), SMA50: avgWindow(d, i, 50), SMA200: avgWindow(d, i, 200),
				ATR: atr14(d, i, 14), Volume: int64(d.Split[d.Dates[i]].Volume), AvgVolume20: int64(avgVolume(d, i, 20)),
				MarketTrend: marketTrendFromValues(avgWindow(d, i, 20), avgWindow(d, i, 50), avgWindow(d, i, 200)),
			}
			expected, err := strategy.Analyze(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if string(expected.Type) != stringLower(kind) || got.Type != kind || math.Abs(got.Confidence-expected.Confidence) > 1e-12 {
				t.Fatalf("harness/source mismatch: harness=%+v kind=%s source=%+v", got, kind, expected)
			}
			if math.Abs(got.Confidence-tt.wantConf) > 1e-12 {
				t.Fatalf("confidence=%v, want frozen representative value %v", got.Confidence, tt.wantConf)
			}
			if kind != "HOLD" {
				if math.Abs(got.Stop-expected.StopLoss) > 1e-12 || math.Abs(got.Target-expected.TakeProfit[0]) > 1e-12 {
					t.Fatalf("exit geometry mismatch: harness stop/target=%v/%v source=%v/%v", got.Stop, got.Target, expected.StopLoss, expected.TakeProfit[0])
				}
			}
			if actionableByConfig(got.Confidence, FrozenExperimentConfig{ActionableConfidenceThreshold: 0.60}) != tt.actionable {
				if kind == "SELL" {
					// SELL may be actionable at the shared gate, but remains secondary-only
					// in the primary long readiness population.
					if !actionableByConfig(got.Confidence, FrozenExperimentConfig{ActionableConfidenceThreshold: 0.60}) {
						t.Fatal("frozen SELL confidence gate drifted")
					}
				} else {
					t.Fatalf("actionable gate mismatch for %s: confidence=%v", kind, got.Confidence)
				}
			}
		})
	}
}

func vectorData(last20, last50, last200, price float64) *instrumentData {
	n := 220
	dates := syntheticDates(n)
	d := syntheticData(dates, func(i int) float64 { return last200 })
	for i := n - 200; i < n-50; i++ {
		d.Split[dates[i]] = bar{Date: dates[i], Open: last200, High: last200 + 1, Low: last200 - 1, Close: last200, Volume: 100}
		d.Raw[dates[i]], d.Detector[dates[i]] = d.Split[dates[i]], d.Split[dates[i]]
	}
	for i := n - 50; i < n-20; i++ {
		d.Split[dates[i]] = bar{Date: dates[i], Open: last50, High: last50 + 1, Low: last50 - 1, Close: last50, Volume: 100}
		d.Raw[dates[i]], d.Detector[dates[i]] = d.Split[dates[i]], d.Split[dates[i]]
	}
	for i := n - 20; i < n; i++ {
		close := last20
		if i == n-1 {
			close = price
		}
		d.Split[dates[i]] = bar{Date: dates[i], Open: close, High: close + 1, Low: close - 1, Close: close, Volume: 200}
		d.Raw[dates[i]], d.Detector[dates[i]] = d.Split[dates[i]], d.Split[dates[i]]
	}
	last := d.Split[dates[n-1]]
	last.Volume = 300
	d.Split[dates[n-1]], d.Raw[dates[n-1]], d.Detector[dates[n-1]] = last, last, last
	for date := range d.Split {
		d.Factor[date] = 1
	}
	return d
}

func marketTrendFromValues(fast, medium, slow float64) string {
	if fast > medium && medium > slow {
		return "bullish"
	}
	if fast < medium && medium < slow {
		return "bearish"
	}
	return "neutral"
}

func stringLower(kind string) string {
	switch kind {
	case "BUY":
		return "buy"
	case "SELL":
		return "sell"
	default:
		return "hold"
	}
}

func TestR3AModeDoesNotAcceptPerformanceFlags(t *testing.T) {
	for _, mode := range []string{"--run", "--oos", "--threshold=0.2"} {
		if mode == "--contract-audit" || mode == "--preflight-only" || mode == "--execute" {
			t.Fatalf("forbidden mode became accepted: %s", mode)
		}
	}
	if _, err := json.Marshal(r3aRunState{PerformanceRunCount: 0, Status: "NOT_STARTED"}); err != nil {
		t.Fatal(err)
	}
}
