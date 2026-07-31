package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/libs/marketdata"
)

var safeMarketSymbol = regexp.MustCompile(`^[A-Z][A-Z0-9.]{0,14}$`)

type sourcedCandleFetcher interface {
	GetCandlesWithSource(context.Context, string, marketdata.Timeframe, int) ([]marketdata.Candle, string, error)
}

type genuineCandleCollectionRequest struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	From      string `json:"from"`
	Limit     int    `json:"limit"`
}

type genuineCandleCollectionResult struct {
	Symbol                   string     `json:"symbol"`
	Provider                 string     `json:"provider"`
	Mode                     string     `json:"mode"`
	Timeframe                string     `json:"timeframe"`
	TimestampSemantics       string     `json:"timestampSemantics"`
	RegularTradingHours      *bool      `json:"regularTradingHours,omitempty"`
	MarketDataClassification string     `json:"marketDataClassification"`
	AdjustedState            string     `json:"adjustedState"`
	ProviderTimezone         string     `json:"providerTimezone"`
	Historical               bool       `json:"historical"`
	RequestedFrom            time.Time  `json:"requestedFrom"`
	Received                 int        `json:"received"`
	Persisted                int        `json:"persisted"`
	Earliest                 *time.Time `json:"earliest,omitempty"`
	Latest                   *time.Time `json:"latest,omitempty"`
}

func genuineCandleCollectionHandler(pool *pgxpool.Pool, fetcher sourcedCandleFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req genuineCandleCollectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid candle collection request", http.StatusBadRequest)
			return
		}
		from, err := time.Parse(time.RFC3339, strings.TrimSpace(req.From))
		if err != nil {
			http.Error(w, "from must be an RFC3339 timestamp", http.StatusBadRequest)
			return
		}
		if req.Limit <= 0 {
			req.Limit = 168
		}
		if req.Limit > 1000 {
			http.Error(w, "limit must not exceed 1000", http.StatusBadRequest)
			return
		}
		result, err := collectGenuineCandles(r.Context(), pool, fetcher, req.Symbol, req.Timeframe, from, req.Limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		jsonOK(w, result)
	}
}

func collectGenuineCandles(ctx context.Context, pool *pgxpool.Pool, fetcher sourcedCandleFetcher, rawSymbol, rawTimeframe string, from time.Time, limit int) (genuineCandleCollectionResult, error) {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if !safeMarketSymbol.MatchString(symbol) {
		return genuineCandleCollectionResult{}, errors.New("invalid market symbol")
	}
	timeframe, label, err := collectionTimeframe(rawTimeframe)
	if err != nil {
		return genuineCandleCollectionResult{}, err
	}
	candles, provider, err := fetcher.GetCandlesWithSource(ctx, symbol, timeframe, limit)
	if err != nil {
		return genuineCandleCollectionResult{}, fmt.Errorf("genuine market-data provider unavailable: %w", err)
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if !genuineProvider(provider) {
		return genuineCandleCollectionResult{}, fmt.Errorf("rejected non-genuine candle source %q", provider)
	}
	prepared := prepareGenuineCandles(symbol, from.UTC(), time.Now().UTC(), candles)
	semantics, regularHours := providerCandleSemantics(provider)
	adjustedState, providerTimezone := providerCandleProvenance(provider)
	result := genuineCandleCollectionResult{Symbol: symbol, Provider: provider, Mode: "paper_read_only", Timeframe: label,
		TimestampSemantics: semantics, RegularTradingHours: regularHours, MarketDataClassification: "unknown", AdjustedState: adjustedState, ProviderTimezone: providerTimezone, Historical: true, RequestedFrom: from.UTC(), Received: len(prepared)}
	if len(prepared) == 0 {
		return result, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin candle persistence: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, c := range prepared {
		_, err = tx.Exec(ctx, `INSERT INTO candles (symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,regular_trading_hours,market_data_classification,adjusted_state,provider_timezone,ingested_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
			ON CONFLICT(symbol,timestamp) DO UPDATE SET open=EXCLUDED.open,high=EXCLUDED.high,low=EXCLUDED.low,close=EXCLUDED.close,volume=EXCLUDED.volume,vwap=EXCLUDED.vwap,timeframe=EXCLUDED.timeframe,source=EXCLUDED.source,timestamp_semantics=EXCLUDED.timestamp_semantics,regular_trading_hours=EXCLUDED.regular_trading_hours,market_data_classification=EXCLUDED.market_data_classification,adjusted_state=EXCLUDED.adjusted_state,provider_timezone=EXCLUDED.provider_timezone,ingested_at=NOW()
			WHERE (candles.source=EXCLUDED.source AND candles.timeframe=EXCLUDED.timeframe) OR candles.source='unknown'`,
			c.Symbol, c.Timestamp, c.Open, c.High, c.Low, c.Close, c.Volume, c.VWAP, label, provider, semantics, regularHours, "unknown", adjustedState, providerTimezone)
		if err != nil {
			return result, fmt.Errorf("persist genuine candle %s at %s: %w", symbol, c.Timestamp.Format(time.RFC3339), err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit genuine candles: %w", err)
	}
	err = pool.QueryRow(ctx, `SELECT COUNT(*),MIN(timestamp),MAX(timestamp) FROM candles WHERE symbol=$1 AND timeframe=$2 AND source=$3 AND timestamp >= $4`, symbol, label, provider, from.UTC()).Scan(&result.Persisted, &result.Earliest, &result.Latest)
	if err != nil {
		return result, fmt.Errorf("verify genuine candle persistence: %w", err)
	}
	return result, nil
}

func prepareGenuineCandles(symbol string, from, now time.Time, candles []marketdata.Candle) []marketdata.Candle {
	out := make([]marketdata.Candle, 0, len(candles))
	seen := map[time.Time]bool{}
	for _, c := range candles {
		ts := c.Timestamp.UTC()
		if ts.Before(from) || ts.After(now) || seen[ts] || c.Open <= 0 || c.High <= 0 || c.Low <= 0 || c.Close <= 0 || c.High < c.Low {
			continue
		}
		c.Symbol = symbol
		c.Timestamp = ts
		seen[ts] = true
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}
func genuineProvider(source string) bool {
	return source != "" && source != "unknown" && source != "test" && source != "synthetic" && source != "fixture"
}
func collectionTimeframe(raw string) (marketdata.Timeframe, string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1h", "60":
		return marketdata.Timeframe1Hour, "1h", nil
	case "1d":
		return marketdata.Timeframe1Day, "1d", nil
	default:
		return "", "", errors.New("supported timeframes are 1h and 1d")
	}
}
func providerCandleSemantics(provider string) (string, *bool) {
	if provider == "ib-bridge" {
		v := true
		return "interval_start", &v
	}
	if provider == "alpaca" || provider == "polygon" {
		return "interval_start", nil
	}
	return "provider_timestamp", nil
}

func providerCandleProvenance(provider string) (string, string) {
	// The current provider abstraction does not expose an adjustment selector.
	// Record that honestly instead of inferring split/dividend treatment.
	return "unknown", "UTC"
}
