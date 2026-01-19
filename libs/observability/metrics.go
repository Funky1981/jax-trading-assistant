package observability

import (
	"context"
	"time"
)

func RecordToolCall(ctx context.Context, providerID, toolName string, duration time.Duration, err error) {
	fields := map[string]any{
		"name":       "tool_call",
		"provider":   providerID,
		"tool":       toolName,
		"latency_ms": duration.Milliseconds(),
		"success":    err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}

func RecordRecall(ctx context.Context, providerID, toolName string, err error) {
	fields := map[string]any{
		"name":     "memory_recall",
		"provider": providerID,
		"tool":     toolName,
		"success":  err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}

func RecordRetain(ctx context.Context, bank string, err error) {
	fields := map[string]any{
		"name":    "memory_retain",
		"bank":    bank,
		"success": err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}

func RecordReflectionDuration(ctx context.Context, duration time.Duration, beliefs int) {
	LogEvent(ctx, "info", "metric", map[string]any{
		"name":       "reflection_duration",
		"latency_ms": duration.Milliseconds(),
		"beliefs":    beliefs,
	})
}
