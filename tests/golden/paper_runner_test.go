package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
)

type paperGoldenCase struct {
	Name                        string                `json:"name"`
	Mode                        string                `json:"mode"`
	TicketRequest               paper.TicketRequest   `json:"ticket_request"`
	ApprovalRequest             paper.ApprovalRequest `json:"approval_request"`
	ExpectedValid               bool                  `json:"expected_valid"`
	ExpectedCanCreateTicket     bool                  `json:"expected_can_create_ticket"`
	ExpectedCanApproveForPaper  bool                  `json:"expected_can_approve_for_paper"`
	ExpectedHumanApprovalStatus paper.ApprovalStatus  `json:"expected_human_approval_status"`
	ExpectedPaperOnly           bool                  `json:"expected_paper_only"`
	ExpectedLiveTradingBlocked  bool                  `json:"expected_live_trading_blocked"`
	ExpectedForbiddenActions    []string              `json:"expected_forbidden_actions"`
	ExpectedErrorsContain       []string              `json:"expected_errors_contain"`
}

func TestPaperGoldenCases(t *testing.T) {
	files := []string{
		"valid_trade_candidate_creates_pending_ticket.json",
		"no_trade_cannot_create_ticket.json",
		"watch_cannot_create_ticket.json",
		"risk_rejected_candidate_cannot_create_ticket.json",
		"missing_invalidation_rejected.json",
		"poor_risk_reward_rejected.json",
		"expired_ticket_cannot_be_approved.json",
		"rejected_by_user_ticket_cannot_be_approved.json",
		"valid_pending_ticket_can_be_approved.json",
		"auto_approval_rejected.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc paperGoldenCase
			readJSON(t, filepath.Join("paper", file), &tc)

			switch tc.Mode {
			case "create":
				ticket, got := paper.NewTicket(tc.TicketRequest)
				assertPaperValidation(t, tc, got)
				if tc.ExpectedCanCreateTicket {
					if ticket.HumanApprovalStatus != tc.ExpectedHumanApprovalStatus {
						t.Fatalf("%s ticket status = %s, want %s", tc.Name, ticket.HumanApprovalStatus, tc.ExpectedHumanApprovalStatus)
					}
					if ticket.PaperOnly != tc.ExpectedPaperOnly {
						t.Fatalf("%s paper_only = %v, want %v", tc.Name, ticket.PaperOnly, tc.ExpectedPaperOnly)
					}
					if ticket.LiveTradingBlocked != tc.ExpectedLiveTradingBlocked {
						t.Fatalf("%s live_trading_blocked = %v, want %v", tc.Name, ticket.LiveTradingBlocked, tc.ExpectedLiveTradingBlocked)
					}
					assertContainsAll(t, ticket.ForbiddenActions, tc.ExpectedForbiddenActions)
				}
			case "approve":
				got := paper.ApproveForPaper(tc.ApprovalRequest)
				assertPaperValidation(t, tc, got.Validation)
				if got.HumanApprovalStatus != tc.ExpectedHumanApprovalStatus {
					t.Fatalf("%s approval status = %s, want %s", tc.Name, got.HumanApprovalStatus, tc.ExpectedHumanApprovalStatus)
				}
				assertContainsAll(t, got.Validation.ForbiddenActions, tc.ExpectedForbiddenActions)
			default:
				t.Fatalf("unsupported paper golden mode %q", tc.Mode)
			}
		})
	}
}

func assertPaperValidation(t *testing.T, tc paperGoldenCase, got paper.ValidationResult) {
	t.Helper()
	if got.IsValid != tc.ExpectedValid {
		t.Fatalf("%s is_valid = %v, want %v; errors=%v", tc.Name, got.IsValid, tc.ExpectedValid, got.ValidationErrors)
	}
	if got.CanCreateTicket != tc.ExpectedCanCreateTicket {
		t.Fatalf("%s can_create_ticket = %v, want %v", tc.Name, got.CanCreateTicket, tc.ExpectedCanCreateTicket)
	}
	if got.CanApproveForPaper != tc.ExpectedCanApproveForPaper {
		t.Fatalf("%s can_approve_for_paper = %v, want %v", tc.Name, got.CanApproveForPaper, tc.ExpectedCanApproveForPaper)
	}
	if got.RequiredHumanApproval != true {
		t.Fatalf("%s must require human approval", tc.Name)
	}
	if got.PaperOnly != true {
		t.Fatalf("%s must remain paper-only", tc.Name)
	}
	if got.LiveTradingBlocked != true {
		t.Fatalf("%s must block live trading", tc.Name)
	}
	assertTextContainsAll(t, got.ValidationErrors, tc.ExpectedErrorsContain)
	assertContainsAll(t, got.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
}
