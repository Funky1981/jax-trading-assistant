package paper

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/risk"
)

func TestValidateTicketRequestRules(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*TicketRequest)
		wantValid  bool
		wantCreate bool
	}{
		{
			name:       "valid risk-approved trade candidate creates pending ticket",
			wantValid:  true,
			wantCreate: true,
		},
		{
			name: "no trade cannot create paper ticket",
			mutate: func(req *TicketRequest) {
				req.SourceDecision.Decision = core.DecisionNoTrade
				req.RiskAssessment.OriginalDecision = core.DecisionNoTrade
				req.RiskAssessment.FinalDecision = core.DecisionNoTrade
			},
			wantValid:  false,
			wantCreate: false,
		},
		{
			name: "watch cannot create paper ticket",
			mutate: func(req *TicketRequest) {
				req.SourceDecision.Decision = core.DecisionWatch
				req.RiskAssessment.OriginalDecision = core.DecisionWatch
				req.RiskAssessment.FinalDecision = core.DecisionWatch
			},
			wantValid:  false,
			wantCreate: false,
		},
		{
			name: "setup forming cannot create paper ticket",
			mutate: func(req *TicketRequest) {
				req.SourceDecision.Decision = core.DecisionSetupForming
				req.RiskAssessment.OriginalDecision = core.DecisionSetupForming
				req.RiskAssessment.FinalDecision = core.DecisionSetupForming
			},
			wantValid:  false,
			wantCreate: false,
		},
		{
			name: "risk rejected candidate cannot create paper ticket",
			mutate: func(req *TicketRequest) {
				req.RiskAssessment.RiskDecision = risk.RiskDecisionRejectedByRisk
				req.RiskAssessment.FinalDecision = core.DecisionRejectedByRisk
			},
			wantValid:  false,
			wantCreate: false,
		},
		{
			name: "missing invalidation rejected",
			mutate: func(req *TicketRequest) {
				req.InvalidationConditions = nil
			},
			wantValid:  false,
			wantCreate: false,
		},
		{
			name: "poor risk reward rejected",
			mutate: func(req *TicketRequest) {
				req.RiskReward = 1.6
			},
			wantValid:  false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validTicketRequest()
			if tt.mutate != nil {
				tt.mutate(&req)
			}

			ticket, got := NewTicket(req)
			if got.IsValid != tt.wantValid {
				t.Fatalf("is_valid = %v, want %v; errors=%v", got.IsValid, tt.wantValid, got.ValidationErrors)
			}
			if got.CanCreateTicket != tt.wantCreate {
				t.Fatalf("can_create_ticket = %v, want %v", got.CanCreateTicket, tt.wantCreate)
			}
			if tt.wantCreate {
				if ticket.HumanApprovalStatus != ApprovalPendingReview {
					t.Fatalf("status = %s, want %s", ticket.HumanApprovalStatus, ApprovalPendingReview)
				}
				if !ticket.PaperOnly || !ticket.LiveTradingBlocked {
					t.Fatal("ticket must preserve paper-only and live-trading-blocked boundaries")
				}
				assertPaperContainsAll(t, ticket.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
			}
		})
	}
}

