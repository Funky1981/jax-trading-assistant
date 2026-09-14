package main

import (
	"math"
	"testing"
	"time"
)

func syntheticDates(n int) []string {
	start := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	dates := make([]string, n)
	for i := range dates {
		dates[i] = start.AddDate(0, 0, i).Format("2006-01-02")
	}
	return dates
}

func syntheticData(dates []string, closeFn func(int) float64) *instrumentData {
	d := &instrumentData{Raw: map[string]bar{}, Split: map[string]bar{}, Detector: map[string]bar{}, Factor: map[string]float64{}, Boundaries: map[string]bool{}, Dates: append([]string(nil), dates...)}
	for i, date := range dates {
		close := closeFn(i)
		b := bar{Date: date, Open: close, High: close + 1, Low: close - 1, Close: close, Volume: 100}
		d.Raw[date], d.Split[date], d.Detector[date] = b, b, b
		d.Factor[date] = 1
	}
	return d
}

func testConfig() FrozenExperimentConfig {
	return FrozenExperimentConfig{SMAFast: 1, SMAMedium: 1, SMASlow: 2, ATRPeriod: 1, AvgVolumePeriod: 1, HoldingPeriod: 2, BootstrapN: 10000, ActionableConfidenceThreshold: .6}
}

func TestMatchedPlaceboEligibilityAndTradingSessionDistance(t *testing.T) {
	dates := syntheticDates(365)
	data := map[string]*instrumentData{"SPY": syntheticData(dates, func(i int) float64 {
		if i == 30 || i == 28 {
			return 10
		}
		if i == 29 {
			return 8
		}
		if i < 30 {
			return 9
		}
		return 9
	})}
	cfg := testConfig()
	ep := episode{Instrument: "SPY", SignalDate: dates[30], ExitDate: dates[31]}
	intervals := []activeInterval{newActiveInterval("SPY", 30, 31)}
	got := matchedPlaceboDate(data, ep, dates[0], dates[364], map[string]bool{}, intervals, cfg, "manifest")
	if got != dates[28] {
		t.Fatalf("matched placebo = %s, want farther same-regime date %s; wrong-regime closer date was incorrectly eligible", got, dates[28])
	}
	if dateInActiveIntervals(30, intervals) == false || dateInActiveIntervals(32, intervals) {
		t.Fatal("active interval boundaries are not signal-through-exit inclusive")
	}

	used := map[string]bool{placeboKey("SPY", dates[28]): true}
	if got := matchedPlaceboDate(data, ep, dates[0], dates[364], used, intervals, cfg, "manifest"); got == dates[28] {
		t.Fatal("one-to-one placebo matching reused a date")
	}
	if absInt(90-30) != 60 || absInt(91-30) <= 60 {
		t.Fatal("trading-session distance boundary is not exact")
	}
	actual := syntheticData(dates[:8], func(i int) float64 {
		if i == 3 {
			return 5
		}
		return 1
	})
	actualCfg := FrozenExperimentConfig{SMAFast: 2, SMAMedium: 3, SMASlow: 4, ATRPeriod: 1, AvgVolumePeriod: 1, ActionableConfidenceThreshold: .6}
	if !actualSignalDate(actual, 3, actualCfg) {
		t.Fatal("synthetic actionable MA date was not recognized")
	}
}

func TestTimestampPlaceboRegisteredReplicatesAndEligibility(t *testing.T) {
	dates := syntheticDates(40)
	data := map[string]*instrumentData{"SPY": syntheticData(dates, func(i int) float64 { return 10 })}
	cfg := testConfig()
	first := timestampPlaceboSelections(data, dates[0], dates[len(dates)-1], "manifest", cfg, map[string][]activeInterval{"SPY": {newActiveInterval("SPY", 10, 12)}})
	second := timestampPlaceboSelections(data, dates[0], dates[len(dates)-1], "manifest", cfg, map[string][]activeInterval{"SPY": {newActiveInterval("SPY", 10, 12)}})
	if len(first) != cfg.BootstrapN || len(second) != cfg.BootstrapN {
		t.Fatalf("timestamp replicate count = %d/%d, want %d", len(first), len(second), cfg.BootstrapN)
	}
	for i := range first {
		if first[i] != second[i] || dateInActiveIntervals(indexOf(dates, first[i].Date), []activeInterval{newActiveInterval("SPY", 10, 12)}) {
			t.Fatal("timestamp placebo is not deterministic or selected an active date")
		}
	}
	if seedFor("manifest", "|timestamp-placebo-v1|") == seedFor("manifest", "|secondary-sign-permutation-v1|") {
		t.Fatal("timestamp seed domain is not separated")
	}
}

