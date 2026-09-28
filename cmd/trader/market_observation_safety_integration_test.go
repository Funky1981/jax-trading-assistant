package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanonicalMarketLoadersEnforceQuoteAndCandleTimeBounds(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suffix := uuid.NewString()
	symbol := "T" + suffix[:7]
	asOf := time.Now().UTC().Truncate(time.Microsecond)
	policy := marketDataSafetyPolicy{AllowedSources: []string{"core-readiness-fixture"}, Timeframe: "1m", MaxAge: 5 * time.Minute, allowNonProductionSources: true}
	if _, err := pool.Exec(ctx, `INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,exchange,provider,received_at) VALUES($1,100,99,101,1,1,1,$2,'fixture-exchange','core-readiness-fixture',$2)`, symbol, asOf.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM quotes WHERE symbol=$1`, symbol) })
	quote, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf, policy)
	if err != nil || quote.Source != "core-readiness-fixture" || quote.ProviderAt.After(asOf) {
		t.Fatalf("valid quote=%+v err=%v", quote, err)
	}
	if _, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf.Add(-2*time.Minute), policy); err == nil {
		t.Fatal("future quote was accepted")
	}
	if _, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf.Add(10*time.Minute), policy); err == nil {
		t.Fatal("stale quote was accepted")
	}
	if _, err := loadCanonicalQuoteObservation(ctx, pool, symbol+"X", asOf, policy); err == nil {
		t.Fatal("wrong symbol quote was accepted")
	}

	for _, row := range []struct {
		at        time.Time
		frame     string
		source    string
		semantics string
		class     string
	}{
		{asOf.Add(-time.Minute), "1m", "core-readiness-fixture", "test_fixture", "synthetic"},
		{asOf.Add(time.Minute), "1m", "core-readiness-fixture", "test_fixture", "synthetic"},
		{asOf.Add(-time.Minute), "1d", "core-readiness-fixture", "test_fixture", "synthetic"},
		{asOf.Add(-time.Minute), "1m", "unapproved-provider", "provider_observation", "real"},
		{asOf.Add(-10 * time.Minute), "1m", "core-readiness-fixture", "test_fixture", "synthetic"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification,ingested_at) VALUES($1,$2,99,101,98,100,1,100,$3,$4,$5,$6,$2) ON CONFLICT(symbol,timestamp) DO NOTHING`, symbol, row.at, row.frame, row.source, row.semantics, row.class); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM candles WHERE symbol=$1`, symbol) })
	candles, err := loadBoundedCandles(ctx, pool, symbol, asOf, policy, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || !candles[0].ProviderAt.Equal(asOf.Add(-time.Minute)) || candles[0].Source != "core-readiness-fixture" || candles[0].Timeframe != "1m" {
		t.Fatalf("eligible candle set=%+v, want only the fresh, bounded, allowed-source 1m row", candles)
	}
}

func TestMarketDataSafetyPolicyFailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("JAX_MARKET_DATA_ALLOWED_SOURCES", "")
	t.Setenv("JAX_MARKET_DATA_TIMEFRAME", "")
	t.Setenv("JAX_MARKET_DATA_MAX_AGE", "")
	if _, err := marketDataSafetyPolicyFromEnv(); err == nil {
		t.Fatal("empty production market-data policy was accepted")
	}
	t.Setenv("JAX_MARKET_DATA_ALLOWED_SOURCES", "alpaca,synthetic-fixture")
	t.Setenv("JAX_MARKET_DATA_TIMEFRAME", "1d")
	t.Setenv("JAX_MARKET_DATA_MAX_AGE", "24h")
	if _, err := marketDataSafetyPolicyFromEnv(); err == nil {
		t.Fatal("non-production source was accepted in production policy")
	}
}
