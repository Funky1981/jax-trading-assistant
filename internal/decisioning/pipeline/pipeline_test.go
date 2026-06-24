package pipeline

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/review"
	"jax-trading-assistant/internal/decisioning/risk"
)

func TestRunPipelineTable(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name                   string
		input                  Input
		wantStatus             FinalStatus
		wantPaperTicket        bool
		wantResearchResult     bool
		wantWarnings           []string
		wantForbidden          []string
		wantReviewSchedule     bool
		wantHumanApproval      bool
		wantPaperOnly          bool
		wantLiveTradingBlocked bool
	}{
		{
			name:                   "FTSE oil labour conflict records no trade and review",
			input:                  ftseConflictInput(now),
			wantStatus:             StatusNoTradeRecorded,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "valid swing candidate without research needs research",
			input:                  candidateInput(now, nil, risk.AccountModePaper, true),
			wantStatus:             StatusTradeCandidateNeedsResearch,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "valid swing candidate with promising research prepares pending paper review",
			input:                  candidateInput(now, promisingEvidence(), risk.AccountModePaper, true),
			wantStatus:             StatusTradeCandidateReadyForPaperReview,
			wantPaperTicket:        true,
			wantResearchResult:     true,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "risk rejected candidate records risk rejection",
			input:                  riskRejectedInput(now),
			wantStatus:             StatusTradeCandidateRejectedByRisk,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "watch cannot be upgraded",
			input:                  watchInput(now),
			wantStatus:             StatusWatchRecorded,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "no trade cannot be upgraded",
			input:                  noTradeInput(now),
			wantStatus:             StatusNoTradeRecorded,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "live account mode is blocked",
			input:                  candidateInput(now, promisingEvidence(), risk.AccountModeLive, true),
			wantStatus:             StatusTradeCandidateRejectedByRisk,
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
		{
			name:                   "missing portfolio context warns and avoids paper readiness",
			input:                  candidateInput(now, promisingEvidence(), risk.AccountModePaper, false),
			wantStatus:             StatusPaperReviewBlocked,
			wantWarnings:           []string{"portfolio context is missing"},
			wantForbidden:          mandatoryForbiddenActions(),
			wantReviewSchedule:     true,
			wantHumanApproval:      true,
			wantPaperOnly:          true,
			wantLiveTradingBlocked: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Run(tc.input)

			if got.FinalStatus != tc.wantStatus {
				t.Fatalf("final status = %s, want %s; validation errors=%v warnings=%v", got.FinalStatus, tc.wantStatus, got.ValidationErrors, got.ValidationWarnings)
			}
			if (got.PaperTicketResult != nil) != tc.wantPaperTicket {
				t.Fatalf("paper ticket present = %v, want %v", got.PaperTicketResult != nil, tc.wantPaperTicket)
			}
			if tc.wantPaperTicket && got.PaperTicketResult.HumanApprovalStatus != paper.ApprovalPendingReview {
				t.Fatalf("paper ticket status = %s, want %s", got.PaperTicketResult.HumanApprovalStatus, paper.ApprovalPendingReview)
			}
			if (got.ResearchEvidenceResult != nil) != tc.wantResearchResult {
				t.Fatalf("research result present = %v, want %v", got.ResearchEvidenceResult != nil, tc.wantResearchResult)
			}
			if (got.ReviewScheduleResult.ScheduleID != "") != tc.wantReviewSchedule {
				t.Fatalf("review schedule present = %v, want %v", got.ReviewScheduleResult.ScheduleID != "", tc.wantReviewSchedule)
			}
			if got.HumanApprovalRequired != tc.wantHumanApproval {
				t.Fatalf("human approval required = %v, want %v", got.HumanApprovalRequired, tc.wantHumanApproval)
			}
			if got.PaperOnly != tc.wantPaperOnly {
				t.Fatalf("paper only = %v, want %v", got.PaperOnly, tc.wantPaperOnly)
			}
			if got.LiveTradingBlocked != tc.wantLiveTradingBlocked {
				t.Fatalf("live trading blocked = %v, want %v", got.LiveTradingBlocked, tc.wantLiveTradingBlocked)
			}
			assertContainsAll(t, got.ForbiddenActions, tc.wantForbidden)
			assertTextContainsAll(t, got.ValidationWarnings, tc.wantWarnings)
			assertDoesNotContain(t, got.AllowedActions, core.ActionExecuteTrade)
			assertDoesNotContain(t, got.AllowedActions, core.ActionCreateLiveOrder)
			assertDoesNotContain(t, got.AllowedActions, core.ActionAutoApprove)
		})
	}
}

