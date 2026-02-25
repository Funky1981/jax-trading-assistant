package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type NewsShockMomentum struct{}

func NewNewsShockMomentum() *NewsShockMomentum { return &NewsShockMomentum{} }

func (s *NewsShockMomentum) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "news_shock_momentum_v1",
		Name:        "News Shock Momentum v1",
		Description: "Follows sharp post-news momentum bursts when volume confirms a high-materiality shock.",
		RequiredInputs: RequiredInputs{
			Candles:   []string{"1m", "5m"},
			NeedsNews: true,
		},
		Parameters: []ParameterDef{
			{Key: "entryDelayMins", Type: "int", Default: 5, Minimum: ptr(1), Maximum: ptr(60)},
			{Key: "momentumWindowMins", Type: "int", Default: 15, Minimum: ptr(5), Maximum: ptr(60)},
			{Key: "minMovePct", Type: "float", Default: 0.6, Minimum: ptr(0.1), Maximum: ptr(10)},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.3, Minimum: ptr(0.5), Maximum: ptr(10)},
		},
	}
}

func (s *NewsShockMomentum) Validate(params map[string]any) error {
	entryDelayMins, err := getInt(params, "entryDelayMins", 5)
	if err != nil {
		return err
	}
	momentumWindowMins, err := getInt(params, "momentumWindowMins", 15)
	if err != nil {
		return err
	}
	minMovePct, err := getFloat(params, "minMovePct", 0.6)
	if err != nil {
		return err
	}
	minVolumeMultiple, err := getFloat(params, "minVolumeMultiple", 1.3)
	if err != nil {
		return err
	}
	if err := requireRangeInt("entryDelayMins", entryDelayMins, 1, 60); err != nil {
		return err
	}
	if err := requireRangeInt("momentumWindowMins", momentumWindowMins, 5, 60); err != nil {
		return err
	}
	if err := requireRangeFloat("minMovePct", minMovePct, 0.1, 10); err != nil {
		return err
	}
	return requireRangeFloat("minVolumeMultiple", minVolumeMultiple, 0.5, 10)
}

func (s *NewsShockMomentum) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if len(input.News) == 0 {
		return nil, errors.New("missing required inputs: structured news events")
	}
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	entryDelayMins, _ := getInt(input.Parameters, "entryDelayMins", 5)
	momentumWindowMins, _ := getInt(input.Parameters, "momentumWindowMins", 15)
	minMovePct, _ := getFloat(input.Parameters, "minMovePct", 0.6)
	minVolumeMultiple, _ := getFloat(input.Parameters, "minVolumeMultiple", 1.3)

	event := input.News[0]
	if !strings.EqualFold(event.Materiality, "high") && !strings.EqualFold(event.Materiality, "medium") {
		return nil, nil
	}
	direction := ""
	switch {
	case strings.EqualFold(event.Sentiment, "positive"):
		direction = "BUY"
	case strings.EqualFold(event.Sentiment, "negative"):
		direction = "SELL"
	default:
		return nil, nil
	}

	entryTime := event.Timestamp.Add(time.Duration(entryDelayMins) * time.Minute)
	entryBar, entryIdx := candleAtOrAfter(candles, entryTime)
	if entryIdx < 0 {
		return nil, nil
	}
	windowEnd := entryBar.Timestamp.Add(time.Duration(momentumWindowMins) * time.Minute)
	exitBar, _ := barAtOrAfter(candles, windowEnd)
	if exitBar.Timestamp.IsZero() {
		return nil, nil
	}

	movePct := pctChange(entryBar.Close, exitBar.Close)
	if direction == "BUY" && movePct < minMovePct {
		return nil, nil
	}
	if direction == "SELL" && movePct > -minMovePct {
		return nil, nil
	}

	volBaseline := 0.0
	if entryIdx >= 0 {
		volBaseline = avgVolume(candles[:entryIdx+1], minInt(entryIdx+1, 30))
	}
	if volBaseline > 0 && (exitBar.Volume/volBaseline) < minVolumeMultiple {
		return nil, nil
	}

	reason := fmt.Sprintf("news shock momentum %s %.2f%%", strings.ToLower(direction), math.Abs(movePct))
	return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, direction, reason, exitBar)}, nil
}

func candleAtOrAfter(candles []Candle, ts time.Time) (Candle, int) {
	for i, c := range candles {
		if !c.Timestamp.Before(ts) {
			return c, i
		}
	}
	return Candle{}, -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
