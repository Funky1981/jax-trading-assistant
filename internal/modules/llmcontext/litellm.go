package llmcontext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LiteLLMConfig struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type LiteLLMClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewLiteLLMClient(config LiteLLMConfig) LiteLLMClient {
	client := config.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return LiteLLMClient{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		apiKey:  config.APIKey,
		http:    client,
	}
}

func (c LiteLLMClient) Complete(ctx context.Context, pkg PromptPackage) (LLMResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		return LLMResult{}, fmt.Errorf("AI_GATEWAY_BASE_URL is required")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return LLMResult{}, fmt.Errorf("AI_GATEWAY_API_KEY is required")
	}
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type requestBody struct {
		Model     string        `json:"model"`
		Messages  []chatMessage `json:"messages"`
		MaxTokens int           `json:"max_tokens"`
	}
	type responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	body, err := json.Marshal(requestBody{
		Model: pkg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: pkg.CacheablePrefix},
			{Role: "user", Content: strings.Join(nonEmpty([]string{pkg.RetrievedMemory, pkg.DynamicContext, pkg.ResponseSchema}), "\n\n")},
		},
		MaxTokens: pkg.EstimatedOutputTokens,
	})
	if err != nil {
		return LLMResult{}, fmt.Errorf("litellm request marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return LLMResult{}, fmt.Errorf("litellm request create: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return LLMResult{}, fmt.Errorf("litellm request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return LLMResult{}, fmt.Errorf("litellm response read: %w", err)
	}
	var decoded responseBody
	if err := json.Unmarshal(respBytes, &decoded); err != nil {
		return LLMResult{}, fmt.Errorf("litellm response decode: %w", err)
	}
	if decoded.Error != nil {
		return LLMResult{}, fmt.Errorf("litellm API error: %s", decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 {
		return LLMResult{}, fmt.Errorf("litellm response has no choices")
	}
	return LLMResult{
		CorrelationID: pkg.CorrelationID,
		Text:          decoded.Choices[0].Message.Content,
		InputTokens:   decoded.Usage.PromptTokens,
		OutputTokens:  decoded.Usage.CompletionTokens,
	}, nil
}
