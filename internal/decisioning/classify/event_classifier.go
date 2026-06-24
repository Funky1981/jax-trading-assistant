package classify

import (
	"strings"

	"jax-trading-assistant/internal/decisioning/core"
)

type EventType string

const (
	EventTypeMacroData               EventType = "MACRO_DATA"
	EventTypeCentralBank             EventType = "CENTRAL_BANK"
	EventTypeCommodityShock          EventType = "COMMODITY_SHOCK"
	EventTypeEarnings                EventType = "EARNINGS"
	EventTypeGuidance                EventType = "GUIDANCE"
	EventTypeRegulatory              EventType = "REGULATORY"
	EventTypeGeopolitical            EventType = "GEOPOLITICAL"
	EventTypeSectorRotation          EventType = "SECTOR_ROTATION"
	EventTypeIndexComposition        EventType = "INDEX_COMPOSITION"
	EventTypeCompanySpecific         EventType = "COMPANY_SPECIFIC"
	EventTypeMarketStructure         EventType = "MARKET_STRUCTURE"
	EventTypeMacroCommodityIndexMove EventType = "MACRO_COMMODITY_INDEX_MOVE"
	EventTypeUnknown                 EventType = "UNKNOWN"
)

const (
	PressureSupportsNoTrade  = "supports_no_trade"
	PressureSupportsWatch    = "supports_watch"
	PressureNeedsMoreContext = "needs_more_context"
)

type Classification struct {
	EventType           EventType
	SecondaryEventTypes []EventType
	ConfidenceScore     float64
	UncertaintyNotes    []string
}

type EventIntelligence struct {
	Event             core.Event
	ClassifiedType    EventType
	NormalisedDrivers []Driver
	Conflicts         []Conflict
	AffectedAssets    []string
	UncertaintyNotes  []string
	ConfidenceScore   float64
	DecisionPressure  []string
}

func ClassifyEvent(event core.Event) Classification {
	text := eventText(event)
	drivers := ExtractDrivers(event)

	if hasAnyDriver(drivers, DriverOil, DriverGas) && hasIndexMove(text) && containsAny(text, "ftse", "index", "stocks", "market") {
		return Classification{
			EventType: EventTypeMacroCommodityIndexMove,
			SecondaryEventTypes: []EventType{
				EventTypeMacroData,
				EventTypeCommodityShock,
				EventTypeIndexComposition,
			},
			ConfidenceScore: 0.82,
		}
	}
	if containsAny(text, "boe", "bank of england", "fed", "federal reserve", "ecb", "rate decision", "rate cut", "rate hold", "rate hike", "minutes", "policy statement", "speech") {
		return Classification{EventType: EventTypeCentralBank, ConfidenceScore: 0.84}
	}
	if containsAny(text, "cpi", "inflation", "jobs", "wages", "unemployment", "gdp", "retail sales", "labour data", "labor data") {
		return Classification{EventType: EventTypeMacroData, ConfidenceScore: 0.78}
	}
	if hasAnyDriver(drivers, DriverOil, DriverGas) || containsAny(text, "metals", "supply disruption", "inventory shock") {
		return Classification{EventType: EventTypeCommodityShock, ConfidenceScore: 0.76}
	}
	if containsAny(text, "guidance", "outlook", "profit warning") && containsAny(text, "raise", "raises", "raised", "cut", "cuts", "warning", "warns") {
		return Classification{EventType: EventTypeGuidance, ConfidenceScore: 0.82}
	}
	if containsAny(text, "earnings", "results", "profit", "revenue") && containsAny(text, "beat", "beats", "miss", "misses", "quarterly", "reports") {
		return Classification{EventType: EventTypeEarnings, ConfidenceScore: 0.80}
	}
	if containsAny(text, "sanctions", "war", "escalation", "de-escalation", "conflict", "strait of hormuz") {
		return Classification{EventType: EventTypeGeopolitical, ConfidenceScore: 0.76}
	}
	if containsAny(text, "approval", "fine", "ban", "investigation", "regulatory", "regulator") {
		return Classification{EventType: EventTypeRegulatory, ConfidenceScore: 0.72}
	}
	if containsAny(text, "sector rotation", "rotation from", "rotation into") {
		return Classification{EventType: EventTypeSectorRotation, ConfidenceScore: 0.72}
	}
	if containsAny(text, "heavyweight", "index composition", "constituents", "one sector dragging") {
		return Classification{EventType: EventTypeIndexComposition, ConfidenceScore: 0.70}
	}
	if containsAny(text, "management", "product", "debt", "m&a", "merger", "takeover") {
		return Classification{EventType: EventTypeCompanySpecific, ConfidenceScore: 0.66}
	}
	if containsAny(text, "liquidity", "breakout", "breakdown", "technical", "flows") {
		return Classification{EventType: EventTypeMarketStructure, ConfidenceScore: 0.66}
	}

	return Classification{
		EventType:        EventTypeUnknown,
		ConfidenceScore:  0.30,
		UncertaintyNotes: []string{"Event type unclear."},
	}
}

func EnrichEvent(event core.Event) EventIntelligence {
	classification := ClassifyEvent(event)
	drivers := ExtractDrivers(event)
	conflicts := DetectConflicts(event, drivers)
	assets := MapAffectedAssets(event, drivers)
	notes := uniqueStrings(append(cloneStrings(event.UncertaintyNotes), classification.UncertaintyNotes...))
	pressures := decisionPressures(classification, conflicts)

	enriched := event
	enriched.EventType = string(classification.EventType)
	enriched.PrimaryDrivers = driverStrings(drivers)
	enriched.ConflictingDrivers = conflictCategories(conflicts)
	enriched.AffectedAssets = assets
	enriched.UncertaintyNotes = notes

	return EventIntelligence{
		Event:             enriched,
		ClassifiedType:    classification.EventType,
		NormalisedDrivers: drivers,
		Conflicts:         conflicts,
		AffectedAssets:    assets,
		UncertaintyNotes:  notes,
		ConfidenceScore:   classification.ConfidenceScore,
		DecisionPressure:  pressures,
	}
}

func decisionPressures(classification Classification, conflicts []Conflict) []string {
	var pressures []string
	if classification.EventType == EventTypeUnknown || classification.ConfidenceScore < 0.50 {
		pressures = append(pressures, PressureSupportsNoTrade, PressureNeedsMoreContext)
	}
	if len(conflicts) > 0 {
		pressures = append(pressures, PressureSupportsNoTrade, PressureNeedsMoreContext)
	}
	if len(pressures) == 0 {
		pressures = append(pressures, PressureSupportsWatch)
	}
	return uniqueStrings(pressures)
}

func eventText(event core.Event) string {
	parts := []string{
		event.Headline,
		event.Summary,
		strings.Join(event.PrimaryDrivers, " "),
		strings.Join(event.ConflictingDrivers, " "),
		strings.Join(event.UncertaintyNotes, " "),
		strings.Join(event.AffectedAssets, " "),
		strings.Join(event.Geography, " "),
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func hasIndexMove(text string) bool {
	return containsAny(text, "fall", "falls", "slump", "slumps", "drop", "drops", "rally", "rises", "move", "weakness")
}
