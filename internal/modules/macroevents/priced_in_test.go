package macroevents

import "testing"

func TestScorePricedInLargeSurpriseSmallPreMoveNotPricedIn(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	result := ScorePricedIn(PricedInInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event: EventInput{
			ActualValue:   &actual,
			ExpectedValue: &expected,
		},
		PreEventMovePercent: 0.001,
		Reaction: ReactionSnapshot{
			Status:        ReactionStatusAvailable,
			ConfirmsEvent: true,
			ChangePercent: -0.008,
		},
	})

	if result.Verdict != PricedInVerdictNotPricedIn {
		t.Fatalf("verdict = %q, want not_priced_in", result.Verdict)
	}
}

func TestScorePricedInSmallSurpriseBigPreMovePricedIn(t *testing.T) {
	actual := 101.0
	expected := 100.0
	result := ScorePricedIn(PricedInInput{
		MacroEventID:          "macro-1",
		Symbol:                "SPY",
		Event:                 EventInput{ActualValue: &actual, ExpectedValue: &expected},
		PreEventMovePercent:   0.026,
		NewsSaturationScore:   0.8,
		VolatilityElevated:    true,
		AnalystConsensusTight: true,
		Reaction: ReactionSnapshot{
			Status:        ReactionStatusAvailable,
			ConfirmsEvent: false,
			ChangePercent: 0.001,
		},
	})

	if result.Verdict != PricedInVerdictPricedIn {
		t.Fatalf("verdict = %q, want priced_in", result.Verdict)
	}
	if !result.BlocksCandidate {
		t.Fatal("priced_in must block candidate")
	}
}

func TestScorePricedInMissingDataUnclear(t *testing.T) {
	result := ScorePricedIn(PricedInInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Reaction:     ReactionSnapshot{Status: ReactionStatusUnavailable},
	})

	if result.Verdict != PricedInVerdictUnclear {
		t.Fatalf("verdict = %q, want unclear", result.Verdict)
	}
	if !result.BlocksCandidate {
		t.Fatal("unclear must block candidate")
	}
}

func TestScorePricedInOverextendedMoveOverreaction(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	result := ScorePricedIn(PricedInInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event:        EventInput{ActualValue: &actual, ExpectedValue: &expected},
		Reaction: ReactionSnapshot{
			Status:        ReactionStatusAvailable,
			TooExtended:   true,
			ChangePercent: -0.032,
		},
	})

	if result.Verdict != PricedInVerdictOverreaction {
		t.Fatalf("verdict = %q, want overreaction", result.Verdict)
	}
	if !result.BlocksCandidate {
		t.Fatal("overreaction must block immediate candidate")
	}
}

func TestDetectConfoundersBlocksHighSeverityOverlap(t *testing.T) {
	confounders := DetectConfounders([]ConfounderInput{
		{
			Type:     "fed_speaker",
			Headline: "Fed chair speaks during CPI reaction window",
			Source:   "calendar",
			Severity: "high",
			Reason:   "same-time Fed speech can explain rate-sensitive ETF move",
		},
	})

	if len(confounders) != 1 {
		t.Fatalf("confounders = %d, want 1", len(confounders))
	}
	if !confounders[0].BlocksCandidate {
		t.Fatal("high-severity confounder must block candidate")
	}
}
