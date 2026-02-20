package observability

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// ─── Registry / WriteText ─────────────────────────────────────────────────────

func TestRegistry_WriteText_Empty(t *testing.T) {
	r := NewRegistry()
	var buf bytes.Buffer
	r.WriteText(&buf)
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %s", buf.String())
	}
}

// ─── Counter ─────────────────────────────────────────────────────────────────

func TestCounter_Inc(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_counter", "test help")
	c.Inc()
	c.Inc()
	if v := c.Value(); v != 2 {
		t.Errorf("expected 2, got %f", v)
	}
}

func TestCounter_Add(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_add", "help")
	c.Add(5)
	c.Add(3)
	if v := c.Value(); v != 8 {
		t.Errorf("expected 8, got %f", v)
	}
}

func TestCounter_NegativeDelta_Ignored(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_neg", "help")
	c.Add(10)
	c.Add(-5) // should be ignored
	if v := c.Value(); v != 10 {
		t.Errorf("expected 10 (negative ignored), got %f", v)
	}
}

func TestCounter_WithLabels(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("signals", "signals by strategy")
	c.Inc("strategy", "rsi_v1", "direction", "buy")
	c.Inc("strategy", "rsi_v1", "direction", "buy")
	c.Inc("strategy", "macd_v1", "direction", "sell")

	if v := c.Value("strategy", "rsi_v1", "direction", "buy"); v != 2 {
		t.Errorf("expected 2 for rsi_v1/buy, got %f", v)
	}
	if v := c.Value("strategy", "macd_v1", "direction", "sell"); v != 1 {
		t.Errorf("expected 1 for macd_v1/sell, got %f", v)
	}
	if v := c.Value("strategy", "unknown", "direction", "buy"); v != 0 {
		t.Errorf("expected 0 for unknown, got %f", v)
	}
}

func TestCounter_WriteText(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("http_requests_total", "Total HTTP requests")
	c.Inc("method", "GET")
	c.Inc("method", "GET")
	c.Inc("method", "POST")

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()

	assertContains(t, out, "# HELP http_requests_total Total HTTP requests")
	assertContains(t, out, "# TYPE http_requests_total counter")
	assertContains(t, out, `http_requests_total{method="GET"} 2`)
	assertContains(t, out, `http_requests_total{method="POST"} 1`)
}

func TestCounter_Concurrent(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("concurrent_counter", "concurrent test")

	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()

	if v := c.Value(); v != float64(n) {
		t.Errorf("expected %d, got %f", n, v)
	}
}

// ─── Gauge ───────────────────────────────────────────────────────────────────

func TestGauge_Set(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("equity", "account equity")
	g.Set(100_000)
	if v := g.Value(); v != 100_000 {
		t.Errorf("expected 100000, got %f", v)
	}
	g.Set(99_500)
	if v := g.Value(); v != 99_500 {
		t.Errorf("expected 99500, got %f", v)
	}
}

func TestGauge_Add(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("positions", "open positions")
	g.Set(3)
	g.Add(2)
	if v := g.Value(); v != 5 {
		t.Errorf("expected 5, got %f", v)
	}
	g.Add(-1)
	if v := g.Value(); v != 4 {
		t.Errorf("expected 4, got %f", v)
	}
}

func TestGauge_WithLabels(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("price", "price by symbol")
	g.Set(150.0, "symbol", "AAPL")
	g.Set(200.0, "symbol", "MSFT")

	if v := g.Value("symbol", "AAPL"); v != 150.0 {
		t.Errorf("expected 150, got %f", v)
	}
	if v := g.Value("symbol", "MSFT"); v != 200.0 {
		t.Errorf("expected 200, got %f", v)
	}
}

func TestGauge_WriteText(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("jax_equity", "Account equity")
	g.Set(100000.5)

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()

	assertContains(t, out, "# HELP jax_equity Account equity")
	assertContains(t, out, "# TYPE jax_equity gauge")
	assertContains(t, out, "jax_equity 100000.5")
}

// ─── Histogram ───────────────────────────────────────────────────────────────

