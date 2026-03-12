// Package chat provides the assistant session and message persistence layer.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Role constants for chat messages.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Session is a single conversation thread.
type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    *string   `json:"userId,omitempty"`
	Title     *string   `json:"title,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Message is one turn within a session.
type Message struct {
	ID         uuid.UUID        `json:"id"`
	SessionID  uuid.UUID        `json:"sessionId"`
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolName   *string          `json:"toolName,omitempty"`
	ToolArgs   *json.RawMessage `json:"toolArgs,omitempty"`
	ToolResult *json.RawMessage `json:"toolResult,omitempty"`
	CreatedAt  time.Time        `json:"createdAt"`
}

// SessionStore handles persistence for chat sessions and messages.
type SessionStore struct {
	pool *pgxpool.Pool
}

// NewSessionStore creates a SessionStore.
func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

// CreateSession creates and returns a new chat session.
func (s *SessionStore) CreateSession(ctx context.Context, userID *string, title *string) (*Session, error) {
	sess := &Session{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO chat_sessions (id, user_id, title, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		sess.ID, sess.UserID, sess.Title, sess.CreatedAt, sess.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("chat.SessionStore.CreateSession: %w", err)
	}
	return sess, nil
}

// GetSession returns a session by ID.
func (s *SessionStore) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM chat_sessions WHERE id = $1`, id)
	var sess Session
	if err := row.Scan(&sess.ID, &sess.UserID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
		return nil, fmt.Errorf("chat.SessionStore.GetSession: %w", err)
	}
	return &sess, nil
}

// ListSessions returns recent sessions for a user (or all if userID is empty).
func (s *SessionStore) ListSessions(ctx context.Context, userID string, limit int) ([]*Session, error) {
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}
	var err error
	if userID != "" {
		rows, err = s.pool.Query(ctx,
			`SELECT id, user_id, title, created_at, updated_at FROM chat_sessions
			  WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT id, user_id, title, created_at, updated_at FROM chat_sessions
			  ORDER BY updated_at DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("chat.SessionStore.ListSessions: %w", err)
	}
	defer rows.Close()
	var out []*Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &sess)
	}
	return out, rows.Err()
}

// AppendMessage adds a message to a session and bumps the session's updated_at.
func (s *SessionStore) AppendMessage(ctx context.Context, msg *Message) (*Message, error) {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.CreatedAt = time.Now().UTC()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO chat_messages (id, session_id, role, content, tool_name, tool_args, tool_result, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content,
		msg.ToolName, msg.ToolArgs, msg.ToolResult, msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("chat.SessionStore.AppendMessage: %w", err)
	}
	_, _ = s.pool.Exec(ctx,
		`UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1`, msg.SessionID)
	return msg, nil
}

// GetHistory returns all messages for a session in chronological order.
func (s *SessionStore) GetHistory(ctx context.Context, sessionID uuid.UUID) ([]*Message, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, session_id, role, content, tool_name, tool_args, tool_result, created_at
		   FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("chat.SessionStore.GetHistory: %w", err)
	}
	defer rows.Close()
	var out []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.Content,
			&m.ToolName, &m.ToolArgs, &m.ToolResult, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}
