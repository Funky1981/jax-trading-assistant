package evidencequality

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestEvaluationPopulationFiltering(t *testing.T) {
	rules := testRules()
	coverage := mustTime("2026-07-30T20:00:00Z")
	base := testEvent("genuine-1")
	synthetic := testEvent("synthetic")
	synthetic.IsSynthetic = true
	proof := testEvent("real-qqq-proof-1")
	testURL := testEvent("test-url")
	testURL.SourceURL = "https://example.com/a"
	late := testEvent("late")
	late.ReceiptAt = mustTime("2026-07-31T10:00:00Z")
	late.DecisionAt = late.ReceiptAt.Add(time.Minute)
	included, excluded := BuildPopulation([]Event{base, synthetic, proof, testURL, late}, coverage, rules)
	if len(included) != 1 || len(excluded) != 4 {
		t.Fatalf("included/excluded=%d/%d", len(included), len(excluded))
	}
}

func TestDuplicateEventsAreExcluded(t *testing.T) {
	rules := testRules()
	first := testEvent("one")
	first.ContentHash = "same"
	second := testEvent("two")
	second.ContentHash = "same"
	second.ReceiptAt = second.ReceiptAt.Add(time.Minute)
	second.DecisionAt = second.DecisionAt.Add(time.Minute)
	included, excluded := BuildPopulation([]Event{second, first}, mustTime("2026-07-30T20:00:00Z"), rules)
	if len(included) != 1 || included[0].SourceEventIdentity != first.SourceEventIdentity || excluded[0].Reason != "duplicate_event" {
		t.Fatalf("unexpected duplicate result: %+v %+v", included, excluded)
	}
}

func TestTimestampAnchorSelection(t *testing.T) {
	event := testEvent("anchors")
	event.PrimarySymbol = "QQQ"
	collection := event.PublicationAt.Add(5 * time.Minute)
	event.CollectionAt = &collection
	candles := testCandles()
	outcomes := calculateOutcomes(event, MapEvent(event, testRules()), newMarketIndex(candles), testRules())
	anchors := map[string]bool{}
	for _, outcome := range outcomes {
		anchors[outcome.Anchor] = true
	}
	for _, want := range []string{"publication", "collection", "receipt", "decision"} {
		if !anchors[want] {
			t.Fatalf("missing anchor %s in %+v", want, outcomes)
		}
	}
}

func TestDeterministicAssetMapping(t *testing.T) {
	event := testEvent("direct")
	event.PrimarySymbol = "aapl"
	mapping := MapEvent(event, testRules())
	if !mapping.Mapped || mapping.Symbol != "AAPL" || !mapping.Direct || mapping.Benchmark != "QQQ" {
		t.Fatalf("direct mapping=%+v", mapping)
	}
	event.PrimarySymbol = ""
	event.AffectedAssets = nil
	event.EventType = "energy_oil"
	mapping = MapEvent(event, testRules())
	if mapping.Symbol != "XLE" || mapping.Direct || mapping.MappingType != "event_category_proxy" {
		t.Fatalf("proxy mapping=%+v", mapping)
	}
}

func TestUnknownAssetHandling(t *testing.T) {
	event := testEvent("unknown")
	mapping := MapEvent(event, testRules())
	if mapping.Mapped || mapping.Symbol != "" {
		t.Fatalf("unknown event was forced to an asset: %+v", mapping)
	}
}

func TestV2RejectsLiveMappingCreatedAfterInitialDecision(t *testing.T) {
	event := testEvent("late-live-map")
	event.ResolutionStatus = "resolved"
	event.ResolutionSymbol = "AAPL"
	event.ResolutionRelationship = "direct"
	event.ResolutionRuleset = "event-asset-resolution-v1"
	event.MappingKnowableAtAnchor = true
	event.MappingKnownAtDecision = false
	event.DecisionOrigin = "live_origin"
	mapping := MapEvent(event, testRulesV2())
	if mapping.Mapped || mapping.Reason != "asset mapping was created after the live initial decision" {
		t.Fatalf("late live mapping accepted: %+v", mapping)
	}
}

func TestV2AcceptsDisclosedBackfillMappingKnowableAtAnchor(t *testing.T) {
	event := testEvent("backfill-map")
	event.ResolutionStatus = "resolved"
	event.ResolutionSymbol = "AAPL"
	event.ResolutionRelationship = "direct"
	event.ResolutionRuleset = "event-asset-resolution-v1"
	event.MappingKnowableAtAnchor = true
	event.DecisionOrigin = "historical_backfill"
	mapping := MapEvent(event, testRulesV2())
	if !mapping.Mapped || mapping.Symbol != "AAPL" {
		t.Fatalf("valid disclosed backfill mapping rejected: %+v", mapping)
	}
}

