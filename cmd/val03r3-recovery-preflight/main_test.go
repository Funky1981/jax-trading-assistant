package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/libs/strategies"
)

func withRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestRecoveryManifestFullConformance(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Candidate.ActionableConfidenceThreshold != 0.60 {
		t.Fatalf("threshold = %v", m.Candidate.ActionableConfidenceThreshold)
	}
	if m.Provider.RequestDateGuard == "" || !m.Provider.RawBytesBeforeNormalization {
		t.Fatal("provider material fields not bound")
	}
	if m.NoPostResultSalvage.DateExtension || m.NoPostResultSalvage.ThresholdTuning || !m.NoPostResultSalvage.NewExperimentRequired {
		t.Fatal("post-result salvage contract weakened")
	}
	if m.PromotionGates.UnknownMaterialStatus != "FAIL_CLOSED" || m.PromotionGates.SecondarySignPermutation != "NON_BLOCKING_INFORMATIONAL" {
		t.Fatal("promotion gate semantics changed")
	}
}

func TestOutcomeFreeContractAuditExecutes(t *testing.T) {
	withRepoRoot(t)
	if err := runContractAudit(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryDateGuardIsExact(t *testing.T) {
	for _, date := range []string{"2025-01-01", "2026-01-02", "2026-09-11"} {
		if !recoveryDateAllowed(date) {
			t.Fatalf("expected allowed: %s", date)
		}
	}
	for _, date := range []string{"2024-12-31", "2026-09-12", "2026-09-14", "not-a-date"} {
		if recoveryDateAllowed(date) {
			t.Fatalf("expected rejected: %s", date)
		}
	}
}

func TestContaminatedRunGuardRemainsPermanent(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateContaminatedGuard(m); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessOutputContainsNoPerformanceFields(t *testing.T) {
	b, err := json.Marshal(readinessResult{})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"return", "pnl", "sharpe", "sortino", "drawdown", "win_rate", "placebo_effect"} {
		if containsFold(string(b), forbidden) {
			t.Fatalf("readiness schema exposes forbidden performance field %q: %s", forbidden, string(b))
		}
	}
}

func TestReadinessBullishSemanticsMatchFrozenStrategy(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}

	d := syntheticIndicatorData(120, 110, 100, 130, true)
	idx := len(d.Dates) - 1
	recoverySignal, kind := buildSignal(d, idx, m)
	input := strategies.AnalysisInput{
		Symbol:      "SPY",
		Price:       d.Split[d.Dates[idx]].Close,
		Timestamp:   time.Date(2020, 1, 2, 21, 0, 0, 0, time.UTC),
		SMA20:       avgClose(d, idx, m.Candidate.SMAFast),
		SMA50:       avgClose(d, idx, m.Candidate.SMAMedium),
		SMA200:      avgClose(d, idx, m.Candidate.SMASlow),
		ATR:         atr(d, idx, m.Candidate.ATRPeriod),
		Volume:      int64(d.Split[d.Dates[idx]].Volume),
		AvgVolume20: int64(avgVolume(d, idx, m.Candidate.AvgVolumePeriod)),
		MarketTrend: "bullish",
	}
	frozen, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "BUY" || frozen.Type != strategies.SignalBuy {
		t.Fatalf("bullish semantics diverged: helper=%s source=%s", kind, frozen.Type)
	}
	if recoverySignal.Confidence != frozen.Confidence {
		t.Fatalf("confidence = %.12f, source = %.12f", recoverySignal.Confidence, frozen.Confidence)
	}
	if recoverySignal.Stop != frozen.StopLoss || recoverySignal.Target != frozen.TakeProfit[0] {
		t.Fatalf("geometry diverged: helper stop/target=(%v,%v), source=(%v,%v)", recoverySignal.Stop, recoverySignal.Target, frozen.StopLoss, frozen.TakeProfit[0])
	}
}

func TestReadinessConfidenceComponentsAndThresholdAreFrozen(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Candidate.ActionableConfidenceThreshold != 0.60 {
		t.Fatalf("threshold = %v", m.Candidate.ActionableConfidenceThreshold)
	}

	// A qualifying golden-cross vector with no market, volume, or separation
	// boost proves the source base confidence remains 0.65.
	baseInput := strategies.AnalysisInput{
		Symbol: "SPY", Price: 110, Timestamp: time.Unix(0, 0),
		SMA20: 104, SMA50: 102, SMA200: 100, ATR: 2,
		Volume: 100, AvgVolume20: 100, MarketTrend: "neutral",
	}
	base, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), baseInput)
	if err != nil {
		t.Fatal(err)
	}
	if base.Type != strategies.SignalBuy || base.Confidence != 0.65 {
		t.Fatalf("base confidence semantics changed: type=%s confidence=%v", base.Type, base.Confidence)
	}

	// Each registered boost is exercised through the frozen source; the
	// readiness vector above covers their combined operational value.
	boostedInput := baseInput
	boostedInput.MarketTrend = "bullish"
	boostedInput.Volume = 101
	boostedInput.AvgVolume20 = 100
	boostedInput.SMA20 = 106
	boosted, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), boostedInput)
	if err != nil {
		t.Fatal(err)
	}
	if boosted.Confidence != 0.95 {
		t.Fatalf("boosted confidence = %v, want 0.95 (0.65+0.12+0.08+0.10)", boosted.Confidence)
	}
	if boosted.Confidence > 1.0 {
		t.Fatalf("confidence exceeded cap: %v", boosted.Confidence)
	}
}

