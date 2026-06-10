package macroevents

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

func TestEvaluateTechnicalSnapshotReturnsInsufficientDataOnMissingCandles(t *testing.T) {
	snapshot := EvaluateTechnicalSnapshot(TechnicalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		Bias:         TechnicalBiasBearish,
		Candles:      nil,
	})

	if snapshot.Verdict != TechnicalVerdictInsufficientData {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictInsufficientData)
	}
	if snapshot.TechnicalScore != 0 {
		t.Fatalf("score = %v, want 0", snapshot.TechnicalScore)
	}
}

func TestEvaluateTechnicalSnapshotComputesScoreAndConfirmedBearish(t *testing.T) {
	eventTime := technicalTestEventTime()
	snapshot := EvaluateTechnicalSnapshot(TechnicalInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		Timeframe:       TimeframePostEvent15M,
		AnalysisTimeUTC: eventTime,
		TrendState:      "downtrend",
		StructureState:  "breakdown",
		Bias:            TechnicalBiasBearish,
		KeyLevels: map[string]float64{
			"pre_event_high": 401,
			"pre_event_low":  398,
			"vwap":           399,
		},
		EventReaction: TechnicalEventReaction{
			BreaksPreEventRange: true,
			ConfirmationPresent: true,
			VWAPReject:          true,
		},
		VolumeVolatility: TechnicalVolumeVolatility{VolumeRatio: 1.35, ATRRatio: 1.25},
		RelativeStrength: TechnicalRelativeStrength{BenchmarkSymbol: "SPY", SpreadToBenchmark: -0.0022, AlignsWithScenario: true},
		HasStopLevel:     true,
		RewardRisk:       1.9,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 400, 401, 399, 400, 1000),
			candle("QQQ", eventTime.Add(15*time.Minute), 400, 400, 397, 397.6, 1800),
		},
	})

	if snapshot.Verdict != TechnicalVerdictConfirmedBearish {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictConfirmedBearish)
	}
	if snapshot.TechnicalScore < 75 {
		t.Fatalf("score = %v, want >= 75", snapshot.TechnicalScore)
	}
	if len(snapshot.Reasons) == 0 {
		t.Fatal("expected non-empty reasons")
	}
}

func TestEvaluateTechnicalSnapshotFlagsTooExtended(t *testing.T) {
	snapshot := EvaluateTechnicalSnapshot(TechnicalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		Bias:         TechnicalBiasBearish,
		EventReaction: TechnicalEventReaction{
			TooExtended: true,
		},
		Candles: []marketdata.Candle{
			candle("QQQ", technicalTestEventTime().Add(-5*time.Minute), 100, 101, 99, 100, 1000),
			candle("QQQ", technicalTestEventTime().Add(15*time.Minute), 100, 100, 95, 96, 3000),
		},
	})

	if snapshot.Verdict != TechnicalVerdictTooExtended {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictTooExtended)
	}
	if !strings.Contains(strings.Join(snapshot.Reasons, " "), "extended") {
		t.Fatalf("reasons = %#v, want extended reason", snapshot.Reasons)
	}
}

func TestEvaluateTechnicalSnapshotMissingStopFailsSafely(t *testing.T) {
	snapshot := EvaluateTechnicalSnapshot(TechnicalInput{
		MacroEventID:   "macro-1",
		Symbol:         "TLT",
		Timeframe:      TimeframePostEvent15M,
		TrendState:     "uptrend",
		StructureState: "breakout",
		Bias:           TechnicalBiasBullish,
		EventReaction: TechnicalEventReaction{
			BreaksPreEventRange: true,
			ConfirmationPresent: true,
			VWAPHold:            true,
		},
		VolumeVolatility: TechnicalVolumeVolatility{VolumeRatio: 1.1, ATRRatio: 1.15},
		RelativeStrength: TechnicalRelativeStrength{BenchmarkSymbol: "SPY", SpreadToBenchmark: 0.0015, AlignsWithScenario: true},
		HasStopLevel:     false,
		RewardRisk:       2.0,
		Candles: []marketdata.Candle{
			candle("TLT", technicalTestEventTime().Add(-5*time.Minute), 90, 91, 89, 90, 1000),
			candle("TLT", technicalTestEventTime().Add(15*time.Minute), 90, 92, 89.5, 91.5, 1500),
		},
	})

	if snapshot.Verdict != TechnicalVerdictNoConfirmation {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictNoConfirmation)
	}
	if len(snapshot.Reasons) == 0 || !strings.Contains(strings.Join(snapshot.Reasons, " "), "stop") {
		t.Fatalf("reasons = %#v, want stop-level reason", snapshot.Reasons)
	}
}

func technicalTestEventTime() time.Time {
	return time.Date(2026, 6, 10, 13, 30, 0, 0, time.UTC)
}
