package macroevents

import (
	"math"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type TechnicalVerdict string

type TechnicalBias string

const (
	TechnicalVerdictConfirmedBullish TechnicalVerdict = "confirmed_bullish"
	TechnicalVerdictConfirmedBearish TechnicalVerdict = "confirmed_bearish"
	TechnicalVerdictWatchOnly        TechnicalVerdict = "watch_only"
	TechnicalVerdictNoConfirmation   TechnicalVerdict = "no_confirmation"
	TechnicalVerdictConflicting      TechnicalVerdict = "conflicting"
	TechnicalVerdictTooExtended      TechnicalVerdict = "too_extended"
	TechnicalVerdictWhipsaw          TechnicalVerdict = "whipsaw"
	TechnicalVerdictInsufficientData TechnicalVerdict = "insufficient_data"
)

const (
	TechnicalBiasBullish TechnicalBias = "bullish"
	TechnicalBiasBearish TechnicalBias = "bearish"
	TechnicalBiasNeutral TechnicalBias = "neutral"
)

type TechnicalEventReaction struct {
	BreaksPreEventRange bool
	ConfirmationPresent bool
	VWAPHold            bool
	VWAPReject          bool
	TooExtended         bool
	Whipsaw             bool
}

type TechnicalVolumeVolatility struct {
	VolumeRatio float64
	ATRRatio    float64
}

type TechnicalRelativeStrength struct {
	BenchmarkSymbol    string
	SpreadToBenchmark  float64
	AlignsWithScenario bool
	ConflictingBasket  bool
}

type TechnicalInput struct {
	MacroEventID     string
	Symbol           string
	Timeframe        ReactionTimeframe
	AnalysisTimeUTC  time.Time
	TrendState       string
	StructureState   string
	Bias             TechnicalBias
	KeyLevels        map[string]float64
	EventReaction    TechnicalEventReaction
	VolumeVolatility TechnicalVolumeVolatility
	RelativeStrength TechnicalRelativeStrength
	Candles          []marketdata.Candle
	HasStopLevel     bool
	RewardRisk       float64
}

type TechnicalSnapshot struct {
	ID                string
	MacroEventID      string
	Symbol            string
	AnalysisTimeUTC   time.Time
	Timeframe         ReactionTimeframe
	TrendState        string
	StructureState    string
	KeyLevels         map[string]float64
	EventReaction     TechnicalEventReaction
	VolumeVolatility  TechnicalVolumeVolatility
	RelativeStrength  TechnicalRelativeStrength
	TechnicalScore    float64
	Verdict           TechnicalVerdict
	Reasons           []string
	InvalidationRules []string
}

func EvaluateTechnicalSnapshot(input TechnicalInput) TechnicalSnapshot {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	candles := normalizeTechnicalCandles(input.Candles)
	snapshot := TechnicalSnapshot{
		MacroEventID:      strings.TrimSpace(input.MacroEventID),
		Symbol:            symbol,
		AnalysisTimeUTC:   analysisTime(input.AnalysisTimeUTC),
		Timeframe:         input.Timeframe,
		TrendState:        normalizeState(input.TrendState),
		StructureState:    normalizeState(input.StructureState),
		KeyLevels:         cloneLevels(input.KeyLevels),
		EventReaction:     input.EventReaction,
		VolumeVolatility:  input.VolumeVolatility,
		RelativeStrength:  input.RelativeStrength,
		Reasons:           []string{},
		InvalidationRules: []string{},
	}

	if len(candles) < 2 {
		snapshot.Verdict = TechnicalVerdictInsufficientData
		snapshot.Reasons = append(snapshot.Reasons, "missing candles for technical analysis")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "require at least two candles before candidate review")
		return snapshot
	}

	if input.EventReaction.Whipsaw {
		snapshot.Verdict = TechnicalVerdictWhipsaw
		snapshot.Reasons = append(snapshot.Reasons, "whipsaw reaction blocks technical confirmation")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "wait for post-event trend stabilization")
		return snapshot
	}
	if input.EventReaction.TooExtended {
		snapshot.Verdict = TechnicalVerdictTooExtended
		snapshot.Reasons = append(snapshot.Reasons, "price move is too extended for a controlled entry")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "wait for pullback toward event range or VWAP")
		return snapshot
	}
	if input.RelativeStrength.ConflictingBasket {
		snapshot.Verdict = TechnicalVerdictConflicting
		snapshot.Reasons = append(snapshot.Reasons, "conflicting ETF basket blocks technical confirmation")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "require alignment across mapped ETF basket")
		return snapshot
	}

	trendScore := scoreTrend(snapshot.TrendState)
	levelScore := scoreLevels(snapshot.StructureState, snapshot.KeyLevels, candles)
	eventReactionScore := scoreEventReaction(snapshot.EventReaction)
	volumeATRScore := scoreVolumeATR(snapshot.VolumeVolatility)
	relativeStrengthScore := scoreRelativeStrength(snapshot.RelativeStrength)
	entryStopScore := scoreEntryStop(input.HasStopLevel, input.RewardRisk)

	snapshot.TechnicalScore = trendScore + levelScore + eventReactionScore + volumeATRScore + relativeStrengthScore + entryStopScore
	snapshot.TechnicalScore = math.Round(snapshot.TechnicalScore*100) / 100

	if !input.HasStopLevel {
		snapshot.Verdict = TechnicalVerdictNoConfirmation
		snapshot.Reasons = append(snapshot.Reasons, "stop level is missing")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "define a technical invalidation level")
		return snapshot
	}
	if input.RewardRisk < 1.2 {
		snapshot.Verdict = TechnicalVerdictNoConfirmation
		snapshot.Reasons = append(snapshot.Reasons, "reward:risk below minimum threshold")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "require reward:risk at or above 1.2")
		return snapshot
	}
	if !input.EventReaction.ConfirmationPresent {
		snapshot.Verdict = TechnicalVerdictNoConfirmation
		snapshot.Reasons = append(snapshot.Reasons, "event confirmation candle is missing")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "wait for post-event confirmation candle")
		return snapshot
	}
	if !input.EventReaction.BreaksPreEventRange {
		snapshot.Verdict = TechnicalVerdictNoConfirmation
		snapshot.Reasons = append(snapshot.Reasons, "pre-event range break was not confirmed")
		snapshot.InvalidationRules = append(snapshot.InvalidationRules, "require clear break and hold/reject of event range")
		return snapshot
	}

	if snapshot.TechnicalScore < 40 {
		snapshot.Verdict = TechnicalVerdictNoConfirmation
		snapshot.Reasons = append(snapshot.Reasons, "technical score is below tradable threshold")
	} else if snapshot.TechnicalScore < 75 {
		snapshot.Verdict = TechnicalVerdictWatchOnly
		snapshot.Reasons = append(snapshot.Reasons, "technical score supports watch-only handling")
	} else {
		snapshot.Verdict = verdictFromBias(input.Bias)
		snapshot.Reasons = append(snapshot.Reasons, "technical score confirms event-aligned setup")
	}

	snapshot.InvalidationRules = append(snapshot.InvalidationRules,
		"reject if price closes back inside pre-event range",
		"reject if VWAP behavior flips against setup",
	)
	return snapshot
}

