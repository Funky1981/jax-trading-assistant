package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"jax-trading-assistant/libs/marketdata"
)

func TestCanonicalMarketLoadersSeparateQuoteAndCandleTemporalBounds(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	suffix := uuid.NewString()[:8]
	source := "core-readiness-fixture"
	policy := marketDataSafetyPolicy{AllowedSources: []string{source}, Timeframe: "1h", QuoteMaxAge: time.Minute,
		LatestCompletedCandleMaxAge: 90 * time.Minute, CandleHistoryLookback: 7 * 24 * time.Hour, allowNonProductionSources: true}
	cleanup := func(symbol string) {
		t.Helper()
		_, _ = pool.Exec(context.Background(), `DELETE FROM quotes WHERE symbol=$1`, symbol)
		_, _ = pool.Exec(context.Background(), `DELETE FROM candles WHERE symbol=$1`, symbol)
	}
	insertCandle := func(symbol string, start, ingested time.Time, timeframe, semantics, classification string) {
		t.Helper()
		_, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification,ingested_at) VALUES($1,$2,99,101,98,100,1,100,$3,$4,$5,$6,$7)`, symbol, start, timeframe, source, semantics, classification, ingested)
		if err != nil {
			t.Fatalf("insert candle: %v", err)
		}
	}

	// Quote age is independent from the much longer candle-history lookback.
	for _, tc := range []struct {
		name string
		age  time.Duration
		want bool
	}{{"30-seconds-fresh", 30 * time.Second, true}, {"61-seconds-stale", 61 * time.Second, false}} {
		t.Run("quote/"+tc.name, func(t *testing.T) {
			symbol := "Q" + suffix + map[bool]string{true: "A", false: "B"}[tc.want]
			t.Cleanup(func() { cleanup(symbol) })
			asOf := time.Now().UTC().Truncate(time.Microsecond)
			observedAt := asOf.Add(-tc.age)
			if _, err := pool.Exec(ctx, `INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,last_trade_timestamp,exchange,provider,received_at) VALUES($1,100,99,101,1,1,1,$2,$2,'fixture',$3,$2)`, symbol, observedAt, source); err != nil {
				t.Fatal(err)
			}
			_, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf, policy)
			if (err == nil) != tc.want {
				t.Fatalf("quote age %s accepted=%v err=%v, want accepted=%v", tc.age, err == nil, err, tc.want)
			}
		})
	}
	t.Run("quote/persisted-three-timestamps-roundtrip-exactly", func(t *testing.T) {
		symbol := "Q" + suffix + "R"
		t.Cleanup(func() { cleanup(symbol) })
		requestStartedAt := time.Date(2026, 9, 29, 14, 0, 0, 122456000, time.UTC)
		receivedAt := requestStartedAt.Add(time.Millisecond)
		quoteAt := receivedAt.Add(77 * time.Millisecond)
		tradeAt := receivedAt.Add(60 * time.Millisecond)
		asOf := receivedAt.Add(100 * time.Millisecond)
		quote := &marketdata.Quote{Symbol: symbol, Price: 100, Bid: 99, Ask: 101, BidSize: 1, AskSize: 1, Volume: 1, Timestamp: quoteAt, TradeTimestamp: tradeAt, Exchange: "fixture"}
		if err := persistQuoteObservation(ctx, pool, quote, source, requestStartedAt, receivedAt); err != nil {
			t.Fatal(err)
		}
		got, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf, policy)
		if err != nil {
			t.Fatal(err)
		}
		if !got.ProviderAt.Equal(quoteAt) || !got.LastProviderAt.Equal(tradeAt) || !got.ReceivedAt.Equal(receivedAt) {
			t.Fatalf("timestamp roundtrip mismatch: quote=%s trade=%s received=%s", got.ProviderAt, got.LastProviderAt, got.ReceivedAt)
		}
		if got.FreshnessAge != 40*time.Millisecond {
			t.Fatalf("paired observation freshness=%s, want oldest source age 40ms", got.FreshnessAge)
		}
		if _, err := loadCanonicalQuoteObservation(ctx, pool, symbol, receivedAt.Add(50*time.Millisecond), policy); err == nil {
			t.Fatal("77ms-future provider quote was accepted before AvailableAt")
		}
	})
	t.Run("quote/legacy-missing-trade-provenance-fails-closed", func(t *testing.T) {
		symbol := "Q" + suffix + "L"
		t.Cleanup(func() { cleanup(symbol) })
		asOf := time.Now().UTC().Truncate(time.Microsecond)
		if _, err := pool.Exec(ctx, `INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,exchange,provider,received_at) VALUES($1,100,99,101,1,1,1,$2,'fixture',$3,$2)`, symbol, asOf.Add(-time.Second), source); err != nil {
			t.Fatal(err)
		}
		if _, err := loadCanonicalQuoteObservation(ctx, pool, symbol, asOf, policy); err == nil {
			t.Fatal("legacy quote without trade timestamp provenance was accepted")
		}
		var tradeAt *time.Time
		if err := pool.QueryRow(ctx, `SELECT last_trade_timestamp FROM quotes WHERE symbol=$1`, symbol).Scan(&tradeAt); err != nil || tradeAt != nil {
			t.Fatalf("legacy timestamp was rewritten: value=%v err=%v", tradeAt, err)
		}
	})

	// An interval-start hourly candle is unavailable until its hour completes;
	// an earlier ingestion timestamp is rejected even when read after completion.
	t.Run("candle/partial-and-early-ingestion-rejected", func(t *testing.T) {
		symbol := "C" + suffix + "A"
		t.Cleanup(func() { cleanup(symbol) })
		start := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
		insertCandle(symbol, start, start.Add(15*time.Minute), "1h", "interval_start", "real")
		for _, asOf := range []time.Time{start.Add(30 * time.Minute), start.Add(61 * time.Minute)} {
			got, err := loadBoundedCandles(ctx, pool, symbol, asOf, policy, 20)
			if err != nil || len(got) != 0 {
				t.Fatalf("partial/early-ingested candle accepted at %s: got=%+v err=%v", asOf, got, err)
			}
		}
	})
	t.Run("candle/completed-and-late-ingested-accepted", func(t *testing.T) {
		symbol := "C" + suffix + "B"
		t.Cleanup(func() { cleanup(symbol) })
		start := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
		insertCandle(symbol, start, start.Add(time.Hour+time.Second), "1h", "interval_start", "real")
		got, err := loadBoundedCandles(ctx, pool, symbol, start.Add(61*time.Minute), policy, 20)
		if err != nil || len(got) != 1 || !got[0].CompletedAt.Equal(start.Add(time.Hour)) {
			t.Fatalf("completed candle result=%+v err=%v", got, err)
		}
	})
	t.Run("candle/20-history-across-sessions-with-fresh-latest", func(t *testing.T) {
		symbol := "C" + suffix + "C"
		t.Cleanup(func() { cleanup(symbol) })
		asOf := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
		latestStart := asOf.Add(-90 * time.Minute)
		for i := 19; i >= 0; i-- {
			start := latestStart.Add(-time.Duration(i) * 8 * time.Hour)
			insertCandle(symbol, start, start.Add(time.Hour+time.Second), "1h", "interval_start", "real")
		}
		got, err := loadBoundedCandles(ctx, pool, symbol, asOf, policy, 20)
		if err != nil || len(got) != 20 {
			t.Fatalf("20-candle history result count=%d err=%v", len(got), err)
		}
		if got[0].FreshnessAge != 30*time.Minute || got[len(got)-1].FreshnessAge <= 90*time.Minute {
			t.Fatalf("unexpected newest/oldest candle ages: newest=%s oldest=%s", got[0].FreshnessAge, got[len(got)-1].FreshnessAge)
		}
	})
	t.Run("candle/stale-latest-fails", func(t *testing.T) {
		symbol := "C" + suffix + "D"
		t.Cleanup(func() { cleanup(symbol) })
		asOf := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
		start := asOf.Add(-3 * time.Hour)
		insertCandle(symbol, start, start.Add(time.Hour), "1h", "interval_start", "real")
		if _, err := loadBoundedCandles(ctx, pool, symbol, asOf, policy, 20); err == nil {
			t.Fatal("latest completed candle older than 90m was accepted")
		}
	})
	t.Run("candle/fixture-and-unknown-semantics-excluded", func(t *testing.T) {
		symbol := "C" + suffix + "E"
		t.Cleanup(func() { cleanup(symbol) })
		asOf := time.Now().UTC().Truncate(time.Microsecond)
		start := asOf.Add(-time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification,ingested_at) VALUES($1,$2,99,101,98,100,1,100,'1h','alpaca','interval_start','synthetic',$3)`, symbol, start, start.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		unknownStart := start.Add(-2 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification,ingested_at) VALUES($1,$2,99,101,98,100,1,100,'1h','alpaca','provider_observation','real',$3)`, symbol, unknownStart, unknownStart.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		strictPolicy := policy
		strictPolicy.AllowedSources = []string{"alpaca"}
		strictPolicy.allowNonProductionSources = false
		got, err := loadBoundedCandles(ctx, pool, symbol, asOf, strictPolicy, 20)
		if err != nil || len(got) != 0 {
			t.Fatalf("synthetic/test-fixture candle was not excluded: got=%+v err=%v", got, err)
		}
	})
}

func TestMarketDataSafetyPolicyRequiresSplitBounds(t *testing.T) {
	t.Setenv("JAX_MARKET_DATA_ALLOWED_SOURCES", "alpaca")
	t.Setenv("JAX_MARKET_DATA_TIMEFRAME", "1h")
	t.Setenv("JAX_MARKET_DATA_QUOTE_MAX_AGE", "60s")
	t.Setenv("JAX_MARKET_DATA_LATEST_CANDLE_MAX_AGE", "90m")
	t.Setenv("JAX_MARKET_DATA_CANDLE_LOOKBACK", "7d")
	t.Setenv("JAX_MARKET_DATA_MAX_AGE", "30d") // legacy value must not override the split contract.
	policy, err := marketDataSafetyPolicyFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if policy.QuoteMaxAge != time.Minute || policy.LatestCompletedCandleMaxAge != 90*time.Minute || policy.CandleHistoryLookback != 7*24*time.Hour {
		t.Fatalf("split policy bounds=%+v", policy)
	}
	t.Setenv("JAX_MARKET_DATA_QUOTE_MAX_AGE", "")
	if _, err := marketDataSafetyPolicyFromEnv(); err == nil {
		t.Fatal("missing split quote bound was accepted")
	}
	t.Setenv("JAX_MARKET_DATA_QUOTE_MAX_AGE", "60s")
	t.Setenv("JAX_MARKET_DATA_ALLOWED_SOURCES", "alpaca,synthetic-fixture")
	if _, err := marketDataSafetyPolicyFromEnv(); err == nil {
		t.Fatal("non-production source was accepted in production policy")
	}
}

func TestCandleIntervalUsesConservativeDailyCompletion(t *testing.T) {
	hourly, err := candleInterval("1h")
	if err != nil || hourly.duration != time.Hour || hourly.semantics != "interval_start" {
		t.Fatalf("hourly interval=%+v err=%v", hourly, err)
	}
	daily, err := candleInterval("1d")
	if err != nil || daily.duration != 24*time.Hour {
		t.Fatalf("daily interval=%+v err=%v", daily, err)
	}
	if _, err := candleInterval("1w"); err == nil {
		t.Fatal("unsupported timeframe was accepted")
	}
}
