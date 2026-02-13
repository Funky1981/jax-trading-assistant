package testing

import (
	"context"
	"time"
)

// Clock provides time for strategies (injectable for tests)
type Clock interface {
	Now() time.Time
}

// SystemClock uses real system time (production)
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

// FixedClock returns fixed time (tests)
type FixedClock struct {
	T time.Time
}

func (fc FixedClock) Now() time.Time {
	return fc.T
}

// ManualClock allows manual control of time for testing
type ManualClock struct {
	current time.Time
}

// NewManualClock creates a new manual clock with the given start time
func NewManualClock(start time.Time) *ManualClock {
	return &ManualClock{current: start}
}

func (mc *ManualClock) Now() time.Time {
	return mc.current
}

// Advance moves the clock forward by the given duration
func (mc *ManualClock) Advance(d time.Duration) {
	mc.current = mc.current.Add(d)
}

// Set sets the clock to a specific time
func (mc *ManualClock) Set(t time.Time) {
	mc.current = t
}

// Context keys
type clockKey struct{}

// WithClock returns a new context with the given clock
func WithClock(ctx context.Context, c Clock) context.Context {
	return context.WithValue(ctx, clockKey{}, c)
}

// ClockFromContext retrieves the clock from the context.
// If no clock is set, returns SystemClock as the default.
func ClockFromContext(ctx context.Context) Clock {
	if c, ok := ctx.Value(clockKey{}).(Clock); ok {
		return c
	}
	return SystemClock{} // default to system time
}

// Now is a convenience function that gets the current time from the context's clock
func Now(ctx context.Context) time.Time {
	return ClockFromContext(ctx).Now()
}
