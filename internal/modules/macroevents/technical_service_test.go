package macroevents

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

func TestTechnicalServiceEvaluatesAndPersistsSnapshot(t *testing.T) {
	store := &fakeTechnicalStore{}
	service := NewTechnicalService(store)
	eventTime := technicalTestEventTime()

	snapshot, err := service.EvaluateAndSave(context.Background(), TechnicalInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		Timeframe:       TimeframePostEvent15M,
		AnalysisTimeUTC: eventTime,
		TrendState:      "downtrend",
		StructureState:  "breakdown",
		Bias:            TechnicalBiasBearish,
		KeyLevels:       map[string]float64{"pre_event_low": 398, "vwap": 399},
		EventReaction:   TechnicalEventReaction{BreaksPreEventRange: true, ConfirmationPresent: true, VWAPReject: true},
		VolumeVolatility: TechnicalVolumeVolatility{
			VolumeRatio: 1.25,
			ATRRatio:    1.2,
		},
		RelativeStrength: TechnicalRelativeStrength{BenchmarkSymbol: "SPY", SpreadToBenchmark: -0.002, AlignsWithScenario: true},
		HasStopLevel:     true,
		RewardRisk:       1.8,
		Candles: []marketdata.Candle{
			candle("QQQ", eventTime.Add(-5*time.Minute), 400, 401, 399, 400, 1000),
			candle("QQQ", eventTime.Add(15*time.Minute), 400, 400, 397, 397.5, 1800),
		},
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}

	if snapshot.ID == "" {
		t.Fatal("expected persisted snapshot id")
	}
	if len(store.snapshots) != 1 {
		t.Fatalf("saved snapshots = %d, want 1", len(store.snapshots))
	}
}

func TestTechnicalServiceWithoutStoreReturnsEvaluatedSnapshot(t *testing.T) {
	service := NewTechnicalService(nil)

	snapshot, err := service.EvaluateAndSave(context.Background(), TechnicalInput{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Timeframe:    TimeframePostEvent15M,
		Bias:         TechnicalBiasBearish,
		Candles:      nil,
	})
	if err != nil {
		t.Fatalf("EvaluateAndSave returned error: %v", err)
	}
	if snapshot.Verdict != TechnicalVerdictInsufficientData {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictInsufficientData)
	}
}

func TestTechnicalServiceBuildAndSaveDetectsRangeBreakVWAPAndRelativeStrength(t *testing.T) {
	eventTime := technicalTestEventTime()
	provider := &fakeTechnicalCandleProvider{
		candles: map[string][]marketdata.Candle{
			"QQQ": {
				candle("QQQ", eventTime.Add(-25*time.Minute), 400, 401, 399.2, 400.5, 900),
				candle("QQQ", eventTime.Add(-15*time.Minute), 400.5, 401.2, 399.8, 400.8, 950),
				candle("QQQ", eventTime.Add(-5*time.Minute), 400.8, 401.0, 400.0, 400.6, 1000),
				candle("QQQ", eventTime.Add(5*time.Minute), 400.6, 400.7, 397.5, 398.1, 1800),
				candle("QQQ", eventTime.Add(15*time.Minute), 398.1, 398.4, 396.9, 397.2, 2200),
			},
			"SPY": {
				candle("SPY", eventTime.Add(-25*time.Minute), 500, 500.7, 499.6, 500.3, 1500),
				candle("SPY", eventTime.Add(-15*time.Minute), 500.3, 500.8, 499.9, 500.2, 1450),
				candle("SPY", eventTime.Add(-5*time.Minute), 500.2, 500.6, 499.8, 500.1, 1400),
				candle("SPY", eventTime.Add(5*time.Minute), 500.1, 500.2, 498.9, 499.4, 1700),
				candle("SPY", eventTime.Add(15*time.Minute), 499.4, 499.5, 498.5, 499.0, 1650),
			},
		},
	}
	service := NewTechnicalServiceWithProvider(provider, nil)

	snapshot, err := service.BuildAndSave(context.Background(), TechnicalEngineRequest{
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Timeframe:    TimeframePostEvent15M,
		HasStopLevel: true,
		RewardRisk:   1.8,
	})
	if err != nil {
		t.Fatalf("BuildAndSave returned error: %v", err)
	}

	if !snapshot.EventReaction.BreaksPreEventRange {
		t.Fatalf("expected pre-event range break, snapshot=%#v", snapshot.EventReaction)
	}
	if !snapshot.EventReaction.VWAPReject {
		t.Fatalf("expected VWAP reject for bearish setup, snapshot=%#v", snapshot.EventReaction)
	}
	if snapshot.VolumeVolatility.VolumeRatio <= 1.0 {
		t.Fatalf("volume ratio = %v, want > 1.0", snapshot.VolumeVolatility.VolumeRatio)
	}
	if snapshot.VolumeVolatility.ATRRatio <= 1.0 {
		t.Fatalf("atr ratio = %v, want > 1.0", snapshot.VolumeVolatility.ATRRatio)
	}
	if snapshot.RelativeStrength.BenchmarkSymbol != "SPY" {
		t.Fatalf("benchmark = %q, want SPY", snapshot.RelativeStrength.BenchmarkSymbol)
	}
	if snapshot.RelativeStrength.SpreadToBenchmark >= 0 {
		t.Fatalf("relative spread = %v, want < 0 for bearish outperformance", snapshot.RelativeStrength.SpreadToBenchmark)
	}
}

