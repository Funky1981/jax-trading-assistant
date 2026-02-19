package ingester

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	sharedIngest "jax-trading-assistant/libs/ingest"
	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/services/jax-ingest/internal/config"
)

// Ingester handles market data ingestion and storage
type Ingester struct {
	client *marketdata.Client
	db     *sql.DB
	config *config.Config
}

// New creates a new Ingester
func New(client *marketdata.Client, db *sql.DB, config *config.Config) *Ingester {
	return &Ingester{
		client: client,
		db:     db,
		config: config,
	}
}

// Start begins the ingestion scheduler
func (i *Ingester) Start(ctx context.Context) {
	interval := time.Duration(i.config.IngestInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	i.ingestAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ingester stopped")
			return
		case <-ticker.C:
			i.ingestAll(ctx)
		}
	}
}

// ingestAll ingests data for all configured symbols
func (i *Ingester) ingestAll(ctx context.Context) {
	log.Printf("starting ingestion for %d symbols", len(i.config.Symbols))
	start := time.Now()

	successCount := 0
	errorCount := 0

	for _, symbol := range i.config.Symbols {
		if err := i.ingestSymbol(ctx, symbol); err != nil {
			log.Printf("failed to ingest %s: %v", symbol, err)
			errorCount++
		} else {
			successCount++
		}
	}

	duration := time.Since(start)
	log.Printf("ingestion complete: %d success, %d errors in %v", successCount, errorCount, duration)
}

// ingestSymbol ingests quote and candles for a single symbol
func (i *Ingester) ingestSymbol(ctx context.Context, symbol string) error {
	// Fetch current quote
	quote, err := i.client.GetQuote(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get quote: %w", err)
	}

	// Store quote
	if err := i.storeQuote(ctx, quote); err != nil {
		return fmt.Errorf("failed to store quote: %w", err)
	}

	// Fetch daily candles (last 30 days)
	candles, err := i.client.GetCandles(ctx, symbol, marketdata.Timeframe1Day, 30)
	if err != nil {
		return fmt.Errorf("failed to get candles: %w", err)
	}

	// Store candles
	if err := i.storeCandles(ctx, candles); err != nil {
		return fmt.Errorf("failed to store candles: %w", err)
	}

	log.Printf("ingested %s: price=$%.2f, %d candles", symbol, quote.Price, len(candles))
	return nil
}

// storeQuote stores a quote in the database
func (i *Ingester) storeQuote(ctx context.Context, quote *marketdata.Quote) error {
	quoteData := sharedIngest.QuoteData{
		Symbol:    quote.Symbol,
		Price:     quote.Price,
		Bid:       quote.Bid,
		Ask:       quote.Ask,
		BidSize:   quote.BidSize,
		AskSize:   quote.AskSize,
		Volume:    quote.Volume,
		Timestamp: quote.Timestamp,
		Exchange:  quote.Exchange,
	}
	return sharedIngest.StoreQuote(ctx, i.db, quoteData)
}

// storeCandles stores candles in the database
func (i *Ingester) storeCandles(ctx context.Context, candles []marketdata.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	// Convert to shared ingest format
	candleData := make([]sharedIngest.CandleData, len(candles))
	for idx, candle := range candles {
		candleData[idx] = sharedIngest.CandleData{
			Symbol:    candle.Symbol,
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			VWAP:      candle.VWAP,
		}
	}

	return sharedIngest.StoreCandles(ctx, i.db, candleData)
}
