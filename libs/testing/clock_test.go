package testing

import (
	"context"
	"testing"
	"time"
)

func TestSystemClock(t *testing.T) {
	clock := SystemClock{}

	before := time.Now()
	clockTime := clock.Now()
	after := time.Now()

	// Clock time should be between before and after
	if clockTime.Before(before) || clockTime.After(after) {
		t.Errorf("SystemClock.Now() returned time outside expected range: %v (should be between %v and %v)",
			clockTime, before, after)
	}
}

func TestFixedClock(t *testing.T) {
	fixedTime := time.Date(2026, 2, 13, 9, 30, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}

	// Should always return the same time
	for i := 0; i < 10; i++ {
		got := clock.Now()
		if !got.Equal(fixedTime) {
			t.Errorf("FixedClock.Now() = %v, want %v", got, fixedTime)
		}
		time.Sleep(1 * time.Millisecond) // Sleep to ensure system time changes
	}
}

func TestManualClock(t *testing.T) {
	startTime := time.Date(2026, 2, 13, 9, 30, 0, 0, time.UTC)
	clock := NewManualClock(startTime)

	// Initial time
	if got := clock.Now(); !got.Equal(startTime) {
		t.Errorf("ManualClock.Now() = %v, want %v", got, startTime)
	}

	// Advance by 1 hour
	clock.Advance(1 * time.Hour)
	expected := startTime.Add(1 * time.Hour)
	if got := clock.Now(); !got.Equal(expected) {
		t.Errorf("After Advance(1h), ManualClock.Now() = %v, want %v", got, expected)
	}

	// Advance by 30 minutes
	clock.Advance(30 * time.Minute)
	expected = expected.Add(30 * time.Minute)
	if got := clock.Now(); !got.Equal(expected) {
		t.Errorf("After Advance(30m), ManualClock.Now() = %v, want %v", got, expected)
	}

	// Set to specific time
	newTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	clock.Set(newTime)
	if got := clock.Now(); !got.Equal(newTime) {
		t.Errorf("After Set(), ManualClock.Now() = %v, want %v", got, newTime)
	}
}

func TestWithClock(t *testing.T) {
	fixedTime := time.Date(2026, 2, 13, 9, 30, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}

	ctx := context.Background()
	ctxWithClock := WithClock(ctx, clock)

	// Context should contain the clock
	retrievedClock := ClockFromContext(ctxWithClock)
	if retrievedClock == nil {
		t.Fatal("ClockFromContext() returned nil")
	}

	// Clock should return the fixed time
	if got := retrievedClock.Now(); !got.Equal(fixedTime) {
		t.Errorf("Clock from context returned %v, want %v", got, fixedTime)
	}
}

func TestClockFromContextDefault(t *testing.T) {
	ctx := context.Background()

	// Should return SystemClock when no clock is set
	clock := ClockFromContext(ctx)
	if clock == nil {
		t.Fatal("ClockFromContext() returned nil")
	}

	// Should return current system time (within a reasonable range)
	before := time.Now()
	clockTime := clock.Now()
	after := time.Now()

	if clockTime.Before(before.Add(-1*time.Second)) || clockTime.After(after.Add(1*time.Second)) {
		t.Errorf("Default clock returned time outside expected range: %v", clockTime)
	}
}

func TestNowConvenienceFunction(t *testing.T) {
	fixedTime := time.Date(2026, 2, 13, 14, 45, 30, 0, time.UTC)
	clock := FixedClock{T: fixedTime}
	ctx := WithClock(context.Background(), clock)

	// Now() convenience function should work
	got := Now(ctx)
	if !got.Equal(fixedTime) {
		t.Errorf("Now(ctx) = %v, want %v", got, fixedTime)
	}
}

func TestNowConvenienceFunctionWithSystemClock(t *testing.T) {
	ctx := context.Background()

	before := time.Now()
	got := Now(ctx)
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now(ctx) with system clock returned time outside expected range: %v", got)
	}
}

func TestMultipleClockTypes(t *testing.T) {
	tests := []struct {
		name     string
		clock    Clock
		validate func(t *testing.T, clock Clock)
	}{
		{
			name:  "SystemClock",
			clock: SystemClock{},
			validate: func(t *testing.T, clock Clock) {
				before := time.Now()
				got := clock.Now()
				after := time.Now()
				if got.Before(before) || got.After(after) {
					t.Errorf("Time %v not in range [%v, %v]", got, before, after)
				}
			},
		},
		{
			name:  "FixedClock",
			clock: FixedClock{T: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			validate: func(t *testing.T, clock Clock) {
				expected := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				got := clock.Now()
				if !got.Equal(expected) {
					t.Errorf("Got %v, want %v", got, expected)
				}
			},
		},
		{
			name:  "ManualClock",
			clock: NewManualClock(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)),
			validate: func(t *testing.T, clock Clock) {
				expected := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
				got := clock.Now()
				if !got.Equal(expected) {
					t.Errorf("Got %v, want %v", got, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.clock)
		})
	}
}

func TestClockPropagationThroughContext(t *testing.T) {
	// Create a chain of contexts
	fixedTime := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}

	ctx1 := context.Background()
	ctx2 := WithClock(ctx1, clock)
	ctx3 := context.WithValue(ctx2, "key", "value")
	ctx4, cancel := context.WithTimeout(ctx3, 5*time.Minute)
	defer cancel()

	// Clock should still be retrievable from ctx4
	retrievedClock := ClockFromContext(ctx4)
	got := retrievedClock.Now()
	if !got.Equal(fixedTime) {
		t.Errorf("Clock propagation failed: got %v, want %v", got, fixedTime)
	}
}

func TestConcurrentClockAccess(t *testing.T) {
	fixedTime := time.Date(2026, 2, 13, 11, 30, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}
	ctx := WithClock(context.Background(), clock)

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			got := Now(ctx)
			if !got.Equal(fixedTime) {
				t.Errorf("Concurrent access returned %v, want %v", got, fixedTime)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

func BenchmarkClockFromContext(b *testing.B) {
	fixedTime := time.Date(2026, 2, 13, 9, 30, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}
	ctx := WithClock(context.Background(), clock)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClockFromContext(ctx)
	}
}

func BenchmarkNow(b *testing.B) {
	fixedTime := time.Date(2026, 2, 13, 9, 30, 0, 0, time.UTC)
	clock := FixedClock{T: fixedTime}
	ctx := WithClock(context.Background(), clock)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Now(ctx)
	}
}

func BenchmarkSystemClockNow(b *testing.B) {
	clock := SystemClock{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clock.Now()
	}
}

func BenchmarkFixedClockNow(b *testing.B) {
	clock := FixedClock{T: time.Now()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clock.Now()
	}
}
