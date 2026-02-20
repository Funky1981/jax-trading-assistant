package replay

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

func newStore(t *testing.T) *TraceStore {
	t.Helper()
	ts, err := OpenTraceStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

// ─── TraceStore ───────────────────────────────────────────────────────────────

func TestTraceStore_AppendAndReadAll(t *testing.T) {
	ts := newStore(t)

	e1 := TraceEntry{StrategyID: "rsi_v1", Symbol: "AAPL",
		SignalPrice: 180.5, StopLoss: 178.0, PositionSize: 10,
		Decision: DecisionEmit, OrderID: "ord-001"}
	e2 := TraceEntry{StrategyID: "rsi_v1", Symbol: "MSFT",
		SignalPrice: 320.0, StopLoss: 315.0, PositionSize: 5,
		Decision: DecisionBlock, Reason: "blackout"}

	got1, err := ts.Append(e1)
	if err != nil {
		t.Fatal(err)
	}
	if got1.Sequence != 1 {
		t.Fatalf("want Sequence=1, got %d", got1.Sequence)
	}

	got2, err := ts.Append(e2)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Sequence != 2 {
		t.Fatalf("want Sequence=2, got %d", got2.Sequence)
	}

	all, err := ts.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 entries, got %d", len(all))
	}
	if all[0].Symbol != "AAPL" || all[1].Symbol != "MSFT" {
		t.Fatalf("unexpected order: %v", all)
	}
}

func TestTraceStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ts, err := OpenTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "AAPL",
		Decision: DecisionEmit, OrderID: "ord-001"})
	if err != nil {
		t.Fatal(err)
	}

	// Reopen from same dir.
	ts2, err := OpenTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	all, err := ts2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 entry after reload, got %d", len(all))
	}
	if ts2.seq != 1 {
		t.Fatalf("want seq=1 after reload, got %d", ts2.seq)
	}
}

func TestTraceStore_SequenceMonotonic(t *testing.T) {
	ts := newStore(t)
	for i := range 5 {
		e := TraceEntry{StrategyID: "rsi_v1", Symbol: fmt.Sprintf("SYM%d", i),
			Decision: DecisionEmit}
		got, err := ts.Append(e)
		if err != nil {
			t.Fatal(err)
		}
		if got.Sequence != uint64(i+1) {
			t.Fatalf("step %d: want seq=%d, got %d", i, i+1, got.Sequence)
		}
	}
}

func TestTraceStore_Filter_ByDecision(t *testing.T) {
	ts := newStore(t)
	_, _ = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "AAPL", Decision: DecisionEmit})
	_, _ = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "MSFT", Decision: DecisionBlock})
	_, _ = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "GOOGL", Decision: DecisionEmit})

	emitted, err := ts.Filter("", "", DecisionEmit)
	if err != nil {
		t.Fatal(err)
	}
	if len(emitted) != 2 {
		t.Fatalf("want 2 emitted, got %d", len(emitted))
	}
	blocked, err := ts.Filter("", "", DecisionBlock)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked) != 1 {
		t.Fatalf("want 1 blocked, got %d", len(blocked))
	}
}

func TestTraceStore_Filter_BySymbol(t *testing.T) {
	ts := newStore(t)
	_, _ = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "AAPL", Decision: DecisionEmit})
	_, _ = ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "MSFT", Decision: DecisionEmit})

	got, err := ts.Filter("", "AAPL", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Symbol != "AAPL" {
		t.Fatalf("filter by symbol failed: %+v", got)
	}
}

func TestTraceStore_RecordedAtIsSet(t *testing.T) {
	ts := newStore(t)
	before := time.Now().UTC().Add(-time.Second)
	got, _ := ts.Append(TraceEntry{StrategyID: "rsi_v1", Symbol: "X", Decision: DecisionEmit})
	after := time.Now().UTC().Add(time.Second)
	if got.RecordedAt.Before(before) || got.RecordedAt.After(after) {
		t.Fatalf("RecordedAt out of range: %v", got.RecordedAt)
	}
}

