package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/risk"
)

type riskGoldenCase struct {
	Name                       string                `json:"name"`
	SwingDecision              swing.Decision        `json:"swing_decision"`
	Asset                      string                `json:"asset"`
	SetupFamily                swing.SetupFamily     `json:"setup_family"`
	ProposedEntry              float64               `json:"proposed_entry"`
	ProposedStop               float64               `json:"proposed_stop"`
	ProposedTarget             float64               `json:"proposed_target"`
	RiskRewardRatio            float64               `json:"risk_reward_ratio"`
	AccountMode                risk.AccountMode      `json:"account_mode"`
	Portfolio                  risk.PortfolioContext `json:"portfolio"`
	CurrentExposure            risk.Exposure         `json:"current_exposure"`
	SectorExposure             risk.Exposure         `json:"sector_exposure"`
	CorrelatedExposure         risk.Exposure         `json:"correlated_exposure"`
	LiquiditySpreadNotes       []string              `json:"liquidity_spread_notes"`
	UnresolvedEventRisk        []string              `json:"unresolved_event_risk"`
	UpcomingMajorEventRisk     []string              `json:"upcoming_major_event_risk"`
	BrokerExecutionRequested   bool                  `json:"broker_execution_requested"`
	LiveOrderRequested         bool                  `json:"live_order_requested"`
	ExpectedRiskDecision       risk.RiskDecision     `json:"expected_risk_decision"`
	ExpectedFinalDecision      core.DecisionValue    `json:"expected_final_decision"`
	ExpectedForbiddenActions   []string              `json:"expected_forbidden_actions"`
	ExpectedRequiredActions    []string              `json:"expected_required_actions"`
	ExpectedHumanApproval      bool                  `json:"expected_requires_human_approval"`
	ExpectedPaperOnly          bool                  `json:"expected_paper_only"`
	ExpectedLiveTradingBlocked bool                  `json:"expected_live_trading_blocked"`
}

func TestRiskGoldenCases(t *testing.T) {
	files := []string{
		"valid_swing_candidate_passes.json",
		"poor_risk_reward_rejected.json",
		"missing_stop_rejected.json",
		"live_account_rejected.json",
		"high_sector_concentration_downgraded.json",
		"correlated_exposure_downgraded.json",
		"watch_cannot_upgrade.json",
		"no_trade_cannot_upgrade.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc riskGoldenCase
			readJSON(t, filepath.Join("risk", file), &tc)

			got := risk.Assess(risk.AssessmentInput{
				SwingDecision:            tc.SwingDecision,
				Asset:                    tc.Asset,
				SetupFamily:              tc.SetupFamily,
				ProposedEntry:            tc.ProposedEntry,
				ProposedStop:             tc.ProposedStop,
				ProposedTarget:           tc.ProposedTarget,
				RiskRewardRatio:          tc.RiskRewardRatio,
				AccountMode:              tc.AccountMode,
				Portfolio:                tc.Portfolio,
				CurrentExposure:          tc.CurrentExposure,
				SectorExposure:           tc.SectorExposure,
				CorrelatedExposure:       tc.CorrelatedExposure,
				LiquiditySpreadNotes:     tc.LiquiditySpreadNotes,
				UnresolvedEventRisk:      tc.UnresolvedEventRisk,
				UpcomingMajorEventRisk:   tc.UpcomingMajorEventRisk,
				BrokerExecutionRequested: tc.BrokerExecutionRequested,
				LiveOrderRequested:       tc.LiveOrderRequested,
			})

			if got.RiskDecision != tc.ExpectedRiskDecision {
				t.Fatalf("%s risk decision = %s, want %s", tc.Name, got.RiskDecision, tc.ExpectedRiskDecision)
			}
			if got.FinalDecision != tc.ExpectedFinalDecision {
				t.Fatalf("%s final decision = %s, want %s", tc.Name, got.FinalDecision, tc.ExpectedFinalDecision)
			}
			if got.RequiresHumanApproval != tc.ExpectedHumanApproval {
				t.Fatalf("%s requires human approval = %v, want %v", tc.Name, got.RequiresHumanApproval, tc.ExpectedHumanApproval)
			}
			if got.PaperOnly != tc.ExpectedPaperOnly {
				t.Fatalf("%s paper only = %v, want %v", tc.Name, got.PaperOnly, tc.ExpectedPaperOnly)
			}
			if got.LiveTradingBlocked != tc.ExpectedLiveTradingBlocked {
				t.Fatalf("%s live trading blocked = %v, want %v", tc.Name, got.LiveTradingBlocked, tc.ExpectedLiveTradingBlocked)
			}
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			assertContainsAll(t, got.RequiredActions, tc.ExpectedRequiredActions)
		})
	}
}
