// Package inference owns the small Jax model-transport boundary.
//
// Business logic selects a bounded request and receives advisory data through
// this package. Provider adapters do not receive tools, broker clients,
// portfolio writers, or execution authority.
package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const ContractVersion = "jax-inference/v1"

type Provider string

const (
	ProviderDeterministic Provider = "deterministic"
	ProviderOpenAI        Provider = "openai"
	ProviderOllama        Provider = "ollama"
)

var (
	ErrInvalidConfig  = errors.New("inference: invalid configuration")
	ErrInvalidRequest = errors.New("inference: invalid request")
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type Request struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
}

type Usage struct {
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
}

type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

type Response struct {
	Text      string
	ToolCalls []ToolCall
	Provider  Provider
	Model     string
	RequestID string
	Usage     Usage
}

type Client interface {
	Complete(context.Context, Request) (Response, error)
}

type Config struct {
	Provider Provider
	BaseURL  string
	APIKey   string
	Model    string
	HTTP     *http.Client
}

// ConfigFromEnv is the single configuration owner for operational planner
// and chat inference. JAX_MODEL_PROVIDER defaults to deterministic; paid or
// networked providers are never selected implicitly.
func ConfigFromEnv(lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	provider := Provider(strings.ToLower(strings.TrimSpace(value(lookup, "JAX_MODEL_PROVIDER"))))
	if provider == "" {
		provider = ProviderDeterministic
	}
	model := strings.TrimSpace(value(lookup, "JAX_MODEL_MODEL"))
	baseURL := strings.TrimSpace(value(lookup, "JAX_MODEL_BASE_URL"))
	apiKey := strings.TrimSpace(value(lookup, "JAX_MODEL_API_KEY"))
	switch provider {
	case ProviderDeterministic:
		if model == "" {
			model = "local-small"
		}
	case ProviderOpenAI:
		if model == "" {
			model = strings.TrimSpace(value(lookup, "OPENAI_MODEL"))
		}
		if baseURL == "" {
			baseURL = strings.TrimSpace(value(lookup, "OPENAI_BASE_URL"))
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		if apiKey == "" {
			apiKey = strings.TrimSpace(value(lookup, "OPENAI_API_KEY"))
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
	case ProviderOllama:
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		if model == "" {
			model = "qwen2.5:7b"
		}
	default:
		return Config{}, fmt.Errorf("%w: JAX_MODEL_PROVIDER must be deterministic, openai, or ollama", ErrInvalidConfig)
	}
	return Config{Provider: provider, BaseURL: baseURL, APIKey: apiKey, Model: model}, nil
}

func NewClient(config Config) (Client, error) {
	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(config)
	case ProviderOllama:
		return NewOllamaClient(config)
	case ProviderDeterministic:
		return nil, fmt.Errorf("%w: deterministic provider has no network client", ErrInvalidConfig)
	default:
		return nil, fmt.Errorf("%w: unsupported provider %q", ErrInvalidConfig, config.Provider)
	}
}

func validateHTTPConfig(config Config, requireKey bool) (*http.Client, error) {
	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("%w: model is required", ErrInvalidConfig)
	}
	if requireKey && strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("%w: API key is required for openai provider", ErrInvalidConfig)
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%w: base URL must be an absolute HTTP(S) URL", ErrInvalidConfig)
	}
	client := config.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return client, nil
}

func validateRequest(request Request) error {
	if strings.TrimSpace(request.Model) == "" || len(request.Messages) == 0 || request.MaxTokens <= 0 || request.MaxTokens > 8192 {
		return fmt.Errorf("%w: model, messages, and bounded max_tokens are required", ErrInvalidRequest)
	}
	for _, message := range request.Messages {
		if message.Role == "" || len(message.Content) > 1<<20 {
			return fmt.Errorf("%w: invalid message", ErrInvalidRequest)
		}
	}
	for _, tool := range request.Tools {
		if strings.TrimSpace(tool.Name) == "" || tool.Parameters == nil {
			return fmt.Errorf("%w: invalid tool definition", ErrInvalidRequest)
		}
	}
	return nil
}

type OpenAIClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewOpenAIClient(config Config) (*OpenAIClient, error) {
	client, err := validateHTTPConfig(config, true)
	if err != nil {
		return nil, err
	}
	return &OpenAIClient{baseURL: strings.TrimRight(config.BaseURL, "/"), apiKey: config.APIKey, model: config.Model, http: client}, nil
}