// ─── SimBroker ────────────────────────────────────────────────────────────────

func TestSimBroker_MarketBuyFill(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	order := SimOrder{
		ID:    "ord-001",
		Symbol: "AAPL",
		Side:  SideBuy,
		Type:  OrderMarket,
		Quantity: 10,
		SubmittedAt: mustTime("2024-01-05T09:30:00Z"),
	}
	broker.SubmitOrder(order)

	candle := Candle{
		Timestamp: mustTime("2024-01-05T09:31:00Z"),
		Open: 180.0, High: 181.0, Low: 179.0, Close: 180.5, Volume: 10000,
	}
	fills := broker.ProcessCandle(candle)
	if len(fills) != 1 {
		t.Fatalf("want 1 fill, got %d", len(fills))
	}
	f := fills[0]
	if f.Symbol != "AAPL" || f.Side != SideBuy {
		t.Fatalf("unexpected fill: %+v", f)
	}
	// Fill price should be open + slippage for buy.
	expectedPrice := 180.0 * (1 + 5.0/10_000)
	if math.Abs(f.FillPrice-expectedPrice) > 0.01 {
		t.Fatalf("fill price: want ~%.4f, got %.4f", expectedPrice, f.FillPrice)
	}
}

func TestSimBroker_LimitOrderNotTriggered(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	order := SimOrder{
		ID:          "ord-001",
		Symbol:      "AAPL",
		Side:        SideBuy,
		Type:        OrderLimit,
		Quantity:    10,
		LimitPrice:  175.0, // well below current market
		SubmittedAt: mustTime("2024-01-05T09:30:00Z"),
	}
	broker.SubmitOrder(order)

	// Candle low is 179 — does not reach limit 175.
	candle := Candle{
		Timestamp: mustTime("2024-01-05T09:31:00Z"),
		Open: 180.0, High: 181.0, Low: 179.0, Close: 180.5,
	}
	fills := broker.ProcessCandle(candle)
	if len(fills) != 0 {
		t.Fatalf("limit not triggered: want 0 fills, got %d", len(fills))
	}
}

func TestSimBroker_LimitOrderTriggered(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	order := SimOrder{
		ID:          "ord-001",
		Symbol:      "AAPL",
		Side:        SideBuy,
		Type:        OrderLimit,
		Quantity:    10,
		LimitPrice:  178.0,
		SubmittedAt: mustTime("2024-01-05T09:30:00Z"),
	}
	broker.SubmitOrder(order)

	// Low dips to 177 — limit hit.
	candle := Candle{
		Timestamp: mustTime("2024-01-05T09:31:00Z"),
		Open: 180.0, High: 181.0, Low: 177.0, Close: 180.5,
	}
	fills := broker.ProcessCandle(candle)
	if len(fills) != 1 {
		t.Fatalf("limit triggered: want 1 fill, got %d", len(fills))
	}
}

func TestSimBroker_StopOrderTriggered(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	// Protective stop on a short: buy-stop triggers when price breaks up.
	order := SimOrder{
		ID:        "ord-001",
		Symbol:    "AAPL",
		Side:      SideBuy,
		Type:      OrderStop,
		Quantity:  10,
		StopPrice: 182.0,
		SubmittedAt: mustTime("2024-01-05T09:30:00Z"),
	}
	broker.SubmitOrder(order)

	candle := Candle{
		Timestamp: mustTime("2024-01-05T09:31:00Z"),
		Open: 180.0, High: 183.0, Low: 179.0, Close: 182.5,
	}
	fills := broker.ProcessCandle(candle)
	if len(fills) != 1 {
		t.Fatalf("stop triggered: want 1 fill, got %d", len(fills))
	}
}

