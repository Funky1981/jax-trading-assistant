package core

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateConflictAndWeakEdgeReturnsNoTrade(t *testing.T) {
	event := Event{
		EventID:            "evt_2026_06_18_ftse_oil_labour",
		SourceType:         "news_url",
		SourceURL:          "https://example.com/article",
		ReceivedAt:         mustTime(t, "2026-06-18T10:30:00Z"),
		Headline:           "FTSE falls as oil slump outweighs strong UK labour data",
		Summary:            "FTSE weakness appears driven by oil weakness while labour data is stronger than expected.",
		EventType:          "MACRO_COMMODITY_INDEX_MOVE",
		PrimaryDrivers:     []string{"oil_price_drop"},
		ConflictingDrivers: []string{"strong_uk_labour_data", "boe_policy_uncertainty"},
		AffectedAssets:     []string{"FTSE100", "BP", "SHEL", "GBP", "UK_GILTS"},
		AssetClasses:       []string{"equity_index", "single_stock", "fx", "rates"},
		Geography:          []string{"UK"},
		TimeSensitivity:    "medium",
		UncertaintyNotes:   []string{"BoE decision pending", "Index move may be composition-driven"},
	}

	bundle := Evaluate(EvaluationInput{
		Event: event,
		Scores: Scores{
			ClarityScore:  0.38,
			EdgeScore:     0.22,
			ConflictScore: 0.79,
			RiskScore:     0.66,
		},
		GeneratedAt: mustTime(t, "2026-06-18T10:35:00Z"),
	})

	decision := bundle.FinalDecision
	if decision.Decision != DecisionNoTrade {
		t.Fatalf("decision = %s, want %s", decision.Decision, DecisionNoTrade)
	}
	if decision.EventID != event.EventID {
		t.Fatalf("event id = %q, want %q", decision.EventID, event.EventID)
	}
	if decision.Brain != BrainDecisionCore {
		t.Fatalf("brain = %q, want %q", decision.Brain, BrainDecisionCore)
	}
	if !strings.Contains(strings.ToLower(decision.PrimaryReason), "conflicting macro drivers") {
		t.Fatalf("primary reason = %q, want conflicting macro drivers", decision.PrimaryReason)
	}
	assertContainsAll(t, decision.AllowedActions, []string{ActionStoreEvent, ActionMonitor, ActionReviewLater})
	assertContainsAll(t, decision.ForbiddenActions, []string{ActionExecuteTrade, ActionCreateLiveOrder, ActionAutoApprove})
	assertContainsAll(t, decision.ReviewAfter, []string{ReviewAfter1Day, ReviewAfter1Week, ReviewAfter1Month})
	if decision.IsError() {
		t.Fatal("NO_TRADE must be a valid successful decision, not an error")
	}
}

func TestEvaluateRulesV1(t *testing.T) {
	tests := []struct {
		name string
		in   EvaluationInput
		want DecisionValue
	}{
		{
			name: "low clarity returns no trade",
			in: EvaluationInput{
				Event:  minimalEvent(),
				Scores: Scores{ClarityScore: 0.49, EdgeScore: 0.80, ConflictScore: 0.10, RiskScore: 0.20},
			},
			want: DecisionNoTrade,
		},
		{
			name: "high risk returns no trade",
			in: EvaluationInput{
				Event:  minimalEvent(),
				Scores: Scores{ClarityScore: 0.80, EdgeScore: 0.80, ConflictScore: 0.10, RiskScore: 0.71},
			},
			want: DecisionNoTrade,
		},
		{
			name: "medium edge with missing confirmations returns watch",
			in: EvaluationInput{
				Event:                minimalEvent(),
				Scores:               Scores{ClarityScore: 0.70, EdgeScore: 0.60, ConflictScore: 0.20, RiskScore: 0.30},
				MissingConfirmations: []string{"volume confirmation"},
			},
			want: DecisionWatch,
		},
		{
			name: "strong clean edge returns trade candidate",
			in: EvaluationInput{
				Event:                  minimalEvent(),
				Scores:                 Scores{ClarityScore: 0.72, EdgeScore: 0.76, ConflictScore: 0.30, RiskScore: 0.60},
				SupportingReasons:      []string{"Confirmed catalyst with clean asset-specific edge."},
				RequiredConfirmations:  []string{"risk veto pass"},
				InvalidationConditions: []string{"setup loses catalyst support"},
			},
			want: DecisionTradeCandidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.in).FinalDecision
			if got.Decision != tt.want {
				t.Fatalf("decision = %s, want %s", got.Decision, tt.want)
			}
			assertContainsAll(t, got.ForbiddenActions, []string{ActionExecuteTrade, ActionCreateLiveOrder, ActionAutoApprove})
			if got.Decision == DecisionPaperApprovalRequired {
				t.Fatal("Phase 1 must not return PAPER_APPROVAL_REQUIRED")
			}
		})
	}
}

func TestAllowedDecisionValues(t *testing.T) {
	got := AllowedDecisions()
	want := []DecisionValue{
		DecisionNoTrade,
		DecisionWatch,
		DecisionSetupForming,
		DecisionTradeCandidate,
		DecisionPaperApprovalRequired,
		DecisionRejectedByRisk,
	}
	if len(got) != len(want) {
		t.Fatalf("allowed decisions length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allowed decisions[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func minimalEvent() Event {
	return Event{
		EventID:         "evt_test",
		SourceType:      "manual",
		ReceivedAt:      time.Date(2026, 6, 18, 10, 30, 0, 0, time.UTC),
		Headline:        "Structured market event",
		Summary:         "A deterministic structured event.",
		EventType:       "MARKET_EVENT",
		PrimaryDrivers:  []string{"confirmed_catalyst"},
		AffectedAssets:  []string{"SPY"},
		AssetClasses:    []string{"equity_index"},
		Geography:       []string{"US"},
		TimeSensitivity: "medium",
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func assertContainsAll(t *testing.T, got []string, want []string) {
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
