package macroevents

import (
	"context"
	"testing"
)

func TestMultiAnalystReviewServiceEvaluatesAndPersists(t *testing.T) {
	store := &fakeMultiAnalystReviewStore{}
	service := NewMultiAnalystReviewService(store)

	record, err := service.EvaluateAndSave(context.Background(), MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			ID:             "ad-1",
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
		},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected persisted review id")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved reviews = %d, want 1", len(store.saved))
	}
}

func TestMultiAnalystReviewServiceWithoutStoreReturnsDecision(t *testing.T) {
	service := NewMultiAnalystReviewService(nil)

	record, err := service.EvaluateAndSave(context.Background(), MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{},
		Technical:    TechnicalSnapshot{},
		AnalystDecision: AnalystDecisionRecord{
			Decision: AnalystDecisionInsufficientEvidence,
		},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if record.Review.Decision != AnalystDecisionInsufficientEvidence {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionInsufficientEvidence)
	}
}

type fakeMultiAnalystReviewStore struct {
	saved []MultiAnalystReviewRecord
}

func (s *fakeMultiAnalystReviewStore) SaveMultiAnalystReview(_ context.Context, review MultiAnalystReviewRecord) (MultiAnalystReviewRecord, error) {
	review.ID = "mar-1"
	s.saved = append(s.saved, review)
	return review, nil
}
