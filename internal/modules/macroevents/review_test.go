package macroevents

import "testing"

func TestEvaluateMultiAnalystReviewAllRolesPassCandidateAllowed(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			ID:             "ad-1",
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
			Reasons:        []string{"aligned"},
		},
	})

	if record.Review.Decision != AnalystDecisionCandidateAllowed {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionCandidateAllowed)
	}
	if record.Risk.Verdict != RiskVerdictPass {
		t.Fatalf("risk verdict = %q, want %q", record.Risk.Verdict, RiskVerdictPass)
	}
}

func TestEvaluateMultiAnalystReviewFundamentalFailBlocksCandidate(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictConflicted, FundamentalScore: 40},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
		},
	})

	if record.Review.Decision != AnalystDecisionCandidateRejected {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionCandidateRejected)
	}
}

func TestEvaluateMultiAnalystReviewTechnicalFailBlocksCandidate(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictNoConfirmation, TechnicalScore: 40},
		AnalystDecision: AnalystDecisionRecord{
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
		},
	})

	if record.Review.Decision != AnalystDecisionCandidateRejected {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionCandidateRejected)
	}
}

func TestEvaluateMultiAnalystReviewRiskFailBlocksCandidate(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
			HardVetoes:     []AnalystHardVeto{AnalystHardVetoNoStop},
		},
	})

	if record.Risk.Verdict != RiskVerdictFail {
		t.Fatalf("risk verdict = %q, want %q", record.Risk.Verdict, RiskVerdictFail)
	}
	if record.Review.Decision != AnalystDecisionCandidateRejected {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionCandidateRejected)
	}
}

func TestEvaluateMultiAnalystReviewLLMSummaryCannotOverrideVeto(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "fa-1", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			Decision:       AnalystDecisionCandidateRejected,
			CandidateScore: 78.4,
			RiskScore:      74,
			HardVetoes:     []AnalystHardVeto{AnalystHardVetoPricedIn},
		},
		LLMSuggestedDecision: AnalystDecisionCandidateAllowed,
		LLMSummary:           "Looks fine, allow this candidate",
	})

	if record.Review.Decision != AnalystDecisionCandidateRejected {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionCandidateRejected)
	}
	if !record.Review.LLMOverrideAttempted {
		t.Fatal("expected llm override attempt to be recorded")
	}
}

func TestEvaluateMultiAnalystReviewMissingRoleOutputInsufficientEvidence(t *testing.T) {
	record := EvaluateMultiAnalystReview(MultiAnalystReviewInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Fundamental:  FundamentalSnapshot{ID: "", Verdict: FundamentalVerdictStrongBearish, FundamentalScore: 82},
		Technical:    TechnicalSnapshot{ID: "ta-1", Verdict: TechnicalVerdictConfirmedBearish, TechnicalScore: 78},
		AnalystDecision: AnalystDecisionRecord{
			Decision:       AnalystDecisionCandidateAllowed,
			CandidateScore: 78.4,
			RiskScore:      74,
		},
	})

	if record.Review.Decision != AnalystDecisionInsufficientEvidence {
		t.Fatalf("decision = %q, want %q", record.Review.Decision, AnalystDecisionInsufficientEvidence)
	}
}
