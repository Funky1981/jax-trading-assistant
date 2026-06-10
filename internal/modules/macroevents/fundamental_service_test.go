package macroevents

import (
	"context"
	"testing"
)

func TestFundamentalServiceEvaluatesAndPersistsSnapshot(t *testing.T) {
	store := &fakeFundamentalStore{}
	service := NewFundamentalService(store)
	actual := 3.1
	expected := 2.7

	snapshot, err := service.EvaluateAndSave(t.Context(), FundamentalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event: EventInput{
			Headline:      "CPI hotter than expected",
			EventType:     EventTypeUSCPIHeadline,
			Direction:     DirectionInflationHot,
			ActualValue:   &actual,
			ExpectedValue: &expected,
			Confidence:    0.9,
			AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
		},
		Scenario:          ScenarioEvaluation{ScenarioKey: ScenarioHawkishRates, Result: ScenarioResultEligibleForReactionCheck},
		CrossMarketChecks: []FundamentalCheck{{Symbol: "TLT", Expected: "down", Observed: "down", Confirmed: true, Reason: "duration sold off"}},
		AffectedThemes:    []string{"growth/technology", "rates_duration"},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}

	if snapshot.ID == "" {
		t.Fatal("expected persisted snapshot id")
	}
	if len(store.snapshots) != 1 {
		t.Fatalf("saved snapshots = %d, want 1", len(store.snapshots))
	}
}

func TestFundamentalServiceWithoutStoreReturnsEvaluatedSnapshot(t *testing.T) {
	service := NewFundamentalService(nil)
	actual := 2.4
	expected := 2.7

	snapshot, err := service.EvaluateAndSave(t.Context(), FundamentalInput{
		MacroEventID: "macro-1",
		Symbol:       "TLT",
		Event: EventInput{
			Headline:      "CPI cooler than expected",
			EventType:     EventTypeUSCPIHeadline,
			Direction:     DirectionInflationCool,
			ActualValue:   &actual,
			ExpectedValue: &expected,
			Confidence:    0.9,
			AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
		},
		Scenario:          ScenarioEvaluation{ScenarioKey: ScenarioDovishRates, Result: ScenarioResultEligibleForReactionCheck},
		CrossMarketChecks: []FundamentalCheck{{Symbol: "TLT", Expected: "up", Observed: "up", Confirmed: true, Reason: "duration bid"}},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if snapshot.Verdict != FundamentalVerdictStrongBullish {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, FundamentalVerdictStrongBullish)
	}
}

type fakeFundamentalStore struct {
	snapshots []FundamentalSnapshot
}

func (s *fakeFundamentalStore) SaveFundamentalAnalysisSnapshot(_ context.Context, snapshot FundamentalSnapshot) (FundamentalSnapshot, error) {
	snapshot.ID = "fa-1"
	s.snapshots = append(s.snapshots, snapshot)
	return snapshot, nil
}
