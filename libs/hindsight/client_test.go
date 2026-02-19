package hindsight_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/hindsight"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestClient(t *testing.T, srv *httptest.Server) *hindsight.Client {
	t.Helper()
	c, err := hindsight.New(srv.URL)
	if err != nil {
		t.Fatalf("hindsight.New: %v", err)
	}
	return c
}

func validItem() contracts.MemoryItem {
	return contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "signal",
		Symbol:  "AAPL",
		Summary: "MACD crossover detected for AAPL",
		Tags:    []string{"signal", "aapl"},
		Data:    map[string]any{"confidence": 0.9},
		Source:  &contracts.MemorySource{System: "jax-test"},
	}
}

// ── constructor tests ─────────────────────────────────────────────────────────

func TestNew_InvalidURL(t *testing.T) {
	_, err := hindsight.New("not-a-url")
	if err == nil {
		t.Error("expected error for URL without scheme, got nil")
	}
}

func TestNew_EmptyURL(t *testing.T) {
	_, err := hindsight.New("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}

func TestNew_ValidURL(t *testing.T) {
	c, err := hindsight.New("http://localhost:8888")
	if err != nil {
		t.Errorf("expected no error for valid URL, got: %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client")
	}
}

// ── Ping ─────────────────────────────────────────────────────────────────────

func TestPing_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Ping: unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping returned error: %v", err)
	}
}

func TestPing_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Ping(context.Background())
	if err == nil {
		t.Error("expected error for 503 response, got nil")
	}
}

// ── Retain ────────────────────────────────────────────────────────────────────

func TestRetain_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Retain: expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/memories") {
			t.Errorf("Retain: unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success":     true,
			"bank_id":     "default",
			"items_count": 1,
			"async":       false,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	id, err := c.Retain(context.Background(), "default", validItem())
	if err != nil {
		t.Fatalf("Retain returned error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty memory ID")
	}
}

func TestRetain_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Retain(context.Background(), "default", validItem())
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

// ── Recall ────────────────────────────────────────────────────────────────────

func TestRecall_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/recall") {
			t.Errorf("Recall: unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":   "m1",
					"text": "MACD crossover for AAPL",
					"type": nil,
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	items, err := c.Recall(context.Background(), "default", contracts.MemoryQuery{Q: "AAPL signal"})
	if err != nil {
		t.Fatalf("Recall returned error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least one recalled item")
	}
}

func TestRecall_EmptyQuery(t *testing.T) {
	c, _ := hindsight.New("http://localhost:9999") // unreachable — query check is local
	_, err := c.Recall(context.Background(), "default", contracts.MemoryQuery{})
	if err == nil {
		t.Error("expected error for empty query, got nil")
	}
}

func TestRecall_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	items, err := c.Recall(context.Background(), "default", contracts.MemoryQuery{Q: "nonexistent"})
	if err != nil {
		t.Errorf("Recall with empty results returned error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// ── Reflect ───────────────────────────────────────────────────────────────────

func TestReflect_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/reflect") {
			t.Errorf("Reflect: unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"text":     "AAPL has shown consistent MACD crossovers during earnings season.",
			"based_on": nil,
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	items, err := c.Reflect(context.Background(), "default", contracts.ReflectionParams{
		Query: "What patterns does AAPL show?",
	})
	if err != nil {
		t.Fatalf("Reflect returned error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least one reflection item")
	}
	if items[0].Summary == "" {
		t.Error("expected non-empty reflection summary")
	}
}

func TestReflect_EmptyQuery(t *testing.T) {
	c, _ := hindsight.New("http://localhost:9999")
	_, err := c.Reflect(context.Background(), "default", contracts.ReflectionParams{Query: ""})
	if err == nil {
		t.Error("expected error for empty reflect query, got nil")
	}
}

func TestReflect_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model error", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Reflect(context.Background(), "default", contracts.ReflectionParams{Query: "test"})
	if err == nil {
		t.Error("expected error for 502 response, got nil")
	}
}

// ── timeout ───────────────────────────────────────────────────────────────────

func TestPing_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow handler — context will already be cancelled.
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	c := newTestClient(t, srv)
	err := c.Ping(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}
