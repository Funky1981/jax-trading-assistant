# 04 — Event Study and Historical Research

## Goal

Upgrade event study from intraday-only research into a dual-window research engine:

```text
intraday reaction windows + swing follow-through windows
```

## Required Windows

### Intraday Windows

```text
-1d_to_event
-4h_to_event
-1h_to_event
event_to_+5m
event_to_+15m
event_to_+1h
event_to_+4h
event_to_close
```

### Swing Windows

```text
event_to_+1d
event_to_+2d
event_to_+3d
event_to_+5d
event_to_+10d
```

## Required Fix — Bounds From Event Times

Replace time-now bounds with event-derived bounds.

Target helper shape:

```go
func studyBoundsForEvents(events []backfillNormalizedEvent, windows []eventStudyWindow) (time.Time, time.Time, error) {
    if len(events) == 0 {
        return time.Time{}, time.Time{}, errors.New("events required")
    }
    minEvent := events[0].EventTime
    maxEvent := events[0].EventTime
    maxBefore := time.Duration(0)
    maxAfter := time.Duration(0)

    for _, event := range events {
        if event.EventTime.Before(minEvent) { minEvent = event.EventTime }
        if event.EventTime.After(maxEvent) { maxEvent = event.EventTime }
    }
    for _, window := range windows {
        if window.Before > maxBefore { maxBefore = window.Before }
        if window.After > maxAfter { maxAfter = window.After }
    }
    return minEvent.Add(-maxBefore), maxEvent.Add(maxAfter), nil
}
```

Implementation options:

```text
Option A: Load events first, then load candles by event-derived bounds.
Option B: Add store method LoadEventsForStudy, then LoadCandlesForStudy.
```

Option B is cleaner and more testable.

## Store Interface Target

```go
type eventStudyStore interface {
    LoadNormalizedEvents(ctx context.Context, eventIDs []string) ([]backfillNormalizedEvent, error)
    LoadCandlesForStudy(ctx context.Context, symbols []string, from time.Time, to time.Time) (map[string][]marketdata.Candle, error)
    UpsertEventStudy(ctx context.Context, windows []eventWindowResult, scores []pricedInScoreResult, confounders []eventConfounderResult) (backfillEventStudyWriteSummary, error)
    UpsertResearchSummaries(ctx context.Context, bundles []researchEvidenceBundle) (int, error)
}
```

## Swing Edge Metrics

Add to `pricedInScoreResult` or a new `swingEdgeScoreResult`:

```go
type SwingEdgeScoreResult struct {
    EventID string
    Symbol string
    Direction string
    Score float64
    Verdict string
    Reason string
    Return1D float64
    Return2D float64
    Return3D float64
    Return5D float64
    Return10D float64
    MaxDrawdown5D float64
    MaxAdverseExcursion float64
    FollowThroughQuality float64
    ReversalRisk float64
    DataQuality string
    HardReject bool
    HardRejectReasons []string
}
```

## Swing Verdicts

```text
follow_through
mean_reversion
one_day_spike_only
choppy_no_edge
confounded
insufficient_data
priced_in
```

## Minimum Swing Acceptance Logic

Swing thesis can proceed only if:

```text
historical sample count >= configured minimum
+3d or +5d return direction supports thesis
max adverse excursion is within risk limit
priced-in verdict is not hard reject
no unresolved high-relevance confounder
calendar risk passes
```

## Historical Comparison Requirements

For each event type + ETF pair, store:

```text
sample_count
median_1d_return
median_3d_return
median_5d_return
win_rate_3d
win_rate_5d
median_max_drawdown
worst_case_drawdown
similar_event_examples
```

## Tests

- Event-study bounds use event time, not current time.
- Swing windows are computed correctly over non-trading days.
- Missing candles produce `insufficient_data`, not silent pass.
- Confounded samples are flagged separately.
- Swing edge score rejects if sample count is too low.
- Swing edge score rejects if +1d pops but +3d/+5d fades.
