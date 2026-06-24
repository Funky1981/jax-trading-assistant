package classify

import "jax-trading-assistant/internal/decisioning/core"

type Driver string

const (
	DriverRates              Driver = "rates"
	DriverInflation          Driver = "inflation"
	DriverLabourData         Driver = "labour_data"
	DriverOil                Driver = "oil"
	DriverGas                Driver = "gas"
	DriverEarnings           Driver = "earnings"
	DriverGuidance           Driver = "guidance"
	DriverLiquidity          Driver = "liquidity"
	DriverCurrency           Driver = "currency"
	DriverGeopolitical       Driver = "geopolitical"
	DriverRegulatory         Driver = "regulatory"
	DriverSentiment          Driver = "sentiment"
	DriverTechnicalBreakout  Driver = "technical_breakout"
	DriverTechnicalBreakdown Driver = "technical_breakdown"
	DriverValuation          Driver = "valuation"
	DriverSectorRotation     Driver = "sector_rotation"
	DriverIndexComposition   Driver = "index_composition"
	DriverCentralBank        Driver = "central_bank"
)

func ExtractDrivers(event core.Event) []Driver {
	text := eventText(event)
	var drivers []Driver

	addIf := func(driver Driver, keywords ...string) {
		if containsAny(text, keywords...) {
			drivers = append(drivers, driver)
		}
	}

	addIf(DriverOil, "oil", "crude", "brent", "wti", "oil_price")
	addIf(DriverGas, "gas", "lng")
	addIf(DriverLabourData, "labour", "labor", "wages", "unemployment", "jobs", "payroll")
	addIf(DriverCentralBank, "boe", "bank of england", "fed", "federal reserve", "ecb", "central bank", "policy statement", "minutes")
	addIf(DriverRates, "rate", "rates", "gilt", "gilts", "yield", "yields", "boe", "fed", "ecb")
	addIf(DriverInflation, "cpi", "inflation", "prices")
	addIf(DriverEarnings, "earnings", "results", "profit", "revenue", "beat", "miss")
	addIf(DriverGuidance, "guidance", "outlook", "profit warning", "raise guidance", "cut guidance")
	addIf(DriverLiquidity, "liquidity", "flows")
	addIf(DriverCurrency, "gbp", "sterling", "dollar", "usd", "fx", "currency")
	addIf(DriverGeopolitical, "sanctions", "war", "conflict", "escalation", "de-escalation", "strait of hormuz")
	addIf(DriverRegulatory, "regulatory", "regulator", "approval", "fine", "ban", "investigation")
	addIf(DriverSentiment, "sentiment", "rumour", "rumor")
	addIf(DriverTechnicalBreakout, "breakout")
	addIf(DriverTechnicalBreakdown, "breakdown")
	addIf(DriverValuation, "valuation", "priced in", "multiple")
	addIf(DriverSectorRotation, "sector rotation", "rotation into", "rotation from")
	addIf(DriverIndexComposition, "heavyweight", "index composition", "constituent", "energy-heavy", "one sector")

	return uniqueDrivers(drivers)
}

func hasAnyDriver(drivers []Driver, want ...Driver) bool {
	seen := map[Driver]bool{}
	for _, driver := range drivers {
		seen[driver] = true
	}
	for _, driver := range want {
		if seen[driver] {
			return true
		}
	}
	return false
}

func driverStrings(drivers []Driver) []string {
	out := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		out = append(out, string(driver))
	}
	return out
}

func uniqueDrivers(drivers []Driver) []Driver {
	seen := map[Driver]bool{}
	out := make([]Driver, 0, len(drivers))
	for _, driver := range drivers {
		if seen[driver] {
			continue
		}
		seen[driver] = true
		out = append(out, driver)
	}
	return out
}
