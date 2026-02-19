package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func defaultTestConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
}

// okHandler is the downstream handler used in all CORS tests.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := CORS(defaultTestConfig())(okHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/signals", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d; want 204", rw.Code)
	}
}

func TestCORS_AllowedOriginSetsHeader(t *testing.T) {
	handler := CORS(defaultTestConfig())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	got := rw.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:5173" {
		t.Errorf("ACAO = %q; want %q", got, "http://localhost:5173")
	}
}

func TestCORS_BlockedOriginNoHeader(t *testing.T) {
	handler := CORS(defaultTestConfig())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	got := rw.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Errorf("blocked origin: ACAO = %q; want empty", got)
	}
}

func TestCORS_CredentialHeader(t *testing.T) {
	handler := CORS(defaultTestConfig())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	got := rw.Header().Get("Access-Control-Allow-Credentials")
	if got != "true" {
		t.Errorf("ACAC = %q; want \"true\"", got)
	}
}

func TestCORS_CredentialsDisabled(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.AllowCredentials = false
	handler := CORS(cfg)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	got := rw.Header().Get("Access-Control-Allow-Credentials")
	if got != "" {
		t.Errorf("ACAC = %q; want empty when AllowCredentials=false", got)
	}
}

func TestCORS_NoOriginHeader_PassThrough(t *testing.T) {
	handler := CORS(defaultTestConfig())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	// Without an Origin header the downstream still receives the request.
	if rw.Code != http.StatusOK {
		t.Errorf("status = %d; want 200", rw.Code)
	}
}

func TestCORS_WildcardOrigin(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.AllowedOrigins = []string{"*"}
	handler := CORS(cfg)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://anything.example.com")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	got := rw.Header().Get("Access-Control-Allow-Origin")
	if got == "" {
		t.Error("wildcard origin: ACAO header should be set")
	}
}

func TestDefaultCORSConfig_HasExpectedOrigins(t *testing.T) {
	cfg := DefaultCORSConfig()
	if len(cfg.AllowedOrigins) == 0 {
		t.Error("DefaultCORSConfig.AllowedOrigins should not be empty")
	}
	if !cfg.AllowCredentials {
		t.Error("DefaultCORSConfig.AllowCredentials should be true")
	}
	if cfg.MaxAge <= 0 {
		t.Errorf("DefaultCORSConfig.MaxAge = %d; want > 0", cfg.MaxAge)
	}
}
