package macroevents

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type ReactionDirection string
type ReactionStatus string
type ReactionTimeframe string

const (
	ReactionDirectionUp      ReactionDirection = "up"
	ReactionDirectionDown    ReactionDirection = "down"
	ReactionDirectionFlat    ReactionDirection = "flat"
	ReactionDirectionWhipsaw ReactionDirection = "whipsaw"
	ReactionDirectionUnknown ReactionDirection = "unknown"
)

const (
	ReactionStatusAvailable   ReactionStatus = "available"
	ReactionStatusUnavailable ReactionStatus = "unavailable"
)

const (
	TimeframePreEvent30M  ReactionTimeframe = "pre_event_30m"
	TimeframePreEvent5M   ReactionTimeframe = "pre_event_5m"
	TimeframePostEvent5M  ReactionTimeframe = "post_event_5m"
	TimeframePostEvent15M ReactionTimeframe = "post_event_15m"
	TimeframePostEvent30M ReactionTimeframe = "post_event_30m"
	TimeframePostEvent60M ReactionTimeframe = "post_event_60m"
	TimeframeSessionToNow ReactionTimeframe = "session_to_now"
)

type ReactionInput struct {
	MacroEventID string
	Symbol       string
	Timeframe    ReactionTimeframe
	EventType    EventType
	Direction    Direction
	EventTimeUTC time.Time
	Candles      []marketdata.Candle
}

type ReactionSnapshot struct {
	ID            string
	MacroEventID  string
	Symbol        string
	Timeframe     ReactionTimeframe
	PrePrice      float64
	PostPrice     float64
	ChangeAbs     float64
	ChangePercent float64
	HighAfter     *float64
	LowAfter      *float64
	VolumeRatio   *float64
	ATRRatio      *float64
	Direction     ReactionDirection
	ConfirmsEvent bool
	TooExtended   bool
	Noisy         bool
	Reason        string
	RawCandles    []marketdata.Candle
	Status        ReactionStatus
}

func EvaluateReaction(input ReactionInput) ReactionSnapshot {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	candles := normalizeCandles(input.Candles)
	if len(candles) < 2 {
		return ReactionSnapshot{
			MacroEventID: input.MacroEventID,
			Symbol:       symbol,
			Timeframe:    input.Timeframe,
			Direction:    ReactionDirectionUnknown,
			Status:       ReactionStatusUnavailable,
			Reason:       "missing candles for reaction snapshot",
			RawCandles:   candles,
		}
	}

	pre := candles[0]
	post := candles[len(candles)-1]
	if pre.Close <= 0 {
		return ReactionSnapshot{
			MacroEventID: input.MacroEventID,
			Symbol:       symbol,
			Timeframe:    input.Timeframe,
			Direction:    ReactionDirectionUnknown,
			Status:       ReactionStatusUnavailable,
			Reason:       "pre-event candle close must be greater than zero",
			RawCandles:   candles,
		}
	}

	changeAbs := post.Close - pre.Close
	changePercent := changeAbs / pre.Close
	direction := directionFromChange(changePercent)
	highAfter, lowAfter := postEventRange(candles, input.EventTimeUTC)
	noisy := isNoisyReaction(candles, pre.Close)
	if noisy {
		direction = ReactionDirectionWhipsaw
	}

	minMove := confirmationThreshold(symbol)
	maxMove := maxEventMoveThreshold(symbol)
	expected := expectedReactionDirection(input.EventType, input.Direction, symbol)
	tooExtended := math.Abs(changePercent) >= maxMove
	confirms := expected != ReactionDirectionUnknown &&
		direction == expected &&
		math.Abs(changePercent) >= minMove &&
		!tooExtended &&
		!noisy

	reason := reactionReason(expected, direction, changePercent, minMove, tooExtended, noisy)
	return ReactionSnapshot{
		MacroEventID:  input.MacroEventID,
		Symbol:        symbol,
		Timeframe:     input.Timeframe,
		PrePrice:      pre.Close,
		PostPrice:     post.Close,
		ChangeAbs:     changeAbs,
		ChangePercent: changePercent,
		HighAfter:     highAfter,
		LowAfter:      lowAfter,
		Direction:     direction,
		ConfirmsEvent: confirms,
		TooExtended:   tooExtended,
		Noisy:         noisy,
		Reason:        reason,
		RawCandles:    candles,
		Status:        ReactionStatusAvailable,
	}
}

