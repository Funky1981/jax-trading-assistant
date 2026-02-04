package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	Enabled           bool
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		RequestsPerHour:   1000,
		Enabled:           true,
	}
}

// RateLimitConfigFromEnv creates rate limit configuration from environment variables
func RateLimitConfigFromEnv() RateLimitConfig {
	config := DefaultRateLimitConfig()

	if rpm := os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil && val > 0 {
			config.RequestsPerMinute = val
		}
	}

	if rph := os.Getenv("RATE_LIMIT_REQUESTS_PER_HOUR"); rph != "" {
		if val, err := strconv.Atoi(rph); err == nil && val > 0 {
			config.RequestsPerHour = val
		}
	}

	if enabled := os.Getenv("RATE_LIMIT_ENABLED"); enabled != "" {
		config.Enabled = enabled != "false" && enabled != "0"
	}

	return config
}

// clientBucket tracks request counts for a client
type clientBucket struct {
	minuteCount     int
	hourCount       int
	minuteResetTime time.Time
	hourResetTime   time.Time
	mu              sync.Mutex
}

// RateLimiter implements in-memory rate limiting
type RateLimiter struct {
	config  RateLimitConfig
	clients map[string]*clientBucket
	mu      sync.RWMutex
}

// NewRateLimiter creates a new in-memory rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	limiter := &RateLimiter{
		config:  config,
		clients: make(map[string]*clientBucket),
	}

	// Start cleanup goroutine to remove stale entries
	go limiter.cleanup()

	return limiter
}

// NewRateLimiterFromEnv creates a rate limiter from environment variables
func NewRateLimiterFromEnv() *RateLimiter {
	return NewRateLimiter(RateLimitConfigFromEnv())
}

// Allow checks if a request from the given client should be allowed
func (rl *RateLimiter) Allow(clientIP string) (bool, string) {
	if !rl.config.Enabled {
		return true, ""
	}

	now := time.Now()

	// Get or create client bucket
	rl.mu.RLock()
	bucket, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if !exists {
		bucket = &clientBucket{
			minuteResetTime: now.Add(time.Minute),
			hourResetTime:   now.Add(time.Hour),
		}
		rl.mu.Lock()
		rl.clients[clientIP] = bucket
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Reset minute counter if needed
	if now.After(bucket.minuteResetTime) {
		bucket.minuteCount = 0
		bucket.minuteResetTime = now.Add(time.Minute)
	}

	// Reset hour counter if needed
	if now.After(bucket.hourResetTime) {
		bucket.hourCount = 0
		bucket.hourResetTime = now.Add(time.Hour)
	}

	// Check limits
	if bucket.minuteCount >= rl.config.RequestsPerMinute {
		retryAfter := bucket.minuteResetTime.Sub(now)
		return false, fmt.Sprintf("Rate limit exceeded: %d requests per minute. Retry after %v", 
			rl.config.RequestsPerMinute, retryAfter.Round(time.Second))
	}

	if bucket.hourCount >= rl.config.RequestsPerHour {
		retryAfter := bucket.hourResetTime.Sub(now)
		return false, fmt.Sprintf("Rate limit exceeded: %d requests per hour. Retry after %v", 
			rl.config.RequestsPerHour, retryAfter.Round(time.Second))
	}

	// Increment counters
	bucket.minuteCount++
	bucket.hourCount++

	return true, ""
}

// cleanup removes stale client entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for ip, bucket := range rl.clients {
			bucket.mu.Lock()
			// Remove if both counters have expired
			if now.After(bucket.minuteResetTime) && now.After(bucket.hourResetTime) {
				if bucket.minuteCount == 0 && bucket.hourCount == 0 {
					delete(rl.clients, ip)
				}
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that enforces rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		
		allowed, message := rl.Allow(clientIP)
		if !allowed {
			log.Printf("Rate limit exceeded for IP: %s, Path: %s", clientIP, r.URL.Path)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", "0")
			http.Error(w, message, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns an HTTP middleware function that enforces rate limiting
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		
		allowed, message := rl.Allow(clientIP)
		if !allowed {
			log.Printf("Rate limit exceeded for IP: %s, Path: %s", clientIP, r.URL.Path)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", "0")
			http.Error(w, message, http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// getClientIP extracts the client IP address from the request
// Handles X-Forwarded-For and X-Real-IP headers for proxied requests
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (standard for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				return xff[:idx]
			}
		}
		return xff
	}

	// Check X-Real-IP header (used by some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in format "IP:port", so we need to strip the port
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"enabled":              rl.config.Enabled,
		"requests_per_minute":  rl.config.RequestsPerMinute,
		"requests_per_hour":    rl.config.RequestsPerHour,
		"active_clients":       len(rl.clients),
	}
}
