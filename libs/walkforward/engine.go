// Package walkforward implements L05: rolling out-of-sample (OOS) validation
// to prevent strategy overfitting.
//
// A walk-forward test splits a historical date range into overlapping windows.
// Each window has an in-sample (IS) period for calibration and an out-of-sample
// (OOS) period for forward testing.  The engine runs a backtest on each OOS
// slice independently, then aggregates the results.
//
// The key metric is the WF Efficiency Ratio (WFER):
//
//	WFER = mean(OOS annualised return) / IS annualised return
//
// A WFER > 0.5 is generally considered sufficient for a strategy to be
// deployable.  A WFER < 0 means the OOS periods lost money.
package walkforward

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"jax-trading-assistant/internal/modules/backtest"
	"jax-trading-assistant/libs/dataset"
	"jax-trading-assistant/libs/strategies"
)

// ─── Config ───────────────────────────────────────────────────────────────────

// Config defines a single walk-forward validation run.
type Config struct {
	// StrategyName must match a strategy registered in the Registry.
	StrategyName string
	// Symbols is the list of tickers.
	Symbols []string
	// FullStart / FullEnd bound the entire date range to split.
	FullStart time.Time
	FullEnd   time.Time
	// ISPeriod is the length of each in-sample window.
	// Defaults to 252 calendar days (~1 trading year) when zero.
	ISPeriod time.Duration
	// OOSPeriod is the length of each out-of-sample window.
	// Defaults to 63 calendar days (~1 trading quarter) when zero.
	OOSPeriod time.Duration
	// DatasetID is the dataset UUID to use from the registry.
	DatasetID string
	// InitialCapital defaults to 100 000 when zero.
	InitialCapital float64
	// RiskPerTrade defaults to 0.01 when zero.
	RiskPerTrade float64
	// Seed for the underlying backtest runs.  0 = auto-generate.
	Seed int64
}

// ─── Window ───────────────────────────────────────────────────────────────────

// Window describes one IS/OOS pair.
type Window struct {
	Index    int
	ISStart  time.Time
	ISEnd    time.Time
	OOSStart time.Time
	OOSEnd   time.Time
}

// ─── WindowResult ─────────────────────────────────────────────────────────────

// WindowResult holds the outcomes for one walk-forward window.
type WindowResult struct {
	Window
	// OOS metrics
	TotalTrades   int
	WinRate       float64
	TotalReturn   float64 // absolute $ return in OOS period
	AnnualisedRet float64 // return annualised to 252 trading days
	MaxDrawdown   float64
	SharpeRatio   float64
	FinalCapital  float64
}

// ─── Result ───────────────────────────────────────────────────────────────────

// Result is the aggregate output of a walk-forward validation run.
type Result struct {
	// Config echoes back the parameters used.
	Config Config

	// Windows contains per-window OOS results in chronological order.
	Windows []WindowResult

	// IS result from running the full IS range (the "calibrated" reference).
	ISResult *backtest.Result

	// Aggregate metrics across all OOS windows.
	MeanOOSReturn  float64 // mean of AnnualisedRet across windows
	WFER           float64 // WF Efficiency Ratio = MeanOOSReturn / IS annualised return
	PassRate       float64 // fraction of windows with positive OOS return
	TotalOOSTrades int

	// Stability score in [0, 1]: fraction of windows beating 0 return,
	// weighted by trade count.
	StabilityScore float64
}

// ─── Engine ───────────────────────────────────────────────────────────────────

// Engine orchestrates walk-forward validation using the backtest engine and
// dataset registry.
type Engine struct {
	bt       *backtest.Engine
	datasets *dataset.Registry
}

// New creates a new walk-forward Engine.
func New(bt *backtest.Engine, datasets *dataset.Registry) *Engine {
	return &Engine{bt: bt, datasets: datasets}
}

