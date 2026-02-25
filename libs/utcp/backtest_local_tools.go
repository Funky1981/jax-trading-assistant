package utcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	sharedbacktest "jax-trading-assistant/libs/backtest"
	"jax-trading-assistant/libs/runtimepolicy"
	"jax-trading-assistant/libs/strategies"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func syntheticBacktestAllowed() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_SYNTHETIC_BACKTEST_DATA")), "true") {
		return true
	}
	return runtimepolicy.CurrentMode().AllowsSyntheticBacktest()
}

type BacktestEngine struct {
	mu    sync.RWMutex
	runs  map[string]GetRunOutput
	bt    *sharedbacktest.Engine
	store *sql.DB
}

func NewBacktestEngine() *BacktestEngine {
	reg := strategies.NewRegistry()
	registerDefaultStrategies(reg)
	engine := &BacktestEngine{
		runs: make(map[string]GetRunOutput),
		bt:   sharedbacktest.New(reg),
	}
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		db, err := sql.Open("pgx", dsn)
		if err == nil {
			engine.store = db
		}
	}
	return engine
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

func (e *BacktestEngine) runStrategyTool(ctx context.Context, input any, output any) error {
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
	ds, err := e.buildDataSource(ctx, in.Symbols, in.From, in.To)
	if err != nil {
		return fmt.Errorf("backtest.run_strategy: datasource: %w", err)
	}

	seed := in.Seed
	if seed == 0 {
		seed = time.Now().UTC().UnixNano()
	}
	strategyName := normalizeStrategyID(in.StrategyConfigID)
	result, err := e.bt.Run(ctx, sharedbacktest.Config{
		StrategyName:   strategyName,
		Symbols:        in.Symbols,
		StartDate:      in.From.UTC(),
		EndDate:        in.To.UTC(),
		DataSource:     ds,
		Seed:           seed,
		InitialCapital: in.InitialCapital,
		RiskPerTrade:   in.RiskPerTrade,
	})
	if err != nil {
		return fmt.Errorf("backtest.run_strategy: %w", err)
	}

	run := buildRunOutput(result)
	e.mu.Lock()
	e.runs[run.RunID] = run
	e.mu.Unlock()

	if err := e.persistRun(ctx, in, run); err != nil {
		log.Printf("backtest.run_strategy: persist warning: %v", err)
	}

	out := RunStrategyOutput{RunID: run.RunID, Stats: run.Stats}
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

func normalizeStrategyID(id string) string {
	normalized := strings.TrimSpace(strings.ToLower(id))
	switch normalized {
	case "rsi_momentum_v1", "macd_crossover_v1", "ma_crossover_v1":
		return normalized
	default:
		return "rsi_momentum_v1"
	}
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

func registerDefaultStrategies(reg *strategies.Registry) {
	rsi := strategies.NewRSIMomentumStrategy()
	_ = reg.Register(rsi, rsi.GetMetadata())
	macd := strategies.NewMACDCrossoverStrategy()
	_ = reg.Register(macd, macd.GetMetadata())
	ma := strategies.NewMACrossoverStrategy()
	_ = reg.Register(ma, ma.GetMetadata())
}

func buildRunOutput(result *sharedbacktest.Result) GetRunOutput {
	stats := BacktestStats{
		Trades:       result.TotalTrades,
		WinRate:      result.WinRate,
		AvgR:         result.AvgR,
		MaxDrawdown:  result.MaxDrawdown,
		Sharpe:       result.SharpeRatio,
		FinalCapital: result.FinalCapital,
		TotalReturn:  result.TotalReturn,
	}
	bySymbol := make(map[string]*RunBySymbol)
	trades := make([]BacktestTrade, 0, len(result.Trades))
	for _, t := range result.Trades {
		row := bySymbol[t.Symbol]
		if row == nil {
			row = &RunBySymbol{Symbol: t.Symbol}
			bySymbol[t.Symbol] = row
		}
		row.Trades++
		if t.PnL > 0 {
			row.WinRate++
		}
		trades = append(trades, BacktestTrade{
			Symbol:     t.Symbol,
			Direction:  string(t.Direction),
			EntryDate:  t.EntryDate.UTC(),
			ExitDate:   t.ExitDate.UTC(),
			EntryPrice: t.EntryPrice,
			ExitPrice:  t.ExitPrice,
			Quantity:   t.Quantity,
			PnL:        t.PnL,
			PnLPct:     t.PnLPct,
			RMultiple:  t.RMultiple,
			ExitReason: t.ExitReason,
		})
	}
	bySymbolList := make([]RunBySymbol, 0, len(bySymbol))
	for _, entry := range bySymbol {
		if entry.Trades > 0 {
			entry.WinRate = entry.WinRate / float64(entry.Trades)
		}
		bySymbolList = append(bySymbolList, *entry)
	}
	return GetRunOutput{
		RunID:       result.RunID,
		Stats:       stats,
		BySymbol:    bySymbolList,
		StartedAt:   result.RunAt.UTC(),
		CompletedAt: result.RunAt.UTC().Add(time.Duration(result.DurationMs) * time.Millisecond),
		Trades:      trades,
	}
}

func (e *BacktestEngine) persistRun(ctx context.Context, in RunStrategyInput, run GetRunOutput) error {
	if e.store == nil {
		return nil
	}
	dataSourceType := "real"
	sourceProvider := "postgres.candles"
	isSynthetic := false
	statsJSON, _ := json.Marshal(run.Stats)
	configJSON, _ := json.Marshal(map[string]any{
		"strategyConfigId": in.StrategyConfigID,
		"symbols":          in.Symbols,
		"from":             in.From.UTC(),
		"to":               in.To.UTC(),
		"datasetId":        in.DatasetID,
		"riskPerTrade":     in.RiskPerTrade,
	})
	startedAt := run.StartedAt
	completedAt := run.CompletedAt

	var runPK string
	err := e.store.QueryRowContext(ctx, `
		INSERT INTO backtest_runs (
			external_run_id, instance_id, strategy_type_id, strategy_config_id, symbols, run_from, run_to,
			seed, dataset_id, status, stats, config_snapshot, flow_id, started_at, completed_at,
			data_source_type, source_provider, dataset_hash, is_synthetic, synthetic_reason, provenance_verified_at
		) VALUES (
			$1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7,
			$8, $9, 'completed', $10::jsonb, $11::jsonb, $12, $13, $14,
			$15, $16, $17, $18, $19, NOW()
		)
		ON CONFLICT (external_run_id)
		DO UPDATE SET
			stats = EXCLUDED.stats,
			config_snapshot = EXCLUDED.config_snapshot,
			completed_at = EXCLUDED.completed_at,
			data_source_type = EXCLUDED.data_source_type,
			source_provider = EXCLUDED.source_provider,
			dataset_hash = EXCLUDED.dataset_hash,
			is_synthetic = EXCLUDED.is_synthetic,
			synthetic_reason = EXCLUDED.synthetic_reason,
			provenance_verified_at = EXCLUDED.provenance_verified_at
		RETURNING id::text
	`,
		run.RunID, in.InstanceID, normalizeStrategyID(in.StrategyConfigID), in.StrategyConfigID, in.Symbols,
		in.From.UTC(), in.To.UTC(), in.Seed, in.DatasetID, string(statsJSON), string(configJSON),
		in.FlowID, startedAt, completedAt, dataSourceType, sourceProvider, "", isSynthetic, "",
	).Scan(&runPK)
	if err != nil {
		return err
	}

	for _, t := range run.Trades {
		metaJSON, _ := json.Marshal(map[string]any{
			"rMultiple":  t.RMultiple,
			"exitReason": t.ExitReason,
		})
		_, _ = e.store.ExecContext(ctx, `
			INSERT INTO backtest_trades (
				run_id, symbol, side, entry_price, exit_price, quantity, pnl, pnl_pct, opened_at, closed_at, metadata
			) VALUES (
				$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb
			)
		`, runPK, t.Symbol, strings.ToUpper(t.Direction), t.EntryPrice, t.ExitPrice,
			t.Quantity, t.PnL, t.PnLPct, t.EntryDate.UTC(), t.ExitDate.UTC(), string(metaJSON))
	}
	return nil
}

type historicalDataSource struct {
	candles    map[string][]strategies.Candle
	indicators map[string]map[int64]strategies.AnalysisInput
}

func (d *historicalDataSource) GetCandles(_ context.Context, symbol string, start, end time.Time) ([]strategies.Candle, error) {
	rows := d.candles[strings.ToUpper(symbol)]
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]strategies.Candle, 0, len(rows))
	for _, c := range rows {
		if c.Timestamp.Before(start) || c.Timestamp.After(end) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (d *historicalDataSource) GetIndicators(_ context.Context, symbol string, timestamp time.Time) (strategies.AnalysisInput, error) {
	sym := strings.ToUpper(symbol)
	if m := d.indicators[sym]; m != nil {
		if in, ok := m[timestamp.UTC().UnixNano()]; ok {
			return in, nil
		}
	}
	return strategies.AnalysisInput{
		Symbol:      sym,
		Timestamp:   timestamp.UTC(),
		Price:       0,
		ATR:         1,
		RSI:         50,
		MarketTrend: "neutral",
		SectorTrend: "neutral",
	}, nil
}

func (e *BacktestEngine) buildDataSource(ctx context.Context, symbols []string, from, to time.Time) (*historicalDataSource, error) {
	ds := &historicalDataSource{
		candles:    make(map[string][]strategies.Candle, len(symbols)),
		indicators: make(map[string]map[int64]strategies.AnalysisInput, len(symbols)),
	}
	if e.store == nil {
		if !syntheticBacktestAllowed() {
			return nil, fmt.Errorf("synthetic backtest candles disabled in runtime mode %s", runtimepolicy.CurrentMode())
		}
		for _, symbol := range symbols {
			sym := strings.ToUpper(strings.TrimSpace(symbol))
			candles := syntheticCandles(sym, from.UTC(), to.UTC())
			ds.candles[sym] = candles
			ds.indicators[sym] = buildIndicators(sym, candles)
		}
		return ds, nil
	}
	for _, symbol := range symbols {
		sym := strings.ToUpper(strings.TrimSpace(symbol))
		if sym == "" {
			continue
		}
		rows, err := e.store.QueryContext(ctx, `
			SELECT timestamp, open, high, low, close, volume
			FROM candles
			WHERE symbol = $1
			  AND timestamp >= $2
			  AND timestamp <= $3
			ORDER BY timestamp ASC
		`, sym, from.UTC(), to.UTC())
		if err != nil {
			return nil, err
		}
		candles := make([]strategies.Candle, 0, 512)
		for rows.Next() {
			var c strategies.Candle
			if err := rows.Scan(&c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err == nil {
				c.Symbol = sym
				c.Timestamp = c.Timestamp.UTC()
				candles = append(candles, c)
			}
		}
		rows.Close()
		if len(candles) == 0 {
			return nil, fmt.Errorf("no candles found for %s in range", sym)
		}
		ds.candles[sym] = candles
		ds.indicators[sym] = buildIndicators(sym, candles)
	}
	return ds, nil
}

func syntheticCandles(symbol string, from, to time.Time) []strategies.Candle {
	if !to.After(from) {
		to = from.Add(24 * time.Hour)
	}
	step := time.Hour
	if to.Sub(from) <= 72*time.Hour {
		step = 5 * time.Minute
	}
	count := int(to.Sub(from)/step) + 1
	out := make([]strategies.Candle, 0, count)
	price := 100.0 + float64(len(symbol))
	for i := 0; i < count; i++ {
		ts := from.Add(time.Duration(i) * step)
		move := math.Sin(float64(i)/6.0)*0.4 + 0.03
		open := price
		close := open + move
		high := max(open, close) + 0.15
		low := min(open, close) - 0.15
		out = append(out, strategies.Candle{
			Symbol:    symbol,
			Timestamp: ts,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    int64(1000 + (i % 50 * 20)),
		})
		price = close
	}
	return out
}

func buildIndicators(symbol string, candles []strategies.Candle) map[int64]strategies.AnalysisInput {
	out := make(map[int64]strategies.AnalysisInput, len(candles))
	closes := make([]float64, 0, len(candles))
	volumes := make([]float64, 0, len(candles))
	gains := make([]float64, 0, len(candles))
	losses := make([]float64, 0, len(candles))
	for i, c := range candles {
		closes = append(closes, c.Close)
		volumes = append(volumes, float64(c.Volume))
		if i > 0 {
			delta := closes[i] - closes[i-1]
			if delta > 0 {
				gains = append(gains, delta)
				losses = append(losses, 0)
			} else {
				gains = append(gains, 0)
				losses = append(losses, math.Abs(delta))
			}
		} else {
			gains = append(gains, 0)
			losses = append(losses, 0)
		}
		rsi := calcRSI(gains, losses, 14)
		sma20 := calcSMA(closes, 20)
		sma50 := calcSMA(closes, 50)
		sma200 := calcSMA(closes, 200)
		macdV, macdSignal := calcMACD(closes)
		atr := calcATR(candles[:i+1], 14)
		avgVol := int64(calcSMA(volumes, 20))
		if avgVol <= 0 {
			avgVol = c.Volume
		}
		trend := "neutral"
		if c.Close > sma50 {
			trend = "bullish"
		} else if c.Close < sma50 {
			trend = "bearish"
		}
		out[c.Timestamp.UnixNano()] = strategies.AnalysisInput{
			Symbol:      symbol,
			Price:       c.Close,
			Timestamp:   c.Timestamp.UTC(),
			RSI:         rsi,
			MACD:        strategies.MACD{Value: macdV, Signal: macdSignal, Histogram: macdV - macdSignal},
			SMA20:       sma20,
			SMA50:       sma50,
			SMA200:      sma200,
			ATR:         atr,
			Volume:      c.Volume,
			AvgVolume20: avgVol,
			MarketTrend: trend,
			SectorTrend: trend,
		}
	}
	return out
}

func calcSMA(values []float64, n int) float64 {
	if len(values) == 0 {
		return 0
	}
	if n <= 0 || n > len(values) {
		n = len(values)
	}
	sum := 0.0
	for i := len(values) - n; i < len(values); i++ {
		sum += values[i]
	}
	return sum / float64(n)
}

func calcRSI(gains, losses []float64, period int) float64 {
	if len(gains) < 2 {
		return 50
	}
	avgGain := calcSMA(gains, period)
	avgLoss := calcSMA(losses, period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func calcMACD(closes []float64) (float64, float64) {
	ema12 := calcEMA(closes, 12)
	ema26 := calcEMA(closes, 26)
	macd := ema12 - ema26
	series := []float64{macd}
	return macd, calcEMA(series, 9)
}

func calcEMA(values []float64, period int) float64 {
	if len(values) == 0 {
		return 0
	}
	k := 2.0 / float64(period+1)
	ema := values[0]
	for i := 1; i < len(values); i++ {
		ema = values[i]*k + ema*(1-k)
	}
	return ema
}

func calcATR(candles []strategies.Candle, period int) float64 {
	if len(candles) == 0 {
		return 1
	}
	tr := make([]float64, 0, len(candles))
	for i := range candles {
		highLow := candles[i].High - candles[i].Low
		if i == 0 {
			tr = append(tr, highLow)
			continue
		}
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		tr = append(tr, max(highLow, max(highClose, lowClose)))
	}
	atr := calcSMA(tr, period)
	if atr <= 0 {
		return 1
	}
	return atr
}
