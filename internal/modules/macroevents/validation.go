package macroevents

import (
	"fmt"
	"strings"
	"time"
)

const freshnessWindow = 24 * time.Hour

func Validate(input EventInput, now time.Time) ValidationResult {
	input = NormalizeInput(input)
	if strings.TrimSpace(input.Source) == "" {
		return rejected("source is required")
	}
	if strings.TrimSpace(input.SourceEventID) == "" {
		return rejected("source_event_id is required")
	}
	if !supportedEventType(input.EventType) {
		return rejected("event_type is not supported")
	}
	if strings.TrimSpace(input.Region) == "" {
		return rejected("region is required")
	}
	if !strings.EqualFold(strings.TrimSpace(input.Region), "US") {
		return rejected("region is not supported in phase 1")
	}
	if input.EventTimeUTC.IsZero() {
		return rejected("event_time_utc is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if input.EventTimeUTC.UTC().Before(now.UTC().Add(-freshnessWindow)) {
		return rejected("event is older than freshness window")
	}
	if strings.TrimSpace(input.Headline) == "" {
		return rejected("headline is required")
	}
	if numericReleaseRequiresValues(input.EventType) {
		if input.ActualValue == nil {
			return rejected("actual_value is required for numeric macro release")
		}
		if input.ExpectedValue == nil {
			return rejected("expected_value is required for numeric macro release")
		}
	}
	if input.Confidence <= 0 || input.Confidence > 1 {
		return rejected("confidence must be greater than 0 and less than or equal to 1")
	}
	if len(nonEmptyStrings(input.AffectedETFs)) == 0 {
		return rejected("affected_etfs are required")
	}
	if containsForbiddenPayloadKey(input.RawPayload, "") {
		return rejected("payload contains forbidden trading or runtime override fields")
	}
	if input.Confidence < 0.5 {
		return ValidationResult{Valid: false, Status: StatusQuarantined, Reason: "confidence below threshold"}
	}
	return ValidationResult{Valid: true, Status: StatusAccepted}
}

func NormalizeInput(input EventInput) EventInput {
	input.Source = strings.ToLower(strings.TrimSpace(input.Source))
	input.SourceEventID = strings.TrimSpace(input.SourceEventID)
	input.EventType = EventType(strings.ToUpper(strings.TrimSpace(string(input.EventType))))
	input.Region = strings.ToUpper(strings.TrimSpace(input.Region))
	input.Headline = strings.TrimSpace(input.Headline)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Direction = Direction(strings.ToLower(strings.TrimSpace(string(input.Direction))))
	return input
}

func rejected(reason string) ValidationResult {
	return ValidationResult{Valid: false, Status: StatusRejected, Reason: reason}
}

func supportedEventType(eventType EventType) bool {
	switch EventType(strings.ToUpper(strings.TrimSpace(string(eventType)))) {
	case EventTypeUSNonfarmPayrolls,
		EventTypeUSUnemploymentRate,
		EventTypeUSAverageHourlyEarnings,
		EventTypeUSCPIHeadline,
		EventTypeUSCPICore,
		EventTypeUSPPI,
		EventTypeFOMCRateDecision,
		EventTypeFOMCStatement,
		EventTypeFOMCDotPlot,
		EventTypeFedChairPressConference:
		return true
	default:
		return false
	}
}

func numericReleaseRequiresValues(eventType EventType) bool {
	switch EventType(strings.ToUpper(strings.TrimSpace(string(eventType)))) {
	case EventTypeUSNonfarmPayrolls,
		EventTypeUSUnemploymentRate,
		EventTypeUSAverageHourlyEarnings,
		EventTypeUSCPIHeadline,
		EventTypeUSCPICore,
		EventTypeUSPPI,
		EventTypeFOMCRateDecision:
		return true
	default:
		return false
	}
}

func containsForbiddenPayloadKey(payload map[string]any, prefix string) bool {
	for key, value := range payload {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if prefix != "" {
			normalized = prefix + "." + normalized
		}
		if forbiddenPayloadKey(normalized) {
			return true
		}
		if nested, ok := value.(map[string]any); ok && containsForbiddenPayloadKey(nested, normalized) {
			return true
		}
	}
	return false
}

func forbiddenPayloadKey(key string) bool {
	switch key {
	case "runtime_mode", "execution_enabled", "broker_order", "order", "position_size", "risk_override":
		return true
	default:
		return strings.HasSuffix(key, ".runtime_mode") ||
			strings.HasSuffix(key, ".execution_enabled") ||
			strings.HasSuffix(key, ".broker_order") ||
			strings.HasSuffix(key, ".order") ||
			strings.HasSuffix(key, ".position_size") ||
			strings.HasSuffix(key, ".risk_override")
	}
}

func nonEmptyStrings(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ComputeSurprise(actual, expected *float64) (*float64, *float64) {
	if actual == nil || expected == nil {
		return nil, nil
	}
	value := *actual - *expected
	var percent *float64
	if *expected != 0 {
		pct := value / *expected
		percent = &pct
	}
	return &value, percent
}

func ValidateDirection(direction Direction) error {
	switch Direction(strings.ToLower(strings.TrimSpace(string(direction)))) {
	case DirectionHawkishRates,
		DirectionDovishRates,
		DirectionRiskOn,
		DirectionRiskOff,
		DirectionInflationHot,
		DirectionInflationCool,
		DirectionGrowthStrong,
		DirectionGrowthWeak,
		DirectionUnclear:
		return nil
	default:
		return fmt.Errorf("direction is not supported")
	}
}
