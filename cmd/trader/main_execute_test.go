package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jax-trading-assistant/libs/runtimepolicy"
)

func TestHandleExecuteBlocksDirectPaperExecution(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(`{"signal_id":"123","approved_by":"tester"}`))
	rec := httptest.NewRecorder()

	handleExecute(nil, runtimepolicy.ModePaper).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "approve a candidate") {
		t.Fatalf("body = %q, want candidate approval message", rec.Body.String())
	}
}

func TestHandleExecuteInternalPaperRequestStillValidatesBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", strings.NewReader(`{"approved_by":"tester"}`))
	req.Header.Set("X-JAX-INTERNAL-EXECUTE", "true")
	rec := httptest.NewRecorder()

	handleExecute(nil, runtimepolicy.ModePaper).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "signal_id is required") {
		t.Fatalf("body = %q, want signal_id validation message", rec.Body.String())
	}
}
