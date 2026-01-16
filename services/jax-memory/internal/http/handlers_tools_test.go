package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

type fakeStore struct {
	retained int
}

func (s *fakeStore) Ping(context.Context) error { return nil }

func (s *fakeStore) Retain(_ context.Context, _ string, _ contracts.MemoryItem) (contracts.MemoryID, error) {
	s.retained++
	return "id1", nil
}

func (s *fakeStore) Recall(context.Context, string, contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	return nil, nil
}

func (s *fakeStore) Reflect(context.Context, string, contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	return nil, nil
}

func TestTools_MemoryRetain(t *testing.T) {
	store := &fakeStore{}
	srv := NewServer(store)
	srv.RegisterTools()

	body := map[string]any{
		"tool": "memory.retain",
		"input": map[string]any{
			"bank": "trade_decisions",
			"item": map[string]any{
				"ts":      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				"type":    "decision",
				"summary": "test",
			},
		},
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.retained != 1 {
		t.Fatalf("expected retain called")
	}
}
