package main

import (
	"math"
	"reflect"
	"sort"
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
	result := secondarySignPermutation(secondaryDirectionalDiagnostics{Observations: mixed}, "manifest", cfg)
	if result.Status != FalsificationInformational || result.Replicates != 10000 || result.MixedStrata != 1 || result.PValue < 0 || result.PValue > 1 || result.Algorithm != signPermutationAlgorithmVer {
		t.Fatalf("unexpected mixed sign permutation: %+v", result)
	}
	na := secondarySignPermutation(secondaryDirectionalDiagnostics{Observations: []directionalObservation{{Instrument: "SPY", Year: 2023, Direction: "BUY", Magnitude: .1}}}, "manifest", cfg)
	if na.Status != FalsificationNotApplicable {
		t.Fatalf("single-direction stratum status = %s, want NOT_APPLICABLE", na.Status)
	}
	reordered := secondarySignPermutation(secondaryDirectionalDiagnostics{Observations: []directionalObservation{mixed[1], mixed[0]}}, "manifest", cfg)
	if !reflect.DeepEqual(result, reordered) {
		t.Fatal("sign permutation changed with input order")
	}
	if seedFor("manifest", "|secondary-sign-permutation-v1|") == seedFor("manifest", "|different-domain|") {
		t.Fatal("sign seed domain is not separated")
	}
	if math.Abs(signedMean([]directionalObservation{{Direction: "BUY", Magnitude: .1}, {Direction: "BUY", Magnitude: .2}})-.15) > 1e-12 {
		t.Fatal("all-positive observed directional statistic is not stable")
	}
	withSingle := append(append([]directionalObservation{}, mixed...), directionalObservation{Instrument: "QQQ", Year: 2023, Direction: "BUY", Magnitude: .9})
	filtered := secondarySignPermutation(secondaryDirectionalDiagnostics{Observations: withSingle}, "manifest", cfg)
	if filtered.ObservationsUsed != 2 || filtered.ObservationsExcluded != 1 || !reflect.DeepEqual(filtered.MixedStrataIDs, []string{"SPY-2023"}) || !reflect.DeepEqual(filtered.ExcludedSingleDirectionStrataIDs, []string{"QQQ-2023"}) {
		t.Fatalf("single-direction strata were not excluded: %+v", filtered)
	}
}

