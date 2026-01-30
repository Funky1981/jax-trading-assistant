package marketdata

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofinance/ib"
)

// IBProvider implements the Provider interface for Interactive Brokers
type IBProvider struct {
	config    IBConfig
	engine    *ib.Engine
	mu        sync.RWMutex
	connected bool
}

// IBConfig holds IB Gateway configuration
type IBConfig struct {
	Host     string // Default: "127.0.0.1"
	Port     int    // 7497 for paper trading, 7496 for live
	ClientID int    // Any integer to identify this connection
}

// NewIBProvider creates a new Interactive Brokers provider
func NewIBProvider(config ProviderConfig) (*IBProvider, error) {
	// Use config values or defaults
	host := config.IBHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.IBPort
	if port == 0 {
		port = 7497 // Paper trading default
	}
	clientID := config.IBClientID
	if clientID == 0 {
		clientID = 1
	}

	p := &IBProvider{
		config: IBConfig{
			Host:     host,
			Port:     port,
			ClientID: clientID,
		},
	}

	// Connect to IB Gateway
	if err := p.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to IB Gateway at %s:%d: %w", host, port, err)
	}

	log.Printf("IB provider connected to %s:%d (client ID: %d)", host, port, clientID)
	return p, nil
}

// connect establishes connection to IB Gateway
func (p *IBProvider) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	// Create connection string
	gateway := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	// Create engine options
	opts := ib.EngineOptions{
		Gateway: gateway,
		Client:  int64(p.config.ClientID),
	}

	// Create and connect engine
	engine, err := ib.NewEngine(opts)
	if err != nil {
		return fmt.Errorf("failed to create IB engine: %w", err)
	}

	p.engine = engine
	p.connected = true
	return nil
}

// Name returns the provider name
func (p *IBProvider) Name() string {
	return "ib"
}

// GetQuote fetches a real-time quote from IB Gateway
func (p *IBProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	p.mu.RLock()
	if !p.connected || p.engine == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected to IB Gateway")
	}
	engine := p.engine
	p.mu.RUnlock()

	// Create stock contract
	contract := ib.Contract{
		Symbol:       symbol,
		SecurityType: "STK",
		Exchange:     "SMART",
		Currency:     "USD",
	}

	// Create instrument manager for market data
	mgr, err := ib.NewInstrumentManager(engine, contract)
	if err != nil {
		return nil, fmt.Errorf("failed to create instrument manager: %w", err)
	}
	defer mgr.Close()

	// Wait for data with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			// Try to get whatever data we have
			bid := mgr.Bid()
			ask := mgr.Ask()
			last := mgr.Last()

			if last == 0 && bid == 0 && ask == 0 {
				return nil, fmt.Errorf("no market data received for %s (check market hours and data subscriptions)", symbol)
			}

			price := last
			if price == 0 {
				price = (bid + ask) / 2 // Use mid price if no last trade
			}

			return &Quote{
				Symbol:    symbol,
				Price:     price,
				Bid:       bid,
				Ask:       ask,
				Timestamp: time.Now(),
				Exchange:  "SMART",
			}, nil
		case <-ticker.C:
			// Check if we have data
			last := mgr.Last()
			bid := mgr.Bid()
			ask := mgr.Ask()

			if last > 0 || (bid > 0 && ask > 0) {
				price := last
				if price == 0 {
					price = (bid + ask) / 2
				}

				return &Quote{
					Symbol:    symbol,
					Price:     price,
					Bid:       bid,
					Ask:       ask,
					Timestamp: time.Now(),
					Exchange:  "SMART",
				}, nil
			}
		}
	}
}

// GetCandles fetches historical OHLCV data
func (p *IBProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	p.mu.RLock()
	if !p.connected || p.engine == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected to IB Gateway")
	}
	engine := p.engine
	p.mu.RUnlock()

	// Create stock contract
	contract := ib.Contract{
		Symbol:       symbol,
		SecurityType: "STK",
		Exchange:     "SMART",
		Currency:     "USD",
	}

	// Convert timeframe to IB bar size
	barSize := timeframeToIBBarSize(timeframe)

	// Calculate duration based on limit and timeframe
	duration := calculateIBDuration(timeframe, limit)

	// Create historical data request
	req := ib.RequestHistoricalData{
		Contract:    contract,
		Duration:    duration,
		BarSize:     barSize,
		WhatToShow:  ib.HistTrades,
		UseRTH:      true,
		EndDateTime: time.Now(),
	}

	// Create historical data manager
	mgr, err := ib.NewHistoricalDataManager(engine, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create historical data manager: %w", err)
	}
	defer mgr.Close()

	// Wait for data with timeout
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			items := mgr.Items()
			if len(items) == 0 {
				return nil, fmt.Errorf("no historical data received for %s", symbol)
			}
			return convertIBBarsToCandles(symbol, items, limit), nil
		case <-ticker.C:
			// Check manager error
			if err := mgr.FatalError(); err != nil {
				return nil, fmt.Errorf("historical data error: %w", err)
			}
			items := mgr.Items()
			if len(items) > 0 {
				return convertIBBarsToCandles(symbol, items, limit), nil
			}
		}
	}
}

