package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/libs/chattools"
)

// LLMClient sends a message history and returns the assistant's reply plus any tool calls.
// The implementation is advisory only and must never execute or approve trades.
type LLMClient interface {
	Complete(ctx context.Context, msgs []LLMMessage) (string, []harness.ToolCall, error)
}

// LLMMessage carries a single turn for the LLM context window.
type LLMMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content string `json:"content"`
}

type openAIToolDefinition struct {
	Type     string             `json:"type"`
	Function openAIFunctionSpec `json:"function"`
}

type openAIFunctionSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  openAIFunctionArgument `json:"parameters"`
}

type openAIFunctionArgument struct {
	Type       string                     `json:"type"`
	Properties map[string]openAIFieldSpec `json:"properties"`
	Required   []string                   `json:"required,omitempty"`
}

type openAIFieldSpec struct {
	Type string `json:"type"`
}

type openAIToolCallResponse struct {
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// OpenAIChatClient calls any OpenAI-compatible chat completions endpoint.
// Set OPENAI_API_KEY and optionally OPENAI_BASE_URL (default: api.openai.com).
// Set OPENAI_MODEL to override the default model (default: gpt-4o-mini).
type OpenAIChatClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewOpenAIChatClientFromEnv creates a client from environment variables.
// Returns nil if OPENAI_API_KEY is not set.
func NewOpenAIChatClientFromEnv() *OpenAIChatClient {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil
	}
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = "https://api.openai.com"
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIChatClient{
		baseURL: base,
		apiKey:  key,
		model:   model,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Complete sends msgs to the chat completions endpoint and returns reply text plus any requested tool calls.
func (c *OpenAIChatClient) Complete(ctx context.Context, msgs []LLMMessage) (string, []harness.ToolCall, error) {
	if c == nil {
		return "", nil, fmt.Errorf("llm client not configured")
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model       string                 `json:"model"`
		Messages    []chatMsg              `json:"messages"`
		Tools       []openAIToolDefinition `json:"tools,omitempty"`
		ToolChoice  string                 `json:"tool_choice,omitempty"`
		MaxTokens   int                    `json:"max_tokens"`
		Temperature float64                `json:"temperature"`
	}
	type choice struct {
		Message struct {
			Content   json.RawMessage          `json:"content"`
			ToolCalls []openAIToolCallResponse `json:"tool_calls"`
		} `json:"message"`
	}
	type response struct {
		Choices []choice `json:"choices"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	chatMsgs := make([]chatMsg, 0, len(msgs))
	for _, m := range msgs {
		chatMsgs = append(chatMsgs, chatMsg(m))
	}

	body, err := json.Marshal(request{
		Model:       c.model,
		Messages:    chatMsgs,
		Tools:       openAIToolDefs(),
		ToolChoice:  "auto",
		MaxTokens:   800,
		Temperature: 0.3,
	})
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: read body: %w", err)
	}

	var result response
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: decode: %w", err)
	}
	if result.Error != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: empty choices")
	}

	content, err := decodeAssistantContent(result.Choices[0].Message.Content)
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: decode content: %w", err)
	}
	toolCalls, err := decodeToolCalls(result.Choices[0].Message.ToolCalls)
	if err != nil {
		return "", nil, fmt.Errorf("OpenAIChatClient.Complete: decode tool calls: %w", err)
	}
	return content, toolCalls, nil
}

func openAIToolDefs() []openAIToolDefinition {
	tools := chattools.DefaultTools()
	out := make([]openAIToolDefinition, 0, len(tools))
	for _, tool := range tools {
		def := openAIToolDefinition{
			Type: "function",
			Function: openAIFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters: openAIFunctionArgument{
					Type: "object",
					Properties: map[string]openAIFieldSpec{
						tool.ArgKey: {Type: openAIArgType(tool.ArgKey)},
					},
				},
			},
		}
		if !strings.Contains(strings.ToLower(tool.ArgLabel), "optional") {
			def.Function.Parameters.Required = []string{tool.ArgKey}
		}
		out = append(out, def)
	}
	return out
}

func openAIArgType(argKey string) string {
	if argKey == "limit" {
		return "integer"
	}
	return "string"
}

func decodeAssistantContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, part := range parts {
		if textPart, ok := part["text"].(string); ok {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(textPart)
		}
	}
	return sb.String(), nil
}

func decodeToolCalls(rawCalls []openAIToolCallResponse) ([]harness.ToolCall, error) {
	toolCalls := make([]harness.ToolCall, 0, len(rawCalls))
	for _, call := range rawCalls {
		args := json.RawMessage(`{}`)
		if strings.TrimSpace(call.Function.Arguments) != "" {
			args = json.RawMessage(call.Function.Arguments)
		}
		toolCalls = append(toolCalls, harness.ToolCall{
			Name: call.Function.Name,
			Args: args,
		})
	}
	return toolCalls, nil
}
