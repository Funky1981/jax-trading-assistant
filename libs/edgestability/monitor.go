// Package edgestability implements L08: edge stability monitoring — drift and
// decay detection for live-trading strategies.
//
// After a strategy is deployed, its win-rate and return distribution can drift
// away from what was observed in backtesting.  This monitor ingests a stream
// of trade outcomes and computes:
//
//   - Rolling performance metrics over a configurable look-back window
//   - Decay score: how much the rolling Sharpe has degraded from the IS baseline
//   - Drift flag: whether performance has fallen below a configurable threshold
//
// When drift is detected the monitor returns a Halt recommendation which the
// risk enforcer (L16) and human guardrails (L18) can act on.
package edgestability

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// ─── Outcome ──────────────────────────────────────────────────────────────────

// Outcome records the result of one completed trade.
type Outcome struct {
	// TradeID is an opaque identifier (e.g. a database UUID string).
	TradeID string
	// Symbol is the ticker.
	Symbol string
	// StrategyID identifies the strategy that generated the signal.
	StrategyID string
	// PnL is the realised profit/loss in USD (negative for losses).
	PnL float64
	// ReturnFrac is PnL / initial position value (signed fraction).
	ReturnFrac float64
	// ClosedAt is when the trade was closed.
	ClosedAt time.Time
}

// ─── Config ───────────────────────────────────────────────────────────────────

// Config controls the stability monitor behaviour.
type Config struct {
	// WindowSize is the number of trades in the rolling window.
	// Defaults to 50 when 0.
	WindowSize int
	// BaselineSharpe is the in-sample Sharpe ratio established during
	// backtesting.  The decay score is expressed relative to this value.
	BaselineSharpe float64
	// MinSharpe is the Sharpe ratio below which a drift alert is raised.
	// Defaults to 0.0 (any negative Sharpe triggers an alert).
	MinSharpe float64
	// MinWinRate is the win-rate below which a drift alert is raised (0–1).
	// Defaults to 0.30 when 0.
	MinWinRate float64
	// MaxDrawdown is the maximum peak-to-trough drawdown fraction before an
	// alert fires.  Defaults to 0.20 (20 %) when 0.
	MaxDrawdown float64
}

func (c *Config) applyDefaults() {
	if c.WindowSize <= 0 {
		c.WindowSize = 50
	}
	if c.MinWinRate <= 0 {
		c.MinWinRate = 0.30
	}
	if c.MaxDrawdown <= 0 {
		c.MaxDrawdown = 0.20
	}
}

// ─── Alert ────────────────────────────────────────────────────────────────────

// AlertCode identifies the specific condition that triggered an alert.
type AlertCode string

const (
	AlertSharpeDecay   AlertCode = "SHARPE_DECAY"
	AlertLowWinRate    AlertCode = "LOW_WIN_RATE"
	AlertDrawdownBreak AlertCode = "DRAWDOWN_BREAK"
)

// Alert signals that the strategy edge may be deteriorating.
type Alert struct {
	Code     AlertCode `json:"code"`
	Message  string    `json:"message"`
	Limit    float64   `json:"limit"`
	Observed float64   `json:"observed"`
}

func (a Alert) Error() string { return string(a.Code) + ": " + a.Message }

// ─── Snapshot ─────────────────────────────────────────────────────────────────

// Snapshot holds the current rolling performance metrics for a strategy.
type Snapshot struct {
	// StrategyID is the strategy this snapshot belongs to.
	StrategyID string `json:"strategy_id"`
	// TradesInWindow is the number of completed trades in the rolling window.
	TradesInWindow int `json:"trades_in_window"`
	// WinRate is the fraction of profitable trades in the window.
	WinRate float64 `json:"win_rate"`
	// MeanReturn is the mean ReturnFrac of trades in the window.
	MeanReturn float64 `json:"mean_return"`
	// SharpeRatio is the annualised Sharpe (mean/std * sqrt(252)).
	SharpeRatio float64 `json:"sharpe_ratio"`
	// DecayScore is (BaselineSharpe - SharpeRatio) / BaselineSharpe.
	// 0 = no decay; 1 = fully decayed; >1 = strategy worse than random.
	DecayScore float64 `json:"decay_score"`
	// MaxDrawdown is the peak-to-trough loss fraction within the window.
	MaxDrawdown float64 `json:"max_drawdown"`
	// Alerts contains the list of current drift conditions.
	Alerts []Alert `json:"alerts,omitempty"`
	// IsDrifting is true when len(Alerts) > 0.
	IsDrifting bool `json:"is_drifting"`
	// SnapshotAt is the time of this reading.
	SnapshotAt time.Time `json:"snapshot_at"`
}

// ─── Monitor ──────────────────────────────────────────────────────────────────

// strategySeries holds outcomes for one strategy.
type strategySeries struct {
	outcomes []Outcome
	peak     float64 // cumulative equity peak (for drawdown)
	equity   float64 // current running equity
}

// Monitor tracks edge stability across one or more strategies.
// Strategies are added implicitly as outcomes are recorded.
type Monitor struct {
	mu       sync.Mutex
	cfg      Config
	series   map[string]*strategySeries // keyed by strategyID
}

