package observability

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", 0)

func LogEvent(ctx context.Context, level string, event string, fields map[string]any) {
	payload := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339),
		"level": level,
		"event": event,
	}

	info := RunInfoFromContext(ctx)
	if info.FlowID != "" {
		payload["flow_id"] = info.FlowID
	}
	if info.RunID != "" {
		payload["run_id"] = info.RunID
	}
	if info.TaskID != "" {
		payload["task_id"] = info.TaskID
	}
	if info.Symbol != "" {
		payload["symbol"] = info.Symbol
	}

	for key, value := range normalizeFields(fields) {
		payload[key] = value
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		logger.Printf("{\"level\":\"error\",\"event\":\"log_marshal_failed\",\"error\":%q}", err.Error())
		return
	}
	logger.Print(string(raw))
}

func LogToolStart(ctx context.Context, providerID, toolName string, input any) {
	LogEvent(ctx, "info", "tool_start", map[string]any{
		"provider": providerID,
		"tool":     toolName,
		"input":    input,
	})
}

func LogToolEnd(ctx context.Context, providerID, toolName string, duration time.Duration, err error) {
	fields := map[string]any{
		"provider":   providerID,
		"tool":       toolName,
		"latency_ms": duration.Milliseconds(),
		"success":    err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "tool_end", fields)
}

func LogRetention(ctx context.Context, bank string, err error) {
	fields := map[string]any{
		"bank":    bank,
		"success": err == nil,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogEvent(ctx, "info", "memory_retain", fields)
}

func normalizeFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		switch key {
		case "input", "payload":
			out[key] = RedactValue(value)
			continue
		}
		if err, ok := value.(error); ok {
			out[key] = err.Error()
			continue
		}
		out[key] = value
	}
	return out
}
