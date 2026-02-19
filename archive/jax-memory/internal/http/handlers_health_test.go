package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	jaxTesting "jax-trading-assistant/libs/testing"
)

func TestHealthHandler_OK(t *testing.T) {
	store := jaxTesting.NewInMemoryMemoryStore()
	srv := NewServer(store)
	srv.RegisterHealth()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	okValue, ok := body["ok"].(bool)
	if !ok || !okValue {
		t.Fatalf("expected ok=true, got %#v", body["ok"])
	}
}
