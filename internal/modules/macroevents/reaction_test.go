package macroevents

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

func TestEvaluateReactionHotCPIQQQDownConfirmsEvent(t *testing.T) {
	eventTime := reactionTestEventTime()
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 100, 101, 99, 100, 1000),
			candle("QQQ", eventTime.Add(15*time.Minute), 100, 100, 98, 99.2, 1200),
		},
	})

	if result.Status != ReactionStatusAvailable {
		t.Fatalf("status = %q, want %q", result.Status, ReactionStatusAvailable)
	}
	if !result.ConfirmsEvent {
		t.Fatalf("expected confirmation, got %#v", result)
	}
	if result.Direction != ReactionDirectionDown {
		t.Fatalf("direction = %q, want %q", result.Direction, ReactionDirectionDown)
	}
}

func TestEvaluateReactionHotCPIQQQUpDoesNotConfirm(t *testing.T) {
	eventTime := reactionTestEventTime()
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 100, 101, 99, 100, 1000),
			candle("QQQ", eventTime.Add(15*time.Minute), 100, 102, 100, 101, 1200),
		},
	})

	if result.ConfirmsEvent {
		t.Fatalf("did not expect confirmation, got %#v", result)
	}
	if !strings.Contains(result.Reason, "opposite") {
		t.Fatalf("reason = %q, want opposite-direction reason", result.Reason)
	}
}

func TestEvaluateReactionTinyMoveDoesNotConfirm(t *testing.T) {
	eventTime := reactionTestEventTime()
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "SPY",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Candles: []marketdata.Candle{
			candle("SPY", eventTime.Add(-5*time.Minute), 400, 401, 399, 400, 1000),
			candle("SPY", eventTime.Add(15*time.Minute), 400, 401, 399, 399.6, 1000),
		},
	})

	if result.ConfirmsEvent {
		t.Fatalf("did not expect tiny move to confirm, got %#v", result)
	}
	if !strings.Contains(result.Reason, "below confirmation threshold") {
		t.Fatalf("reason = %q, want threshold reason", result.Reason)
	}
}

func TestEvaluateReactionHugeMoveMarksTooExtended(t *testing.T) {
	eventTime := reactionTestEventTime()
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 100, 101, 99, 100, 1000),
			candle("QQQ", eventTime.Add(15*time.Minute), 100, 100, 95, 96.8, 2000),
		},
	})

	if !result.TooExtended {
		t.Fatalf("expected too_extended, got %#v", result)
	}
	if result.ConfirmsEvent {
		t.Fatalf("too-extended move must not confirm candidate flow, got %#v", result)
	}
}

func TestEvaluateReactionMissingCandlesUnavailable(t *testing.T) {
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: reactionTestEventTime(),
		Candles:      nil,
	})

	if result.Status != ReactionStatusUnavailable {
		t.Fatalf("status = %q, want %q", result.Status, ReactionStatusUnavailable)
	}
	if result.ConfirmsEvent {
		t.Fatalf("missing candles must not confirm, got %#v", result)
	}
}

func TestEvaluateReactionWhipsawCandlesMarkNoisy(t *testing.T) {
	eventTime := reactionTestEventTime()
	result := EvaluateReaction(ReactionInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 100, 101, 99, 100, 1000),
			candle("QQQ", eventTime.Add(5*time.Minute), 100, 104, 96, 103, 4000),
			candle("QQQ", eventTime.Add(15*time.Minute), 103, 104, 96, 99.1, 4500),
		},
	})

	if !result.Noisy {
		t.Fatalf("expected noisy whipsaw, got %#v", result)
	}
	if result.Direction != ReactionDirectionWhipsaw {
		t.Fatalf("direction = %q, want whipsaw", result.Direction)
	}
	if result.ConfirmsEvent {
		t.Fatalf("noisy reaction must not confirm, got %#v", result)
	}
}

func reactionTestEventTime() time.Time {
	return time.Date(2026, 6, 10, 13, 30, 0, 0, time.UTC)
}

func candle(symbol string, ts time.Time, open, high, low, close float64, volume int64) marketdata.Candle {
	return marketdata.Candle{
		Symbol:    symbol,
		Timestamp: ts,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
}
