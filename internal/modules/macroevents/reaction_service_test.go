package macroevents

import (
	"context"
	"errors"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

func TestReactionServiceBuildsAndPersistsSnapshots(t *testing.T) {
	eventTime := reactionTestEventTime()
	store := &fakeReactionStore{}
	provider := &fakeCandleProvider{
		candles: map[string][]marketdata.Candle{
			"QQQ": {
				candle("QQQ", eventTime.Add(-5*time.Minute), 100, 101, 99, 100, 1000),
				candle("QQQ", eventTime.Add(15*time.Minute), 100, 100, 98, 99.2, 1200),
			},
		},
	}
	service := NewReactionService(provider, store)

	summary, err := service.BuildSnapshots(context.Background(), ReactionRequest{
		MacroEventID: "macro-1",
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Symbols:      []string{"QQQ"},
		Timeframes:   []ReactionTimeframe{TimeframePostEvent15M},
	})
	if err != nil {
		t.Fatalf("BuildSnapshots returned error: %v", err)
	}

	if len(summary.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(summary.Snapshots))
	}
	if !summary.Snapshots[0].ConfirmsEvent {
		t.Fatalf("expected confirming snapshot, got %#v", summary.Snapshots[0])
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved snapshots = %d, want 1", len(store.saved))
	}
}

func TestReactionServiceMissingCandlesReturnsUnavailableSnapshot(t *testing.T) {
	eventTime := reactionTestEventTime()
	store := &fakeReactionStore{}
	provider := &fakeCandleProvider{
		err: errors.New("provider unavailable"),
	}
	service := NewReactionService(provider, store)

	summary, err := service.BuildSnapshots(context.Background(), ReactionRequest{
		MacroEventID: "macro-1",
		EventType:    EventTypeUSCPIHeadline,
		Direction:    DirectionInflationHot,
		EventTimeUTC: eventTime,
		Symbols:      []string{"QQQ"},
		Timeframes:   []ReactionTimeframe{TimeframePostEvent15M},
	})
	if err != nil {
		t.Fatalf("BuildSnapshots returned error: %v", err)
	}

	if len(summary.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(summary.Snapshots))
	}
	if summary.Snapshots[0].Status != ReactionStatusUnavailable {
		t.Fatalf("status = %q, want unavailable", summary.Snapshots[0].Status)
	}
	if summary.Snapshots[0].ConfirmsEvent {
		t.Fatalf("unavailable snapshot must not confirm, got %#v", summary.Snapshots[0])
	}
	if len(store.saved) != 0 {
		t.Fatalf("saved snapshots = %d, want none for unavailable provider", len(store.saved))
	}
}

func TestReactionServiceDoesNotExposeCandidateOrOrderHooks(t *testing.T) {
	var _ reactionStore = &fakeReactionStore{}
	var _ CandleProvider = &fakeCandleProvider{}
}

type fakeReactionStore struct {
	saved []ReactionSnapshot
}

func (s *fakeReactionStore) SaveReactionSnapshot(_ context.Context, snapshot ReactionSnapshot) (ReactionSnapshot, error) {
	snapshot.ID = "snapshot-1"
	s.saved = append(s.saved, snapshot)
	return snapshot, nil
}

type fakeCandleProvider struct {
	candles map[string][]marketdata.Candle
	err     error
}

func (p *fakeCandleProvider) Candles(_ context.Context, symbol string, _ time.Time, _ time.Time) ([]marketdata.Candle, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.candles[symbol], nil
}
