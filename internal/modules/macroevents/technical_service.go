package macroevents

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type technicalStore interface {
	SaveTechnicalAnalysisSnapshot(ctx context.Context, snapshot TechnicalSnapshot) (TechnicalSnapshot, error)
}

type TechnicalService struct {
	store    technicalStore
	provider CandleProvider
}

func NewTechnicalService(store technicalStore) *TechnicalService {
	return &TechnicalService{store: store}
}

func NewTechnicalServiceWithProvider(provider CandleProvider, store technicalStore) *TechnicalService {
	return &TechnicalService{store: store, provider: provider}
}

type TechnicalEngineRequest struct {
	MacroEventID    string
	Symbol          string
	EventType       EventType
	Direction       Direction
	EventTimeUTC    time.Time
	Timeframe       ReactionTimeframe
	BenchmarkSymbol string
	HasStopLevel    bool
	RewardRisk      float64
}

func (s *TechnicalService) EvaluateAndSave(ctx context.Context, input TechnicalInput) (TechnicalSnapshot, error) {
	snapshot := EvaluateTechnicalSnapshot(input)
	if s.store == nil {
		return snapshot, nil
	}
	return s.store.SaveTechnicalAnalysisSnapshot(ctx, snapshot)
}

func (s *TechnicalService) BuildAndSave(ctx context.Context, req TechnicalEngineRequest) (TechnicalSnapshot, error) {
	if s.provider == nil {
		return TechnicalSnapshot{}, fmt.Errorf("technical service provider is required")
	}
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		return TechnicalSnapshot{}, fmt.Errorf("symbol is required")
	}
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = TimeframePostEvent15M
	}

	preFrom, preTo := reactionBounds(req.EventTimeUTC, TimeframePreEvent30M)
	reactionFrom, reactionTo := reactionBounds(req.EventTimeUTC, timeframe)
	preCandles, err := s.provider.Candles(ctx, symbol, preFrom, preTo)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("load pre-event candles for %s: %w", symbol, err)
	}
	reactionCandles, err := s.provider.Candles(ctx, symbol, reactionFrom, reactionTo)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("load reaction candles for %s: %w", symbol, err)
	}
	allCandles := normalizeTechnicalCandles(append(append([]marketdata.Candle{}, preCandles...), reactionCandles...))

	bias := technicalBiasFromDirection(req.Direction)
	benchmarkSymbol := strings.ToUpper(strings.TrimSpace(req.BenchmarkSymbol))
	if benchmarkSymbol == "" {
		benchmarkSymbol = "SPY"
	}
	relativeStrength := TechnicalRelativeStrength{BenchmarkSymbol: benchmarkSymbol}
	if benchmarkSymbol != symbol {
		benchPre, err := s.provider.Candles(ctx, benchmarkSymbol, preFrom, preTo)
		if err != nil {
			return TechnicalSnapshot{}, fmt.Errorf("load benchmark pre-event candles for %s: %w", benchmarkSymbol, err)
		}
		benchPost, err := s.provider.Candles(ctx, benchmarkSymbol, reactionFrom, reactionTo)
		if err != nil {
			return TechnicalSnapshot{}, fmt.Errorf("load benchmark reaction candles for %s: %w", benchmarkSymbol, err)
		}
		relativeStrength = deriveRelativeStrength(allCandles, append(append([]marketdata.Candle{}, benchPre...), benchPost...), benchmarkSymbol, bias)
	}

	eventReaction, keyLevels := deriveEventReaction(allCandles, req.EventTimeUTC, bias, req.EventType, req.Direction, timeframe, req.MacroEventID, symbol)
	volumeVolatility := deriveVolumeVolatility(allCandles, req.EventTimeUTC)
	trendState, structureState := deriveTrendAndStructure(allCandles, req.EventTimeUTC, keyLevels)

	return s.EvaluateAndSave(ctx, TechnicalInput{
		MacroEventID:     req.MacroEventID,
		Symbol:           symbol,
		Timeframe:        timeframe,
		AnalysisTimeUTC:  time.Now().UTC(),
		TrendState:       trendState,
		StructureState:   structureState,
		Bias:             bias,
		KeyLevels:        keyLevels,
		EventReaction:    eventReaction,
		VolumeVolatility: volumeVolatility,
		RelativeStrength: relativeStrength,
		Candles:          allCandles,
		HasStopLevel:     req.HasStopLevel,
		RewardRisk:       req.RewardRisk,
	})
}

