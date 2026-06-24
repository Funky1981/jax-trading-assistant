package classify

import (
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestMapAffectedAssetsForOilAndUKLabourRates(t *testing.T) {
	event := core.Event{
		Headline:  "FTSE falls as oil slump outweighs strong UK labour data and BoE rates risk",
		Geography: []string{"UK"},
	}

	got := MapAffectedAssets(event, []Driver{DriverOil, DriverLabourData, DriverRates, DriverCentralBank})
	assertContainsAll(t, got, []string{"FTSE100", "BP", "SHEL", "GBP", "UK_GILTS"})
}

func TestMapAffectedAssetsByDriver(t *testing.T) {
	tests := []struct {
		name    string
		event   core.Event
		drivers []Driver
		want    []string
	}{
		{
			name:    "gas",
			drivers: []Driver{DriverGas},
			want:    []string{"GAS_PRODUCERS", "UTILITIES", "ENERGY_ETFS"},
		},
		{
			name:    "earnings and guidance company",
			event:   core.Event{AffectedAssets: []string{"ACME"}},
			drivers: []Driver{DriverEarnings, DriverGuidance},
			want:    []string{"ACME", "PEERS", "SECTOR_ETF"},
		},
		{
			name:    "geopolitical oil risk",
			drivers: []Driver{DriverGeopolitical, DriverOil},
			want:    []string{"OIL", "ENERGY", "AIRLINES", "DEFENCE", "BROAD_INDICES"},
		},
		{
			name:    "inflation",
			drivers: []Driver{DriverInflation},
			want:    []string{"BONDS", "FX", "RATE_SENSITIVE_EQUITIES"},
		},
		{
			name:    "currency",
			drivers: []Driver{DriverCurrency},
			want:    []string{"FX_PAIRS", "EXPORTERS", "IMPORTERS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapAffectedAssets(tt.event, tt.drivers)
			assertContainsAll(t, got, tt.want)
		})
	}
}
