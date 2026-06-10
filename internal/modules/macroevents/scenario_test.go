package macroevents

import "testing"

func TestEvaluateScenarioMapsNFPBeatToHawkishRatesPlaybook(t *testing.T) {
	actual := 172000.0
	expected := 85000.0
	result := EvaluateScenario(EventInput{
		EventType:     EventTypeUSNonfarmPayrolls,
		ActualValue:   &actual,
		ExpectedValue: &expected,
		Direction:     DirectionHawkishRates,
		AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
	})

	if result.Result != ScenarioResultEligibleForReactionCheck {
		t.Fatalf("result = %q, want eligible", result.Result)
	}
	if result.PlaybookKey != "strong_jobs_hawkish_rates" {
		t.Fatalf("playbook = %q, want strong_jobs_hawkish_rates", result.PlaybookKey)
	}
	if result.ScenarioKey != ScenarioHawkishRates {
		t.Fatalf("scenario = %q, want %q", result.ScenarioKey, ScenarioHawkishRates)
	}
	if result.ExpectedReactions["QQQ"] != ReactionDirectionDown {
		t.Fatalf("QQQ expected reaction = %q, want down", result.ExpectedReactions["QQQ"])
	}
}

func TestEvaluateScenarioMapsHotCPIToHawkishRatesPlaybook(t *testing.T) {
	actual := 3.4
	expected := 3.1
	result := EvaluateScenario(EventInput{
		EventType:     EventTypeUSCPIHeadline,
		ActualValue:   &actual,
		ExpectedValue: &expected,
		Direction:     DirectionInflationHot,
		AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
	})

	if result.ScenarioKey != ScenarioHawkishRates {
		t.Fatalf("scenario = %q, want hawkish rates", result.ScenarioKey)
	}
	if result.PlaybookKey != "hot_cpi_rates_hawkish" {
		t.Fatalf("playbook = %q, want hot_cpi_rates_hawkish", result.PlaybookKey)
	}
	if result.CandidateBias != CandidateBiasShortOrAvoidLong {
		t.Fatalf("candidate bias = %q, want short/avoid-long", result.CandidateBias)
	}
}

func TestEvaluateScenarioMapsCoolCPIToDovishRatesPlaybook(t *testing.T) {
	actual := 2.9
	expected := 3.1
	result := EvaluateScenario(EventInput{
		EventType:     EventTypeUSCPIHeadline,
		ActualValue:   &actual,
		ExpectedValue: &expected,
		Direction:     DirectionInflationCool,
		AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
	})

	if result.ScenarioKey != ScenarioDovishRates {
		t.Fatalf("scenario = %q, want dovish rates", result.ScenarioKey)
	}
	if result.PlaybookKey != "cool_cpi_rates_dovish" {
		t.Fatalf("playbook = %q, want cool_cpi_rates_dovish", result.PlaybookKey)
	}
	if result.ExpectedReactions["TLT"] != ReactionDirectionUp {
		t.Fatalf("TLT expected reaction = %q, want up", result.ExpectedReactions["TLT"])
	}
}

func TestEvaluateScenarioFedPressConferenceRequiresDelayedConfirmation(t *testing.T) {
	result := EvaluateScenario(EventInput{
		EventType:    EventTypeFedChairPressConference,
		Direction:    DirectionHawkishRates,
		AffectedETFs: []string{"QQQ", "SPY", "TLT"},
	})

	if !containsString(result.RequiredConfirmations, string(TimeframePostEvent15M)) {
		t.Fatalf("required confirmations = %#v, want post_event_15m", result.RequiredConfirmations)
	}
	if !containsString(result.RequiredConfirmations, string(TimeframePostEvent30M)) {
		t.Fatalf("required confirmations = %#v, want post_event_30m", result.RequiredConfirmations)
	}
}

func TestEvaluateScenarioUnknownEventBlocksCandidateCreation(t *testing.T) {
	result := EvaluateScenario(EventInput{
		EventType:    EventType("UNKNOWN_MACRO"),
		Direction:    DirectionUnclear,
		AffectedETFs: []string{"QQQ"},
	})

	if result.Result != ScenarioResultBlockedUnknownEvent {
		t.Fatalf("result = %q, want blocked_unknown_event", result.Result)
	}
}

func TestEvaluateScenarioRejectsDisallowedETFMapping(t *testing.T) {
	result := EvaluateScenario(EventInput{
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		AffectedETFs: []string{"TQQQ"},
	})

	if result.Result != ScenarioResultBlockedDisallowedInstrument {
		t.Fatalf("result = %q, want blocked_disallowed_instrument", result.Result)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
