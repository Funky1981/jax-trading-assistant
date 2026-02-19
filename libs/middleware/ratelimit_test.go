package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.RequestsPerMinute <= 0 {
		t.Errorf("RequestsPerMinute = %d; want > 0", cfg.RequestsPerMinute)
	}
	if cfg.RequestsPerHour <= 0 {
		t.Errorf("RequestsPerHour = %d; want > 0", cfg.RequestsPerHour)
	}
	if !cfg.Enabled {
		t.Error("default config should be enabled")
	}
}

func TestRateLimiter_Allow_UnderLimit(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 100,
		RequestsPerHour:   1000,
		Enabled:           true,
	}
	rl := NewRateLimiter(cfg)

	allowed, msg := rl.Allow("127.0.0.1")
	if !allowed {
		t.Errorf("first request should be allowed, got: %s", msg)
	}
}

func TestRateLimiter_Allow_ExceedsMinuteLimit(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 2,
		RequestsPerHour:   1000,
		Enabled:           true,
	}
	rl := NewRateLimiter(cfg)

	const ip = "10.0.0.1"
	// Exhaust the per-minute budget.
	for i := 0; i < 2; i++ {
		rl.Allow(ip)
	}

	allowed, msg := rl.Allow(ip)
	if allowed {
		t.Error("request beyond per-minute limit should be denied")
	}
	if msg == "" {
		t.Error("expected non-empty denial reason")
	}
}

func TestRateLimiter_Allow_DisabledPassesAll(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 1,
		RequestsPerHour:   1,
		Enabled:           false,
	}
	rl := NewRateLimiter(cfg)

	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow("192.168.1.1")
		if !allowed {
			t.Errorf("disabled limiter should pass all; denied on request %d", i+1)
		}
	}
}

func TestRateLimiter_Allow_IsolatedPerIP(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 1,
		RequestsPerHour:   100,
		Enabled:           true,
	}
	rl := NewRateLimiter(cfg)

	rl.Allow("1.1.1.1") // exhaust ip A

	allowed, _ := rl.Allow("2.2.2.2") // ip B is fresh
	if !allowed {
		t.Error("separate IP should not be rate-limited by another IP's counter")
	}
}

func TestRateLimiter_Middleware_Returns429WhenExceeded(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 1,
		RequestsPerHour:   100,
		Enabled:           true,
	}
	rl := NewRateLimiter(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(inner)

	makeReq := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		return rw.Code
	}

	first := makeReq()
	if first != http.StatusOK {
		t.Errorf("first request status = %d; want 200", first)
	}

	second := makeReq()
	if second != http.StatusTooManyRequests {
		t.Errorf("second request status = %d; want 429", second)
	}
}

func TestRateLimiter_Middleware_SetsRateLimitHeaders(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerMinute: 1,
		RequestsPerHour:   100,
		Enabled:           true,
	}
	rl := NewRateLimiter(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := rl.Middleware(inner)

	// Exhaust limit.
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		if rw.Code == http.StatusTooManyRequests {
			if rw.Header().Get("X-RateLimit-Limit") == "" {
				t.Error("429 response missing X-RateLimit-Limit header")
			}
			if rw.Header().Get("X-RateLimit-Remaining") == "" {
				t.Error("429 response missing X-RateLimit-Remaining header")
			}
			return
		}
	}
	t.Error("expected a 429 before test exhausted the request loop")
}
