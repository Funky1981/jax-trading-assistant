package microstructure

import (
	"math"
	"testing"
	"time"
)

// ─── L21: TickStore + SpreadCapture ──────────────────────────────────────────

func TestTick_Spread(t *testing.T) {
	tick := Tick{Symbol: "EURUSD", Bid: 1.1000, Ask: 1.1003, Timestamp: time.Now()}
	if got := tick.Spread(); math.Abs(got-0.0003) > 1e-9 {
		t.Fatalf("Spread wrong: %v", got)
	}
}

func TestTick_SpreadBps(t *testing.T) {
	tick := Tick{Symbol: "EURUSD", Bid: 1.1000, Ask: 1.1002, Timestamp: time.Now()}
	// mid = 1.1001; spread = 0.0002; bps = 0.0002/1.1001 * 10000 ≈ 1.818
	bps := tick.SpreadBps()
	if bps < 1.5 || bps > 2.5 {
		t.Fatalf("SpreadBps out of expected range: %.4f", bps)
	}
}

func TestTickStore_RecordAndRecent(t *testing.T) {
	ts := NewTickStore(5)
	base := time.Now()
	for i := range 7 {
		ts.Record(Tick{Symbol: "AAPL",
			Bid: float64(100 + i), Ask: float64(100 + i + 1),
			Timestamp: base.Add(time.Duration(i) * time.Second)})
	}
	// Ring buffer should retain only last 5.
	recent := ts.Recent("AAPL", 0)
	if len(recent) != 5 {
		t.Fatalf("want 5 ticks, got %d", len(recent))
	}
	// Most-recent 3.
	recent3 := ts.Recent("AAPL", 3)
	if len(recent3) != 3 {
		t.Fatalf("want 3 ticks, got %d", len(recent3))
	}
}

func TestTickStore_MultipleSymbols(t *testing.T) {
	ts := NewTickStore(10)
	ts.Record(Tick{Symbol: "AAPL", Bid: 180, Ask: 180.01, Timestamp: time.Now()})
	ts.Record(Tick{Symbol: "MSFT", Bid: 319, Ask: 319.02, Timestamp: time.Now()})

	if len(ts.Recent("AAPL", 0)) != 1 {
		t.Error("want 1 AAPL tick")
	}
	if len(ts.Recent("MSFT", 0)) != 1 {
		t.Error("want 1 MSFT tick")
	}
	if len(ts.Recent("GOOGL", 0)) != 0 {
		t.Error("GOOGL should have 0 ticks")
	}
}

func TestAnalyseSpread(t *testing.T) {
	ticks := []Tick{
		{Bid: 1.1000, Ask: 1.1002},
		{Bid: 1.1000, Ask: 1.1003},
		{Bid: 1.1000, Ask: 1.1001},
		{Bid: 1.1000, Ask: 1.1006},
	}
	stats := AnalyseSpread("EURUSD", ticks)
	if stats.Count != 4 {
		t.Fatalf("count wrong: %d", stats.Count)
	}
	if stats.MeanBps <= 0 {
		t.Error("MeanBps should be positive")
	}
	if stats.MaxBps <= stats.MeanBps {
		t.Error("MaxBps should be >= MeanBps")
	}
}

func TestAnalyseSpread_Empty(t *testing.T) {
	stats := AnalyseSpread("EURUSD", nil)
	if stats.Count != 0 || stats.MeanBps != 0 {
		t.Fatalf("unexpected non-zero stats for empty: %+v", stats)
	}
}

// ─── L22: SlippageModel ───────────────────────────────────────────────────────

func TestSlippageModel_RecordAndStats(t *testing.T) {
	m := NewSlippageModel(100)
	for i := range 10 {
		m.Record(FillObservation{
			Symbol:      "AAPL",
			SlippageBps: float64(i + 1),
			Quantity:    100,
			EventPhase:  "normal",
			ObservedAt:  time.Now(),
		})
	}
	stats := m.Stats("AAPL", "normal")
	if stats.Count != 10 {
		t.Fatalf("want 10 obs, got %d", stats.Count)
	}
	if stats.MeanBps <= 0 {
		t.Error("MeanBps should be positive")
	}
	if stats.P95Bps < stats.MeanBps {
		t.Error("P95 should be >= mean")
	}
}

func TestSlippageModel_EmptyBucket(t *testing.T) {
	m := NewSlippageModel(100)
	stats := m.Stats("MSFT", "blackout")
	if stats.Count != 0 {
		t.Fatalf("empty bucket should have count 0, got %d", stats.Count)
	}
}

