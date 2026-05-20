package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"jax-trading-assistant/libs/marketdata"

	"github.com/google/uuid"
)

var phaseOneETFs = map[string]struct{}{
	"SPY": {}, "QQQ": {}, "DIA": {}, "IWM": {}, "XLK": {}, "XLF": {},
	"XLE": {}, "SMH": {}, "SOXX": {}, "TLT": {}, "GLD": {},
}

type BackfillCandlesRequest struct {
	Symbols   []string            `json:"symbols"`
	Timeframe string              `json:"timeframe"`
	From      string              `json:"from"`
	To        string              `json:"to"`
	Provider  string              `json:"provider"`
	Limit     int                 `json:"limit,omitempty"`
	Candles   []BackfillCandleRow `json:"candles,omitempty"`
}

type BackfillCandleRow struct {
	Symbol    string  `json:"symbol"`
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
	VWAP      float64 `json:"vwap,omitempty"`
}

type BackfillEventsRequest struct {
	Symbols  []string             `json:"symbols,omitempty"`
	Themes   []string             `json:"themes,omitempty"`
	From     string               `json:"from,omitempty"`
	To       string               `json:"to,omitempty"`
	Provider string               `json:"provider"`
	Events   []BackfillEventInput `json:"events"`
}

type BackfillEventInput struct {
	SourceEventID string         `json:"sourceEventId"`
	EventKind     string         `json:"eventKind"`
	EventTime     string         `json:"eventTime"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary,omitempty"`
	Severity      string         `json:"severity,omitempty"`
	Symbols       []string       `json:"symbols,omitempty"`
	CanonicalKey  string         `json:"canonicalKey,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
}

type BackfillEventStudyRequest struct {
	EventIDs []string `json:"eventIds"`
	Symbols  []string `json:"symbols"`
	Windows  []string `json:"windows,omitempty"`
}

type BackfillFailure struct {
	Symbol string `json:"symbol,omitempty"`
	Event  string `json:"event,omitempty"`
	Stage  string `json:"stage"`
	Error  string `json:"error"`
}

type BackfillRunResponse struct {
	RunID          string            `json:"runId"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	Provider       string            `json:"provider,omitempty"`
	Symbols        []string          `json:"symbols,omitempty"`
	InsertedRows   int               `json:"insertedRows,omitempty"`
	NormalizedRows int               `json:"normalizedRows,omitempty"`
	SymbolMaps     int               `json:"symbolMaps,omitempty"`
	Windows        int               `json:"windows,omitempty"`
	Scores         int               `json:"scores,omitempty"`
	Confounders    int               `json:"confounders,omitempty"`
	Failures       []BackfillFailure `json:"failures,omitempty"`
	StartedAt      time.Time         `json:"startedAt"`
	CompletedAt    time.Time         `json:"completedAt"`
}

type backfillStore interface {
	UpsertCandles(context.Context, []marketdata.Candle) (int, error)
	UpsertEvents(context.Context, []backfillEventRecord) (backfillEventWriteSummary, error)
	LoadEventStudyInputs(context.Context, []string, []string, time.Time, time.Time) ([]backfillNormalizedEvent, map[string][]marketdata.Candle, error)
	UpsertEventStudy(context.Context, []eventWindowResult, []pricedInScoreResult, []eventConfounderResult) (backfillEventStudyWriteSummary, error)
}

type candleFetcher interface {
	GetCandles(context.Context, string, marketdata.Timeframe, int) ([]marketdata.Candle, error)
}

type backfillRunner struct {
	store   backfillStore
	fetcher candleFetcher
}

func newBackfillRunner(store backfillStore, fetcher candleFetcher) *backfillRunner {
	return &backfillRunner{store: store, fetcher: fetcher}
}

type backfillManager struct {
	runner *backfillRunner
	mu     sync.RWMutex
	runs   map[string]BackfillRunResponse
}

func newBackfillManager(runner *backfillRunner) *backfillManager {
	return &backfillManager{runner: runner, runs: map[string]BackfillRunResponse{}}
}

func (m *backfillManager) save(resp BackfillRunResponse) BackfillRunResponse {
	if resp.RunID == "" {
		resp.RunID = uuid.NewString()
	}
	m.mu.Lock()
	m.runs[resp.RunID] = resp
	m.mu.Unlock()
	return resp
}

func (m *backfillManager) get(id string) (BackfillRunResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	resp, ok := m.runs[id]
	return resp, ok
}

