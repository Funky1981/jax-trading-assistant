package ingest

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func NormalizeDexterObservation(obs DexterObservation) (NormalizedObservation, error) {
	obsType := normalizeObservationType(obs.Type)
	if obsType == "" {
		return NormalizedObservation{}, fmt.Errorf("dexter observation type is required")
	}

	ts := obs.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	symbol := strings.ToUpper(strings.TrimSpace(obs.Symbol))
	baseTags := []string{obsType}
	if obs.Bookmarked {
		baseTags = append(baseTags, "bookmarked")
	}
	tags := normalizeTags(baseTags, obs.Tags)

	switch obsType {
	case ObservationEarnings, ObservationNewsHeadline:
		summary := summarizeMarketEvent(obsType, symbol, obs.Headline)
		event := MarketEvent{
			TS:             ts.UTC(),
			EventType:      obsType,
			Symbol:         symbol,
			Tags:           tags,
			Summary:        summary,
			ImpactEstimate: obs.ImpactEstimate,
			Confidence:     obs.Confidence,
			Headline:       obs.Headline,
			SourceRef:      obs.SourceRef,
			Bookmarked:     obs.Bookmarked,
		}
		return NormalizedObservation{Kind: KindMarketEvent, Event: &event}, nil
	case ObservationUnusualVolume, ObservationPriceGap:
		summary := summarizeSignal(obsType, symbol, obs.VolumeMultiple, obs.GapPercent)
		signal := Signal{
			TS:             ts.UTC(),
			SignalType:     obsType,
			Symbol:         symbol,
			Tags:           tags,
			Summary:        summary,
			ImpactEstimate: obs.ImpactEstimate,
			Confidence:     obs.Confidence,
			VolumeMultiple: obs.VolumeMultiple,
			GapPercent:     obs.GapPercent,
			SourceRef:      obs.SourceRef,
			Bookmarked:     obs.Bookmarked,
		}
		return NormalizedObservation{Kind: KindSignal, Signal: &signal}, nil
	default:
		return NormalizedObservation{}, fmt.Errorf("unsupported dexter observation type %q", obs.Type)
	}
}

func normalizeObservationType(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	switch s {
	case "earnings_detected", "earnings_event":
		return ObservationEarnings
	case "news", "headline":
		return ObservationNewsHeadline
	case "volume_spike", "volume":
		return ObservationUnusualVolume
	case "gap", "gap_up", "gap_down":
		return ObservationPriceGap
	default:
		return s
	}
}

func normalizeTags(base []string, extra []string) []string {
	tags := append([]string{}, base...)
	tags = append(tags, extra...)
	tags = contracts.NormalizeMemoryTags(tags)
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) >= contracts.MaxTags {
			break
		}
	}
	return out
}

func summarizeMarketEvent(eventType, symbol, headline string) string {
	label := humanizeType(eventType)
	headline = strings.TrimSpace(headline)
	if eventType == ObservationNewsHeadline && headline != "" {
		if symbol != "" {
			return fmt.Sprintf("Dexter news for %s: %s.", symbol, headline)
		}
		return fmt.Sprintf("Dexter news: %s.", headline)
	}
	if symbol != "" {
		return fmt.Sprintf("Dexter detected %s for %s.", label, symbol)
	}
	return fmt.Sprintf("Dexter detected %s.", label)
}

func summarizeSignal(signalType, symbol string, volumeMultiple, gapPercent float64) string {
	label := humanizeType(signalType)
	detail := ""
	switch signalType {
	case ObservationUnusualVolume:
		if volumeMultiple > 0 {
			detail = fmt.Sprintf(" (%.2fx avg)", volumeMultiple)
		}
	case ObservationPriceGap:
		if gapPercent != 0 {
			detail = fmt.Sprintf(" (%.2f%%)", gapPercent)
		}
	}
	if symbol != "" {
		return fmt.Sprintf("Dexter detected %s for %s%s.", label, symbol, detail)
	}
	return fmt.Sprintf("Dexter detected %s%s.", label, detail)
}

func humanizeType(raw string) string {
	return strings.ReplaceAll(raw, "_", " ")
}
