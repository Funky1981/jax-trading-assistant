// prometheus.go implements a zero-dependency Prometheus text-format metrics
// registry suitable for the EJLayer trading system.
//
// It produces valid Prometheus exposition format (text/plain; version=0.0.4)
// and can be served directly from any HTTP handler via [Registry.WriteText].
//
// # Metric types supported
//   - Counter   — monotonically increasing float64
//   - Gauge     — arbitrary float64, can go up or down
//   - Histogram — exponential-bucket observations with count/sum/_bucket
//
// All types support label sets.  Thread-safe.
package observability

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─── Registry ────────────────────────────────────────────────────────────────

// Registry is the root metrics registry.  Create one per process (or per test).
// The zero-value is not valid; use [NewRegistry].
type Registry struct {
	mu      sync.RWMutex
	metrics []metric
}

// metric is the internal interface every collector satisfies.
type metric interface {
	desc() metricDesc
	writeText(w io.Writer)
}

type metricDesc struct {
	name   string
	help   string
	mtype  string // "counter", "gauge", "histogram"
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{} }

// WriteText writes all registered metrics in Prometheus text format to w.
func (r *Registry) WriteText(w io.Writer) {
	r.mu.RLock()
	ms := append([]metric(nil), r.metrics...)
	r.mu.RUnlock()

	for _, m := range ms {
		d := m.desc()
		fmt.Fprintf(w, "# HELP %s %s\n", d.name, d.help)
		fmt.Fprintf(w, "# TYPE %s %s\n", d.name, d.mtype)
		m.writeText(w)
	}
}

func (r *Registry) register(m metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, m)
}

// ─── Labels ──────────────────────────────────────────────────────────────────

// Labels is an ordered list of key=value pairs attached to a metric sample.
type Labels []string // alternating key, value

// NewLabels constructs Labels from alternating key-value pairs.
// Panics if pairs are not even.
func NewLabels(kv ...string) Labels {
	if len(kv)%2 != 0 {
		panic("observability: NewLabels requires even number of arguments")
	}
	return Labels(kv)
}

// format renders the label set as {k="v",...} or empty string.
func (l Labels) format() string {
	if len(l) == 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteByte('{')
	for i := 0; i < len(l); i += 2 {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(l[i])
		sb.WriteString(`="`)
		sb.WriteString(strings.ReplaceAll(l[i+1], `"`, `\"`))
		sb.WriteByte('"')
	}
	sb.WriteByte('}')
	return sb.String()
}

// labelKey returns a map-friendly key string.
func (l Labels) labelKey() string { return strings.Join(l, "\x00") }

// ─── Counter ─────────────────────────────────────────────────────────────────

// Counter is a monotonically increasing metric.
type Counter struct {
	d    metricDesc
	mu   sync.RWMutex
	rows map[string]counterRow
}

type counterRow struct {
	labels Labels
	value  uint64 // stored as bits of float64 for atomic ops
}

// NewCounter registers and returns a new Counter.
func (r *Registry) NewCounter(name, help string) *Counter {
	c := &Counter{
		d:    metricDesc{name: name, help: help, mtype: "counter"},
		rows: make(map[string]counterRow),
	}
	r.register(c)
	return c
}

func (c *Counter) desc() metricDesc { return c.d }

// Inc increments the counter by 1 for the given labels.
func (c *Counter) Inc(labels ...string) { c.Add(1, labels...) }

// Add adds delta (must be ≥ 0) to the counter for the given labels.
func (c *Counter) Add(delta float64, labels ...string) {
	if delta < 0 {
		return // counters are monotonic
	}
	key := Labels(labels).labelKey()
	c.mu.Lock()
	row, ok := c.rows[key]
	if !ok {
		row = counterRow{labels: Labels(labels)}
	}
	old := math.Float64frombits(atomic.LoadUint64(&row.value))
	atomic.StoreUint64(&row.value, math.Float64bits(old+delta))
	c.rows[key] = row
	c.mu.Unlock()
}

// Value returns the current value for the given labels (0 if never set).
func (c *Counter) Value(labels ...string) float64 {
	key := Labels(labels).labelKey()
	c.mu.RLock()
	row, ok := c.rows[key]
	c.mu.RUnlock()
	if !ok {
		return 0
	}
	return math.Float64frombits(atomic.LoadUint64(&row.value))
}