func normalizeCandles(candles []marketdata.Candle) []marketdata.Candle {
	out := append([]marketdata.Candle(nil), candles...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out
}

func directionFromChange(changePercent float64) ReactionDirection {
	switch {
	case changePercent > 0:
		return ReactionDirectionUp
	case changePercent < 0:
		return ReactionDirectionDown
	default:
		return ReactionDirectionFlat
	}
}

func postEventRange(candles []marketdata.Candle, eventTime time.Time) (*float64, *float64) {
	var high, low float64
	found := false
	for _, candle := range candles {
		if !eventTime.IsZero() && candle.Timestamp.Before(eventTime) {
			continue
		}
		if !found {
			high = candle.High
			low = candle.Low
			found = true
			continue
		}
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
	}
	if !found {
		return nil, nil
	}
	return &high, &low
}

func isNoisyReaction(candles []marketdata.Candle, prePrice float64) bool {
	if len(candles) < 3 || prePrice <= 0 {
		return false
	}
	seenUp := false
	seenDown := false
	for _, candle := range candles[1:] {
		if (candle.High-prePrice)/prePrice >= 0.02 {
			seenUp = true
		}
		if (prePrice-candle.Low)/prePrice >= 0.02 {
			seenDown = true
		}
	}
	return seenUp && seenDown
}

func confirmationThreshold(symbol string) float64 {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "QQQ":
		return 0.0035
	case "SPY", "TLT":
		return 0.0025
	default:
		return 0.003
	}
}

func maxEventMoveThreshold(symbol string) float64 {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "QQQ":
		return 0.025
	case "SPY":
		return 0.018
	case "TLT":
		return 0.015
	case "IWM":
		return 0.022
	default:
		return 0.02
	}
}

func expectedReactionDirection(eventType EventType, direction Direction, symbol string) ReactionDirection {
	switch Direction(strings.ToLower(strings.TrimSpace(string(direction)))) {
	case DirectionInflationHot, DirectionHawkishRates, DirectionGrowthStrong:
		switch strings.ToUpper(strings.TrimSpace(symbol)) {
		case "QQQ", "SPY", "TLT", "IWM", "DIA", "XLK", "SMH", "SOXX":
			return ReactionDirectionDown
		default:
			return ReactionDirectionUnknown
		}
	case DirectionInflationCool, DirectionDovishRates, DirectionGrowthWeak:
		switch strings.ToUpper(strings.TrimSpace(symbol)) {
		case "QQQ", "SPY", "TLT", "IWM", "DIA", "XLK", "SMH", "SOXX":
			return ReactionDirectionUp
		default:
			return ReactionDirectionUnknown
		}
	case DirectionRiskOn:
		return ReactionDirectionUp
	case DirectionRiskOff:
		return ReactionDirectionDown
	default:
		return fallbackExpectedDirection(eventType)
	}
}

func fallbackExpectedDirection(eventType EventType) ReactionDirection {
	switch EventType(strings.ToUpper(strings.TrimSpace(string(eventType)))) {
	case EventTypeUSCPIHeadline, EventTypeUSCPICore, EventTypeUSPPI:
		return ReactionDirectionDown
	default:
		return ReactionDirectionUnknown
	}
}

func reactionReason(expected, actual ReactionDirection, changePercent, minMove float64, tooExtended, noisy bool) string {
	if noisy {
		return "reaction is noisy or whipsaw"
	}
	if tooExtended {
		return "reaction move is too extended to chase"
	}
	if expected == ReactionDirectionUnknown {
		return "expected reaction direction is unknown"
	}
	if actual != expected {
		return fmt.Sprintf("reaction moved opposite expected direction: expected %s, got %s", expected, actual)
	}
	if math.Abs(changePercent) < minMove {
		return "reaction move below confirmation threshold"
	}
	return "reaction confirms expected macro direction"
}
