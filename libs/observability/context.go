package observability

import "context"

type contextKey string

const (
	runIDKey  contextKey = "run_id"
	taskIDKey contextKey = "task_id"
	symbolKey contextKey = "symbol"
	flowIDKey contextKey = "flow_id"
)

// RunInfo carries trace identifiers through a request context.
// FlowID spans the entire trade decision chain (quote → signal → execution).
// RunID is per-orchestration run. TaskID is per agent task.
type RunInfo struct {
	RunID  string
	TaskID string
	Symbol string
	FlowID string
}

func WithRunInfo(ctx context.Context, info RunInfo) context.Context {
	if info.RunID != "" {
		ctx = context.WithValue(ctx, runIDKey, info.RunID)
	}
	if info.TaskID != "" {
		ctx = context.WithValue(ctx, taskIDKey, info.TaskID)
	}
	if info.Symbol != "" {
		ctx = context.WithValue(ctx, symbolKey, info.Symbol)
	}
	if info.FlowID != "" {
		ctx = context.WithValue(ctx, flowIDKey, info.FlowID)
	}
	return ctx
}

func RunInfoFromContext(ctx context.Context) RunInfo {
	info := RunInfo{}
	if value := ctx.Value(runIDKey); value != nil {
		if runID, ok := value.(string); ok {
			info.RunID = runID
		}
	}
	if value := ctx.Value(taskIDKey); value != nil {
		if taskID, ok := value.(string); ok {
			info.TaskID = taskID
		}
	}
	if value := ctx.Value(symbolKey); value != nil {
		if symbol, ok := value.(string); ok {
			info.Symbol = symbol
		}
	}
	if value := ctx.Value(flowIDKey); value != nil {
		if flowID, ok := value.(string); ok {
			info.FlowID = flowID
		}
	}
	return info
}

// WithFlowID attaches a flow_id to the context. A flow_id traces the full
// lifecycle of a trade decision: quote ingested → signal → approval → execution.
func WithFlowID(ctx context.Context, flowID string) context.Context {
	if flowID == "" {
		return ctx
	}
	return context.WithValue(ctx, flowIDKey, flowID)
}

// FlowIDFromContext retrieves the flow_id set by WithFlowID.
func FlowIDFromContext(ctx context.Context) string {
	if v := ctx.Value(flowIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}
