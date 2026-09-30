package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// marketDataSafetyPolicy is explicit configuration, not a strategy choice.
// Empty configuration is intentionally not replaced by permissive defaults.
type marketDataSafetyPolicy struct {
	AllowedSources              []string
	Timeframe                   string
	QuoteMaxAge                 time.Duration
	LatestCompletedCandleMaxAge time.Duration
	CandleHistoryLookback       time.Duration
	allowNonProductionSources   bool
}

func marketDataSafetyPolicyFromEnv() (marketDataSafetyPolicy, error) {
	policy := marketDataSafetyPolicy{
		Timeframe: strings.TrimSpace(os.Getenv("JAX_MARKET_DATA_TIMEFRAME")),
	}
	for _, source := range strings.Split(os.Getenv("JAX_MARKET_DATA_ALLOWED_SOURCES"), ",") {
		if source = strings.ToLower(strings.TrimSpace(source)); source != "" {
			policy.AllowedSources = append(policy.AllowedSources, source)
		}
	}
	sort.Strings(policy.AllowedSources)
	for _, configured := range []struct {
		name   string
		target *time.Duration
	}{
		{"JAX_MARKET_DATA_QUOTE_MAX_AGE", &policy.QuoteMaxAge},
		{"JAX_MARKET_DATA_LATEST_CANDLE_MAX_AGE", &policy.LatestCompletedCandleMaxAge},
		{"JAX_MARKET_DATA_CANDLE_LOOKBACK", &policy.CandleHistoryLookback},
	} {
		raw := strings.TrimSpace(os.Getenv(configured.name))
		value, err := parseMarketDataDuration(raw)
		if raw == "" || err != nil || value <= 0 {
			return marketDataSafetyPolicy{}, fmt.Errorf("%s must be a positive duration", configured.name)
		}
		*configured.target = value
	}
	if len(policy.AllowedSources) == 0 || policy.Timeframe == "" {
		return marketDataSafetyPolicy{}, fmt.Errorf("JAX_MARKET_DATA_ALLOWED_SOURCES and JAX_MARKET_DATA_TIMEFRAME are required")
	}
	for _, source := range policy.AllowedSources {
		if strings.Contains(source, "unknown") || strings.Contains(source, "test") || strings.Contains(source, "synthetic") || strings.Contains(source, "fixture") {
			return marketDataSafetyPolicy{}, fmt.Errorf("non-genuine source %q cannot be configured for economic market data", source)
		}
	}
	return policy, nil
}

func parseMarketDataDuration(raw string) (time.Duration, error) {
	value, err := time.ParseDuration(raw)
	if err == nil {
		return value, nil
	}
	if strings.HasSuffix(raw, "d") {
		days, dayErr := time.ParseDuration(strings.TrimSuffix(raw, "d") + "h")
		if dayErr == nil {
			return days * 24, nil
		}
	}
	return 0, err
}

func (policy marketDataSafetyPolicy) validateObservation(observation marketdata.EconomicObservation, symbol, timeframe string, asOf time.Time, maxAge time.Duration, requireQuoteSides bool) error {
	return observation.ValidateForMode(symbol, timeframe, asOf, maxAge, policy.AllowedSources, requireQuoteSides, policy.allowNonProductionSources)
}

func loadCanonicalQuoteObservation(ctx context.Context, pool *pgxpool.Pool, symbol string, asOf time.Time, policy marketDataSafetyPolicy) (marketdata.EconomicObservation, error) {
	return loadCanonicalQuoteObservationFrom(ctx, pool, symbol, asOf, policy)
}

type marketObservationQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadCanonicalQuoteObservationFrom(ctx context.Context, queryer marketObservationQueryer, symbol string, asOf time.Time, policy marketDataSafetyPolicy) (marketdata.EconomicObservation, error) {
	var observation marketdata.EconomicObservation
	err := queryer.QueryRow(ctx, `
		SELECT symbol,provider,timestamp,last_trade_timestamp,received_at,bid,ask,price
		FROM quotes
		WHERE UPPER(symbol)=UPPER($1)
		  AND provider = ANY($2::text[])
		  AND timestamp <= $3
		  AND last_trade_timestamp <= $3
		  AND received_at <= $3
		AND timestamp >= $3 - make_interval(secs => $4::double precision)
		ORDER BY timestamp DESC, last_trade_timestamp DESC, received_at DESC, provider ASC
		LIMIT 1
	`, symbol, policy.AllowedSources, asOf.UTC(), policy.QuoteMaxAge.Seconds()).Scan(&observation.Symbol, &observation.Source, &observation.ProviderAt, &observation.LastProviderAt, &observation.ReceivedAt, &observation.Bid, &observation.Ask, &observation.Last)
	if err != nil {
		return marketdata.EconomicObservation{}, fmt.Errorf("load bounded provider quote: %w", err)
	}
	observation.Mode, observation.AsOf, observation.Provenance = "QUOTE", asOf.UTC(), "persisted-provider-quote"
	observation.FreshnessAge = asOf.Sub(observation.ProviderAt)
	if tradeAge := asOf.Sub(observation.LastProviderAt); tradeAge > observation.FreshnessAge {
		observation.FreshnessAge = tradeAge
	}
	if err := policy.validateObservation(observation, symbol, "", asOf, policy.QuoteMaxAge, true); err != nil {
		return marketdata.EconomicObservation{}, err
	}
	return observation, nil
}

