package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
	jaxtest "jax-trading-assistant/libs/testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestStore() contracts.MemoryStore {
	return jaxtest.NewInMemoryMemoryStore()
}

func validTestItem() contracts.MemoryItem {
	return contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "signal",
		Symbol:  "AAPL",
		Summary: "MACD crossover detected on AAPL daily chart",
		Tags:    []string{"signal", "aapl"},
		Data:    map[string]any{"confidence": 0.85},
		Source:  &contracts.MemorySource{System: "jax-test"},
	}
}

func postTool(t *testing.T, handler http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	return rw
}

// ── memoryToolHandler tests ───────────────────────────────────────────────────

func TestMemoryToolHandler_Retain_Valid(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	retainInput := contracts.MemoryRetainRequest{
		Bank: "default",
		Item: validTestItem(),
	}
	inputJSON, _ := json.Marshal(retainInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.retain",
		Input: inputJSON,
	})

	if rw.Code != http.StatusOK {
		t.Errorf("retain status = %d; want 200; body: %s", rw.Code, rw.Body.String())
	}

	var resp toolResponse
	if err := json.NewDecoder(rw.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestMemoryToolHandler_Retain_MissingBank(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	retainInput := contracts.MemoryRetainRequest{
		Bank: "", // intentionally empty
		Item: validTestItem(),
	}
	inputJSON, _ := json.Marshal(retainInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.retain",
		Input: inputJSON,
	})

	if rw.Code != http.StatusBadRequest {
		t.Errorf("missing bank: status = %d; want 400", rw.Code)
	}
}

func TestMemoryToolHandler_Retain_InvalidItem(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	bad := validTestItem()
	bad.Summary = "" // fails validation
	retainInput := contracts.MemoryRetainRequest{Bank: "default", Item: bad}
	inputJSON, _ := json.Marshal(retainInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.retain",
		Input: inputJSON,
	})

	if rw.Code != http.StatusBadRequest {
		t.Errorf("invalid item: status = %d; want 400", rw.Code)
	}
}

func TestMemoryToolHandler_Recall_Success(t *testing.T) {
	store := newTestStore()
	// Pre-populate a memory so recall has something to return.
	item := validTestItem()
	store.Retain(nil, "default", item) //nolint:errcheck

	handler := memoryToolHandler(store)

	recallInput := contracts.MemoryRecallRequest{
		Bank:  "default",
		Query: contracts.MemoryQuery{Q: "AAPL signal"},
	}
	inputJSON, _ := json.Marshal(recallInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.recall",
		Input: inputJSON,
	})

	if rw.Code != http.StatusOK {
		t.Errorf("recall status = %d; want 200; body: %s", rw.Code, rw.Body.String())
	}
}

func TestMemoryToolHandler_Recall_MissingBank(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	recallInput := contracts.MemoryRecallRequest{
		Bank:  "",
		Query: contracts.MemoryQuery{Q: "test"},
	}
	inputJSON, _ := json.Marshal(recallInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.recall",
		Input: inputJSON,
	})

	if rw.Code != http.StatusBadRequest {
		t.Errorf("missing bank: status = %d; want 400", rw.Code)
	}
}

func TestMemoryToolHandler_Reflect_Success(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	reflectInput := contracts.MemoryReflectRequest{
		Bank:   "default",
		Params: contracts.ReflectionParams{Query: "What patterns does AAPL show?"},
	}
	inputJSON, _ := json.Marshal(reflectInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.reflect",
		Input: inputJSON,
	})

	// In-memory store returns an empty slice for reflect; 200 is still expected.
	if rw.Code != http.StatusOK {
		t.Errorf("reflect status = %d; want 200; body: %s", rw.Code, rw.Body.String())
	}
}

func TestMemoryToolHandler_Reflect_MissingBank(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	reflectInput := contracts.MemoryReflectRequest{
		Bank:   "",
		Params: contracts.ReflectionParams{Query: "test"},
	}
	inputJSON, _ := json.Marshal(reflectInput)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.reflect",
		Input: inputJSON,
	})

	if rw.Code != http.StatusBadRequest {
		t.Errorf("missing bank: status = %d; want 400", rw.Code)
	}
}

func TestMemoryToolHandler_UnknownTool(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	rw := postTool(t, handler, toolRequest{
		Tool:  "memory.unknown",
		Input: json.RawMessage(`{}`),
	})

	if rw.Code != http.StatusBadRequest {
		t.Errorf("unknown tool: status = %d; want 400", rw.Code)
	}
}

func TestMemoryToolHandler_MethodNotAllowed(t *testing.T) {
	store := newTestStore()
	handler := memoryToolHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/tools", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /tools status = %d; want 405", rw.Code)
	}
}

// ── memoryBanksHandler tests ──────────────────────────────────────────────────

func TestMemoryBanksHandler_Returns200WithBanks(t *testing.T) {
	handler := memoryBanksHandler()

	req := httptest.NewRequest(http.MethodGet, "/v1/memory/banks", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("banks status = %d; want 200", rw.Code)
	}

	var banks []string
	if err := json.NewDecoder(rw.Body).Decode(&banks); err != nil {
		t.Fatalf("decode banks: %v", err)
	}
	if len(banks) == 0 {
		t.Error("expected at least one bank in response")
	}

	found := false
	for _, b := range banks {
		if b == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'default' bank in list, got %v", banks)
	}
}

// ── memorySearchHandler tests ─────────────────────────────────────────────────

func TestMemorySearchHandler_WithQuery(t *testing.T) {
	store := newTestStore()
	item := validTestItem()
	store.Retain(nil, "default", item) //nolint:errcheck

	handler := memorySearchHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/memory/search?q=AAPL&bank=default", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("search status = %d; want 200", rw.Code)
	}
}

func TestMemorySearchHandler_EmptyQuery_ReturnsEmptyArray(t *testing.T) {
	store := newTestStore()
	handler := memorySearchHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/memory/search", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("empty q status = %d; want 200", rw.Code)
	}

	body := strings.TrimSpace(rw.Body.String())
	if body == "" || body == "null" {
		t.Error("expected JSON response for empty query, got empty/null")
	}
}

func TestMemorySearchHandler_DefaultBankUsed(t *testing.T) {
	store := newTestStore()
	item := validTestItem()
	store.Retain(nil, "default", item) //nolint:errcheck

	handler := memorySearchHandler(store)

	// No bank param — should default to "default".
	req := httptest.NewRequest(http.MethodGet, "/v1/memory/search?q=AAPL", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("default bank search status = %d; want 200", rw.Code)
	}
}
