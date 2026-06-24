package classify

import "jax-trading-assistant/internal/decisioning/core"

type Conflict struct {
	Category         Driver
	Description      string
	DecisionPressure []string
}

func DetectConflicts(event core.Event, drivers []Driver) []Conflict {
	text := eventText(event)
	var conflicts []Conflict
	add := func(category Driver, description string) {
		conflicts = append(conflicts, Conflict{
			Category:         category,
			Description:      description,
			DecisionPressure: []string{PressureSupportsNoTrade, PressureNeedsMoreContext},
		})
	}

	if hasAnyDriver(drivers, DriverOil) && hasAnyDriver(drivers, DriverLabourData) && (hasAnyDriver(drivers, DriverCentralBank, DriverRates) || containsAny(text, "boe pending", "decision pending")) {
		add(DriverLabourData, "Oil pressure conflicts with strong labour data and unresolved central bank/rates risk.")
		add(DriverCentralBank, "Major central bank decision risk remains unresolved.")
		add(DriverRates, "Rates reaction may contradict the commodity/index move.")
	}
	if hasAnyDriver(drivers, DriverEarnings, DriverGuidance) && containsAny(text, "beat", "beats") && containsAny(text, "guidance cut", "cuts guidance", "cut guidance", "outlook warning") {
		add(DriverGuidance, "Historic earnings strength conflicts with weaker forward guidance.")
	}
	if hasAnyDriver(drivers, DriverIndexComposition) || containsAny(text, "heavyweight", "one sector", "composition-driven", "energy-heavy") {
		add(DriverIndexComposition, "Index move may be driven by heavyweight sector exposure rather than broad market weakness.")
	}
	if hasAnyDriver(drivers, DriverLabourData, DriverInflation) && hasAnyDriver(drivers, DriverRates, DriverCurrency) && containsAny(text, "but", "contradict", "falls", "drop", "disagree") {
		add(DriverCurrency, "Strong macro data conflicts with rates or FX reaction.")
	}
	if hasAnyDriver(drivers, DriverCentralBank) && containsAny(text, "pending", "tomorrow", "ahead of", "decision risk") {
		add(DriverCentralBank, "Major central bank decision is pending.")
	}
	if containsAny(text, "rumour", "rumor", "rumour-only", "rumor-only") {
		add(DriverSentiment, "Catalyst is rumour-only and source quality is weak.")
	}
	if containsAny(text, "already happened", "no clean asset-specific edge", "no clean edge") {
		add(DriverSentiment, "Event explains a past move without a clean asset-specific edge.")
	}

	return uniqueConflicts(conflicts)
}

func conflictCategories(conflicts []Conflict) []string {
	out := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		out = append(out, string(conflict.Category))
	}
	return uniqueStrings(out)
}

func uniqueConflicts(conflicts []Conflict) []Conflict {
	seen := map[Driver]bool{}
	out := make([]Conflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if seen[conflict.Category] {
			continue
		}
		seen[conflict.Category] = true
		out = append(out, conflict)
	}
	return out
}
