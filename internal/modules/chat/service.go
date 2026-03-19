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
// The assistant is advisory only and must never directly execute or approve trades.
type Service struct {
	store  *SessionStore
	router *ToolRouter
	llm    LLMClient
	pool   *pgxpool.Pool
}

// NewService creates a chat Service.
// llm may be nil; when nil, the assistant falls back to static advisory replies.
func NewService(pool *pgxpool.Pool, llm LLMClient) *Service {
	return &Service{
		store:  NewSessionStore(pool),
		router: NewToolRouter(pool),
		llm:    llm,
		pool:   pool,
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
	history, err := s.store.GetHistory(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("chat.Service.SendMessage: load history: %w", err)
	}

	var saved []*Message

	userMsg, err := s.store.AppendMessage(ctx, &Message{
		SessionID: sessionID,
		Role:      RoleUser,
		Content:   userContent,
	})
	if err != nil {
		return nil, err
	}
	saved = append(saved, userMsg)

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
	s.logAssistantDecision(ctx, sessionID, userContent, toolCall, replyText)

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
			if m.Role == RoleTool {
				continue
			}
			msgs = append(msgs, LLMMessage{Role: string(m.Role), Content: m.Content})
		}
		msgs = append(msgs, LLMMessage{Role: "user", Content: userContent})
		if reply, err := s.llm.Complete(ctx, msgs); err == nil {
			return reply
		}
	}
	if call != nil {
		switch call.Name {
		case "get_candidate_trade":
			return "Here's the candidate trade I fetched. Check the tool result above for the full details including symbol, direction, confidence score, provenance, and execution linkage."
		case "explain_trade_blockers":
			return "Here's the blocker analysis. The tool result above lists the guardrails or policy reasons that prevented this candidate from being promoted."
		case "get_signal":
			return "Here's the signal I retrieved. The tool result above shows the signal strength, the strategy that generated it, and the timestamp."
		case "get_strategy":
			return "Here's the strategy definition. The tool result above describes the strategy parameters and its last known state."
		case "get_strategy_instance":
			return "Here's the strategy instance. The tool result above shows the running configuration, schedule, and active trading window."
		case "get_trade":
			return "Here's the executed trade record. The tool result above includes the trade status, quantity, and fill linkage."
		case "get_orchestration_run":
			return "Here's the orchestration run. The tool result above shows the run outcome, signals produced, and any errors encountered."
		case "search_research_runs":
			return "Here are the recent research runs matching your query. The tool result above shows each run's outcome and timestamps."
		case "list_pending_approvals":
			return "Here is the current approval queue. The tool result above shows the candidates still waiting for a human decision."
		case "list_recent_blocked_candidates":
			return "Here are the most recent blocked candidates. The tool result above shows the blocker codes and human-readable reasons."
		case "search_candidates":
			return "Here are the matching candidates. The tool result above shows recent candidates filtered by symbol or status."
		case "query_knowledge":
			return "Here are the matching knowledge snippets. The tool result above shows local markdown excerpts related to your query."
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
		return "Hello. I'm Jax Assistant, advisory only. I can explain candidate trades, signals, strategy behaviour, research runs, and blocked-candidate reasons."
	case has("signal", "high-confidence", "high confidence"):
		return "To inspect a specific signal, use get_signal. For recent run activity that generated signals, try search_research_runs."
	case has("candidate", "waiting", "approval", "approve", "pending"):
		return "Use list_pending_approvals to see the live approval queue, or get_candidate_trade to inspect one candidate in detail. To find out why a candidate was blocked, use explain_trade_blockers or list_recent_blocked_candidates."
	case has("block", "blocker", "rejected", "prevent", "why was", "why wasn"):
		return "Use explain_trade_blockers for one candidate, or list_recent_blocked_candidates to review the latest blocked setups and their reason codes."
	case has("strateg"):
		return "To inspect a strategy definition, use get_strategy. For a live running instance, use get_strategy_instance."
	case has("research", "orchestration"):
		return "Use search_research_runs to browse recent orchestration and research runs. You can filter by symbol to narrow the results."
	case has("knowledge", "docs", "documentation", "playbook", "runbook"):
		return "Use query_knowledge to search the local markdown knowledge base when it is configured."
	case has("trade", "executed", "filled", "order", "position"):
		return "To look up an executed trade, use get_trade with the trade ID. Approvals still have to be made through the Approvals page."
	case has("help", "what can", "what do", "capabilities", "tools", "available"):
		return "I can look up candidates, signals, executed trades, strategy definitions and instances, orchestration runs, blocked candidates, pending approvals, and local knowledge snippets. Approvals still have to be made through the Approvals page."
	default:
		return "I'm Jax Assistant, advisory only. Use the Tool Picker to query candidates, signals, trades, runs, approval queues, blocked candidates, or local knowledge."
	}
}

func (s *Service) logAssistantDecision(ctx context.Context, sessionID uuid.UUID, userContent string, toolCall *ToolCall, replyText string) {
	if s.pool == nil {
		return
	}
	provider := "tool-only"
	model := "static-fallback"
	if s.llm != nil {
		provider = "openai-compatible"
		model = "configured"
	}
	ruleTrace := map[string]any{
		"sessionId": sessionID.String(),
	}
	if toolCall != nil {
		ruleTrace["toolName"] = toolCall.Name
		ruleTrace["toolArgs"] = json.RawMessage(toolCall.Args)
		var args map[string]any
		if err := json.Unmarshal(toolCall.Args, &args); err == nil {
			for _, key := range []string{"candidateId", "tradeId", "runId", "signalId", "instanceId"} {
				if value, ok := args[key]; ok {
					ruleTrace[key] = value
				}
			}
		}
	}
	promptJSON, _ := json.Marshal(map[string]any{"userMessage": userContent})
	responseJSON, _ := json.Marshal(map[string]any{"assistantReply": replyText})
	ruleTraceJSON, _ := json.Marshal(ruleTrace)

	decisionID := uuid.New()
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO ai_decisions (
			id, run_id, flow_id, role, provider, model, prompt, response,
			schema_valid, decision, reasoning, rule_trace, created_at
		) VALUES (
			$1, NULL, $2, 'assistant', $3, $4, $5::jsonb, $6::jsonb,
			TRUE, $7, $8, $9::jsonb, NOW()
		)
	`, decisionID, sessionID.String(), provider, model, string(promptJSON), string(responseJSON), replyText, "assistant advisory reply", string(ruleTraceJSON)); err != nil {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO ai_decision_acceptance (id, decision_id, accepted, accepted_by, reason, rule_trace, created_at)
		VALUES ($1, $2, TRUE, 'assistant_service', 'assistant reply emitted', $3::jsonb, NOW())
	`, uuid.New(), decisionID, string(ruleTraceJSON))
}
