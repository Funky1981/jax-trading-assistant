package golden

import (
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
)

type swingGoldenCase struct {
	Name                  string               `json:"name"`
	Event                 core.Event           `json:"event"`
	Scores                core.Scores          `json:"scores"`
	Catalyst              string               `json:"catalyst"`
	SetupFamily           swing.SetupFamily    `json:"setup_family"`
	RequiredConfirmations []string             `json:"required_confirmations"`
	PresentConfirmations  []string             `json:"present_confirmations"`
	MissingConfirmations  []string             `json:"missing_confirmations"`
	Invalidation          []string             `json:"invalidation_conditions"`
	RiskRewardRatio       float64              `json:"risk_reward_ratio"`
	UnresolvedEventRisk   []string             `json:"unresolved_event_risk"`
	MarketAlignment       string               `json:"market_sector_alignment"`
	ExpectedDecision      core.DecisionValue   `json:"expected_decision"`
	ExpectedPrimaryReason string               `json:"expected_primary_reason"`
	ExpectedAllowed       []string             `json:"expected_allowed_actions"`
	ExpectedForbidden     []string             `json:"expected_forbidden_actions"`
	ExpectedReviewAfter   []string             `json:"expected_review_after"`
	NeverDecision         []core.DecisionValue `json:"never_decision"`
}

func TestSwingGoldenCases(t *testing.T) {
	files := []string{
		"ftse_oil_labour_conflict.json",
		"missing_invalidation.json",
		"poor_risk_reward.json",
		"missing_confirmation.json",
		"valid_swing_candidate.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc swingGoldenCase
			readJSON(t, filepath.Join("swing", file), &tc)

			decision := swing.Evaluate(swing.EvaluationInput{
				Event:                     tc.Event,
				Scores:                    tc.Scores,
				Catalyst:                  tc.Catalyst,
				SetupFamily:               tc.SetupFamily,
				RequiredConfirmations:     tc.RequiredConfirmations,
				PresentConfirmations:      tc.PresentConfirmations,
				MissingConfirmations:      tc.MissingConfirmations,
				InvalidationConditions:    tc.Invalidation,
				RiskRewardRatio:           tc.RiskRewardRatio,
				UnresolvedEventRisk:       tc.UnresolvedEventRisk,
				MarketSectorAlignmentNote: tc.MarketAlignment,
				GeneratedAt:               time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC),
			})

			coreDecision := decision.Decision
			if coreDecision.Decision != tc.ExpectedDecision {
				t.Fatalf("%s decision = %s, want %s", tc.Name, coreDecision.Decision, tc.ExpectedDecision)
			}
			for _, forbiddenDecision := range tc.NeverDecision {
				if coreDecision.Decision == forbiddenDecision {
					t.Fatalf("%s decision must never be %s", tc.Name, forbiddenDecision)
				}
			}
			if coreDecision.PrimaryReason != tc.ExpectedPrimaryReason {
				t.Fatalf("%s primary reason = %q, want %q", tc.Name, coreDecision.PrimaryReason, tc.ExpectedPrimaryReason)
			}
			if coreDecision.Brain != swing.BrainSwingV1 {
				t.Fatalf("%s brain = %q, want %q", tc.Name, coreDecision.Brain, swing.BrainSwingV1)
			}
			assertContainsAll(t, coreDecision.AllowedActions, tc.ExpectedAllowed)
			assertContainsAll(t, coreDecision.ForbiddenActions, tc.ExpectedForbidden)
			assertContainsAll(t, coreDecision.ReviewAfter, tc.ExpectedReviewAfter)
		})
	}
}