func (c *Counter) writeText(w io.Writer) {
	c.mu.RLock()
	rows := make([]counterRow, 0, len(c.rows))
	for _, r := range c.rows {
		rows = append(rows, r)
	}
	c.mu.RUnlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].labels.labelKey() < rows[j].labels.labelKey() })
	for _, r := range rows {
		v := math.Float64frombits(atomic.LoadUint64(&r.value))
		fmt.Fprintf(w, "%s%s %s\n", c.d.name, r.labels.format(), formatFloat(v))
	}
}

// ─── Gauge ───────────────────────────────────────────────────────────────────

// Gauge is an arbitrary floating-point metric.
type Gauge struct {
	d    metricDesc
	mu   sync.RWMutex
	rows map[string]gaugeRow
}

type gaugeRow struct {
	labels Labels
	value  uint64 // float64 bits
}

// NewGauge registers and returns a new Gauge.
func (r *Registry) NewGauge(name, help string) *Gauge {
	g := &Gauge{
		d:    metricDesc{name: name, help: help, mtype: "gauge"},
		rows: make(map[string]gaugeRow),
	}
	r.register(g)
	return g
}

func (g *Gauge) desc() metricDesc { return g.d }

// Set sets the gauge to the given value.
func (g *Gauge) Set(v float64, labels ...string) {
	key := Labels(labels).labelKey()
	g.mu.Lock()
	row, ok := g.rows[key]
	if !ok {
		row = gaugeRow{labels: Labels(labels)}
	}
	atomic.StoreUint64(&row.value, math.Float64bits(v))
	g.rows[key] = row
	g.mu.Unlock()
}

// Add adds delta to the gauge (may be negative).
func (g *Gauge) Add(delta float64, labels ...string) {
	key := Labels(labels).labelKey()
	g.mu.Lock()
	row, ok := g.rows[key]
	if !ok {
		row = gaugeRow{labels: Labels(labels)}
	}
	old := math.Float64frombits(atomic.LoadUint64(&row.value))
	atomic.StoreUint64(&row.value, math.Float64bits(old+delta))
	g.rows[key] = row
	g.mu.Unlock()
}

// Value returns the current gauge value (0 if never set).
func (g *Gauge) Value(labels ...string) float64 {
	key := Labels(labels).labelKey()
	g.mu.RLock()
	row, ok := g.rows[key]
	g.mu.RUnlock()
	if !ok {
		return 0
	}
	return math.Float64frombits(atomic.LoadUint64(&row.value))
}

func (g *Gauge) writeText(w io.Writer) {
	g.mu.RLock()
	rows := make([]gaugeRow, 0, len(g.rows))
	for _, r := range g.rows {
		rows = append(rows, r)
	}
	g.mu.RUnlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].labels.labelKey() < rows[j].labels.labelKey() })
	for _, r := range rows {
		v := math.Float64frombits(atomic.LoadUint64(&r.value))
		fmt.Fprintf(w, "%s%s %s\n", g.d.name, r.labels.format(), formatFloat(v))
	}
}

// ─── Histogram ───────────────────────────────────────────────────────────────

// DefaultBuckets are log-spaced latency buckets suitable for trading systems
// (1ms → 10s in powers of ~2.15).
var DefaultBuckets = []float64{
	0.001, 0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1.0, 2.5, 5.0, 10.0,
}

// Histogram tracks observations across configurable buckets.
type Histogram struct {
	d       metricDesc
	bounds  []float64 // upper bounds, sorted ascending
	mu      sync.RWMutex
	rows    map[string]*histRow
}

type histRow struct {
	labels  Labels
	count   int64
	sum     float64
	buckets []int64 // len == len(bounds)+1; last bucket is +Inf
}

// NewHistogram registers and returns a new Histogram with the given bucket
// upper bounds.  If bounds is nil, [DefaultBuckets] is used.
func (r *Registry) NewHistogram(name, help string, bounds []float64) *Histogram {
	if bounds == nil {
		bounds = DefaultBuckets
	}
	sorted := append([]float64(nil), bounds...)
	sort.Float64s(sorted)
	h := &Histogram{
		d:      metricDesc{name: name, help: help, mtype: "histogram"},
		bounds: sorted,
		rows:   make(map[string]*histRow),
	}
	r.register(h)
	return h
}

func (h *Histogram) desc() metricDesc { return h.d }