func TestHistogram_Observe(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("latency", "latency in seconds", []float64{0.01, 0.1, 1.0})

	// Cumulative buckets: each counts all observations <= upper bound.
	h.Observe(0.005) // ≤0.01 ≤0.1 ≤1.0 ≤+Inf
	h.Observe(0.05)  //       ≤0.1 ≤1.0 ≤+Inf
	h.Observe(0.5)   //            ≤1.0 ≤+Inf
	h.Observe(2.0)   //                 ≤+Inf

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()

	assertContains(t, out, `latency_bucket{le="0.01"} 1`)  // only 0.005
	assertContains(t, out, `latency_bucket{le="0.1"} 2`)   // 0.005 + 0.05 (cumulative)
	assertContains(t, out, `latency_bucket{le="1"} 3`)     // 0.005 + 0.05 + 0.5 (cumulative)
	assertContains(t, out, `latency_bucket{le="+Inf"} 4`)  // all 4
	assertContains(t, out, `latency_count 4`)
}

func TestHistogram_ObserveDuration(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("fill_latency", "fill latency", DefaultBuckets)
	h.ObserveDuration(25 * time.Millisecond)
	h.ObserveDuration(75 * time.Millisecond)

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()
	assertContains(t, out, "fill_latency_count 2")
}

func TestHistogram_WithLabels(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("slippage", "slippage bps", []float64{1, 5, 10})
	h.Observe(3, "symbol", "AAPL")
	h.Observe(8, "symbol", "AAPL")
	h.Observe(1, "symbol", "MSFT")

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()

	assertContains(t, out, `slippage_count{symbol="AAPL"} 2`)
	assertContains(t, out, `slippage_count{symbol="MSFT"} 1`)
}

func TestHistogram_NilBounds_UsesDefault(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("default_hist", "test", nil)
	h.Observe(0.5)

	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()
	assertContains(t, out, "default_hist_count 1")
}

// ─── Labels ───────────────────────────────────────────────────────────────────

func TestLabels_Format(t *testing.T) {
	l := NewLabels("method", "GET", "status", "200")
	got := l.format()
	want := `{method="GET",status="200"}`
	if got != want {
		t.Errorf("want %s, got %s", want, got)
	}

	// Empty labels.
	empty := Labels(nil)
	if f := empty.format(); f != "" {
		t.Errorf("expected empty format, got %s", f)
	}
}

func TestLabels_QuoteEscape(t *testing.T) {
	l := NewLabels("msg", `say "hi"`)
	got := l.format()
	if !strings.Contains(got, `\"hi\"`) {
		t.Errorf("expected escaped quotes in %s", got)
	}
}

// ─── TradingMetrics ───────────────────────────────────────────────────────────

func TestTradingMetrics_Wiring(t *testing.T) {
	reg := NewRegistry()
	tm := NewTradingMetrics(reg)

	tm.SignalsPublished.Inc("strategy", "rsi_v1", "direction", "buy")
	tm.GateDecisions.Inc("decision", "allow")
	tm.FillLatency.ObserveDuration(15 * time.Millisecond)
	tm.Equity.Set(102_500.0)
	tm.ActivePositions.Set(2)
	tm.HaltEvents.Inc("reason", "feed_failure")
	tm.SlippageBps.Observe(3.5)

	var buf bytes.Buffer
	reg.WriteText(&buf)
	out := buf.String()

	assertContains(t, out, "jax_signals_published_total")
	assertContains(t, out, "jax_gate_decisions_total")
	assertContains(t, out, "jax_fill_latency_seconds")
	assertContains(t, out, "jax_account_equity 102500")
	assertContains(t, out, "jax_active_positions 2")
	assertContains(t, out, "jax_guardrail_halt_total")
	assertContains(t, out, "jax_slippage_bps")
}

// ─── formatFloat ─────────────────────────────────────────────────────────────

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{1.0, "1"},
		{0.5, "0.5"},
		{100000.5, "100000.5"},
	}
	for _, tc := range cases {
		got := formatFloat(tc.in)
		if got != tc.want {
			t.Errorf("formatFloat(%f) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func assertContains(t testing.TB, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("expected output to contain:\n  %q\ngot:\n%s", sub, s)
	}
}
