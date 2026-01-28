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

func RecordStrategySignal(ctx context.Context, strategy string, signalType string, confidence float64) {
	LogEvent(ctx, "info", "metric", map[string]any{
		"name":       "strategy_signal",
		"strategy":   strategy,
		"type":       signalType,
		"confidence": confidence,
	})
}

func RecordOrchestrationRun(ctx context.Context, duration time.Duration, stages int, err error) {
	fields := map[string]any{
		"name":       "orchestration_run",
		"latency_ms": duration.Milliseconds(),
		"stages":     stages,
		"success":    err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}

func RecordResearchQuery(ctx context.Context, service string, duration time.Duration, err error) {
	fields := map[string]any{
		"name":       "research_query",
		"service":    service,
		"latency_ms": duration.Milliseconds(),
		"success":    err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}

func RecordAgent0Plan(ctx context.Context, duration time.Duration, steps int, confidence float64, err error) {
	fields := map[string]any{
		"name":       "agent0_plan",
		"latency_ms": duration.Milliseconds(),
		"steps":      steps,
		"confidence": confidence,
		"success":    err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "metric", fields)
}