func normalizeTechnicalCandles(candles []marketdata.Candle) []marketdata.Candle {
	out := append([]marketdata.Candle(nil), candles...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out
}

func cloneLevels(levels map[string]float64) map[string]float64 {
	if len(levels) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(levels))
	for k, v := range levels {
		out[strings.TrimSpace(k)] = v
	}
	return out
}

func analysisTime(v time.Time) time.Time {
	if v.IsZero() {
		return time.Now().UTC()
	}
	return v.UTC()
}

func normalizeState(v string) string {
	trimmed := strings.TrimSpace(strings.ToLower(v))
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func scoreTrend(trendState string) float64 {
	switch trendState {
	case "uptrend", "downtrend":
		return 20
	case "transition":
		return 12
	case "range":
		return 8
	case "high-volatility chop", "chop":
		return 2
	default:
		return 4
	}
}

func scoreLevels(structureState string, levels map[string]float64, candles []marketdata.Candle) float64 {
	score := 0.0
	switch structureState {
	case "breakout", "breakdown":
		score += 14
	case "retest_hold", "retest_reject":
		score += 10
	case "range":
		score += 6
	default:
		score += 4
	}
	if len(candles) == 0 {
		return score
	}
	last := candles[len(candles)-1].Close
	if preEventHigh, ok := levels["pre_event_high"]; ok && last > preEventHigh {
		score += 3
	}
	if preEventLow, ok := levels["pre_event_low"]; ok && last < preEventLow {
		score += 3
	}
	if _, ok := levels["vwap"]; ok {
		score += 2
	}
	if score > 20 {
		return 20
	}
	return score
}

func scoreEventReaction(v TechnicalEventReaction) float64 {
	score := 0.0
	if v.BreaksPreEventRange {
		score += 8
	}
	if v.ConfirmationPresent {
		score += 6
	}
	if v.VWAPHold || v.VWAPReject {
		score += 6
	}
	if score > 20 {
		return 20
	}
	return score
}

func scoreVolumeATR(v TechnicalVolumeVolatility) float64 {
	score := 0.0
	if v.VolumeRatio >= 1.2 {
		score += 8
	} else if v.VolumeRatio >= 1.0 {
		score += 4
	}
	if v.ATRRatio >= 1.1 && v.ATRRatio <= 2.4 {
		score += 7
	} else if v.ATRRatio > 0 {
		score += 3
	}
	if score > 15 {
		return 15
	}
	return score
}

func scoreRelativeStrength(v TechnicalRelativeStrength) float64 {
	if v.ConflictingBasket {
		return 0
	}
	score := 0.0
	if v.AlignsWithScenario {
		score += 8
	}
	if math.Abs(v.SpreadToBenchmark) >= 0.0015 {
		score += 7
	} else if math.Abs(v.SpreadToBenchmark) >= 0.0005 {
		score += 4
	}
	if score > 15 {
		return 15
	}
	return score
}

func scoreEntryStop(hasStop bool, rewardRisk float64) float64 {
	if !hasStop {
		return 0
	}
	score := 4.0
	if rewardRisk >= 2.0 {
		score += 6
	} else if rewardRisk >= 1.5 {
		score += 4
	} else if rewardRisk >= 1.2 {
		score += 2
	}
	if score > 10 {
		return 10
	}
	return score
}

func verdictFromBias(bias TechnicalBias) TechnicalVerdict {
	switch TechnicalBias(strings.ToLower(strings.TrimSpace(string(bias)))) {
	case TechnicalBiasBullish:
		return TechnicalVerdictConfirmedBullish
	case TechnicalBiasBearish:
		return TechnicalVerdictConfirmedBearish
	default:
		return TechnicalVerdictWatchOnly
	}
}