// convertIBBarsToCandles converts IB historical bars to our Candle type
func convertIBBarsToCandles(symbol string, items []ib.HistoricalDataItem, limit int) []Candle {
	candles := make([]Candle, 0, len(items))
	for _, bar := range items {
		candles = append(candles, Candle{
			Symbol:    symbol,
			Timestamp: bar.Date,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    bar.Volume,
			VWAP:      bar.WAP,
		})
	}

	// Trim to limit if we got more
	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}

	return candles
}

// calculateIBDuration calculates the IB duration string based on timeframe and limit
func calculateIBDuration(tf Timeframe, limit int) string {
	switch tf {
	case Timeframe1Min:
		// Each trading day has ~390 minutes
		days := limit/390 + 1
		if days > 1 {
			return fmt.Sprintf("%d D", days)
		}
		return "1 D"
	case Timeframe5Min:
		days := limit/78 + 1 // ~78 5-min bars per day
		if days > 1 {
			return fmt.Sprintf("%d D", days)
		}
		return "1 D"
	case Timeframe15Min:
		days := limit/26 + 1 // ~26 15-min bars per day
		if days > 1 {
			return fmt.Sprintf("%d D", days)
		}
		return "1 D"
	case Timeframe1Hour:
		days := limit/7 + 1 // ~7 hourly bars per day
		return fmt.Sprintf("%d D", days)
	case Timeframe1Day:
		return fmt.Sprintf("%d D", limit)
	default:
		return fmt.Sprintf("%d D", limit)
	}
}

// timeframeToIBBarSize converts our Timeframe to IB bar size type
func timeframeToIBBarSize(tf Timeframe) ib.HistDataBarSize {
	switch tf {
	case Timeframe1Min:
		return ib.HistBarSize1Min
	case Timeframe5Min:
		return ib.HistBarSize5Min
	case Timeframe15Min:
		return ib.HistBarSize15Min
	case Timeframe1Hour:
		return ib.HistBarSize1Hour
	case Timeframe1Day:
		return ib.HistBarSize1Day
	default:
		return ib.HistBarSize1Day
	}
}

// GetTrades fetches recent trades (not typically used with IB Gateway)
func (p *IBProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	return nil, fmt.Errorf("GetTrades not available via IB Gateway - use GetQuote for real-time data")
}

// GetEarnings fetches earnings data (not available via IB market data)
func (p *IBProvider) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	return nil, fmt.Errorf("earnings data not available via IB market data API")
}

// StreamQuotes streams real-time quotes
func (p *IBProvider) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	p.mu.RLock()
	if !p.connected || p.engine == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected to IB Gateway")
	}
	engine := p.engine
	p.mu.RUnlock()

	updates := make(chan StreamUpdate, 100)

	go func() {
		defer close(updates)

		// Create instrument managers for each symbol
		type symbolManager struct {
			symbol string
			mgr    *ib.InstrumentManager
		}
		managers := make([]symbolManager, 0, len(symbols))

		for _, symbol := range symbols {
			contract := ib.Contract{
				Symbol:       symbol,
				SecurityType: "STK",
				Exchange:     "SMART",
				Currency:     "USD",
			}

			mgr, err := ib.NewInstrumentManager(engine, contract)
			if err != nil {
				log.Printf("failed to create instrument manager for %s: %v", symbol, err)
				continue
			}
			managers = append(managers, symbolManager{symbol: symbol, mgr: mgr})
		}

		// Cleanup on exit
		defer func() {
			for _, sm := range managers {
				sm.mgr.Close()
			}
		}()

		// Poll for updates
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		lastPrices := make(map[string]float64)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, sm := range managers {
					last := sm.mgr.Last()
					bid := sm.mgr.Bid()
					ask := sm.mgr.Ask()

					price := last
					if price == 0 && bid > 0 && ask > 0 {
						price = (bid + ask) / 2
					}

					// Only send if price changed and is valid
					if price > 0 && price != lastPrices[sm.symbol] {
						lastPrices[sm.symbol] = price
						updates <- StreamUpdate{
							Type: "quote",
							Quote: &Quote{
								Symbol:    sm.symbol,
								Price:     price,
								Bid:       bid,
								Ask:       ask,
								Timestamp: time.Now(),
								Exchange:  "SMART",
							},
						}
					}
				}
			}
		}
	}()

	return updates, nil
}

// HealthCheck verifies connection to IB Gateway
func (p *IBProvider) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.connected || p.engine == nil {
		return fmt.Errorf("not connected to IB Gateway")
	}

	// Check engine state
	if p.engine.State() != ib.EngineReady {
		return fmt.Errorf("IB Gateway engine not ready (state: %v)", p.engine.State())
	}

	return nil
}

// Close disconnects from IB Gateway
func (p *IBProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.engine != nil && p.connected {
		p.engine.Stop()
		p.connected = false
		p.engine = nil
	}
	return nil
}