// Run executes a full walk-forward validation.
func (e *Engine) Run(ctx context.Context, cfg Config) (*Result, error) {
	// Apply defaults.
	if cfg.ISPeriod == 0 {
		cfg.ISPeriod = 252 * 24 * time.Hour
	}
	if cfg.OOSPeriod == 0 {
		cfg.OOSPeriod = 63 * 24 * time.Hour
	}
	if cfg.InitialCapital <= 0 {
		cfg.InitialCapital = 100_000
	}
	if cfg.RiskPerTrade <= 0 {
		cfg.RiskPerTrade = 0.01
	}

	// Verify dataset integrity before starting.
	if err := e.datasets.VerifyHash(cfg.DatasetID); err != nil {
		return nil, fmt.Errorf("walkforward: %w", err)
	}

	ds, err := e.datasets.Get(cfg.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("walkforward: dataset: %w", err)
	}

	log.Printf("[wf] starting strategy=%q dataset=%s IS=%v OOS=%v range=%s→%s",
		cfg.StrategyName, ds.ID[:8],
		cfg.ISPeriod.String(), cfg.OOSPeriod.String(),
		cfg.FullStart.Format("2006-01-02"), cfg.FullEnd.Format("2006-01-02"))

	// Build windows.
	windows := buildWindows(cfg.FullStart, cfg.FullEnd, cfg.ISPeriod, cfg.OOSPeriod)
	if len(windows) == 0 {
		return nil, fmt.Errorf("walkforward: date range too short to form a single IS+OOS window (need ≥%v)",
			cfg.ISPeriod+cfg.OOSPeriod)
	}

	// ── Full IS run (reference) ────────────────────────────────────────────
	csvSrc, err := e.datasets.LoadDataSource(ctx, cfg.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("walkforward: load dataset: %w", err)
	}

	// IS reference covers FullStart → last IS end
	isEnd := windows[len(windows)-1].ISEnd
	isRef, err := e.bt.Run(ctx, backtest.Config{
		StrategyName:   cfg.StrategyName,
		Symbols:        cfg.Symbols,
		StartDate:      cfg.FullStart,
		EndDate:        isEnd,
		DataSource:     csvSrc,
		Seed:           cfg.Seed,
		InitialCapital: cfg.InitialCapital,
		RiskPerTrade:   cfg.RiskPerTrade,
	})
	if err != nil {
		return nil, fmt.Errorf("walkforward: IS reference run: %w", err)
	}

	isAnnualised := annualise(isRef.TotalReturn/cfg.InitialCapital,
		cfg.FullStart, isEnd)

	// ── OOS windows ───────────────────────────────────────────────────────
	var winResults []WindowResult
	for _, w := range windows {
		// Fresh data source per window (GetCandles filters by date internally).
		wSrc, err := e.datasets.LoadDataSource(ctx, cfg.DatasetID)
		if err != nil {
			return nil, fmt.Errorf("walkforward: window %d: load dataset: %w", w.Index, err)
		}

		res, err := e.bt.Run(ctx, backtest.Config{
			StrategyName:   cfg.StrategyName,
			Symbols:        cfg.Symbols,
			StartDate:      w.OOSStart,
			EndDate:        w.OOSEnd,
			DataSource:     wSrc,
			Seed:           cfg.Seed + int64(w.Index), // different seed per window
			InitialCapital: cfg.InitialCapital,
			RiskPerTrade:   cfg.RiskPerTrade,
		})
		if err != nil {
			log.Printf("[wf] window %d OOS run failed: %v (skipping)", w.Index, err)
			continue
		}

		oosRet := res.TotalReturn / cfg.InitialCapital
		oosAnn := annualise(oosRet, w.OOSStart, w.OOSEnd)

		wr := WindowResult{
			Window:        w,
			TotalTrades:   res.TotalTrades,
			WinRate:       res.WinRate,
			TotalReturn:   res.TotalReturn,
			AnnualisedRet: oosAnn,
			MaxDrawdown:   res.MaxDrawdown,
			SharpeRatio:   res.SharpeRatio,
			FinalCapital:  res.FinalCapital,
		}
		winResults = append(winResults, wr)

		log.Printf("[wf] window %d OOS %s→%s trades=%d annRet=%.2f%%",
			w.Index,
			w.OOSStart.Format("2006-01-02"),
			w.OOSEnd.Format("2006-01-02"),
			res.TotalTrades, oosAnn*100)
	}

	if len(winResults) == 0 {
		return nil, fmt.Errorf("walkforward: all OOS windows failed to produce results")
	}

	// ── Aggregate ─────────────────────────────────────────────────────────
	result := &Result{
		Config:   cfg,
		Windows:  winResults,
		ISResult: isRef,
	}

	var sumRet float64
	var sumTrades int
	var positiveWindows int
	var weightedPositive float64
	var totalWeight float64

	for _, w := range winResults {
		sumRet += w.AnnualisedRet
		sumTrades += w.TotalTrades
		if w.AnnualisedRet > 0 {
			positiveWindows++
		}
		weight := math.Max(float64(w.TotalTrades), 1)
		totalWeight += weight
		if w.AnnualisedRet > 0 {
			weightedPositive += weight
		}
	}

	result.MeanOOSReturn = sumRet / float64(len(winResults))
	result.TotalOOSTrades = sumTrades
	result.PassRate = float64(positiveWindows) / float64(len(winResults))
	if totalWeight > 0 {
		result.StabilityScore = weightedPositive / totalWeight
	}
	if isAnnualised != 0 {
		result.WFER = result.MeanOOSReturn / isAnnualised
	}

	log.Printf("[wf] done windows=%d WFER=%.2f passRate=%.0f%% stabilityScore=%.2f",
		len(winResults), result.WFER, result.PassRate*100, result.StabilityScore)

	return result, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// buildWindows generates IS/OOS window pairs anchored to fullStart.
// Each subsequent window slides forward by OOSPeriod.
func buildWindows(fullStart, fullEnd time.Time, is, oos time.Duration) []Window {
	var windows []Window
	idx := 0
	for {
		isStart := fullStart.Add(time.Duration(idx) * oos)
		isEnd := isStart.Add(is)
		oosStart := isEnd
		oosEnd := oosStart.Add(oos)

		if oosEnd.After(fullEnd) {
			break
		}

		windows = append(windows, Window{
			Index:    idx,
			ISStart:  isStart,
			ISEnd:    isEnd,
			OOSStart: oosStart,
			OOSEnd:   oosEnd,
		})
		idx++
	}
	return windows
}

// annualise converts a fractional return over a date span to an annualised rate.
// The calendar conversion uses 252 trading days ≈ 1 year.
func annualise(ret float64, start, end time.Time) float64 {
	days := end.Sub(start).Hours() / 24
	if days <= 0 {
		return 0
	}
	tradingYears := days / 252
	if tradingYears <= 0 {
		return 0
	}
	// Compound annual growth rate: (1 + ret)^(1/years) - 1
	return math.Pow(1+ret, 1/tradingYears) - 1
}

// WFERVerdict returns a human-readable summary of the walk-forward quality.
func WFERVerdict(r *Result) string {
	switch {
	case r.WFER >= 0.7:
		return "EXCELLENT — strategy transfers to OOS data well"
	case r.WFER >= 0.5:
		return "GOOD — strategy is deployable"
	case r.WFER >= 0.0:
		return "MARGINAL — live performance likely to underperform IS"
	default:
		return "FAIL — strategy loses money out-of-sample; do not deploy"
	}
}

// Ensure the package imports strategies even if only through backtest.
var _ *strategies.Registry
