package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/dataset"
	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/libs/strategytypes"
)

type strategyTypeTrade struct {
	Symbol    string
	Direction string
	Entry     float64
	Exit      float64
	Qty       float64
	PnL       float64
	OpenedAt  time.Time
	ClosedAt  time.Time
}

func runStrategyTypeBacktest(ctx context.Context, deps *backtestDeps, req BacktestRequest, src *dataset.CSVDataSource, start, end time.Time) (BacktestResponse, error) {
	started := time.Now().UTC()
	st, ok := deps.strategyTypeReg.Get(req.Strategy)
	if !ok {
		return BacktestResponse{}, fmt.Errorf("strategy type not registered: %s", req.Strategy)
	}

	params := extractStrategyParams(req.Parameters)
	if err := st.Validate(params); err != nil {
		return BacktestResponse{}, err
	}

	loc, err := time.LoadLocation(strings.TrimSpace(req.SessionTimezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}
	if req.SessionTimezone == "" {
		req.SessionTimezone = "America/New_York"
	}
	if req.FlattenByCloseTime == "" {
		req.FlattenByCloseTime = "15:55"
	}

	trades := make([]strategyTypeTrade, 0, 128)
	for _, symbol := range req.Symbols {
		candles, err := src.GetCandles(ctx, symbol, start, end)
		if err != nil {
			return BacktestResponse{}, fmt.Errorf("strategy type backtest: load candles for %s: %w", symbol, err)
		}
		byDay := groupCandlesBySessionDay(candles, loc)
		earningsByDay, newsByDay := fetchEventBuckets(ctx, deps.db, symbol, start, end, loc)
		days := make([]string, 0, len(byDay))
		for day := range byDay {
			days = append(days, day)
		}
		sort.Strings(days)
		prevClose := 0.0
		for _, day := range days {
			sessionCandles := byDay[day]
			if len(sessionCandles) == 0 {
				continue
			}
			sessionDate := sessionCandles[0].Timestamp.In(loc)
			input := strategytypes.StrategyInput{
				Symbol:      symbol,
				SessionDate: sessionDate,
				Timezone:    req.SessionTimezone,
				Candles: map[string][]strategytypes.Candle{
					"1m": sessionCandles,
				},
				Earnings:    withPrevClose(earningsByDay[day], prevClose),
				News:        newsByDay[day],
				Parameters:  params,
				FlattenTime: req.FlattenByCloseTime,
			}
			signals, err := st.Generate(ctx, input)
			if err != nil {
				return BacktestResponse{}, fmt.Errorf("strategy type generate failed (%s): %w", symbol, err)
			}
			for _, sig := range signals {
				trade, ok := simulateTrade(sessionCandles, sig, req.InitialCapital, req.RiskPerTrade)
				if ok {
					trades = append(trades, trade)
				}
			}
			prevClose = sessionCandles[len(sessionCandles)-1].Close
		}
	}

	resp := summarizeStrategyTypeTrades(req, trades)
	resp.DurationMs = time.Since(started).Milliseconds()
	return resp, nil
}

func extractStrategyParams(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	if params, ok := input["parameters"].(map[string]any); ok {
		return params
	}
	return input
}

func groupCandlesBySessionDay(candles []strategies.Candle, loc *time.Location) map[string][]strategytypes.Candle {
	out := make(map[string][]strategytypes.Candle)
	for _, c := range candles {
		dayKey := c.Timestamp.In(loc).Format("2006-01-02")
		out[dayKey] = append(out[dayKey], strategytypes.Candle{
			Timestamp: c.Timestamp,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    float64(c.Volume),
		})
	}
	for day, list := range out {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Timestamp.Before(list[j].Timestamp)
		})
		out[day] = list
	}
	return out
}

func fetchEventBuckets(ctx context.Context, db *sql.DB, symbol string, start, end time.Time, loc *time.Location) (map[string][]strategytypes.EarningsEvent, map[string][]strategytypes.NewsEvent) {
	earnings := make(map[string][]strategytypes.EarningsEvent)
	news := make(map[string][]strategytypes.NewsEvent)
	if db == nil {
		return earnings, news
	}
	rows, err := db.QueryContext(ctx, `
		SELECT en.event_kind, en.event_time, en.severity, en.attributes::text, en.payload::text
		FROM event_normalized en
		JOIN event_symbol_map esm ON en.id = esm.normalized_event_id
		WHERE esm.symbol = $1
		  AND en.event_time >= $2
		  AND en.event_time <= $3
		ORDER BY en.event_time ASC
	`, strings.ToUpper(strings.TrimSpace(symbol)), start, end)
	if err != nil {
		return earnings, news
	}
	defer rows.Close()

	for rows.Next() {
		var kind, severity string
		var attrsRaw, payloadRaw sql.NullString
		var eventTime time.Time
		if err := rows.Scan(&kind, &eventTime, &severity, &attrsRaw, &payloadRaw); err != nil {
			continue
		}
		dayKey := eventTime.In(loc).Format("2006-01-02")
		attrs := parseJSONMap(attrsRaw.String)
		payload := parseJSONMap(payloadRaw.String)
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "earnings":
			surprise := floatFromMap(attrs, "surprisePct")
			if surprise == 0 {
				surprise = floatFromMap(payload, "surprise_pct")
			}
			earnings[dayKey] = append(earnings[dayKey], strategytypes.EarningsEvent{
				Timestamp:   eventTime,
				SurprisePct: surprise,
				Guidance:    stringFromMap(attrs, "guidance"),
			})
		case "news":
			materiality := stringFromMap(attrs, "materiality")
			if materiality == "" {
				materiality = materialityFromSeverity(severity)
			}
			news[dayKey] = append(news[dayKey], strategytypes.NewsEvent{
				Timestamp:   eventTime,
				Category:    stringFromMap(attrs, "category"),
				Materiality: materiality,
				Sentiment:   sentimentFromPayload(attrs, payload),
			})
		}
	}
	return earnings, news
}

func materialityFromSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func sentimentFromPayload(attrs, payload map[string]any) string {
	for _, key := range []string{"sentiment", "tone"} {
		if v := stringFromMap(attrs, key); v != "" {
			return v
		}
		if v := stringFromMap(payload, key); v != "" {
			return v
		}
	}
	return "neutral"
}

func withPrevClose(events []strategytypes.EarningsEvent, prevClose float64) []strategytypes.EarningsEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]strategytypes.EarningsEvent, 0, len(events))
	for _, ev := range events {
		ev.PreviousClose = prevClose
		out = append(out, ev)
	}
	return out
}

func simulateTrade(candles []strategytypes.Candle, signal strategytypes.Signal, initialCapital, riskPerTrade float64) (strategyTypeTrade, bool) {
	if len(candles) == 0 {
		return strategyTypeTrade{}, false
	}
	entryIdx := 0
	for i, c := range candles {
		if !c.Timestamp.Before(signal.GeneratedAt) {
			entryIdx = i
			break
		}
	}
	entry := signal.EntryPrice
	if entry == 0 {
		entry = candles[entryIdx].Close
	}
	stop := signal.StopLoss
	if stop == 0 {
		stop = entry * 0.995
	}
	take := signal.TakeProfit
	if take == 0 {
		take = entry * 1.01
	}
	riskAmt := initialCapital * riskPerTrade
	if riskAmt <= 0 {
		riskAmt = 1000
	}
	stopDist := math.Abs(entry - stop)
	qty := 1.0
	if stopDist > 0 {
		qty = riskAmt / stopDist
	}
	exit := candles[len(candles)-1].Close
	exitTime := candles[len(candles)-1].Timestamp
	for i := entryIdx; i < len(candles); i++ {
		c := candles[i]
		if strings.EqualFold(signal.Direction, "BUY") {
			if c.Low <= stop {
				exit = stop
				exitTime = c.Timestamp
				break
			}
			if c.High >= take {
				exit = take
				exitTime = c.Timestamp
				break
			}
		} else {
			if c.High >= stop {
				exit = stop
				exitTime = c.Timestamp
				break
			}
			if c.Low <= take {
				exit = take
				exitTime = c.Timestamp
				break
			}
		}
	}
	pnl := (exit - entry) * qty
	if strings.EqualFold(signal.Direction, "SELL") {
		pnl = (entry - exit) * qty
	}
	return strategyTypeTrade{
		Symbol:    signal.Symbol,
		Direction: signal.Direction,
		Entry:     entry,
		Exit:      exit,
		Qty:       qty,
		PnL:       pnl,
		OpenedAt:  candles[entryIdx].Timestamp,
		ClosedAt:  exitTime,
	}, true
}

func summarizeStrategyTypeTrades(req BacktestRequest, trades []strategyTypeTrade) BacktestResponse {
	capital := req.InitialCapital
	if capital <= 0 {
		capital = 100_000
	}
	seed := req.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	peak := capital
	drawdown := 0.0
	wins := 0
	losses := 0
	returns := make([]float64, 0, len(trades))
	for _, tr := range trades {
		capital += tr.PnL
		if tr.PnL >= 0 {
			wins++
		} else {
			losses++
		}
		if peak < capital {
			peak = capital
		}
		if peak > 0 {
			dd := (peak - capital) / peak
			if dd > drawdown {
				drawdown = dd
			}
		}
		if capital > 0 {
			returns = append(returns, tr.PnL/peak)
		}
	}
	winRate := 0.0
	if len(trades) > 0 {
		winRate = float64(wins) / float64(len(trades))
	}
	sharpe := computeSharpe(returns)
	totalReturn := 0.0
	if req.InitialCapital > 0 {
		totalReturn = (capital - req.InitialCapital) / req.InitialCapital
	}
	runID := fmt.Sprintf("bt_%s_%d", req.Strategy, seed)
	return BacktestResponse{
		RunID:         runID,
		Strategy:      req.Strategy,
		Symbols:       req.Symbols,
		Seed:          seed,
		DurationMs:    0,
		TotalTrades:   len(trades),
		WinningTrades: wins,
		LosingTrades:  losses,
		WinRate:       winRate,
		TotalReturn:   totalReturn,
		SharpeRatio:   sharpe,
		MaxDrawdown:   drawdown,
		FinalCapital:  capital,
	}
}

func computeSharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	avg := 0.0
	for _, r := range returns {
		avg += r
	}
	avg /= float64(len(returns))
	variance := 0.0
	for _, r := range returns {
		diff := r - avg
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	if variance <= 0 {
		return 0
	}
	return avg / math.Sqrt(variance)
}

func parseJSONMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func floatFromMap(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		switch typed := v.(type) {
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int:
			return float64(typed)
		case int64:
			return float64(typed)
		case json.Number:
			if f, err := typed.Float64(); err == nil {
				return f
			}
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		switch typed := v.(type) {
		case string:
			return strings.TrimSpace(typed)
		case fmt.Stringer:
			return strings.TrimSpace(typed.String())
		}
	}
	return ""
}