func technicalBiasFromDirection(direction Direction) TechnicalBias {
	switch Direction(strings.ToLower(strings.TrimSpace(string(direction)))) {
	case DirectionInflationHot, DirectionHawkishRates, DirectionGrowthStrong, DirectionRiskOff:
		return TechnicalBiasBearish
	case DirectionInflationCool, DirectionDovishRates, DirectionGrowthWeak, DirectionRiskOn:
		return TechnicalBiasBullish
	default:
		return TechnicalBiasNeutral
	}
}

func deriveEventReaction(candles []marketdata.Candle, eventTime time.Time, bias TechnicalBias, eventType EventType, direction Direction, timeframe ReactionTimeframe, macroEventID, symbol string) (TechnicalEventReaction, map[string]float64) {
	candles = normalizeTechnicalCandles(candles)
	pre, post := splitPrePostCandles(candles, eventTime)
	levels := map[string]float64{}
	reaction := TechnicalEventReaction{}
	if len(pre) == 0 || len(post) == 0 {
		return reaction, levels
	}

	preHigh, preLow := highLow(pre)
	levels["pre_event_high"] = preHigh
	levels["pre_event_low"] = preLow
	vwap, hasVWAP := computeVWAP(post)
	if hasVWAP {
		levels["vwap"] = vwap
	}

	last := post[len(post)-1]
	reaction.BreaksPreEventRange = last.Close > preHigh || last.Close < preLow
	reaction.ConfirmationPresent = len(post) >= 2
	if hasVWAP {
		switch bias {
		case TechnicalBiasBullish:
			reaction.VWAPHold = last.Close >= vwap
			if reaction.BreaksPreEventRange {
				reaction.VWAPReject = !reaction.VWAPHold
			}
		case TechnicalBiasBearish:
			reaction.VWAPReject = last.Close <= vwap
			if reaction.BreaksPreEventRange {
				reaction.VWAPHold = !reaction.VWAPReject
			}
		default:
			reaction.VWAPHold = last.Close >= vwap
			reaction.VWAPReject = last.Close <= vwap
		}
	}

	reactionSnapshot := EvaluateReaction(ReactionInput{
		MacroEventID: macroEventID,
		Symbol:       symbol,
		Timeframe:    timeframe,
		EventType:    eventType,
		Direction:    direction,
		EventTimeUTC: eventTime,
		Candles:      candles,
	})
	reaction.TooExtended = reactionSnapshot.TooExtended
	reaction.Whipsaw = reactionSnapshot.Noisy || reactionSnapshot.Direction == ReactionDirectionWhipsaw

	return reaction, levels
}

func deriveVolumeVolatility(candles []marketdata.Candle, eventTime time.Time) TechnicalVolumeVolatility {
	pre, post := splitPrePostCandles(candles, eventTime)
	preVol := averageVolume(pre)
	postVol := averageVolume(post)
	preTR := averageTrueRange(pre)
	postTR := averageTrueRange(post)

	out := TechnicalVolumeVolatility{}
	if preVol > 0 {
		out.VolumeRatio = postVol / preVol
	}
	if preTR > 0 {
		out.ATRRatio = postTR / preTR
	}
	return out
}

