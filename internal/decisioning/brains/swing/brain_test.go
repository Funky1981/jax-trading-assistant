package swing

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestEvaluateSwingRules(t *testing.T) {
	baseEvent := core.Event{
		EventID:         "evt_swing_test",
		ReceivedAt:      time.Date(2026, 6, 24, 9, 0, 0, 0, time.UTC),
		Headline:        "Energy stock confirms pullback continuation after clear oil catalyst",
		Summary:         "A confirmed multi-day move has a clear catalyst, invalidation, and defined reward.",
		EventType:       "COMMODITY_SHOCK",
		PrimaryDrivers:  []string{"oil"},
		AffectedAssets:  []string{"BP"},
		AssetClasses:    []string{"single_stock"},
		TimeSensitivity: "medium",
	}

	tests := []struct {
		name              string
		input             EvaluationInput
		wantDecision      core.DecisionValue
		wantPrimaryReason string
		wantAllowed       []string
	}{
		{
			name: "defaults to NO_TRADE when required trading evidence is absent",
			input: EvaluationInput{
				Event: baseEvent,
				Scores: core.Scores{
					ClarityScore:  0.80,
					EdgeScore:     0.80,
					ConflictScore: 0.10,
					RiskScore:     0.20,
				},
				Catalyst: "oil supply disruption",
			},
			wantDecision:      core.DecisionNoTrade,
			wantPrimaryReason: "Missing invalidation condition.",
			wantAllowed:       []string{core.ActionStoreEvent, core.ActionReviewLater},
		},
		{
			name: "rejects conflicting weak edge",
			input: EvaluationInput{
				Event: baseEvent,
				Scores: core.Scores{
					ClarityScore:  0.70,
					EdgeScore:     0.40,
					ConflictScore: 0.75,
					RiskScore:     0.40,
				},
				Catalyst:               "oil price move",
				InvalidationConditions: []string{"BP closes below prior swing low"},
				RiskRewardRatio:        2.5,
			},
			wantDecision:      core.DecisionNoTrade,
			wantPrimaryReason: "Conflicting drivers dominate and edge is weak.",
			wantAllowed:       []string{core.ActionStoreEvent, core.ActionReviewLater},
		},
		{
			name: "missing confirmation can only watch",
			input: EvaluationInput{
				Event: baseEvent,
				Scores: core.Scores{
					ClarityScore:  0.78,
					EdgeScore:     0.72,
					ConflictScore: 0.20,
					RiskScore:     0.35,
				},
				Catalyst:                  "oil supply disruption",
				SetupFamily:               CommodityLinkedEquityDislocation,
				InvalidationConditions:    []string{"BP closes below 470p"},
				RiskRewardRatio:           2.4,
				MissingConfirmations:      []string{"sector confirmation"},
				RequiredConfirmations:     []string{"sector confirmation", "price holds support"},
				MarketSectorAlignmentNote: "sector confirmation unavailable",
			},
			wantDecision:      core.DecisionWatch,
			wantPrimaryReason: "Required confirmations are missing.",
			wantAllowed:       []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionReviewLater, core.ActionPrepareResearch},
		},
		{
			name: "valid setup becomes trade candidate without execution authority",
			input: EvaluationInput{
				Event: baseEvent,
				Scores: core.Scores{
					ClarityScore:      0.84,
					EdgeScore:         0.82,
					ConflictScore:     0.18,
					RiskScore:         0.34,
					ConfirmationScore: 0.76,
					TimingScore:       0.70,
				},
				Catalyst:                  "oil supply disruption",
				SetupFamily:               CommodityLinkedEquityDislocation,
				RequiredConfirmations:     []string{"price holds support", "energy sector confirms"},
				PresentConfirmations:      []string{"price holds support", "energy sector confirms"},
				InvalidationConditions:    []string{"BP closes below 470p"},
				RiskRewardRatio:           2.6,
				MarketSectorAlignmentNote: "energy sector aligned",
			},
			wantDecision:      core.DecisionTradeCandidate,
			wantPrimaryReason: "Swing trade candidate meets catalyst, confirmation, invalidation, and risk/reward requirements.",
			wantAllowed:       []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Evaluate(tt.input)

			coreDecision := decision.Decision
			if coreDecision.Brain != BrainSwingV1 {
				t.Fatalf("brain = %q, want %q", coreDecision.Brain, BrainSwingV1)
			}
			if coreDecision.Decision != tt.wantDecision {
				t.Fatalf("decision = %s, want %s", coreDecision.Decision, tt.wantDecision)
			}
			if coreDecision.PrimaryReason != tt.wantPrimaryReason {
				t.Fatalf("primary reason = %q, want %q", coreDecision.PrimaryReason, tt.wantPrimaryReason)
			}
			assertContainsAll(t, coreDecision.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
			assertContainsAll(t, coreDecision.AllowedActions, tt.wantAllowed)
			if decision.Scores.ClarityScore != tt.input.Scores.ClarityScore {
				t.Fatalf("clarity score = %v, want %v", decision.Scores.ClarityScore, tt.input.Scores.ClarityScore)
			}
			if coreDecision.Decision == core.DecisionTradeCandidate && contains(coreDecision.ForbiddenActions, core.ActionExecuteTrade) == false {
				t.Fatal("trade candidate must still forbid live execution")
			}
		})
	}
}

func assertContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	for _, expected := range want {
		if !contains(got, expected) {
			t.Fatalf("expected %q in %v", expected, got)
		}
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
