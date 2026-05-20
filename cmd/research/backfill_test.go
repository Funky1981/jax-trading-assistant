package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type fakeBackfillStore struct {
	candleUpserts     int
	eventUpserts      int
	windowUpserts     int
	scoreUpserts      int
	candles           map[string][]marketdata.Candle
	events            map[string]backfillEventRecord
	normalizedEvents  map[string]backfillNormalizedEvent
	lastCandleSymbols []string
}

func (s *fakeBackfillStore) UpsertCandles(ctx context.Context, candles []marketdata.Candle) (int, error) {
	s.candleUpserts += len(candles)
	for _, c := range candles {
		s.lastCandleSymbols = append(s.lastCandleSymbols, c.Symbol)
	}
	return len(candles), nil
}

func (s *fakeBackfillStore) UpsertEvents(ctx context.Context, events []backfillEventRecord) (backfillEventWriteSummary, error) {
	s.eventUpserts += len(events)
	if s.events == nil {
		s.events = map[string]backfillEventRecord{}
	}
	for _, e := range events {
		s.events[e.CanonicalKey] = e
	}
	return backfillEventWriteSummary{RawRows: len(events), NormalizedRows: len(events), SymbolMaps: len(events)}, nil
}

func (s *fakeBackfillStore) LoadEventStudyInputs(ctx context.Context, eventIDs []string, symbols []string, from, to time.Time) ([]backfillNormalizedEvent, map[string][]marketdata.Candle, error) {
	return mapValues(s.normalizedEvents), s.candles, nil
}

func (s *fakeBackfillStore) UpsertEventStudy(ctx context.Context, windows []eventWindowResult, scores []pricedInScoreResult, confounders []eventConfounderResult) (backfillEventStudyWriteSummary, error) {
	s.windowUpserts += len(windows)
	s.scoreUpserts += len(scores)
	return backfillEventStudyWriteSummary{Windows: len(windows), Scores: len(scores), Confounders: len(confounders)}, nil
}

type fakeCandleFetcher struct {
	candles map[string][]marketdata.Candle
}

func (f fakeCandleFetcher) GetCandles(ctx context.Context, symbol string, timeframe marketdata.Timeframe, limit int) ([]marketdata.Candle, error) {
	return f.candles[symbol], nil
}

func TestBackfillCandlesNormalizesToETFAllowlistAndWritesSummary(t *testing.T) {
	store := &fakeBackfillStore{}
	fetcher := fakeCandleFetcher{candles: map[string][]marketdata.Candle{
		"SPY": {{Symbol: "SPY", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Open: 1, High: 2, Low: 1, Close: 2, Volume: 100}},
	}}
	runner := newBackfillRunner(store, fetcher)

	resp := runner.RunCandles(context.Background(), BackfillCandlesRequest{
		Symbols:   []string{"spy", "AAPL"},
		Timeframe: string(marketdata.Timeframe1Day),
		From:      "2026-01-01",
		To:        "2026-01-03",
		Provider:  "test",
	})

	if resp.Status != "degraded" {
		t.Fatalf("expected degraded status because AAPL is rejected, got %q", resp.Status)
	}
	if resp.InsertedRows != 1 {
		t.Fatalf("expected 1 inserted candle row, got %d", resp.InsertedRows)
	}
	if len(resp.Failures) != 1 || resp.Failures[0].Symbol != "AAPL" {
		t.Fatalf("expected AAPL allowlist failure, got %#v", resp.Failures)
	}
	if got := store.lastCandleSymbols; len(got) != 1 || got[0] != "SPY" {
		t.Fatalf("expected only SPY writes, got %#v", got)
	}
}

func TestBackfillEventsUpsertsRawNormalizedAndSymbolMaps(t *testing.T) {
	store := &fakeBackfillStore{}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEvents(context.Background(), BackfillEventsRequest{
		Provider: "finnhub",
		Events: []BackfillEventInput{{
			SourceEventID: "evt-1",
			EventKind:     "news",
			EventTime:     "2026-01-02T15:04:05Z",
			Title:         "Rates fall",
			Summary:       "Bonds rally",
			Symbols:       []string{"TLT", "AAPL"},
			CanonicalKey:  "finnhub:evt-1",
		}},
	})

	if resp.Status != "degraded" {
		t.Fatalf("expected degraded status because AAPL is rejected, got %q", resp.Status)
	}
	if resp.InsertedRows != 1 || resp.NormalizedRows != 1 || resp.SymbolMaps != 1 {
		t.Fatalf("unexpected write counts: %#v", resp)
	}
	if _, ok := store.events["finnhub:evt-1"]; !ok {
		t.Fatal("expected canonical event to be written")
	}
}

func TestBackfillEventStudyComputesWindowsAndPricedInScores(t *testing.T) {
	eventTime := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	store := &fakeBackfillStore{
		normalizedEvents: map[string]backfillNormalizedEvent{
			"event-1": {ID: "event-1", EventTime: eventTime, PrimarySymbol: "SPY"},
		},
		candles: map[string][]marketdata.Candle{
			"SPY": {
				{Symbol: "SPY", Timestamp: eventTime.Add(-24 * time.Hour), Close: 100},
				{Symbol: "SPY", Timestamp: eventTime, Close: 101},
				{Symbol: "SPY", Timestamp: eventTime.Add(time.Hour), Close: 104},
			},
		},
	}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEventStudy(context.Background(), BackfillEventStudyRequest{
		EventIDs: []string{"event-1"},
		Symbols:  []string{"SPY"},
		Windows:  []string{"-1d_to_event", "event_to_+1h"},
	})

	if resp.Status != "completed" {
		t.Fatalf("expected completed status, got %q failures=%#v", resp.Status, resp.Failures)
	}
	if resp.Windows != 2 {
		t.Fatalf("expected 2 windows, got %d", resp.Windows)
	}
	if resp.Scores != 1 {
		t.Fatalf("expected 1 priced-in score, got %d", resp.Scores)
	}
	if store.windowUpserts != 2 || store.scoreUpserts != 1 {
		t.Fatalf("unexpected store writes: windows=%d scores=%d", store.windowUpserts, store.scoreUpserts)
	}
}

func TestBackfillHTTPRoutesSubmitAndFetchRun(t *testing.T) {
	store := &fakeBackfillStore{}
	fetcher := fakeCandleFetcher{candles: map[string][]marketdata.Candle{
		"QQQ": {{Symbol: "QQQ", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Open: 1, High: 2, Low: 1, Close: 2, Volume: 100}},
	}}
	manager := newBackfillManager(newBackfillRunner(store, fetcher))
	mux := http.NewServeMux()
	registerBackfillRoutes(mux, manager)

	body := bytes.NewBufferString(`{"symbols":["QQQ"],"timeframe":"1D","from":"2026-01-01","to":"2026-01-03","provider":"test"}`)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/research/backfill/candles", body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created BackfillRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created response: %v", err)
	}
	if created.RunID == "" || created.Status != "completed" {
		t.Fatalf("unexpected created response: %#v", created)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/research/backfill/runs/"+created.RunID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func mapValues[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
