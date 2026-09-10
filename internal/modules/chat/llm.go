package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/internal/modules/inference"
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

// OpenAIChatClient is retained as the compatibility name for the chat
// adapter. Its transport is the Jax-owned inference client and may be backed
// by OpenAI or local Ollama according to explicit configuration.
type OpenAIChatClient struct {
	baseURL string
	apiKey  string
	model   string
	client  inference.Client
}

// NewOpenAIChatClientFromEnv creates an explicitly selected OpenAI chat
// client. It returns nil unless JAX_MODEL_PROVIDER=openai and the existing
// OpenAI configuration is valid. The name remains for source compatibility.
func NewOpenAIChatClientFromEnv() *OpenAIChatClient {
	config, err := inference.ConfigFromEnv(os.LookupEnv)
	if err != nil || config.Provider != inference.ProviderOpenAI {
		return nil
	}
	client, err := inference.NewClient(config)
	if err != nil {
		return nil
	}
	return &OpenAIChatClient{baseURL: config.BaseURL, apiKey: config.APIKey, model: config.Model, client: client}
}

// NewChatClientFromEnv selects the shared Jax model transport for chat. The
// deterministic default returns nil so the chat service uses its safe local
// advisory fallback rather than silently making a paid request.
func NewChatClientFromEnv() LLMClient {
	config, err := inference.ConfigFromEnv(os.LookupEnv)
	if err != nil || config.Provider == inference.ProviderDeterministic {
		return nil
	}
	client, err := inference.NewClient(config)
	if err != nil {
		return nil
	}
	return &OpenAIChatClient{baseURL: config.BaseURL, apiKey: config.APIKey, model: config.Model, client: client}
}

// Complete sends a bounded request through the shared Jax inference client
// and returns advisory reply text plus tool calls for the separate Jax
// harness to validate and execute under its own policy.
func (c *OpenAIChatClient) Complete(ctx context.Context, msgs []LLMMessage) (string, []harness.ToolCall, error) {
	if c == nil || c.client == nil {
		return "", nil, fmt.Errorf("llm client not configured")
	}
	chatMsgs := make([]inference.Message, 0, len(msgs))
	for _, m := range msgs {
		chatMsgs = append(chatMsgs, inference.Message{Role: m.Role, Content: m.Content})
	}
	defs := openAIToolDefs()
	tools := make([]inference.ToolDefinition, 0, len(defs))
	for _, def := range defs {
		properties := make(map[string]any, len(def.Function.Parameters.Properties))
		for key, field := range def.Function.Parameters.Properties {
			properties[key] = map[string]any{"type": field.Type}
		}
		params := map[string]any{"type": def.Function.Parameters.Type, "properties": properties}
		if len(def.Function.Parameters.Required) > 0 {
			params["required"] = def.Function.Parameters.Required
		}
		tools = append(tools, inference.ToolDefinition{Name: def.Function.Name, Description: def.Function.Description, Parameters: params})
	}
	response, err := c.client.Complete(ctx, inference.Request{Model: c.model, Messages: chatMsgs, Tools: tools, ToolChoice: "auto", MaxTokens: 800, Temperature: 0.3})
	if err != nil {
		return "", nil, fmt.Errorf("chat model provider: %w", err)
	}
	calls := make([]harness.ToolCall, 0, len(response.ToolCalls))
	for _, call := range response.ToolCalls {
		calls = append(calls, harness.ToolCall{Name: call.Name, Args: call.Arguments})
	}
	return response.Text, calls, nil
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
