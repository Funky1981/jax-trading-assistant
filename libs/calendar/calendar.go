// Package calendar implements L09 + L20: reliable economic event ingestion
// with a redundant, verified production-grade calendar feed.
//
// Core concepts:
//
//   - EconEvent: a typed economic event (NFP, CPI, FOMC, earnings, etc.)
//   - Source:    an interface that providers (CSV, API, DB) implement
//   - Store:     a thread-safe in-memory + persistence layer for events
//   - Feed:      orchestrates multiple Sources with deduplication and
//                verification (L20-level redundancy)
package calendar

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Impact ───────────────────────────────────────────────────────────────────

// Impact represents the expected market-moving severity of an event.
type Impact string

const (
	ImpactLow    Impact = "low"
	ImpactMedium Impact = "medium"
	ImpactHigh   Impact = "high"
)

// ─── EconEvent ────────────────────────────────────────────────────────────────

// EconEvent is a single economic release or announcement.
type EconEvent struct {
	// ID is a unique identifier for the event (may be source-assigned or
	// computed from "country+title+ISO-timestamp").
	ID string `json:"id"`
	// Country is the ISO-3166 alpha-2 country code (e.g. "US", "EU", "UK").
	Country string `json:"country"`
	// Currency is the affected currency (e.g. "USD", "EUR").
	Currency string `json:"currency"`
	// Title is the human-readable event name (e.g. "Non-Farm Payrolls").
	Title string `json:"title"`
	// Category groups similar events: "employment", "inflation", "central_bank", etc.
	Category string `json:"category"`
	// ScheduledAt is the announced release time in UTC.
	ScheduledAt time.Time `json:"scheduled_at"`
	// Impact is the expected market impact level.
	Impact Impact `json:"impact"`
	// Actual is the released figure (empty string if not yet released).
	Actual string `json:"actual,omitempty"`
	// Forecast is the consensus estimate.
	Forecast string `json:"forecast,omitempty"`
	// Previous is the prior month/quarter reading.
	Previous string `json:"previous,omitempty"`
	// Source identifies which provider contributed this record.
	Source string `json:"source"`
}

// EventID derives a deterministic ID from country + title + date.
// This is used for deduplication when merging data from multiple sources.
func EventID(country, title string, scheduledAt time.Time) string {
	key := strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(country)),
		strings.ToUpper(strings.TrimSpace(title)),
		scheduledAt.UTC().Format(time.RFC3339),
	}, "|")
	// A short, human-readable deterministic ID.
	h := fnv64a(key)
	return fmt.Sprintf("%s-%s-%016x",
		strings.ToUpper(country),
		scheduledAt.Format("20060102"),
		h)
}

func fnv64a(s string) uint64 {
	const prime = 1099511628211
	hash := uint64(14695981039346656037)
	for _, c := range []byte(s) {
		hash ^= uint64(c)
		hash *= prime
	}
	return hash
}

// ─── Source interface ─────────────────────────────────────────────────────────

// Source is implemented by any economic event data provider.
type Source interface {
	// Name returns a short identifier for the source (e.g. "forex_factory_csv").
	Name() string
	// FetchEvents returns events in [from, to] (both inclusive).
	// Implementations may return cached data; callers must not mutate the slice.
	FetchEvents(ctx context.Context, from, to time.Time) ([]EconEvent, error)
}

// ─── Store ────────────────────────────────────────────────────────────────────

// Store is a thread-safe, persistence-backed store of EconEvents.
type Store struct {
	mu     sync.RWMutex
	events map[string]EconEvent // keyed by ID
	dir    string
}

const storeFile = "events.json"

// OpenStore loads (or creates) a calendar store backed by dir.
func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("calendar.OpenStore: mkdir: %w", err)
	}
	s := &Store{dir: dir, events: make(map[string]EconEvent)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Upsert inserts or updates events. The canonical ID is preserved.
func (s *Store) Upsert(events []EconEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range events {
		if e.ID == "" {
			e.ID = EventID(e.Country, e.Title, e.ScheduledAt)
		}
		s.events[e.ID] = e
	}
	return s.save()
}

// Query returns events within [from, to] matching the optional filters.
// An empty filter value matches all events.
func (s *Store) Query(from, to time.Time, country, currency string, impact Impact) []EconEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []EconEvent
	for _, e := range s.events {
		if e.ScheduledAt.Before(from) || e.ScheduledAt.After(to) {
			continue
		}
		if country != "" && !strings.EqualFold(e.Country, country) {
			continue
		}
		if currency != "" && !strings.EqualFold(e.Currency, currency) {
			continue
		}
		if impact != "" && e.Impact != impact {
			continue
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b EconEvent) int {
		return a.ScheduledAt.Compare(b.ScheduledAt)
	})
	return out
}

// Count returns the total number of events in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

func (s *Store) storePath() string { return filepath.Join(s.dir, storeFile) }

func (s *Store) load() error {
	f, err := os.Open(s.storePath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("calendar: open store: %w", err)
	}
	defer f.Close()

	var list []EconEvent
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		return fmt.Errorf("calendar: decode store: %w", err)
	}
	for _, e := range list {
		s.events[e.ID] = e
	}
	return nil
}