func ftseConflictInput(now time.Time) Input {
	return Input{
		Event: core.Event{
			EventID:            "evt_2026_06_18_ftse_oil_labour",
			ReceivedAt:         now,
			Headline:           "FTSE falls as oil slump outweighs strong UK labour data",
			Summary:            "FTSE weakness appears driven by oil weakness while labour data is stronger than expected and the BoE decision is pending.",
			PrimaryDrivers:     []string{"oil_price_drop"},
			ConflictingDrivers: []string{"strong_uk_labour_data", "boe_policy_uncertainty"},
			AffectedAssets:     []string{"FTSE100", "BP", "SHEL", "GBP", "UK_GILTS"},
			UncertaintyNotes:   []string{"BoE decision pending"},
		},
		MarketContext: marketContext(),
		PortfolioContext: &risk.PortfolioContext{
			AccountEquity:      100000,
			CashAvailable:      50000,
			MaxRiskPerTradePct: 0.01,
			AsOf:               now,
		},
		AccountMode: risk.AccountModePaper,
		CurrentTime: now,
		DecisionScores: core.Scores{
			ClarityScore:  0.45,
			EdgeScore:     0.25,
			ConflictScore: 0.85,
			RiskScore:     0.60,
		},
		Swing: SwingInput{
			Scores: core.Scores{
				ClarityScore:      0.45,
				EdgeScore:         0.25,
				ConflictScore:     0.85,
				RiskScore:         0.60,
				ConfirmationScore: 0.20,
				TimingScore:       0.40,
			},
			Catalyst:            "Oil drop conflicts with strong labour data and pending BoE policy risk.",
			SetupFamily:         swing.IndexHeavyweightDistortionWatch,
			RiskRewardRatio:     0,
			UnresolvedEventRisk: []string{"BoE decision pending"},
		},
	}
}

func candidateInput(now time.Time, evidence *research.BacktestEvidence, mode risk.AccountMode, withPortfolio bool) Input {
	input := Input{
		Event: core.Event{
			EventID:        "evt_valid_swing_candidate",
			ReceivedAt:     now,
			Headline:       "Energy stock confirms commodity-linked dislocation",
			Summary:        "A commodity-linked equity has not fully repriced after a confirmed commodity move.",
			PrimaryDrivers: []string{"oil_price_rise", "sector_relative_strength"},
			AffectedAssets: []string{"SHEL"},
		},
		MarketContext:    marketContext(),
		AccountMode:      mode,
		CurrentTime:      now,
		ResearchEvidence: evidence,
		DecisionScores: core.Scores{
			ClarityScore:  0.82,
			EdgeScore:     0.82,
			ConflictScore: 0.20,
			RiskScore:     0.35,
		},
		Swing: SwingInput{
			Scores: core.Scores{
				ClarityScore:      0.84,
				EdgeScore:         0.86,
				ConflictScore:     0.18,
				RiskScore:         0.35,
				ConfirmationScore: 0.82,
				TimingScore:       0.78,
			},
			Catalyst:                  "Confirmed commodity move with lagging equity response.",
			SetupFamily:               swing.CommodityLinkedEquityDislocation,
			RequiredConfirmations:     []string{"commodity move remains confirmed", "equity holds support"},
			PresentConfirmations:      []string{"volume confirmed", "sector relative strength confirmed"},
			InvalidationConditions:    []string{"daily close below support"},
			RiskRewardRatio:           2.6,
			Asset:                     "SHEL",
			ProposedEntry:             2500,
			ProposedEntryHigh:         2530,
			ProposedStop:              2400,
			ProposedTarget:            2760,
			MaxPaperPositionSize:      25,
			MarketSectorAlignmentNote: "energy sector confirmation present",
		},
	}
	if withPortfolio {
		input.PortfolioContext = &risk.PortfolioContext{
			AccountEquity:      100000,
			CashAvailable:      50000,
			MaxRiskPerTradePct: 0.01,
			AsOf:               now,
		}
	}
	return input
}