func registerBackfillRoutes(mux *http.ServeMux, manager *backfillManager) {
	mux.HandleFunc("/research/backfill/candles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req BackfillCandlesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		writeBackfillRun(w, http.StatusCreated, manager.save(manager.runner.RunCandles(r.Context(), req)))
	})
	mux.HandleFunc("/research/backfill/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req BackfillEventsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		writeBackfillRun(w, http.StatusCreated, manager.save(manager.runner.RunEvents(r.Context(), req)))
	})
	mux.HandleFunc("/research/backfill/event-study", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req BackfillEventStudyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		writeBackfillRun(w, http.StatusCreated, manager.save(manager.runner.RunEventStudy(r.Context(), req)))
	})
	mux.HandleFunc("/research/backfill/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/research/backfill/runs/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		resp, ok := manager.get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeBackfillRun(w, http.StatusOK, resp)
	})
}

func writeBackfillRun(w http.ResponseWriter, status int, resp BackfillRunResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (r *backfillRunner) RunCandles(ctx context.Context, req BackfillCandlesRequest) BackfillRunResponse {
	start := time.Now().UTC()
	resp := BackfillRunResponse{RunID: uuid.NewString(), Type: "candles", Status: "completed", Provider: req.Provider, StartedAt: start}
	defer finishBackfillResponse(&resp)

	symbols, failures := normalizeETFSymbols(req.Symbols)
	resp.Symbols = symbols
	resp.Failures = append(resp.Failures, failures...)
	if req.Timeframe == "" {
		req.Timeframe = string(marketdata.Timeframe1Day)
	}

	var candles []marketdata.Candle
	for _, row := range req.Candles {
		c, err := row.toCandle()
		if err != nil {
			resp.Failures = append(resp.Failures, BackfillFailure{Symbol: row.Symbol, Stage: "parse_candle", Error: err.Error()})
			continue
		}
		if _, ok := phaseOneETFs[c.Symbol]; !ok {
			resp.Failures = append(resp.Failures, BackfillFailure{Symbol: c.Symbol, Stage: "allowlist", Error: "symbol is not in the phase-one ETF allowlist"})
			continue
		}
		candles = append(candles, c)
	}

	if len(req.Candles) == 0 {
		if r.fetcher == nil {
			resp.Failures = append(resp.Failures, BackfillFailure{Stage: "provider", Error: "no candle provider configured"})
		} else {
			limit := req.limitOrDefault()
			for _, symbol := range symbols {
				got, err := r.fetcher.GetCandles(ctx, symbol, marketdata.Timeframe(req.Timeframe), limit)
				if err != nil {
					resp.Failures = append(resp.Failures, BackfillFailure{Symbol: symbol, Stage: "fetch_candles", Error: err.Error()})
					continue
				}
				for _, c := range got {
					c.Symbol = strings.ToUpper(strings.TrimSpace(c.Symbol))
					if c.Symbol == "" {
						c.Symbol = symbol
					}
					candles = append(candles, c)
				}
			}
		}
	}

	if len(candles) > 0 {
		rows, err := r.store.UpsertCandles(ctx, candles)
		if err != nil {
			resp.Failures = append(resp.Failures, BackfillFailure{Stage: "write_candles", Error: err.Error()})
		} else {
			resp.InsertedRows = rows
		}
	}
	resp.Status = statusFrom(resp.InsertedRows > 0, resp.Failures)
	return resp
}

func (r *backfillRunner) RunEvents(ctx context.Context, req BackfillEventsRequest) BackfillRunResponse {
	start := time.Now().UTC()
	resp := BackfillRunResponse{RunID: uuid.NewString(), Type: "events", Status: "completed", Provider: req.Provider, StartedAt: start}
	defer finishBackfillResponse(&resp)

	records := make([]backfillEventRecord, 0, len(req.Events))
	for _, input := range req.Events {
		record, failures := buildEventRecord(req.Provider, input)
		resp.Failures = append(resp.Failures, failures...)
		if record.CanonicalKey != "" {
			records = append(records, record)
		}
	}
	if len(records) == 0 {
		if len(resp.Failures) == 0 {
			resp.Failures = append(resp.Failures, BackfillFailure{Stage: "events", Error: "no events supplied"})
		}
		resp.Status = "failed"
		return resp
	}

	summary, err := r.store.UpsertEvents(ctx, records)
	if err != nil {
		resp.Failures = append(resp.Failures, BackfillFailure{Stage: "write_events", Error: err.Error()})
		resp.Status = "failed"
		return resp
	}
	resp.InsertedRows = summary.RawRows
	resp.NormalizedRows = summary.NormalizedRows
	resp.SymbolMaps = summary.SymbolMaps
	resp.Status = statusFrom(summary.NormalizedRows > 0, resp.Failures)
	return resp
}

func (r *backfillRunner) RunEventStudy(ctx context.Context, req BackfillEventStudyRequest) BackfillRunResponse {
	start := time.Now().UTC()
	resp := BackfillRunResponse{RunID: uuid.NewString(), Type: "event-study", Status: "completed", StartedAt: start}
	defer finishBackfillResponse(&resp)

	symbols, failures := normalizeETFSymbols(req.Symbols)
	resp.Symbols = symbols
	resp.Failures = append(resp.Failures, failures...)
	if len(req.EventIDs) == 0 {
		resp.Failures = append(resp.Failures, BackfillFailure{Stage: "event_ids", Error: "eventIds is required"})
		resp.Status = "failed"
		return resp
	}
	if len(symbols) == 0 {
		resp.Failures = append(resp.Failures, BackfillFailure{Stage: "symbols", Error: "at least one allowlisted ETF symbol is required"})
		resp.Status = "failed"
		return resp
	}

	windows := parseEventStudyWindows(req.Windows)
	from, to := studyBounds(windows)
	events, candlesBySymbol, err := r.store.LoadEventStudyInputs(ctx, req.EventIDs, symbols, from, to)
	if err != nil {
		resp.Failures = append(resp.Failures, BackfillFailure{Stage: "load_event_study", Error: err.Error()})
		resp.Status = "failed"
		return resp
	}

	windowResults := make([]eventWindowResult, 0, len(events)*len(symbols)*len(windows))
	scoreInputs := map[string][]eventWindowResult{}
	for _, event := range events {
		for _, symbol := range symbols {
			candles := candlesBySymbol[symbol]
			for _, window := range windows {
				result, ok := computeEventWindow(event, symbol, window, candles)
				if !ok {
					resp.Failures = append(resp.Failures, BackfillFailure{Symbol: symbol, Event: event.ID, Stage: "compute_window", Error: "insufficient candles for " + window.Name})
					continue
				}
				windowResults = append(windowResults, result)
				scoreInputs[event.ID+"|"+symbol] = append(scoreInputs[event.ID+"|"+symbol], result)
			}
		}
	}
	scores := make([]pricedInScoreResult, 0, len(scoreInputs))
	for key, wins := range scoreInputs {
		parts := strings.Split(key, "|")
		scores = append(scores, computePricedInScore(parts[0], parts[1], wins))
	}
	summary, err := r.store.UpsertEventStudy(ctx, windowResults, scores, nil)
	if err != nil {
		resp.Failures = append(resp.Failures, BackfillFailure{Stage: "write_event_study", Error: err.Error()})
		resp.Status = "failed"
		return resp
	}
	resp.Windows = summary.Windows
	resp.Scores = summary.Scores
	resp.Confounders = summary.Confounders
	resp.Status = statusFrom(summary.Windows > 0, resp.Failures)
	return resp
}

func finishBackfillResponse(resp *BackfillRunResponse) {
	resp.CompletedAt = time.Now().UTC()
	if resp.Status == "" {
		resp.Status = statusFrom(false, resp.Failures)
	}
}

func statusFrom(hadWrites bool, failures []BackfillFailure) string {
	if len(failures) == 0 {
		return "completed"
	}
	if hadWrites {
		return "degraded"
	}
	return "failed"
}

func normalizeETFSymbols(raw []string) ([]string, []BackfillFailure) {
	if len(raw) == 0 {
		raw = []string{"SPY", "QQQ", "DIA", "IWM", "XLK", "XLF", "XLE", "SMH", "SOXX", "TLT", "GLD"}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	var failures []BackfillFailure
	for _, symbol := range raw {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		if _, ok := phaseOneETFs[symbol]; !ok {
			failures = append(failures, BackfillFailure{Symbol: symbol, Stage: "allowlist", Error: "symbol is not in the phase-one ETF allowlist"})
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	sort.Strings(out)
	return out, failures
}

func (req BackfillCandlesRequest) limitOrDefault() int {
	if req.Limit > 0 {
		return req.Limit
	}
	from, okFrom := parseBackfillTime(req.From)
	to, okTo := parseBackfillTime(req.To)
	if okFrom && okTo && to.After(from) {
		days := int(math.Ceil(to.Sub(from).Hours()/24)) + 1
		if req.Timeframe == string(marketdata.Timeframe1Day) || req.Timeframe == "" {
			return max(days, 1)
		}
		return max(days*390, 1)
	}
	return 520
}

func (row BackfillCandleRow) toCandle() (marketdata.Candle, error) {
	ts, ok := parseBackfillTime(row.Timestamp)
	if !ok {
		return marketdata.Candle{}, fmt.Errorf("timestamp must be RFC3339 or YYYY-MM-DD")
	}
	return marketdata.Candle{
		Symbol:    strings.ToUpper(strings.TrimSpace(row.Symbol)),
		Timestamp: ts,
		Open:      row.Open,
		High:      row.High,
		Low:       row.Low,
		Close:     row.Close,
		Volume:    row.Volume,
		VWAP:      row.VWAP,
	}, nil
}

func parseBackfillTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(dateFmt, raw); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

type backfillEventRecord struct {
	SourceID      string
	SourceEventID string
	CanonicalKey  string
	EventKind     string
	EventTime     time.Time
	Title         string
	Summary       string
	Severity      string
	PrimarySymbol string
	Symbols       []string
	Attributes    map[string]any
	Payload       map[string]any
	ContentHash   string
}

type backfillEventWriteSummary struct {
	RawRows        int
	NormalizedRows int
	SymbolMaps     int
}

func buildEventRecord(provider string, input BackfillEventInput) (backfillEventRecord, []BackfillFailure) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "calendar"
	}
	eventTime, ok := parseBackfillTime(input.EventTime)
	if !ok {
		return backfillEventRecord{}, []BackfillFailure{{Event: input.SourceEventID, Stage: "parse_event", Error: "eventTime must be RFC3339 or YYYY-MM-DD"}}
	}
	symbols, failures := normalizeETFSymbols(input.Symbols)
	if len(symbols) == 0 {
		failures = append(failures, BackfillFailure{Event: input.SourceEventID, Stage: "symbols", Error: "event must map to at least one allowlisted ETF"})
		return backfillEventRecord{}, failures
	}
	sourceEventID := strings.TrimSpace(input.SourceEventID)
	if sourceEventID == "" {
		sourceEventID = deterministicEventID(provider, input.Title, eventTime)
	}
	eventKind := strings.TrimSpace(input.EventKind)
	if eventKind == "" {
		eventKind = "news"
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = eventKind + " event"
	}
	canonicalKey := strings.TrimSpace(input.CanonicalKey)
	if canonicalKey == "" {
		canonicalKey = provider + ":" + sourceEventID
	}
	severity := strings.TrimSpace(input.Severity)
	if severity == "" {
		severity = "unknown"
	}
	payload := input.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	attributes := input.Attributes
	if attributes == nil {
		attributes = map[string]any{}
	}
	hashPayload, _ := json.Marshal(map[string]any{
		"provider": provider, "sourceEventId": sourceEventID, "title": title, "summary": input.Summary, "time": eventTime,
	})
	sum := sha256.Sum256(hashPayload)
	return backfillEventRecord{
		SourceID:      provider,
		SourceEventID: sourceEventID,
		CanonicalKey:  canonicalKey,
		EventKind:     eventKind,
		EventTime:     eventTime,
		Title:         title,
		Summary:       input.Summary,
		Severity:      severity,
		PrimarySymbol: symbols[0],
		Symbols:       symbols,
		Attributes:    attributes,
		Payload:       payload,
		ContentHash:   hex.EncodeToString(sum[:]),
	}, failures
}

func deterministicEventID(provider, title string, eventTime time.Time) string {
	sum := sha256.Sum256([]byte(provider + "|" + title + "|" + eventTime.Format(time.RFC3339)))
	return hex.EncodeToString(sum[:12])
}

type backfillNormalizedEvent struct {
	ID            string
	EventTime     time.Time
	PrimarySymbol string
}

type eventStudyWindow struct {
	Name   string
	Before time.Duration
	After  time.Duration
}

type eventWindowResult struct {
	EventID        string
	Symbol         string
	WindowName     string
	WindowStart    time.Time
	WindowEnd      time.Time
	BaselinePrice  float64
	EventPrice     float64
	EndPrice       float64
	ReturnPct      float64
	DriftPct       float64
	VolatilityPct  float64
	DataQuality    string
	ObservationCnt int
}

type pricedInScoreResult struct {
	EventID    string
	Symbol     string
	Score      float64
	Verdict    string
	Components map[string]any
}

type eventConfounderResult struct {
	EventID            string
	ConfoundingEventID string
	Symbol             string
	RelationshipType   string
	RelevanceScore     float64
	Notes              string
}

type backfillEventStudyWriteSummary struct {
	Windows     int
	Scores      int
	Confounders int
}

func parseEventStudyWindows(raw []string) []eventStudyWindow {
	if len(raw) == 0 {
		raw = []string{"-1d_to_event", "-4h_to_event", "-1h_to_event", "event_to_+5m", "event_to_+15m", "event_to_+1h", "event_to_+4h", "event_to_+1d"}
	}
	out := make([]eventStudyWindow, 0, len(raw))
	for _, name := range raw {
		switch strings.TrimSpace(name) {
		case "-1d_to_event":
			out = append(out, eventStudyWindow{Name: "-1d_to_event", Before: 24 * time.Hour})
		case "-4h_to_event":
			out = append(out, eventStudyWindow{Name: "-4h_to_event", Before: 4 * time.Hour})
		case "-1h_to_event":
			out = append(out, eventStudyWindow{Name: "-1h_to_event", Before: time.Hour})
		case "event_to_+5m":
			out = append(out, eventStudyWindow{Name: "event_to_+5m", After: 5 * time.Minute})
		case "event_to_+15m":
			out = append(out, eventStudyWindow{Name: "event_to_+15m", After: 15 * time.Minute})
		case "event_to_+1h":
			out = append(out, eventStudyWindow{Name: "event_to_+1h", After: time.Hour})
		case "event_to_+4h":
			out = append(out, eventStudyWindow{Name: "event_to_+4h", After: 4 * time.Hour})
		case "event_to_+1d":
			out = append(out, eventStudyWindow{Name: "event_to_+1d", After: 24 * time.Hour})
		}
	}
	if len(out) == 0 {
		return parseEventStudyWindows(nil)
	}
	return out
}

func studyBounds(windows []eventStudyWindow) (time.Time, time.Time) {
	maxBefore := time.Duration(0)
	maxAfter := time.Duration(0)
	for _, w := range windows {
		if w.Before > maxBefore {
			maxBefore = w.Before
		}
		if w.After > maxAfter {
			maxAfter = w.After
		}
	}
	now := time.Now().UTC()
	return now.Add(-maxBefore), now.Add(maxAfter)
}

func computeEventWindow(event backfillNormalizedEvent, symbol string, window eventStudyWindow, candles []marketdata.Candle) (eventWindowResult, bool) {
	if len(candles) == 0 {
		return eventWindowResult{}, false
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].Timestamp.Before(candles[j].Timestamp) })
	start := event.EventTime.Add(-window.Before)
	end := event.EventTime.Add(window.After)
	if window.Before == 0 {
		start = event.EventTime
	}
	if window.After == 0 {
		end = event.EventTime
	}
	startCandle, okStart := firstCandleAtOrAfter(candles, start)
	endCandle, okEnd := lastCandleAtOrBefore(candles, end)
	eventCandle, okEvent := nearestCandleAtOrBefore(candles, event.EventTime)
	if !okStart || !okEnd || !okEvent || startCandle.Close == 0 {
		return eventWindowResult{}, false
	}
	observations := 0
	for _, c := range candles {
		if !c.Timestamp.Before(start) && !c.Timestamp.After(end) {
			observations++
		}
	}
	returnPct := (endCandle.Close - startCandle.Close) / startCandle.Close
	driftPct := (eventCandle.Close - startCandle.Close) / startCandle.Close
	return eventWindowResult{
		EventID:        event.ID,
		Symbol:         symbol,
		WindowName:     window.Name,
		WindowStart:    start,
		WindowEnd:      end,
		BaselinePrice:  startCandle.Close,
		EventPrice:     eventCandle.Close,
		EndPrice:       endCandle.Close,
		ReturnPct:      returnPct,
		DriftPct:       driftPct,
		VolatilityPct:  windowVolatility(candles, start, end),
		DataQuality:    "complete",
		ObservationCnt: observations,
	}, true
}

