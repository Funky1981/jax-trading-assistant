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
	summaryUpserts    int
	candles           map[string][]marketdata.Candle
	events            map[string]backfillEventRecord
	normalizedEvents  map[string]backfillNormalizedEvent
	lastCandleSymbols []string
	lastScores        []pricedInScoreResult
	lastBundles       []researchEvidenceBundle
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
	s.lastScores = append([]pricedInScoreResult(nil), scores...)
	return backfillEventStudyWriteSummary{Windows: len(windows), Scores: len(scores), Confounders: len(confounders)}, nil
}

func (s *fakeBackfillStore) UpsertResearchSummaries(ctx context.Context, bundles []researchEvidenceBundle) (int, error) {
	s.summaryUpserts += len(bundles)
	s.lastBundles = append([]researchEvidenceBundle(nil), bundles...)
	return len(bundles), nil
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

func TestClassifyETFEventMapsSectorNewsWithoutExplicitTickers(t *testing.T) {
	store := &fakeBackfillStore{}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEvents(context.Background(), BackfillEventsRequest{
		Provider: "finnhub",
		Events: []BackfillEventInput{{
			SourceEventID: "evt-ai",
			EventKind:     "news",
			EventTime:     "2026-01-02T15:04:05Z",
			Title:         "Nvidia chip demand surges as AI capex accelerates",
			Summary:       "Semiconductor suppliers rally on datacenter demand.",
			CanonicalKey:  "finnhub:evt-ai",
		}},
	})

	if resp.Status != "completed" {
		t.Fatalf("expected completed status, got %q failures=%#v", resp.Status, resp.Failures)
	}
	record := store.events["finnhub:evt-ai"]
	if record.PrimarySymbol != "SMH" {
		t.Fatalf("primary ETF = %q, want SMH", record.PrimarySymbol)
	}
	wantSymbols := []string{"SMH", "SOXX", "QQQ"}
	if got := record.Symbols; !stringSlicesEqual(got, wantSymbols) {
		t.Fatalf("symbols = %#v, want %#v", got, wantSymbols)
	}
	if got := record.Attributes["event_type"]; got != "semiconductor_ai" {
		t.Fatalf("event_type = %#v, want semiconductor_ai", got)
	}
	if got := record.Attributes["tradeable"]; got != true {
		t.Fatalf("tradeable = %#v, want true", got)
	}
	if got := record.Attributes["classification_source"]; got != "rule" {
		t.Fatalf("classification_source = %#v, want rule", got)
	}
}

func TestClassifyETFEventMarksUnclearNewsUnknownAndNotTradeable(t *testing.T) {
	store := &fakeBackfillStore{}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEvents(context.Background(), BackfillEventsRequest{
		Provider: "finnhub",
		Events: []BackfillEventInput{{
			SourceEventID: "evt-unclear",
			EventKind:     "news",
			EventTime:     "2026-01-02T15:04:05Z",
			Title:         "Company updates office lease plan",
			Summary:       "Executives discussed facilities planning.",
			CanonicalKey:  "finnhub:evt-unclear",
		}},
	})

	if resp.Status != "failed" {
		t.Fatalf("expected failed status for unmapped event, got %q", resp.Status)
	}
	if len(resp.Failures) == 0 || resp.Failures[0].Stage != "classification" {
		t.Fatalf("expected classification failure, got %#v", resp.Failures)
	}
}

func TestClassifyETFEventPreservesExplicitAllowlistedSymbolsAndAddsClassification(t *testing.T) {
	store := &fakeBackfillStore{}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEvents(context.Background(), BackfillEventsRequest{
		Provider: "calendar",
		Events: []BackfillEventInput{{
			SourceEventID: "evt-cpi",
			EventKind:     "macro",
			EventTime:     "2026-01-02T15:04:05Z",
			Title:         "Inflation shock lifts rate hike fears",
			Summary:       "CPI surprised higher and Treasury yields jumped.",
			Symbols:       []string{"GLD"},
			CanonicalKey:  "calendar:evt-cpi",
		}},
	})

	if resp.Status != "completed" {
		t.Fatalf("expected completed status, got %q failures=%#v", resp.Status, resp.Failures)
	}
	record := store.events["calendar:evt-cpi"]
	if record.PrimarySymbol != "TLT" {
		t.Fatalf("primary ETF = %q, want TLT", record.PrimarySymbol)
	}
	wantSymbols := []string{"TLT", "SPY", "QQQ", "GLD"}
	if got := record.Symbols; !stringSlicesEqual(got, wantSymbols) {
		t.Fatalf("symbols = %#v, want %#v", got, wantSymbols)
	}
	if got := record.Attributes["event_type"]; got != "inflation" {
		t.Fatalf("event_type = %#v, want inflation", got)
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

func TestEvidenceBundleBuilderIncludesClassificationPricedInAndBeginnerSummary(t *testing.T) {
	eventTime := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	event := backfillNormalizedEvent{
		ID:            "event-1",
		EventTime:     eventTime,
		PrimarySymbol: "SMH",
		EventKind:     "news",
		Title:         "Nvidia chip demand surges",
		Summary:       "Semiconductor suppliers rally on AI demand.",
		SourceID:      "finnhub",
		Attributes: map[string]any{
			"event_type":            "semiconductor_ai",
			"classification_reason": "Headline affects semiconductor demand.",
			"affected_etfs":         []any{"SMH", "SOXX", "QQQ"},
		},
	}
	windows := []eventWindowResult{
		{EventID: event.ID, Symbol: "SMH", WindowName: "-1h_to_event", ReturnPct: 0.002, DataQuality: "complete"},
		{EventID: event.ID, Symbol: "SMH", WindowName: "event_to_+15m", ReturnPct: 0.011, DataQuality: "complete"},
		{EventID: event.ID, Symbol: "SMH", WindowName: "event_to_+1h", ReturnPct: 0.017, DataQuality: "complete"},
	}
	score := computePricedInScore(event.ID, "SMH", windows)

	bundle := buildResearchEvidenceBundle(event, "SMH", windows, score, nil)
	if bundle.EventID != event.ID || bundle.Symbol != "SMH" {
		t.Fatalf("unexpected identity fields: %#v", bundle)
	}
	if bundle.EventType != "semiconductor_ai" {
		t.Fatalf("event type = %q, want semiconductor_ai", bundle.EventType)
	}
	if bundle.PricedIn.Verdict != "not_priced_in" || bundle.PricedIn.Reason == "" {
		t.Fatalf("priced-in block not populated: %#v", bundle.PricedIn)
	}
	if bundle.PriceReaction.Post15M == 0 || bundle.PriceReaction.Post1H == 0 {
		t.Fatalf("price reaction not populated: %#v", bundle.PriceReaction)
	}
	if bundle.BeginnerSummary.WhatHappened == "" || bundle.BeginnerSummary.WalkAway == "" {
		t.Fatalf("beginner summary missing required text: %#v", bundle.BeginnerSummary)
	}
	if !bundle.Guardrails.AllowlistPass || !bundle.Guardrails.ApprovalRequired {
		t.Fatalf("guardrails not populated: %#v", bundle.Guardrails)
	}
}

func TestBackfillEventStudyPersistsEvidenceBundles(t *testing.T) {
	eventTime := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	store := &fakeBackfillStore{
		normalizedEvents: map[string]backfillNormalizedEvent{
			"event-1": {
				ID:            "event-1",
				EventTime:     eventTime,
				PrimarySymbol: "SMH",
				EventKind:     "news",
				Title:         "Nvidia chip demand surges",
				Summary:       "Semiconductor suppliers rally on AI demand.",
				SourceID:      "finnhub",
				Attributes: map[string]any{
					"event_type":            "semiconductor_ai",
					"classification_reason": "Headline affects semiconductor demand.",
				},
			},
		},
		candles: map[string][]marketdata.Candle{
			"SMH": {
				{Symbol: "SMH", Timestamp: eventTime.Add(-1 * time.Hour), Close: 100},
				{Symbol: "SMH", Timestamp: eventTime, Close: 101},
				{Symbol: "SMH", Timestamp: eventTime.Add(15 * time.Minute), Close: 102},
				{Symbol: "SMH", Timestamp: eventTime.Add(time.Hour), Close: 104},
			},
		},
	}
	runner := newBackfillRunner(store, nil)

	resp := runner.RunEventStudy(context.Background(), BackfillEventStudyRequest{
		EventIDs: []string{"event-1"},
		Symbols:  []string{"SMH"},
		Windows:  []string{"-1h_to_event", "event_to_+15m", "event_to_+1h"},
	})

	if resp.Status != "completed" {
		t.Fatalf("expected completed status, got %q failures=%#v", resp.Status, resp.Failures)
	}
	if store.summaryUpserts != 1 || len(store.lastBundles) != 1 {
		t.Fatalf("expected one persisted evidence bundle, count=%d bundles=%#v", store.summaryUpserts, store.lastBundles)
	}
	bundle := store.lastBundles[0]
	if bundle.EventType != "semiconductor_ai" || bundle.PricedIn.Verdict == "" || bundle.BeginnerSummary.WhatHappened == "" {
		t.Fatalf("bundle missing required evidence fields: %#v", bundle)
	}
}

func TestPricedInEngineVerdictsAndHardRejectReasons(t *testing.T) {
	windows := []eventWindowResult{
		{EventID: "event-priced", Symbol: "SPY", WindowName: "-4h_to_event", ReturnPct: 0.032, DataQuality: "complete"},
		{EventID: "event-priced", Symbol: "SPY", WindowName: "event_to_+15m", ReturnPct: 0.001, DataQuality: "complete"},
		{EventID: "event-priced", Symbol: "SPY", WindowName: "event_to_+1h", ReturnPct: -0.004, DataQuality: "complete"},
	}

	score := computePricedInScore("event-priced", "SPY", windows)
	if score.Verdict != "priced_in" {
		t.Fatalf("verdict = %q, want priced_in", score.Verdict)
	}
	if !score.HardReject {
		t.Fatal("expected priced_in score to be a hard reject")
	}
	if len(score.HardRejectReasons) == 0 || score.HardRejectReasons[0] != "priced_in" {
		t.Fatalf("hard reject reasons = %#v, want priced_in", score.HardRejectReasons)
	}
	if score.PreEvent4HReturn == 0 || score.PostEvent15MReturn == 0 {
		t.Fatalf("expected pre/post component returns to be populated: %#v", score)
	}
	if score.Reason == "" {
		t.Fatal("expected reason to be stored")
	}
}

func TestPricedInEngineDetectsNotPricedInConfirmation(t *testing.T) {
	windows := []eventWindowResult{
		{EventID: "event-fresh", Symbol: "QQQ", WindowName: "-1h_to_event", ReturnPct: 0.001, DataQuality: "complete"},
		{EventID: "event-fresh", Symbol: "QQQ", WindowName: "-4h_to_event", ReturnPct: 0.002, DataQuality: "complete"},
		{EventID: "event-fresh", Symbol: "QQQ", WindowName: "event_to_+5m", ReturnPct: 0.009, DataQuality: "complete"},
		{EventID: "event-fresh", Symbol: "QQQ", WindowName: "event_to_+15m", ReturnPct: 0.014, DataQuality: "complete"},
		{EventID: "event-fresh", Symbol: "QQQ", WindowName: "event_to_+1h", ReturnPct: 0.018, DataQuality: "complete"},
	}

	score := computePricedInScore("event-fresh", "QQQ", windows)
	if score.Verdict != "not_priced_in" {
		t.Fatalf("verdict = %q, want not_priced_in", score.Verdict)
	}
	if score.HardReject {
		t.Fatalf("did not expect hard reject: %#v", score.HardRejectReasons)
	}
	if score.PostEvent5MReturn == 0 || score.PostEvent15MReturn == 0 || score.PostEvent1HReturn == 0 {
		t.Fatalf("expected post-event components to be populated: %#v", score)
	}
}

func TestPricedInEngineMarksPoorDataUnclear(t *testing.T) {
	windows := []eventWindowResult{
		{EventID: "event-unclear", Symbol: "TLT", WindowName: "-1h_to_event", ReturnPct: 0.001, DataQuality: "missing"},
		{EventID: "event-unclear", Symbol: "TLT", WindowName: "event_to_+15m", ReturnPct: 0.006, DataQuality: "complete"},
	}

	score := computePricedInScore("event-unclear", "TLT", windows)
	if score.Verdict != "unclear" {
		t.Fatalf("verdict = %q, want unclear", score.Verdict)
	}
	if !score.HardReject {
		t.Fatal("expected unclear score to be a hard reject")
	}
	if len(score.HardRejectReasons) == 0 || score.HardRejectReasons[0] != "poor_data_quality" {
		t.Fatalf("hard reject reasons = %#v, want poor_data_quality", score.HardRejectReasons)
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

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
