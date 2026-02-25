package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/calendar"
	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"

	"github.com/jackc/pgx/v5/pgxpool"
)

type marketTools struct {
	pool       *pgxpool.Pool
	mdClient   *marketdata.Client
	events     *eventAggregator
	httpClient *http.Client
}

type marketToolRequest struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

type marketToolResponse struct {
	Output any `json:"output"`
}

func newMarketTools(pool *pgxpool.Pool, ibBridgeURL string) *marketTools {
	mt := &marketTools{
		pool:       pool,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}

	providers := make([]marketdata.ProviderConfig, 0, 3)
	if strings.TrimSpace(ibBridgeURL) != "" {
		providers = append(providers, marketdata.ProviderConfig{
			Name:        marketdata.ProviderIBBridge,
			IBBridgeURL: ibBridgeURL,
			Priority:    1,
			Enabled:     true,
		})
	}
	if alpacaKey := strings.TrimSpace(os.Getenv("ALPACA_API_KEY")); alpacaKey != "" {
		alpacaSecret := strings.TrimSpace(os.Getenv("ALPACA_API_SECRET"))
		providers = append(providers, marketdata.ProviderConfig{
			Name:      marketdata.ProviderAlpaca,
			APIKey:    alpacaKey,
			APISecret: alpacaSecret,
			Priority:  2,
			Enabled:   true,
		})
	}
	if polygonKey := strings.TrimSpace(os.Getenv("POLYGON_API_KEY")); polygonKey != "" {
		providers = append(providers, marketdata.ProviderConfig{
			Name:     marketdata.ProviderPolygon,
			APIKey:   polygonKey,
			Tier:     envStr("POLYGON_TIER", "starter"),
			Priority: 3,
			Enabled:  true,
		})
	}
	if len(providers) > 0 {
		cfg := &marketdata.Config{Providers: providers}
		mdClient, err := marketdata.NewClient(cfg)
		if err == nil {
			mt.mdClient = mdClient
		}
	}

	mt.events = newEventAggregator(mt.httpClient, pool)
	return mt
}

func (m *marketTools) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req marketToolRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		var (
			out any
			err error
		)
		switch req.Tool {
		case utcp.ToolMarketGetQuote:
			var in utcp.GetQuoteInput
			if err = json.Unmarshal(req.Input, &in); err == nil {
				out, err = m.getQuote(r.Context(), in)
			}
		case utcp.ToolMarketGetCandles:
			var in utcp.GetCandlesInput
			if err = json.Unmarshal(req.Input, &in); err == nil {
				out, err = m.getCandles(r.Context(), in)
			}
		case utcp.ToolMarketGetEarnings:
			var in utcp.GetEarningsInput
			if err = json.Unmarshal(req.Input, &in); err == nil {
				out, err = m.getEarnings(r.Context(), in)
			}
		case utcp.ToolMarketGetNews:
			var in utcp.GetNewsInput
			if err = json.Unmarshal(req.Input, &in); err == nil {
				out, err = m.getNews(r.Context(), in)
			}
		default:
			http.Error(w, "unknown tool", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(marketToolResponse{Output: out})
	}
}

func (m *marketTools) getQuote(ctx context.Context, in utcp.GetQuoteInput) (utcp.GetQuoteOutput, error) {
	symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
	if symbol == "" {
		return utcp.GetQuoteOutput{}, errors.New("symbol is required")
	}

	if m.pool != nil {
		var (
			price float64
			ts    time.Time
		)
		err := m.pool.QueryRow(ctx, `
			SELECT price, timestamp
			FROM quotes
			WHERE symbol = $1
			  AND timestamp >= NOW() - INTERVAL '15 minutes'
			ORDER BY timestamp DESC
			LIMIT 1
		`, symbol).Scan(&price, &ts)
		if err == nil {
			return utcp.GetQuoteOutput{
				Symbol:    symbol,
				Price:     price,
				Currency:  "USD",
				Timestamp: ts.UTC(),
			}, nil
		}
	}

	if m.mdClient == nil {
		return utcp.GetQuoteOutput{}, errors.New("market quote provider unavailable")
	}
	quote, err := m.mdClient.GetQuote(ctx, symbol)
	if err != nil {
		return utcp.GetQuoteOutput{}, err
	}
	return utcp.GetQuoteOutput{
		Symbol:    symbol,
		Price:     quote.Price,
		Currency:  "USD",
		Timestamp: quote.Timestamp.UTC(),
	}, nil
}