func loadBoundedCandles(ctx context.Context, pool *pgxpool.Pool, symbol string, asOf time.Time, policy marketDataSafetyPolicy, limit int) ([]marketdata.EconomicObservation, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("candle limit must be positive")
	}
	interval, err := candleInterval(policy.Timeframe)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT symbol,source,timestamp,ingested_at,timeframe,close::float8,timestamp_semantics
		FROM candles
		WHERE UPPER(symbol)=UPPER($1)
		  AND timeframe=$2
		  AND source = ANY($3::text[])
		  AND timestamp <= $4
		  AND ingested_at >= timestamp
		  AND ingested_at <= $4
		  AND timestamp_semantics = $8
		  AND timestamp + make_interval(secs => $9::double precision) <= $4
		  AND ingested_at >= timestamp + make_interval(secs => $9::double precision)
		  AND ($7::boolean OR market_data_classification NOT IN ('synthetic', 'fixture', 'test'))
		  AND timestamp >= $4 - make_interval(secs => $5::double precision)
		ORDER BY timestamp DESC, ingested_at DESC, source ASC
		LIMIT $6
	`, symbol, policy.Timeframe, policy.AllowedSources, asOf.UTC(), policy.CandleHistoryLookback.Seconds(), limit, policy.allowNonProductionSources, interval.semantics, interval.duration.Seconds())
	if err != nil {
		return nil, fmt.Errorf("load bounded provider candles: %w", err)
	}
	defer rows.Close()
	observations := make([]marketdata.EconomicObservation, 0, limit)
	var semantics string
	for rows.Next() {
		var observation marketdata.EconomicObservation
		if err := rows.Scan(&observation.Symbol, &observation.Source, &observation.ProviderAt, &observation.ReceivedAt, &observation.Timeframe, &observation.Last, &semantics); err != nil {
			return nil, err
		}
		observation.CompletedAt = observation.ProviderAt
		if semantics == "interval_start" {
			observation.CompletedAt = observation.ProviderAt.Add(interval.duration)
		}
		observation.Mode, observation.AsOf, observation.Provenance = "MODELED_CANDLE_CLOSE", asOf.UTC(), "persisted-provider-candle"
		observation.FreshnessAge = asOf.Sub(observation.CompletedAt)
		if err := policy.validateObservation(observation, symbol, policy.Timeframe, asOf, policy.CandleHistoryLookback, false); err != nil {
			continue
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(observations) > 0 && observations[0].FreshnessAge > policy.LatestCompletedCandleMaxAge {
		return nil, fmt.Errorf("latest completed candle is stale: age %s exceeds %s", observations[0].FreshnessAge, policy.LatestCompletedCandleMaxAge)
	}
	return observations, nil
}

type candleIntervalSpec struct {
	semantics string
	duration  time.Duration
}

func candleInterval(timeframe string) (candleIntervalSpec, error) {
	switch strings.TrimSpace(timeframe) {
	case "1m":
		return candleIntervalSpec{semantics: "interval_start", duration: time.Minute}, nil
	case "5m":
		return candleIntervalSpec{semantics: "interval_start", duration: 5 * time.Minute}, nil
	case "15m":
		return candleIntervalSpec{semantics: "interval_start", duration: 15 * time.Minute}, nil
	case "1h":
		return candleIntervalSpec{semantics: "interval_start", duration: time.Hour}, nil
	case "1d":
		// Provider session-close semantics are not canonical in this table. Use
		// a deliberately conservative 24-hour interval from the stored start.
		return candleIntervalSpec{semantics: "interval_start", duration: 24 * time.Hour}, nil
	default:
		return candleIntervalSpec{}, fmt.Errorf("unsupported economic candle timeframe %q", timeframe)
	}
}
