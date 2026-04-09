package chat

import (
	"errors"
	"testing"
)

func TestLoadRuntimeConfigUsesHarnessFlags(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("HARNESS_ENABLED", "false")
	t.Setenv("HARNESS_SHADOW_MODE", "true")
	t.Setenv("HARNESS_SESSION_RATE_LIMIT_PER_MINUTE", "7")

	cfg := loadRuntimeConfig()
	if cfg.Mode != "paper" {
		t.Fatalf("expected paper mode, got %s", cfg.Mode)
	}
	if !cfg.HarnessEnabled {
		t.Fatal("expected shadow mode to force harness enabled")
	}
	if !cfg.ShadowMode {
		t.Fatal("expected shadow mode enabled")
	}
	if cfg.SessionRateLimitPerMinute != 7 {
		t.Fatalf("expected rate limit 7, got %d", cfg.SessionRateLimitPerMinute)
	}
}

func TestSessionRateLimiterRejectsAfterLimit(t *testing.T) {
	limiter := newSessionRateLimiter(1)
	if err := limiter.Allow("session-1"); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}
	err := limiter.Allow("session-1")
	if !errors.Is(err, ErrSessionRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}