func (m *marketTools) getCandles(ctx context.Context, in utcp.GetCandlesInput) (utcp.GetCandlesOutput, error) {
	symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
	if symbol == "" {
		return utcp.GetCandlesOutput{}, errors.New("symbol is required")
	}
	tf, mdTF, err := normalizeTimeframe(in.Timeframe)
	if err != nil {
		return utcp.GetCandlesOutput{}, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 200
	}

	from, to := parseOptionalRange(in.From, in.To)
	candles := make([]utcp.Candle, 0, limit)
	if tf == "1d" && m.pool != nil {
		rows, qErr := m.pool.Query(ctx, `
			SELECT timestamp, open, high, low, close, volume
			FROM candles
			WHERE symbol = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`, symbol, limit)
		if qErr == nil {
			for rows.Next() {
				var c utcp.Candle
				if err := rows.Scan(&c.TS, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err == nil {
					candles = append(candles, c)
				}
			}
			rows.Close()
		}
		slices.Reverse(candles)
	}
	if len(candles) == 0 {
		if m.mdClient == nil {
			return utcp.GetCandlesOutput{}, errors.New("market candle provider unavailable")
		}
		mdCandles, err := m.mdClient.GetCandles(ctx, symbol, mdTF, limit)
		if err != nil {
			return utcp.GetCandlesOutput{}, err
		}
		candles = make([]utcp.Candle, 0, len(mdCandles))
		for _, c := range mdCandles {
			candles = append(candles, utcp.Candle{
				TS:     c.Timestamp.UTC(),
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
			})
		}
	}

	if !from.IsZero() || !to.IsZero() {
		filtered := candles[:0]
		for _, c := range candles {
			if !from.IsZero() && c.TS.Before(from) {
				continue
			}
			if !to.IsZero() && c.TS.After(to) {
				continue
			}
			filtered = append(filtered, c)
		}
		candles = filtered
	}
	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}

	return utcp.GetCandlesOutput{
		Symbol:    symbol,
		Timeframe: tf,
		Candles:   candles,
	}, nil
}

func (m *marketTools) getEarnings(ctx context.Context, in utcp.GetEarningsInput) (utcp.GetEarningsOutput, error) {
	symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
	if symbol == "" {
		return utcp.GetEarningsOutput{}, errors.New("symbol is required")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	events, err := m.events.getEarnings(ctx, symbol, limit)
	if err != nil {
		return utcp.GetEarningsOutput{}, err
	}
	return utcp.GetEarningsOutput{
		Symbol:   symbol,
		Earnings: events,
	}, nil
}

func (m *marketTools) getNews(ctx context.Context, in utcp.GetNewsInput) (utcp.GetNewsOutput, error) {
	symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
	if symbol == "" {
		return utcp.GetNewsOutput{}, errors.New("symbol is required")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	news, err := m.events.getNews(ctx, symbol, limit, in.From, in.To)
	if err != nil {
		return utcp.GetNewsOutput{}, err
	}
	return utcp.GetNewsOutput{
		Symbol: symbol,
		News:   news,
	}, nil
}

func normalizeTimeframe(raw string) (string, marketdata.Timeframe, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "1d", "1d ":
		return "1d", marketdata.Timeframe1Day, nil
	case "1m", "1":
		return "1m", marketdata.Timeframe1Min, nil
	case "5m", "5":
		return "5m", marketdata.Timeframe5Min, nil
	case "15m", "15":
		return "15m", marketdata.Timeframe15Min, nil
	case "1h", "60":
		return "1h", marketdata.Timeframe1Hour, nil
	case "1w":
		return "1d", marketdata.Timeframe1Week, nil
	default:
		return "", "", fmt.Errorf("unsupported timeframe %q; use 1m,5m,15m,1h,1d", raw)
	}
}

func parseOptionalRange(fromRaw, toRaw string) (time.Time, time.Time) {
	parse := func(raw string) time.Time {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return time.Time{}
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			return t.UTC()
		}
		return time.Time{}
	}
	return parse(fromRaw), parse(toRaw)
}

type eventAggregator struct {
	httpClient    *http.Client
	polygonKey    string
	finnhubKey    string
	failClosed    bool
	calendarStore *calendar.Store
	store         *eventStore
}

func newEventAggregator(httpClient *http.Client, pool *pgxpool.Pool) *eventAggregator {
	storeDir := envStr("CALENDAR_STORE_DIR", "data/calendar")
	store, _ := calendar.OpenStore(storeDir)
	return &eventAggregator{
		httpClient:    httpClient,
		polygonKey:    strings.TrimSpace(os.Getenv("POLYGON_API_KEY")),
		finnhubKey:    strings.TrimSpace(os.Getenv("FINNHUB_API_KEY")),
		failClosed:    strings.EqualFold(os.Getenv("APP_ENV"), "production") || strings.EqualFold(os.Getenv("ENV"), "production"),
		calendarStore: store,
		store:         newEventStore(pool),
	}
}

