package eventtrader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/libs/calendar"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

// buildStore creates a calendar.Store populated with the given events.
func buildStore(t *testing.T, events []calendar.EconEvent) *calendar.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := calendar.OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(events); err != nil {
		t.Fatal(err)
	}
	return store
}

func nfpAt(ts time.Time) calendar.EconEvent {
	return calendar.EconEvent{
		ID: calendar.EventID("US", "Non-Farm Payrolls", ts),
		Country: "US", Currency: "USD",
		Title: "Non-Farm Payrolls", Impact: calendar.ImpactHigh,
		ScheduledAt: ts, Source: "test",
	}
}

// ─── PhaseDetector ────────────────────────────────────────────────────────────

func newDetector(t *testing.T, events []calendar.EconEvent) *PhaseDetector {
	t.Helper()
	store := buildStore(t, events)
	return NewPhaseDetector(store, DefaultPhaseDetectorConfig())
}

func TestPhaseDetector_Normal(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	d := newDetector(t, []calendar.EconEvent{nfpAt(eventAt)})

	// 2 hours before — still outside pre-event window (60 min)
	now := eventAt.Add(-2 * time.Hour)
	pr := d.Detect(now, "USD")
	if pr.Phase != PhaseNormal {
		t.Fatalf("want PhaseNormal 2h before, got %q", pr.Phase)
	}
}

func TestPhaseDetector_PreEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	d := newDetector(t, []calendar.EconEvent{nfpAt(eventAt)})

	// 30 min before — inside pre-event window (60 min), outside blackout (10 min)
	now := eventAt.Add(-30 * time.Minute)
	pr := d.Detect(now, "USD")
	if pr.Phase != PhasePreEvent {
		t.Fatalf("want PhasePreEvent 30m before, got %q", pr.Phase)
	}
	if pr.Event == nil {
		t.Fatal("PhaseResult.Event should not be nil")
	}
}

func TestPhaseDetector_Blackout(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	d := newDetector(t, []calendar.EconEvent{nfpAt(eventAt)})

	// 5 min before — inside blackout (10 min)
	now := eventAt.Add(-5 * time.Minute)
	pr := d.Detect(now, "USD")
	if pr.Phase != PhaseBlackout {
		t.Fatalf("want PhaseBlackout 5m before, got %q", pr.Phase)
	}
}

func TestPhaseDetector_PostEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	d := newDetector(t, []calendar.EconEvent{nfpAt(eventAt)})

	// 15 min after — inside post-event window (30 min)
	now := eventAt.Add(15 * time.Minute)
	pr := d.Detect(now, "USD")
	if pr.Phase != PhasePostEvent {
		t.Fatalf("want PhasePostEvent 15m after, got %q", pr.Phase)
	}
}

func TestPhaseDetector_NormalAfterPostEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	d := newDetector(t, []calendar.EconEvent{nfpAt(eventAt)})

	// 45 min after — outside post-event window (30 min)
	now := eventAt.Add(45 * time.Minute)
	pr := d.Detect(now, "USD")
	if pr.Phase != PhaseNormal {
		t.Fatalf("want PhaseNormal 45m after, got %q", pr.Phase)
	}
}

func TestPhaseDetector_BlackoutBeatsPreEvent(t *testing.T) {
	// Two events: one in pre-event, one in blackout. Detector should return blackout.
	t1 := mustTime("2024-01-05T13:30:00Z") // 5 min away = blackout
	t2 := mustTime("2024-01-05T14:15:00Z") // 50 min away = pre-event

	e1 := nfpAt(t1)
	e2 := calendar.EconEvent{
		ID: calendar.EventID("US", "CPI", t2),
		Country: "US", Currency: "USD", Title: "CPI",
		Impact: calendar.ImpactHigh, ScheduledAt: t2, Source: "test",
	}
	d := newDetector(t, []calendar.EconEvent{e1, e2})

	now := mustTime("2024-01-05T13:25:00Z") // 5 min before t1
	pr := d.Detect(now, "USD")
	if pr.Phase != PhaseBlackout {
		t.Fatalf("want Blackout (higher priority), got %q", pr.Phase)
	}
}

func TestPhaseDetector_UnrelatedCurrencyIgnored(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	e := nfpAt(eventAt) // USD event
	d := newDetector(t, []calendar.EconEvent{e})

	// Querying for EUR only — should not trigger USD event.
	now := eventAt.Add(-5 * time.Minute)
	pr := d.Detect(now, "EUR")
	if pr.Phase != PhaseNormal {
		t.Fatalf("want PhaseNormal for EUR (USD event), got %q", pr.Phase)
	}
}

// ─── EventGate ────────────────────────────────────────────────────────────────

func newGate(t *testing.T, events []calendar.EconEvent, cfg EventGateConfig) *EventGate {
	t.Helper()
	d := newDetector(t, events)
	return NewEventGate(d, cfg)
}

func TestEventGate_BlockOnBlackout(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	gate := newGate(t, []calendar.EconEvent{nfpAt(eventAt)}, DefaultEventGateConfig())

	now := eventAt.Add(-5 * time.Minute) // blackout
	res := gate.Check("rsi_momentum_v1", now, "USD")
	if res.Decision != GateBlock {
		t.Fatalf("want GateBlock, got %q", res.Decision)
	}
}

func TestEventGate_HoldOnPreEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	gate := newGate(t, []calendar.EconEvent{nfpAt(eventAt)}, DefaultEventGateConfig())

	now := eventAt.Add(-30 * time.Minute) // pre-event
	res := gate.Check("rsi_momentum_v1", now, "USD")
	if res.Decision != GateHold {
		t.Fatalf("want GateHold, got %q", res.Decision)
	}
}

