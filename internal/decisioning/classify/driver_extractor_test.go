package classify

import (
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestExtractDriversFromFTSEOilLabourEvent(t *testing.T) {
	event := core.Event{
		Headline:           "FTSE 100 today: stocks fall as oil slump outweighs strong UK labour data",
		Summary:            "FTSE falls because oil weakness drags energy names while UK labour data is stronger than expected and BoE decision risk remains.",
		PrimaryDrivers:     []string{"oil_price_drop"},
		ConflictingDrivers: []string{"strong_uk_labour_data", "boe_policy_uncertainty"},
		UncertaintyNotes:   []string{"BoE decision pending"},
	}

	got := ExtractDrivers(event)
	assertContainsAll(t, driverStrings(got), []string{
		string(DriverOil),
		string(DriverLabourData),
		string(DriverCentralBank),
		string(DriverRates),
	})
}

func TestExtractDriversNormalisesCommonCategories(t *testing.T) {
	tests := []struct {
		name  string
		event core.Event
		want  []string
	}{
		{
			name:  "earnings beat and guidance cut",
			event: core.Event{Headline: "Earnings beat but guidance cut hits outlook"},
			want:  []string{string(DriverEarnings), string(DriverGuidance)},
		},
		{
			name:  "currency and geopolitical",
			event: core.Event{Headline: "Sterling and dollar move as sanctions and Strait of Hormuz risk rise"},
			want:  []string{string(DriverCurrency), string(DriverGeopolitical)},
		},
		{
			name:  "gas and regulatory",
			event: core.Event{Headline: "Gas producers fall after regulatory investigation"},
			want:  []string{string(DriverGas), string(DriverRegulatory)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := driverStrings(ExtractDrivers(tt.event))
			assertContainsAll(t, got, tt.want)
		})
	}
}