func TestV2RejectsMappingFromDifferentResolverVersion(t *testing.T) {
	event := testEvent("wrong-resolver")
	event.ResolutionStatus = "resolved"
	event.ResolutionSymbol = "AAPL"
	event.ResolutionRuleset = "event-asset-resolution-v0"
	event.MappingKnowableAtAnchor = true
	mapping := MapEvent(event, testRulesV2())
	if mapping.Mapped || mapping.Reason != "asset resolution belongs to a different resolver ruleset" {
		t.Fatalf("wrong resolver mapping accepted: %+v", mapping)
	}
}

func TestV2SelectsOnlyImmutableInitialBackfill(t *testing.T) {
	initial := testEvent("initial")
	initial.RulesetVersion = "genuine-event-decision-v2"
	initial.IsInitial = true
	initial.DecisionOrigin = "historical_backfill"
	noTrade := testEvent("initial-no-trade")
	noTrade.Decision = DecisionNoTrade
	noTrade.RulesetVersion = "genuine-event-decision-v2"
	noTrade.IsInitial = true
	noTrade.DecisionOrigin = "historical_backfill"
	noTrade.SubjectCurrentDecision = DecisionCandidate
	later := testEvent("later")
	later.RulesetVersion = "genuine-event-decision-v2"
	later.IsInitial = false
	later.DecisionOrigin = "historical_replay"
	included, excluded := BuildPopulation([]Event{initial, noTrade, later}, mustTime("2026-07-30T20:00:00Z"), testRulesV2())
	if len(included) != 2 || included[0].Decision != DecisionWatch || included[1].Decision != DecisionNoTrade || len(excluded) != 1 || excluded[0].Reason != "later_decision_projection" {
		t.Fatalf("unexpected v2 selection: %+v %+v", included, excluded)
	}
	if included[1].Decision == included[1].SubjectCurrentDecision {
		t.Fatal("later subject/candidate projection leaked into the initial event label")
	}
}

func TestBenchmarkSelection(t *testing.T) {
	event := testEvent("benchmark")
	event.PrimarySymbol = "QQQ"
	mapping := MapEvent(event, testRules())
	if mapping.Benchmark != "SPY" || mapping.BenchmarkReason == "" {
		t.Fatalf("benchmark=%+v", mapping)
	}
}

func TestReturnAndAbnormalReturnCalculation(t *testing.T) {
	event := testEvent("returns")
	event.PrimarySymbol = "QQQ"
	outcomes := calculateOutcomes(event, MapEvent(event, testRules()), newMarketIndex(testCandles()), testRules())
	var found *Outcome
	for i := range outcomes {
		if outcomes[i].Anchor == "receipt" && outcomes[i].Horizon == "1d" {
			found = &outcomes[i]
		}
	}
	if found == nil {
		t.Fatal("missing 1d outcome")
	}
	if math.Abs(found.RawReturn-0.02) > 1e-9 {
		t.Fatalf("raw return=%f", found.RawReturn)
	}
	if found.AbnormalReturn == nil || math.Abs(*found.AbnormalReturn-0.01) > 1e-9 {
		t.Fatalf("abnormal=%v", found.AbnormalReturn)
	}
}

func TestMarketSessionBoundaryHandling(t *testing.T) {
	series := []Candle{daily("QQQ", "2026-07-20T04:00:00Z", 100, 101), daily("QQQ", "2026-07-21T04:00:00Z", 110, 111)}
	beforeOpen := mustTime("2026-07-20T12:00:00Z")
	result, ok := dailyOutcome(series, beforeOpen, 1)
	if !ok || result.Start != 100 {
		t.Fatalf("pre-open result=%+v/%t", result, ok)
	}
	duringSession := mustTime("2026-07-20T15:00:00Z")
	result, ok = dailyOutcome(series, duringSession, 1)
	if !ok || result.Start != 110 {
		t.Fatalf("during-session used look-ahead bar: %+v/%t", result, ok)
	}
}

func TestWeekendAnchorUsesNextPersistedSession(t *testing.T) {
	series := []Candle{daily("QQQ", "2026-07-17T04:00:00Z", 100, 101), daily("QQQ", "2026-07-20T04:00:00Z", 102, 103), daily("QQQ", "2026-07-21T04:00:00Z", 104, 105)}
	result, ok := dailyOutcome(series, mustTime("2026-07-18T12:00:00Z"), 1)
	if !ok || !result.EffectiveAnchor.Equal(mustTime("2026-07-20T13:30:00Z")) || result.Start != 102 {
		t.Fatalf("weekend anchor result=%+v/%t", result, ok)
	}
}

