// Package microstructure implements L21–L24:
//
//   - L21: Tick/bid-ask spread capture during events (SpreadCapture, TickStore)
//   - L22: Realized slippage model built from actual fills (SlippageModel)
//   - L23: Correlation shock detection with caps (CorrelationMonitor)
//   - L24: Broker latency distribution tracking and trading-pause logic
//     (LatencyTracker)
package microstructure

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// ─── L21: Tick / Spread Capture ───────────────────────────────────────────────

// Tick is a single bid/ask update.
type Tick struct {
	Symbol    string
	Bid       float64
	Ask       float64
	Timestamp time.Time
}

// Spread returns the bid-ask spread in price units.
func (t Tick) Spread() float64 {
	return t.Ask - t.Bid
}

// SpreadBps returns the spread in basis points relative to mid.
func (t Tick) SpreadBps() float64 {
	mid := (t.Bid + t.Ask) / 2
	if mid == 0 {
		return 0
	}
	return (t.Spread() / mid) * 10_000
}

// TickStore is a bounded ring buffer of recent ticks for each symbol.
// It is safe for concurrent use.
type TickStore struct {
	mu      sync.RWMutex
	maxTicks int
	ticks   map[string][]Tick // symbol → ring buffer (in append order)
}

// NewTickStore creates a TickStore that retains at most maxTicks ticks per symbol.
func NewTickStore(maxTicks int) *TickStore {
	if maxTicks <= 0 {
		maxTicks = 1000
	}
	return &TickStore{maxTicks: maxTicks, ticks: make(map[string][]Tick)}
}

// Record adds a tick to the store.
func (ts *TickStore) Record(t Tick) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	buf := ts.ticks[t.Symbol]
	buf = append(buf, t)
	if len(buf) > ts.maxTicks {
		buf = buf[len(buf)-ts.maxTicks:]
	}
	ts.ticks[t.Symbol] = buf
}

// Recent returns the most recent n ticks for symbol. If n <= 0, all are returned.
func (ts *TickStore) Recent(symbol string, n int) []Tick {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	buf := ts.ticks[symbol]
	if n <= 0 || n >= len(buf) {
		out := make([]Tick, len(buf))
		copy(out, buf)
		return out
	}
	out := make([]Tick, n)
	copy(out, buf[len(buf)-n:])
	return out
}

// SpreadStats computes spread statistics for the given ticks.
type SpreadStats struct {
	Symbol  string
	Count   int
	MinBps  float64
	MaxBps  float64
	MeanBps float64
	P95Bps  float64
}

// AnalyseSpread computes spread statistics over the given ticks.
func AnalyseSpread(symbol string, ticks []Tick) SpreadStats {
	if len(ticks) == 0 {
		return SpreadStats{Symbol: symbol}
	}
	spreads := make([]float64, len(ticks))
	for i, t := range ticks {
		spreads[i] = t.SpreadBps()
	}
	sort.Float64s(spreads)

	sum := 0.0
	for _, s := range spreads {
		sum += s
	}
	mean := sum / float64(len(spreads))

	p95idx := int(math.Ceil(0.95*float64(len(spreads)))) - 1
	if p95idx < 0 {
		p95idx = 0
	}
	if p95idx >= len(spreads) {
		p95idx = len(spreads) - 1
	}

	return SpreadStats{
		Symbol:  symbol,
		Count:   len(spreads),
		MinBps:  spreads[0],
		MaxBps:  spreads[len(spreads)-1],
		MeanBps: mean,
		P95Bps:  spreads[p95idx],
	}
}

// ─── L22: Realized Slippage Model ────────────────────────────────────────────

// FillObservation is one data point for the slippage model.
type FillObservation struct {
	Symbol       string
	// SlippageBps is the observed slippage in basis points for this fill
	// (signed: positive = paid more / received less than expected).
	SlippageBps  float64
	// Quantity is the fill size (used as weight in weighted average).
	Quantity     float64
	EventPhase   string    // e.g. "blackout", "pre_event", "normal"
	ObservedAt   time.Time
}

// SlippageStats summarises the model's estimates for a symbol/phase bucket.
type SlippageStats struct {
	Symbol       string
	EventPhase   string
	Count        int
	MeanBps      float64
	P95Bps       float64
	MaxBps       float64
}

// SlippageModel maintains per-symbol, per-phase rolling slippage statistics.
// It is safe for concurrent use.
type SlippageModel struct {
	mu      sync.RWMutex
	maxObs  int
	obs     map[string][]FillObservation // key = symbol+"|"+phase
}

// NewSlippageModel creates a model retaining at most maxObs observations per bucket.
func NewSlippageModel(maxObs int) *SlippageModel {
	if maxObs <= 0 {
		maxObs = 500
	}
	return &SlippageModel{maxObs: maxObs, obs: make(map[string][]FillObservation)}
}

func slippageKey(symbol, phase string) string {
	return symbol + "|" + phase
}

