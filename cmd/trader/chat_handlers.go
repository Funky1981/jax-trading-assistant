package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	chatmod "jax-trading-assistant/internal/modules/chat"
)

// registerChatRoutes wires the assistant chat endpoints.
// The assistant is advisory only and cannot execute or approve trades.
func registerChatRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	svc := chatmod.NewService(pool, chatmod.NewOpenAIChatClientFromEnv())

	// Session management
	mux.HandleFunc("/api/v1/chat/sessions", protect(chatSessionsHandler(svc)))
	mux.HandleFunc("/api/v1/chat/sessions/", protect(chatSessionDetailHandler(svc)))

	// Message endpoint (send + history)
	mux.HandleFunc("/api/v1/chat", protect(chatHandler(svc)))

	// Tool catalogue
	mux.HandleFunc("/api/v1/chat/tools", protect(chatToolsHandler()))
}

// POST /api/v1/chat — send a message in a session (or create one).
// GET  /api/v1/chat?session={id} — get history for a session.
func chatHandler(svc *chatmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			sessionIDStr := r.URL.Query().Get("session")
			if sessionIDStr == "" {
				http.Error(w, "session query param required", http.StatusBadRequest)
				return
			}
			sid, err := uuid.Parse(sessionIDStr)
			if err != nil {
				http.Error(w, "invalid session id", http.StatusBadRequest)
				return
			}
			history, err := svc.GetHistory(r.Context(), sid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if history == nil {
				history = []*chatmod.Message{}
			}
			jsonOK(w, history)

		case http.MethodPost:
			var body struct {
				SessionID string            `json:"sessionId"`
				Content   string            `json:"content"`
				ToolCall  *chatmod.ToolCall `json:"toolCall,omitempty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.Content) == "" {
				http.Error(w, "content is required", http.StatusBadRequest)
				return
			}

			var sessionID uuid.UUID
			if body.SessionID != "" {
				sid, err := uuid.Parse(body.SessionID)
				if err != nil {
					http.Error(w, "invalid sessionId", http.StatusBadRequest)
					return
				}
				sessionID = sid
			} else {
				// Auto-create a session for convenience.
				sess, err := svc.StartSession(r.Context(), actorFromRequest(r), "")
				if err != nil {
					http.Error(w, fmt.Sprintf("create session: %v", err), http.StatusInternalServerError)
					return
				}
				sessionID = sess.ID
			}

			msgs, err := svc.SendMessage(r.Context(), sessionID, body.Content, body.ToolCall)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, map[string]any{
				"sessionId": sessionID,
				"messages":  msgs,
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// GET  /api/v1/chat/sessions  — list sessions.
// POST /api/v1/chat/sessions  — create a session.
func chatSessionsHandler(svc *chatmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userID := actorFromRequest(r)
			sessions, err := svc.ListSessions(r.Context(), userID, 20)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if sessions == nil {
				sessions = []*chatmod.Session{}
			}
			jsonOK(w, sessions)

		case http.MethodPost:
			var body struct {
				Title string `json:"title"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			sess, err := svc.StartSession(r.Context(), actorFromRequest(r), body.Title)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, sess)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// GET /api/v1/chat/sessions/{id} — get a single session + history.
func chatSessionDetailHandler(svc *chatmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rawID := strings.TrimPrefix(r.URL.Path, "/api/v1/chat/sessions/")
		sid, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}
		sess, err := svc.GetSession(r.Context(), sid)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				http.Error(w, "session not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		history, err := svc.GetHistory(r.Context(), sid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if history == nil {
			history = []*chatmod.Message{}
		}
		jsonOK(w, map[string]any{
			"session":  sess,
			"messages": history,
		})
	}
}

// GET /api/v1/chat/tools — returns the list of available assistant tools.
func chatToolsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonOK(w, map[string]any{
			"tools":  chatmod.AvailableTools(),
			"notice": "Assistant is advisory only. It cannot execute trades or approve decisions on behalf of the user.",
		})
	}
}