func firstCandleAtOrAfter(candles []marketdata.Candle, t time.Time) (marketdata.Candle, bool) {
	for _, c := range candles {
		if !c.Timestamp.Before(t) {
			return c, true
		}
	}
	return marketdata.Candle{}, false
}

func lastCandleAtOrBefore(candles []marketdata.Candle, t time.Time) (marketdata.Candle, bool) {
	var out marketdata.Candle
	ok := false
	for _, c := range candles {
		if c.Timestamp.After(t) {
			break
		}
		out = c
		ok = true
	}
	return out, ok
}

func nearestCandleAtOrBefore(candles []marketdata.Candle, t time.Time) (marketdata.Candle, bool) {
	return lastCandleAtOrBefore(candles, t)
}

func windowVolatility(candles []marketdata.Candle, start, end time.Time) float64 {
	var returns []float64
	var prev *marketdata.Candle
	for i := range candles {
		c := candles[i]
		if c.Timestamp.Before(start) || c.Timestamp.After(end) {
			continue
		}
		if prev != nil && prev.Close != 0 {
			returns = append(returns, (c.Close-prev.Close)/prev.Close)
		}
		prev = &candles[i]
	}
	if len(returns) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range returns {
		mean += v
	}
	mean /= float64(len(returns))
	var sumSq float64
	for _, v := range returns {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(returns)-1))
}