// Observe records a single observation v.
func (h *Histogram) Observe(v float64, labels ...string) {
	key := Labels(labels).labelKey()

	h.mu.Lock()
	row, ok := h.rows[key]
	if !ok {
		row = &histRow{labels: Labels(labels), buckets: make([]int64, len(h.bounds)+1)}
		h.rows[key] = row
	}
	atomic.AddInt64(&row.count, 1)
	row.sum += v
	for i, ub := range h.bounds {
		if v <= ub {
			atomic.AddInt64(&row.buckets[i], 1)
		}
	}
	// Always increment +Inf bucket.
	atomic.AddInt64(&row.buckets[len(h.bounds)], 1)
	h.mu.Unlock()
}

// ObserveDuration records a duration as seconds.
func (h *Histogram) ObserveDuration(d time.Duration, labels ...string) {
	h.Observe(d.Seconds(), labels...)
}

func (h *Histogram) writeText(w io.Writer) {
	h.mu.RLock()
	rows := make([]*histRow, 0, len(h.rows))
	for _, r := range h.rows {
		rows = append(rows, r)
	}
	h.mu.RUnlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].labels.labelKey() < rows[j].labels.labelKey() })

	for _, r := range rows {
		lf := r.labels.format()
		// Strip trailing } from label set to insert le= into it.
		prefix := labelSetWithLE(r.labels)

		// Buckets are already cumulative (each Observe call increments all
		// matching buckets), so output them directly.
		for i, ub := range h.bounds {
			cnt := atomic.LoadInt64(&r.buckets[i])
			fmt.Fprintf(w, "%s_bucket%s %d\n", h.d.name, insertLE(prefix, formatFloat(ub)), cnt)
		}
		// +Inf bucket = total count.
		cnt := atomic.LoadInt64(&r.count)
		fmt.Fprintf(w, "%s_bucket%s %d\n", h.d.name, insertLE(prefix, "+Inf"), cnt)
		fmt.Fprintf(w, "%s_sum%s %s\n", h.d.name, lf, formatFloat(r.sum))
		fmt.Fprintf(w, "%s_count%s %d\n", h.d.name, lf, cnt)
	}
}

// labelSetWithLE strips the closing } from a label set so we can inject le=.
func labelSetWithLE(l Labels) string {
	if len(l) == 0 {
		return ""
	}
	s := l.format()
	return s[:len(s)-1] // drop trailing }
}

func insertLE(prefix, le string) string {
	if prefix == "" {
		return fmt.Sprintf(`{le="%s"}`, le)
	}
	return fmt.Sprintf(`%s,le="%s"}`, prefix, le)
}

// ─── Trading-specific helpers ─────────────────────────────────────────────────

// TradingMetrics is a pre-wired set of metrics for the EJLayer trading system.
// Register once and pass the pointer to components that need to record metrics.
type TradingMetrics struct {
	// Signals published by the signal product layer.
	SignalsPublished *Counter
	// Gate decisions (allow / hold / block).
	GateDecisions *Counter
	// Trade fill latency in seconds.
	FillLatency *Histogram
	// Current equity (mark-to-market).
	Equity *Gauge
	// Active positions.
	ActivePositions *Gauge
	// Guardrail halt events.
	HaltEvents *Counter
	// Slippage observations in basis-points.
	SlippageBps *Histogram
}

// NewTradingMetrics registers all standard trading metrics into reg.
func NewTradingMetrics(reg *Registry) *TradingMetrics {
	return &TradingMetrics{
		SignalsPublished: reg.NewCounter(
			"jax_signals_published_total",
			"Total signals published by the signal product layer, by strategy and direction."),
		GateDecisions: reg.NewCounter(
			"jax_gate_decisions_total",
			"Event-gate decisions by decision type (allow/hold/block)."),
		FillLatency: reg.NewHistogram(
			"jax_fill_latency_seconds",
			"Latency from signal to fill in seconds.",
			[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}),
		Equity: reg.NewGauge(
			"jax_account_equity",
			"Current account equity mark-to-market."),
		ActivePositions: reg.NewGauge(
			"jax_active_positions",
			"Number of currently open positions."),
		HaltEvents: reg.NewCounter(
			"jax_guardrail_halt_total",
			"Total guardrail-triggered halt events by reason."),
		SlippageBps: reg.NewHistogram(
			"jax_slippage_bps",
			"Realised slippage in basis-points per fill.",
			[]float64{0, 1, 2, 5, 10, 20, 50, 100, 200}),
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// formatFloat renders a float64 in Prometheus-compatible form.
func formatFloat(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	}
	s := fmt.Sprintf("%g", v)
	return s
}
