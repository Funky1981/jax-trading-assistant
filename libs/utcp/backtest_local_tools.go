package utcp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type BacktestEngine struct {
	mu   sync.RWMutex
	runs map[string]GetRunOutput
}

func NewBacktestEngine() *BacktestEngine {
	return &BacktestEngine{runs: make(map[string]GetRunOutput)}
}

func RegisterBacktestTools(registry *LocalRegistry, engine *BacktestEngine) error {
	if registry == nil {
		return fmt.Errorf("register backtest tools: registry is nil")
	}
	if engine == nil {
		return fmt.Errorf("register backtest tools: engine is nil")
	}

	if err := registry.Register(BacktestProviderID, ToolBacktestRunStrategy, engine.runStrategyTool); err != nil {
		return err
	}
	if err := registry.Register(BacktestProviderID, ToolBacktestGetRun, engine.getRunTool); err != nil {
		return err
	}
	return nil
}

func (e *BacktestEngine) runStrategyTool(_ context.Context, input any, output any) error {
	var in RunStrategyInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("backtest.run_strategy: %w", err)
	}
	if strings.TrimSpace(in.StrategyConfigID) == "" {
		return fmt.Errorf("backtest.run_strategy: strategyConfigId is required")
	}
	if len(in.Symbols) == 0 {
		return fmt.Errorf("backtest.run_strategy: symbols is required")
	}
	if in.From.IsZero() || in.To.IsZero() {
		return fmt.Errorf("backtest.run_strategy: from/to are required")
	}
	if !in.To.After(in.From) {
		return fmt.Errorf("backtest.run_strategy: to must be after from")
	}

	runID := deterministicRunID(in.StrategyConfigID, in.Symbols, in.From, in.To)
	run := e.generateRun(runID, in)

	e.mu.Lock()
	e.runs[runID] = run
	e.mu.Unlock()

	out := RunStrategyOutput{RunID: runID, Stats: run.Stats}
	if output == nil {
		return nil
	}
	typed, ok := output.(*RunStrategyOutput)
	if !ok {
		return fmt.Errorf("backtest.run_strategy: output must be *utcp.RunStrategyOutput")
	}
	*typed = out
	return nil
}

func (e *BacktestEngine) getRunTool(_ context.Context, input any, output any) error {
	var in GetRunInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("backtest.get_run: %w", err)
	}
	if strings.TrimSpace(in.RunID) == "" {
		return fmt.Errorf("backtest.get_run: runId is required")
	}

	e.mu.RLock()
	run, ok := e.runs[in.RunID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("backtest.get_run: run not found: %s", in.RunID)
	}

	if output == nil {
		return nil
	}
	typed, ok := output.(*GetRunOutput)
	if !ok {
		return fmt.Errorf("backtest.get_run: output must be *utcp.GetRunOutput")
	}
	*typed = run
	return nil
}

func (e *BacktestEngine) generateRun(runID string, in RunStrategyInput) GetRunOutput {
	symbols := append([]string(nil), in.Symbols...)
	sort.Strings(symbols)

	trades := len(symbols) * 10
	winRate := 0.45 + float64(len(symbols))*0.01
	if winRate > 0.65 {
		winRate = 0.65
	}

	avgR := 1.0 + float64(len(in.StrategyConfigID)%5)*0.1
	maxDD := -0.2 + float64(len(symbols))*0.01
	if maxDD > -0.05 {
		maxDD = -0.05
	}
	sharpe := 1.0 + float64(len(in.StrategyConfigID)%7)*0.1

	bySymbol := make([]RunBySymbol, 0, len(symbols))
	for _, s := range symbols {
		bySymbol = append(bySymbol, RunBySymbol{
			Symbol:  s,
			Trades:  10,
			WinRate: winRate,
		})
	}

	return GetRunOutput{
		RunID: runID,
		Stats: BacktestStats{
			Trades:      trades,
			WinRate:     winRate,
			AvgR:        avgR,
			MaxDrawdown: maxDD,
			Sharpe:      sharpe,
		},
		BySymbol: bySymbol,
	}
}

func deterministicRunID(strategyID string, symbols []string, from time.Time, to time.Time) string {
	s := append([]string(nil), symbols...)
	sort.Strings(s)
	key := strategyID + "|" + strings.Join(s, ",") + "|" + from.UTC().Format(time.RFC3339) + "|" + to.UTC().Format(time.RFC3339)
	sum := sha1.Sum([]byte(key))
	return "bt_" + hex.EncodeToString(sum[:8])
}
