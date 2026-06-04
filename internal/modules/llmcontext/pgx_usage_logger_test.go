package llmcontext

import (
	"context"
	"testing"
)

func TestPGXUsageLoggerRecordsPlannedAndActual(t *testing.T) {
	exec := &recordingPGXExec{}
	logger := NewPGXUsageLogger(exec)

	err := logger.RecordPlanned(PromptPackage{
		TaskType:              TaskApprovalSummary,
		Model:                 "local-small",
		Provider:              "litellm",
		EstimatedInputTokens:  10,
		EstimatedOutputTokens: 4,
		EstimatedCostUSD:      0.001,
		CorrelationID:         "corr-pgx",
		Symbol:                "SPY",
		CacheEligible:         false,
	}, CostDecision{Allowed: true})
	if err != nil {
		t.Fatalf("RecordPlanned returned error: %v", err)
	}
	err = logger.RecordActual(LLMResult{CorrelationID: "corr-pgx", InputTokens: 10, OutputTokens: 2, ActualCostUSD: 0.0004})
	if err != nil {
		t.Fatalf("RecordActual returned error: %v", err)
	}
	if len(exec.queries) != 2 {
		t.Fatalf("expected 2 execs, got %d", len(exec.queries))
	}
}

type recordingPGXExec struct {
	queries []string
	args    [][]any
}

func (e *recordingPGXExec) Exec(_ context.Context, query string, args ...any) error {
	e.queries = append(e.queries, query)
	e.args = append(e.args, args)
	return nil
}
