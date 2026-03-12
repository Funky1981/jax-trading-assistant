package chat

import (
	"context"
	"encoding/json"
	"fmt"

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
// Falls back to static advisory replies if the LLM is unavailable or returns an error.
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
		return fmt.Sprintf("I looked up %q for you. Check the tool result above for the details.", call.Name)
	}
	if userContent == "" {
		return "How can I help you analyse the current trading situation?"
	}
	return "I'm Jax Assistant (advisory only). I can explain candidate trades, signals, strategy behaviour, and research runs. What would you like to know?"
}