func TestHolidayAnchorUsesNextPersistedSession(t *testing.T) {
	series := []Candle{daily("QQQ", "2026-07-02T04:00:00Z", 100, 101), daily("QQQ", "2026-07-06T04:00:00Z", 102, 103), daily("QQQ", "2026-07-07T04:00:00Z", 104, 105)}
	result, ok := dailyOutcome(series, mustTime("2026-07-03T12:00:00Z"), 1)
	if !ok || !result.EffectiveAnchor.Equal(mustTime("2026-07-06T13:30:00Z")) || result.Start != 102 {
		t.Fatalf("holiday anchor result=%+v/%t", result, ok)
	}
}

func TestSymbolWithoutBenchmarkHasRawButNoAbnormalOutcome(t *testing.T) {
	event := testEvent("no-benchmark")
	event.PrimarySymbol = "IBM"
	outcomes := calculateOutcomes(event, MapEvent(event, testRules()), newMarketIndex([]Candle{
		daily("IBM", "2026-07-20T04:00:00Z", 100, 101), daily("IBM", "2026-07-21T04:00:00Z", 102, 103),
	}), testRules())
	if len(outcomes) == 0 {
		t.Fatal("missing raw outcome for symbol without benchmark")
	}
	for _, outcome := range outcomes {
		if outcome.AbnormalReturn != nil {
			t.Fatalf("invented abnormal return without benchmark: %+v", outcome)
		}
	}
}

func TestMissingCandles(t *testing.T) {
	series := []Candle{daily("QQQ", "2026-07-20T04:00:00Z", 100, 101), daily("QQQ", "2026-07-28T04:00:00Z", 102, 103)}
	if _, ok := dailyOutcome(series, mustTime("2026-07-20T12:00:00Z"), 2); ok {
		t.Fatal("large candle gap should invalidate the window")
	}
}

func TestPublicationBeforeReceiptLatency(t *testing.T) {
	event := testEvent("bad-order")
	event.PublicationAt = event.ReceiptAt.Add(time.Minute)
	included, excluded := BuildPopulation([]Event{event}, mustTime("2026-07-30T20:00:00Z"), testRules())
	if len(included) != 0 || len(excluded) != 1 || excluded[0].Reason != "invalid_timestamp_order" {
		t.Fatalf("unexpected timing result: %+v %+v", included, excluded)
	}
}

func TestNoLookAheadBias(t *testing.T) {
	series := []Candle{{Symbol: "QQQ", Timeframe: "1h", Source: "alpaca", TimestampSemantics: "interval_start", Timestamp: mustTime("2026-07-20T14:00:00Z"), Open: 100, High: 105, Low: 99, Close: 104}, {Symbol: "QQQ", Timeframe: "1h", Source: "alpaca", TimestampSemantics: "interval_start", Timestamp: mustTime("2026-07-20T15:00:00Z"), Open: 110, High: 112, Low: 109, Close: 111}}
	result, ok := intradayOutcome(series, mustTime("2026-07-20T14:30:00Z"), 24*time.Hour)
	if !ok || !result.EffectiveAnchor.Equal(mustTime("2026-07-20T15:00:00Z")) || result.Start != 110 {
		t.Fatalf("pre-anchor candle leaked into result: %+v/%t", result, ok)
	}
}

func TestDeterministicRerun(t *testing.T) {
	snapshot := testSnapshot()
	first, err := Evaluate(snapshot, testRules(), testRuntime())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate(snapshot, testRules(), testRuntime())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) || first.InputFingerprint != second.InputFingerprint {
		t.Fatal("unchanged input did not produce byte-stable JSON")
	}
}

func TestNoMutationOfLiveRecords(t *testing.T) {
	snapshot := testSnapshot()
	snapshot.SafetyAfter.OrderIntents = 1
	if _, err := Evaluate(snapshot, testRules(), testRuntime()); err == nil {
		t.Fatal("changed prohibited counts were accepted")
	}
}

func TestDescriptiveAndNonParametricStatistics(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	low, high := bootstrapMedianCI(values, 1000)
	if low > 3 || high < 3 {
		t.Fatalf("bootstrap interval %f..%f", low, high)
	}
	if mannWhitneyU([]float64{3, 4}, []float64{1, 2}) != 4 {
		t.Fatal("unexpected Mann-Whitney U")
	}
	if cliffsDelta([]float64{3, 4}, []float64{1, 2}) != 1 {
		t.Fatal("unexpected Cliff's delta")
	}
	p1 := permutationP([]float64{3, 4, 5}, []float64{0, 1, 2}, 1000)
	p2 := permutationP([]float64{3, 4, 5}, []float64{0, 1, 2}, 1000)
	if p1 != p2 {
		t.Fatal("permutation test is not deterministic")
	}
}

