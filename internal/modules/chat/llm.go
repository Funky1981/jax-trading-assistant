package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMClient sends a message history and returns the assistant's reply.
// The implementation is advisory only and must never execute or approve trades.
type LLMClient interface {
	Complete(ctx context.Context, msgs []LLMMessage) (string, error)
}

// LLMMessage carries a single turn for the LLM context window.
type LLMMessage struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant" | "tool"
	Content string `json:"content"`
}

// systemPrompt is sent as the first message to every LLM context.
// It enforces the advisory-only boundary.
const systemPrompt = `You are Jax Assistant — a read-only trading research assistant embedded in the Jax Trading platform.

You have access to the following read-only tools:
- get_candidate_trade: look up a candidate trade by ID
- get_signal: look up a strategy signal by ID
- get_trade: look up an executed trade by ID
- get_strategy: look up a strategy definition by ID
- get_strategy_instance: look up a strategy instance by ID  
- get_orchestration_run: look up an orchestration/research run by ID
- search_research_runs: search recent research runs
- explain_trade_blockers: explain why a candidate trade was blocked

IMPORTANT CONSTRAINTS — you must never violate these:
1. You CANNOT execute trades, place orders, or approve candidates.
2. You CANNOT call any tool not listed above.
3. You CANNOT reference real-time external market data or make specific price predictions.
4. All your responses are ADVISORY ONLY. The human trader makes all final decisions.
5. Always remind the user that approvals must be done via the Approvals page.

Your role is to explain, analyse, and answer questions about the data in the Jax system.
Be concise and accurate. If you don't know something, say so rather than guessing.`

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

// Complete sends msgs to the chat completions endpoint and returns the reply text.
func (c *OpenAIChatClient) Complete(ctx context.Context, msgs []LLMMessage) (string, error) {
	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model       string    `json:"model"`
		Messages    []chatMsg `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
	}
	type choice struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	type response struct {
		Choices []choice `json:"choices"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	// Build the messages list with system prompt prepended.
	chatMsgs := make([]chatMsg, 0, len(msgs)+1)
	chatMsgs = append(chatMsgs, chatMsg{Role: "system", Content: systemPrompt})
	for _, m := range msgs {
		chatMsgs = append(chatMsgs, chatMsg{Role: m.Role, Content: m.Content})
	}

	body, err := json.Marshal(request{
		Model:       c.model,
		Messages:    chatMsgs,
		MaxTokens:   800,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: read body: %w", err)
	}

	var result response
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("OpenAIChatClient.Complete: API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("OpenAIChatClient.Complete: empty choices")
	}
	return result.Choices[0].Message.Content, nil
}