func TestEventGate_AllowNormal(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	gate := newGate(t, []calendar.EconEvent{nfpAt(eventAt)}, DefaultEventGateConfig())

	now := eventAt.Add(-2 * time.Hour) // normal
	res := gate.Check("rsi_momentum_v1", now, "USD")
	if res.Decision != GateAllow {
		t.Fatalf("want GateAllow, got %q", res.Decision)
	}
}

func TestEventGate_AllowListedStrategyPassesBlackout(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	cfg := DefaultEventGateConfig()
	cfg.AllowStrategies = []string{"news_fade_v1"}
	gate := newGate(t, []calendar.EconEvent{nfpAt(eventAt)}, cfg)

	now := eventAt.Add(-2 * time.Minute) // deep blackout
	res := gate.Check("news_fade_v1", now, "USD")
	if res.Decision != GateAllow {
		t.Fatalf("allow-listed strategy should pass blackout, got %q", res.Decision)
	}
}

func TestEventGate_NonAllowListedBlocked(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	cfg := DefaultEventGateConfig()
	cfg.AllowStrategies = []string{"news_fade_v1"}
	gate := newGate(t, []calendar.EconEvent{nfpAt(eventAt)}, cfg)

	now := eventAt.Add(-2 * time.Minute) // deep blackout
	res := gate.Check("rsi_momentum_v1", now, "USD")
	if res.Decision != GateBlock {
		t.Fatalf("non-allow-listed strategy should be blocked, got %q", res.Decision)
	}
}

// ─── EventBucketizer ─────────────────────────────────────────────────────────

func TestEventBucketizer_BasketByEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	store := buildStore(t, []calendar.EconEvent{nfpAt(eventAt)})
	detector := NewPhaseDetector(store, DefaultPhaseDetectorConfig())
	bk := NewEventBucketizer(detector)

	ctx := context.Background()

	// Signal 30 min before NFP → should land in NFP bucket
	s1 := Signal{ID: "s1", StrategyID: "rsi_v1", Symbol: "EURUSD",
		Currency: "USD", GeneratedAt: eventAt.Add(-30 * time.Minute)}
	bucket1 := bk.Classify(ctx, s1)
	if bucket1.EventID == "baseline" {
		t.Fatal("signal near NFP should be in NFP bucket, not baseline")
	}

	// Second signal for same event
	s2 := Signal{ID: "s2", StrategyID: "rsi_v1", Symbol: "GBPUSD",
		Currency: "USD", GeneratedAt: eventAt.Add(-20 * time.Minute)}
	bucket2 := bk.Classify(ctx, s2)
	if bucket1.EventID != bucket2.EventID {
		t.Fatal("both USD signals near NFP should share the same bucket")
	}
	if len(bucket1.Signals) != 2 {
		t.Fatalf("bucket should have 2 signals, got %d", len(bucket1.Signals))
	}
}

func TestEventBucketizer_BaselineBucketWhenNoEvent(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	store := buildStore(t, []calendar.EconEvent{nfpAt(eventAt)})
	detector := NewPhaseDetector(store, DefaultPhaseDetectorConfig())
	bk := NewEventBucketizer(detector)

	ctx := context.Background()
	// 3 hours before — no event in window
	s := Signal{ID: "s1", StrategyID: "rsi_v1", Symbol: "EURUSD",
		Currency: "USD", GeneratedAt: eventAt.Add(-3 * time.Hour)}
	bucket := bk.Classify(ctx, s)
	if bucket.EventID != "baseline" {
		t.Fatalf("want baseline bucket, got %q", bucket.EventID)
	}
}

func TestEventBucketizer_Reset(t *testing.T) {
	eventAt := mustTime("2024-01-05T13:30:00Z")
	store := buildStore(t, []calendar.EconEvent{nfpAt(eventAt)})
	detector := NewPhaseDetector(store, DefaultPhaseDetectorConfig())
	bk := NewEventBucketizer(detector)

	ctx := context.Background()
	s := Signal{ID: "s1", StrategyID: "rsi_v1", Symbol: "EURUSD",
		Currency: "USD", GeneratedAt: eventAt.Add(-30 * time.Minute)}
	bk.Classify(ctx, s)
	if len(bk.Buckets()) == 0 {
		t.Fatal("expected at least one bucket")
	}

	bk.Reset()
	if len(bk.Buckets()) != 0 {
		t.Fatalf("expected empty buckets after reset, got %d", len(bk.Buckets()))
	}
}

// ─── phaseRank ordering ───────────────────────────────────────────────────────

func TestPhaseRank_Order(t *testing.T) {
	if phaseRank(PhaseBlackout) <= phaseRank(PhasePostEvent) {
		t.Error("Blackout should outrank PostEvent")
	}
	if phaseRank(PhasePostEvent) <= phaseRank(PhasePreEvent) {
		t.Error("PostEvent should outrank PreEvent")
	}
	if phaseRank(PhasePreEvent) <= phaseRank(PhaseNormal) {
		t.Error("PreEvent should outrank Normal")
	}
}

// Ensure the package compiles even without a live CSV on disk.
func TestCSVSourceCompileCheck(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ff.csv")
	_ = os.WriteFile(p, []byte("date,time,currency,impact,event,actual,forecast,previous\n"), 0o644)
	src := calendar.NewCSVSource("test", p)
	if src.Name() != "test" {
		t.Fatalf("unexpected name: %s", src.Name())
	}
}