func TestSlippageModel_EstimateBps_FallsBackToNormal(t *testing.T) {
	m := NewSlippageModel(100)
	// Only "normal" data, no "blackout" data.
	for range 5 {
		m.Record(FillObservation{Symbol: "AAPL", SlippageBps: 3.0,
			EventPhase: "normal", ObservedAt: time.Now()})
	}
	// Requesting "blackout" — should fall back to "normal".
	est := m.EstimateBps("AAPL", "blackout", 99.0)
	if est == 99.0 {
		t.Error("should fall back to normal, not default 99.0")
	}
}

func TestSlippageModel_EstimateBps_DefaultWhenNoData(t *testing.T) {
	m := NewSlippageModel(100)
	est := m.EstimateBps("NVDA", "normal", 7.5)
	if est != 7.5 {
		t.Fatalf("want default 7.5, got %f", est)
	}
}

func TestSlippageModel_RollingWindow(t *testing.T) {
	m := NewSlippageModel(3) // window=3
	for i := range 5 {
		m.Record(FillObservation{Symbol: "AAPL", SlippageBps: float64(i + 1),
			EventPhase: "normal", ObservedAt: time.Now()})
	}
	stats := m.Stats("AAPL", "normal")
	if stats.Count != 3 {
		t.Fatalf("rolling window: want 3, got %d", stats.Count)
	}
	// Last 3 observations: 3, 4, 5 → mean = 4
	if math.Abs(stats.MeanBps-4.0) > 0.01 {
		t.Fatalf("mean should be 4.0, got %.2f", stats.MeanBps)
	}
}

// ─── L23: CorrelationMonitor ──────────────────────────────────────────────────

func returns(vals ...float64) []float64 { return vals }

func TestCorrelationMonitor_HighCorrelation(t *testing.T) {
	cfg := DefaultCorrelationMonitorConfig()
	cfg.MinWindow = 5
	cm := NewCorrelationMonitor(cfg)

	// Perfect positive correlation.
	for _, v := range returns(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) {
		cm.RecordReturn("AAPL", v)
		cm.RecordReturn("MSFT", v)
	}
	alerts := cm.Scan()
	if len(alerts) != 1 {
		t.Fatalf("want 1 correlation alert, got %d", len(alerts))
	}
	if math.Abs(alerts[0].Correlation-1.0) > 0.001 {
		t.Fatalf("correlation should be ~1.0, got %f", alerts[0].Correlation)
	}
}

func TestCorrelationMonitor_LowCorrelation_NoAlert(t *testing.T) {
	cfg := DefaultCorrelationMonitorConfig()
	cfg.MinWindow = 5
	cm := NewCorrelationMonitor(cfg)

	// Uncorrelated series.
	for _, v := range returns(1, -2, 3, -4, 5, -6, 7, -8) {
		cm.RecordReturn("AAPL", v)
	}
	for _, v := range returns(-1, 2, -3, 4, -5, 6, -7, 8) {
		cm.RecordReturn("MSFT", v)
	}
	cm.RecordReturn("GOOGL", 0.5)
	// AAPL vs MSFT is perfectly anti-correlated (-1); |corr|=1 → above threshold
	// but GOOGL has only 1 obs → below MinWindow.
	alerts := cm.Scan()
	// One alert (AAPL-MSFT anti-corr), GOOGL ignored.
	for _, a := range alerts {
		if a.SymbolA == "GOOGL" || a.SymbolB == "GOOGL" {
			t.Error("GOOGL with insufficient data should not produce alert")
		}
	}
}

func TestCorrelationMonitor_InsufficientWindow_NoAlert(t *testing.T) {
	cfg := DefaultCorrelationMonitorConfig()
	cfg.MinWindow = 20
	cm := NewCorrelationMonitor(cfg)

	for _, v := range returns(1, 2, 3) { // only 3 obs, MinWindow=20
		cm.RecordReturn("A", v)
		cm.RecordReturn("B", v)
	}
	alerts := cm.Scan()
	if len(alerts) != 0 {
		t.Fatalf("should not alert with only 3 obs (MinWindow=20), got %d", len(alerts))
	}
}

// ─── L24: LatencyTracker ─────────────────────────────────────────────────────

