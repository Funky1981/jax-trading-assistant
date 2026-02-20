package calendar

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

func tmpDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	return d
}

// buildCSV writes a ForexFactory-style CSV to a temp file and returns its path.
func buildCSV(t *testing.T, header []string, rows [][]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ff.csv")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	_ = w.Write(header)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
	f.Close()
	return p
}

// ─── EventID ──────────────────────────────────────────────────────────────────

func TestEventID_Deterministic(t *testing.T) {
	ts := mustTime("2024-06-14T14:00:00Z")
	id1 := EventID("US", "Non-Farm Payrolls", ts)
	id2 := EventID("US", "Non-Farm Payrolls", ts)
	if id1 != id2 {
		t.Fatalf("EventID not deterministic: %q vs %q", id1, id2)
	}
}

func TestEventID_Unique(t *testing.T) {
	ts := mustTime("2024-06-14T14:00:00Z")
	a := EventID("US", "NFP", ts)
	b := EventID("US", "CPI", ts)
	c := EventID("EU", "NFP", ts)
	if a == b || a == c || b == c {
		t.Fatalf("EventID collision: a=%s b=%s c=%s", a, b, c)
	}
}

// ─── Store ────────────────────────────────────────────────────────────────────

func TestStore_UpsertAndQuery(t *testing.T) {
	store, err := OpenStore(tmpDir(t))
	if err != nil {
		t.Fatal(err)
	}

	events := []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP", Impact: ImpactHigh,
			ScheduledAt: mustTime("2024-01-05T13:30:00Z"), Source: "test"},
		{Country: "US", Currency: "USD", Title: "CPI", Impact: ImpactMedium,
			ScheduledAt: mustTime("2024-01-11T13:30:00Z"), Source: "test"},
		{Country: "EU", Currency: "EUR", Title: "ECB Rate", Impact: ImpactHigh,
			ScheduledAt: mustTime("2024-01-25T12:00:00Z"), Source: "test"},
	}
	if err := store.Upsert(events); err != nil {
		t.Fatal(err)
	}
	if store.Count() != 3 {
		t.Fatalf("want 3 events, got %d", store.Count())
	}

	// all events in January
	got := store.Query(
		mustTime("2024-01-01T00:00:00Z"),
		mustTime("2024-01-31T23:59:59Z"),
		"", "", "")
	if len(got) != 3 {
		t.Fatalf("query all: want 3, got %d", len(got))
	}

	// only USD events
	got = store.Query(
		mustTime("2024-01-01T00:00:00Z"),
		mustTime("2024-01-31T23:59:59Z"),
		"", "USD", "")
	if len(got) != 2 {
		t.Fatalf("query USD: want 2, got %d", len(got))
	}

	// only high-impact EU event
	got = store.Query(
		mustTime("2024-01-01T00:00:00Z"),
		mustTime("2024-01-31T23:59:59Z"),
		"EU", "", ImpactHigh)
	if len(got) != 1 || got[0].Title != "ECB Rate" {
		t.Fatalf("query EU high: want 1 ECB Rate, got %+v", got)
	}
}

func TestStore_Deduplication(t *testing.T) {
	store, err := OpenStore(tmpDir(t))
	if err != nil {
		t.Fatal(err)
	}

	ts := mustTime("2024-02-02T13:30:00Z")
	e := EconEvent{Country: "US", Currency: "USD", Title: "NFP",
		Impact: ImpactHigh, ScheduledAt: ts, Source: "test",
		Actual: "", Forecast: "180K"}
	if err := store.Upsert([]EconEvent{e}); err != nil {
		t.Fatal(err)
	}

	// Upsert again with Actual filled in.
	e2 := e
	e2.Actual = "185K"
	if err := store.Upsert([]EconEvent{e2}); err != nil {
		t.Fatal(err)
	}

	got := store.Query(ts.Add(-time.Hour), ts.Add(time.Hour), "", "", "")
	if len(got) != 1 {
		t.Fatalf("want 1 after dedup, got %d", len(got))
	}
	if got[0].Actual != "185K" {
		t.Fatalf("want Actual=185K, got %q", got[0].Actual)
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := tmpDir(t)
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := EconEvent{Country: "US", Currency: "USD", Title: "FOMC",
		Impact: ImpactHigh, ScheduledAt: mustTime("2024-03-20T18:00:00Z"),
		Source: "test"}
	if err := store.Upsert([]EconEvent{e}); err != nil {
		t.Fatal(err)
	}

	// Reload from disk.
	store2, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store2.Count() != 1 {
		t.Fatalf("after reload: want 1, got %d", store2.Count())
	}
}

func TestStore_QueryOrdering(t *testing.T) {
	store, err := OpenStore(tmpDir(t))
	if err != nil {
		t.Fatal(err)
	}
	events := []EconEvent{
		{Country: "US", Currency: "USD", Title: "C",
			ScheduledAt: mustTime("2024-04-03T16:00:00Z"), Source: "t", Impact: ImpactLow},
		{Country: "US", Currency: "USD", Title: "A",
			ScheduledAt: mustTime("2024-04-01T12:00:00Z"), Source: "t", Impact: ImpactLow},
		{Country: "US", Currency: "USD", Title: "B",
			ScheduledAt: mustTime("2024-04-02T14:00:00Z"), Source: "t", Impact: ImpactLow},
	}
	_ = store.Upsert(events)
	got := store.Query(
		mustTime("2024-04-01T00:00:00Z"),
		mustTime("2024-04-30T23:59:59Z"),
		"", "", "")
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	titles := []string{got[0].Title, got[1].Title, got[2].Title}
	want := []string{"A", "B", "C"}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("order wrong: got %v, want %v", titles, want)
		}
	}
}

