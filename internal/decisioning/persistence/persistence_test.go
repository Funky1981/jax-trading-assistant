package persistence

import (
	"context"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/pipeline"
	"jax-trading-assistant/internal/decisioning/review"
	"jax-trading-assistant/internal/decisioning/risk"
)

func TestMemoryRepositorySavesAndRetrievesPipelineResults(t *testing.T) {
	now := fixedTime()
	cases := []struct {
		name             string
		result           PipelineResultRecord
		wantPaperTicket  bool
		wantRiskRejected bool
	}{
		{
			name: "no trade preserves forbidden actions and review schedule",
			result: pipelineResultRecord(now, core.DecisionNoTrade, pipeline.StatusNoTradeRecorded, func(record *PipelineResultRecord) {
				record.NoTradeReason = "Conflicting macro drivers and no clean trade edge."
			}),
		},
		{
			name: "risk rejected preserves rejection reason and has no paper ticket",
			result: pipelineResultRecord(now, core.DecisionRejectedByRisk, pipeline.StatusTradeCandidateRejectedByRisk, func(record *PipelineResultRecord) {
				record.RiskAssessmentSummary = RiskAssessmentSummary{
					RiskDecision:       string(risk.RiskDecisionReject),
					FinalDecision:      string(core.DecisionRejectedByRisk),
					VetoReasons:        []string{"poor liquidity and wide spread"},
					RequiredActions:    []string{risk.RequiredActionReviewExposure},
					LiveTradingBlocked: true,
				}
				record.RejectionReason = "poor liquidity and wide spread"
			}),
			wantRiskRejected: true,
		},
		{
			name: "paper review ready remains paper only and requires human approval",
			result: pipelineResultRecord(now, core.DecisionTradeCandidate, pipeline.StatusTradeCandidateReadyForPaperReview, func(record *PipelineResultRecord) {
				record.PaperTicketSummary = &PaperTicketSummary{
					PaperTicketID:       "pt_dec_phase9",
					HumanApprovalStatus: "PENDING_REVIEW",
					PaperOnly:           true,
					LiveTradingBlocked:  true,
				}
				record.HumanApprovalRequired = true
				record.PaperOnly = true
				record.LiveTradingBlocked = true
			}),
			wantPaperTicket: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			if err := repo.SavePipelineResult(context.Background(), tc.result); err != nil {
				t.Fatalf("SavePipelineResult returned error: %v", err)
			}

			got, err := repo.GetPipelineResult(context.Background(), tc.result.PipelineID)
			if err != nil {
				t.Fatalf("GetPipelineResult returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.result) {
				t.Fatalf("retrieved result changed:\ngot:  %#v\nwant: %#v", got, tc.result)
			}
			if len(got.ForbiddenActions) == 0 {
				t.Fatal("forbidden actions were not preserved")
			}
			if got.ReviewSchedule.ScheduleID == "" {
				t.Fatal("review schedule was not preserved")
			}
			if (got.PaperTicketSummary != nil) != tc.wantPaperTicket {
				t.Fatalf("paper ticket present = %v, want %v", got.PaperTicketSummary != nil, tc.wantPaperTicket)
			}
			if tc.wantRiskRejected && got.RejectionReason == "" {
				t.Fatal("risk rejection reason was not preserved")
			}
			if got.FinalDecision != string(tc.result.FinalDecision) || got.FinalStatus != string(tc.result.FinalStatus) {
				t.Fatal("persistence upgraded or altered final decision state")
			}
		})
	}
}

