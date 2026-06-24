package observability

import (
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/persistence"
	"jax-trading-assistant/internal/decisioning/pipeline"
	"jax-trading-assistant/internal/decisioning/review"
)

func TestSummaryForNoTrade(t *testing.T) {
	record := observableRecord(core.DecisionNoTrade, pipeline.StatusNoTradeRecorded)
	record.NoTradeReason = "Conflicting macro drivers and no clean trade edge."
	record.ValidationWarnings = []string{"portfolio context is missing"}

	summary := NewSummary(record)

	if summary.PipelineID != record.PipelineID || summary.EventID != record.EventID {
		t.Fatalf("summary identifiers = %s/%s, want %s/%s", summary.PipelineID, summary.EventID, record.PipelineID, record.EventID)
	}
	if summary.FinalStatus != string(pipeline.StatusNoTradeRecorded) || summary.FinalDecision != string(core.DecisionNoTrade) {
		t.Fatalf("summary final state = %s/%s", summary.FinalStatus, summary.FinalDecision)
	}
	if summary.NoTradeReason != record.NoTradeReason {
		t.Fatalf("no trade reason = %q, want %q", summary.NoTradeReason, record.NoTradeReason)
	}
	if !summary.ReviewScheduled || !summary.LiveTradingBlocked {
		t.Fatalf("review/live flags not preserved: %#v", summary)
	}
	if summary.WarningCount != 1 || summary.ErrorCount != 0 {
		t.Fatalf("warning/error counts = %d/%d, want 1/0", summary.WarningCount, summary.ErrorCount)
	}
}

func TestSummaryForPaperReviewReadyCandidate(t *testing.T) {
	record := observableRecord(core.DecisionTradeCandidate, pipeline.StatusTradeCandidateReadyForPaperReview)
	record.PaperTicketSummary = &persistence.PaperTicketSummary{
		PaperTicketID:       "pt_phase9",
		HumanApprovalStatus: "PENDING_REVIEW",
		PaperOnly:           true,
		LiveTradingBlocked:  true,
	}

	summary := NewSummary(record)

	if !summary.PaperReviewReady {
		t.Fatal("paper review ready candidate was not surfaced")
	}
	if !summary.HumanApprovalRequired || !summary.PaperOnly || !summary.LiveTradingBlocked {
		t.Fatalf("paper-only approval flags not preserved: %#v", summary)
	}
	if summary.FinalDecision != string(core.DecisionTradeCandidate) || summary.FinalStatus != string(pipeline.StatusTradeCandidateReadyForPaperReview) {
		t.Fatalf("summary final state = %s/%s", summary.FinalStatus, summary.FinalDecision)
	}
}

func TestSummaryDoesNotExposeHiddenReasoning(t *testing.T) {
	summaryType := reflect.TypeOf(Summary{})
	for _, forbidden := range []string{"HiddenReasoning", "ChainOfThought", "ReasoningTrace", "ReasoningTraceSummary"} {
		if _, ok := summaryType.FieldByName(forbidden); ok {
			t.Fatalf("summary exposes hidden reasoning field %q", forbidden)
		}
	}
}

func TestTraceRecordsPipelineRunIdentifiersAndCounts(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	trace := NewTrace(TraceInput{
		TraceID:        "trace_phase9",
		PipelineID:     "pipe_phase9",
		EventID:        "evt_phase9",
		DecisionID:     "dec_phase9",
		StartedAt:      now,
		CompletedAt:    now.Add(time.Second),
		ModulesVisited: []string{"decision_core", "risk_veto", "review_schedule"},
		FinalStatus:    string(pipeline.StatusNoTradeRecorded),
		FinalDecision:  string(core.DecisionNoTrade),
		Warnings:       []string{"portfolio context is missing"},
	})

	if trace.TraceID == "" || trace.PipelineID == "" || trace.EventID == "" || trace.DecisionID == "" {
		t.Fatalf("trace identifiers missing: %#v", trace)
	}
	if trace.FinalStatus != string(pipeline.StatusNoTradeRecorded) || trace.FinalDecision != string(core.DecisionNoTrade) {
		t.Fatalf("trace final state = %s/%s", trace.FinalStatus, trace.FinalDecision)
	}
	if len(trace.ModulesVisited) != 3 || len(trace.Warnings) != 1 || len(trace.Errors) != 0 {
		t.Fatalf("trace module/warning/error counts changed: %#v", trace)
	}
}

func observableRecord(decision core.DecisionValue, status pipeline.FinalStatus) persistence.PipelineResultRecord {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	decisionID := "dec_obs_" + string(decision)
	return persistence.PipelineResultRecord{
		TraceID:               "trace_" + decisionID,
		PipelineID:            "pipe_" + decisionID,
		EventID:               "evt_obs",
		DecisionID:            decisionID,
		FinalDecision:         string(decision),
		FinalStatus:           string(status),
		SourceModules:         []string{"decision_core", "risk_veto", "review_schedule"},
		HumanApprovalRequired: true,
		PaperOnly:             true,
		LiveTradingBlocked:    true,
		ReviewSchedule:        review.NewReviewSchedule(decisionID, now),
		ForbiddenActions:      []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
		CreatedAt:             now,
	}
}