// ─── CSVSource ────────────────────────────────────────────────────────────────

var ffHeader = []string{"Date", "Time", "Currency", "Impact", "Event", "Actual", "Forecast", "Previous"}

func TestCSVSource_FetchEvents(t *testing.T) {
	rows := [][]string{
		{"Jan 05, 2024", "8:30am", "USD", "High", "Non-Farm Payrolls", "216K", "170K", "199K"},
		{"", "8:30am", "USD", "Medium", "Unemployment Rate", "3.7%", "3.8%", "3.7%"},
		{"Jan 11, 2024", "8:30am", "USD", "High", "CPI m/m", "0.3%", "0.2%", "0.1%"},
	}
	p := buildCSV(t, ffHeader, rows)
	src := NewCSVSource("ff_csv", p)

	from := mustTime("2024-01-01T00:00:00Z")
	to := mustTime("2024-01-31T23:59:59Z")
	events, err := src.FetchEvents(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
	if events[0].Title != "Non-Farm Payrolls" {
		t.Fatalf("unexpected first event: %q", events[0].Title)
	}
	if events[1].Title != "Unemployment Rate" {
		t.Fatalf("blank date row should inherit previous date, got: %q", events[1].Title)
	}
	if events[2].Impact != ImpactHigh {
		t.Fatalf("CPI should be High impact, got %q", events[2].Impact)
	}
}

func TestCSVSource_DateFiltering(t *testing.T) {
	rows := [][]string{
		{"Jan 05, 2024", "8:30am", "USD", "High", "NFP", "216K", "170K", "199K"},
		{"Feb 02, 2024", "8:30am", "USD", "High", "NFP", "275K", "200K", "216K"},
	}
	p := buildCSV(t, ffHeader, rows)
	src := NewCSVSource("ff_csv", p)

	from := mustTime("2024-01-01T00:00:00Z")
	to := mustTime("2024-01-31T23:59:59Z")
	events, err := src.FetchEvents(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 (Jan only), got %d", len(events))
	}
}

func TestCSVSource_MissingRequiredColumnsError(t *testing.T) {
	// Omit "event" column
	rows := [][]string{
		{"Jan 05, 2024", "8:30am", "USD", "High"},
	}
	p := buildCSV(t, []string{"date", "time", "currency", "impact"}, rows)
	src := NewCSVSource("ff_csv", p)

	from := mustTime("2024-01-01T00:00:00Z")
	to := mustTime("2024-01-31T23:59:59Z")
	_, err := src.FetchEvents(context.Background(), from, to)
	if err == nil {
		t.Fatal("expected error for missing 'event' column")
	}
}

// ─── InMemorySource ───────────────────────────────────────────────────────────

func TestInMemorySource_FetchEvents(t *testing.T) {
	events := []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP",
			ScheduledAt: mustTime("2024-01-05T13:30:00Z"), Impact: ImpactHigh, Source: "test"},
		{Country: "EU", Currency: "EUR", Title: "ECB",
			ScheduledAt: mustTime("2024-02-01T12:00:00Z"), Impact: ImpactHigh, Source: "test"},
	}
	src := NewInMemorySource("mem", events)

	got, err := src.FetchEvents(context.Background(),
		mustTime("2024-01-01T00:00:00Z"),
		mustTime("2024-01-31T23:59:59Z"))
	if err != nil || len(got) != 1 {
		t.Fatalf("want 1 event in Jan, got %d err=%v", len(got), err)
	}
}

