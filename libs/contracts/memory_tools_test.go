package contracts

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestMemoryToolJSON_Golden(t *testing.T) {
	item := MemoryItem{
		TS:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:    "decision",
		Symbol:  "AAPL",
		Tags:    []string{"earnings"},
		Summary: "Entered on earnings gap.",
		Data:    map[string]any{"confidence": 0.72},
		Source:  &MemorySource{System: "dexter"},
	}

	req := MemoryRetainRequest{Bank: "trade_decisions", Item: item}
	retOut := MemoryRetainResponse{ID: "mem_123"}
	recOut := MemoryRecallResponse{Items: []MemoryItem{item}}
	refOut := MemoryReflectResponse{Items: []MemoryItem{item}}

	assertJSONEqual(t, req, `{"bank":"trade_decisions","item":{"ts":"2025-01-01T00:00:00Z","type":"decision","symbol":"AAPL","tags":["earnings"],"summary":"Entered on earnings gap.","data":{"confidence":0.72},"source":{"system":"dexter"}}}`)
	assertJSONEqual(t, retOut, `{"id":"mem_123"}`)
	assertJSONEqual(t, recOut, `{"items":[{"ts":"2025-01-01T00:00:00Z","type":"decision","symbol":"AAPL","tags":["earnings"],"summary":"Entered on earnings gap.","data":{"confidence":0.72},"source":{"system":"dexter"}}]}`)
	assertJSONEqual(t, refOut, `{"items":[{"ts":"2025-01-01T00:00:00Z","type":"decision","symbol":"AAPL","tags":["earnings"],"summary":"Entered on earnings gap.","data":{"confidence":0.72},"source":{"system":"dexter"}}]}`)
}

func assertJSONEqual(t *testing.T, got any, expected string) {
	t.Helper()

	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}

	var gotObj any
	var expectedObj any
	if err := json.Unmarshal(gotRaw, &gotObj); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}

	if !reflect.DeepEqual(gotObj, expectedObj) {
		t.Fatalf("json mismatch\nexpected: %s\nactual:   %s", expected, string(gotRaw))
	}
}
