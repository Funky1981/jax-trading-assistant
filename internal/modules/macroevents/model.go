package macroevents

import "time"

type EventType string
type Direction string
type Status string

const (
	EventTypeUSNonfarmPayrolls       EventType = "US_NONFARM_PAYROLLS"
	EventTypeUSUnemploymentRate      EventType = "US_UNEMPLOYMENT_RATE"
	EventTypeUSAverageHourlyEarnings EventType = "US_AVERAGE_HOURLY_EARNINGS"
	EventTypeUSCPIHeadline           EventType = "US_CPI_HEADLINE"
	EventTypeUSCPICore               EventType = "US_CPI_CORE"
	EventTypeUSPPI                   EventType = "US_PPI"
	EventTypeFOMCRateDecision        EventType = "FOMC_RATE_DECISION"
	EventTypeFOMCStatement           EventType = "FOMC_STATEMENT"
	EventTypeFOMCDotPlot             EventType = "FOMC_DOT_PLOT"
	EventTypeFedChairPressConference EventType = "FED_CHAIR_PRESS_CONFERENCE"
)

const (
	DirectionHawkishRates  Direction = "hawkish_rates"
	DirectionDovishRates   Direction = "dovish_rates"
	DirectionRiskOn        Direction = "risk_on"
	DirectionRiskOff       Direction = "risk_off"
	DirectionInflationHot  Direction = "inflation_hot"
	DirectionInflationCool Direction = "inflation_cool"
	DirectionGrowthStrong  Direction = "growth_strong"
	DirectionGrowthWeak    Direction = "growth_weak"
	DirectionUnclear       Direction = "unclear"
)

const (
	StatusAccepted    Status = "accepted"
	StatusRejected    Status = "rejected"
	StatusQuarantined Status = "quarantined"
)

type EventInput struct {
	MacroEventID  string
	Source        string
	SourceEventID string
	EventType     EventType
	Region        string
	EventTimeUTC  time.Time
	Headline      string
	Summary       string
	ActualValue   *float64
	ExpectedValue *float64
	PreviousValue *float64
	Unit          string
	Direction     Direction
	Confidence    float64
	RawPayload    map[string]any
	AffectedETFs  []string
}

type ETFMapping struct {
	Symbol        string
	Theme         string
	MappingReason string
	Confidence    float64
}

type StoredEvent struct {
	ID              string
	Input           EventInput
	Status          Status
	RejectionReason string
	Mappings        []ETFMapping
}

type Receipt struct {
	MacroEventID    string
	Status          Status
	RejectionReason string
	Duplicate       bool
	MappedETFs      []string
}

type ValidationResult struct {
	Valid  bool
	Status Status
	Reason string
}
