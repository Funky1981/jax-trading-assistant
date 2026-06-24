package classify

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestClassifyEventTypes(t *testing.T) {
	tests := []struct {
		name  string
		event core.Event
		want  EventType
	}{
		{
			name: "ftse oil labour classified as macro commodity index move",
			event: core.Event{
				Headline:       "FTSE 100 today: stocks fall as oil slump outweighs strong UK labour data",
				Summary:        "FTSE falls because oil weakness drags energy names while UK labour data is stronger than expected and BoE decision risk remains.",
				PrimaryDrivers: []string{"oil_price_drop"},
			},
			want: EventTypeMacroCommodityIndexMove,
		},
		{
			name:  "central bank rate event",
			event: core.Event{Headline: "BoE holds rates after Fed and ECB policy statements"},
			want:  EventTypeCentralBank,
		},
		{
			name:  "macro data event",
			event: core.Event{Headline: "US CPI cools while jobs, wages and unemployment data point to softer GDP"},
			want:  EventTypeMacroData,
		},
		{
			name:  "commodity shock event",
			event: core.Event{Headline: "Brent oil jumps after confirmed supply disruption"},
			want:  EventTypeCommodityShock,
		},
		{
			name:  "earnings event",
			event: core.Event{Headline: "Company reports earnings beat and revenue miss in quarterly results"},
			want:  EventTypeEarnings,
		},
		{
			name:  "guidance event",
			event: core.Event{Headline: "Company cuts guidance after outlook warning"},
			want:  EventTypeGuidance,
		},
		{
			name:  "unclear event",
			event: core.Event{Headline: "Shares move after mixed update"},
			want:  EventTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyEvent(tt.event)
			if got.EventType != tt.want {
				t.Fatalf("event type = %s, want %s", got.EventType, tt.want)
			}
		})
	}
}

func TestEnrichUnknownEventPreservesUncertaintyAndSupportsNoTrade(t *testing.T) {
	event := core.Event{
		EventID:          "evt_unknown",
		SourceType:       "manual",
		ReceivedAt:       time.Date(2026, 6, 18, 10, 30, 0, 0, time.UTC),
		Headline:         "Shares move after mixed update",
		Summary:          "The catalyst and affected assets are not clear.",
		UncertaintyNotes: []string{"Source does not explain the driver."},
	}

	got := EnrichEvent(event)
	if got.Event.EventType != string(EventTypeUnknown) {
		t.Fatalf("event type = %s, want %s", got.Event.EventType, EventTypeUnknown)
	}
	if !contains(got.UncertaintyNotes, "Source does not explain the driver.") {
		t.Fatalf("uncertainty notes = %v, want original note preserved", got.UncertaintyNotes)
	}
	if !contains(got.DecisionPressure, PressureSupportsNoTrade) {
		t.Fatalf("decision pressure = %v, want %s", got.DecisionPressure, PressureSupportsNoTrade)
	}
	if got.ConfidenceScore >= 0.50 {
		t.Fatalf("confidence score = %.2f, want low confidence for unknown", got.ConfidenceScore)
	}
}
