package swingresearch

import "testing"

func TestEngineBlocksMissingEventSource(t *testing.T) {
	engine := NewEngine()
	out := engine.Evaluate(Input{
		Symbol:       "QQQ",
		DailyCandles: 20,
	})

	if out.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", out.Status)
	}
	if !contains(out.BlockerReasons, "missing_event_source") {
		t.Fatalf("blockers = %#v, want missing_event_source", out.BlockerReasons)
	}
}

func TestEngineBlocksMissingDailyCandles(t *testing.T) {
	engine := NewEngine()
	out := engine.Evaluate(Input{
		Symbol:      "QQQ",
		EventSource: EvidenceSource{ID: "event-1", URL: "https://example.com/event"},
	})

	if out.Status != StatusBlocked {
		t.Fatalf("status = %q, want blocked", out.Status)
	}
	if !contains(out.BlockerReasons, "missing_daily_candles") {
		t.Fatalf("blockers = %#v, want missing_daily_candles", out.BlockerReasons)
	}
}

func TestEngineDowngradesConfounderToWatch(t *testing.T) {
	engine := NewEngine()
	out := engine.Evaluate(Input{
		Symbol:       "QQQ",
		EventSource:  EvidenceSource{ID: "event-1", URL: "https://example.com/event"},
		DailyCandles: 20,
		Confounders:  []string{"FOMC decision due tomorrow"},
	})

	if out.Status != StatusWatch {
		t.Fatalf("status = %q, want watch", out.Status)
	}
	if !contains(out.BlockerReasons, "confounder_present") {
		t.Fatalf("blockers = %#v, want confounder_present", out.BlockerReasons)
	}
}

func TestEngineReturnsSwingThesisNotOrder(t *testing.T) {
	engine := NewEngine()
	out := engine.Evaluate(Input{
		Symbol:                   "QQQ",
		EventSource:              EvidenceSource{ID: "event-1", URL: "https://example.com/event"},
		DailyCandles:             30,
		Headline:                 "Semiconductor demand lifts growth ETFs",
		MappedETFs:               []string{"QQQ", "XLK"},
		HistoricalReactionWindow: "2-5 trading days",
	})

	if out.Status != StatusThesis {
		t.Fatalf("status = %q, want thesis", out.Status)
	}
	if out.OrderInstruction != "" {
		t.Fatalf("order instruction = %q, want empty", out.OrderInstruction)
	}
	if out.ThesisSummary == "" || len(out.MappedETFs) == 0 || len(out.Invalidators) == 0 {
		t.Fatalf("incomplete thesis output: %#v", out)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
