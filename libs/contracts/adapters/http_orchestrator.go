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

// HTTPOrchestrator implements services.Orchestrator using HTTP calls
type HTTPOrchestrator struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPOrchestrator creates a new HTTP-based orchestrator client
func NewHTTPOrchestrator(baseURL string) *HTTPOrchestrator {
	return &HTTPOrchestrator{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for AI operations
		},
	}
}

// RunOrchestration executes an orchestration query
func (c *HTTPOrchestrator) RunOrchestration(ctx context.Context, userID, query string) (*domain.OrchestrationRun, error) {
	url := fmt.Sprintf("%s/api/v1/orchestrate", c.baseURL)

	reqBody, err := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"query":   query,
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var run domain.OrchestrationRun
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &run, nil
}

// GetRunHistory returns recent orchestration runs
func (c *HTTPOrchestrator) GetRunHistory(ctx context.Context, userID string, limit int) ([]domain.OrchestrationRun, error) {
	url := fmt.Sprintf("%s/api/v1/runs?user_id=%s&limit=%d", c.baseURL, userID, limit)

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

	var runs []domain.OrchestrationRun
	if err := json.NewDecoder(resp.Body).Decode(&runs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return runs, nil
}

// Health checks if the service is healthy
func (c *HTTPOrchestrator) Health(ctx context.Context) error {
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
