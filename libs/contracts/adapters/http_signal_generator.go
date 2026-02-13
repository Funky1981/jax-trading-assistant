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

// HTTPSignalGenerator implements services.SignalGenerator using HTTP calls
type HTTPSignalGenerator struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPSignalGenerator creates a new HTTP-based signal generator client
func NewHTTPSignalGenerator(baseURL string) *HTTPSignalGenerator {
	return &HTTPSignalGenerator{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateSignals analyzes multiple symbols and returns signals
func (c *HTTPSignalGenerator) GenerateSignals(ctx context.Context, symbols []string) ([]domain.Signal, error) {
	url := fmt.Sprintf("%s/api/v1/signals/generate", c.baseURL)

	reqBody, err := json.Marshal(map[string]interface{}{
		"symbols": symbols,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var signals []domain.Signal
	if err := json.NewDecoder(resp.Body).Decode(&signals); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return signals, nil
}

// GetSignalHistory returns recent signals
func (c *HTTPSignalGenerator) GetSignalHistory(ctx context.Context, symbol string, limit int) ([]domain.Signal, error) {
	url := fmt.Sprintf("%s/api/v1/signals?symbol=%s&limit=%d", c.baseURL, symbol, limit)

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

	var signals []domain.Signal
	if err := json.NewDecoder(resp.Body).Decode(&signals); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return signals, nil
}

// Health checks if the service is healthy
func (c *HTTPSignalGenerator) Health(ctx context.Context) error {
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