func computePricedInScore(eventID, symbol string, windows []eventWindowResult) pricedInScoreResult {
	pre := 0.0
	post := 0.0
	for _, w := range windows {
		if strings.Contains(w.WindowName, "_to_event") {
			pre += math.Abs(w.ReturnPct)
		}
		if strings.Contains(w.WindowName, "event_to_") {
			post += math.Abs(w.ReturnPct)
		}
	}
	score := 0.5
	if pre+post > 0 {
		score = pre / (pre + post)
	}
	verdict := "unclear"
	switch {
	case score >= 0.67:
		verdict = "priced_in"
	case score <= 0.33:
		verdict = "not_priced_in"
	default:
		verdict = "partially_priced_in"
	}
	return pricedInScoreResult{
		EventID: eventID,
		Symbol:  symbol,
		Score:   score,
		Verdict: verdict,
		Components: map[string]any{
			"pre_drift_abs":  pre,
			"post_drift_abs": post,
			"window_count":   len(windows),
		},
	}
}

func pricedInReason(score pricedInScoreResult) string {
	pre, _ := score.Components["pre_drift_abs"].(float64)
	post, _ := score.Components["post_drift_abs"].(float64)
	return fmt.Sprintf("priced-in score %.2f from pre-event absolute drift %.4f and post-event absolute drift %.4f", score.Score, pre, post)
}

