package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IBBridgeProvider implements the Provider interface by calling the Python IB bridge HTTP API.
// The bridge already handles reqMarketDataType(3) (delayed data) and the ib-insync connection.
type IBBridgeProvider struct {
	baseURL    string
	httpClient *http.Client
}

// ibBridgeQuote is the JSON response from GET /quotes/{symbol}
type ibBridgeQuote struct {
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Bid          float64 `json:"bid"`
	Ask          float64 `json:"ask"`
	Volume       int64   `json:"volume"`
	Timestamp    string  `json:"timestamp"`
	DelaySeconds int     `json:"delay_seconds"`
}

// ibBridgeHealth is the JSON response from GET /health
type ibBridgeHealth struct {
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
}

// NewIBBridgeProvider creates a provider that forwards quote requests to the IB Python bridge.
// config.IBBridgeURL should be set to the bridge base URL, e.g. "http://ib-bridge:8092".
func NewIBBridgeProvider(config ProviderConfig) (*IBBridgeProvider, error) {
	url := config.IBBridgeURL
	if url == "" {
		url = "http://ib-bridge:8092"
	}
	p := &IBBridgeProvider{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	return p, nil
}

// Name returns the provider name.
func (p *IBBridgeProvider) Name() string {
	return "ib-bridge"
}

// GetQuote fetches a delayed or real-time quote from the IB bridge.
func (p *IBBridgeProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/quotes/%s", p.baseURL, symbol), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to build request: %v", ErrProviderError, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: ib-bridge unreachable: %v", ErrProviderError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: ib-bridge returned HTTP %d for %s", ErrProviderError, resp.StatusCode, symbol)
	}

	var q ibBridgeQuote
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("%w: failed to decode bridge response: %v", ErrProviderError, err)
	}

	if q.Price == 0 && q.Bid == 0 && q.Ask == 0 {
		return nil, fmt.Errorf("%w: ib-bridge returned zero price for %s", ErrNoData, symbol)
	}

	ts, err := parseBridgeTimestamp(q.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid quote timestamp for %s: %v", ErrProviderError, symbol, err)
	}

	return &Quote{
		Symbol:    symbol,
		Price:     q.Price,
		Bid:       q.Bid,
		Ask:       q.Ask,
		Volume:    q.Volume,
		Timestamp: ts,
		Exchange:  "SMART",
	}, nil
}

// HealthCheck verifies the bridge is reachable and IB-connected.
func (p *IBBridgeProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ib-bridge health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ib-bridge health returned HTTP %d", resp.StatusCode)
	}

	var h ibBridgeHealth
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return fmt.Errorf("ib-bridge health decode failed: %w", err)
	}
	if !h.Connected {
		return fmt.Errorf("ib-bridge is not connected to IB Gateway")
	}
	return nil
}

// GetCandles fetches historical OHLCV bars from the IB bridge GET /candles/{symbol} endpoint.
func (p *IBBridgeProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	reqURL := fmt.Sprintf("%s/candles/%s?limit=%d&timeframe=%s", p.baseURL, symbol, limit, ibBridgeTimeframe(timeframe))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to build candles request: %v", ErrProviderError, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: ib-bridge candles unreachable: %v", ErrProviderError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: ib-bridge returned HTTP %d for candles %s", ErrProviderError, resp.StatusCode, symbol)
	}

	var result struct {
		Symbol  string `json:"symbol"`
		Candles []struct {
			Timestamp string  `json:"timestamp"`
			Open      float64 `json:"open"`
			High      float64 `json:"high"`
			Low       float64 `json:"low"`
			Close     float64 `json:"close"`
			Volume    int64   `json:"volume"`
			VWAP      float64 `json:"vwap"`
		} `json:"candles"`
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode candles response: %v", ErrProviderError, err)
	}

	if len(result.Candles) == 0 {
		return nil, fmt.Errorf("%w: no candle data returned for %s", ErrNoData, symbol)
	}

	candles := make([]Candle, 0, len(result.Candles))
	for _, c := range result.Candles {
		ts, err := parseBridgeTimestamp(c.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid candle timestamp for %s: %v", ErrProviderError, symbol, err)
		}
		candles = append(candles, Candle{
			Symbol:    symbol,
			Timestamp: ts,
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

func ibBridgeTimeframe(timeframe Timeframe) string {
	switch timeframe {
	case Timeframe1Min:
		return "1m"
	case Timeframe5Min:
		return "5m"
	case Timeframe15Min:
		return "15m"
	case Timeframe1Hour:
		return "1h"
	case Timeframe1Day:
		return "1d"
	case Timeframe1Week:
		return "1w"
	default:
		return string(timeframe)
	}
}

func parseBridgeTimestamp(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	layouts := []struct {
		layout string
		utc    bool
	}{
		{time.RFC3339Nano, false},
		{time.RFC3339, false},
		{"2006-01-02T15:04:05.999999999", true},
		{"2006-01-02T15:04:05.999999", true},
		{"2006-01-02T15:04:05", true},
		{"2006-01-02", true},
	}

	for _, item := range layouts {
		ts, err := time.Parse(item.layout, raw)
		if err == nil {
			if item.utc {
				return ts.UTC(), nil
			}
			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format %q", raw)
}

// GetTrades is not supported by the IB bridge — return ErrNoData.
func (p *IBBridgeProvider) GetTrades(_ context.Context, _ string, _ int) ([]Trade, error) {
	return nil, fmt.Errorf("%w: trades not supported by ib-bridge provider", ErrNoData)
}

// GetEarnings is not supported by the IB bridge — return ErrNoData.
func (p *IBBridgeProvider) GetEarnings(_ context.Context, _ string, _ int) ([]Earnings, error) {
	return nil, fmt.Errorf("%w: earnings not supported by ib-bridge provider", ErrNoData)
}

// StreamQuotes is not supported by the IB bridge HTTP API — return ErrNoData.
func (p *IBBridgeProvider) StreamQuotes(_ context.Context, _ []string) (<-chan StreamUpdate, error) {
	return nil, fmt.Errorf("%w: streaming not supported by ib-bridge provider", ErrNoData)
}
