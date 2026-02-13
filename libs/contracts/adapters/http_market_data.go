package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"jax-trading-assistant/libs/contracts/services"
)

// HTTPMarketData implements services.MarketData using HTTP calls
type HTTPMarketData struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPMarketData creates a new HTTP-based market data client
func NewHTTPMarketData(baseURL string) *HTTPMarketData {
	return &HTTPMarketData{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetQuote returns current quote for a symbol
func (c *HTTPMarketData) GetQuote(ctx context.Context, symbol string) (*services.Quote, error) {
	url := fmt.Sprintf("%s/api/v1/quotes/%s", c.baseURL, url.PathEscape(symbol))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var quote services.Quote
	if err := json.NewDecoder(resp.Body).Decode(&quote); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &quote, nil
}

// GetHistoricalBars returns historical price bars
func (c *HTTPMarketData) GetHistoricalBars(ctx context.Context, symbol string, start, end time.Time, timeframe string) ([]services.Bar, error) {
	url := fmt.Sprintf("%s/api/v1/bars/%s?start=%s&end=%s&timeframe=%s",
		c.baseURL,
		url.PathEscape(symbol),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		timeframe,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var bars []services.Bar
	if err := json.NewDecoder(resp.Body).Decode(&bars); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return bars, nil
}

// Health checks if the service is healthy
func (c *HTTPMarketData) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service unhealthy (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