func (c *OpenAIClient) Complete(ctx context.Context, request Request) (Response, error) {
	if c == nil {
		return Response{}, fmt.Errorf("inference: openai client is nil")
	}
	if request.Model == "" {
		request.Model = c.model
	}
	if err := validateRequest(request); err != nil {
		return Response{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("inference openai marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("inference openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("inference openai transport: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Response{}, fmt.Errorf("inference openai read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("inference openai HTTP %d", resp.StatusCode)
	}
	decoded, err := decodeChatResponse(raw)
	if err != nil {
		return Response{}, fmt.Errorf("inference openai response: %w", err)
	}
	decoded.Provider = ProviderOpenAI
	return decoded, nil
}

type OllamaClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllamaClient(config Config) (*OllamaClient, error) {
	client, err := validateHTTPConfig(config, false)
	if err != nil {
		return nil, err
	}
	return &OllamaClient{baseURL: strings.TrimRight(config.BaseURL, "/"), model: config.Model, http: client}, nil
}

func (c *OllamaClient) Complete(ctx context.Context, request Request) (Response, error) {
	if c == nil {
		return Response{}, fmt.Errorf("inference: ollama client is nil")
	}
	if request.Model == "" {
		request.Model = c.model
	}
	if err := validateRequest(request); err != nil {
		return Response{}, err
	}
	body, err := json.Marshal(struct {
		Model    string           `json:"model"`
		Messages []Message        `json:"messages"`
		Tools    []ToolDefinition `json:"tools,omitempty"`
		Stream   bool             `json:"stream"`
		Think    bool             `json:"think"`
		Options  map[string]any   `json:"options,omitempty"`
	}{Model: request.Model, Messages: request.Messages, Tools: request.Tools, Stream: false, Think: false, Options: map[string]any{"temperature": request.Temperature}})
	if err != nil {
		return Response{}, fmt.Errorf("inference ollama marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("inference ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("inference ollama transport: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Response{}, fmt.Errorf("inference ollama read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("inference ollama HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Model   string `json:"model"`
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		PromptTokens     int    `json:"prompt_eval_count"`
		CompletionTokens int    `json:"eval_count"`
		Error            string `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Response{}, fmt.Errorf("inference ollama decode: %w", err)
	}
	if decoded.Error != "" {
		return Response{}, fmt.Errorf("inference ollama provider error")
	}
	response := Response{Provider: ProviderOllama, Model: decoded.Model, Text: decoded.Message.Content, Usage: Usage{InputTokens: decoded.PromptTokens, OutputTokens: decoded.CompletionTokens}}
	for _, call := range decoded.Message.ToolCalls {
		args, err := json.Marshal(call.Function.Arguments)
		if err != nil {
			return Response{}, fmt.Errorf("inference ollama tool arguments: %w", err)
		}
		response.ToolCalls = append(response.ToolCalls, ToolCall{Name: call.Function.Name, Arguments: args})
	}
	if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
		return Response{}, fmt.Errorf("inference ollama response has no content")
	}
	return response, nil
}

func decodeChatResponse(raw []byte) (Response, error) {
	var decoded struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content   json.RawMessage `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			Details          struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Response{}, err
	}
	if decoded.Error != nil {
		return Response{}, fmt.Errorf("provider error")
	}
	if len(decoded.Choices) == 0 {
		return Response{}, fmt.Errorf("response has no choices")
	}
	text, err := decodeContent(decoded.Choices[0].Message.Content)
	if err != nil {
		return Response{}, err
	}
	response := Response{Text: text, Model: decoded.Model, RequestID: decoded.ID, Usage: Usage{InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens, CachedInputTokens: decoded.Usage.Details.CachedTokens}}
	for _, call := range decoded.Choices[0].Message.ToolCalls {
		args := json.RawMessage(call.Function.Arguments)
		if !json.Valid(args) {
			return Response{}, fmt.Errorf("tool call arguments are not JSON")
		}
		response.ToolCalls = append(response.ToolCalls, ToolCall{Name: call.Function.Name, Arguments: args})
	}
	if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
		return Response{}, fmt.Errorf("response has no content")
	}
	return response, nil
}

func decodeContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", err
	}
	var out strings.Builder
	for _, part := range parts {
		if part.Text != "" {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(part.Text)
		}
	}
	return out.String(), nil
}

func value(lookup func(string) (string, bool), key string) string {
	value, _ := lookup(key)
	return value
}