func deriveRelativeStrength(primary []marketdata.Candle, benchmark []marketdata.Candle, benchmarkSymbol string, bias TechnicalBias) TechnicalRelativeStrength {
	benchmark = normalizeTechnicalCandles(benchmark)
	primary = normalizeTechnicalCandles(primary)
	out := TechnicalRelativeStrength{BenchmarkSymbol: benchmarkSymbol}
	if len(primary) < 2 || len(benchmark) < 2 {
		return out
	}
	primaryReturn := pctReturn(primary[0].Close, primary[len(primary)-1].Close)
	benchmarkReturn := pctReturn(benchmark[0].Close, benchmark[len(benchmark)-1].Close)
	out.SpreadToBenchmark = primaryReturn - benchmarkReturn
	if math.Signbit(primaryReturn) != math.Signbit(benchmarkReturn) && math.Abs(primaryReturn) > 0.001 && math.Abs(benchmarkReturn) > 0.001 {
		out.ConflictingBasket = true
	}
	switch bias {
	case TechnicalBiasBullish:
		out.AlignsWithScenario = out.SpreadToBenchmark >= 0
	case TechnicalBiasBearish:
		out.AlignsWithScenario = out.SpreadToBenchmark <= 0
	default:
		out.AlignsWithScenario = true
	}
	return out
}

func deriveTrendAndStructure(candles []marketdata.Candle, eventTime time.Time, levels map[string]float64) (string, string) {
	candles = normalizeTechnicalCandles(candles)
	if len(candles) < 2 {
		return "unknown", "range"
	}
	start := candles[0].Close
	end := candles[len(candles)-1].Close
	trend := "range"
	if end > start {
		trend = "uptrend"
	} else if end < start {
		trend = "downtrend"
	}

	structure := "range"
	if len(levels) > 0 {
		preHigh, hasHigh := levels["pre_event_high"]
		preLow, hasLow := levels["pre_event_low"]
		last := candles[len(candles)-1].Close
		if hasHigh && last > preHigh {
			structure = "breakout"
		}
		if hasLow && last < preLow {
			structure = "breakdown"
		}
	}
	_, post := splitPrePostCandles(candles, eventTime)
	if len(post) >= 2 {
		if post[len(post)-1].Close > post[0].Close && structure == "range" {
			structure = "retest_hold"
		}
		if post[len(post)-1].Close < post[0].Close && structure == "range" {
			structure = "retest_reject"
		}
	}
	return trend, structure
}

func splitPrePostCandles(candles []marketdata.Candle, eventTime time.Time) ([]marketdata.Candle, []marketdata.Candle) {
	pre := make([]marketdata.Candle, 0, len(candles))
	post := make([]marketdata.Candle, 0, len(candles))
	for _, candle := range normalizeTechnicalCandles(candles) {
		if candle.Timestamp.Before(eventTime) {
			pre = append(pre, candle)
		} else {
			post = append(post, candle)
		}
	}
	return pre, post
}

func highLow(candles []marketdata.Candle) (float64, float64) {
	high := candles[0].High
	low := candles[0].Low
	for _, candle := range candles[1:] {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
	}
	return high, low
}

func computeVWAP(candles []marketdata.Candle) (float64, bool) {
	var totalPV float64
	var totalVolume float64
	for _, candle := range candles {
		if candle.Volume <= 0 {
			continue
		}
		typical := (candle.High + candle.Low + candle.Close) / 3
		totalPV += typical * float64(candle.Volume)
		totalVolume += float64(candle.Volume)
	}
	if totalVolume == 0 {
		return 0, false
	}
	return totalPV / totalVolume, true
}

func averageVolume(candles []marketdata.Candle) float64 {
	if len(candles) == 0 {
		return 0
	}
	var total float64
	for _, candle := range candles {
		total += float64(candle.Volume)
	}
	return total / float64(len(candles))
}

func averageTrueRange(candles []marketdata.Candle) float64 {
	if len(candles) == 0 {
		return 0
	}
	var total float64
	for _, candle := range candles {
		total += candle.High - candle.Low
	}
	return total / float64(len(candles))
}

func pctReturn(start, end float64) float64 {
	if start == 0 {
		return 0
	}
	return (end - start) / start
}
