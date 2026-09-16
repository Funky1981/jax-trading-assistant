package exploratorypaper

import (
	"fmt"
	"sort"
	"time"
)

type MarketSession string

const (
	SessionOpen    MarketSession = "OPEN"
	SessionClosed  MarketSession = "CLOSED"
	SessionUnknown MarketSession = "UNKNOWN"
)

// SessionCalendar is explicit by design. An unknown calendar must not be
// silently approximated with calendar-day arithmetic.
type SessionCalendar struct{ Sessions map[string]bool }

func (c SessionCalendar) IsTradingSession(day time.Time) (bool, error) {
	if len(c.Sessions) == 0 {
		return false, fmt.Errorf("trading session calendar is unknown")
	}
	v, ok := c.Sessions[day.UTC().Format("2006-01-02")]
	if !ok {
		return false, fmt.Errorf("trading session %s is unknown", day.UTC().Format("2006-01-02"))
	}
	return v, nil
}

func (c SessionCalendar) TradingSessionsBetween(start, end time.Time) ([]time.Time, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return nil, fmt.Errorf("invalid session interval")
	}
	if len(c.Sessions) == 0 {
		return nil, fmt.Errorf("trading session calendar is unknown")
	}
	var out []time.Time
	for d := start.UTC().Truncate(24 * time.Hour); !d.After(end.UTC().Truncate(24 * time.Hour)); d = d.AddDate(0, 0, 1) {
		open, err := c.IsTradingSession(d)
		if err != nil {
			return nil, err
		}
		if open {
			out = append(out, d)
		}
	}
	return out, nil
}

func (c SessionCalendar) HardExitDate(entryAt time.Time) (time.Time, error) {
	if len(c.Sessions) == 0 {
		return time.Time{}, fmt.Errorf("trading session calendar is unknown")
	}
	var sessions []time.Time
	for offset := 0; offset <= 370 && len(sessions) < 5; offset++ {
		d := entryAt.UTC().AddDate(0, 0, offset)
		open, err := c.IsTradingSession(d)
		if err != nil {
			return time.Time{}, err
		}
		if open {
			sessions = append(sessions, d.UTC().Truncate(24*time.Hour))
		}
	}
	if len(sessions) == 0 || sessions[0].Format("2006-01-02") != entryAt.UTC().Format("2006-01-02") {
		return time.Time{}, fmt.Errorf("entry is not on a known trading session")
	}
	if len(sessions) < 5 {
		return time.Time{}, fmt.Errorf("calendar does not cover the fifth trading session")
	}
	return sessions[4], nil
}

func (c SessionCalendar) ReviewSessions(entryAt time.Time) ([]time.Time, error) {
	last, err := c.HardExitDate(entryAt)
	if err != nil {
		return nil, err
	}
	return c.TradingSessionsBetween(entryAt, last)
}

type ExitReason string

const (
	ExitStop              ExitReason = "STOP"
	ExitTarget            ExitReason = "TARGET"
	ExitThesisInvalidated ExitReason = "THESIS_INVALIDATED"
	ExitRiskKill          ExitReason = "RISK_KILL"
	ExitTimeLimit         ExitReason = "TIME_LIMIT"
	ExitManualOperator    ExitReason = "MANUAL_OPERATOR"
)

type ExitAction string

const (
	ExitNow                   ExitAction = "EXIT_NOW"
	ExitAtNextTradableSession ExitAction = "EXIT_AT_NEXT_TRADABLE_SESSION"
	NoExit                    ExitAction = "NO_EXIT"
)

type ExitInput struct {
	Now               time.Time
	Session           MarketSession
	Bid               float64
	Ask               float64
	ThesisInvalidated bool
	RiskKill          bool
	ManualOperator    bool
	Calendar          SessionCalendar
}

type ExitDecision struct {
	Action              ExitAction
	Reason              ExitReason
	At                  time.Time
	NextTradableSession time.Time
}

