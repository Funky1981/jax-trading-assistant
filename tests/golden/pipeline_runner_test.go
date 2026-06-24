package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
	"jax-trading-assistant/internal/decisioning/pipeline"
)

type pipelineGoldenCase struct {
	Name                       string                 `json:"name"`
	Input                      pipeline.Input         `json:"input"`
	ExpectedFinalStatus        pipeline.FinalStatus   `json:"expected_final_status"`
	ExpectedFinalDecision      core.DecisionValue     `json:"expected_final_decision"`
	ExpectedPaperTicket        bool                   `json:"expected_paper_ticket"`
	ExpectedPaperTicketStatus  paper.ApprovalStatus   `json:"expected_paper_ticket_status"`
	ExpectedReviewSchedule     bool                   `json:"expected_review_schedule"`
	ExpectedForbiddenActions   []string               `json:"expected_forbidden_actions"`
	ExpectedWarningsContain    []string               `json:"expected_warnings_contain"`
	ExpectedHumanApproval      bool                   `json:"expected_human_approval"`
	ExpectedPaperOnly          bool                   `json:"expected_paper_only"`
	ExpectedLiveTradingBlocked bool                   `json:"expected_live_trading_blocked"`
	NeverFinalStatus           []pipeline.FinalStatus `json:"never_final_status"`
}

func TestPipelineGoldenCases(t *testing.T) {
	files := []string{
		"ftse_oil_labour_full_pipeline.json",
		"valid_candidate_missing_research.json",
		"valid_candidate_promising_research.json",
		"risk_rejected_candidate.json",
		"watch_cannot_upgrade.json",
		"no_trade_cannot_upgrade.json",
		"live_account_mode_blocked.json",
		"missing_portfolio_context_warning.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc pipelineGoldenCase
			readJSON(t, filepath.Join("pipeline", file), &tc)

			got := pipeline.Run(tc.Input)
			if got.FinalStatus != tc.ExpectedFinalStatus {
				t.Fatalf("%s final status = %s, want %s; errors=%v warnings=%v", tc.Name, got.FinalStatus, tc.ExpectedFinalStatus, got.ValidationErrors, got.ValidationWarnings)
			}
			for _, forbiddenStatus := range tc.NeverFinalStatus {
				if got.FinalStatus == forbiddenStatus {
					t.Fatalf("%s final status must never be %s", tc.Name, forbiddenStatus)
				}
			}
			if got.FinalDecision != tc.ExpectedFinalDecision {
				t.Fatalf("%s final decision = %s, want %s", tc.Name, got.FinalDecision, tc.ExpectedFinalDecision)
			}
			if (got.PaperTicketResult != nil) != tc.ExpectedPaperTicket {
				t.Fatalf("%s paper ticket present = %v, want %v", tc.Name, got.PaperTicketResult != nil, tc.ExpectedPaperTicket)
			}
			if tc.ExpectedPaperTicket && got.PaperTicketResult.HumanApprovalStatus != tc.ExpectedPaperTicketStatus {
				t.Fatalf("%s paper ticket status = %s, want %s", tc.Name, got.PaperTicketResult.HumanApprovalStatus, tc.ExpectedPaperTicketStatus)
			}
			if (got.ReviewScheduleResult.ScheduleID != "") != tc.ExpectedReviewSchedule {
				t.Fatalf("%s review schedule present = %v, want %v", tc.Name, got.ReviewScheduleResult.ScheduleID != "", tc.ExpectedReviewSchedule)
			}
			if got.HumanApprovalRequired != tc.ExpectedHumanApproval {
				t.Fatalf("%s human approval required = %v, want %v", tc.Name, got.HumanApprovalRequired, tc.ExpectedHumanApproval)
			}
			if got.PaperOnly != tc.ExpectedPaperOnly {
				t.Fatalf("%s paper only = %v, want %v", tc.Name, got.PaperOnly, tc.ExpectedPaperOnly)
			}
			if got.LiveTradingBlocked != tc.ExpectedLiveTradingBlocked {
				t.Fatalf("%s live trading blocked = %v, want %v", tc.Name, got.LiveTradingBlocked, tc.ExpectedLiveTradingBlocked)
			}
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			assertTextContainsAll(t, got.ValidationWarnings, tc.ExpectedWarningsContain)
			assertActionAbsent(t, got.AllowedActions, core.ActionExecuteTrade)
			assertActionAbsent(t, got.AllowedActions, core.ActionCreateLiveOrder)
			assertActionAbsent(t, got.AllowedActions, core.ActionAutoApprove)
		})
	}
}

func assertActionAbsent(t *testing.T, got []string, forbidden string) {
	t.Helper()
	for _, actual := range got {
		if actual == forbidden {
			t.Fatalf("unexpected action %q in %v", forbidden, got)
		}
	}
}