func TestMemoryRepositorySavesDecisionLogsSchedulesOutcomeReviewsAndPendingSchedules(t *testing.T) {
	now := fixedTime()
	repo := NewMemoryRepository()
	log := decisionLog(now, core.DecisionNoTrade)
	schedule := log.ReviewSchedule
	schedule.NextReviewAt = now.Add(-time.Hour)
	outcome := outcomeReview(now, log.DecisionID, log.EventID)

	if err := repo.SaveDecisionLog(context.Background(), log); err != nil {
		t.Fatalf("SaveDecisionLog returned error: %v", err)
	}
	if err := repo.SaveReviewSchedule(context.Background(), schedule); err != nil {
		t.Fatalf("SaveReviewSchedule returned error: %v", err)
	}
	if err := repo.SaveOutcomeReview(context.Background(), outcome); err != nil {
		t.Fatalf("SaveOutcomeReview returned error: %v", err)
	}

	gotLog, err := repo.GetDecisionLog(context.Background(), log.DecisionID)
	if err != nil {
		t.Fatalf("GetDecisionLog returned error: %v", err)
	}
	if !reflect.DeepEqual(gotLog, log) {
		t.Fatalf("decision log changed:\ngot:  %#v\nwant: %#v", gotLog, log)
	}
	gotSchedule, err := repo.GetReviewSchedule(context.Background(), schedule.DecisionID)
	if err != nil {
		t.Fatalf("GetReviewSchedule returned error: %v", err)
	}
	if !reflect.DeepEqual(gotSchedule, schedule) {
		t.Fatalf("review schedule changed:\ngot:  %#v\nwant: %#v", gotSchedule, schedule)
	}
	gotOutcome, err := repo.GetOutcomeReview(context.Background(), outcome.ReviewID)
	if err != nil {
		t.Fatalf("GetOutcomeReview returned error: %v", err)
	}
	if !reflect.DeepEqual(gotOutcome, outcome) {
		t.Fatalf("outcome review changed:\ngot:  %#v\nwant: %#v", gotOutcome, outcome)
	}

	pending, err := repo.ListDecisionsPendingReview(context.Background(), now)
	if err != nil {
		t.Fatalf("ListDecisionsPendingReview returned error: %v", err)
	}
	if len(pending) != 1 || pending[0].DecisionID != schedule.DecisionID {
		t.Fatalf("pending schedules = %#v, want decision %s", pending, schedule.DecisionID)
	}
}

