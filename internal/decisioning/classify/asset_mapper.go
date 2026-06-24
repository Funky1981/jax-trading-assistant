package classify

import "jax-trading-assistant/internal/decisioning/core"

func MapAffectedAssets(event core.Event, drivers []Driver) []string {
	assets := cloneStrings(event.AffectedAssets)
	text := eventText(event)

	add := func(values ...string) {
		assets = append(assets, values...)
	}

	if hasAnyDriver(drivers, DriverOil) {
		add("BP", "SHEL", "ENERGY_ETFS", "BRENT", "WTI")
		if hasAnyDriver(drivers, DriverGeopolitical) {
			add("OIL", "ENERGY", "AIRLINES", "DEFENCE", "BROAD_INDICES")
		}
	}
	if hasAnyDriver(drivers, DriverGas) {
		add("GAS_PRODUCERS", "UTILITIES", "ENERGY_ETFS")
	}
	if hasAnyDriver(drivers, DriverLabourData, DriverRates) && containsAny(text, "uk", "ftse", "gbp", "gilt", "boe", "labour") {
		add("GBP", "UK_GILTS", "FTSE100", "UK_BANKS", "HOUSEBUILDERS")
	}
	if hasAnyDriver(drivers, DriverCentralBank) {
		add("INDEX", "RATES", "FX", "BANKS", "HOUSEBUILDERS", "REITS")
	}
	if hasAnyDriver(drivers, DriverEarnings, DriverGuidance) {
		add("PEERS", "SECTOR_ETF")
	}
	if hasAnyDriver(drivers, DriverGeopolitical) && !hasAnyDriver(drivers, DriverOil) {
		add("COMMODITIES", "FX", "DEFENCE", "BROAD_INDICES")
	}
	if hasAnyDriver(drivers, DriverInflation) {
		add("BONDS", "FX", "RATE_SENSITIVE_EQUITIES")
	}
	if hasAnyDriver(drivers, DriverCurrency) {
		add("FX_PAIRS", "EXPORTERS", "IMPORTERS")
	}
	if containsAny(text, "ftse") {
		add("FTSE100")
	}

	return uniqueStrings(assets)
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