// ─── Feed ─────────────────────────────────────────────────────────────────────

func TestFeed_Ingest_MergesActual(t *testing.T) {
	store, _ := OpenStore(tmpDir(t))

	ts := mustTime("2024-01-05T13:30:00Z")
	// Source A has no actual, source B does.
	srcA := NewInMemorySource("A", []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP",
			ScheduledAt: ts, Impact: ImpactHigh, Forecast: "170K", Source: "A"},
	})
	srcB := NewInMemorySource("B", []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP",
			ScheduledAt: ts, Impact: ImpactHigh, Actual: "216K", Source: "B"},
	})
	feed := NewFeed(store, srcA, srcB)

	from := mustTime("2024-01-01T00:00:00Z")
	to := mustTime("2024-01-31T23:59:59Z")
	n, err := feed.Ingest(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 merged event, got %d", n)
	}

	got := store.Query(from, to, "", "", "")
	if len(got) != 1 {
		t.Fatalf("want 1 in store, got %d", len(got))
	}
	if got[0].Actual != "216K" {
		t.Fatalf("want Actual=216K, got %q", got[0].Actual)
	}
	if got[0].Forecast != "170K" {
		t.Fatalf("want Forecast=170K (from A), got %q", got[0].Forecast)
	}
}

func TestFeed_Ingest_SourceFailureContinues(t *testing.T) {
	store, _ := OpenStore(tmpDir(t))

	// bad source returns error
	badSrc := NewCSVSource("bad", "/nonexistent/file.csv")
	ts := mustTime("2024-01-05T13:30:00Z")
	goodSrc := NewInMemorySource("good", []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP",
			ScheduledAt: ts, Impact: ImpactHigh, Source: "good"},
	})

	feed := NewFeed(store, badSrc, goodSrc)
	n, err := feed.Ingest(context.Background(),
		mustTime("2024-01-01T00:00:00Z"),
		mustTime("2024-01-31T23:59:59Z"))
	if err != nil {
		t.Fatalf("expected partial success, got error: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 from good source, got %d", n)
	}
}

func TestFeed_HighImpactWindow(t *testing.T) {
	store, _ := OpenStore(tmpDir(t))

	center := mustTime("2024-01-05T13:30:00Z")
	events := []EconEvent{
		{Country: "US", Currency: "USD", Title: "NFP",
			ScheduledAt: center, Impact: ImpactHigh, Source: "t"},
		{Country: "US", Currency: "USD", Title: "Prelim",
			ScheduledAt: center.Add(15 * time.Minute), Impact: ImpactLow, Source: "t"},
		{Country: "US", Currency: "USD", Title: "Bonds",
			ScheduledAt: center.Add(6 * time.Hour), Impact: ImpactHigh, Source: "t"},
	}
	_ = store.Upsert(events)

	feed := NewFeed(store)
	got := feed.HighImpactWindow(store, center, 30*time.Minute)
	if len(got) != 1 || got[0].Title != "NFP" {
		t.Fatalf("HighImpactWindow: want [NFP], got %+v", got)
	}
}

// ─── Categorise ───────────────────────────────────────────────────────────────

func TestCategorise(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Non-Farm Payrolls", "employment"},
		{"Unemployment Rate", "employment"},
		{"CPI m/m", "inflation"},
		{"Core PCE Price Index", "inflation"},
		{"FOMC Statement", "central_bank"},
		{"ECB Rate Decision", "central_bank"},
		{"GDP q/q", "growth"},
		{"Retail Sales m/m", "consumer"},
		{"ISM Manufacturing PMI", "manufacturing"},
		{"Apple Earnings", "earnings"},
		{"Bond Auction", "other"},
	}
	for _, c := range cases {
		if got := categorise(c.title); got != c.want {
			t.Errorf("categorise(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

// ─── ParseNumericValue ────────────────────────────────────────────────────────

func TestParseNumericValue(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"216K", 216_000, true},
		{"1.2M", 1_200_000, true},
		{"0.3%", 0.3, true},
		{"\u22120.3%", -0.3, true}, // unicode minus sign
		{"3.7%", 3.7, true},
		{"2.5B", 2_500_000_000, true},
		{"abc", 0, false},
	}
	for _, c := range cases {
		v, err := ParseNumericValue(c.in)
		if c.ok && err != nil {
			t.Errorf("ParseNumericValue(%q): unexpected error: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ParseNumericValue(%q): expected error", c.in)
		}
		if c.ok && v != c.want {
			t.Errorf("ParseNumericValue(%q) = %v, want %v", c.in, v, c.want)
		}
	}
}