// New creates a new Monitor with the given Config.
func New(cfg Config) *Monitor {
	cfg.applyDefaults()
	return &Monitor{
		cfg:    cfg,
		series: make(map[string]*strategySeries),
	}
}

// RecordOutcome appends a trade outcome for its strategy and returns the
// current Snapshot (including any newly triggered Alerts).
func (m *Monitor) RecordOutcome(o Outcome) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if o.StrategyID == "" {
		o.StrategyID = "unknown"
	}

	s, ok := m.series[o.StrategyID]
	if !ok {
		s = &strategySeries{}
		m.series[o.StrategyID] = s
	}

	s.outcomes = append(s.outcomes, o)
	// Maintain rolling window.
	if len(s.outcomes) > m.cfg.WindowSize {
		s.outcomes = s.outcomes[len(s.outcomes)-m.cfg.WindowSize:]
	}

	// Update running equity for drawdown.
	s.equity += o.PnL
	if s.equity > s.peak {
		s.peak = s.equity
	}

	snap := m.snapshot(o.StrategyID, s)
	if snap.IsDrifting {
		log.Printf("[edge] drift detected strategy=%s alerts=%d sharpe=%.2f winRate=%.1f%%",
			o.StrategyID, len(snap.Alerts), snap.SharpeRatio, snap.WinRate*100)
	}
	return snap
}

// Snapshot returns the current performance snapshot for a strategy without
// recording a new outcome.  Returns an error if the strategy is unknown.
func (m *Monitor) Snapshot(strategyID string) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.series[strategyID]
	if !ok {
		return Snapshot{}, fmt.Errorf("edgestability: strategy %q not tracked", strategyID)
	}
	return m.snapshot(strategyID, s), nil
}

// SnapshotAll returns current snapshots for every tracked strategy.
func (m *Monitor) SnapshotAll() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Snapshot, 0, len(m.series))
	for id, s := range m.series {
		out = append(out, m.snapshot(id, s))
	}
	return out
}

// Reset clears all recorded outcomes for a strategy (e.g. after a strategy
// parameter update that resets the baseline).
func (m *Monitor) Reset(strategyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.series, strategyID)
}

// ─── internals ────────────────────────────────────────────────────────────────

func (m *Monitor) snapshot(strategyID string, s *strategySeries) Snapshot {
	w := s.outcomes
	n := len(w)
	snap := Snapshot{
		StrategyID:     strategyID,
		TradesInWindow: n,
		SnapshotAt:     time.Now().UTC(),
	}
	if n == 0 {
		return snap
	}

	// Win rate.
	wins := 0
	rets := make([]float64, n)
	for i, o := range w {
		rets[i] = o.ReturnFrac
		if o.ReturnFrac > 0 {
			wins++
		}
	}
	snap.WinRate = float64(wins) / float64(n)

	// Mean return.
	sum := 0.0
	for _, r := range rets {
		sum += r
	}
	mean := sum / float64(n)
	snap.MeanReturn = mean

	// Std dev.
	variance := 0.0
	for _, r := range rets {
		d := r - mean
		variance += d * d
	}
	if n > 1 {
		variance /= float64(n - 1)
	}
	std := math.Sqrt(variance)

	// Annualised Sharpe (using sqrt(252) as trading-day annualiser).
	if std > 0 {
		snap.SharpeRatio = (mean / std) * math.Sqrt(252)
	}

	// Decay score.
	if m.cfg.BaselineSharpe != 0 {
		snap.DecayScore = (m.cfg.BaselineSharpe - snap.SharpeRatio) / m.cfg.BaselineSharpe
	}

	// Drawdown within window: use running equity relative to peak.
	if s.peak > 0 && s.equity < s.peak {
		snap.MaxDrawdown = (s.peak - s.equity) / s.peak
	}

	// Alerts.
	var alerts []Alert
	if snap.SharpeRatio < m.cfg.MinSharpe {
		alerts = append(alerts, Alert{
			Code:     AlertSharpeDecay,
			Message:  fmt.Sprintf("rolling Sharpe %.2f is below minimum %.2f", snap.SharpeRatio, m.cfg.MinSharpe),
			Limit:    m.cfg.MinSharpe,
			Observed: snap.SharpeRatio,
		})
	}
	if snap.WinRate < m.cfg.MinWinRate {
		alerts = append(alerts, Alert{
			Code:     AlertLowWinRate,
			Message:  fmt.Sprintf("win rate %.1f%% is below minimum %.1f%%", snap.WinRate*100, m.cfg.MinWinRate*100),
			Limit:    m.cfg.MinWinRate,
			Observed: snap.WinRate,
		})
	}
	if snap.MaxDrawdown > m.cfg.MaxDrawdown {
		alerts = append(alerts, Alert{
			Code:     AlertDrawdownBreak,
			Message:  fmt.Sprintf("drawdown %.1f%% exceeds maximum %.1f%%", snap.MaxDrawdown*100, m.cfg.MaxDrawdown*100),
			Limit:    m.cfg.MaxDrawdown,
			Observed: snap.MaxDrawdown,
		})
	}

	snap.Alerts = alerts
	snap.IsDrifting = len(alerts) > 0
	return snap
}