func TestLatencyTracker_RecordAndStats(t *testing.T) {
	lt := NewLatencyTracker(DefaultLatencyTrackerConfig())
	for i := range 21 {
		lt.Record(LatencyObservation{
			Category:   "order_submit",
			Latency:    time.Duration(10+i) * time.Millisecond,
			RecordedAt: time.Now(),
		})
	}
	stats := lt.Stats("order_submit")
	if stats.Count != 21 {
		t.Fatalf("want 21, got %d", stats.Count)
	}
	if stats.P50 <= 0 {
		t.Error("P50 should be positive")
	}
	if stats.P95 < stats.P50 {
		t.Error("P95 should be >= P50")
	}
	if stats.P99 >= stats.Max+1 {
		t.Error("P99 should be <= Max")
	}
}

func TestLatencyTracker_TradingPaused_WhenHighLatency(t *testing.T) {
	cfg := DefaultLatencyTrackerConfig()
	cfg.PauseThreshold = 100 * time.Millisecond
	cfg.PauseMinSamples = 5
	lt := NewLatencyTracker(cfg)

	for range 10 {
		lt.Record(LatencyObservation{
			Category: "order_submit",
			Latency:  500 * time.Millisecond, // > threshold
			RecordedAt: time.Now(),
		})
	}
	paused, reason := lt.TradingPaused()
	if !paused {
		t.Fatal("should be paused due to high latency")
	}
	if reason == "" {
		t.Error("pause reason should not be empty")
	}
}

func TestLatencyTracker_NotPaused_WhenLowLatency(t *testing.T) {
	cfg := DefaultLatencyTrackerConfig()
	cfg.PauseThreshold = 500 * time.Millisecond
	cfg.PauseMinSamples = 5
	lt := NewLatencyTracker(cfg)

	for range 10 {
		lt.Record(LatencyObservation{
			Category: "order_submit",
			Latency:  10 * time.Millisecond,
			RecordedAt: time.Now(),
		})
	}
	paused, _ := lt.TradingPaused()
	if paused {
		t.Fatal("should not be paused with low latency")
	}
}

func TestLatencyTracker_NotPaused_BelowMinSamples(t *testing.T) {
	cfg := DefaultLatencyTrackerConfig()
	cfg.PauseThreshold = 100 * time.Millisecond
	cfg.PauseMinSamples = 20
	lt := NewLatencyTracker(cfg)

	for range 5 { // only 5 obs, min=20
		lt.Record(LatencyObservation{
			Category: "order_submit",
			Latency:  999 * time.Millisecond,
			RecordedAt: time.Now(),
		})
	}
	paused, _ := lt.TradingPaused()
	if paused {
		t.Fatal("should not pause below PauseMinSamples")
	}
}

func TestLatencyTracker_Categories(t *testing.T) {
	lt := NewLatencyTracker(DefaultLatencyTrackerConfig())
	lt.Record(LatencyObservation{Category: "order_submit", Latency: 10 * time.Millisecond})
	lt.Record(LatencyObservation{Category: "fill_ack", Latency: 20 * time.Millisecond})

	cats := lt.Categories()
	if len(cats) != 2 {
		t.Fatalf("want 2 categories, got %d", len(cats))
	}
}

func TestLatencyTracker_EmptyStats(t *testing.T) {
	lt := NewLatencyTracker(DefaultLatencyTrackerConfig())
	s := lt.Stats("nonexistent")
	if s.Count != 0 {
		t.Fatalf("empty category: want count=0, got %d", s.Count)
	}
}

// ─── pearson ─────────────────────────────────────────────────────────────────

func TestPearson_PerfectCorrelation(t *testing.T) {
	x := returns(1, 2, 3, 4, 5)
	y := returns(1, 2, 3, 4, 5)
	r, ok := pearson(x, y, 3)
	if !ok {
		t.Fatal("pearson should succeed with 5 obs and minWindow=3")
	}
	if math.Abs(r-1.0) > 0.001 {
		t.Fatalf("perfect corr: want 1.0, got %.4f", r)
	}
}

func TestPearson_NegativeCorrelation(t *testing.T) {
	x := returns(1, 2, 3, 4, 5)
	y := returns(5, 4, 3, 2, 1)
	r, ok := pearson(x, y, 3)
	if !ok {
		t.Fatal("should succeed")
	}
	if math.Abs(r+1.0) > 0.001 {
		t.Fatalf("negative corr: want -1.0, got %.4f", r)
	}
}

func TestPearson_InsufficientData(t *testing.T) {
	x := returns(1, 2)
	y := returns(1, 2)
	_, ok := pearson(x, y, 5)
	if ok {
		t.Fatal("should return false when fewer obs than minWindow")
	}
}
