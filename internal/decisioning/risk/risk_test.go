package risk

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
)

func TestAssessRiskVetoRules(t *testing.T) {
	tests := []struct {
		name               string
		input              AssessmentInput
		wantRiskDecision   RiskDecision
		wantFinalDecision  core.DecisionValue
		wantReasonContains string
	}{
		{
			name: "valid swing candidate passes risk and remains paper only",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.ProposedEntry = 500
				input.ProposedStop = 470
				input.ProposedTarget = 575
				input.RiskRewardRatio = 2.5
			}),
			wantRiskDecision:  RiskDecisionPass,
			wantFinalDecision: core.DecisionTradeCandidate,
		},
		{
			name: "poor risk reward is rejected by risk",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.RiskRewardRatio = 1.6
			}),
			wantRiskDecision:   RiskDecisionRejectedByRisk,
			wantFinalDecision:  core.DecisionRejectedByRisk,
			wantReasonContains: "risk/reward",
		},
		{
			name: "missing stop or invalidation is rejected by risk",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.ProposedStop = 0
				input.SwingDecision.InvalidationConditions = nil
			}),
			wantRiskDecision:   RiskDecisionRejectedByRisk,
			wantFinalDecision:  core.DecisionRejectedByRisk,
			wantReasonContains: "stop or invalidation",
		},
		{
			name: "live account mode is rejected",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.AccountMode = AccountModeLive
			}),
			wantRiskDecision:   RiskDecisionRejectedByRisk,
			wantFinalDecision:  core.DecisionRejectedByRisk,
			wantReasonContains: "live trading is out of scope",
		},
		{
			name: "high sector concentration downgrades trade candidate to watch",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.SectorExposure.PercentOfEquity = 0.31
			}),
			wantRiskDecision:   RiskDecisionDowngradeToWatch,
			wantFinalDecision:  core.DecisionWatch,
			wantReasonContains: "sector exposure",
		},
		{
			name: "correlated exposure downgrades trade candidate to watch",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.CorrelatedExposure.PercentOfEquity = 0.36
			}),
			wantRiskDecision:   RiskDecisionDowngradeToWatch,
			wantFinalDecision:  core.DecisionWatch,
			wantReasonContains: "correlated exposure",
		},
		{
			name: "existing watch cannot be upgraded",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.SwingDecision.Decision.Decision = core.DecisionWatch
			}),
			wantRiskDecision:  RiskDecisionPass,
			wantFinalDecision: core.DecisionWatch,
		},
		{
			name: "existing no trade cannot be upgraded",
			input: baseRiskInput(func(input *AssessmentInput) {
				input.SwingDecision.Decision.Decision = core.DecisionNoTrade
			}),
			wantRiskDecision:  RiskDecisionPass,
			wantFinalDecision: core.DecisionNoTrade,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Assess(tt.input)

			if got.RiskDecision != tt.wantRiskDecision {
				t.Fatalf("risk decision = %s, want %s", got.RiskDecision, tt.wantRiskDecision)
			}
			if got.FinalDecision != tt.wantFinalDecision {
				t.Fatalf("final decision = %s, want %s", got.FinalDecision, tt.wantFinalDecision)
			}
			if !got.RequiresHumanApproval {
				t.Fatal("risk assessment must require human approval")
			}
			if !got.PaperOnly {
				t.Fatal("risk assessment must preserve paper-only boundary")
			}
			if !got.LiveTradingBlocked {
				t.Fatal("risk assessment must block live trading")
			}
			assertRiskContainsAll(t, got.ForbiddenActions, []string{
				core.ActionExecuteTrade,
				core.ActionCreateLiveOrder,
				core.ActionAutoApprove,
			})
			if got.RiskDecision == RiskDecisionPass && got.FinalDecision == core.DecisionTradeCandidate {
				assertRiskContainsAll(t, got.AllowedActions, []string{core.ActionPreparePaper})
			}
			if tt.wantReasonContains != "" && !riskReasonsContain(got, tt.wantReasonContains) {
				t.Fatalf("expected reason containing %q in veto=%v downgrade=%v", tt.wantReasonContains, got.VetoReasons, got.DowngradeReasons)
			}
		})
	}
}

func baseRiskInput(mutators ...func(*AssessmentInput)) AssessmentInput {
	swingDecision := swing.Decision{
		Decision: core.Decision{
			DecisionID:             "swing_evt_risk",
			EventID:                "evt_risk",
			Brain:                  swing.BrainSwingV1,
			Decision:               core.DecisionTradeCandidate,
			ConfidenceScore:        0.82,
			ClarityScore:           0.84,
			EdgeScore:              0.82,
			ConflictScore:          0.18,
			RiskScore:              0.34,
			PrimaryReason:          "Swing trade candidate meets catalyst, confirmation, invalidation, and risk/reward requirements.",
			RequiredConfirmations:  []string{"price holds support", "energy sector confirms"},
			InvalidationConditions: []string{"BP closes below 470p"},
			AllowedActions:         []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper},
			ForbiddenActions:       []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
			ReviewAfter:            []string{core.ReviewAfter1Day, core.ReviewAfter1Week, core.ReviewAfter1Month},
		},
		SetupFamily: swing.CommodityLinkedEquityDislocation,
		Scores: core.Scores{
			ClarityScore:      0.84,
			EdgeScore:         0.82,
			ConflictScore:     0.18,
			RiskScore:         0.34,
			ConfirmationScore: 0.76,
			TimingScore:       0.70,
		},
	}

	input := AssessmentInput{
		SwingDecision:   swingDecision,
		Asset:           "BP",
		SetupFamily:     swing.CommodityLinkedEquityDislocation,
		ProposedEntry:   500,
		ProposedStop:    470,
		ProposedTarget:  575,
		RiskRewardRatio: 2.5,
		AccountMode:     AccountModePaper,
		Portfolio: PortfolioContext{
			AccountEquity:      100000,
			CashAvailable:      60000,
			MaxRiskPerTradePct: 0.01,
			AsOf:               time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC),
		},
		CurrentExposure: Exposure{PercentOfEquity: 0.10},
		SectorExposure: Exposure{
			Name:            "energy",
			PercentOfEquity: 0.18,
		},
		CorrelatedExposure: Exposure{
			Name:            "oil-linked equities",
			PercentOfEquity: 0.20,
		},
	}
	for _, mutate := range mutators {
		mutate(&input)
	}
	return input
}

func assertRiskContainsAll(t *testing.T, got []string, want []string) {
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

func riskReasonsContain(assessment RiskAssessment, want string) bool {
	for _, reason := range append(assessment.VetoReasons, assessment.DowngradeReasons...) {
		if containsText(reason, want) {
			return true
		}
	}
	return false
}

func containsText(s string, want string) bool {
	for i := 0; i+len(want) <= len(s); i++ {
		if s[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
