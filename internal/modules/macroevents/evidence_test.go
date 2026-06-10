package macroevents

import "testing"

func TestBuildEvidenceBundleFullEvidenceAllowsCandidate(t *testing.T) {
	bundle := BuildEvidenceBundle(fullEvidenceInput())

	if bundle.Verdict != EvidenceVerdictCandidateAllowed {
		t.Fatalf("verdict = %q, want candidate_allowed; walkaway=%v missing=%v", bundle.Verdict, bundle.WalkawayReasons, bundle.MissingEvidence)
	}
	if len(bundle.MissingEvidence) != 0 {
		t.Fatalf("missing evidence = %#v, want none", bundle.MissingEvidence)
	}
}

func TestBuildEvidenceBundleMissingChartReactionInsufficientEvidence(t *testing.T) {
	input := fullEvidenceInput()
	input.Reaction = ReactionSnapshot{Status: ReactionStatusUnavailable}

	bundle := BuildEvidenceBundle(input)

	if bundle.Verdict != EvidenceVerdictInsufficientEvidence {
		t.Fatalf("verdict = %q, want insufficient_evidence", bundle.Verdict)
	}
	if !containsString(bundle.MissingEvidence, "chart reaction") {
		t.Fatalf("missing evidence = %#v, want chart reaction", bundle.MissingEvidence)
	}
}

func TestBuildEvidenceBundlePricedInBlocksCandidate(t *testing.T) {
	input := fullEvidenceInput()
	input.PricedIn = PricedInScore{Verdict: PricedInVerdictPricedIn, BlocksCandidate: true}

	bundle := BuildEvidenceBundle(input)

	if bundle.Verdict != EvidenceVerdictCandidateBlocked {
		t.Fatalf("verdict = %q, want candidate_blocked", bundle.Verdict)
	}
	if !containsString(bundle.WalkawayReasons, "priced-in verdict blocks candidate") {
		t.Fatalf("walkaway reasons = %#v, want priced-in block", bundle.WalkawayReasons)
	}
}

func TestBuildEvidenceBundleHighSeverityConfounderBlocksCandidate(t *testing.T) {
	input := fullEvidenceInput()
	input.Confounders = []Confounder{{Severity: "high", BlocksCandidate: true, Reason: "Fed speaker overlap"}}

	bundle := BuildEvidenceBundle(input)

	if bundle.Verdict != EvidenceVerdictCandidateBlocked {
		t.Fatalf("verdict = %q, want candidate_blocked", bundle.Verdict)
	}
	if !containsString(bundle.WalkawayReasons, "high-severity confounder blocks candidate") {
		t.Fatalf("walkaway reasons = %#v, want confounder block", bundle.WalkawayReasons)
	}
}

func TestBuildEvidenceBundleTooExtendedCreatesWatchOnly(t *testing.T) {
	input := fullEvidenceInput()
	input.Reaction.TooExtended = true
	input.Reaction.ConfirmsEvent = false

	bundle := BuildEvidenceBundle(input)

	if bundle.Verdict != EvidenceVerdictWatchOnly {
		t.Fatalf("verdict = %q, want watch_only", bundle.Verdict)
	}
}

func TestBuildEvidenceBundleMissingHistoricalComparisonIsExplicitLimitation(t *testing.T) {
	input := fullEvidenceInput()
	input.HistoricalComparison = ""

	bundle := BuildEvidenceBundle(input)

	if bundle.Verdict != EvidenceVerdictCandidateAllowed {
		t.Fatalf("verdict = %q, want candidate_allowed in phase 1", bundle.Verdict)
	}
	if !containsString(bundle.MissingEvidence, "historical comparison") {
		t.Fatalf("missing evidence = %#v, want historical comparison limitation", bundle.MissingEvidence)
	}
}

func fullEvidenceInput() EvidenceInput {
	return EvidenceInput{
		MacroEvent: EventInput{
			MacroEventID: "macro-1",
			EventType:    EventTypeUSCPIHeadline,
			Direction:    DirectionInflationHot,
			Headline:     "CPI hotter than expected",
			AffectedETFs: []string{"QQQ"},
			Confidence:   0.9,
		},
		Scenario: ScenarioEvaluation{
			ScenarioKey:       ScenarioHawkishRates,
			Result:            ScenarioResultEligibleForReactionCheck,
			PrimarySymbols:    []string{"QQQ"},
			ExpectedReactions: map[string]ReactionDirection{"QQQ": ReactionDirectionDown},
		},
		Reaction: ReactionSnapshot{
			Symbol:        "QQQ",
			Status:        ReactionStatusAvailable,
			ConfirmsEvent: true,
			Direction:     ReactionDirectionDown,
			ChangePercent: -0.008,
		},
		PricedIn: PricedInScore{
			Symbol:  "QQQ",
			Verdict: PricedInVerdictNotPricedIn,
			Reasons: []string{"large surprise with small pre-event drift"},
		},
		Confounders:          nil,
		HistoricalComparison: "phase-1 placeholder: historical comparison engine not complete",
		RiskGuardrail:        "paper mode only; human approval required",
		EntryStopTarget:      "entry after confirmation; stop beyond event candle; target 1R",
		Symbol:               "QQQ",
	}
}
