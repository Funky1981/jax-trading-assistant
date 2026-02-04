package resilience

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker/v2"
)

// CircuitBreakerConfig defines configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	MaxFailures   uint32
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
}

// DefaultConfig returns sensible defaults for a circuit breaker
func DefaultConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		MaxFailures: 5,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("[CircuitBreaker:%s] state changed: %s -> %s", name, from, to)
		},
	}
}

// CircuitBreaker wraps gobreaker with logging and configuration
type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker[any]
	name   string
	config CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker with the given config
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && (counts.ConsecutiveFailures >= config.MaxFailures || failureRatio >= 0.6)
		},
		OnStateChange: config.OnStateChange,
	}

	return &CircuitBreaker{
		cb:     gobreaker.NewCircuitBreaker[any](settings),
		name:   config.Name,
		config: config,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	result, err := cb.cb.Execute(fn)
	if err != nil {
		return nil, fmt.Errorf("circuit breaker %s: %w", cb.name, err)
	}
	return result, nil
}

// ExecuteWithContext runs the given function with context and circuit breaker protection
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() (any, error)) (any, error) {
	// Check context before executing
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Execute with circuit breaker
	result, err := cb.Execute(fn)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() gobreaker.State {
	return cb.cb.State()
}

// Counts returns the current counts
func (cb *CircuitBreaker) Counts() gobreaker.Counts {
	return cb.cb.Counts()
}

// Name returns the circuit breaker name
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// HTTPClientWrapper wraps HTTP calls with circuit breaker
type HTTPClientWrapper struct {
	cb *CircuitBreaker
}

// NewHTTPClientWrapper creates a new HTTP client wrapper with circuit breaker
func NewHTTPClientWrapper(name string) *HTTPClientWrapper {
	config := DefaultConfig(name)
	return &HTTPClientWrapper{
		cb: NewCircuitBreaker(config),
	}
}

// Execute executes an HTTP request with circuit breaker protection
func (w *HTTPClientWrapper) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	return w.cb.ExecuteWithContext(ctx, fn)
}

// GetCircuitBreaker returns the underlying circuit breaker for inspection
func (w *HTTPClientWrapper) GetCircuitBreaker() *CircuitBreaker {
	return w.cb
}
