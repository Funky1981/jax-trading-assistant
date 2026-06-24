package replay

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/review"
)

func TestRunIncludesNoTradeRiskRejectedAndPaperOutcomesByDefault(t *testing.T) {
	got := Run(ReplayInput{
		ReplayID:  "replay_phase_10",
		CreatedAt: fixedReplayNow(),
		Records: []Record{
			{
				RecordID:      "rec_no_trade",
				DecisionID:    "dec_no_trade",
				EventID:       "evt_macro",
				Asset:         "FTSE",
				EventType:     "macro",
				SetupFamily:   "macro_conflict",
				FinalDecision: core.DecisionNoTrade,
				Lessons: []review.Lesson{
					{LessonID: "lesson_correct_no_trade", LessonType: review.LessonCorrectNoTrade},
					{LessonID: "lesson_avoided_loss", LessonType: review.LessonAvoidedLoss},
				},
				CreatedAt: fixedReplayNow(),
			},
			{
				RecordID:       "rec_watch",
				DecisionID:     "dec_watch",
				EventID:        "evt_rates",
				Asset:          "GBP",
				EventType:      "central_bank",
				SetupFamily:    "rates_fx",
				FinalDecision:  core.DecisionWatch,
				ResearchStatus: "BACKTESTED_WEAK",
				CreatedAt:      fixedReplayNow(),
			},
			{
				RecordID:       "rec_risk",
				DecisionID:     "dec_risk",
				EventID:        "evt_oil",
				Asset:          "SHEL",
				EventType:      "commodity",
				SetupFamily:    "commodity_equity",
				FinalDecision:  core.DecisionRejectedByRisk,
				RejectedByRisk: true,
				Lessons: []review.Lesson{
					{LessonID: "lesson_risk_helped", LessonType: review.LessonRiskVetoHelped},
				},
				CreatedAt: fixedReplayNow(),
			},
			{
				RecordID:       "rec_paper",
				DecisionID:     "dec_paper",
				EventID:        "evt_earnings",
				Asset:          "AAPL",
				EventType:      "earnings",
				SetupFamily:    "earnings_revision",
				FinalDecision:  core.DecisionTradeCandidate,
				PaperApproved:  true,
				PaperOutcome:   PaperOutcomeWorked,
				ResearchStatus: "BACKTESTED_PROMISING",
				Lessons: []review.Lesson{
					{LessonID: "lesson_paper_worked", LessonType: review.LessonPaperSetupWorked},
				},
				CreatedAt: fixedReplayNow(),
			},
		},
	})

	if got.RecordsProcessed != 4 {
		t.Fatalf("records processed = %d, want 4", got.RecordsProcessed)
	}
	if got.NoTradeCount != 1 {
		t.Fatalf("no trade count = %d, want 1", got.NoTradeCount)
	}
	if got.WatchCount != 1 {
		t.Fatalf("watch count = %d, want 1", got.WatchCount)
	}
	if got.RejectedByRiskCount != 1 {
		t.Fatalf("rejected by risk count = %d, want 1", got.RejectedByRiskCount)
	}
	if got.PaperApprovedCount != 1 {
		t.Fatalf("paper approved count = %d, want 1", got.PaperApprovedCount)
	}
	if got.CorrectNoTradeCount != 1 {
		t.Fatalf("correct no trade count = %d, want 1", got.CorrectNoTradeCount)
	}
	if got.AvoidedLossCount != 1 {
		t.Fatalf("avoided loss count = %d, want 1", got.AvoidedLossCount)
	}
	if got.RiskVetoHelpedCount != 1 {
		t.Fatalf("risk veto helped count = %d, want 1", got.RiskVetoHelpedCount)
	}
	if got.PaperSetupWorkedCount != 1 {
		t.Fatalf("paper setup worked count = %d, want 1", got.PaperSetupWorkedCount)
	}
	if got.LessonsGenerated != 4 {
		t.Fatalf("lessons generated = %d, want 4", got.LessonsGenerated)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("errors = %v, want none", got.Errors)
	}
}

func TestRunAppliesFiltersAndCanExcludeDefaultIncludedRecords(t *testing.T) {
	got := Run(ReplayInput{
		ReplayID:              "replay_filtered",
		EventTypeFilter:       []string{"macro"},
		IncludeNoTrades:       Bool(false),
		IncludeWatch:          Bool(false),
		IncludeRejectedByRisk: Bool(false),
		Records: []Record{
			{RecordID: "rec_no_trade", EventType: "macro", FinalDecision: core.DecisionNoTrade, CreatedAt: fixedReplayNow()},
			{RecordID: "rec_watch", EventType: "macro", FinalDecision: core.DecisionWatch, CreatedAt: fixedReplayNow()},
			{RecordID: "rec_candidate", EventType: "macro", FinalDecision: core.DecisionTradeCandidate, CreatedAt: fixedReplayNow()},
			{RecordID: "rec_other", EventType: "earnings", FinalDecision: core.DecisionTradeCandidate, CreatedAt: fixedReplayNow()},
		},
		CreatedAt: fixedReplayNow(),
	})

	if got.RecordsProcessed != 1 {
		t.Fatalf("records processed = %d, want 1", got.RecordsProcessed)
	}
	if got.TradeCandidateCount != 1 {
		t.Fatalf("trade candidate count = %d, want 1", got.TradeCandidateCount)
	}
}

func fixedReplayNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}
