package macroevents

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAcceptsValidNFPEvent(t *testing.T) {
	now := fixedValidationNow()
	actual := 172000.0
	expected := 85000.0
	previous := 139000.0

	result := Validate(EventInput{
		Source:        "calendar",
		SourceEventID: "nfp-2026-06",
		EventType:     EventTypeUSNonfarmPayrolls,
		Region:        "US",
		EventTimeUTC:  now.Add(-10 * time.Minute),
		Headline:      "US jobs beat expectations",
		Summary:       "Payroll growth was materially stronger than forecast.",
		ActualValue:   &actual,
		ExpectedValue: &expected,
		PreviousValue: &previous,
		Unit:          "jobs",
		Direction:     DirectionHawkishRates,
		Confidence:    0.91,
		AffectedETFs:  []string{"QQQ", "SPY", "TLT"},
		RawPayload:    map[string]any{"provider": "calendar"},
	}, now)

	if !result.Valid {
		t.Fatalf("expected valid result, got %#v", result)
	}
	if result.Status != StatusAccepted {
		t.Fatalf("status = %q, want %q", result.Status, StatusAccepted)
	}
}

func TestValidateRejectsNFPWithoutExpectedValue(t *testing.T) {
	now := fixedValidationNow()
	actual := 172000.0

	result := Validate(validMacroEventInput(now, EventTypeUSNonfarmPayrolls, actual, nil), now)

	requireInvalidReason(t, result, "expected_value is required")
}

func TestValidateRejectsUnsupportedEventType(t *testing.T) {
	now := fixedValidationNow()
	actual := 1.0
	expected := 2.0
	input := validMacroEventInput(now, EventType("UK_CPI"), actual, &expected)

	result := Validate(input, now)

	requireInvalidReason(t, result, "event_type is not supported")
}

func TestValidateRejectsStaleEvent(t *testing.T) {
	now := fixedValidationNow()
	actual := 1.0
	expected := 2.0
	input := validMacroEventInput(now, EventTypeUSCPIHeadline, actual, &expected)
	input.EventTimeUTC = now.Add(-25 * time.Hour)

	result := Validate(input, now)

	requireInvalidReason(t, result, "event is older than freshness window")
}

func TestValidateQuarantinesLowConfidenceEvent(t *testing.T) {
	now := fixedValidationNow()
	actual := 1.0
	expected := 2.0
	input := validMacroEventInput(now, EventTypeUSCPIHeadline, actual, &expected)
	input.Confidence = 0.49

	result := Validate(input, now)

	if result.Valid {
		t.Fatalf("expected low-confidence event to be non-valid for candidate flow, got %#v", result)
	}
	if result.Status != StatusQuarantined {
		t.Fatalf("status = %q, want %q", result.Status, StatusQuarantined)
	}
	if !strings.Contains(result.Reason, "confidence below threshold") {
		t.Fatalf("reason = %q, want confidence threshold reason", result.Reason)
	}
}

func TestValidateRejectsRuntimeOverridePayload(t *testing.T) {
	now := fixedValidationNow()
	actual := 1.0
	expected := 2.0
	input := validMacroEventInput(now, EventTypeUSCPIHeadline, actual, &expected)
	input.RawPayload = map[string]any{
		"nested": map[string]any{
			"broker_order": true,
		},
	}

	result := Validate(input, now)

	requireInvalidReason(t, result, "payload contains forbidden trading or runtime override fields")
}

func fixedValidationNow() time.Time {
	return time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
}

func validMacroEventInput(now time.Time, eventType EventType, actual float64, expected *float64) EventInput {
	return EventInput{
		Source:        "calendar",
		SourceEventID: "event-1",
		EventType:     eventType,
		Region:        "US",
		EventTimeUTC:  now.Add(-10 * time.Minute),
		Headline:      "Macro event",
		Summary:       "Macro event summary",
		ActualValue:   &actual,
		ExpectedValue: expected,
		Unit:          "percent",
		Direction:     DirectionInflationHot,
		Confidence:    0.85,
		AffectedETFs:  []string{"QQQ"},
		RawPayload:    map[string]any{"provider": "calendar"},
	}
}

func requireInvalidReason(t *testing.T, result ValidationResult, want string) {
	t.Helper()
	if result.Valid {
		t.Fatalf("expected invalid result, got %#v", result)
	}
	if result.Status != StatusRejected {
		t.Fatalf("status = %q, want %q", result.Status, StatusRejected)
	}
	if !strings.Contains(result.Reason, want) {
		t.Fatalf("reason = %q, want substring %q", result.Reason, want)
	}
}