func TestSecondaryDiagnosticsArePopulatedByProductionEvaluator(t *testing.T) {
	dates := syntheticDates(30)
	closes := []float64{10, 9, 8, 7, 9, 11, 13, 13, 12, 11, 10, 9, 8, 7, 6, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	data := map[string]*instrumentData{}
	for _, symbol := range symbols {
		data[symbol] = syntheticData(dates, func(i int) float64 { return closes[i] })
	}
	cfg := testConfig()
	cfg.SMAFast, cfg.SMAMedium, cfg.SMASlow = 2, 3, 4
	cfg.ATRPeriod, cfg.AvgVolumePeriod, cfg.HoldingPeriod = 1, 1, 2
	cfg.StopATR, cfg.TargetATR = 1, 3
	cfg.OOSStart, cfg.OOSEnd = dates[0], dates[len(dates)-1]
	r, err := evaluatePartition(data, "synthetic", cfg.OOSStart, cfg.OOSEnd, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.SecondaryDiagnostics.Policy != secondaryDiagnosticPolicyVer || len(r.SecondaryDiagnostics.Observations) == 0 {
		t.Fatalf("production evaluator did not persist secondary diagnostics: %+v", r.SecondaryDiagnostics)
	}
	seen := map[string]bool{}
	for _, observation := range r.SecondaryDiagnostics.Observations {
		seen[observation.Direction] = true
	}
	if !seen["BUY"] || !seen["SELL"] {
		t.Fatalf("production diagnostics did not retain both directions: %+v", seen)
	}
	falsification := buildFalsification(data, r, "manifest", cfg, r.SecondaryDiagnostics)
	summary, ok := falsification["secondary_sign_permutation"].(signPermutationSummary)
	if !ok || summary.ObservationsUsed == 0 {
		t.Fatalf("production diagnostics did not reach falsification builder: %#v", falsification["secondary_sign_permutation"])
	}
}

func TestSecondaryPermutationUsesExplicitBuilderObservations(t *testing.T) {
	cfg := testConfig()
	observations := secondaryDirectionalDiagnostics{Observations: []directionalObservation{{Instrument: "SPY", Year: 2023, Direction: "BUY", Magnitude: .2}, {Instrument: "SPY", Year: 2023, Direction: "SELL", Magnitude: .1}}}
	result := buildFalsification(map[string]*instrumentData{}, partitionResult{SecondaryDiagnostics: observations}, "manifest", cfg, observations)
	summary, ok := result["secondary_sign_permutation"].(signPermutationSummary)
	if !ok || summary.MixedStrata != 1 || summary.Status != FalsificationInformational {
		t.Fatalf("explicit secondary observations did not reach builder: %#v", result["secondary_sign_permutation"])
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
	if before <= after || len(removed) != 1 || removed[0] != "other" || topFiveRuleVersion != "CEIL_5_PERCENT_V1" {
		t.Fatalf("top-five exclusion = before %v after %v removed %v", before, after, removed)
	}
}

func TestTopFiveCeilingBoundaryAndIDs(t *testing.T) {
	for _, tc := range []struct{ n, want int }{{1, 1}, {19, 1}, {20, 1}, {21, 2}, {39, 2}, {40, 2}, {41, 3}, {100, 5}} {
		if got := topFiveRemovalCount(tc.n); got != tc.want {
			t.Fatalf("N=%d removal count=%d, want %d", tc.n, got, tc.want)
		}
		eps := make([]episode, tc.n)
		for i := range eps {
			eps[i] = episode{ID: string(rune('a' + i%26)), NetReturn: float64(tc.n - i)}
		}
		_, _, removed := topFivePercentExclusion(eps)
		if len(removed) != tc.want {
			t.Fatalf("N=%d removed IDs=%d, want %d", tc.n, len(removed), tc.want)
		}
	}
	boundary := []episode{{ID: "1", NetReturn: .5}, {ID: "2", NetReturn: .4}, {ID: "3", NetReturn: .3}, {ID: "4", NetReturn: .2}, {ID: "5", NetReturn: .1}, {ID: "6", NetReturn: 0}}
	_, _, removed := topFivePercentExclusion(boundary)
	if !reflect.DeepEqual(removed, []string{"1"}) {
		t.Fatalf("removed IDs = %v, want [1]", removed)
	}
}

func TestTopFiveBlockingDispositions(t *testing.T) {
	cfg := testConfig()
	cfg.PairedFloor = 2
	robust := []episode{{ID: "a", NetReturn: .20, PlaceboNetReturn: floatPtr(0)}, {ID: "b", NetReturn: .10, PlaceboNetReturn: floatPtr(0)}, {ID: "c", NetReturn: .05, PlaceboNetReturn: floatPtr(0)}}
	_, disposition := topFiveFalsification(robust, cfg)
	if disposition.Status != FalsificationPass || !disposition.Blocking {
		t.Fatalf("robust remainder = %+v", disposition)
	}
	failed := []episode{{ID: "a", NetReturn: .50, PlaceboNetReturn: floatPtr(0)}, {ID: "b", NetReturn: -.10, PlaceboNetReturn: floatPtr(0)}, {ID: "c", NetReturn: -.20, PlaceboNetReturn: floatPtr(0)}}
	_, disposition = topFiveFalsification(failed, cfg)
	if disposition.Status != FalsificationFail || !disposition.Blocking {
		t.Fatalf("failed remainder = %+v", disposition)
	}
	insufficient := []episode{{ID: "a", NetReturn: .5, PlaceboNetReturn: floatPtr(0)}, {ID: "b", NetReturn: .1, PlaceboNetReturn: floatPtr(0)}}
	_, disposition = topFiveFalsification(insufficient, cfg)
	if disposition.Status != FalsificationInsufficient || !disposition.Blocking {
		t.Fatalf("insufficient remainder = %+v", disposition)
	}
	base := partitionResult{Episodes: robust, MatchedPairs: 2, Blocks: 1, Instruments: 1, MaxInstrumentShare: .2, RegimeSlices: 1, MeanNet: .1, BootstrapLow: .01, MeanPairedDifference: .1, PairedBootstrapLow: .01}
	if got := classifyPromotion(base, []FalsificationDisposition{{Name: "top", Status: FalsificationFail, Blocking: true}}, testConfigWithFloors(cfg)); got != "FAILED_VALIDATION" {
		t.Fatalf("blocking top-five failure classified as %s", got)
	}
	if got := classifyPromotion(base, []FalsificationDisposition{{Name: "top", Status: FalsificationInsufficient, Blocking: true}}, testConfigWithFloors(cfg)); got != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("blocking top-five insufficiency classified as %s", got)
	}
}

func TestCalendarRegimeBreadthCells(t *testing.T) {
	ones := []episode{{Year: 2023, Regime: "SPY_ABOVE_SMA200"}, {Year: 2023, Regime: "SPY_BELOW_SMA200"}}
	if regimeBreadth(ones) != 2 {
		t.Fatalf("one year/two regimes breadth=%d", regimeBreadth(ones))
	}
	twoYears := []episode{{Year: 2023, Regime: "SPY_ABOVE_SMA200"}, {Year: 2024, Regime: "SPY_ABOVE_SMA200"}}
	if regimeBreadth(twoYears) != 2 {
		t.Fatalf("two years/one regime breadth=%d", regimeBreadth(twoYears))
	}
	three := append(twoYears, episode{Year: 2024, Regime: "SPY_BELOW_SMA200"})
	three = append(three, episode{Year: 2024, Regime: "SPY_BELOW_SMA200"}, episode{Year: 2025, Regime: "UNKNOWN"})
	if regimeBreadth(three) != 3 || !reflect.DeepEqual(calendarRegimeCells(three), []string{"2023|SPY_ABOVE_SMA200", "2024|SPY_ABOVE_SMA200", "2024|SPY_BELOW_SMA200"}) {
		t.Fatalf("year-regime cells=%v", calendarRegimeCells(three))
	}
	r := partitionResult{Episodes: three}
	finishPartition(&r)
	if r.RegimeSlices != len(r.CalendarRegimeCells) || !reflect.DeepEqual(r.CalendarRegimeCells, calendarRegimeCells(three)) {
		t.Fatalf("partition did not persist exact calendar-regime cells: %+v", r)
	}
}

func TestUSSessionOverlapIgnoresWeekendAndHoliday(t *testing.T) {
	dates := usSessionDates(45)
	data := map[string]*instrumentData{"SPY": syntheticData(dates, func(i int) float64 { return 10 }), "QQQ": syntheticData(dates, func(i int) float64 { return 10 })}
	eps := []episode{{ID: "early", Instrument: "SPY", EntryDate: dates[0]}, {ID: "conflict", Instrument: "SPY", EntryDate: dates[19]}, {ID: "allowed", Instrument: "SPY", EntryDate: dates[20]}, {ID: "other", Instrument: "QQQ", EntryDate: dates[19]}}
	kept := overlapFilter(eps, data, 20)
	ids := []string{}
	for _, ep := range kept {
		ids = append(ids, ep.ID)
	}
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{"allowed", "early", "other"}) {
		t.Fatalf("session overlap kept=%v", ids)
	}
	if dates[5] != "2023-01-09" {
		t.Fatalf("session fixture did not skip the MLK holiday: %s", dates[5])
	}
}

func floatPtr(v float64) *float64 { return &v }

func testConfigWithFloors(cfg FrozenExperimentConfig) FrozenExperimentConfig {
	cfg.OOSSampleFloor, cfg.PairedFloor, cfg.EffectiveBlockFloor, cfg.InstrumentFloor, cfg.MinimumSliceFloor = 1, 1, 1, 1, 1
	return cfg
}

func usSessionDates(n int) []string {
	date := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result := []string{}
	for len(result) < n {
		weekday := date.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday && date.Format("2006-01-02") != "2023-01-16" {
			result = append(result, date.Format("2006-01-02"))
		}
		date = date.AddDate(0, 0, 1)
	}
	return result
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
	if got := classifyPromotion(oos, []FalsificationDisposition{{Name: "diagnostic", Status: FalsificationFail, Blocking: false}}, cfg); got != "FORWARD_PAPER_ELIGIBLE" {
		t.Fatalf("non-blocking failure unexpectedly blocked promotion: %s", got)
	}
	if got := classifyPromotion(oos, []FalsificationDisposition{{Name: "unknown", Status: FalsificationStatus("UNKNOWN"), Blocking: false}}, cfg); got != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("unknown non-blocking disposition was not fail-closed: %s", got)
	}
	oos.MeanNet = -0.01
	if got := classifyPromotion(oos, info, cfg); got != "FAILED_VALIDATION" {
		t.Fatalf("informational disposition rescued a failed primary gate: %s", got)
	}
}
