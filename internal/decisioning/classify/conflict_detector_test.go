package classify

import (
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestDetectConflictsInFTSEOilLabourEvent(t *testing.T) {
	event := core.Event{
		Headline:         "FTSE 100 today: stocks fall as oil slump outweighs strong UK labour data",
		Summary:          "FTSE falls because oil weakness drags energy names while UK labour data is stronger than expected and BoE decision risk remains.",
		UncertaintyNotes: []string{"BoE decision pending"},
	}

	got := DetectConflicts(event, []Driver{DriverOil, DriverLabourData, DriverCentralBank, DriverRates})
	gotCategories := conflictCategories(got)
	assertContainsAll(t, gotCategories, []string{
		string(DriverLabourData),
		string(DriverCentralBank),
		string(DriverRates),
	})
	if len(got) == 0 {
		t.Fatal("expected at least one conflict")
	}
	if !contains(got[0].DecisionPressure, PressureSupportsNoTrade) {
		t.Fatalf("decision pressure = %v, want %s", got[0].DecisionPressure, PressureSupportsNoTrade)
	}
}

func TestDetectsRequiredConflictPatterns(t *testing.T) {
	tests := []struct {
		name    string
		event   core.Event
		drivers []Driver
		want    string
	}{
		{
			name:    "earnings beat with guidance cut",
			event:   core.Event{Headline: "Company earnings beat expectations but guidance cut follows"},
			drivers: []Driver{DriverEarnings, DriverGuidance},
			want:    string(DriverGuidance),
		},
		{
			name:    "index move from heavyweight sector",
			event:   core.Event{Headline: "Index falls as heavyweight energy sector drags the market"},
			drivers: []Driver{DriverIndexComposition, DriverOil},
			want:    string(DriverIndexComposition),
		},
		{
			name:    "strong macro data but rates and fx contradict",
			event:   core.Event{Headline: "Strong jobs data but GBP falls and gilt yields drop"},
			drivers: []Driver{DriverLabourData, DriverCurrency, DriverRates},
			want:    string(DriverCurrency),
		},
		{
			name:    "rumour only catalyst",
			event:   core.Event{Headline: "Stock jumps on rumour-only takeover catalyst"},
			drivers: []Driver{DriverSentiment},
			want:    string(DriverSentiment),
		},
		{
			name:    "move already happened with no edge",
			event:   core.Event{Headline: "Event explains what already happened but no clean asset-specific edge"},
			drivers: []Driver{DriverSentiment},
			want:    string(DriverSentiment),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := conflictCategories(DetectConflicts(tt.event, tt.drivers))
			if !contains(got, tt.want) {
				t.Fatalf("conflict categories = %v, want %s", got, tt.want)
			}
		})
	}
}
