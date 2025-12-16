package app

import (
	"context"
	"fmt"
	"math"
	"time"

	"jax-trading-assistant/internal/domain"
)

type EventDetector struct {
	market MarketData
}

func NewEventDetector(market MarketData) *EventDetector {
	return &EventDetector{market: market}
}

func (d *EventDetector) DetectGaps(ctx context.Context, symbol string, thresholdPct float64) ([]domain.Event, error) {
	candles, err := d.market.GetDailyCandles(ctx, symbol, 2)
	if err != nil {
		return nil, err
	}
	if len(candles) < 2 {
		return nil, nil
	}

	prev := candles[len(candles)-2]
	cur := candles[len(candles)-1]
	if prev.Close == 0 {
		return nil, nil
	}

	gapPct := ((cur.Open - prev.Close) / prev.Close) * 100
	if math.Abs(gapPct) < thresholdPct {
		return nil, nil
	}

	eventTime := cur.TS
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	e := domain.Event{
		ID:     fmt.Sprintf("ev_gap_%s_%d", symbol, eventTime.UTC().Unix()),
		Symbol: symbol,
		Type:   domain.EventGapOpen,
		Time:   eventTime.UTC(),
		Payload: map[string]any{
			"gapPct":    gapPct,
			"prevClose": prev.Close,
			"open":      cur.Open,
		},
	}

	return []domain.Event{e}, nil
}
