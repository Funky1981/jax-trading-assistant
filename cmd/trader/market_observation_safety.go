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
	AllowedSources            []string
	Timeframe                 string
	MaxAge                    time.Duration
	allowNonProductionSources bool
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
	maxAgeRaw := strings.TrimSpace(os.Getenv("JAX_MARKET_DATA_MAX_AGE"))
	if maxAgeRaw == "" {
		return marketDataSafetyPolicy{}, fmt.Errorf("JAX_MARKET_DATA_MAX_AGE is required")
	}
	maxAge, err := time.ParseDuration(maxAgeRaw)
	if err != nil || maxAge <= 0 {
		return marketDataSafetyPolicy{}, fmt.Errorf("JAX_MARKET_DATA_MAX_AGE must be a positive duration")
	}
	policy.MaxAge = maxAge
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

func (policy marketDataSafetyPolicy) validateObservation(observation marketdata.EconomicObservation, symbol, timeframe string, asOf time.Time, requireQuoteSides bool) error {
	return observation.ValidateForMode(symbol, timeframe, asOf, policy.MaxAge, policy.AllowedSources, requireQuoteSides, policy.allowNonProductionSources)
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
		SELECT symbol,provider,timestamp,received_at,bid,ask,price
		FROM quotes
		WHERE UPPER(symbol)=UPPER($1)
		  AND provider = ANY($2::text[])
		  AND timestamp <= $3
		  AND received_at <= $3
		AND timestamp >= $3 - make_interval(secs => $4::double precision)
		ORDER BY timestamp DESC, received_at DESC, provider ASC
		LIMIT 1
	`, symbol, policy.AllowedSources, asOf.UTC(), policy.MaxAge.Seconds()).Scan(&observation.Symbol, &observation.Source, &observation.ProviderAt, &observation.ReceivedAt, &observation.Bid, &observation.Ask, &observation.Last)
	if err != nil {
		return marketdata.EconomicObservation{}, fmt.Errorf("load bounded provider quote: %w", err)
	}
	observation.Mode, observation.AsOf, observation.Provenance = "QUOTE", asOf.UTC(), "persisted-provider-quote"
	observation.FreshnessAge = asOf.Sub(observation.ProviderAt)
	if err := policy.validateObservation(observation, symbol, "", asOf, true); err != nil {
		return marketdata.EconomicObservation{}, err
	}
	return observation, nil
}

func loadBoundedCandles(ctx context.Context, pool *pgxpool.Pool, symbol string, asOf time.Time, policy marketDataSafetyPolicy, limit int) ([]marketdata.EconomicObservation, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("candle limit must be positive")
	}
	rows, err := pool.Query(ctx, `
		SELECT symbol,source,timestamp,ingested_at,timeframe,close::float8
		FROM candles
		WHERE UPPER(symbol)=UPPER($1)
		  AND timeframe=$2
		  AND source = ANY($3::text[])
		  AND timestamp <= $4
		  AND ingested_at >= timestamp
		  AND ingested_at <= $4
		  AND ($7::boolean OR timestamp_semantics NOT IN ('unknown', 'test_fixture'))
		  AND ($7::boolean OR market_data_classification NOT IN ('synthetic', 'fixture', 'test'))
		  AND timestamp >= $4 - make_interval(secs => $5::double precision)
		ORDER BY timestamp DESC, ingested_at DESC, source ASC
		LIMIT $6
	`, symbol, policy.Timeframe, policy.AllowedSources, asOf.UTC(), policy.MaxAge.Seconds(), limit, policy.allowNonProductionSources)
	if err != nil {
		return nil, fmt.Errorf("load bounded provider candles: %w", err)
	}
	defer rows.Close()
	observations := make([]marketdata.EconomicObservation, 0, limit)
	for rows.Next() {
		var observation marketdata.EconomicObservation
		if err := rows.Scan(&observation.Symbol, &observation.Source, &observation.ProviderAt, &observation.ReceivedAt, &observation.Timeframe, &observation.Last); err != nil {
			return nil, err
		}
		observation.Mode, observation.AsOf, observation.Provenance = "MODELED_CANDLE_CLOSE", asOf.UTC(), "persisted-provider-candle"
		observation.FreshnessAge = asOf.Sub(observation.ProviderAt)
		if err := policy.validateObservation(observation, symbol, policy.Timeframe, asOf, false); err != nil {
			continue
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return observations, nil
}
