package macroevents

import (
	"context"
	"testing"
)

func TestScenarioServiceEvaluatesAndPersistsResult(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	store := &fakeScenarioStore{}
	service := NewScenarioService(store)

	result, err := service.EvaluateAndSave(context.Background(), EventInput{
		SourceEventID: "macro-1",
		EventType:     EventTypeUSNonfarmPayrolls,
		ActualValue:   &actual,
		ExpectedValue: &expected,
		Direction:     DirectionHawkishRates,
		AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}

	if result.ID == "" {
		t.Fatal("expected persisted result id")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved results = %d, want 1", len(store.saved))
	}
	if store.saved[0].Result != ScenarioResultEligibleForReactionCheck {
		t.Fatalf("saved result = %q, want eligible", store.saved[0].Result)
	}
}

type fakeScenarioStore struct {
	saved []ScenarioEvaluation
}

func (s *fakeScenarioStore) SaveScenarioResult(_ context.Context, result ScenarioEvaluation) (ScenarioEvaluation, error) {
	result.ID = "scenario-1"
	s.saved = append(s.saved, result)
	return result, nil
}
