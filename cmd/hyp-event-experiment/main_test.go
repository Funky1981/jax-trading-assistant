package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionTimestampsAreUTCAndOrdered(t *testing.T) {
	open, err := sessionOpenUTC("2024-07-01")
	if err != nil {
		t.Fatal(err)
	}
	close, err := sessionCloseUTC("2024-07-05")
	if err != nil {
		t.Fatal(err)
	}
	if open.Location() != time.UTC || close.Location() != time.UTC || !close.After(open) {
		t.Fatalf("timestamps are not ordered UTC: %s %s", open, close)
	}
}

func TestPreOpenEventUsesSameSessionBarWithOpenTimestampAfterPublication(t *testing.T) {
	rows := []bar{{Open: 10, Close: 11, Time: "2017-02-21T05:00:00Z"}, {Open: 12, Close: 13, Time: "2017-02-22T05:00:00Z"}}
	cut, err := time.Parse(time.RFC3339, "2017-02-21T13:32:27Z")
	if err != nil {
		t.Fatal(err)
	}
	got := nextEntry(rows, cut)
	if got != 0 {
		t.Fatalf("entry index=%d want=0", got)
	}
	open, err := sessionOpenUTC("2017-02-21")
	if err != nil {
		t.Fatal(err)
	}
	if !open.After(cut) {
		t.Fatalf("session open %s is not after publication %s", open, cut)
	}
}

func TestFalsificationHelpersAreDeterministicAndDoNotMutateInput(t *testing.T) {
	rows := []observation{{EventID: "b", IssuerID: "2", EntryDate: "2024-01-02", CostAdjusted: .2}, {EventID: "a", IssuerID: "1", EntryDate: "2024-01-01", CostAdjusted: .1}}
	rotated := rotateReturns(rows)
	if rows[0].CostAdjusted != .2 || rows[1].CostAdjusted != .1 {
		t.Fatal("return rotation mutated input")
	}
	if rotated[0].EventID != "a" || rotated[0].CostAdjusted != .2 {
		t.Fatalf("unexpected deterministic rotation: %+v", rotated)
	}
	if got := len(onePerIssuerAndDate([]observation{{IssuerID: "1", EntryDate: "2024-01-01"}, {IssuerID: "1", EntryDate: "2024-01-01"}, {IssuerID: "1", EntryDate: "2024-01-02"}})); got != 2 {
		t.Fatalf("dedup count=%d want=2", got)
	}
}

func TestDecodePreHoldoutBarsDoesNotMaterializeSealedRows(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"bars": map[string]any{"XYZ": []map[string]any{
		{"t": "2024-12-31T05:00:00Z", "o": 10.0, "c": 11.0},
		{"t": "2025-01-02T05:00:00Z", "o": -999.0, "c": -999.0},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := decodePreHoldoutBars(raw, "XYZ")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Time != "2024-12-31T05:00:00Z" {
		t.Fatalf("sealed row was materialized or pre-holdout row lost: %+v", rows)
	}
}