type sqlBackfillStore struct {
	db *sql.DB
}

func newSQLBackfillStore(db *sql.DB) *sqlBackfillStore {
	return &sqlBackfillStore{db: db}
}

func (s *sqlBackfillStore) UpsertCandles(ctx context.Context, candles []marketdata.Candle) (int, error) {
	for _, c := range candles {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO candles (symbol, timestamp, open, high, low, close, volume, vwap)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (symbol, timestamp) DO UPDATE SET
				open = EXCLUDED.open,
				high = EXCLUDED.high,
				low = EXCLUDED.low,
				close = EXCLUDED.close,
				volume = EXCLUDED.volume,
				vwap = EXCLUDED.vwap`,
			c.Symbol, c.Timestamp, c.Open, c.High, c.Low, c.Close, c.Volume, c.VWAP,
		)
		if err != nil {
			return 0, err
		}
	}
	return len(candles), nil
}

func (s *sqlBackfillStore) UpsertEvents(ctx context.Context, events []backfillEventRecord) (backfillEventWriteSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backfillEventWriteSummary{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	var summary backfillEventWriteSummary
	for _, event := range events {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_sources (id, display_name, provider_type, enabled, priority, metadata)
			VALUES ($1, $1, 'external', TRUE, 100, '{}'::jsonb)
			ON CONFLICT (id) DO NOTHING`, event.SourceID); err != nil {
			return backfillEventWriteSummary{}, err
		}
		payloadJSON, _ := json.Marshal(event.Payload)
		attributesJSON, _ := json.Marshal(event.Attributes)
		var rawID string
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO event_raw (
				source_id, source_event_id, event_kind, event_time, symbol, payload,
				content_hash, data_source_type, source_provider, is_synthetic, provenance_verified_at
			)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, 'real', $1, FALSE, NOW())
			ON CONFLICT (source_id, source_event_id) DO UPDATE SET
				event_kind = EXCLUDED.event_kind,
				event_time = EXCLUDED.event_time,
				symbol = EXCLUDED.symbol,
				payload = EXCLUDED.payload,
				content_hash = EXCLUDED.content_hash,
				source_provider = EXCLUDED.source_provider,
				provenance_verified_at = NOW()
			RETURNING id`,
			event.SourceID, event.SourceEventID, event.EventKind, event.EventTime, event.PrimarySymbol, string(payloadJSON), event.ContentHash,
		).Scan(&rawID); err != nil {
			return backfillEventWriteSummary{}, err
		}
		summary.RawRows++

		var normalizedID string
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO event_normalized (
				raw_event_id, canonical_key, event_kind, title, summary, severity,
				event_time, source_id, primary_symbol, confidence, attributes,
				data_source_type, source_provider, is_synthetic, provenance_verified_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1.0, $10::jsonb, 'real', $8, FALSE, NOW())
			ON CONFLICT (canonical_key) DO UPDATE SET
				raw_event_id = EXCLUDED.raw_event_id,
				event_kind = EXCLUDED.event_kind,
				title = EXCLUDED.title,
				summary = EXCLUDED.summary,
				severity = EXCLUDED.severity,
				event_time = EXCLUDED.event_time,
				source_id = EXCLUDED.source_id,
				primary_symbol = EXCLUDED.primary_symbol,
				attributes = EXCLUDED.attributes,
				source_provider = EXCLUDED.source_provider,
				provenance_verified_at = NOW()
			RETURNING id`,
			rawID, event.CanonicalKey, event.EventKind, event.Title, event.Summary, event.Severity, event.EventTime, event.SourceID, event.PrimarySymbol, string(attributesJSON),
		).Scan(&normalizedID); err != nil {
			return backfillEventWriteSummary{}, err
		}
		summary.NormalizedRows++

		for i, symbol := range event.Symbols {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO event_symbol_map (normalized_event_id, symbol, relevance, mapping_method, is_primary)
				VALUES ($1, $2, $3, 'backfill', $4)
				ON CONFLICT (normalized_event_id, symbol) DO UPDATE SET
					relevance = EXCLUDED.relevance,
					mapping_method = EXCLUDED.mapping_method,
					is_primary = EXCLUDED.is_primary`,
				normalizedID, symbol, 1.0, i == 0,
			)
			if err != nil {
				return backfillEventWriteSummary{}, err
			}
			summary.SymbolMaps++
		}
	}
	if err := tx.Commit(); err != nil {
		return backfillEventWriteSummary{}, err
	}
	return summary, nil
}

func (s *sqlBackfillStore) LoadEventStudyInputs(ctx context.Context, eventIDs []string, symbols []string, from, to time.Time) ([]backfillNormalizedEvent, map[string][]marketdata.Candle, error) {
	events := make([]backfillNormalizedEvent, 0, len(eventIDs))
	for _, id := range eventIDs {
		var event backfillNormalizedEvent
		err := s.db.QueryRowContext(ctx, `
			SELECT id::text, event_time, COALESCE(primary_symbol, '')
			FROM event_normalized
			WHERE id = $1`, id,
		).Scan(&event.ID, &event.EventTime, &event.PrimarySymbol)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, nil, err
		}
		events = append(events, event)
	}
	candles := map[string][]marketdata.Candle{}
	if len(events) == 0 {
		return events, candles, nil
	}
	minTime, maxTime := eventStudyLoadBounds(events, from, to)
	for _, symbol := range symbols {
		rows, err := s.db.QueryContext(ctx, `
			SELECT symbol, timestamp, open, high, low, close, volume, COALESCE(vwap, 0)
			FROM candles
			WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
			ORDER BY timestamp ASC`, symbol, minTime, maxTime,
		)
		if err != nil {
			return nil, nil, err
		}
		for rows.Next() {
			var c marketdata.Candle
			if err := rows.Scan(&c.Symbol, &c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.VWAP); err != nil {
				rows.Close()
				return nil, nil, err
			}
			candles[symbol] = append(candles[symbol], c)
		}
		if err := rows.Close(); err != nil {
			return nil, nil, err
		}
	}
	return events, candles, nil
}

func eventStudyLoadBounds(events []backfillNormalizedEvent, from, to time.Time) (time.Time, time.Time) {
	minTime := events[0].EventTime
	maxTime := events[0].EventTime
	for _, event := range events {
		if event.EventTime.Before(minTime) {
			minTime = event.EventTime
		}
		if event.EventTime.After(maxTime) {
			maxTime = event.EventTime
		}
	}
	before := time.Since(from)
	after := to.Sub(time.Now().UTC())
	if before < 0 {
		before = 24 * time.Hour
	}
	if after < 0 {
		after = 24 * time.Hour
	}
	return minTime.Add(-before), maxTime.Add(after)
}

func (s *sqlBackfillStore) UpsertEventStudy(ctx context.Context, windows []eventWindowResult, scores []pricedInScoreResult, confounders []eventConfounderResult) (backfillEventStudyWriteSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return backfillEventStudyWriteSummary{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	for _, w := range windows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO event_windows (
				event_id, symbol, window_name, window_start, window_end,
				price_before, price_after, return_pct, abnormal_return_pct,
				volatility_adjusted_move, data_quality
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (event_id, symbol, window_name) DO UPDATE SET
				window_start = EXCLUDED.window_start,
				window_end = EXCLUDED.window_end,
				price_before = EXCLUDED.price_before,
				price_after = EXCLUDED.price_after,
				return_pct = EXCLUDED.return_pct,
				abnormal_return_pct = EXCLUDED.abnormal_return_pct,
				volatility_adjusted_move = EXCLUDED.volatility_adjusted_move,
				data_quality = EXCLUDED.data_quality`,
			w.EventID, w.Symbol, w.WindowName, w.WindowStart, w.WindowEnd, w.BaselinePrice,
			w.EndPrice, w.ReturnPct, w.ReturnPct, w.VolatilityPct, w.DataQuality,
		)
		if err != nil {
			return backfillEventStudyWriteSummary{}, err
		}
	}
	for _, score := range scores {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO event_priced_in_scores (event_id, symbol, priced_in_score, verdict, reason)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (event_id, symbol) DO UPDATE SET
				priced_in_score = EXCLUDED.priced_in_score,
				verdict = EXCLUDED.verdict,
				reason = EXCLUDED.reason`,
			score.EventID, score.Symbol, score.Score, score.Verdict, pricedInReason(score),
		)
		if err != nil {
			return backfillEventStudyWriteSummary{}, err
		}
	}
	for _, confounder := range confounders {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO event_confounders (
				event_id, confounding_event_id, symbol, relationship_type, time_distance_minutes, relevance_score, reason
			)
			VALUES ($1, $2, $3, $4, 0, $5, $6)
			ON CONFLICT (event_id, confounding_event_id, symbol) DO UPDATE SET
				relationship_type = EXCLUDED.relationship_type,
				time_distance_minutes = EXCLUDED.time_distance_minutes,
				relevance_score = EXCLUDED.relevance_score,
				reason = EXCLUDED.reason`,
			confounder.EventID, confounder.ConfoundingEventID, confounder.Symbol,
			confounder.RelationshipType, confounder.RelevanceScore, confounder.Notes,
		)
		if err != nil {
			return backfillEventStudyWriteSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return backfillEventStudyWriteSummary{}, err
	}
	return backfillEventStudyWriteSummary{Windows: len(windows), Scores: len(scores), Confounders: len(confounders)}, nil
}