func testRules() Ruleset {
	return Ruleset{Version: "historical-evidence-quality-v1", PrimaryAnchor: "receipt", MinimumComparisonGroupSize: 2, BootstrapIterations: 100, PermutationIterations: 100, MaximumIntradayAnchorDelayMinutes: 1440, ControlledSourcePrefixes: []string{"world-monitor-local-proof"}, ControlledEventIDMarkers: []string{"real-qqq-proof-"}, ControlledHeadlineMarkers: []string{"local proof event:"}, TestHosts: []string{"example.com"}, CategoryProxies: map[string]ProxyRule{"energy_oil": {Symbol: "XLE", Confidence: "medium", Reason: "energy proxy"}}, Benchmarks: map[string]BenchmarkRule{"AAPL": {Symbol: "QQQ", Reason: "technology benchmark"}, "QQQ": {Symbol: "SPY", Reason: "broad benchmark"}, "XLE": {Symbol: "SPY", Reason: "sector benchmark"}}}
}

func testRulesV2() Ruleset {
	rules := testRules()
	rules.Version = "historical-evidence-quality-v2"
	rules.DecisionRulesetVersion = "genuine-event-decision-v2"
	rules.AssetResolverRulesetVersion = "event-asset-resolution-v1"
	rules.IncludedDecisionOrigins = []string{"historical_backfill"}
	rules.CategoryProxies = map[string]ProxyRule{}
	return rules
}
func testEvent(id string) Event {
	publication := mustTime("2026-07-20T12:00:00Z")
	receipt := mustTime("2026-07-20T12:10:00Z")
	return Event{DecisionID: id, InboxID: id, SourceEventIdentity: "world-monitor:" + id, Decision: DecisionWatch, RulesetVersion: "genuine-event-decision-v1", PublicationAt: publication, ReceiptAt: receipt, DecisionAt: receipt.Add(time.Minute), Source: "world-monitor", SourceURL: "https://news.test.org/" + id, EventType: "unknown", SourceName: "Test News", Headline: "Genuine event " + id, SubjectEventCount: 1}
}
func testCandles() []Candle {
	return []Candle{daily("QQQ", "2026-07-20T04:00:00Z", 100, 102), daily("QQQ", "2026-07-21T04:00:00Z", 103, 104), daily("QQQ", "2026-07-22T04:00:00Z", 104, 105), daily("QQQ", "2026-07-23T04:00:00Z", 105, 106), daily("QQQ", "2026-07-24T04:00:00Z", 106, 107), daily("QQQ", "2026-07-27T04:00:00Z", 107, 108), daily("SPY", "2026-07-20T04:00:00Z", 200, 202), daily("SPY", "2026-07-21T04:00:00Z", 202, 203), daily("SPY", "2026-07-22T04:00:00Z", 203, 204), daily("SPY", "2026-07-23T04:00:00Z", 204, 205), daily("SPY", "2026-07-24T04:00:00Z", 205, 206), daily("SPY", "2026-07-27T04:00:00Z", 206, 207), hourly("QQQ", "2026-07-20T13:00:00Z", 100, 101), hourly("QQQ", "2026-07-20T14:00:00Z", 101, 102), hourly("SPY", "2026-07-20T13:00:00Z", 200, 201), hourly("SPY", "2026-07-20T14:00:00Z", 201, 202)}
}
func daily(symbol, at string, open, close float64) Candle {
	return Candle{Symbol: symbol, Timestamp: mustTime(at), Open: open, High: math.Max(open, close) + 1, Low: math.Min(open, close) - 1, Close: close, Timeframe: "1d", Source: "alpaca", TimestampSemantics: "interval_start", MarketDataClassification: "unknown"}
}
func hourly(symbol, at string, open, close float64) Candle {
	return Candle{Symbol: symbol, Timestamp: mustTime(at), Open: open, High: math.Max(open, close), Low: math.Min(open, close), Close: close, Timeframe: "1h", Source: "alpaca", TimestampSemantics: "interval_start", MarketDataClassification: "unknown"}
}
func testSnapshot() Snapshot {
	event := testEvent("deterministic")
	event.PrimarySymbol = "QQQ"
	return Snapshot{Events: []Event{event}, Candles: testCandles(), SafetyBefore: SafetyCounts{Approvals: 2}, SafetyAfter: SafetyCounts{Approvals: 2}}
}
func testRuntime() RuntimeSafety { return RuntimeSafety{RuntimeMode: "paper", MaximumLeverage: 1} }
func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
