package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ToolCall represents an assistant tool invocation.
type ToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// ToolResult is the output of a tool call.
type ToolResult struct {
	Ok    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// ToolRouter dispatches named tool calls to read-mostly data queries.
// The assistant MUST NOT mutate trading state through these tools.
type ToolRouter struct {
	pool *pgxpool.Pool
}

// NewToolRouter creates a ToolRouter.
func NewToolRouter(pool *pgxpool.Pool) *ToolRouter {
	return &ToolRouter{pool: pool}
}

// Dispatch executes a tool call and returns the result.
// Returns ErrUnknownTool if the tool name is not registered.
func (r *ToolRouter) Dispatch(ctx context.Context, call ToolCall) (*ToolResult, error) {
	switch call.Name {
	case "get_candidate_trade":
		return r.getCandidateTrade(ctx, call.Args)
	case "get_signal":
		return r.getSignal(ctx, call.Args)
	case "get_trade":
		return r.getTrade(ctx, call.Args)
	case "get_strategy":
		return r.getStrategy(ctx, call.Args)
	case "get_strategy_instance":
		return r.getStrategyInstance(ctx, call.Args)
	case "get_orchestration_run":
		return r.getOrchestrationRun(ctx, call.Args)
	case "search_research_runs":
		return r.searchResearchRuns(ctx, call.Args)
	case "explain_trade_blockers":
		return r.explainTradeBlockers(ctx, call.Args)
	case "list_pending_approvals":
		return r.listPendingApprovals(ctx, call.Args)
	case "list_recent_blocked_candidates":
		return r.listRecentBlockedCandidates(ctx, call.Args)
	case "search_candidates":
		return r.searchCandidates(ctx, call.Args)
	case "query_knowledge":
		return r.queryKnowledge(call.Args)
	default:
		return nil, ErrUnknownTool
	}
}

// AvailableTools returns human-readable descriptions for the frontend.
// argKey is the primary argument name; argLabel is the placeholder text for the UI input.
func AvailableTools() []map[string]string {
	return []map[string]string{
		{"name": "get_candidate_trade", "description": "Retrieve a candidate trade by ID", "argKey": "candidateId", "argLabel": "Candidate ID"},
		{"name": "get_signal", "description": "Retrieve a strategy signal by ID", "argKey": "signalId", "argLabel": "Signal ID"},
		{"name": "get_trade", "description": "Retrieve an executed trade by ID", "argKey": "tradeId", "argLabel": "Trade ID"},
		{"name": "get_strategy", "description": "Retrieve strategy definition by ID", "argKey": "strategyId", "argLabel": "Strategy ID"},
		{"name": "get_strategy_instance", "description": "Retrieve strategy instance by ID", "argKey": "instanceId", "argLabel": "Instance ID"},
		{"name": "get_orchestration_run", "description": "Retrieve an orchestration run by ID", "argKey": "runId", "argLabel": "Run ID"},
		{"name": "search_research_runs", "description": "Search recent orchestration/research runs", "argKey": "symbol", "argLabel": "Symbol (optional)"},
		{"name": "explain_trade_blockers", "description": "Explain why a candidate was blocked", "argKey": "candidateId", "argLabel": "Candidate ID"},
		{"name": "list_pending_approvals", "description": "List candidates currently awaiting approval", "argKey": "limit", "argLabel": "Limit (optional)"},
		{"name": "list_recent_blocked_candidates", "description": "List recently blocked candidates", "argKey": "limit", "argLabel": "Limit (optional)"},
		{"name": "search_candidates", "description": "Search recent candidates by symbol or status", "argKey": "query", "argLabel": "Symbol or status"},
		{"name": "query_knowledge", "description": "Search local knowledge markdown when configured", "argKey": "query", "argLabel": "Knowledge query"},
	}
}

// ── Tool implementations (read-only DB queries) ───────────────────────────────

func (r *ToolRouter) getCandidateTrade(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		CandidateID string `json:"candidateId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.CandidateID == "" {
		return errResult("candidateId required"), nil
	}
	id, err := uuid.Parse(p.CandidateID)
	if err != nil {
		return errResult("invalid candidateId"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT ct.id::text, ct.signal_id::text, ct.strategy_id, ct.artifact_id::text, ct.symbol, ct.signal_type, ct.status, ct.confidence, ct.entry_price,
			       ct.stop_loss, ct.take_profit, ct.reasoning, ct.block_reason, ct.blocked_reason_code, ct.detected_at,
			       ei.id::text AS execution_instruction_id, ei.trade_id
			FROM candidate_trades ct
			LEFT JOIN LATERAL (
				SELECT id, trade_id
				FROM execution_instructions
				WHERE candidate_id = ct.id
				ORDER BY created_at DESC
				LIMIT 1
			) ei ON TRUE
			WHERE ct.id = $1) t`, id)
}

