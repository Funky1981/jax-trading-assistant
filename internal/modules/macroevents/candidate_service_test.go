package macroevents

import (
	"context"
	"testing"
)

func TestMacroCandidateServiceGeneratesAndPersists(t *testing.T) {
	entry := 430.25
	stop := 435.10
	target := 421.00
	store := &fakeMacroCandidateStore{}
	service := NewMacroCandidateService(store)

	candidate, err := service.GenerateAndSave(context.Background(), CandidateInput{
		Bundle: allowedEvidenceBundle(),
		Side:   CandidateSideShortBias,
		Plan: CandidatePlan{
			EntryType:       EntryTypePullbackRetest,
			EntryPrice:      &entry,
			StopPrice:       &stop,
			TargetPrice:     &target,
			RiskPercent:     0.5,
			TimeLimit:       "end_of_session",
			RewardRiskRatio: 1.9,
		},
	})
	if err != nil {
		t.Fatalf("GenerateAndSave returned error: %v", err)
	}

	if candidate.ID == "" {
		t.Fatal("expected persisted candidate id")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved candidates = %d, want 1", len(store.saved))
	}
}

type fakeMacroCandidateStore struct {
	saved []MacroCandidate
}

func (s *fakeMacroCandidateStore) SaveMacroCandidate(_ context.Context, candidate MacroCandidate) (MacroCandidate, error) {
	candidate.ID = "macro-candidate-1"
	s.saved = append(s.saved, candidate)
	return candidate, nil
}