func riskRejectedInput(now time.Time) Input {
	input := candidateInput(now, promisingEvidence(), risk.AccountModePaper, true)
	input.Event.EventID = "evt_risk_rejected_candidate"
	input.Swing.LiquiditySpreadNotes = []string{"poor liquidity and wide spread"}
	return input
}

func watchInput(now time.Time) Input {
	input := candidateInput(now, promisingEvidence(), risk.AccountModePaper, true)
	input.Event.EventID = "evt_watch_cannot_upgrade"
	input.Swing.Scores.EdgeScore = 0.62
	input.Swing.Scores.ConfirmationScore = 0.30
	input.Swing.MissingConfirmations = []string{"volume confirmation missing"}
	input.Swing.PresentConfirmations = nil
	return input
}

func noTradeInput(now time.Time) Input {
	input := ftseConflictInput(now)
	input.Event.EventID = "evt_no_trade_cannot_upgrade"
	return input
}

func promisingEvidence() *research.BacktestEvidence {
	return &research.BacktestEvidence{
		HypothesisID:       "hyp_commodity_dislocation",
		SetupFamily:        string(swing.CommodityLinkedEquityDislocation),
		DatasetID:          "commodity_equity_daily_2018_2025_v1",
		DatasetHash:        "sha256:phase8",
		DateRange:          research.DateRange{Start: "2018-01-01", End: "2025-12-31"},
		InstrumentUniverse: []string{"UK_large_cap_energy"},
		Benchmark:          "FTSE100",
		Assumptions: research.BacktestAssumptions{
			Execution:       "next_day_open",
			PositionSizing:  "fixed_fractional",
			MaxRiskPerTrade: 0.01,
		},
		SlippageModel:     research.SlippageModel{Type: "bps", Value: 10},
		FeesModel:         research.FeesModel{Commission: "broker_model", SpreadAssumption: "included"},
		InSamplePeriod:    research.DateRange{Start: "2018-01-01", End: "2022-12-31"},
		OutOfSamplePeriod: research.DateRange{Start: "2023-01-01", End: "2025-12-31"},
		DrawdownMetrics:   research.DrawdownMetrics{MaxDrawdown: 0.12, AverageDrawdown: 0.04},
		Expectancy:        0.18,
		SampleSize:        120,
		FailureModes:      []string{"commodity reversal", "sector risk-off"},
		RiskRules:         []string{"risk/reward >= 2:1", "stop below support"},
		PromotionDecision: research.PromotionBacktestedPromising,
	}
}

func marketContext() map[string]string {
	return map[string]string{
		"market_regime":   "risk_neutral",
		"calendar_status": "no_same_day_policy_event_for_candidate",
	}
}

func assertContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	for _, expected := range want {
		found := false
		for _, actual := range got {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", expected, got)
		}
	}
}

func assertTextContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	for _, expected := range want {
		found := false
		for _, actual := range got {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing text %q in %v", expected, got)
		}
	}
}

func assertDoesNotContain(t *testing.T, got []string, forbidden string) {
	t.Helper()
	for _, actual := range got {
		if actual == forbidden {
			t.Fatalf("unexpected %q in %v", forbidden, got)
		}
	}
}

var _ review.ReviewSchedule
