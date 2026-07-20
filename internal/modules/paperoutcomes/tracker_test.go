package paperoutcomes

import (
	"testing"
	"time"
)

func baseTicket(start time.Time) ticket {
	return ticket{ID: "pt_test", Symbol: "QQQ", Direction: "long", Status: "paper_ticket_created", CreatedAt: start, Entry: 500, Stop: 490, Target: 520, Size: 10, PaperOnly: true}
}

func TestCheckpointDoesNotCompleteBeforeDue(t *testing.T) {
	start := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	cp := calculateCheckpoint(baseTicket(start), checkpointDefinition{"1w", 7 * 24 * time.Hour}, start, "paper_ticket_created_at", start.Add(24*time.Hour), nil, "", "", "")
	if cp.CheckpointStatus != "pending_not_due" {
		t.Fatalf("status=%s", cp.CheckpointStatus)
	}
}
func TestCheckpointRequiresPersistedObservation(t *testing.T) {
	start := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	cp := calculateCheckpoint(baseTicket(start), checkpointDefinition{"1h", time.Hour}, start, "paper_ticket_created_at", start.Add(2*time.Hour), nil, "", "", "")
	if cp.CheckpointStatus != "pending_market_data" {
		t.Fatalf("status=%s", cp.CheckpointStatus)
	}
}
func TestCheckpointPreservesIntracandleAmbiguity(t *testing.T) {
	start := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	at := start.Add(time.Hour)
	cp := calculateCheckpoint(baseTicket(start), checkpointDefinition{"1h", time.Hour}, start, "paper_ticket_created_at", at, []observation{{At: at, High: 521, Low: 489, Close: 505}}, "persisted_candles", "unknown", "1h (inferred)")
	if cp.CheckpointStatus != "ambiguous_same_candle" {
		t.Fatalf("status=%s", cp.CheckpointStatus)
	}
	if !cp.TargetTouched || !cp.StopTouched {
		t.Fatal("both touches must be retained")
	}
}
func TestCheckpointCalculatesLongHypotheticalPnL(t *testing.T) {
	start := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	at := start.Add(time.Hour)
	cp := calculateCheckpoint(baseTicket(start), checkpointDefinition{"1h", time.Hour}, start, "paper_ticket_created_at", at, []observation{{At: at, High: 511, Low: 499, Close: 510}}, "persisted_candles", "unknown", "1h (inferred)")
	if cp.HypotheticalPnL == nil || *cp.HypotheticalPnL != 100 {
		t.Fatalf("pnl=%v", cp.HypotheticalPnL)
	}
	if cp.CheckpointStatus != "completed" {
		t.Fatalf("status=%s", cp.CheckpointStatus)
	}
}
