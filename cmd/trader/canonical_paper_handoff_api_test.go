package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/auth"

	"github.com/google/uuid"
)

func TestCanonicalPaperHandoffHTTPBoundaryRequiresValidatedJWTAndRejectsCallerEconomics(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("canonical-handoff-api-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerCanonicalPaperHandoffRoutes(mux, manager.MiddlewareFunc, nil)
	candidateID := uuid.NewString()
	path := "/api/v1/exploratory-paper/handoff/" + candidateID + "/prepare"

	for _, test := range []struct {
		name       string
		body       string
		token      string
		userHeader string
		want       int
	}{
		{name: "missing JWT even with spoof header", userHeader: "victim-user", want: http.StatusUnauthorized},
		{name: "caller-supplied economic state rejected", body: `{"quantity":10,"riskDecision":{},"paperIntent":{}}`, token: "valid", want: http.StatusBadRequest},
		{name: "validated JWT reaches server-owned service despite spoof header", token: "valid", userHeader: "victim-user", want: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(test.body))
			if test.userHeader != "" {
				request.Header.Set("X-User-ID", test.userHeader)
			}
			if test.token == "valid" {
				jwt, err := manager.GenerateToken("authenticated-user", "authenticated-user", "operator")
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+jwt)
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}

func TestCanonicalPaperHandoffHTTPBoundaryRejectsNonPaperMode(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "research")
	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("canonical-handoff-api-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.GenerateToken("authenticated-user", "authenticated-user", "operator")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerCanonicalPaperHandoffRoutes(mux, manager.MiddlewareFunc, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/exploratory-paper/handoff/"+uuid.NewString()+"/approve", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("non-PAPER status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
