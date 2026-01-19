package utcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
)

type UTCPClient struct {
	providers  map[string]ProviderConfig
	httpClient *http.Client
	local      *LocalRegistry
}

type ClientOption func(*UTCPClient)

func WithHTTPClient(c *http.Client) ClientOption {
	return func(u *UTCPClient) {
		u.httpClient = c
	}
}

func WithLocalRegistry(r *LocalRegistry) ClientOption {
	return func(u *UTCPClient) {
		u.local = r
	}
}

func NewUTCPClient(cfg ProvidersConfig, opts ...ClientOption) (*UTCPClient, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	u := &UTCPClient{
		providers: make(map[string]ProviderConfig, len(cfg.Providers)),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(u)
	}

	for _, p := range cfg.Providers {
		u.providers[p.ID] = p
	}

	return u, nil
}

func NewUTCPClientFromFile(path string, opts ...ClientOption) (*UTCPClient, error) {
	cfg, err := LoadProvidersConfig(path)
	if err != nil {
		return nil, err
	}
	return NewUTCPClient(cfg, opts...)
}

func (c *UTCPClient) CallTool(ctx context.Context, providerID, toolName string, input any, output any) error {
	start := time.Now()
	observability.LogToolStart(ctx, providerID, toolName, input)
	var callErr error
	defer func() {
		duration := time.Since(start)
		observability.LogToolEnd(ctx, providerID, toolName, duration, callErr)
		observability.RecordToolCall(ctx, providerID, toolName, duration, callErr)
		if toolName == ToolMemoryRecall {
			observability.RecordRecall(ctx, providerID, toolName, callErr)
		}
		if toolName == ToolMemoryRetain {
			bank := extractBank(input)
			observability.LogRetention(ctx, bank, callErr)
			observability.RecordRetain(ctx, bank, callErr)
		}
	}()

	provider, ok := c.providers[providerID]
	if !ok {
		callErr = fmt.Errorf("utcp: provider not configured: %s", providerID)
		return callErr
	}

	switch strings.ToLower(provider.Transport) {
	case "local":
		if c.local == nil {
			callErr = fmt.Errorf("utcp: local transport requested but no LocalRegistry configured for provider %s", providerID)
			return callErr
		}
		if err := c.local.Call(ctx, providerID, toolName, input, output); err != nil {
			callErr = fmt.Errorf("utcp: local call %s/%s: %w", providerID, toolName, err)
			return callErr
		}
		return nil
	case "http":
		if err := c.callHTTP(ctx, provider.Endpoint, providerID, toolName, input, output); err != nil {
			callErr = err
			return callErr
		}
		return nil
	default:
		callErr = fmt.Errorf("utcp: unsupported transport %q for provider %s", provider.Transport, providerID)
		return callErr
	}
}

func extractBank(input any) string {
	switch typed := input.(type) {
	case contracts.MemoryRetainRequest:
		return typed.Bank
	case *contracts.MemoryRetainRequest:
		if typed != nil {
			return typed.Bank
		}
	case map[string]any:
		if value, ok := typed["bank"]; ok {
			if bank, ok := value.(string); ok {
				return bank
			}
		}
	}
	return ""
}

func (c *UTCPClient) callHTTP(ctx context.Context, endpoint string, providerID string, toolName string, input any, output any) error {
	type toolRequest struct {
		Tool  string `json:"tool"`
		Input any    `json:"input"`
	}

	body, err := json.Marshal(toolRequest{Tool: toolName, Input: input})
	if err != nil {
		return fmt.Errorf("utcp: marshal http request %s/%s: %w", providerID, toolName, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("utcp: build http request %s/%s: %w", providerID, toolName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("utcp: http call %s/%s: %w", providerID, toolName, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("utcp: read http response %s/%s: %w", providerID, toolName, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("utcp: http call %s/%s failed: %s", providerID, toolName, msg)
	}

	if output == nil {
		return nil
	}

	// Accept either:
	// 1) { "output": <json> }
	// 2) <json> (raw output)
	var wrapped struct {
		Output json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal(respBody, &wrapped); err == nil && len(wrapped.Output) > 0 {
		if err := json.Unmarshal(wrapped.Output, output); err != nil {
			return fmt.Errorf("utcp: decode wrapped output %s/%s: %w", providerID, toolName, err)
		}
		return nil
	}

	if err := json.Unmarshal(respBody, output); err != nil {
		return fmt.Errorf("utcp: decode output %s/%s: %w", providerID, toolName, err)
	}
	return nil
}
