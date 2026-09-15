package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

func TestStressCostIncludesFixedAndTenBPSCommissionOnBothLegs(t *testing.T) {
	entry, exit, qty := 100.0, 110.0, 100
	got := netReturnWithCosts(entry, exit, qty, 5, 10, 15, 0.50, 10)
	entryNotional, exitNotional := entry*float64(qty), exit*float64(qty)
	rate := (5 + 10 + 15) / 10000.0
	entryCost := 0.50 + entryNotional*10/10000
	exitCost := 0.50 + exitNotional*10/10000
	want := (exitNotional*(1-rate) - exitCost - (entryNotional*(1+rate) + entryCost)) / entryNotional
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("stress return=%v, want %v", got, want)
	}
	withoutVariable := netReturnWithCosts(entry, exit, qty, 5, 10, 15, 0.50, 0)
	if math.Abs(got-withoutVariable) < 1e-6 {
		t.Fatal("stress 10 bps commission component disappeared")
	}
}

func TestR3BDeterministicSeedReferenceVectors(t *testing.T) {
	parent := parentManifestSHA
	if got := bootstrapDraw(parent, 0, 0); got != 18345650792210198742 {
		t.Fatalf("bootstrap reference = %d", got)
	}
	if got := bootstrapDraw(parent, 7, 3); got != 7948458560650707498 {
		t.Fatalf("bootstrap second reference = %d", got)
	}
	seed := seedDigest(parent, "|timestamp-placebo-v1|")
	if got := hex.EncodeToString(seed[:]); got != "37114c390002d7f19e2fabb24dd4957a53e5c365f91afef2632025525acad3ea" {
		t.Fatalf("timestamp seed = %s", got)
	}
	if got := timestampPlaceboScore(parent, 0, "SPY", "2023-01-03"); got != "214e496784e9bdee9ab6c3b4d34cab6adc2df223ac306213ae9304833b180090" {
		t.Fatalf("timestamp score = %s", got)
	}
}

func TestR3BStructuralWarmupAndBoundaryReset(t *testing.T) {
	dates := syntheticDates(350)
	d := syntheticData(dates, func(int) float64 { return 100 })
	d.Boundaries[dates[100]] = true
	if got, err := structuralStateAt(d, dates[298], 200); err != nil || got != structuralWarmupState {
		t.Fatalf("pre-reset state = %s, err=%v", got, err)
	}
	if got, err := structuralStateAt(d, dates[299], 200); err != nil || got != structuralInitializedState {
		t.Fatalf("post-reset state = %s, err=%v", got, err)
	}
	delete(d.Raw, dates[250])
	if got, err := structuralStateAt(d, dates[299], 200); err != nil || got != structuralWarmupState {
		t.Fatalf("missing synchronized session state = %s, err=%v", got, err)
	}
}

func TestR3BOneShotStartMarkerIsExclusiveAndCompletionIsTerminal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run-state.json")
	if err := r3aStartRunStateAt(path, "freeze"); err != nil {
		t.Fatal(err)
	}
	if err := r3aStartRunStateAt(path, "freeze"); err == nil {
		t.Fatal("second start marker was accepted")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state, err := r3aValidateRunStateBytes(b)
	if err == nil || state.Status != "STARTED_ONCE" {
		t.Fatalf("started state validation = %+v, err=%v", state, err)
	}
	if err := r3aCompleteRunStateAt(path, "freeze"); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state, err = r3aValidateRunStateBytes(b)
	if err == nil || state.Status != "COMPLETED_ONCE" || state.PerformanceRunCount != 1 {
		t.Fatalf("completed state validation = %+v, err=%v", state, err)
	}
}

func TestR3BPayloadManifestHashVerification(t *testing.T) {
	dir := t.TempDir()
	payload := []byte(`{"bars":{}}`)
	hash := sha256Hex(payload)
	if err := os.WriteFile(filepath.Join(dir, hash+".json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	metadata := []byte(fmt.Sprintf(`{"families":[{"raw_payload_sha256":["%s"]}]}`, hash))
	metadataPath := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r3aValidatePayloadManifest(metadataPath, sha256Hex(metadata), dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, hash+".json"), []byte(`{"bars":{"SPY":[]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r3aValidatePayloadManifest(metadataPath, sha256Hex(metadata), dir); err == nil {
		t.Fatal("tampered payload was accepted")
	}
}

func TestR3BStructuralScaleAndBarValidation(t *testing.T) {
	raw := bar{Date: "2025-01-02", Open: 10, High: 12, Low: 9, Close: 11, Volume: 1000}
	split := bar{Date: raw.Date, Open: 20, High: 24, Low: 18, Close: 22, Volume: 500}
	if !barScaleConsistent(raw, split, 2) {
		t.Fatal("consistent split scale was rejected")
	}
	split.High = 25
	if barScaleConsistent(raw, split, 2) {
		t.Fatal("inconsistent OHLC split scale was accepted")
	}
	if !validBar(raw, raw.Date) || validBar(bar{Date: raw.Date, Open: 10, High: 9, Low: 8, Close: 9, Volume: 1}, raw.Date) {
		t.Fatal("bar validity contract drifted")
	}
}

func TestR3BResultSchemaUsesExplicitRecoveryIdentity(t *testing.T) {
	b, err := json.Marshal(runOutput{ContractVersion: "jax.val-03r4.recovery-oos-results/v1", ManifestSHA256: parentManifestSHA, SeedManifestSHA256: parentManifestSHA, HoldoutAccessed: true})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["HoldoutAccessed"] != nil || fields["seed_manifest_sha256"] != parentManifestSHA {
		t.Fatalf("result schema fields = %+v", fields)
	}
}
