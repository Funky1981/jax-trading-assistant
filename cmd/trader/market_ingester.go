// market_ingester.go — Part of cmd/trader (package main).
// Runs the market-data ingestion loop in-process, replacing the jax-market
// microservice (ADR-0012 Phase 6).
//
// On startup it backfills 250 daily candles for each symbol via the IB Bridge.
// Every ingestInterval seconds it refreshes quotes.
// Every 24 hours it re-backfills candles to keep them current.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ingesterConfig holds market ingestion settings read from env.
type ingesterConfig struct {
	IBBridgeURL     string
	Symbols         []string
	IngestInterval  time.Duration
	CandleBackfill  int
	StaleQuoteSecs  int
}

func loadIngesterConfig() ingesterConfig {
	symbols := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "META", "NVDA", "AMD", "NFLX", "SPY"}
	if raw := os.Getenv("MARKET_SYMBOLS"); raw != "" {
		parts := strings.Split(raw, ",")
		symbols = symbols[:0]
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				symbols = append(symbols, s)
			}
		}
	}

	intervalSecs := 60
	if raw := os.Getenv("INGEST_INTERVAL_SECS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			intervalSecs = n
		}
	}

	backfill := 250
	if raw := os.Getenv("CANDLE_BACKFILL"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			backfill = n
		}
	}

	stale := 300
	if raw := os.Getenv("STALE_QUOTE_SECS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			stale = n
		}
	}

	return ingesterConfig{
		IBBridgeURL:    os.Getenv("IB_BRIDGE_URL"),
		Symbols:        symbols,
		IngestInterval: time.Duration(intervalSecs) * time.Second,
		CandleBackfill: backfill,
		StaleQuoteSecs: stale,
	}
}

// startMarketIngester runs the market data ingestion loop until ctx is cancelled.
// It uses the existing pgxpool for DB writes.
func startMarketIngester(ctx context.Context, pool *pgxpool.Pool) {
	cfg := loadIngesterConfig()
	if cfg.IBBridgeURL == "" {
		log.Println("market ingester: IB_BRIDGE_URL not set — skipping")
		return
	}

	mdCfg := &marketdata.Config{
		Providers: []marketdata.ProviderConfig{
			{
				Name:        marketdata.ProviderIBBridge,
				IBBridgeURL: cfg.IBBridgeURL,
				Priority:    1,
				Enabled:     true,
			},
		},
		Symbols: cfg.Symbols,
	}

	client, err := marketdata.NewClient(mdCfg)
	if err != nil {
		log.Printf("market ingester: failed to create client: %v — skipping", err)
		return
	}
	defer client.Close()

	log.Printf("market ingester started: %d symbol(s), interval=%s, backfill=%d",
		len(cfg.Symbols), cfg.IngestInterval, cfg.CandleBackfill)

	var lastCandleRun time.Time

	runOnce := func(forceBackfill bool) {
		shouldBackfill := forceBackfill || time.Since(lastCandleRun) >= 24*time.Hour

		for _, symbol := range cfg.Symbols {
			if err := ingestQuote(ctx, pool, client, symbol); err != nil {
				log.Printf("market ingester: quote %s: %v", symbol, err)
			}
			if shouldBackfill {
				if err := ingestCandles(ctx, pool, client, symbol, cfg.CandleBackfill); err != nil {
					log.Printf("market ingester: candles %s: %v", symbol, err)
				}
			}
		}

		if shouldBackfill {
			lastCandleRun = time.Now()
		}
	}

	// Immediate startup run with backfill.
	runOnce(true)

	ticker := time.NewTicker(cfg.IngestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("market ingester stopped")
			return
		case <-ticker.C:
			runOnce(false)
		}
	}
}

// ingestQuote fetches and upserts the latest quote for symbol.
func ingestQuote(ctx context.Context, pool *pgxpool.Pool, client *marketdata.Client, symbol string) error {
	quote, err := client.GetQuote(ctx, symbol)
	if err != nil {
		return fmt.Errorf("get quote: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (symbol) DO UPDATE SET
			price      = EXCLUDED.price,
			bid        = EXCLUDED.bid,
			ask        = EXCLUDED.ask,
			bid_size   = EXCLUDED.bid_size,
			ask_size   = EXCLUDED.ask_size,
			volume     = EXCLUDED.volume,
			timestamp  = EXCLUDED.timestamp,
			exchange   = EXCLUDED.exchange,
			updated_at = NOW()`,
		quote.Symbol, quote.Price, quote.Bid, quote.Ask,
		quote.BidSize, quote.AskSize, quote.Volume, quote.Timestamp, quote.Exchange,
	)
	if err != nil {
		return fmt.Errorf("upsert quote: %w", err)
	}

	log.Printf("market ingester: quote %s price=%.2f", symbol, quote.Price)
	return nil
}

// ingestCandles backfills daily candles for symbol.
func ingestCandles(ctx context.Context, pool *pgxpool.Pool, client *marketdata.Client, symbol string, limit int) error {
	if limit <= 0 {
		return nil
	}

	candles, err := client.GetCandles(ctx, symbol, marketdata.Timeframe1Day, limit)
	if err != nil {
		return fmt.Errorf("get candles: %w", err)
	}
	if len(candles) == 0 {
		return nil
	}

	for _, c := range candles {
		if _, err := pool.Exec(ctx, `
			INSERT INTO candles (symbol, timestamp, open, high, low, close, volume, vwap)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (symbol, timestamp) DO UPDATE SET
				open   = EXCLUDED.open,
				high   = EXCLUDED.high,
				low    = EXCLUDED.low,
				close  = EXCLUDED.close,
				volume = EXCLUDED.volume,
				vwap   = EXCLUDED.vwap`,
			c.Symbol, c.Timestamp, c.Open, c.High, c.Low, c.Close, c.Volume, c.VWAP,
		); err != nil {
			return fmt.Errorf("upsert candle %s@%s: %w", symbol, c.Timestamp.Format("2006-01-02"), err)
		}
	}

	log.Printf("market ingester: candles %s count=%d", symbol, len(candles))
	return nil
}
