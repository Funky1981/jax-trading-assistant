package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/libs/chattools"
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

type ToolDescriptor struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ArgKey        string `json:"argKey"`
	ArgLabel      string `json:"argLabel"`
	EvidenceLevel string `json:"evidenceLevel"`
	Freshness     string `json:"freshness"`
	Allowed       bool   `json:"allowed"`
	PolicyReason  string `json:"policyReason,omitempty"`
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
		return r.dispatchShared(ctx, call.Args, chattools.GetCandidateTrade)
	case "get_signal":
		return r.dispatchShared(ctx, call.Args, chattools.GetSignal)
	case "get_trade":
		return r.dispatchShared(ctx, call.Args, chattools.GetTrade)
	case "get_strategy":
		return r.dispatchShared(ctx, call.Args, chattools.GetStrategy)
	case "get_strategy_instance":
		return r.dispatchShared(ctx, call.Args, chattools.GetStrategyInstance)
	case "get_orchestration_run":
		return r.dispatchShared(ctx, call.Args, chattools.GetOrchestrationRun)
	case "search_research_runs":
		return r.dispatchShared(ctx, call.Args, chattools.SearchResearchRuns)
	case "explain_trade_blockers":
		return r.dispatchShared(ctx, call.Args, chattools.ExplainTradeBlockers)
	case "list_pending_approvals":
		return r.dispatchShared(ctx, call.Args, chattools.ListPendingApprovals)
	case "list_recent_blocked_candidates":
		return r.dispatchShared(ctx, call.Args, chattools.ListRecentBlockedCandidates)
	case "search_candidates":
		return r.dispatchShared(ctx, call.Args, chattools.SearchCandidates)
	case "query_knowledge":
		return r.dispatchShared(ctx, call.Args, chattools.QueryKnowledge)
	case "compare_runs":
		return r.dispatchShared(ctx, call.Args, chattools.CompareRuns)
	case "strategy_instance_summary":
		return r.dispatchShared(ctx, call.Args, chattools.StrategyInstanceSummary)
	case "blocked_candidate_analysis":
		return r.dispatchShared(ctx, call.Args, chattools.BlockedCandidateAnalysis)
	case "recent_research_narrative":
		return r.dispatchShared(ctx, call.Args, chattools.RecentResearchNarrative)
	case "confidence_drift_summary":
		return r.dispatchShared(ctx, call.Args, chattools.ConfidenceDriftSummary)
	case "signal_clustering_overview":
		return r.dispatchShared(ctx, call.Args, chattools.SignalClusteringOverview)
	default:
		return nil, ErrUnknownTool
	}
}

func (r *ToolRouter) dispatchShared(ctx context.Context, args json.RawMessage, handler chattools.HandlerFunc) (*ToolResult, error) {
	raw, err := handler(ctx, r.pool, args)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return &ToolResult{Ok: true, Data: raw}, nil
}

// AvailableTools returns human-readable descriptions for the frontend.
// argKey is the primary argument name; argLabel is the placeholder text for the UI input.
func AvailableTools(policy harness.Policy) []ToolDescriptor {
	tools := chattools.DefaultTools()
	out := make([]ToolDescriptor, 0, len(tools))
	for _, tool := range tools {
		desc := ToolDescriptor{
			Name:          tool.Name,
			Description:   tool.Description,
			ArgKey:        tool.ArgKey,
			ArgLabel:      tool.ArgLabel,
			EvidenceLevel: tool.EvidenceLevel,
			Freshness:     tool.FreshnessExpectation,
			Allowed:       true,
		}
		if err := policy.CheckToolAllowed(harness.ToolDefinition{
			Name:                 tool.Name,
			ReadOnly:             tool.ReadOnly,
			EvidenceLevel:        harness.EvidenceLevel(tool.EvidenceLevel),
			FreshnessExpectation: tool.FreshnessExpectation,
		}); err != nil {
			desc.Allowed = false
			desc.PolicyReason = err.Error()
		}
		out = append(out, desc)
	}
	return out
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
