package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
)

func TestCircuitBreaker_Success(t *testing.T) {
	config := DefaultConfig("test")
	config.OnStateChange = nil // Disable logging in tests
	cb := NewCircuitBreaker(config)

	result, err := cb.Execute(func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
}

func TestCircuitBreaker_Failure(t *testing.T) {
	config := DefaultConfig("test")
	config.OnStateChange = nil
	config.MaxFailures = 2
	cb := NewCircuitBreaker(config)

	// Trigger failures
	expectedErr := errors.New("test error")
	for i := 0; i < 5; i++ {
		_, err := cb.Execute(func() (any, error) {
			return nil, expectedErr
		})
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	// Circuit should be open now
	if cb.State() != gobreaker.StateOpen {
		t.Errorf("expected state Open, got %v", cb.State())
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := DefaultConfig("test")
	config.MaxFailures = 2
	config.Timeout = 100 * time.Millisecond
	config.OnStateChange = nil

	stateChanges := make([]string, 0)
	config.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
		stateChanges = append(stateChanges, to.String())
	}

	cb := NewCircuitBreaker(config)

	// Initial state should be closed
	if cb.State() != gobreaker.StateClosed {
		t.Errorf("expected initial state Closed, got %v", cb.State())
	}

	// Trigger failures to open circuit
	for i := 0; i < 5; i++ {
		cb.Execute(func() (any, error) {
			return nil, errors.New("fail")
		})
	}

	// Should be open
	if cb.State() != gobreaker.StateOpen {
		t.Errorf("expected state Open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to half-open
	cb.Execute(func() (any, error) {
		return "success", nil
	})

	if len(stateChanges) < 1 {
		t.Error("expected state changes, got none")
	}
}

func TestCircuitBreaker_WithContext(t *testing.T) {
	config := DefaultConfig("test")
	config.OnStateChange = nil
	cb := NewCircuitBreaker(config)

	ctx := context.Background()
	result, err := cb.ExecuteWithContext(ctx, func() (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
}

func TestCircuitBreaker_ContextCanceled(t *testing.T) {
	config := DefaultConfig("test")
	config.OnStateChange = nil
	cb := NewCircuitBreaker(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := cb.ExecuteWithContext(ctx, func() (any, error) {
		return "should not execute", nil
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestHTTPClientWrapper(t *testing.T) {
	wrapper := NewHTTPClientWrapper("test-http")
	ctx := context.Background()

	result, err := wrapper.Execute(ctx, func() (any, error) {
		return "http response", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "http response" {
		t.Errorf("expected 'http response', got %v", result)
	}
}

func TestCircuitBreaker_Counts(t *testing.T) {
	config := DefaultConfig("test")
	config.OnStateChange = nil
	cb := NewCircuitBreaker(config)

	// Execute some requests
	cb.Execute(func() (any, error) { return "ok", nil })
	cb.Execute(func() (any, error) { return nil, errors.New("fail") })
	cb.Execute(func() (any, error) { return "ok", nil })

	counts := cb.Counts()
	if counts.Requests != 3 {
		t.Errorf("expected 3 requests, got %d", counts.Requests)
	}
}
