package chat

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/libs/runtimepolicy"
)

var ErrSessionRateLimited = fmt.Errorf("chat session rate limit exceeded")

type runtimeConfig struct {
	Mode                      harness.Mode
	HarnessEnabled            bool
	ShadowMode                bool
	SessionRateLimitPerMinute int
}

type RuntimeInfo struct {
	Mode                      string `json:"mode"`
	HarnessEnabled            bool   `json:"harnessEnabled"`
	ShadowMode                bool   `json:"shadowMode"`
	SessionRateLimitPerMinute int    `json:"sessionRateLimitPerMinute"`
}

func loadRuntimeConfig() runtimeConfig {
	mode, _, err := runtimepolicy.ResolveModeFromEnv()
	if err != nil {
		mode = runtimepolicy.ModeDev
	}
	cfg := runtimeConfig{
		Mode:                      toHarnessMode(mode),
		HarnessEnabled:            envBool("HARNESS_ENABLED", true),
		ShadowMode:                envBool("HARNESS_SHADOW_MODE", false),
		SessionRateLimitPerMinute: envInt("HARNESS_SESSION_RATE_LIMIT_PER_MINUTE", 20),
	}
	if cfg.ShadowMode {
		cfg.HarnessEnabled = true
	}
	return cfg
}

func toHarnessMode(mode runtimepolicy.Mode) harness.Mode {
	switch mode {
	case runtimepolicy.ModePaper:
		return harness.ModePaper
	case runtimepolicy.ModeLive:
		return harness.ModeLive
	default:
		return harness.ModeResearch
	}
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (c runtimeConfig) info() RuntimeInfo {
	return RuntimeInfo{
		Mode:                      string(c.Mode),
		HarnessEnabled:            c.HarnessEnabled,
		ShadowMode:                c.ShadowMode,
		SessionRateLimitPerMinute: c.SessionRateLimitPerMinute,
	}
}

type sessionRateLimiter struct {
	limit   int
	mu      sync.Mutex
	buckets map[string]*sessionBucket
}

type sessionBucket struct {
	count     int
	resetTime time.Time
}

func newSessionRateLimiter(limit int) *sessionRateLimiter {
	if limit <= 0 {
		return nil
	}
	return &sessionRateLimiter{
		limit:   limit,
		buckets: make(map[string]*sessionBucket),
	}
}

func (l *sessionRateLimiter) Allow(sessionID string) error {
	if l == nil || sessionID == "" {
		return nil
	}

	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[sessionID]
	if !ok || now.After(bucket.resetTime) {
		l.buckets[sessionID] = &sessionBucket{
			count:     1,
			resetTime: now.Add(time.Minute),
		}
		return nil
	}
	if bucket.count >= l.limit {
		retryAfter := time.Until(bucket.resetTime).Round(time.Second)
		return fmt.Errorf("%w: max %d messages per minute for this session, retry after %s", ErrSessionRateLimited, l.limit, retryAfter)
	}
	bucket.count++
	return nil
}
