package app

import (
	"context"
	"fmt"
	"math"
	"time"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type EventDetector struct {
	market MarketData
	audit  *AuditLogger
}

func NewEventDetector(market MarketData, audit *AuditLogger) *EventDetector {
	return &EventDetector{market: market, audit: audit}
}

func (d *EventDetector) DetectGaps(ctx context.Context, symbol string, thresholdPct float64) ([]domain.Event, error) {
	if d.audit != nil {
		_ = d.audit.LogDecision(ctx, "event_detect_start", domain.AuditOutcomeStarted, map[string]any{
			"symbol":       symbol,
			"thresholdPct": thresholdPct,
		}, nil)
	}
	candles, err := d.market.GetDailyCandles(ctx, symbol, 2)
	if err != nil {
		if d.audit != nil {
			_ = d.audit.LogDecision(ctx, "event_detect_error", domain.AuditOutcomeError, map[string]any{
				"symbol": symbol,
			}, err)
		}
		return nil, err
	}
	if len(candles) < 2 {
		if d.audit != nil {
			_ = d.audit.LogDecision(ctx, "event_detect_insufficient_candles", domain.AuditOutcomeSkipped, map[string]any{
				"symbol":       symbol,
				"candleCount":  len(candles),
				"thresholdPct": thresholdPct,
			}, nil)
		}
		return nil, nil
	}

	prev := candles[len(candles)-2]
	cur := candles[len(candles)-1]
	if prev.Close == 0 {
		if d.audit != nil {
			_ = d.audit.LogDecision(ctx, "event_detect_missing_prev_close", domain.AuditOutcomeSkipped, map[string]any{
				"symbol":       symbol,
				"thresholdPct": thresholdPct,
			}, nil)
		}
		return nil, nil
	}

	gapPct := ((cur.Open - prev.Close) / prev.Close) * 100
	if math.Abs(gapPct) < thresholdPct {
		if d.audit != nil {
			_ = d.audit.LogDecision(ctx, "event_detect_below_threshold", domain.AuditOutcomeSkipped, map[string]any{
				"symbol":       symbol,
				"thresholdPct": thresholdPct,
			}, nil)
		}
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

	if d.audit != nil {
		payload := redactEventPayload(e)
		payload["gapDirection"] = inferGapDirection(gapPct)
		_ = d.audit.LogDecision(ctx, "event_detect_success", domain.AuditOutcomeSuccess, payload, nil)
	}

	return []domain.Event{e}, nil
}

func inferGapDirection(gapPct float64) string {
	if gapPct < 0 {
		return "down"
	}
	return "up"
}
