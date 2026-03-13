package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service orchestrates chat sessions, message persistence, and tool calls.
// The assistant is ADVISORY ONLY. It must never directly execute or approve trades.
type Service struct {
	store  *SessionStore
	router *ToolRouter
	llm    LLMClient // nil → fall back to static placeholder replies
}

// NewService creates a chat Service.
// llm may be nil; when nil, the assistant falls back to static advisory replies.
func NewService(pool *pgxpool.Pool, llm LLMClient) *Service {
	return &Service{
		store:  NewSessionStore(pool),
		router: NewToolRouter(pool),
		llm:    llm,
	}
}

// StartSession creates a new chat session, optionally named.
func (s *Service) StartSession(ctx context.Context, userID, title string) (*Session, error) {
	var uid, ttl *string
	if userID != "" {
		uid = &userID
	}
	if title != "" {
		ttl = &title
	}
	return s.store.CreateSession(ctx, uid, ttl)
}

// GetSession returns a session by ID.
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return s.store.GetSession(ctx, id)
}

// ListSessions returns recent sessions for a user.
func (s *Service) ListSessions(ctx context.Context, userID string, limit int) ([]*Session, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.ListSessions(ctx, userID, limit)
}

// GetHistory returns message history for a session.
func (s *Service) GetHistory(ctx context.Context, sessionID uuid.UUID) ([]*Message, error) {
	return s.store.GetHistory(ctx, sessionID)
}

// SendMessage records a user message and generates an assistant reply.
// When an LLMClient is wired, the full session history is forwarded for context.
// Tool calls are executed read-only through ToolRouter before the LLM is called.
func (s *Service) SendMessage(ctx context.Context, sessionID uuid.UUID, userContent string, toolCall *ToolCall) ([]*Message, error) {
	// Load history before the new message so we can give the LLM full context.
	history, err := s.store.GetHistory(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("chat.Service.SendMessage: load history: %w", err)
	}

	var saved []*Message

	// Persist user message.
	userMsg, err := s.store.AppendMessage(ctx, &Message{
		SessionID: sessionID,
		Role:      RoleUser,
		Content:   userContent,
	})
	if err != nil {
		return nil, err
	}
	saved = append(saved, userMsg)

	// If a tool call was supplied, execute it and persist the result.
	if toolCall != nil {
		result, err := s.router.Dispatch(ctx, *toolCall)
		if err != nil && err != ErrUnknownTool {
			return nil, fmt.Errorf("chat.Service.SendMessage: tool dispatch: %w", err)
		}
		if err == ErrUnknownTool {
			result = errResult("tool not available: " + toolCall.Name)
		}
		toolMsg, err := s.persistToolResult(ctx, sessionID, toolCall, result)
		if err != nil {
			return nil, err
		}
		saved = append(saved, toolMsg)
	}

	// Generate the assistant reply — via LLM when available, otherwise static.
	replyText := s.buildReply(ctx, userContent, toolCall, history)
	assistantReply, err := s.store.AppendMessage(ctx, &Message{
		SessionID: sessionID,
		Role:      RoleAssistant,
		Content:   replyText,
	})
	if err != nil {
		return nil, err
	}
	saved = append(saved, assistantReply)

	return saved, nil
}

func (s *Service) persistToolResult(ctx context.Context, sessionID uuid.UUID, call *ToolCall, result *ToolResult) (*Message, error) {
	name := call.Name
	var resultRaw *json.RawMessage
	if b, err := json.Marshal(result); err == nil {
		raw := json.RawMessage(b)
		resultRaw = &raw
	}
	return s.store.AppendMessage(ctx, &Message{
		SessionID:  sessionID,
		Role:       RoleTool,
		Content:    fmt.Sprintf("tool: %s", name),
		ToolName:   &name,
		ToolArgs:   &call.Args,
		ToolResult: resultRaw,
	})
}

