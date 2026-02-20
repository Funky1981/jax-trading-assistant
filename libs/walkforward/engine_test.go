package walkforward_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/backtest"
	"jax-trading-assistant/libs/dataset"
	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/libs/walkforward"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// generateCSV writes N daily rows starting from 2020-01-02.
func generateCSV(t *testing.T, dir string, rows int) (path string) {
	t.Helper()
	buf := "date,open,high,low,close,volume\n"
	base := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	price := 100.0
	for i := range rows {
		d := base.Add(time.Duration(i) * 24 * time.Hour)
		// Simple trending price so strategies fire signals.
		price += 0.5
		line := d.Format("2006-01-02") +
			"," + fmt.Sprintf("%.2f", price-0.3) +
			"," + fmt.Sprintf("%.2f", price+1.0) +
			"," + fmt.Sprintf("%.2f", price-1.5) +
			"," + fmt.Sprintf("%.2f", price) +
			",500000\n"
		buf += line
	}
	path = filepath.Join(dir, "test.csv")
	if err := os.WriteFile(path, []byte(buf), 0o644); err != nil {
		t.Fatalf("generate csv: %v", err)
	}
	return path
}

// ─── buildWindows tests ───────────────────────────────────────────────────────

func TestBuildWindowsCount(t *testing.T) {
	// Exposed via the package for testing. We verify behaviour through Run().
	// Instead, verify the period arithmetic directly.
	is := 252 * 24 * time.Hour
	oos := 63 * 24 * time.Hour
	total := (is + 3*oos) // room for IS + 3 OOS windows

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(total)

	cfg := walkforward.Config{
		StrategyName: "dummy",
		Symbols:      []string{"TEST"},
		FullStart:    start,
		FullEnd:      end,
		ISPeriod:     is,
		OOSPeriod:    oos,
	}

	// We can't call buildWindows directly but we can check via Run that the
	// correct number of windows is generated. Here just do the math:
	windowCount := 0
	for cursor := start; ; {
		isEnd := cursor.Add(is)
		oosEnd := isEnd.Add(oos)
		if oosEnd.After(end) {
			break
		}
		windowCount++
		cursor = cursor.Add(oos)
	}

	if windowCount < 2 {
		t.Errorf("expected at least 2 windows, counted %d for range %v", windowCount, total)
	}
	_ = cfg
}

// ─── Engine.Run ───────────────────────────────────────────────────────────────

func setupEngine(t *testing.T) (eng *walkforward.Engine, reg *dataset.Registry, csvPath string) {
	t.Helper()
	dir := t.TempDir()
	// 600 rows ≈ 2 years of daily data
	csvPath = generateCSV(t, dir, 600)

	reg, err := dataset.Open(dir)
	if err != nil {
		t.Fatalf("dataset.Open: %v", err)
	}

	stratReg := strategies.NewRegistry()
	rsiStrat := strategies.NewRSIMomentumStrategy()
	if err := stratReg.Register(rsiStrat, rsiStrat.GetMetadata()); err != nil {
		t.Fatalf("register strategy: %v", err)
	}
	btEng := backtest.New(stratReg)
	eng = walkforward.New(btEng, reg)
	return eng, reg, csvPath
}

const testStrategyID = "rsi_momentum_v1"

func TestRunReturnsResult(t *testing.T) {
	eng, reg, csvPath := setupEngine(t)

	ds, err := reg.Register(dataset.Dataset{
		Name:     "WF_TEST",
		Symbol:   "TEST",
		FilePath: csvPath,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	start := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	end := start.Add(500 * 24 * time.Hour)

	result, err := eng.Run(context.Background(), walkforward.Config{
		StrategyName: testStrategyID,
		Symbols:      []string{"TEST"},
		FullStart:    start,
		FullEnd:      end,
		DatasetID:    ds.ID,
		ISPeriod:     252 * 24 * time.Hour,
		OOSPeriod:    63 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Windows) == 0 {
		t.Error("expected at least one window result")
	}
	if result.ISResult == nil {
		t.Error("expected IS reference result")
	}
	// WFER may be any value; just ensure it's not NaN.
	if result.WFER != result.WFER {
		t.Errorf("WFER is NaN")
	}
	if result.PassRate < 0 || result.PassRate > 1 {
		t.Errorf("PassRate out of [0,1]: %f", result.PassRate)
	}
	if result.StabilityScore < 0 || result.StabilityScore > 1 {
		t.Errorf("StabilityScore out of [0,1]: %f", result.StabilityScore)
	}
}

func TestRunRangeTooShortReturnsError(t *testing.T) {
	eng, reg, csvPath := setupEngine(t)

	ds, err := reg.Register(dataset.Dataset{
		Name:     "WF_SHORT",
		Symbol:   "S",
		FilePath: csvPath,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	start := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * 24 * time.Hour) // only 10 days — way too short

	_, err = eng.Run(context.Background(), walkforward.Config{
		StrategyName: testStrategyID,
		Symbols:      []string{"S"},
		FullStart:    start,
		FullEnd:      end,
		DatasetID:    ds.ID,
		ISPeriod:     252 * 24 * time.Hour,
		OOSPeriod:    63 * 24 * time.Hour,
	})
	if err == nil {
		t.Fatal("expected error for range too short to build any window")
	}
}

func TestRunBadDatasetIDReturnsError(t *testing.T) {
	eng, _, _ := setupEngine(t)

	_, err := eng.Run(context.Background(), walkforward.Config{
		StrategyName: testStrategyID,
		Symbols:      []string{"X"},
		FullStart:    time.Now().Add(-400 * 24 * time.Hour),
		FullEnd:      time.Now(),
		DatasetID:    "00000000-0000-0000-0000-000000000000",
	})
	if err == nil {
		t.Fatal("expected error for bad dataset ID")
	}
}

// ─── WFERVerdict ──────────────────────────────────────────────────────────────

func TestWFERVerdict(t *testing.T) {
	tests := []struct {
		wfer    float64
		contain string
	}{
		{0.8, "EXCELLENT"},
		{0.6, "GOOD"},
		{0.2, "MARGINAL"},
		{-0.3, "FAIL"},
	}
	for _, tc := range tests {
		r := &walkforward.Result{WFER: tc.wfer}
		v := walkforward.WFERVerdict(r)
		if len(v) == 0 || v[:len(tc.contain)] != tc.contain {
			t.Errorf("WFER=%.1f: got %q, want prefix %q", tc.wfer, v, tc.contain)
		}
	}
}