func (r *ToolRouter) getSignal(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		SignalID string `json:"signalId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.SignalID == "" {
		return errResult("signalId required"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, strategy_id, signal_type, confidence, status, generated_at
			FROM strategy_signals WHERE id = $1::uuid) t`, p.SignalID)
}

func (r *ToolRouter) getTrade(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		TradeID string `json:"tradeId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.TradeID == "" {
		return errResult("tradeId required"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, direction, quantity, entry_price, status, created_at
			FROM trades WHERE id = $1::uuid) t`, p.TradeID)
}

func (r *ToolRouter) getStrategy(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	// Strategy data lives in registry, not DB — return a placeholder.
	var p struct {
		StrategyID string `json:"strategyId"`
	}
	_ = json.Unmarshal(args, &p)
	return okResult(map[string]any{"strategyId": p.StrategyID, "note": "strategy details available via /api/v1/strategies/{id}"})
}

func (r *ToolRouter) getStrategyInstance(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		InstanceID string `json:"instanceId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.InstanceID == "" {
		return errResult("instanceId required"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, name, strategy_type_id, enabled, session_timezone, flatten_by_close_time
			FROM strategy_instances WHERE id = $1::uuid) t`, p.InstanceID)
}

func (r *ToolRouter) getOrchestrationRun(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.RunID == "" {
		return errResult("runId required"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, trigger_type, status, started_at, completed_at
			FROM orchestration_runs WHERE id = $1::uuid) t`, p.RunID)
}

func (r *ToolRouter) searchResearchRuns(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		Symbol string `json:"symbol"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id::text, symbol, trigger_type, status, started_at
		   FROM orchestration_runs
		  WHERE ($1 = '' OR symbol = $1)
		  ORDER BY started_at DESC LIMIT $2`, p.Symbol, p.Limit)
	if err != nil {
		return errResult(err.Error()), nil
	}
	defer rows.Close()
	var runs []map[string]any
	for rows.Next() {
		var id, sym, trig, status string
		var startedAt any
		if err := rows.Scan(&id, &sym, &trig, &status, &startedAt); err != nil {
			continue
		}
		runs = append(runs, map[string]any{"id": id, "symbol": sym, "triggerType": trig, "status": status, "startedAt": startedAt})
	}
	return okResult(runs)
}

func (r *ToolRouter) explainTradeBlockers(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		CandidateID string `json:"candidateId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.CandidateID == "" {
		return errResult("candidateId required"), nil
	}
	return rowQueryResult(ctx, r.pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, status, block_reason, blocked_reason_code, reasoning, detected_at
			FROM candidate_trades WHERE id = $1::uuid) t`, p.CandidateID)
}

func (r *ToolRouter) listPendingApprovals(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		Limit int `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, status, confidence, detected_at, expires_at,
		       COALESCE(strategy_id,''), COALESCE(signal_id::text,''), COALESCE(artifact_id::text,'')
		FROM candidate_trades
		WHERE status = 'awaiting_approval'
		ORDER BY detected_at ASC
		LIMIT $1
	`, p.Limit)
	if err != nil {
		return errResult(err.Error()), nil
	}
	defer rows.Close()
	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType, status, strategyID, signalID, artifactID string
		var confidence *float64
		var detectedAt time.Time
		var expiresAt *time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &status, &confidence, &detectedAt, &expiresAt, &strategyID, &signalID, &artifactID); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":         id,
			"symbol":     symbol,
			"signalType": signalType,
			"status":     status,
			"confidence": confidence,
			"detectedAt": detectedAt,
			"expiresAt":  expiresAt,
			"strategyId": strategyID,
			"signalId":   signalID,
			"artifactId": artifactID,
		})
	}
	return okResult(items)
}

func (r *ToolRouter) listRecentBlockedCandidates(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		Limit int `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, blocked_reason_code, block_reason, confidence, blocked_at, detected_at
		FROM candidate_trades
		WHERE status = 'blocked'
		ORDER BY COALESCE(blocked_at, detected_at) DESC
		LIMIT $1
	`, p.Limit)
	if err != nil {
		return errResult(err.Error()), nil
	}
	defer rows.Close()
	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType string
		var reasonCode, reason *string
		var confidence *float64
		var blockedAt, detectedAt time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &reasonCode, &reason, &confidence, &blockedAt, &detectedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":                id,
			"symbol":            symbol,
			"signalType":        signalType,
			"blockedReasonCode": reasonCode,
			"blockReason":       reason,
			"confidence":        confidence,
			"blockedAt":         blockedAt,
			"detectedAt":        detectedAt,
		})
	}
	return okResult(items)
}

func (r *ToolRouter) searchCandidates(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, status, blocked_reason_code, detected_at
		FROM candidate_trades
		WHERE ($1 = '' OR symbol ILIKE '%' || $1 || '%' OR status ILIKE '%' || $1 || '%')
		ORDER BY detected_at DESC
		LIMIT $2
	`, strings.TrimSpace(p.Query), p.Limit)
	if err != nil {
		return errResult(err.Error()), nil
	}
	defer rows.Close()
	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType, status string
		var reasonCode *string
		var detectedAt time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &status, &reasonCode, &detectedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":                id,
			"symbol":            symbol,
			"signalType":        signalType,
			"status":            status,
			"blockedReasonCode": reasonCode,
			"detectedAt":        detectedAt,
		})
	}
	return okResult(items)
}

