package ingest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"jax-trading-assistant/libs/contracts"
)

const dexterSourceSystem = "dexter"

func RetainDexterObservations(ctx context.Context, store contracts.MemoryStore, observations []DexterObservation, cfg RetentionConfig) (RetentionResult, error) {
	if store == nil {
		return RetentionResult{}, errors.New("memory store is required")
	}

	threshold := cfg.SignificanceThreshold
	if threshold < 0 {
		threshold = 0
	}

	result := RetentionResult{}
	for _, obs := range observations {
		normalized, err := NormalizeDexterObservation(obs)
		if err != nil {
			return result, err
		}

		switch normalized.Kind {
		case KindMarketEvent:
			event := normalized.Event
			score := retentionScore(event.ImpactEstimate, event.Confidence)
			if !shouldRetain(event.Bookmarked, score, threshold) {
				result.Skipped++
				continue
			}
			bank, item := memoryItemFromMarketEvent(*event)
			if err := contracts.ValidateMemoryItem(item); err != nil {
				return result, fmt.Errorf("retain %s: %w", bank, err)
			}
			if _, err := store.Retain(ctx, bank, item); err != nil {
				return result, err
			}
			result.Retained++
		case KindSignal:
			signal := normalized.Signal
			score := retentionScore(signal.ImpactEstimate, signal.Confidence)
			if !shouldRetain(signal.Bookmarked, score, threshold) {
				result.Skipped++
				continue
			}
			bank, item := memoryItemFromSignal(*signal)
			if err := contracts.ValidateMemoryItem(item); err != nil {
				return result, fmt.Errorf("retain %s: %w", bank, err)
			}
			if _, err := store.Retain(ctx, bank, item); err != nil {
				return result, err
			}
			result.Retained++
		default:
			return result, fmt.Errorf("unknown normalized kind %q", normalized.Kind)
		}
	}

	return result, nil
}

func retentionScore(impactEstimate float64, confidence float64) float64 {
	if impactEstimate != 0 {
		return math.Abs(impactEstimate)
	}
	return math.Abs(confidence)
}

func shouldRetain(bookmarked bool, score float64, threshold float64) bool {
	if bookmarked {
		return true
	}
	if threshold <= 0 {
		return true
	}
	return score >= threshold
}

func memoryItemFromMarketEvent(event MarketEvent) (string, contracts.MemoryItem) {
	data := map[string]any{
		"event_type":      event.EventType,
		"impact_estimate": event.ImpactEstimate,
		"confidence":      event.Confidence,
	}
	if strings.TrimSpace(event.Headline) != "" {
		data["headline"] = strings.TrimSpace(event.Headline)
	}

	itemType := event.EventType + "_event"
	item := contracts.MemoryItem{
		TS:      event.TS.UTC(),
		Type:    itemType,
		Symbol:  event.Symbol,
		Tags:    contracts.NormalizeMemoryTags(event.Tags),
		Summary: event.Summary,
		Data:    data,
		Source:  &contracts.MemorySource{System: dexterSourceSystem, Ref: event.SourceRef},
	}
	return "market_events", item
}

func memoryItemFromSignal(signal Signal) (string, contracts.MemoryItem) {
	data := map[string]any{
		"event_type":      signal.SignalType,
		"impact_estimate": signal.ImpactEstimate,
		"confidence":      signal.Confidence,
	}
	if signal.VolumeMultiple > 0 {
		data["volume_multiple"] = signal.VolumeMultiple
	}
	if signal.GapPercent != 0 {
		data["gap_percent"] = signal.GapPercent
	}

	item := contracts.MemoryItem{
		TS:      signal.TS.UTC(),
		Type:    "signal",
		Symbol:  signal.Symbol,
		Tags:    contracts.NormalizeMemoryTags(signal.Tags),
		Summary: signal.Summary,
		Data:    data,
		Source:  &contracts.MemorySource{System: dexterSourceSystem, Ref: signal.SourceRef},
	}
	return "signals", item
}
