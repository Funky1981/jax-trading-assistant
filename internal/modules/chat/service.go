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
}

// NewService creates a chat Service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		store:  NewSessionStore(pool),
		router: NewToolRouter(pool),
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

// SendMessage records a user message and generates a deterministic assistant reply.
// For a real deployment, replace the reply generation with an LLM call.
// Tool calls embedded in the user message are executed read-only through ToolRouter.
func (s *Service) SendMessage(ctx context.Context, sessionID uuid.UUID, userContent string, toolCall *ToolCall) ([]*Message, error) {
	// Verify session exists.
	if _, err := s.store.GetSession(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("chat.Service.SendMessage: session not found: %w", err)
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

	// Persist a minimal assistant echo reply.
	// Replace this with an actual LLM/RAG call when the research runtime is wired.
	assistantReply, err := s.store.AppendMessage(ctx, &Message{
		SessionID: sessionID,
		Role:      RoleAssistant,
		Content:   s.buildReply(userContent, toolCall),
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

// buildReply produces a placeholder reply. In production this calls the LLM.
func (s *Service) buildReply(userContent string, call *ToolCall) string {
	if call != nil {
		return fmt.Sprintf("I looked up %q for you. Check the tool result above for the details.", call.Name)
	}
	if userContent == "" {
		return "How can I help you analyse the current trading situation?"
	}
	return "I'm Jax Assistant (advisory only). I can explain candidate trades, signals, strategy behaviour, and research runs. What would you like to know?"
}
