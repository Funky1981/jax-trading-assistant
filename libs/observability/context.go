package observability

import "context"

type contextKey string

const (
	runIDKey  contextKey = "run_id"
	taskIDKey contextKey = "task_id"
	symbolKey contextKey = "symbol"
)

type RunInfo struct {
	RunID  string
	TaskID string
	Symbol string
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
	return info
}
