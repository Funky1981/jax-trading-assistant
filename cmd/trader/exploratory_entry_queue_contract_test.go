package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicExploratoryEntryQueueIsUnavailableUntilCanonicalHandoff(t *testing.T) {
	t.Setenv("PAPER_ACCOUNT_ID", "disposable-account")
	mux := http.NewServeMux()
	registerExploratoryPaperRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, nil)
	for _, body := range []string{
		`{"candidateId":"candidate-test"}`,
		`{"candidateId":"candidate-test","quantity":100,"paperIntent":{}}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exploratory-paper/entry-queue", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("route status=%d body=%s; candidate entry queue must remain unregistered before 02A4", rec.Code, rec.Body.String())
		}
	}
}