func TestSimBroker_EquityTracking(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 10_000)
	if broker.Equity() != 10_000 {
		t.Fatalf("initial equity should be 10000")
	}

	broker.SubmitOrder(SimOrder{ID: "o1", Symbol: "AAPL",
		Side: SideBuy, Type: OrderMarket, Quantity: 10})
	broker.ProcessCandle(Candle{
		Symbol: "AAPL",
		Timestamp: mustTime("2024-01-05T09:31:00Z"),
		Open: 100.0, High: 102.0, Low: 99.0, Close: 101.0,
	})
	// Equity should reflect position marked to close.
	eq := broker.Equity()
	if eq <= 0 || eq > 20_000 {
		t.Fatalf("equity unreasonable: %f", eq)
	}
}

// ─── Simulator ───────────────────────────────────────────────────────────────

func TestSimulator_BasicRun(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)

	// 5 daily candles for AAPL.
	baseTime := mustTime("2024-01-02T14:30:00Z")
	var candles []Candle
	for i := range 5 {
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Open:      float64(180 + i),
			High:      float64(182 + i),
			Low:       float64(179 + i),
			Close:     float64(181 + i),
			Volume:    100_000,
		})
	}

	signals := []PlaybackSignal{
		{OrderID: "buy-1", StrategyID: "rsi_v1", Symbol: "AAPL",
			Side: SideBuy, Quantity: 10,
			EventAt: baseTime, // triggers on first candle
		},
		{OrderID: "sell-1", StrategyID: "rsi_v1", Symbol: "AAPL",
			Side: SideSell, Quantity: 10,
			EventAt: baseTime.Add(3 * 24 * time.Hour), // triggers on candle 4
		},
	}

	sim := NewSimulator(broker, nil)
	result, err := sim.Run(context.Background(), signals, candles)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalTrades != 2 {
		t.Fatalf("want 2 trades, got %d", result.TotalTrades)
	}
	if result.FinalEquity <= 0 {
		t.Fatalf("final equity should be positive, got %f", result.FinalEquity)
	}
}

func TestSimulator_TraceRecording(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	trace := newStore(t)

	candles := []Candle{
		{Timestamp: mustTime("2024-01-02T14:30:00Z"),
			Open: 180, High: 182, Low: 179, Close: 181, Volume: 100000},
	}
	signals := []PlaybackSignal{
		{OrderID: "buy-1", StrategyID: "rsi_v1", Symbol: "AAPL",
			Side: SideBuy, Quantity: 10,
			EventAt: mustTime("2024-01-02T14:30:00Z")},
	}

	sim := NewSimulator(broker, trace)
	_, err := sim.Run(context.Background(), signals, candles)
	if err != nil {
		t.Fatal(err)
	}

	all, err := trace.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 trace entry, got %d", len(all))
	}
	if all[0].Decision != DecisionEmit {
		t.Fatalf("want DecisionEmit, got %q", all[0].Decision)
	}
}

func TestSimulator_NoCandles(t *testing.T) {
	broker := NewSimBroker(DefaultSimBrokerConfig(), 100_000)
	sim := NewSimulator(broker, nil)
	_, err := sim.Run(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for empty candles")
	}
}

// ─── buildResult ─────────────────────────────────────────────────────────────

func TestBuildResult_WinRate(t *testing.T) {
	fills := []Fill{
		{Symbol: "AAPL", Side: SideBuy, Quantity: 10, FillPrice: 100},
		{Symbol: "AAPL", Side: SideSell, Quantity: 10, FillPrice: 110}, // +100 win
		{Symbol: "MSFT", Side: SideBuy, Quantity: 5, FillPrice: 300},
		{Symbol: "MSFT", Side: SideSell, Quantity: 5, FillPrice: 290}, // -50 loss
	}
	r := buildResult(fills, 100_000, 100_050)
	if r.WinTrades != 1 || r.LossTrades != 1 {
		t.Fatalf("win=%d loss=%d", r.WinTrades, r.LossTrades)
	}
	if r.WinRate != 0.5 {
		t.Fatalf("want win rate 0.5, got %f", r.WinRate)
	}
}


