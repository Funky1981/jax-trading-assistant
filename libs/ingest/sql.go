package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// QuoteData represents a market quote for SQL storage
type QuoteData struct {
	Symbol    string
	Price     float64
	Bid       float64
	Ask       float64
	BidSize   int64
	AskSize   int64
	Volume    int64
	Timestamp time.Time
	Exchange  string
}

// CandleData represents OHLCV candle data for SQL storage
type CandleData struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
	VWAP      float64
}

const (
	// StoreQuoteQuery is the SQL for upserting a quote
	StoreQuoteQuery = `
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

	// StoreCandleQuery is the SQL for upserting a candle
	StoreCandleQuery = `
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
)

// StoreQuote stores a quote in the database
func StoreQuote(ctx context.Context, db *sql.DB, quote QuoteData) error {
	_, err := db.ExecContext(ctx, StoreQuoteQuery,
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

// StoreCandles stores multiple candles in the database
func StoreCandles(ctx context.Context, db *sql.DB, candles []CandleData) error {
	if len(candles) == 0 {
		return nil
	}

	stmt, err := db.PrepareContext(ctx, StoreCandleQuery)
	if err != nil {
		return fmt.Errorf("prepare candle statement: %w", err)
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
			return fmt.Errorf("store candle for %s at %v: %w", candle.Symbol, candle.Timestamp, err)
		}
	}

	return nil
}
