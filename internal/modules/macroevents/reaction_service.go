package macroevents

import (
	"context"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type CandleProvider interface {
	Candles(ctx context.Context, symbol string, from time.Time, to time.Time) ([]marketdata.Candle, error)
}

type reactionStore interface {
	SaveReactionSnapshot(ctx context.Context, snapshot ReactionSnapshot) (ReactionSnapshot, error)
}

type ReactionRequest struct {
	MacroEventID string
	EventType    EventType
	Direction    Direction
	EventTimeUTC time.Time
	Symbols      []string
	Timeframes   []ReactionTimeframe
}

type ReactionSummary struct {
	MacroEventID string
	Snapshots    []ReactionSnapshot
}

type ReactionService struct {
	provider CandleProvider
	store    reactionStore
}

func NewReactionService(provider CandleProvider, store reactionStore) *ReactionService {
	return &ReactionService{provider: provider, store: store}
}

func (s *ReactionService) BuildSnapshots(ctx context.Context, req ReactionRequest) (ReactionSummary, error) {
	timeframes := req.Timeframes
	if len(timeframes) == 0 {
		timeframes = []ReactionTimeframe{TimeframePostEvent5M, TimeframePostEvent15M, TimeframePostEvent30M, TimeframePostEvent60M}
	}

	summary := ReactionSummary{MacroEventID: req.MacroEventID}
	for _, symbol := range dedupeSymbols(req.Symbols) {
		for _, timeframe := range timeframes {
			from, to := reactionBounds(req.EventTimeUTC, timeframe)
			candles, err := s.provider.Candles(ctx, symbol, from, to)
			if err != nil {
				summary.Snapshots = append(summary.Snapshots, unavailableReaction(req, symbol, timeframe, err.Error()))
				continue
			}
			snapshot := EvaluateReaction(ReactionInput{
				MacroEventID: req.MacroEventID,
				Symbol:       symbol,
				Timeframe:    timeframe,
				EventType:    req.EventType,
				Direction:    req.Direction,
				EventTimeUTC: req.EventTimeUTC,
				Candles:      candles,
			})
			if snapshot.Status == ReactionStatusAvailable && s.store != nil {
				stored, err := s.store.SaveReactionSnapshot(ctx, snapshot)
				if err != nil {
					return ReactionSummary{}, err
				}
				snapshot = stored
			}
			summary.Snapshots = append(summary.Snapshots, snapshot)
		}
	}
	return summary, nil
}

func unavailableReaction(req ReactionRequest, symbol string, timeframe ReactionTimeframe, reason string) ReactionSnapshot {
	return ReactionSnapshot{
		MacroEventID: req.MacroEventID,
		Symbol:       symbol,
		Timeframe:    timeframe,
		Direction:    ReactionDirectionUnknown,
		Status:       ReactionStatusUnavailable,
		Reason:       "reaction unavailable: " + reason,
	}
}

func reactionBounds(eventTime time.Time, timeframe ReactionTimeframe) (time.Time, time.Time) {
	switch timeframe {
	case TimeframePreEvent30M:
		return eventTime.Add(-30 * time.Minute), eventTime
	case TimeframePreEvent5M:
		return eventTime.Add(-5 * time.Minute), eventTime
	case TimeframePostEvent5M:
		return eventTime.Add(-5 * time.Minute), eventTime.Add(5 * time.Minute)
	case TimeframePostEvent15M:
		return eventTime.Add(-5 * time.Minute), eventTime.Add(15 * time.Minute)
	case TimeframePostEvent30M:
		return eventTime.Add(-5 * time.Minute), eventTime.Add(30 * time.Minute)
	case TimeframePostEvent60M:
		return eventTime.Add(-5 * time.Minute), eventTime.Add(60 * time.Minute)
	case TimeframeSessionToNow:
		return time.Date(eventTime.Year(), eventTime.Month(), eventTime.Day(), 0, 0, 0, 0, eventTime.Location()), time.Now().UTC()
	default:
		return eventTime.Add(-5 * time.Minute), eventTime.Add(15 * time.Minute)
	}
}