func (e *eventAggregator) getEarnings(ctx context.Context, symbol string, limit int) ([]utcp.EarningsEntry, error) {
	if out, err := e.getPolygonEarnings(ctx, symbol, limit); err == nil && len(out) > 0 {
		if pErr := e.store.SaveEarnings(ctx, symbol, "polygon", out); pErr != nil {
			observability.LogEvent(ctx, "warn", "events.persist_earnings_failed", map[string]any{
				"symbol": symbol,
				"source": "polygon",
				"error":  pErr.Error(),
			})
		}
		return out, nil
	}
	if out, err := e.getFinnhubEarnings(ctx, symbol, limit); err == nil && len(out) > 0 {
		if pErr := e.store.SaveEarnings(ctx, symbol, "finnhub", out); pErr != nil {
			observability.LogEvent(ctx, "warn", "events.persist_earnings_failed", map[string]any{
				"symbol": symbol,
				"source": "finnhub",
				"error":  pErr.Error(),
			})
		}
		return out, nil
	}
	if e.failClosed {
		return nil, errors.New("earnings provider unavailable and fail-closed is enabled")
	}
	return []utcp.EarningsEntry{}, nil
}

func (e *eventAggregator) getNews(ctx context.Context, symbol string, limit int, fromRaw, toRaw string) ([]utcp.NewsEntry, error) {
	if out, err := e.getPolygonNews(ctx, symbol, limit); err == nil && len(out) > 0 {
		if pErr := e.store.SaveNews(ctx, symbol, "polygon", out); pErr != nil {
			observability.LogEvent(ctx, "warn", "events.persist_news_failed", map[string]any{
				"symbol": symbol,
				"source": "polygon",
				"error":  pErr.Error(),
			})
		}
		return out, nil
	}
	if out, err := e.getFinnhubNews(ctx, symbol, limit, fromRaw, toRaw); err == nil && len(out) > 0 {
		if pErr := e.store.SaveNews(ctx, symbol, "finnhub", out); pErr != nil {
			observability.LogEvent(ctx, "warn", "events.persist_news_failed", map[string]any{
				"symbol": symbol,
				"source": "finnhub",
				"error":  pErr.Error(),
			})
		}
		return out, nil
	}
	macro := e.getCalendarMacro(limit)
	if len(macro) > 0 {
		if pErr := e.store.SaveMacroNews(ctx, "calendar", macro); pErr != nil {
			observability.LogEvent(ctx, "warn", "events.persist_news_failed", map[string]any{
				"symbol": symbol,
				"source": "calendar",
				"error":  pErr.Error(),
			})
		}
		return macro, nil
	}
	if e.failClosed {
		return nil, errors.New("news provider unavailable and fail-closed is enabled")
	}
	return []utcp.NewsEntry{}, nil
}

func (e *eventAggregator) getPolygonEarnings(ctx context.Context, symbol string, limit int) ([]utcp.EarningsEntry, error) {
	if e.polygonKey == "" {
		return nil, errors.New("polygon key missing")
	}
	endpoint := "https://api.polygon.io/vX/reference/financials"
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("ticker", symbol)
	q.Set("order", "desc")
	q.Set("sort", "filing_date")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("apiKey", e.polygonKey)
	u.RawQuery = q.Encode()

	var payload struct {
		Results []struct {
			FilingDate string `json:"filing_date"`
		} `json:"results"`
	}
	if err := e.fetchJSON(ctx, u.String(), &payload); err != nil {
		return nil, err
	}
	out := make([]utcp.EarningsEntry, 0, len(payload.Results))
	for _, row := range payload.Results {
		out = append(out, utcp.EarningsEntry{Date: row.FilingDate})
	}
	return out, nil
}

func (e *eventAggregator) getFinnhubEarnings(ctx context.Context, symbol string, limit int) ([]utcp.EarningsEntry, error) {
	if e.finnhubKey == "" {
		return nil, errors.New("finnhub key missing")
	}
	now := time.Now().UTC()
	from := now.AddDate(-1, 0, 0).Format("2006-01-02")
	to := now.AddDate(0, 1, 0).Format("2006-01-02")
	u := fmt.Sprintf("https://finnhub.io/api/v1/calendar/earnings?symbol=%s&from=%s&to=%s&token=%s",
		url.QueryEscape(symbol), from, to, url.QueryEscape(e.finnhubKey))
	var payload struct {
		EarningsCalendar []struct {
			Date        string  `json:"date"`
			EPSActual   float64 `json:"epsActual"`
			EPSEstimate float64 `json:"epsEstimate"`
		} `json:"earningsCalendar"`
	}
	if err := e.fetchJSON(ctx, u, &payload); err != nil {
		return nil, err
	}
	out := make([]utcp.EarningsEntry, 0, min(limit, len(payload.EarningsCalendar)))
	for _, row := range payload.EarningsCalendar {
		if len(out) >= limit {
			break
		}
		sp := 0.0
		if row.EPSEstimate != 0 {
			sp = ((row.EPSActual - row.EPSEstimate) / row.EPSEstimate) * 100
		}
		out = append(out, utcp.EarningsEntry{
			Date:        row.Date,
			EPSActual:   row.EPSActual,
			EPSEstimate: row.EPSEstimate,
			SurprisePct: sp,
		})
	}
	return out, nil
}

