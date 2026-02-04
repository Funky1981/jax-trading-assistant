package ib

import (
	"context"
	"fmt"
	"log"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

// Provider implements the marketdata.Provider interface using the IB Bridge
type Provider struct {
	client *Client
	name   string
}

// NewProvider creates a new IB provider that uses the Python bridge
func NewProvider(bridgeURL string) (*Provider, error) {
	client := NewClient(Config{
		BaseURL: bridgeURL,
		Timeout: 30 * time.Second,
	})

	// Check health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IB bridge at %s: %w", bridgeURL, err)
	}

	if !health.Connected {
		log.Printf("Warning: IB Bridge is running but not connected to IB Gateway")
	}

	log.Printf("IB provider connected to bridge at %s (status: %s)", bridgeURL, health.Status)

	return &Provider{
		client: client,
		name:   "ib",
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.name
}

// GetQuote gets a real-time quote for a symbol
func (p *Provider) GetQuote(ctx context.Context, symbol string) (*marketdata.Quote, error) {
	resp, err := p.client.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	timestamp, err := ParseTime(resp.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return &marketdata.Quote{
		Symbol:    resp.Symbol,
		Price:     resp.Price,
		Bid:       resp.Bid,
		Ask:       resp.Ask,
		BidSize:   resp.BidSize,
		AskSize:   resp.AskSize,
		Volume:    resp.Volume,
		Timestamp: timestamp,
		Exchange:  resp.Exchange,
	}, nil
}

// GetCandles gets historical candles for a symbol
func (p *Provider) GetCandles(ctx context.Context, symbol string, timeframe marketdata.Timeframe, from, to time.Time) ([]marketdata.Candle, error) {
	// Convert timeframe to IB format
	barSize := convertTimeframeToBarSize(timeframe)
	duration := calculateDuration(from, to)

	req := &CandlesRequest{
		Duration:   duration,
		BarSize:    barSize,
		WhatToShow: "TRADES",
	}

	resp, err := p.client.GetCandles(ctx, symbol, req)
	if err != nil {
		return nil, err
	}

	// Convert to marketdata.Candle
	candles := make([]marketdata.Candle, 0, len(resp.Candles))
	for _, c := range resp.Candles {
		timestamp, err := ParseTime(c.Timestamp)
		if err != nil {
			log.Printf("Warning: failed to parse timestamp %s: %v", c.Timestamp, err)
			continue
		}

		candles = append(candles, marketdata.Candle{
			Symbol:    symbol,
			Timestamp: timestamp,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			VWAP:      c.VWAP,
		})
	}

	return candles, nil
}

// GetTrades gets recent trades for a symbol
func (p *Provider) GetTrades(ctx context.Context, symbol string, from, to time.Time) ([]marketdata.Trade, error) {
	// IB doesn't provide tick-by-tick trades through the bridge in this implementation
	// You could extend the Python bridge to support this if needed
	return nil, fmt.Errorf("GetTrades not implemented for IB provider")
}

// SubscribeQuotes subscribes to real-time quotes
func (p *Provider) SubscribeQuotes(ctx context.Context, symbols []string) (<-chan marketdata.StreamUpdate, error) {
	// WebSocket streaming would be implemented here
	// For now, return not implemented
	return nil, fmt.Errorf("SubscribeQuotes not implemented - use WebSocket client directly")
}

// Close closes the provider connection
func (p *Provider) Close() error {
	// No persistent connection to close
	return nil
}

// Helper functions

func convertTimeframeToBarSize(tf marketdata.Timeframe) string {
	switch tf {
	case marketdata.Timeframe1Min:
		return "1 min"
	case marketdata.Timeframe5Min:
		return "5 mins"
	case marketdata.Timeframe15Min:
		return "15 mins"
	case marketdata.Timeframe1Hour:
		return "1 hour"
	case marketdata.Timeframe1Day:
		return "1 day"
	case marketdata.Timeframe1Week:
		return "1 week"
	default:
		return "1 min"
	}
}

func calculateDuration(from, to time.Time) string {
	duration := to.Sub(from)

	days := int(duration.Hours() / 24)
	if days > 365 {
		years := days / 365
		return fmt.Sprintf("%d Y", years)
	}
	if days > 30 {
		months := days / 30
		return fmt.Sprintf("%d M", months)
	}
	if days > 0 {
		return fmt.Sprintf("%d D", days)
	}

	// Default to 1 day
	return "1 D"
}