func TestSignPermutationMechanics(t *testing.T) {
	cfg := testConfig()
	mixed := []directionalObservation{{Instrument: "SPY", Year: 2023, Direction: "BUY", Magnitude: .1}, {Instrument: "SPY", Year: 2023, Direction: "SELL", Magnitude: .1}}
	result := signPermutation(mixed, "manifest", cfg)
	if result.Status != FalsificationPass || result.Replicates != 10000 || result.MixedStrata != 1 || result.PValue < 0 || result.PValue > 1 {
		t.Fatalf("unexpected mixed sign permutation: %+v", result)
	}
	na := signPermutation([]directionalObservation{{Instrument: "SPY", Year: 2023, Direction: "BUY", Magnitude: .1}}, "manifest", cfg)
	if na.Status != FalsificationNotApplicable {
		t.Fatalf("single-direction stratum status = %s, want NOT_APPLICABLE", na.Status)
	}
}

func TestRegimeThemeAndBenchmarkArithmetic(t *testing.T) {
	dates := syntheticDates(4)
	data := map[string]*instrumentData{"SPY": syntheticData(dates, func(i int) float64 { return float64(i + 1) })}
	cfg := testConfig()
	if regimeAtDate(data, dates[2], cfg) != "SPY_ABOVE_SMA200" && regimeAtDate(data, dates[2], cfg) != "SPY_BELOW_SMA200" {
		t.Fatal("regime split did not return a signal-time regime")
	}
	if themeForInstrument("XLK") != "sector_equity" || themeForInstrument("GLD") != "rates_precious_metal" || themeForInstrument("SPY") != "broad_equity" {
		t.Fatal("theme membership does not follow the frozen groups")
	}
	slices := descriptiveSlices(partitionResult{Episodes: []episode{{ID: "a", Theme: "broad_equity", Regime: "SPY_ABOVE_SMA200", NetReturn: .1}, {ID: "b", Theme: "sector_equity", Regime: "SPY_BELOW_SMA200", NetReturn: -.1}}}, "theme")
	themeValues := slices["slices"].(map[string]any)
	if len(themeValues) != 2 || themeValues["broad_equity"].(map[string]any)["episode_count"] != 1 {
		t.Fatal("theme aggregation collapsed distinct groups")
	}
	if got := buyHoldReturn(100, 110); math.Abs(got-.1) > 1e-12 {
		t.Fatalf("buy-and-hold return = %v, want 0.1", got)
	}
	if buyHoldReturn(100, 100) != 0 {
		t.Fatal("zero baseline is not numeric zero")
	}
}

func TestSessionOverlapAndTopFiveExclusion(t *testing.T) {
	dates := syntheticDates(50)
	data := map[string]*instrumentData{"SPY": syntheticData(dates, func(i int) float64 { return 10 })}
	eps := []episode{{ID: "a", Instrument: "SPY", EntryDate: dates[1], NetReturn: .01}, {ID: "b", Instrument: "SPY", EntryDate: dates[20], NetReturn: .02}, {ID: "c", Instrument: "SPY", EntryDate: dates[21], NetReturn: .03}, {ID: "other", Instrument: "SPY", EntryDate: dates[41], NetReturn: .04}}
	kept := overlapFilter(eps, data, 20)
	if len(kept) != 3 || kept[0].ID != "a" || kept[1].ID != "c" || kept[2].ID != "other" {
		t.Fatalf("overlap filter = %+v, want earliest 1,21,41 sessions", kept)
	}
	before, after, removed := topFivePercentExclusion(eps)
	if before <= after || len(removed) != 1 || removed[0] != "other" {
		t.Fatalf("top-five exclusion = before %v after %v removed %v", before, after, removed)
	}
}

func TestPromotionRequiresFalsificationBeforeEligibility(t *testing.T) {
	cfg := testConfig()
	cfg.OOSSampleFloor, cfg.PairedFloor, cfg.EffectiveBlockFloor, cfg.InstrumentFloor, cfg.MinimumSliceFloor = 1, 1, 1, 1, 1
	cfg.ConcentrationCeiling = .4
	oos := partitionResult{Episodes: []episode{{NetReturn: .1}}, MatchedPairs: 1, Blocks: 1, Instruments: 1, MaxInstrumentShare: .2, RegimeSlices: 1, MeanNet: .1, BootstrapLow: .01, MeanPairedDifference: .1, PairedBootstrapLow: .01}
	if got := classifyPromotion(oos, nil, cfg); got != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("empty falsification set classified as %s", got)
	}
	failed := []FalsificationDisposition{{Name: "matched", Status: FalsificationFail, Blocking: true}}
	if got := classifyPromotion(oos, failed, cfg); got != "FAILED_VALIDATION" {
		t.Fatalf("failed falsification classified as %s", got)
	}
	insufficient := []FalsificationDisposition{{Name: "matched", Status: FalsificationInsufficient, Blocking: true}}
	if got := classifyPromotion(oos, insufficient, cfg); got != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("insufficient falsification classified as %s", got)
	}
	info := []FalsificationDisposition{{Name: "robustness", Status: FalsificationInformational}}
	if got := classifyPromotion(oos, info, cfg); got != "FORWARD_PAPER_ELIGIBLE" {
		t.Fatalf("informational-only disposition should not block otherwise passing synthetic gate: %s", got)
	}
	oos.MeanNet = -0.01
	if got := classifyPromotion(oos, info, cfg); got != "FAILED_VALIDATION" {
		t.Fatalf("informational disposition rescued a failed primary gate: %s", got)
	}
}