// buildReply generates the assistant's reply text.
// When s.llm is set, the full session history is forwarded for context.
// Falls back to contextual keyword-based replies if the LLM is unavailable or returns an error.
func (s *Service) buildReply(ctx context.Context, userContent string, call *ToolCall, history []*Message) string {
	if s.llm != nil {
		msgs := make([]LLMMessage, 0, len(history)+1)
		for _, m := range history {
			// Skip internal tool result messages — they add noise without benefit.
			if m.Role == RoleTool {
				continue
			}
			msgs = append(msgs, LLMMessage{Role: string(m.Role), Content: m.Content})
		}
		msgs = append(msgs, LLMMessage{Role: "user", Content: userContent})
		if reply, err := s.llm.Complete(ctx, msgs); err == nil {
			return reply
		}
		// Fall through to static reply on LLM error.
	}
	if call != nil {
		switch call.Name {
		case "get_candidate_trade":
			return "Here's the candidate trade I fetched. Check the tool result above for the full details including symbol, direction, confidence score, and any blocker reasons."
		case "explain_trade_blockers":
			return "Here's the blocker analysis. The tool result above lists every guard-rail or risk constraint that prevented this candidate from being promoted to an order."
		case "get_signal":
			return "Here's the signal I retrieved. The tool result above shows the signal strength, the strategy that generated it, and the timestamp."
		case "get_strategy":
			return "Here's the strategy definition. The tool result above describes the strategy parameters and its last known state."
		case "get_strategy_instance":
			return "Here's the strategy instance. The tool result above shows the running configuration, schedule, and any active positions."
		case "get_trade":
			return "Here's the executed trade record. The tool result above includes fill price, quantity, and execution status."
		case "get_orchestration_run":
			return "Here's the orchestration run. The tool result above shows the run outcome, signals produced, and any errors encountered."
		case "search_research_runs":
			return "Here are the recent research runs matching your query. The tool result above shows each run's outcome, signals produced, and timestamps."
		default:
			return fmt.Sprintf("I fetched the result for %q. Check the tool result above for the details.", call.Name)
		}
	}
	if userContent == "" {
		return "How can I help you analyse the current trading situation?"
	}
	return staticKeywordReply(userContent)
}

// staticKeywordReply provides contextual guidance when no LLM is configured.
// It matches keywords in the user's message to return a relevant, topic-specific reply.
func staticKeywordReply(msg string) string {
	lower := strings.ToLower(msg)
	has := func(keywords ...string) bool {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}
	switch {
	case has("hello", "hi ", "hey ", "howdy", "greetings"):
		return "Hello! I'm Jax Assistant (advisory only). I can help you understand candidate trades, signals, strategy behaviour, and research runs. Use the Tool Picker (⚙) to query live data, or ask me about a specific topic."
	case has("signal", "high-confidence", "high confidence"):
		return "To inspect a specific signal, use the Tool Picker and select get_signal. For recent run activity that generated signals, try search_research_runs. The Signals page in the dashboard shows the full live list."
	case has("candidate", "waiting", "approval", "approve", "pending"):
		return "To inspect a specific candidate trade, use the Tool Picker and select get_candidate_trade. To find out why a candidate was blocked, use explain_trade_blockers. The Approval Queue page shows all pending candidates in real time."
	case has("block", "blocker", "rejected", "prevent", "why was", "why wasn"):
		return "Use the Tool Picker and select explain_trade_blockers to see exactly which guard-rails or risk constraints prevented a candidate from being promoted. You'll need the candidate ID from the Approval Queue."
	case has("strateg"):
		return "To inspect a strategy definition, use get_strategy in the Tool Picker. For a live running instance, use get_strategy_instance. The Strategy Instances page shows all currently active instances and their schedules."
	case has("research", "orchestration"):
		return "Use search_research_runs in the Tool Picker to browse recent orchestration and research runs. You can filter by symbol to narrow the results."
	case has("market", "condition", "price", "quote", "outlook"):
		return "Market data is ingested and evaluated continuously by the strategy engine. The latest assessed opportunities appear in the Approval Queue. For raw price data, check the Market Data panel in the dashboard."
	case has("trade", "executed", "filled", "order", "position"):
		return "To look up an executed trade, use get_trade in the Tool Picker with the trade ID. Open positions and fill history are also visible on the Trades page."
	case has("help", "what can", "what do", "capabilities", "tools", "available"):
		return "I can look up candidate trades, signals, executed trades, strategy definitions and instances, orchestration runs, and trade blocker explanations — use the Tool Picker (⚙) to run any of these. I can also answer questions about how the trading system works. Note: without an OpenAI API key I give keyword-based replies rather than full conversational answers."
	default:
		return "I'm Jax Assistant (advisory only). I can explain candidate trades, signals, strategy behaviour, and research runs. Use the Tool Picker (⚙) above to query live data, or ask me about a specific topic like signals, candidates, strategies, or research runs."
	}
}