func EvaluateExit(position Position, in ExitInput) (ExitDecision, error) {
	if position.State == StateClosed {
		return ExitDecision{}, fmt.Errorf("position is closed")
	}
	if in.Now.IsZero() || in.Calendar.Sessions == nil {
		return ExitDecision{}, fmt.Errorf("exit evaluation requires a known calendar and time")
	}
	if in.Session == SessionUnknown {
		return ExitDecision{}, fmt.Errorf("unknown market session fails closed")
	}
	reason := ExitReason("")
	price := in.Bid
	if position.Thesis.Direction == DirectionShort {
		price = in.Ask
	}
	if in.RiskKill {
		reason = ExitRiskKill
	} else if in.ManualOperator {
		reason = ExitManualOperator
	} else if in.ThesisInvalidated || position.State == StateInvalidated {
		reason = ExitThesisInvalidated
	} else if price > 0 && ((position.Thesis.Direction == DirectionLong && price <= position.Thesis.ProtectiveStop) || (position.Thesis.Direction == DirectionShort && price >= position.Thesis.ProtectiveStop)) {
		reason = ExitStop
	} else if price > 0 && ((position.Thesis.Direction == DirectionLong && price >= position.Thesis.Target) || (position.Thesis.Direction == DirectionShort && price <= position.Thesis.Target)) {
		reason = ExitTarget
	}
	hard, err := in.Calendar.HardExitDate(position.EntryAt)
	if err != nil {
		return ExitDecision{}, err
	}
	if reason == "" && !in.Now.UTC().Truncate(24*time.Hour).Before(hard) {
		reason = ExitTimeLimit
	}
	if reason == "" {
		return ExitDecision{Action: NoExit}, nil
	}
	if in.Session == SessionClosed {
		next, err := nextTradable(in.Calendar, in.Now)
		if err != nil {
			return ExitDecision{}, err
		}
		return ExitDecision{Action: ExitAtNextTradableSession, Reason: reason, At: in.Now, NextTradableSession: next}, nil
	}
	return ExitDecision{Action: ExitNow, Reason: reason, At: in.Now}, nil
}

func nextTradable(c SessionCalendar, after time.Time) (time.Time, error) {
	for i := 1; i <= 370; i++ {
		d := after.UTC().AddDate(0, 0, i)
		open, err := c.IsTradingSession(d)
		if err != nil {
			return time.Time{}, err
		}
		if open {
			return d, nil
		}
	}
	return time.Time{}, fmt.Errorf("no next tradable session in calendar")
}

type ReviewSchedule struct {
	PositionID string
	Sessions   []time.Time
	Cadence    string
}

func BuildReviewSchedule(p Position, c SessionCalendar) (ReviewSchedule, error) {
	sessions, err := c.ReviewSessions(p.EntryAt)
	if err != nil {
		return ReviewSchedule{}, err
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Before(sessions[j]) })
	return ReviewSchedule{PositionID: p.PositionID, Sessions: sessions, Cadence: "one_review_per_trading_session"}, nil
}

type Diagnostic struct {
	At         time.Time `json:"at"`
	Stage      string    `json:"stage"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	Provenance []string  `json:"provenance"`
	Mode       string    `json:"mode"`
}

type PolicyVersions struct {
	TraderModel     string `json:"traderModel"`
	ThesisContract  string `json:"thesisContract"`
	CandidatePolicy string `json:"candidatePolicy"`
	RiskPolicy      string `json:"riskPolicy"`
	EntryPolicy     string `json:"entryPolicy"`
	ExitPolicy      string `json:"exitPolicy"`
	CostModel       string `json:"costModel"`
}

func (p PolicyVersions) Validate() error {
	if p.TraderModel == "" || p.ThesisContract == "" || p.CandidatePolicy == "" || p.RiskPolicy == "" || p.EntryPolicy == "" || p.ExitPolicy == "" || p.CostModel == "" {
		return fmt.Errorf("all exploratory paper policy versions must be bound")
	}
	return nil
}
