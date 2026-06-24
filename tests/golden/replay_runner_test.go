package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/replay"
)

type replayGoldenCase struct {
	Name                        string             `json:"name"`
	Input                       replay.ReplayInput `json:"input"`
	ExpectedRecordsProcessed    int                `json:"expected_records_processed"`
	ExpectedNoTradeCount        int                `json:"expected_no_trade_count"`
	ExpectedWatchCount          int                `json:"expected_watch_count"`
	ExpectedRejectedByRiskCount int                `json:"expected_rejected_by_risk_count"`
	ExpectedPaperApprovedCount  int                `json:"expected_paper_approved_count"`
	ExpectedCorrectNoTradeCount int                `json:"expected_correct_no_trade_count"`
	ExpectedMissedOpportunity   int                `json:"expected_missed_opportunity_count"`
	ExpectedAvoidedLossCount    int                `json:"expected_avoided_loss_count"`
	ExpectedRiskVetoHelped      int                `json:"expected_risk_veto_helped_count"`
	ExpectedRiskVetoTooStrict   int                `json:"expected_risk_veto_too_strict_count"`
	ExpectedPaperSetupWorked    int                `json:"expected_paper_setup_worked_count"`
	ExpectedPaperSetupFailed    int                `json:"expected_paper_setup_failed_count"`
	ExpectedLessonsGenerated    int                `json:"expected_lessons_generated"`
}

func TestReplayGoldenCases(t *testing.T) {
	files := []string{"phase_10_replay_summary.json"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc replayGoldenCase
			readJSON(t, filepath.Join("replay", file), &tc)

			got := replay.Run(tc.Input)
			if got.RecordsProcessed != tc.ExpectedRecordsProcessed {
				t.Fatalf("%s records processed = %d, want %d", tc.Name, got.RecordsProcessed, tc.ExpectedRecordsProcessed)
			}
			if got.NoTradeCount != tc.ExpectedNoTradeCount {
				t.Fatalf("%s no trade count = %d, want %d", tc.Name, got.NoTradeCount, tc.ExpectedNoTradeCount)
			}
			if got.WatchCount != tc.ExpectedWatchCount {
				t.Fatalf("%s watch count = %d, want %d", tc.Name, got.WatchCount, tc.ExpectedWatchCount)
			}
			if got.RejectedByRiskCount != tc.ExpectedRejectedByRiskCount {
				t.Fatalf("%s rejected by risk count = %d, want %d", tc.Name, got.RejectedByRiskCount, tc.ExpectedRejectedByRiskCount)
			}
			if got.PaperApprovedCount != tc.ExpectedPaperApprovedCount {
				t.Fatalf("%s paper approved count = %d, want %d", tc.Name, got.PaperApprovedCount, tc.ExpectedPaperApprovedCount)
			}
			if got.CorrectNoTradeCount != tc.ExpectedCorrectNoTradeCount {
				t.Fatalf("%s correct no trade count = %d, want %d", tc.Name, got.CorrectNoTradeCount, tc.ExpectedCorrectNoTradeCount)
			}
			if got.MissedOpportunityCount != tc.ExpectedMissedOpportunity {
				t.Fatalf("%s missed opportunity count = %d, want %d", tc.Name, got.MissedOpportunityCount, tc.ExpectedMissedOpportunity)
			}
			if got.AvoidedLossCount != tc.ExpectedAvoidedLossCount {
				t.Fatalf("%s avoided loss count = %d, want %d", tc.Name, got.AvoidedLossCount, tc.ExpectedAvoidedLossCount)
			}
			if got.RiskVetoHelpedCount != tc.ExpectedRiskVetoHelped {
				t.Fatalf("%s risk veto helped count = %d, want %d", tc.Name, got.RiskVetoHelpedCount, tc.ExpectedRiskVetoHelped)
			}
			if got.RiskVetoTooStrictCount != tc.ExpectedRiskVetoTooStrict {
				t.Fatalf("%s risk veto too strict count = %d, want %d", tc.Name, got.RiskVetoTooStrictCount, tc.ExpectedRiskVetoTooStrict)
			}
			if got.PaperSetupWorkedCount != tc.ExpectedPaperSetupWorked {
				t.Fatalf("%s paper setup worked count = %d, want %d", tc.Name, got.PaperSetupWorkedCount, tc.ExpectedPaperSetupWorked)
			}
			if got.PaperSetupFailedCount != tc.ExpectedPaperSetupFailed {
				t.Fatalf("%s paper setup failed count = %d, want %d", tc.Name, got.PaperSetupFailedCount, tc.ExpectedPaperSetupFailed)
			}
			if got.LessonsGenerated != tc.ExpectedLessonsGenerated {
				t.Fatalf("%s lessons generated = %d, want %d", tc.Name, got.LessonsGenerated, tc.ExpectedLessonsGenerated)
			}
		})
	}
}
