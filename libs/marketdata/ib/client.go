package ib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jax-trading-assistant/libs/resilience"
)

// Client is an HTTP client for the Python IB Bridge service
type Client struct {
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *resilience.CircuitBreaker
}

// Config holds configuration for the IB Bridge client
type Config struct {
	BaseURL string        // e.g., "http://localhost:8092"
	Timeout time.Duration // HTTP request timeout
}

// NewClient creates a new IB Bridge HTTP client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	cbConfig := resilience.DefaultConfig("ib-bridge")

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		circuitBreaker: resilience.NewCircuitBreaker(cbConfig),
	}
}

// Health checks the health of the IB Bridge service
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	err := c.get(ctx, "/health", &resp)
	return &resp, err
}

// Connect connects to IB Gateway
func (c *Client) Connect(ctx context.Context, req *ConnectRequest) (*ConnectResponse, error) {
	var resp ConnectResponse
	err := c.post(ctx, "/connect", req, &resp)
	return &resp, err
}

// GetQuote gets a real-time quote for a symbol
func (c *Client) GetQuote(ctx context.Context, symbol string) (*QuoteResponse, error) {
	var resp QuoteResponse
	err := c.get(ctx, fmt.Sprintf("/quotes/%s", symbol), &resp)
	return &resp, err
}

// GetCandles gets historical candles for a symbol
func (c *Client) GetCandles(ctx context.Context, symbol string, req *CandlesRequest) (*CandlesResponse, error) {
	var resp CandlesResponse
	err := c.post(ctx, fmt.Sprintf("/candles/%s", symbol), req, &resp)
	return &resp, err
}

// PlaceOrder places an order
func (c *Client) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	var resp OrderResponse
	err := c.post(ctx, "/orders", req, &resp)
	return &resp, err
}

// GetPositions gets current positions
func (c *Client) GetPositions(ctx context.Context) (*PositionsResponse, error) {
	var resp PositionsResponse
	err := c.get(ctx, "/positions", &resp)
	return &resp, err
}

// GetAccount gets account information
func (c *Client) GetAccount(ctx context.Context) (*AccountResponse, error) {
	var resp AccountResponse
	err := c.get(ctx, "/account", &resp)
	return &resp, err
}

// get performs a GET request
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	return c.circuitBreaker.Execute(func() error {
		url := c.baseURL + path

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil
	})
}

// post performs a POST request
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.circuitBreaker.Execute(func() error {
		url := c.baseURL + path

		var reqBody io.Reader
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewBuffer(jsonData)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}

		if result != nil {
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
		}

		return nil
	})
}
