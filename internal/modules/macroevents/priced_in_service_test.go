package macroevents

import (
	"context"
	"testing"
)

func TestPricedInServiceScoresAndPersists(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	store := &fakePricedInStore{}
	service := NewPricedInService(store)

	score, err := service.ScoreAndSave(context.Background(), PricedInInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event:        EventInput{ActualValue: &actual, ExpectedValue: &expected},
		Reaction: ReactionSnapshot{
			Status:        ReactionStatusAvailable,
			ConfirmsEvent: true,
			ChangePercent: -0.008,
		},
	})
	if err != nil {
		t.Fatalf("ScoreAndSave returned error: %v", err)
	}

	if score.ID == "" {
		t.Fatal("expected persisted score id")
	}
	if len(store.scores) != 1 {
		t.Fatalf("scores saved = %d, want 1", len(store.scores))
	}
}

func TestPricedInServiceSavesConfounders(t *testing.T) {
	store := &fakePricedInStore{}
	service := NewPricedInService(store)

	confounders, err := service.DetectAndSaveConfounders(context.Background(), "macro-1", []ConfounderInput{{
		Type:     "geopolitical_shock",
		Headline: "Major unrelated shock hits tape",
		Severity: "critical",
		Reason:   "unrelated shock may explain ETF move",
	}})
	if err != nil {
		t.Fatalf("DetectAndSaveConfounders returned error: %v", err)
	}

	if len(confounders) != 1 {
		t.Fatalf("confounders = %d, want 1", len(confounders))
	}
	if len(store.confounders) != 1 {
		t.Fatalf("confounders saved = %d, want 1", len(store.confounders))
	}
}

type fakePricedInStore struct {
	scores      []PricedInScore
	confounders []Confounder
}

func (s *fakePricedInStore) SavePricedInScore(_ context.Context, score PricedInScore) (PricedInScore, error) {
	score.ID = "score-1"
	s.scores = append(s.scores, score)
	return score, nil
}

func (s *fakePricedInStore) SaveConfounders(_ context.Context, confounders []Confounder) ([]Confounder, error) {
	for i := range confounders {
		confounders[i].ID = "confounder-1"
	}
	s.confounders = append(s.confounders, confounders...)
	return confounders, nil
}
