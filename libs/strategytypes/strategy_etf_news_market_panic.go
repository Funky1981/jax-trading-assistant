package strategytypes

import (
	"context"
	"errors"
	"fmt"
)

// ETFNewsMarketPanicReversal generates BUY signals on broad-market ETFs
// after a confirmed panic selloff stabilises.
type ETFNewsMarketPanicReversal struct{}

func NewETFNewsMarketPanicReversal() *ETFNewsMarketPanicReversal {
	return &ETFNewsMarketPanicReversal{}
}

func (s *ETFNewsMarketPanicReversal) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "etf_news_market_panic_reversal_v1",
		Name:        "ETF News – Market Panic Reversal",
		Description: "Detects a confirmed broad-market panic selloff (news + price drop + volume spike) and generates a BUY signal on stabilisation.",
		RequiredInputs: RequiredInputs{
			Candles:   []string{"1m", "5m"},
			NeedsNews: true,
		},
		Parameters: []ParameterDef{
			{Key: "minDropPct", Type: "float", Default: 1.2, Minimum: ptr(0.5), Maximum: ptr(10.0), Description: "Minimum intraday drop % to qualify as panic"},
			{Key: "minConfirmations", Type: "int", Default: 2, Minimum: ptr(1.0), Maximum: ptr(5.0), Description: "Minimum news events confirming panic"},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.2, Minimum: ptr(1.0), Maximum: ptr(10.0), Description: "Minimum volume vs average to confirm panic"},
			{Key: "stabilizationBars", Type: "int", Default: 3, Minimum: ptr(1.0), Maximum: ptr(20.0), Description: "Number of quiet bars after panic before entry"},
			{Key: "atrStopMultiple", Type: "float", Default: 1.1, Minimum: ptr(0.5), Maximum: ptr(5.0), Description: "ATR multiple for stop loss distance"},
			{Key: "rewardRiskMultiple", Type: "float", Default: 1.5, Minimum: ptr(1.0), Maximum: ptr(10.0), Description: "Reward:Risk ratio for take profit"},
		},
	}
}

func (s *ETFNewsMarketPanicReversal) Validate(params map[string]any) error {
	minDrop, err := getFloat(params, "minDropPct", 1.2)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("minDropPct", minDrop, 0.5, 10.0); err != nil {
		return err
	}
	minConf, err := getInt(params, "minConfirmations", 2)
	if err != nil {
		return err
	}
	if err := requireRangeInt("minConfirmations", minConf, 1, 5); err != nil {
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

func (s *ETFNewsMarketPanicReversal) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	candles, _, err := normalizeCandles(input, "5m", "1m")
	if err != nil {
		return nil, err
	}
	if len(input.News) == 0 {
		return nil, errors.New("missing required inputs: news")
	}
	p := input.Parameters

	minDrop, _ := getFloat(p, "minDropPct", 1.2)
	minConf, _ := getInt(p, "minConfirmations", 2)
	minVol, _ := getFloat(p, "minVolumeMultiple", 1.2)
	stabBars, _ := getInt(p, "stabilizationBars", 3)
	atrMult, _ := getFloat(p, "atrStopMultiple", 1.1)
	rrMult, _ := getFloat(p, "rewardRiskMultiple", 1.5)

	// Require enough news confirmations for a panic event.
	confirmed := etfConfirmedNews(input.News, "panic")
	if len(confirmed) < minConf {
		return nil, nil
	}

	// Require a minimum price drop in the candle window.
	drop := etfDropPct(candles)
	if drop < minDrop {
		return nil, nil
	}

	// Require volume spike.
	volMultiple := etfVolumeMultiple(candles)
	if volMultiple < minVol {
		return nil, nil
	}

	// Require stabilisation: last N bars should have narrowing ranges.
	if len(candles) < stabBars+1 {
		return nil, nil
	}
	for i := len(candles) - stabBars; i < len(candles); i++ {
		if candles[i].High-candles[i].Low > candles[i-1].High-candles[i-1].Low {
			return nil, nil
		}
	}

	latest, _ := etfLatestCandle(candles)
	atr := etfATR(candles, 14)
	stopDist := atr * atrMult

	reason := fmt.Sprintf("panic_reversal: drop=%.2f%% vol_x=%.2fx confirmations=%d", drop, volMultiple, len(confirmed))
	sig := etfSignal(s.Metadata().StrategyID, input.Symbol, "BUY", reason, latest, stopDist, rrMult)
	return []Signal{sig}, nil
}