func (e *eventAggregator) getPolygonNews(ctx context.Context, symbol string, limit int) ([]utcp.NewsEntry, error) {
	if e.polygonKey == "" {
		return nil, errors.New("polygon key missing")
	}
	u := fmt.Sprintf("https://api.polygon.io/v2/reference/news?ticker=%s&limit=%d&order=desc&sort=published_utc&apiKey=%s",
		url.QueryEscape(symbol), limit, url.QueryEscape(e.polygonKey))
	var payload struct {
		Results []struct {
			PublishedUTC string `json:"published_utc"`
			Title        string `json:"title"`
			Description  string `json:"description"`
			ArticleURL   string `json:"article_url"`
			Publisher    struct {
				Name string `json:"name"`
			} `json:"publisher"`
		} `json:"results"`
	}
	if err := e.fetchJSON(ctx, u, &payload); err != nil {
		return nil, err
	}
	out := make([]utcp.NewsEntry, 0, len(payload.Results))
	for _, row := range payload.Results {
		ts, _ := time.Parse(time.RFC3339, row.PublishedUTC)
		out = append(out, utcp.NewsEntry{
			Timestamp: ts.UTC(),
			Headline:  row.Title,
			Summary:   row.Description,
			URL:       row.ArticleURL,
			Source:    row.Publisher.Name,
			Category:  "company_news",
		})
	}
	return out, nil
}

func (e *eventAggregator) getFinnhubNews(ctx context.Context, symbol string, limit int, fromRaw, toRaw string) ([]utcp.NewsEntry, error) {
	if e.finnhubKey == "" {
		return nil, errors.New("finnhub key missing")
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7).Format("2006-01-02")
	to := now.Format("2006-01-02")
	if strings.TrimSpace(fromRaw) != "" {
		from = strings.TrimSpace(fromRaw)
	}
	if strings.TrimSpace(toRaw) != "" {
		to = strings.TrimSpace(toRaw)
	}
	u := fmt.Sprintf("https://finnhub.io/api/v1/company-news?symbol=%s&from=%s&to=%s&token=%s",
		url.QueryEscape(symbol), url.QueryEscape(from), url.QueryEscape(to), url.QueryEscape(e.finnhubKey))
	var payload []struct {
		Datetime int64  `json:"datetime"`
		Headline string `json:"headline"`
		Summary  string `json:"summary"`
		URL      string `json:"url"`
		Source   string `json:"source"`
		Category string `json:"category"`
	}
	if err := e.fetchJSON(ctx, u, &payload); err != nil {
		return nil, err
	}
	out := make([]utcp.NewsEntry, 0, min(limit, len(payload)))
	for i := range payload {
		if len(out) >= limit {
			break
		}
		out = append(out, utcp.NewsEntry{
			Timestamp: time.Unix(payload[i].Datetime, 0).UTC(),
			Headline:  payload[i].Headline,
			Summary:   payload[i].Summary,
			URL:       payload[i].URL,
			Source:    payload[i].Source,
			Category:  payload[i].Category,
		})
	}
	return out, nil
}

func (e *eventAggregator) getCalendarMacro(limit int) []utcp.NewsEntry {
	if e.calendarStore == nil {
		return nil
	}
	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC().Add(7 * 24 * time.Hour)
	events := e.calendarStore.Query(from, to, "US", "USD", calendar.ImpactHigh)
	out := make([]utcp.NewsEntry, 0, min(limit, len(events)))
	for _, ev := range events {
		if len(out) >= limit {
			break
		}
		out = append(out, utcp.NewsEntry{
			Timestamp: ev.ScheduledAt.UTC(),
			Headline:  ev.Title,
			Summary:   fmt.Sprintf("%s (%s)", ev.Category, ev.Impact),
			Source:    ev.Source,
			Category:  "macro",
		})
	}
	return out
}

func (e *eventAggregator) fetchJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("provider HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