func buildBackfillCandleFetcherFromEnv() candleFetcher {
	var providers []marketdata.ProviderConfig
	if raw := strings.TrimSpace(os.Getenv("IB_BRIDGE_URL")); raw != "" {
		providers = append(providers, marketdata.ProviderConfig{Name: marketdata.ProviderIBBridge, IBBridgeURL: raw, Enabled: true, Priority: len(providers) + 1})
	}
	if raw := strings.TrimSpace(os.Getenv("POLYGON_API_KEY")); raw != "" {
		providers = append(providers, marketdata.ProviderConfig{Name: marketdata.ProviderPolygon, APIKey: raw, Enabled: true, Priority: len(providers) + 1})
	}
	if raw := strings.TrimSpace(os.Getenv("FINANCIAL_DATASETS_API_KEY")); raw != "" {
		providers = append(providers, marketdata.ProviderConfig{Name: marketdata.ProviderFinancialDatasets, APIKey: raw, Enabled: true, Priority: len(providers) + 1})
	}
	if len(providers) == 0 {
		return nil
	}
	client, err := marketdata.NewClient(&marketdata.Config{
		Providers: providers,
		Cache:     marketdata.CacheConfig{Enabled: false, TTL: 5 * time.Minute},
	})
	if err != nil {
		return nil
	}
	return client
}