// Record adds a fill observation to the model.
func (m *SlippageModel) Record(o FillObservation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := slippageKey(o.Symbol, o.EventPhase)
	buf := m.obs[key]
	buf = append(buf, o)
	if len(buf) > m.maxObs {
		buf = buf[len(buf)-m.maxObs:]
	}
	m.obs[key] = buf
}

// Stats returns slippage statistics for a symbol/phase bucket.
func (m *SlippageModel) Stats(symbol, phase string) SlippageStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := slippageKey(symbol, phase)
	obs := m.obs[key]
	if len(obs) == 0 {
		return SlippageStats{Symbol: symbol, EventPhase: phase}
	}

	slippages := make([]float64, len(obs))
	for i, o := range obs {
		slippages[i] = o.SlippageBps
	}
	sort.Float64s(slippages)

	sum := 0.0
	for _, s := range slippages {
		sum += s
	}

	p95idx := int(math.Ceil(0.95*float64(len(slippages)))) - 1
	if p95idx < 0 {
		p95idx = 0
	}
	if p95idx >= len(slippages) {
		p95idx = len(slippages) - 1
	}

	return SlippageStats{
		Symbol:     symbol,
		EventPhase: phase,
		Count:      len(slippages),
		MeanBps:    sum / float64(len(slippages)),
		P95Bps:     slippages[p95idx],
		MaxBps:     slippages[len(slippages)-1],
	}
}

// EstimateBps returns the P95 slippage estimate for a symbol/phase,
// falling back to the "normal" phase if the specific phase has no data.
// If no data at all, returns defaultBps.
func (m *SlippageModel) EstimateBps(symbol, phase string, defaultBps float64) float64 {
	s := m.Stats(symbol, phase)
	if s.Count > 0 {
		return s.P95Bps
	}
	// fall back to "normal"
	s = m.Stats(symbol, "normal")
	if s.Count > 0 {
		return s.P95Bps
	}
	return defaultBps
}

// ─── L23: Correlation Shock Monitoring ───────────────────────────────────────

// ReturnSeries holds time-aligned returns for one symbol.
type ReturnSeries struct {
	Symbol  string
	Returns []float64 // chronological order
}

// CorrelationAlert is fired when a pairwise correlation spike is detected.
type CorrelationAlert struct {
	SymbolA        string
	SymbolB        string
	Correlation    float64
	Threshold      float64
	DetectedAt     time.Time
}

// CorrelationMonitorConfig controls detection sensitivity.
type CorrelationMonitorConfig struct {
	// SpikeThreshold: absolute correlation above which we alert (e.g. 0.85).
	SpikeThreshold float64
	// MinWindow: minimum number of aligned return observations required.
	MinWindow int
}

// DefaultCorrelationMonitorConfig returns sensible defaults.
func DefaultCorrelationMonitorConfig() CorrelationMonitorConfig {
	return CorrelationMonitorConfig{
		SpikeThreshold: 0.85,
		MinWindow:      20,
	}
}

// CorrelationMonitor detects correlation spikes across a portfolio of symbols.
type CorrelationMonitor struct {
	cfg  CorrelationMonitorConfig
	mu   sync.Mutex
	data map[string]*ReturnSeries
}

// NewCorrelationMonitor creates a CorrelationMonitor.
func NewCorrelationMonitor(cfg CorrelationMonitorConfig) *CorrelationMonitor {
	return &CorrelationMonitor{cfg: cfg, data: make(map[string]*ReturnSeries)}
}

// RecordReturn adds a return observation for symbol.
func (c *CorrelationMonitor) RecordReturn(symbol string, r float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.data[symbol]; !ok {
		c.data[symbol] = &ReturnSeries{Symbol: symbol}
	}
	c.data[symbol].Returns = append(c.data[symbol].Returns, r)
}

// Scan computes all pairwise correlations and returns alerts for pairs
// above the spike threshold. Only pairs with ≥ MinWindow aligned observations
// are evaluated.
func (c *CorrelationMonitor) Scan() []CorrelationAlert {
	c.mu.Lock()
	defer c.mu.Unlock()

	symbols := make([]string, 0, len(c.data))
	for sym := range c.data {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols) // deterministic order

	now := time.Now().UTC()
	var alerts []CorrelationAlert

	for i := 0; i < len(symbols)-1; i++ {
		for j := i + 1; j < len(symbols); j++ {
			a := c.data[symbols[i]]
			b := c.data[symbols[j]]
			corr, ok := pearson(a.Returns, b.Returns, c.cfg.MinWindow)
			if !ok {
				continue
			}
			if math.Abs(corr) >= c.cfg.SpikeThreshold {
				alert := CorrelationAlert{
					SymbolA:     symbols[i],
					SymbolB:     symbols[j],
					Correlation: corr,
					Threshold:   c.cfg.SpikeThreshold,
					DetectedAt:  now,
				}
				log.Printf("[microstructure] correlation spike: %s⟷%s corr=%.3f",
					symbols[i], symbols[j], corr)
				alerts = append(alerts, alert)
			}
		}
	}
	return alerts
}

