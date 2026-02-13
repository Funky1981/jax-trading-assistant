package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jax-trading-assistant/libs/contracts/domain"
)

// HTTPTradeExecutor implements services.TradeExecutor using HTTP calls
type HTTPTradeExecutor struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPTradeExecutor creates a new HTTP-based trade executor client
func NewHTTPTradeExecutor(baseURL string) *HTTPTradeExecutor {
	return &HTTPTradeExecutor{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteSignal converts a signal into one or more orders
func (c *HTTPTradeExecutor) ExecuteSignal(ctx context.Context, signal domain.Signal) ([]domain.Order, error) {
	url := fmt.Sprintf("%s/api/v1/execute", c.baseURL)

	reqBody, err := json.Marshal(signal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var orders []domain.Order
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return orders, nil
}

// GetPositions returns current positions
func (c *HTTPTradeExecutor) GetPositions(ctx context.Context, accountID string) ([]domain.Position, error) {
	url := fmt.Sprintf("%s/api/v1/positions?account_id=%s", c.baseURL, accountID)

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

	var positions []domain.Position
	if err := json.NewDecoder(resp.Body).Decode(&positions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return positions, nil
}

// GetPortfolio returns full portfolio snapshot
func (c *HTTPTradeExecutor) GetPortfolio(ctx context.Context, accountID string) (*domain.Portfolio, error) {
	url := fmt.Sprintf("%s/api/v1/portfolio?account_id=%s", c.baseURL, accountID)

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

	var portfolio domain.Portfolio
	if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &portfolio, nil
}

// Health checks if the service is healthy
func (c *HTTPTradeExecutor) Health(ctx context.Context) error {
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
