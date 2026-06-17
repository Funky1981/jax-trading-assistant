package tradingmodes

import "fmt"

type TradingHorizon string

const (
	HorizonResearchOnly TradingHorizon = "research_only"
	HorizonIntraday     TradingHorizon = "intraday"
	HorizonSwing        TradingHorizon = "swing"
)

type CandidateHorizonPolicy struct {
	Horizon              TradingHorizon `json:"horizon"`
	HoldTargetDays       int            `json:"holdTargetDays,omitempty"`
	MaxHoldDays          int            `json:"maxHoldDays,omitempty"`
	FlattenByClose       bool           `json:"flattenByClose"`
	OvernightRiskAllowed bool           `json:"overnightRiskAllowed"`
	WeekendHoldAllowed   bool           `json:"weekendHoldAllowed"`
	RequiresDailyReview  bool           `json:"requiresDailyReview"`
	RevalidationSchedule string         `json:"revalidationSchedule,omitempty"`
	ThesisInvalidators   []string       `json:"thesisInvalidators,omitempty"`
}

func IntradayHorizonPolicy() CandidateHorizonPolicy {
	return CandidateHorizonPolicy{
		Horizon:              HorizonIntraday,
		FlattenByClose:       true,
		OvernightRiskAllowed: false,
		WeekendHoldAllowed:   false,
		RequiresDailyReview:  false,
	}
}

func SwingHorizonPolicy(holdTargetDays, maxHoldDays int) CandidateHorizonPolicy {
	return CandidateHorizonPolicy{
		Horizon:              HorizonSwing,
		HoldTargetDays:       holdTargetDays,
		MaxHoldDays:          maxHoldDays,
		FlattenByClose:       false,
		OvernightRiskAllowed: true,
		WeekendHoldAllowed:   false,
		RequiresDailyReview:  true,
		RevalidationSchedule: "daily_after_close",
		ThesisInvalidators: []string{
			"event_thesis_invalidated",
			"confounder_detected",
			"daily_close_breaks_risk_level",
		},
	}
}

func (p CandidateHorizonPolicy) Validate() error {
	switch p.Horizon {
	case HorizonResearchOnly:
		if p.OvernightRiskAllowed {
			return fmt.Errorf("research-only horizon cannot allow overnight risk")
		}
	case HorizonIntraday:
		if p.OvernightRiskAllowed {
			return fmt.Errorf("intraday horizon cannot allow overnight risk")
		}
		if !p.FlattenByClose {
			return fmt.Errorf("intraday horizon must flatten by close")
		}
		if p.MaxHoldDays > 0 {
			return fmt.Errorf("intraday horizon cannot set max hold days")
		}
	case HorizonSwing:
		if p.FlattenByClose {
			return fmt.Errorf("swing horizon cannot use intraday flatten-by-close policy")
		}
		if !p.OvernightRiskAllowed {
			return fmt.Errorf("swing horizon must allow overnight risk")
		}
		if !p.RequiresDailyReview {
			return fmt.Errorf("swing horizon requires daily review")
		}
		if p.HoldTargetDays < 2 || p.HoldTargetDays > 10 {
			return fmt.Errorf("swing hold target days must be between 2 and 10")
		}
		if p.MaxHoldDays > 10 {
			return fmt.Errorf("swing max hold days cannot exceed 10")
		}
	default:
		return fmt.Errorf("unknown trading horizon %q", p.Horizon)
	}
	return nil
}
