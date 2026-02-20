package agent0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps Agent0 Python service for planning/execution
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// Option configures the Agent0 client
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// New creates a new Agent0 client
func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("agent0 client: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("agent0 client: base url must include scheme and host")
	}

	c := &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // LLM inference can be slow; allow up to 5 minutes
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// PlanRequest contains inputs for Agent0 planning
type PlanRequest struct {
	Task        string                 `json:"task"`
	Context     string                 `json:"context"`
	Symbol      string                 `json:"symbol,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
	Memories    []Memory               `json:"memories,omitempty"`
}

// Memory represents recalled context
type Memory struct {
	Summary string                 `json:"summary"`
	Type    string                 `json:"type"`
	Symbol  string                 `json:"symbol,omitempty"`
	Tags    []string               `json:"tags,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// PlanResponse contains Agent0 plan output
type PlanResponse struct {
	Summary        string   `json:"summary"`
	Steps          []string `json:"steps"`
	Action         string   `json:"action"`
	Confidence     float64  `json:"confidence"`
	ReasoningNotes string   `json:"reasoning_notes"`
}

// Plan sends a planning request to Agent0
func (c *Client) Plan(ctx context.Context, req PlanRequest) (PlanResponse, error) {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/v1/plan"})

	body, err := json.Marshal(req)
	if err != nil {
		return PlanResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return PlanResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return PlanResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return PlanResponse{}, fmt.Errorf("agent0 plan failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result PlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PlanResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

// ExecuteRequest contains tool execution inputs
type ExecuteRequest struct {
	Plan         PlanResponse           `json:"plan"`
	ToolContext  map[string]interface{} `json:"tool_context,omitempty"`
	MaxToolCalls int                    `json:"max_tool_calls,omitempty"`
}

// ToolCall represents a single tool invocation
type ToolCall struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     string                 `json:"result"`
	Success    bool                   `json:"success"`
}

// ExecuteResponse contains tool execution results
type ExecuteResponse struct {
	ToolCalls []ToolCall `json:"tool_calls"`
	Success   bool       `json:"success"`
	Summary   string     `json:"summary"`
}

// Execute sends an execution request to Agent0
func (c *Client) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/v1/execute"})

	if req.MaxToolCalls == 0 {
		req.MaxToolCalls = 10 // Default limit
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ExecuteResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return ExecuteResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ExecuteResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return ExecuteResponse{}, fmt.Errorf("agent0 execute failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ExecuteResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

// Health checks if the Agent0 service is available
func (c *Client) Health(ctx context.Context) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/health"})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("agent0 unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}
