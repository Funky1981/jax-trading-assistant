package replay

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

type BoolValue *bool

func Bool(value bool) BoolValue {
	return &value
}

type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type ReplayInput struct {
	ReplayID              string               `json:"replay_id"`
	DateRange             DateRange            `json:"date_range"`
	EventTypeFilter       []string             `json:"event_type_filter"`
	SetupFamilyFilter     []string             `json:"setup_family_filter"`
	DecisionFilter        []core.DecisionValue `json:"decision_filter"`
	AssetFilter           []string             `json:"asset_filter"`
	IncludeNoTrades       BoolValue            `json:"include_no_trades"`
	IncludeWatch          BoolValue            `json:"include_watch"`
	IncludeRejectedByRisk BoolValue            `json:"include_rejected_by_risk"`
	IncludePaperOutcomes  BoolValue            `json:"include_paper_outcomes"`
	IncludeLessons        BoolValue            `json:"include_lessons"`
	Records               []Record             `json:"records"`
	CreatedAt             time.Time            `json:"created_at"`
}

func includeDefault(value BoolValue) bool {
	if value == nil {
		return true
	}
	return *value
}

func recordMatches(input ReplayInput, record Record) bool {
	if !input.DateRange.From.IsZero() && record.CreatedAt.Before(input.DateRange.From) {
		return false
	}
	if !input.DateRange.To.IsZero() && record.CreatedAt.After(input.DateRange.To) {
		return false
	}
	if !stringAllowed(input.EventTypeFilter, record.EventType) {
		return false
	}
	if !stringAllowed(input.SetupFamilyFilter, record.SetupFamily) {
		return false
	}
	if !decisionAllowed(input.DecisionFilter, record.FinalDecision) {
		return false
	}
	if !stringAllowed(input.AssetFilter, record.Asset) {
		return false
	}
	if record.FinalDecision == core.DecisionNoTrade && !includeDefault(input.IncludeNoTrades) {
		return false
	}
	if record.FinalDecision == core.DecisionWatch && !includeDefault(input.IncludeWatch) {
		return false
	}
	if (record.RejectedByRisk || record.FinalDecision == core.DecisionRejectedByRisk) && !includeDefault(input.IncludeRejectedByRisk) {
		return false
	}
	return true
}

func stringAllowed(filter []string, value string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, allowed := range filter {
		if allowed == value {
			return true
		}
	}
	return false
}

func decisionAllowed(filter []core.DecisionValue, value core.DecisionValue) bool {
	if len(filter) == 0 {
		return true
	}
	for _, allowed := range filter {
		if allowed == value {
			return true
		}
	}
	return false
}
