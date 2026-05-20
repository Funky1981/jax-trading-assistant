package strategytypes

import (
	"context"
	"errors"
	"fmt"
)

// ETFNewsSectorMomentum generates BUY or SELL signals on sector ETFs
// after confirmed sector-specific news with sufficient volume.
type ETFNewsSectorMomentum struct{}

func NewETFNewsSectorMomentum() *ETFNewsSectorMomentum {
	return &ETFNewsSectorMomentum{}
}

func (s *ETFNewsSectorMomentum) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "etf_news_sector_momentum_v1",
		Name:        "ETF News – Sector Momentum",
		Description: "Detects confirmed sector news (semiconductors, tech, energy, financials) with volume confirmation and generates a directional signal on the relevant sector ETF.",
		RequiredInputs: RequiredInputs{
			Candles:   []string{"1m", "5m"},
			NeedsNews: true,
		},
		Parameters: []ParameterDef{
			{Key: "minConfirmations", Type: "int", Default: 2, Minimum: ptr(1.0), Maximum: ptr(5.0), Description: "Minimum news events confirming sector catalyst"},
			{Key: "minMovePct", Type: "float", Default: 0.6, Minimum: ptr(0.1), Maximum: ptr(5.0), Description: "Minimum price move % from open to confirm momentum"},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.2, Minimum: ptr(1.0), Maximum: ptr(10.0), Description: "Minimum volume vs average to confirm momentum"},
			{Key: "stabilizationBars", Type: "int", Default: 3, Minimum: ptr(1.0), Maximum: ptr(20.0), Description: "Bars of stabilisation before entry"},
			{Key: "atrStopMultiple", Type: "float", Default: 1.1, Minimum: ptr(0.5), Maximum: ptr(5.0), Description: "ATR multiple for stop loss"},
			{Key: "rewardRiskMultiple", Type: "float", Default: 1.5, Minimum: ptr(1.0), Maximum: ptr(10.0), Description: "Reward:Risk ratio"},
		},
	}
}

func (s *ETFNewsSectorMomentum) Validate(params map[string]any) error {
	minConf, err := getInt(params, "minConfirmations", 2)
	if err != nil {
		return err
	}
	if err := requireRangeInt("minConfirmations", minConf, 1, 5); err != nil {
		return err
	}
	minMove, err := getFloat(params, "minMovePct", 0.6)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("minMovePct", minMove, 0.1, 5.0); err != nil {
		return err
	}
	minVol, err := getFloat(params, "minVolumeMultiple", 1.2)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("minVolumeMultiple", minVol, 1.0, 10.0); err != nil {
		return err
	}
	stabBars, err := getInt(params, "stabilizationBars", 3)
	if err != nil {
		return err
	}
	if err := requireRangeInt("stabilizationBars", stabBars, 1, 20); err != nil {
		return err
	}
	atrMult, err := getFloat(params, "atrStopMultiple", 1.1)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("atrStopMultiple", atrMult, 0.5, 5.0); err != nil {
		return err
	}
	rrMult, err := getFloat(params, "rewardRiskMultiple", 1.5)
	if err != nil {
		return err
	}
	return requireRangeFloat("rewardRiskMultiple", rrMult, 1.0, 10.0)
}

func (s *ETFNewsSectorMomentum) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	candles, _, err := normalizeCandles(input, "5m", "1m")
	if err != nil {
		return nil, err
	}
	if len(input.News) == 0 {
		return nil, errors.New("missing required inputs: news")
	}
	p := input.Parameters

	minConf, _ := getInt(p, "minConfirmations", 2)
	minMove, _ := getFloat(p, "minMovePct", 0.6)
	minVol, _ := getFloat(p, "minVolumeMultiple", 1.2)
	atrMult, _ := getFloat(p, "atrStopMultiple", 1.1)
	rrMult, _ := getFloat(p, "rewardRiskMultiple", 1.5)

	// Accept bullish or bearish sector news.
	confirmed := etfConfirmedNews(input.News, "sector", "bullish", "bearish")
	if len(confirmed) < minConf {
		return nil, nil
	}

	if len(candles) < 2 {
		return nil, nil
	}

	// Volume check.
	volMultiple := etfVolumeMultiple(candles)
	if volMultiple < minVol {
		return nil, nil
	}

	// Price move from open.
	open := candles[0].Open
	latest, _ := etfLatestCandle(candles)
	move := pctChange(open, latest.Close)

	direction := "BUY"
	if move < 0 {
		direction = "SELL"
		move = -move
	}
	if move < minMove {
		return nil, nil
	}

	atr := etfATR(candles, 14)
	stopDist := atr * atrMult

	reason := fmt.Sprintf("sector_momentum: move=%.2f%% vol_x=%.2fx confirmations=%d direction=%s", move, volMultiple, len(confirmed), direction)
	sig := etfSignal(s.Metadata().StrategyID, input.Symbol, direction, reason, latest, stopDist, rrMult)
	return []Signal{sig}, nil
}
