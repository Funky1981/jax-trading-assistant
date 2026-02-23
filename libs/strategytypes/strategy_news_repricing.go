package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type SameDayNewsRepricing struct{}

func NewSameDayNewsRepricing() *SameDayNewsRepricing { return &SameDayNewsRepricing{} }

func (s *SameDayNewsRepricing) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "same_day_news_repricing_v1",
		Name:        "Same-day News Repricing v1",
		Description: "Trades post-news consolidation breakouts after deterministic materiality filtering.",
		RequiredInputs: RequiredInputs{
			Candles:   []string{"1m", "5m"},
			NeedsNews: true,
		},
		Parameters: []ParameterDef{
			{Key: "entryDelayMins", Type: "int", Default: 15, Minimum: ptr(1), Maximum: ptr(120)},
			{Key: "consolidationBars", Type: "int", Default: 5, Minimum: ptr(2), Maximum: ptr(30)},
			{Key: "maxRangePct", Type: "float", Default: 0.5, Minimum: ptr(0.05), Maximum: ptr(5)},
		},
	}
}

func (s *SameDayNewsRepricing) Validate(params map[string]any) error {
	entryDelayMins, err := getInt(params, "entryDelayMins", 15)
	if err != nil {
		return err
	}
	consolidationBars, err := getInt(params, "consolidationBars", 5)
	if err != nil {
		return err
	}
	maxRangePct, err := getFloat(params, "maxRangePct", 0.5)
	if err != nil {
		return err
	}
	if err := requireRangeInt("entryDelayMins", entryDelayMins, 1, 120); err != nil {
		return err
	}
	if err := requireRangeInt("consolidationBars", consolidationBars, 2, 30); err != nil {
		return err
	}
	return requireRangeFloat("maxRangePct", maxRangePct, 0.05, 5)
}

func (s *SameDayNewsRepricing) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
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
	entryDelayMins, _ := getInt(input.Parameters, "entryDelayMins", 15)
	consolidationBars, _ := getInt(input.Parameters, "consolidationBars", 5)
	maxRangePct, _ := getFloat(input.Parameters, "maxRangePct", 0.5)

	event := input.News[0]
	isMaterial := strings.EqualFold(event.Materiality, "high") || strings.EqualFold(event.Materiality, "medium")
	if !isMaterial {
		return nil, nil
	}

	searchStart := event.Timestamp.Add(time.Duration(entryDelayMins) * time.Minute)
	startIdx := -1
	for i := range candles {
		if !candles[i].Timestamp.Before(searchStart) {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return nil, nil
	}

	for i := startIdx + consolidationBars; i < len(candles)-1; i++ {
		window := candles[i-consolidationBars : i]
		winHigh := window[0].High
		winLow := window[0].Low
		for _, c := range window {
			if c.High > winHigh {
				winHigh = c.High
			}
			if c.Low < winLow {
				winLow = c.Low
			}
		}
		rangePct := math.Abs(pctChange(winLow, winHigh))
		if rangePct > maxRangePct {
			continue
		}
		next := candles[i]
		if next.Close > winHigh {
			return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "BUY",
				fmt.Sprintf("news repricing breakout after %d-bar consolidation", consolidationBars), next)}, nil
		}
		if next.Close < winLow {
			return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "SELL",
				fmt.Sprintf("news repricing breakdown after %d-bar consolidation", consolidationBars), next)}, nil
		}
	}
	return nil, nil
}