// pearson computes the Pearson correlation of the last min(n,minWindow) aligned
// observations from x and y. Returns false if fewer than minWindow aligned
// points are available.
func pearson(x, y []float64, minWindow int) (float64, bool) {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n < minWindow {
		return 0, false
	}
	x = x[len(x)-n:]
	y = y[len(y)-n:]

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	fn := float64(n)
	for i := range n {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}
	num := fn*sumXY - sumX*sumY
	den := math.Sqrt((fn*sumX2 - sumX*sumX) * (fn*sumY2 - sumY*sumY))
	if den == 0 {
		return 0, false
	}
	return num / den, true
}

// ─── L24: Broker Latency Tracking ────────────────────────────────────────────

// LatencyObservation is one round-trip latency measurement.
type LatencyObservation struct {
	// Category tags the type of call: "order_submit", "order_ack", "fill_ack", etc.
	Category  string
	Latency   time.Duration
	RecordedAt time.Time
}

// LatencyStats holds percentile statistics for a latency bucket.
type LatencyStats struct {
	Category string
	Count    int
	P50      time.Duration
	P95      time.Duration
	P99      time.Duration
	Max      time.Duration
}

// LatencyTrackerConfig controls the tracker's behaviour.
type LatencyTrackerConfig struct {
	// MaxObsPerCategory is the rolling window size per category.
	MaxObsPerCategory int
	// PauseThreshold: if P99 exceeds this, TradingPaused() returns true.
	PauseThreshold time.Duration
	// PauseMinSamples: minimum samples required before pause logic activates.
	PauseMinSamples int
}

// DefaultLatencyTrackerConfig returns sensible defaults.
func DefaultLatencyTrackerConfig() LatencyTrackerConfig {
	return LatencyTrackerConfig{
		MaxObsPerCategory: 500,
		PauseThreshold:    500 * time.Millisecond,
		PauseMinSamples:   20,
	}
}

// LatencyTracker records broker latency and exposes percentile stats and pause logic.
type LatencyTracker struct {
	cfg LatencyTrackerConfig
	mu  sync.RWMutex
	obs map[string][]LatencyObservation
}

// NewLatencyTracker creates a LatencyTracker.
func NewLatencyTracker(cfg LatencyTrackerConfig) *LatencyTracker {
	return &LatencyTracker{cfg: cfg, obs: make(map[string][]LatencyObservation)}
}

// Record adds a latency observation.
func (l *LatencyTracker) Record(o LatencyObservation) {
	l.mu.Lock()
	defer l.mu.Unlock()

	buf := l.obs[o.Category]
	buf = append(buf, o)
	if len(buf) > l.cfg.MaxObsPerCategory {
		buf = buf[len(buf)-l.cfg.MaxObsPerCategory:]
	}
	l.obs[o.Category] = buf
}

// Stats computes percentile stats for a category. Returns zero Stats if no data.
func (l *LatencyTracker) Stats(category string) LatencyStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	obs := l.obs[category]
	if len(obs) == 0 {
		return LatencyStats{Category: category}
	}

	durations := make([]int64, len(obs))
	for i, o := range obs {
		durations[i] = int64(o.Latency)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	pct := func(p float64) time.Duration {
		idx := int(math.Ceil(p*float64(len(durations)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(durations) {
			idx = len(durations) - 1
		}
		return time.Duration(durations[idx])
	}

	return LatencyStats{
		Category: category,
		Count:    len(durations),
		P50:      pct(0.50),
		P95:      pct(0.95),
		P99:      pct(0.99),
		Max:      time.Duration(durations[len(durations)-1]),
	}
}

// TradingPaused returns true when any tracked category has P99 latency exceeding
// the configured threshold. Requires PauseMinSamples to activate.
func (l *LatencyTracker) TradingPaused() (bool, string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for cat, obs := range l.obs {
		if len(obs) < l.cfg.PauseMinSamples {
			continue
		}
		// Inline P99 without lock contention.
		durations := make([]int64, len(obs))
		for i, o := range obs {
			durations[i] = int64(o.Latency)
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		idx := int(math.Ceil(0.99*float64(len(durations)))) - 1
		if idx < 0 {
			idx = 0
		}
		p99 := time.Duration(durations[idx])

		if p99 > l.cfg.PauseThreshold {
			reason := fmt.Sprintf("latency P99 %.0fms > threshold %.0fms (category=%s)",
				float64(p99)/float64(time.Millisecond),
				float64(l.cfg.PauseThreshold)/float64(time.Millisecond),
				cat)
			log.Printf("[microstructure] trading pause: %s", reason)
			return true, reason
		}
	}
	return false, ""
}

// Categories returns all categories that have at least one observation.
func (l *LatencyTracker) Categories() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	cats := make([]string, 0, len(l.obs))
	for cat := range l.obs {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}