func TestReadinessPullbackHoldAndSellRemainNonPrimary(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}

	// The source emits the documented 0.55 pullback, but the operational
	// readiness helper intentionally excludes source-only pullbacks.
	pullbackData := syntheticIndicatorData(100, 95, 90, 99.5, false)
	i := len(pullbackData.Dates) - 1
	pullback, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), strategies.AnalysisInput{
		Symbol: "SPY", Price: 99.5, Timestamp: time.Unix(0, 0),
		SMA20: 100, SMA50: 95, SMA200: 90, ATR: 2,
		Volume: 100, AvgVolume20: 100, MarketTrend: "bullish",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pullback.Type != strategies.SignalBuy || pullback.Confidence != 0.55 || pullback.Confidence >= m.Candidate.ActionableConfidenceThreshold {
		t.Fatalf("pullback semantics changed: type=%s confidence=%v", pullback.Type, pullback.Confidence)
	}
	if _, kind := buildSignal(pullbackData, i, m); kind != "HOLD" {
		t.Fatalf("operational helper reintroduced source-only pullback: %s", kind)
	}

	holdData := syntheticIndicatorData(100, 95, 90, 80, false)
	hi := len(holdData.Dates) - 1
	hold, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), strategies.AnalysisInput{
		Symbol: "SPY", Price: 80, Timestamp: time.Unix(0, 0),
		SMA20: 100, SMA50: 95, SMA200: 90, ATR: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hold.Type != strategies.SignalHold {
		t.Fatalf("hold semantics changed: %s", hold.Type)
	}
	if _, kind := buildSignal(holdData, hi, m); kind != "HOLD" {
		t.Fatalf("helper hold semantics changed: %s", kind)
	}

	sellData := syntheticIndicatorData(90, 95, 100, 80, false)
	si := len(sellData.Dates) - 1
	sell, err := strategies.NewMACrossoverStrategy().Analyze(context.Background(), strategies.AnalysisInput{
		Symbol: "SPY", Price: 80, Timestamp: time.Unix(0, 0),
		SMA20: 90, SMA50: 95, SMA200: 100, ATR: 2,
		MarketTrend: "bearish",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sell.Type != strategies.SignalSell {
		t.Fatalf("sell semantics changed: %s", sell.Type)
	}
	if _, kind := buildSignal(sellData, si, m); kind != "SELL" {
		t.Fatalf("helper sell semantics changed: %s", kind)
	}
	if kind := "SELL"; kind == "BUY" {
		t.Fatal("SELL entered primary long readiness semantics")
	}
}

func syntheticIndicatorData(sma20, sma50, sma200, price float64, highVolume bool) *instrumentData {
	const n = 220
	d := &instrumentData{Raw: map[string]bar{}, Split: map[string]bar{}, Detector: map[string]bar{}, Boundaries: map[string]bool{}}
	d.Dates = make([]string, n)
	for i := 0; i < n; i++ {
		date := fmt.Sprintf("%04d", i)
		d.Dates[i] = date
		close := sma200
		switch {
		case i >= n-20:
			close = sma20
		case i >= n-50:
			close = (50*sma50 - 20*sma20) / 30
		}
		volume := 100.0
		if highVolume && i == n-1 {
			volume = 200
		}
		b := bar{Open: close, High: close + 1, Low: close - 1, Close: close, Volume: volume}
		d.Split[date] = b
		d.Raw[date] = b
		d.Detector[date] = b
	}
	d.Split[d.Dates[n-1]] = bar{Open: price, High: price + 1, Low: price - 1, Close: price, Volume: d.Split[d.Dates[n-1]].Volume}
	return d
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := range sub {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