func TestMemoryRepositoryCreatesAuditRecordForPipelineResultSaved(t *testing.T) {
	now := fixedTime()
	repo := NewMemoryRepository()
	record := pipelineResultRecord(now, core.DecisionNoTrade, pipeline.StatusNoTradeRecorded, nil)

	if err := repo.SavePipelineResult(context.Background(), record); err != nil {
		t.Fatalf("SavePipelineResult returned error: %v", err)
	}

	audits, err := repo.ListAuditRecordsForDecision(context.Background(), record.DecisionID)
	if err != nil {
		t.Fatalf("ListAuditRecordsForDecision returned error: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("audit record count = %d, want 1", len(audits))
	}
	audit := audits[0]
	if audit.TraceID != record.TraceID || audit.PipelineID != record.PipelineID || audit.DecisionID != record.DecisionID || audit.EventID != record.EventID {
		t.Fatalf("audit identifiers were not populated from pipeline record: %#v", audit)
	}
	if audit.SourceModule != SourceModulePersistence || audit.Action != AuditActionPipelineResultSaved {
		t.Fatalf("audit action/source = %s/%s, want %s/%s", audit.Action, audit.SourceModule, AuditActionPipelineResultSaved, SourceModulePersistence)
	}
	if !reflect.DeepEqual(audit.ForbiddenActions, record.ForbiddenActions) {
		t.Fatalf("audit forbidden actions = %#v, want %#v", audit.ForbiddenActions, record.ForbiddenActions)
	}
}

func TestMemoryRepositoryCreatesAuditRecordForReviewScheduleSaved(t *testing.T) {
	now := fixedTime()
	repo := NewMemoryRepository()
	schedule := review.NewReviewSchedule("dec_schedule_phase9", now)

	if err := repo.SaveReviewSchedule(context.Background(), schedule); err != nil {
		t.Fatalf("SaveReviewSchedule returned error: %v", err)
	}

	audits, err := repo.ListAuditRecordsForDecision(context.Background(), schedule.DecisionID)
	if err != nil {
		t.Fatalf("ListAuditRecordsForDecision returned error: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("audit record count = %d, want 1", len(audits))
	}
	if audits[0].Action != AuditActionReviewScheduleSaved || audits[0].SourceModule != SourceModulePersistence {
		t.Fatalf("audit action/source = %s/%s", audits[0].Action, audits[0].SourceModule)
	}
}

func TestMemoryRepositoryDoesNotUpgradeWatchOrNoTrade(t *testing.T) {
	now := fixedTime()
	repo := NewMemoryRepository()
	cases := []PipelineResultRecord{
		pipelineResultRecord(now, core.DecisionWatch, pipeline.StatusWatchRecorded, nil),
		pipelineResultRecord(now, core.DecisionNoTrade, pipeline.StatusNoTradeRecorded, nil),
	}

	for _, want := range cases {
		if err := repo.SavePipelineResult(context.Background(), want); err != nil {
			t.Fatalf("SavePipelineResult returned error: %v", err)
		}
		got, err := repo.GetPipelineResult(context.Background(), want.PipelineID)
		if err != nil {
			t.Fatalf("GetPipelineResult returned error: %v", err)
		}
		if got.FinalDecision != string(want.FinalDecision) || got.FinalStatus != string(want.FinalStatus) {
			t.Fatalf("saved/retrieved state changed from %s/%s to %s/%s", want.FinalDecision, want.FinalStatus, got.FinalDecision, got.FinalStatus)
		}
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func pipelineResultRecord(now time.Time, decision core.DecisionValue, status pipeline.FinalStatus, mutate func(*PipelineResultRecord)) PipelineResultRecord {
	decisionID := "dec_phase9_" + string(decision)
	record := PipelineResultRecord{
		TraceID:               "trace_" + decisionID,
		PipelineID:            "pipe_" + decisionID,
		EventID:               "evt_phase9",
		DecisionID:            decisionID,
		FinalDecision:         string(decision),
		FinalStatus:           string(status),
		AllowedActions:        []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionReviewLater},
		ForbiddenActions:      []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
		HumanApprovalRequired: true,
		PaperOnly:             true,
		LiveTradingBlocked:    true,
		ReviewSchedule:        review.NewReviewSchedule(decisionID, now),
		CreatedAt:             now,
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func decisionLog(now time.Time, decision core.DecisionValue) review.DecisionLog {
	log, validation := review.NewDecisionLog(review.DecisionLogInput{
		Decision: core.Decision{
			DecisionID:             "dec_log_phase9",
			EventID:                "evt_phase9",
			Brain:                  core.BrainDecisionCore,
			Decision:               decision,
			PrimaryReason:          "Conflicting macro drivers and no clean trade edge.",
			AllowedActions:         []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionReviewLater},
			ForbiddenActions:       []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
			RequiredConfirmations:  []string{"oil stabilises"},
			InvalidationConditions: []string{"FTSE breaks lower on strong volume"},
		},
		FinalDecision: string(decision),
		CreatedAt:     now,
		MemoryTags:    []string{"no_trade", "phase9"},
	})
	if !validation.IsValid {
		panic(validation.ValidationErrors)
	}
	return log
}

func outcomeReview(now time.Time, decisionID string, eventID string) review.OutcomeReview {
	outcome, validation := review.NewOutcomeReview(review.OutcomeReviewInput{
		DecisionID:         decisionID,
		EventID:            eventID,
		ReviewWindow:       review.ReviewWindow1Day,
		OriginalDecision:   core.DecisionNoTrade,
		FinalDecision:      string(core.DecisionNoTrade),
		WasDecisionCorrect: true,
		AvoidedLoss:        true,
		LessonSummary:      "No-trade remained valid after the review window.",
		MemoryTags:         []string{"no_trade", "reviewed"},
		CreatedAt:          now,
	})
	if !validation.IsValid {
		panic(validation.ValidationErrors)
	}
	return outcome
}