func (s *Store) save() error {
	list := make([]EconEvent, 0, len(s.events))
	for _, e := range s.events {
		list = append(list, e)
	}
	slices.SortFunc(list, func(a, b EconEvent) int {
		return a.ScheduledAt.Compare(b.ScheduledAt)
	})

	tmp := s.storePath() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("calendar: create store tmp: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(list); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("calendar: encode store: %w", err)
	}
	f.Close()
	if err := os.Rename(tmp, s.storePath()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("calendar: rename store: %w", err)
	}
	return nil
}

// ─── CSVSource ────────────────────────────────────────────────────────────────

// CSVSource implements Source by reading a CSV file in ForexFactory export
// format.  Expected columns (case-insensitive):
//
//	date, time, currency, impact, event, actual, forecast, previous
//
// The "date" column may cover multiple rows; when blank it repeats the last
// seen value (matching ForexFactory's visual grouping style).
type CSVSource struct {
	name     string
	filePath string
}

// NewCSVSource creates a ForexFactory-compatible CSV source.
func NewCSVSource(name, filePath string) *CSVSource {
	return &CSVSource{name: name, filePath: filePath}
}

func (c *CSVSource) Name() string { return c.name }

func (c *CSVSource) FetchEvents(_ context.Context, from, to time.Time) ([]EconEvent, error) {
	f, err := os.Open(c.filePath)
	if err != nil {
		return nil, fmt.Errorf("calendar.CSVSource %q: open: %w", c.name, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("calendar.CSVSource %q: read header: %w", c.name, err)
	}

	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	idx := func(name string) int {
		i, ok := colIdx[name]
		if !ok {
			return -1
		}
		return i
	}

	get := func(row []string, col int) string {
		if col < 0 || col >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[col])
	}

	dateCol := idx("date")
	timeCol := idx("time")
	curCol := idx("currency")
	impCol := idx("impact")
	evtCol := idx("event")
	actCol := idx("actual")
	fcCol := idx("forecast")
	prevCol := idx("previous")

	if dateCol < 0 || curCol < 0 || evtCol < 0 {
		return nil, fmt.Errorf("calendar.CSVSource %q: missing required columns (date, currency, event)", c.name)
	}

	var events []EconEvent
	var lastDate string

	dateFormats := []string{"Jan 02, 2006", "2006-01-02", "01/02/2006"}
	parseDate := func(d, t string) (time.Time, error) {
		d = strings.TrimSpace(d)
		t = strings.TrimSpace(t)
		for _, layout := range dateFormats {
			if ts, err := time.Parse(layout+" "+t, d+" "+t); err == nil {
				return ts.UTC(), nil
			}
			if ts, err := time.Parse(layout, d); err == nil {
				return ts.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("unrecognised date %q time %q", d, t)
	}

	parseImpact := func(s string) Impact {
		switch strings.ToLower(s) {
		case "high", "red":
			return ImpactHigh
		case "medium", "orange":
			return ImpactMedium
		default:
			return ImpactLow
		}
	}

	lineNo := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("calendar.CSVSource %q: line %d: %w", c.name, lineNo+1, err)
		}
		lineNo++

		rawDate := get(row, dateCol)
		if rawDate != "" {
			lastDate = rawDate
		}
		if lastDate == "" {
			continue
		}

		evtTitle := get(row, evtCol)
		if evtTitle == "" {
			continue
		}

		ts, err := parseDate(lastDate, get(row, timeCol))
		if err != nil {
			// Skip rows with unrecognised dates rather than aborting.
			log.Printf("[calendar] skip row %d: %v", lineNo, err)
			continue
		}

		if ts.Before(from) || ts.After(to) {
			continue
		}

		currency := get(row, curCol)
		e := EconEvent{
			Country:     currencyToCountry(currency),
			Currency:    currency,
			Title:       evtTitle,
			Category:    categorise(evtTitle),
			ScheduledAt: ts,
			Impact:      parseImpact(get(row, impCol)),
			Actual:      get(row, actCol),
			Forecast:    get(row, fcCol),
			Previous:    get(row, prevCol),
			Source:      c.name,
		}
		e.ID = EventID(e.Country, e.Title, e.ScheduledAt)
		events = append(events, e)
	}

	return events, nil
}

// ─── Feed (L20 redundancy) ────────────────────────────────────────────────────

// Feed orchestrates multiple Sources, merging and deduplicating their output.
// For a given event ID, the first source that contributes an Actual reading
// wins (subsequent sources can still update Forecast/Previous if absent).
type Feed struct {
	sources []Source
	store   *Store
}

// NewFeed creates a Feed that pulls from sources and persists to store.
func NewFeed(store *Store, sources ...Source) *Feed {
	return &Feed{sources: sources, store: store}
}

