package ingester

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/services/jax-market/internal/config"
)

// Ingester handles market data ingestion and storage
type Ingester struct {
	client          *marketdata.Client
	db              *sql.DB
	config          *config.Config
	metricsCallback func(successCount, errorCount int, duration time.Duration, staleCount int)
	lastCandleRun   time.Time
}

// New creates a new Ingester
func New(client *marketdata.Client, db *sql.DB, config *config.Config) *Ingester {
	return &Ingester{
		client: client,
		db:     db,
		config: config,
	}
}

// SetMetricsCallback sets a callback function to report metrics
func (i *Ingester) SetMetricsCallback(callback func(successCount, errorCount int, duration time.Duration, staleCount int)) {
	i.metricsCallback = callback
}

// Start begins the ingestion scheduler
func (i *Ingester) Start(ctx context.Context) {
	interval := time.Duration(i.config.IngestInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup with candle backfill
	i.ingestAll(ctx, true)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ingester stopped")
			return
		case <-ticker.C:
			i.ingestAll(ctx, false)
		}
	}
}

// ingestAll ingests data for all configured symbols
func (i *Ingester) ingestAll(ctx context.Context, forceBackfill bool) {
	log.Printf("starting ingestion for %d symbols", len(i.config.Symbols))
	start := time.Now()

	successCount := 0
	errorCount := 0
	staleCount := 0

	shouldBackfill := forceBackfill || i.shouldBackfill()

	for _, symbol := range i.config.Symbols {
		if err := i.ingestQuote(ctx, symbol); err != nil {
			log.Printf("failed to ingest %s: %v", symbol, err)
			errorCount++
		} else {
			successCount++
		}

		if shouldBackfill {
			if err := i.ingestCandles(ctx, symbol, i.config.CandleBackfill); err != nil {
				log.Printf("failed to backfill candles for %s: %v", symbol, err)
				errorCount++
			}
		}
	}

	if shouldBackfill {
		i.lastCandleRun = time.Now().UTC()
	}

	staleCount = i.checkStaleQuotes(ctx)
	if staleCount > 0 {
		log.Printf("stale quote check: %d symbol(s) older than %ds", staleCount, i.config.StaleQuoteSecs)
	}

	duration := time.Since(start)
	log.Printf("ingestion complete: %d success, %d errors in %v", successCount, errorCount, duration)

	// Call metrics callback if set
	if i.metricsCallback != nil {
		i.metricsCallback(successCount, errorCount, duration, staleCount)
	}
}

func (i *Ingester) shouldBackfill() bool {
	if i.lastCandleRun.IsZero() {
		return true
	}
	return time.Since(i.lastCandleRun) >= 24*time.Hour
}

// ingestQuote ingests the latest quote for a symbol
func (i *Ingester) ingestQuote(ctx context.Context, symbol string) error {
	// Fetch current quote
	quote, err := i.client.GetQuote(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get quote: %w", err)
	}

	// Store quote
	if err := i.storeQuote(ctx, quote); err != nil {
		return fmt.Errorf("failed to store quote: %w", err)
	}

	log.Printf("ingested %s: price=$%.2f", symbol, quote.Price)
	return nil
}

// ingestCandles backfills daily candles for a single symbol
func (i *Ingester) ingestCandles(ctx context.Context, symbol string, limit int) error {
	if limit <= 0 {
		return nil
	}

	// Fetch daily candles
	candles, err := i.client.GetCandles(ctx, symbol, marketdata.Timeframe1Day, limit)
	if err != nil {
		return fmt.Errorf("failed to get candles: %w", err)
	}

	// Store candles
	if err := i.storeCandles(ctx, candles); err != nil {
		return fmt.Errorf("failed to store candles: %w", err)
	}

	log.Printf("backfilled %s candles: %d", symbol, len(candles))
	return nil
}

// storeQuote stores a quote in the database
func (i *Ingester) storeQuote(ctx context.Context, quote *marketdata.Quote) error {
	query := `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (symbol) DO UPDATE SET
			price = EXCLUDED.price,
			bid = EXCLUDED.bid,
			ask = EXCLUDED.ask,
			bid_size = EXCLUDED.bid_size,
			ask_size = EXCLUDED.ask_size,
			volume = EXCLUDED.volume,
			timestamp = EXCLUDED.timestamp,
			exchange = EXCLUDED.exchange,
			updated_at = NOW()
	`

	_, err := i.db.ExecContext(ctx, query,
		quote.Symbol,
		quote.Price,
		quote.Bid,
		quote.Ask,
		quote.BidSize,
		quote.AskSize,
		quote.Volume,
		quote.Timestamp,
		quote.Exchange,
	)

	return err
}

// storeCandles stores candles in the database
func (i *Ingester) storeCandles(ctx context.Context, candles []marketdata.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	query := `
		INSERT INTO candles (symbol, timestamp, open, high, low, close, volume, vwap)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (symbol, timestamp) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			vwap = EXCLUDED.vwap
	`

	stmt, err := i.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, candle := range candles {
		_, err := stmt.ExecContext(ctx,
			candle.Symbol,
			candle.Timestamp,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.VWAP,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Ingester) checkStaleQuotes(ctx context.Context) int {
	if i.config.StaleQuoteSecs <= 0 {
		return 0
	}

	query := `
		SELECT COUNT(*)
		FROM quotes
		WHERE updated_at < NOW() - ($1 * INTERVAL '1 second')
	`
	var count int
	if err := i.db.QueryRowContext(ctx, query, i.config.StaleQuoteSecs).Scan(&count); err != nil {
		log.Printf("stale quote check failed: %v", err)
		return 0
	}
	return count
}
