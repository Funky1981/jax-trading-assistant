package services

import (
	"context"
	"time"
)

// MarketData provides real-time and historical market data
type MarketData interface {
	// GetQuote returns current quote for a symbol
	GetQuote(ctx context.Context, symbol string) (*Quote, error)

	// GetHistoricalBars returns historical price bars
	GetHistoricalBars(ctx context.Context, symbol string, start, end time.Time, timeframe string) ([]Bar, error)

	// Health checks if the service is healthy
	Health(ctx context.Context) error
}

// Quote represents a real-time market quote
type Quote struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Last      float64   `json:"last"`
	Volume    int64     `json:"volume"`
}

// Bar represents a price bar (OHLCV)
type Bar struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
}