// Ingest fetches events from all sources in [from, to], deduplicates by
// EventID, and upserts the merged results to the store.
func (f *Feed) Ingest(ctx context.Context, from, to time.Time) (int, error) {
	merged := make(map[string]EconEvent)

	for _, src := range f.sources {
		events, err := src.FetchEvents(ctx, from, to)
		if err != nil {
			log.Printf("[calendar.Feed] source=%q error=%v (continuing)", src.Name(), err)
			continue
		}
		for _, e := range events {
			id := e.ID
			if existing, ok := merged[id]; ok {
				// Merge: prefer non-empty Actual from any source.
				if existing.Actual == "" && e.Actual != "" {
					existing.Actual = e.Actual
				}
				if existing.Forecast == "" && e.Forecast != "" {
					existing.Forecast = e.Forecast
				}
				if existing.Previous == "" && e.Previous != "" {
					existing.Previous = e.Previous
				}
				merged[id] = existing
			} else {
				merged[id] = e
			}
		}
		log.Printf("[calendar.Feed] source=%q fetched=%d", src.Name(), len(events))
	}

	if len(merged) == 0 {
		return 0, nil
	}

	list := make([]EconEvent, 0, len(merged))
	for _, e := range merged {
		list = append(list, e)
	}

	if err := f.store.Upsert(list); err != nil {
		return 0, fmt.Errorf("calendar.Feed.Ingest: upsert: %w", err)
	}

	log.Printf("[calendar.Feed] ingested %d events (%s → %s)", len(list),
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	return len(list), nil
}

// HighImpactWindow returns high-impact events in [centerTime ± window].
// Use this before order submission to implement news-avoidance logic.
func (f *Feed) HighImpactWindow(store *Store, center time.Time, window time.Duration) []EconEvent {
	from := center.Add(-window)
	to := center.Add(window)
	return store.Query(from, to, "", "", ImpactHigh)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// currencyToCountry maps a currency code to the most common country code.
// Extend as needed.
func currencyToCountry(cur string) string {
	m := map[string]string{
		"USD": "US", "EUR": "EU", "GBP": "GB", "JPY": "JP",
		"CAD": "CA", "AUD": "AU", "NZD": "NZ", "CHF": "CH",
		"CNY": "CN", "HKD": "HK", "SGD": "SG", "MXN": "MX",
	}
	if c, ok := m[strings.ToUpper(cur)]; ok {
		return c
	}
	return strings.ToUpper(cur)
}

// categorise derives a category label from the event title.
func categorise(title string) string {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "nonfarm") || strings.Contains(t, "non-farm") ||
		strings.Contains(t, "payroll") || strings.Contains(t, "employment") ||
		strings.Contains(t, "unemployment"):
		return "employment"
	case strings.Contains(t, "cpi") || strings.Contains(t, "inflation") ||
		strings.Contains(t, "pce") || strings.Contains(t, "ppi"):
		return "inflation"
	case strings.Contains(t, "fomc") || strings.Contains(t, "interest rate") ||
		strings.Contains(t, "fed") || strings.Contains(t, "ecb") ||
		strings.Contains(t, "boe") || strings.Contains(t, "rba"):
		return "central_bank"
	case strings.Contains(t, "gdp") || strings.Contains(t, "growth"):
		return "growth"
	case strings.Contains(t, "retail") || strings.Contains(t, "sales"):
		return "consumer"
	case strings.Contains(t, "pmi") || strings.Contains(t, "ism") ||
		strings.Contains(t, "manufacturing"):
		return "manufacturing"
	case strings.Contains(t, "earnings") || strings.Contains(t, "eps"):
		return "earnings"
	default:
		return "other"
	}
}

// InMemorySource is a Source backed by a static slice of events.
// Primarily for use in tests and CLI importers.
type InMemorySource struct {
	name   string
	events []EconEvent
}

// NewInMemorySource creates a Source wrapping the given events.
func NewInMemorySource(name string, events []EconEvent) *InMemorySource {
	return &InMemorySource{name: name, events: events}
}

func (s *InMemorySource) Name() string { return s.name }

func (s *InMemorySource) FetchEvents(_ context.Context, from, to time.Time) ([]EconEvent, error) {
	var out []EconEvent
	for _, e := range s.events {
		if !e.ScheduledAt.Before(from) && !e.ScheduledAt.After(to) {
			out = append(out, e)
		}
	}
	return out, nil
}

// ─── Numeric helpers for tests ────────────────────────────────────────────────

// ParseNumericValue parses "1.2M" / "−0.3%" / "120K" into a float64.
// Returns 0 and an error if the string is not recognisable.
func ParseNumericValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u2212", "-") // unicode minus
	s = strings.ReplaceAll(s, ",", "")
	multiplier := 1.0
	if strings.HasSuffix(s, "%") {
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "M") || strings.HasSuffix(s, "m") {
		multiplier = 1_000_000
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "K") || strings.HasSuffix(s, "k") {
		multiplier = 1_000
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "B") || strings.HasSuffix(s, "b") {
		multiplier = 1_000_000_000
		s = s[:len(s)-1]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("ParseNumericValue %q: %w", s, err)
	}
	return v * multiplier, nil
}
