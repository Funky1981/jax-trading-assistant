package macroevents

import (
	"context"
	"testing"
)

func TestAnalystScoringServiceEvaluatesAndPersistsDecision(t *testing.T) {
	store := &fakeAnalystDecisionStore{}
	service := NewAnalystScoringService(store)

	record, err := service.EvaluateAndSave(context.Background(), AnalystDecisionInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		Technical:       TechnicalSnapshot{ID: "ta-1", TechnicalScore: 78, Verdict: TechnicalVerdictConfirmedBearish},
		Fundamental:     FundamentalSnapshot{ID: "fa-1", FundamentalScore: 82, Verdict: FundamentalVerdictStrongBearish},
		PricedIn:        PricedInScore{Verdict: PricedInVerdictNotPricedIn},
		RiskScore:       74,
		ConfidenceScore: 70,
		HasStopLevel:    true,
		RewardRisk:      1.8,
		Allowlisted:     true,
		RiskGuardrail:   "paper mode only; human approval required",
		EntryStopTarget: "entry after confirmation; stop beyond event candle; target 1R",
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected persisted analyst decision id")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved decisions = %d, want 1", len(store.saved))
	}
	if store.saved[0].Decision != AnalystDecisionCandidateAllowed {
		t.Fatalf("saved decision = %q, want %q", store.saved[0].Decision, AnalystDecisionCandidateAllowed)
	}
}

func TestAnalystScoringServiceWithoutStoreReturnsDecision(t *testing.T) {
	service := NewAnalystScoringService(nil)

	record, err := service.EvaluateAndSave(context.Background(), AnalystDecisionInput{
		MacroEventID:      "macro-1",
		Symbol:            "QQQ",
		Technical:         TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictInsufficientData},
		Fundamental:       FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictInsufficientData},
		PricedIn:          PricedInScore{},
		RiskScore:         0,
		ConfidenceScore:   0,
		HasStopLevel:      true,
		RewardRisk:        1.8,
		Allowlisted:       true,
		MarketDataMissing: true,
		RiskGuardrail:     "paper mode only",
		EntryStopTarget:   "entry after confirmation; stop beyond event candle; target 1R",
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if record.Decision != AnalystDecisionInsufficientEvidence {
		t.Fatalf("decision = %q, want %q", record.Decision, AnalystDecisionInsufficientEvidence)
	}
}

type fakeAnalystDecisionStore struct {
	saved []AnalystDecisionRecord
}

func (s *fakeAnalystDecisionStore) SaveAnalystDecision(_ context.Context, decision AnalystDecisionRecord) (AnalystDecisionRecord, error) {
	decision.ID = "ad-1"
	s.saved = append(s.saved, decision)
	return decision, nil
}