func TestTechnicalServiceBuildAndSaveFlagsTooExtendedAndWhipsaw(t *testing.T) {
	eventTime := technicalTestEventTime()
	t.Run("too_extended", func(t *testing.T) {
		provider := &fakeTechnicalCandleProvider{
			candles: map[string][]marketdata.Candle{
				"QQQ": {
					candle("QQQ", eventTime.Add(-20*time.Minute), 100, 101, 99.5, 100, 900),
					candle("QQQ", eventTime.Add(-5*time.Minute), 100, 100.5, 99.7, 100, 950),
					candle("QQQ", eventTime.Add(5*time.Minute), 100, 100.2, 96.8, 97.2, 2500),
					candle("QQQ", eventTime.Add(15*time.Minute), 97.2, 97.4, 96.5, 96.9, 2700),
				},
				"SPY": {
					candle("SPY", eventTime.Add(-20*time.Minute), 500, 501, 499.5, 500, 1200),
					candle("SPY", eventTime.Add(-5*time.Minute), 500, 500.5, 499.8, 500, 1250),
					candle("SPY", eventTime.Add(5*time.Minute), 500, 500.1, 498.9, 499.2, 1600),
					candle("SPY", eventTime.Add(15*time.Minute), 499.2, 499.4, 498.7, 499.0, 1500),
				},
			},
		}
		service := NewTechnicalServiceWithProvider(provider, nil)
		snapshot, err := service.BuildAndSave(context.Background(), TechnicalEngineRequest{
			MacroEventID: "macro-2",
			Symbol:       "QQQ",
			EventType:    EventTypeUSCPIHeadline,
			Direction:    DirectionInflationHot,
			EventTimeUTC: eventTime,
			Timeframe:    TimeframePostEvent15M,
			HasStopLevel: true,
			RewardRisk:   2.0,
		})
		if err != nil {
			t.Fatalf("BuildAndSave returned error: %v", err)
		}
		if snapshot.Verdict != TechnicalVerdictTooExtended {
			t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictTooExtended)
		}
	})

	t.Run("whipsaw", func(t *testing.T) {
		provider := &fakeTechnicalCandleProvider{
			candles: map[string][]marketdata.Candle{
				"QQQ": {
					candle("QQQ", eventTime.Add(-20*time.Minute), 100, 101, 99.2, 100, 900),
					candle("QQQ", eventTime.Add(-5*time.Minute), 100, 100.8, 99.6, 100, 950),
					candle("QQQ", eventTime.Add(5*time.Minute), 100, 104.2, 96.4, 103.2, 4000),
					candle("QQQ", eventTime.Add(15*time.Minute), 103.2, 103.5, 96.2, 99.1, 4200),
				},
				"SPY": {
					candle("SPY", eventTime.Add(-20*time.Minute), 500, 501, 499.2, 500, 1200),
					candle("SPY", eventTime.Add(-5*time.Minute), 500, 500.8, 499.5, 500, 1250),
					candle("SPY", eventTime.Add(5*time.Minute), 500, 502, 497.2, 499.1, 1800),
					candle("SPY", eventTime.Add(15*time.Minute), 499.1, 500.1, 497.8, 498.8, 1750),
				},
			},
		}
		service := NewTechnicalServiceWithProvider(provider, nil)
		snapshot, err := service.BuildAndSave(context.Background(), TechnicalEngineRequest{
			MacroEventID: "macro-3",
			Symbol:       "QQQ",
			EventType:    EventTypeUSCPIHeadline,
			Direction:    DirectionInflationHot,
			EventTimeUTC: eventTime,
			Timeframe:    TimeframePostEvent15M,
			HasStopLevel: true,
			RewardRisk:   2.0,
		})
		if err != nil {
			t.Fatalf("BuildAndSave returned error: %v", err)
		}
		if snapshot.Verdict != TechnicalVerdictWhipsaw {
			t.Fatalf("verdict = %q, want %q", snapshot.Verdict, TechnicalVerdictWhipsaw)
		}
	})
}

type fakeTechnicalStore struct {
	snapshots []TechnicalSnapshot
}

func (s *fakeTechnicalStore) SaveTechnicalAnalysisSnapshot(_ context.Context, snapshot TechnicalSnapshot) (TechnicalSnapshot, error) {
	snapshot.ID = "ta-1"
	s.snapshots = append(s.snapshots, snapshot)
	return snapshot, nil
}

type fakeTechnicalCandleProvider struct {
	candles map[string][]marketdata.Candle
}

func (p *fakeTechnicalCandleProvider) Candles(_ context.Context, symbol string, from time.Time, to time.Time) ([]marketdata.Candle, error) {
	pool := p.candles[symbol]
	out := make([]marketdata.Candle, 0, len(pool))
	for _, c := range pool {
		if (c.Timestamp.Equal(from) || c.Timestamp.After(from)) && (c.Timestamp.Equal(to) || c.Timestamp.Before(to)) {
			out = append(out, c)
		}
	}
	return out, nil
}
