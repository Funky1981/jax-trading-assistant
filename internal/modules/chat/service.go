package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/internal/modules/harness"
)

// Service orchestrates chat sessions, message persistence, and tool calls.
// The assistant is advisory only and must never directly execute or approve trades.
type Service struct {
	store   *SessionStore
	router  *ToolRouter
	llm     LLMClient
	pool    *pgxpool.Pool
	reg     *harness.Registry
	policy  harness.Policy
	prompts *harness.PromptBuilder
	harness *harness.Service
	traces  *harness.PostgresTraceSink
	runtime runtimeConfig
	limiter *sessionRateLimiter
}

// NewService creates a chat Service.
// llm may be nil; when nil, the assistant falls back to static advisory replies.
func NewService(pool *pgxpool.Pool, llm LLMClient) *Service {
	runtimeCfg := loadRuntimeConfig()
	reg := harness.NewRegistry()
	if err := harness.RegisterDefaultTools(reg, pool); err != nil {
		panic(fmt.Sprintf("chat.NewService: register default tools: %v", err))
	}

	policy := harness.DefaultPolicy(runtimeCfg.Mode)
	prompts := harness.NewPromptBuilder()
	traceSink := harness.NewPostgresTraceSink(pool)

	var harnessSvc *harness.Service
	if runtimeCfg.HarnessEnabled {
		harnessSvc = harness.NewService(policy, reg, prompts, harness.NewValidator(), traceSink, newHarnessModelAdapter(llm))
	}

	return &Service{
		store:   NewSessionStore(pool),
		router:  NewToolRouter(pool),
		llm:     llm,
		pool:    pool,
		reg:     reg,
		policy:  policy,
		prompts: prompts,
		harness: harnessSvc,
		traces:  traceSink,
		runtime: runtimeCfg,
		limiter: newSessionRateLimiter(runtimeCfg.SessionRateLimitPerMinute),
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

func (s *Service) GetTrace(ctx context.Context, traceID string) (*harness.Trace, error) {
	if s.traces == nil {
		return nil, fmt.Errorf("trace storage unavailable")
	}
	return s.traces.GetTrace(ctx, traceID)
}

func (s *Service) RuntimeInfo() RuntimeInfo {
	return s.runtime.info()
}

func (s *Service) AvailableTools() []ToolDescriptor {
	return AvailableTools(s.policy)
}

// SendMessage records a user message and generates an assistant reply.
// When an LLMClient is wired, the harness loop is used first. If harness execution fails,
// the service falls back to the previous direct tool-plus-reply flow.
func (s *Service) SendMessage(ctx context.Context, sessionID uuid.UUID, userContent string, toolCall *ToolCall) ([]*Message, error) {
	if err := s.allowSession(sessionID); err != nil {
		return nil, err
	}
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

	if s.runtime.ShadowMode && s.harness != nil && (s.llm != nil || toolCall != nil) {
		s.runHarnessShadow(sessionID, userContent, toolCall, history)
	}

	if !s.runtime.ShadowMode && s.harness != nil && (s.llm != nil || toolCall != nil) {
		if assistantMsg, toolMsgs, err := s.sendWithHarness(ctx, sessionID, userContent, toolCall, history); err == nil {
			saved = append(saved, toolMsgs...)
			saved = append(saved, assistantMsg)
			s.logAssistantDecision(ctx, sessionID, userContent, toolCall, assistantMsg.Content)
			return saved, nil
		}
	}

	toolMsg, fallbackToolCall, err := s.persistFallbackTool(ctx, sessionID, toolCall)
	if err != nil {
		return nil, err
	}
	if toolMsg != nil {
		saved = append(saved, toolMsg)
	}

	replyText := s.buildFallbackReply(ctx, userContent, fallbackToolCall, history)
	assistantReply, err := s.store.AppendMessage(ctx, &Message{
		SessionID: sessionID,
		Role:      RoleAssistant,
		Content:   replyText,
	})
	if err != nil {
		return nil, err
	}
	saved = append(saved, assistantReply)
	s.logAssistantDecision(ctx, sessionID, userContent, fallbackToolCall, replyText)

	return saved, nil
}

func (s *Service) sendWithHarness(ctx context.Context, sessionID uuid.UUID, userContent string, toolCall *ToolCall, history []*Message) (*Message, []*Message, error) {
	answer, bundle, trace, err := s.harness.AnswerWithEvidence(ctx, sessionID.String(), userContent, s.toHarnessHistory(history), toHarnessToolCall(toolCall))
	if err != nil {
		return nil, nil, err
	}

	toolMsgs := make([]*Message, 0, len(trace.ToolRuns))
	for _, run := range trace.ToolRuns {
		msg, err := s.persistHarnessToolRun(ctx, sessionID, run)
		if err != nil {
			return nil, nil, err
		}
		toolMsgs = append(toolMsgs, msg)
	}

	var bundleRaw *json.RawMessage
	if b, err := json.Marshal(bundle); err == nil {
		raw := json.RawMessage(b)
		bundleRaw = &raw
	}
	assistantMsg, err := s.store.AppendMessage(ctx, &Message{
		SessionID:      sessionID,
		Role:           RoleAssistant,
		Content:        answer,
		TraceID:        &trace.TraceID,
		EvidenceBundle: bundleRaw,
	})
	if err != nil {
		return nil, nil, err
	}

	return assistantMsg, toolMsgs, nil
}

func (s *Service) persistHarnessToolRun(ctx context.Context, sessionID uuid.UUID, run harness.ToolRun) (*Message, error) {
	name := run.Call.Name
	callArgs := run.Call.Args
	result := &ToolResult{Ok: run.Error == "", Data: run.Result, Error: run.Error}
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
		ToolArgs:   &callArgs,
		ToolResult: resultRaw,
	})
}

