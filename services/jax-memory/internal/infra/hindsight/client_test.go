package hindsight

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func TestClient_Ping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestClient_Retain_SendsExpectedShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/default/banks/trade_decisions/memories" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body retainRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(body.Items) != 1 || body.Items[0].Content == "" {
			t.Fatalf("unexpected body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(retainResponse{Success: true, BankID: "trade_decisions", ItemsCount: 1, Async: false})
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	id, err := c.Retain(context.Background(), "trade_decisions", contracts.MemoryItem{
		ID:      "doc1",
		TS:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:    "decision",
		Symbol:  "AAPL",
		Tags:    []string{"earnings"},
		Summary: "AAPL earnings gap long.",
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	if id != "doc1" {
		t.Fatalf("expected doc1, got %s", id)
	}
}

func TestClient_Recall_MapsResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/default/banks/trade_decisions/memories/recall" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req recallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Query == "" {
			t.Fatalf("expected query")
		}
		typ := "experience"
		entities := []string{"AAPL"}
		_ = json.NewEncoder(w).Encode(recallResponse{
			Results: []recallResult{
				{ID: "m1", Text: "Test memory", Type: &typ, Entities: &entities},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	out, err := c.Recall(context.Background(), "trade_decisions", contracts.MemoryQuery{Q: "earnings"})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(out) != 1 || out[0].ID != "m1" || out[0].Summary != "Test memory" {
		t.Fatalf("unexpected: %#v", out)
	}
}

func TestClient_Recall_RequiresQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	_, err = c.Recall(context.Background(), "trade_decisions", contracts.MemoryQuery{})
	if err == nil {
		t.Fatalf("expected error for empty query")
	}
}

func TestClient_Reflect_MapsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/default/banks/trade_decisions/reflect" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(reflectResponse{
			Text:    "Summary of prior trades",
			BasedOn: json.RawMessage(`{"items":2}`),
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	items, err := c.Reflect(context.Background(), "trade_decisions", contracts.ReflectionParams{Query: "earnings"})
	if err != nil {
		t.Fatalf("reflect: %v", err)
	}
	if len(items) != 1 || items[0].Summary == "" {
		t.Fatalf("unexpected: %#v", items)
	}
}

func TestClient_Retain_HandlesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	_, err = c.Retain(context.Background(), "trade_decisions", contracts.MemoryItem{Summary: "x"})
	if err == nil {
		t.Fatalf("expected error for http 400")
	}
}