func TestApproveForPaperRules(t *testing.T) {
	validTicket, validation := NewTicket(validTicketRequest())
	if !validation.CanCreateTicket {
		t.Fatalf("base ticket invalid: %v", validation.ValidationErrors)
	}

	tests := []struct {
		name        string
		ticket      PaperTicket
		mutate      func(*ApprovalRequest)
		wantApprove bool
		wantStatus  ApprovalStatus
	}{
		{
			name:        "valid pending ticket can be approved for paper",
			ticket:      validTicket,
			wantApprove: true,
			wantStatus:  ApprovalApprovedForPaper,
		},
		{
			name: "expired ticket cannot be approved",
			ticket: func() PaperTicket {
				ticket := validTicket
				ticket.ExpiresAt = fixedNow().Add(-time.Hour)
				return ticket
			}(),
			wantApprove: false,
			wantStatus:  ApprovalPendingReview,
		},
		{
			name: "rejected by user ticket cannot be approved",
			ticket: func() PaperTicket {
				ticket := validTicket
				ticket.HumanApprovalStatus = ApprovalRejectedByUser
				ticket.LifecycleState = LifecycleRejectedByUser
				return ticket
			}(),
			wantApprove: false,
			wantStatus:  ApprovalRejectedByUser,
		},
		{
			name:   "auto approval rejected",
			ticket: validTicket,
			mutate: func(req *ApprovalRequest) {
				req.ExplicitHumanApproval = false
				req.AutomaticApproval = true
			},
			wantApprove: false,
			wantStatus:  ApprovalPendingReview,
		},
		{
			name: "deferred ticket requires new review",
			ticket: func() PaperTicket {
				ticket := validTicket
				ticket.HumanApprovalStatus = ApprovalDeferred
				ticket.LifecycleState = LifecycleDeferred
				return ticket
			}(),
			wantApprove: false,
			wantStatus:  ApprovalDeferred,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ApprovalRequest{
				Ticket:                tt.ticket,
				ApprovedBy:            "human-reviewer",
				ApprovedAt:            fixedNow(),
				Now:                   fixedNow(),
				ExplicitHumanApproval: true,
			}
			if tt.mutate != nil {
				tt.mutate(&req)
			}

			got := ApproveForPaper(req)
			if got.CanApproveForPaper != tt.wantApprove {
				t.Fatalf("can approve = %v, want %v; errors=%v", got.CanApproveForPaper, tt.wantApprove, got.Validation.ValidationErrors)
			}
			if got.HumanApprovalStatus != tt.wantStatus {
				t.Fatalf("status = %s, want %s", got.HumanApprovalStatus, tt.wantStatus)
			}
		})
	}
}

func validTicketRequest() TicketRequest {
	now := fixedNow()
	return TicketRequest{
		PaperTicketID: "pt_dec_paper_001",
		SourceDecision: core.Decision{
			DecisionID:             "dec_paper_001",
			EventID:                "evt_paper_001",
			Brain:                  "SWING_BRAIN_V1",
			Decision:               core.DecisionTradeCandidate,
			ConfidenceScore:        0.84,
			ClarityScore:           0.82,
			EdgeScore:              0.81,
			ConflictScore:          0.18,
			RiskScore:              0.32,
			PrimaryReason:          "Swing candidate has evidence, confirmation, invalidation, and valid risk/reward.",
			SupportingReasons:      []string{"risk veto passed", "research evidence is paper-ready"},
			RequiredConfirmations:  []string{"price holds support", "sector confirms"},
			InvalidationConditions: []string{"BP closes below 470p"},
			AllowedActions:         []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper},
			ForbiddenActions:       []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
			ReviewAfter:            []string{core.ReviewAfter1Day, core.ReviewAfter1Week},
		},
		RiskAssessment: risk.RiskAssessment{
			RiskDecision:          risk.RiskDecisionPass,
			OriginalDecision:      core.DecisionTradeCandidate,
			FinalDecision:         core.DecisionTradeCandidate,
			RequiredActions:       []string{risk.RequiredActionHumanApprovalRequired},
			ForbiddenActions:      []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
			AllowedActions:        []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper},
			MaxPositionRisk:       1000,
			MaxPositionSize:       33.33,
			RequiresHumanApproval: true,
			PaperOnly:             true,
			LiveTradingBlocked:    true,
			ReviewAfter:           []string{core.ReviewAfter1Day, core.ReviewAfter1Week},
		},
		ResearchEvidenceSummary: ResearchEvidenceSummary{
			HypothesisID:      "hyp_swing_001",
			SetupFamily:       "commodity_linked_equity_dislocation",
			PromotionDecision: research.PromotionPaperReady,
			Summary:           "Complete evidence passed Phase 5 validation.",
		},
		Asset:                         "BP",
		SetupFamily:                   "commodity_linked_equity_dislocation",
		ProposedEntryZone:             EntryZone{Low: 500, High: 505},
		ProposedStop:                  470,
		ProposedTarget:                575,
		RiskReward:                    2.5,
		MaxPaperPositionSize:          33.33,
		CreatedAt:                     now,
		ExpiresAt:                     now.Add(24 * time.Hour),
		RequiredConfirmations:         []string{"price holds support", "sector confirms"},
		InvalidationConditions:        []string{"BP closes below 470p"},
		ExplicitHumanApprovalRequired: true,
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertPaperContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q in %v", item, got)
		}
	}
}
