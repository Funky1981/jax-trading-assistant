package macroevents

import "testing"

func TestScoreAnalystDecisionAllowsHighAlignment(t *testing.T) {
	record := ScoreAnalystDecision(AnalystDecisionInput{
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

	if record.Decision != AnalystDecisionCandidateAllowed {
		t.Fatalf("decision = %q, want %q", record.Decision, AnalystDecisionCandidateAllowed)
	}
	if record.CandidateScore < 75 {
		t.Fatalf("candidate score = %v, want >= 75", record.CandidateScore)
	}
	if len(record.HardVetoes) != 0 {
		t.Fatalf("hard vetoes = %#v, want none", record.HardVetoes)
	}
}

func TestScoreAnalystDecisionPricedInOverridesHighScore(t *testing.T) {
	record := ScoreAnalystDecision(AnalystDecisionInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		Technical:       TechnicalSnapshot{ID: "ta-1", TechnicalScore: 90, Verdict: TechnicalVerdictConfirmedBearish},
		Fundamental:     FundamentalSnapshot{ID: "fa-1", FundamentalScore: 92, Verdict: FundamentalVerdictStrongBearish},
		PricedIn:        PricedInScore{Verdict: PricedInVerdictPricedIn, BlocksCandidate: true},
		RiskScore:       80,
		ConfidenceScore: 80,
		HasStopLevel:    true,
		RewardRisk:      2.0,
		Allowlisted:     true,
		RiskGuardrail:   "paper mode only; human approval required",
		EntryStopTarget: "entry after confirmation; stop beyond event candle; target 1R",
	})

	if record.Decision != AnalystDecisionCandidateRejected {
		t.Fatalf("decision = %q, want %q", record.Decision, AnalystDecisionCandidateRejected)
	}
	if !containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoPricedIn) {
		t.Fatalf("hard vetoes = %#v, want priced_in", record.HardVetoes)
	}
}

func TestScoreAnalystDecisionMissingDataReturnsInsufficientEvidence(t *testing.T) {
	record := ScoreAnalystDecision(AnalystDecisionInput{
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

	if record.Decision != AnalystDecisionInsufficientEvidence {
		t.Fatalf("decision = %q, want %q", record.Decision, AnalystDecisionInsufficientEvidence)
	}
	if !containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoMarketDataMissing) {
		t.Fatalf("hard vetoes = %#v, want market_data_missing", record.HardVetoes)
	}
}
