package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client handles HTTP communication with jax-orchestrator service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new orchestrator client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OrchestrateRequest represents a request to trigger orchestration
type OrchestrateRequest struct {
	SignalID    string `json:"signal_id"`
	Symbol      string `json:"symbol"`
	TriggerType string `json:"trigger_type"`
	Context     string `json:"context"`
}

// OrchestrateResponse represents the orchestration response
type OrchestrateResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// TriggerOrchestration triggers an orchestration run for a signal
func (c *Client) TriggerOrchestration(ctx context.Context, signalID uuid.UUID, symbol, context string) (uuid.UUID, error) {
	req := OrchestrateRequest{
		SignalID:    signalID.String(),
		Symbol:      symbol,
		TriggerType: "signal",
		Context:     context,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/orchestrate", bytes.NewReader(body))
	if err != nil {
		return uuid.Nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return uuid.Nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("orchestrator returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result OrchestrateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, fmt.Errorf("decode response: %w", err)
	}

	runID, err := uuid.Parse(result.RunID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse run ID: %w", err)
	}

	return runID, nil
}