func (r *ToolRouter) queryKnowledge(args json.RawMessage) (*ToolResult, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Query) == "" {
		return errResult("query required"), nil
	}
	if p.Limit <= 0 || p.Limit > 10 {
		p.Limit = 5
	}
	root := os.Getenv("JAX_KNOWLEDGE_ROOT")
	if strings.TrimSpace(root) == "" {
		root = filepath.Join("knowledge", "md")
	}
	if _, err := os.Stat(root); err != nil {
		return errResult("knowledge root not configured"), nil
	}
	query := strings.ToLower(strings.TrimSpace(p.Query))
	matches := make([]map[string]any, 0, p.Limit)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || len(matches) >= p.Limit {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(raw)
		idx := strings.Index(strings.ToLower(text), query)
		if idx < 0 {
			return nil
		}
		start := idx - 120
		if start < 0 {
			start = 0
		}
		end := idx + len(query) + 160
		if end > len(text) {
			end = len(text)
		}
		matches = append(matches, map[string]any{
			"path":    path,
			"excerpt": strings.TrimSpace(text[start:end]),
		})
		return nil
	})
	return okResult(matches)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func rowQueryResult(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (*ToolResult, error) {
	var raw []byte
	if err := pool.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return errResult(fmt.Sprintf("not found: %v", err)), nil
	}
	return &ToolResult{Ok: true, Data: json.RawMessage(raw)}, nil
}

func errResult(msg string) *ToolResult {
	return &ToolResult{Ok: false, Error: msg}
}

func okResult(data any) (*ToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &ToolResult{Ok: true, Data: json.RawMessage(b)}, nil
}

// ErrUnknownTool is returned when a tool name is not registered.
var ErrUnknownTool = fmt.Errorf("unknown tool")