func (s *Service) persistFallbackTool(ctx context.Context, sessionID uuid.UUID, toolCall *ToolCall) (*Message, *ToolCall, error) {
	if toolCall == nil {
		return nil, nil, nil
	}

	result := (*ToolResult)(nil)
	def, ok := s.reg.Get(toolCall.Name)
	if !ok {
		result = errResult("tool not available: " + toolCall.Name)
	} else if err := s.policy.CheckToolAllowed(def); err != nil {
		result = errResult(err.Error())
	} else {
		var err error
		result, err = s.router.Dispatch(ctx, *toolCall)
		if err != nil && err != ErrUnknownTool {
			return nil, nil, fmt.Errorf("chat.Service.SendMessage: tool dispatch: %w", err)
		}
		if err == ErrUnknownTool {
			result = errResult("tool not available: " + toolCall.Name)
		}
	}

	toolMsg, err := s.persistToolResult(ctx, sessionID, toolCall, result)
	if err != nil {
		return nil, nil, err
	}
	return toolMsg, toolCall, nil
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

// buildFallbackReply generates the assistant's reply text.
// When s.llm is set, the full session history is forwarded for context.
// Falls back to contextual keyword-based replies if the LLM is unavailable or returns an error.
func (s *Service) buildFallbackReply(ctx context.Context, userContent string, call *ToolCall, history []*Message) string {
	if s.llm != nil {
		msgs := make([]LLMMessage, 0, len(history)+2)
		msgs = append(msgs, LLMMessage{
			Role:    "system",
			Content: s.prompts.SystemPrompt(s.policy, s.toolNames()),
		})
		for _, m := range history {
			if m.Role == RoleTool {
				continue
			}
			msgs = append(msgs, LLMMessage{Role: string(m.Role), Content: m.Content})
		}
		msgs = append(msgs, LLMMessage{Role: "user", Content: userContent})
		if reply, _, err := s.llm.Complete(ctx, msgs); err == nil && strings.TrimSpace(reply) != "" {
			return reply
		} else if err != nil {
			return s.safeModelFallbackReply(userContent, call)
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
		case "compare_runs":
			return "Here is the run comparison. The tool result above lines up the selected runs so you can compare type, status, timing, and summary payloads."
		case "strategy_instance_summary":
			return "Here is the strategy instance summary. The tool result above shows the instance configuration alongside recent signal, trade, and run activity."
		case "blocked_candidate_analysis":
			return "Here is the blocked candidate analysis. The tool result above aggregates recent blocked setups by symbol and blocker code."
		case "recent_research_narrative":
			return "Here is the recent research narrative. The tool result above summarises the latest research and orchestration activity."
		case "confidence_drift_summary":
			return "Here is the confidence drift summary. The tool result above shows recent confidence ranges and averages across signals and candidates."
		case "signal_clustering_overview":
			return "Here is the signal clustering overview. The tool result above groups recent signals by symbol, strategy, and direction."
		default:
			return fmt.Sprintf("I fetched the result for %q. Check the tool result above for the details.", call.Name)
		}
	}
	if userContent == "" {
		return "How can I help you analyse the current trading situation?"
	}
	return staticKeywordReply(userContent)
}

func (s *Service) toHarnessHistory(history []*Message) []harness.Message {
	out := make([]harness.Message, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case RoleTool:
			if msg.ToolName != nil && msg.ToolResult != nil {
				out = append(out, harness.Message{
					Role:    "tool",
					Content: fmt.Sprintf("tool %s result: %s", *msg.ToolName, string(*msg.ToolResult)),
				})
			}
		case RoleUser, RoleAssistant:
			out = append(out, harness.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	return out
}

func (s *Service) allowSession(sessionID uuid.UUID) error {
	if s.limiter == nil {
		return nil
	}
	return s.limiter.Allow(sessionID.String())
}

func (s *Service) runHarnessShadow(sessionID uuid.UUID, userContent string, toolCall *ToolCall, history []*Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _, _, _ = s.harness.AnswerWithEvidence(ctx, sessionID.String(), userContent, s.toHarnessHistory(history), toHarnessToolCall(toolCall))
	}()
}

func toHarnessToolCall(call *ToolCall) *harness.ToolCall {
	if call == nil {
		return nil
	}
	return &harness.ToolCall{
		Name: call.Name,
		Args: call.Args,
	}
}

type harnessModelAdapter struct {
	llm LLMClient
}

func newHarnessModelAdapter(llm LLMClient) harness.Model {
	if llm == nil {
		return nil
	}
	return harnessModelAdapter{llm: llm}
}

func (a harnessModelAdapter) Complete(ctx context.Context, msgs []harness.Message) (string, []harness.ToolCall, error) {
	chatMsgs := make([]LLMMessage, 0, len(msgs))
	for _, msg := range msgs {
		chatMsgs = append(chatMsgs, LLMMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return a.llm.Complete(ctx, chatMsgs)
}

func (s *Service) toolNames() []string {
	tools := s.reg.AllTools()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func (s *Service) safeModelFallbackReply(_ string, call *ToolCall) string {
	if call != nil {
		return fmt.Sprintf("I couldn't complete a model-backed explanation for %q. Check the tool result above and treat it as advisory-only context.", call.Name)
	}
	return "I couldn't complete a model-backed answer. I'm advisory only, so please rely on the recorded tool evidence or ask a narrower question."
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
		return "Use search_research_runs to browse recent orchestration and research runs, compare_runs to line up specific runs, or recent_research_narrative for a broader summary."
	case has("knowledge", "docs", "documentation", "playbook", "runbook"):
		return "Use query_knowledge to search the local markdown knowledge base when it is configured."
	case has("cluster", "grouped signals", "signal cluster"):
		return "Use signal_clustering_overview to group recent signals by symbol, strategy, and direction."
	case has("confidence drift", "confidence trend", "confidence summary"):
		return "Use confidence_drift_summary to summarise recent confidence ranges across signals and candidates."
	case has("trade", "executed", "filled", "order", "position"):
		return "To look up an executed trade, use get_trade with the trade ID. Approvals still have to be made through the Approvals page."
	case has("help", "what can", "what do", "capabilities", "tools", "available"):
		return "I can look up candidates, signals, executed trades, strategy definitions and instances, orchestration runs, blocked candidates, run comparisons, strategy instance summaries, research narratives, confidence drift, signal clusters, and local knowledge snippets. Approvals still have to be made through the Approvals page."
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
