package macroevents

import "testing"

func TestEvaluateFundamentalSnapshotReturnsInsufficientDataOnMissingExpectedValues(t *testing.T) {
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event: EventInput{
			Headline:     "Hot CPI",
			EventType:    EventTypeUSCPIHeadline,
			Direction:    DirectionInflationHot,
			AffectedETFs: []string{"QQQ", "SPY", "TLT"},
			Confidence:   0.9,
		},
	})

	if snapshot.Verdict != FundamentalVerdictInsufficientData {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, FundamentalVerdictInsufficientData)
	}
	if len(snapshot.MissingEvidence) == 0 {
		t.Fatal("expected missing evidence")
	}
}

func TestEvaluateFundamentalSnapshotAllowsQualitativeFedEventsWithoutNumericValues(t *testing.T) {
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
		MacroEventID:   "macro-1",
		Symbol:         "QQQ",
		ExpectedImpact: "hawkish rates pressure growth equities",
		AffectedThemes: []string{"growth", "rates"},
		CrossMarketChecks: []FundamentalCheck{
			{Symbol: "TLT", Expected: "down", Observed: "down", Confirmed: true, Reason: "rates confirmed"},
		},
		Event: EventInput{
			Headline:     "Fed statement reads hawkish",
			EventType:    EventTypeFOMCStatement,
			Direction:    DirectionHawkishRates,
			AffectedETFs: []string{"QQQ", "SPY", "TLT"},
			Confidence:   0.85,
		},
	})

	if snapshot.Verdict == FundamentalVerdictInsufficientData {
		t.Fatalf("verdict = %q, want qualitative Fed event to proceed", snapshot.Verdict)
	}
	if containsStringFold(snapshot.MissingEvidence, "actual vs expected") {
		t.Fatalf("missing evidence = %v, did not expect numeric release requirement", snapshot.MissingEvidence)
	}
}

func TestEvaluateFundamentalSnapshotStrongBearishForHotCPI(t *testing.T) {
	actual := 3.1
	expected := 2.7
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
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

	if snapshot.Verdict != FundamentalVerdictStrongBearish {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, FundamentalVerdictStrongBearish)
	}
	if snapshot.FundamentalScore < 70 {
		t.Fatalf("score = %v, want >= 70", snapshot.FundamentalScore)
	}
	if snapshot.ExpectedMarketImpact == "" {
		t.Fatal("expected market impact")
	}
}

func TestEvaluateFundamentalSnapshotStrongBullishForCoolCPI(t *testing.T) {
	actual := 2.4
	expected := 2.7
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
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
		AffectedThemes:    []string{"rates_duration"},
	})

	if snapshot.Verdict != FundamentalVerdictStrongBullish {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, FundamentalVerdictStrongBullish)
	}
	if snapshot.FundamentalScore < 70 {
		t.Fatalf("score = %v, want >= 70", snapshot.FundamentalScore)
	}
}

func TestEvaluateFundamentalSnapshotFlagsConflictedWithMajorConfounder(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Event: EventInput{
			Headline:      "Strong jobs",
			EventType:     EventTypeUSNonfarmPayrolls,
			Direction:     DirectionHawkishRates,
			ActualValue:   &actual,
			ExpectedValue: &expected,
			Confidence:    0.9,
			AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
		},
		Scenario:          ScenarioEvaluation{ScenarioKey: ScenarioHawkishRates, Result: ScenarioResultEligibleForReactionCheck},
		Confounders:       []Confounder{{Type: "fed_speaker", Headline: "Fed speaker due now", Severity: "high", Reason: "can reverse move", BlocksCandidate: true}},
		CrossMarketChecks: []FundamentalCheck{{Symbol: "TLT", Expected: "down", Observed: "up", Confirmed: false, Reason: "rates not confirming"}},
	})

	if snapshot.Verdict != FundamentalVerdictConflicted {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, FundamentalVerdictConflicted)
	}
}

func TestEvaluateFundamentalSnapshotFlagsSymbolMismatch(t *testing.T) {
	actual := 3.0
	expected := 3.2
	snapshot := EvaluateFundamentalSnapshot(FundamentalInput{
		MacroEventID: "macro-1",
		Symbol:       "SMH",
		Event: EventInput{
			Headline:      "Nvidia AI demand",
			EventType:     EventTypeUSCPIHeadline,
			Direction:     DirectionInflationCool,
			ActualValue:   &actual,
			ExpectedValue: &expected,
			Confidence:    0.9,
			AffectedETFs:  []string{"SMH", "SOXX", "QQQ"},
		},
		Scenario: ScenarioEvaluation{ScenarioKey: ScenarioDovishRates, Result: ScenarioResultEligibleForReactionCheck},
	})

	if len(snapshot.MissingEvidence) != 0 {
		t.Fatalf("missing evidence = %#v, want none for mapped symbol", snapshot.MissingEvidence)
	}
}
