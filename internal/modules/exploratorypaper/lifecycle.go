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
// silently approximated with calendar-day arithmetic or caller-provided flags.
type SessionCalendar struct {
	Sessions  map[string]bool `json:"sessions"`
	Timezone  string          `json:"timezone"`
	OpenTime  string          `json:"openTime"`
	CloseTime string          `json:"closeTime"`
}

func (c SessionCalendar) Validate() error {
	if len(c.Sessions) == 0 || c.Timezone == "" {
		return fmt.Errorf("trading session calendar is unknown")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("trading session timezone is unsupported: %w", err)
	}
	if _, _, err := c.sessionWindow(); err != nil {
		return err
	}
	return nil
}

func (c SessionCalendar) location() (*time.Location, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return nil, err
	}
	return location, nil
}

func (c SessionCalendar) sessionWindow() (time.Duration, time.Duration, error) {
	open, err := time.Parse("15:04", c.OpenTime)
	if err != nil {
		return 0, 0, fmt.Errorf("session open time must be HH:MM: %w", err)
	}
	close, err := time.Parse("15:04", c.CloseTime)
	if err != nil || !close.After(open) {
		return 0, 0, fmt.Errorf("session close time must be after open time")
	}
	return time.Duration(open.Hour())*time.Hour + time.Duration(open.Minute())*time.Minute,
		time.Duration(close.Hour())*time.Hour + time.Duration(close.Minute())*time.Minute, nil
}

func (c SessionCalendar) dateKey(day time.Time) string {
	location, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return ""
	}
	return day.In(location).Format("2006-01-02")
}

func (c SessionCalendar) SessionState(at time.Time) (MarketSession, error) {
	location, err := c.location()
	if err != nil || at.IsZero() || at.Location() != time.UTC {
		if err != nil {
			return SessionUnknown, err
		}
		return SessionUnknown, fmt.Errorf("session timestamp must be UTC")
	}
	key := at.In(location).Format("2006-01-02")
	open, known := c.Sessions[key]
	if !known {
		return SessionUnknown, fmt.Errorf("trading session %s is unknown", key)
	}
	if !open {
		return SessionClosed, nil
	}
	openOffset, closeOffset, err := c.sessionWindow()
	if err != nil {
		return SessionUnknown, err
	}
	local := at.In(location)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	openAt := midnight.Add(openOffset)
	closeAt := midnight.Add(closeOffset)
	if !local.Before(openAt) && local.Before(closeAt) {
		return SessionOpen, nil
	}
	return SessionClosed, nil
}

func (c SessionCalendar) sessionStart(day time.Time) (time.Time, error) {
	location, err := c.location()
	if err != nil {
		return time.Time{}, err
	}
	openOffset, _, err := c.sessionWindow()
	if err != nil {
		return time.Time{}, err
	}
	local := day.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location).Add(openOffset).UTC(), nil
}

func (c SessionCalendar) IsTradingSession(day time.Time) (bool, error) {
	if err := c.Validate(); err != nil {
		return false, err
	}
	v, ok := c.Sessions[c.dateKey(day)]
	if !ok {
		return false, fmt.Errorf("trading session %s is unknown", c.dateKey(day))
	}
	return v, nil
}

func (c SessionCalendar) TradingSessionsBetween(start, end time.Time) ([]time.Time, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return nil, fmt.Errorf("invalid session interval")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	var out []time.Time
	location, _ := c.location()
	startLocal, endLocal := start.In(location), end.In(location)
	for d := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, location); !d.After(time.Date(endLocal.Year(), endLocal.Month(), endLocal.Day(), 0, 0, 0, 0, location)); d = d.AddDate(0, 0, 1) {
		open, err := c.IsTradingSession(d)
		if err != nil {
			return nil, err
		}
		if open {
			out = append(out, d.UTC())
		}
	}
	return out, nil
}

func (c SessionCalendar) HardExitDate(entryAt time.Time) (time.Time, error) {
	if err := c.Validate(); err != nil {
		return time.Time{}, err
	}
	if state, err := c.SessionState(entryAt); err != nil || state != SessionOpen {
		if err != nil {
			return time.Time{}, err
		}
		return time.Time{}, fmt.Errorf("entry must occur during a known tradable session")
	}
	location, _ := c.location()
	var sessions []time.Time
	for offset := 0; offset <= 370 && len(sessions) < 5; offset++ {
		localEntry := entryAt.In(location)
		d := time.Date(localEntry.Year(), localEntry.Month(), localEntry.Day(), 0, 0, 0, 0, location).AddDate(0, 0, offset)
		open, err := c.IsTradingSession(d)
		if err != nil {
			return time.Time{}, err
		}
		if open {
			start, err := c.sessionStart(d)
			if err != nil {
				return time.Time{}, err
			}
			sessions = append(sessions, start)
		}
	}
	if len(sessions) == 0 || sessions[0].In(location).Format("2006-01-02") != c.dateKey(entryAt) {
		return time.Time{}, fmt.Errorf("entry is not on a known trading session")
	}
	if len(sessions) < 5 {
		return time.Time{}, fmt.Errorf("calendar does not cover the fifth trading session")
	}
	return sessions[4], nil
}

func (c SessionCalendar) ReviewSessions(entryAt time.Time) ([]time.Time, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	state, err := c.SessionState(entryAt)
	if err != nil || state != SessionOpen {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("entry must occur during a known tradable session")
	}
	last, err := c.HardExitDate(entryAt)
	if err != nil {
		return nil, err
	}
	location, _ := c.location()
	var sessions []time.Time
	for offset := 0; offset <= 370 && len(sessions) < 5; offset++ {
		localEntry := entryAt.In(location)
		d := time.Date(localEntry.Year(), localEntry.Month(), localEntry.Day(), 0, 0, 0, 0, location).AddDate(0, 0, offset)
		open, err := c.IsTradingSession(d)
		if err != nil {
			return nil, err
		}
		if open {
			start, err := c.sessionStart(d)
			if err != nil {
				return nil, err
			}
			sessions = append(sessions, start)
		}
	}
	if len(sessions) != 5 || sessions[4] != last {
		return nil, fmt.Errorf("calendar does not cover five trading-session review points")
	}
	return sessions, nil
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
	if err := in.Calendar.Validate(); err != nil || in.Now.IsZero() || in.Now.Location() != time.UTC {
		if err != nil {
			return ExitDecision{}, err
		}
		return ExitDecision{}, fmt.Errorf("exit evaluation requires a UTC timestamp")
	}
	knownSession, err := in.Calendar.SessionState(in.Now)
	if err != nil || in.Session == SessionUnknown || in.Session != knownSession {
		if err != nil {
			return ExitDecision{}, err
		}
		return ExitDecision{}, fmt.Errorf("caller session disagrees with explicit calendar")
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
	if reason == "" && !in.Now.Before(hard) {
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
			return c.sessionStart(d)
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
