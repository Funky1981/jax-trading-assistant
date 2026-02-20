package signalproduct

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ─── Validate ────────────────────────────────────────────────────────────────

func validSignal() *SignalProduct {
	return &SignalProduct{
		ID:         "sig-001",
		Version:    "1.0",
		ProducedAt: time.Now().UTC(),
		Symbol:     "AAPL",
		Direction:  DirectionBuy,
		Strength:   StrengthStrong,
		Confidence: 0.82,
		StrategyID: "rsi_momentum_v1",
	}
}

func TestValidate_Valid(t *testing.T) {
	if err := validSignal().Validate(); err != nil {
		t.Errorf("expected valid signal, got: %v", err)
	}
}

func TestValidate_MissingID(t *testing.T) {
	s := validSignal()
	s.ID = ""
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestValidate_MissingSymbol(t *testing.T) {
	s := validSignal()
	s.Symbol = ""
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing Symbol")
	}
}

func TestValidate_BadDirection(t *testing.T) {
	s := validSignal()
	s.Direction = "lateral"
	if err := s.Validate(); err == nil {
		t.Error("expected error for unknown Direction")
	}
}

func TestValidate_BadStrength(t *testing.T) {
	s := validSignal()
	s.Strength = "extreme"
	if err := s.Validate(); err == nil {
		t.Error("expected error for unknown Strength")
	}
}

func TestValidate_ConfidenceOutOfRange(t *testing.T) {
	s := validSignal()
	s.Confidence = 1.5
	if err := s.Validate(); err == nil {
		t.Error("expected error for confidence > 1")
	}
	s.Confidence = -0.1
	if err := s.Validate(); err == nil {
		t.Error("expected error for confidence < 0")
	}
}

func TestValidate_MissingStrategyID(t *testing.T) {
	s := validSignal()
	s.StrategyID = ""
	if err := s.Validate(); err == nil {
		t.Error("expected error for missing StrategyID")
	}
}

// ─── IsExpired ────────────────────────────────────────────────────────────────

func TestIsExpired_NoExpiry(t *testing.T) {
	s := validSignal()
	if s.IsExpired(time.Now()) {
		t.Error("signal with no expiry should never be expired")
	}
}

func TestIsExpired_Future(t *testing.T) {
	s := validSignal()
	exp := time.Now().Add(10 * time.Minute)
	s.ExpiresAt = &exp
	if s.IsExpired(time.Now()) {
		t.Error("signal with future expiry should not be expired yet")
	}
}

func TestIsExpired_Past(t *testing.T) {
	s := validSignal()
	exp := time.Now().Add(-1 * time.Second)
	s.ExpiresAt = &exp
	if !s.IsExpired(time.Now()) {
		t.Error("signal past expiry should be expired")
	}
}

// ─── Filters ─────────────────────────────────────────────────────────────────

func TestNoFilter(t *testing.T) {
	f := NoFilter()
	if !f(validSignal()) {
		t.Error("NoFilter should accept any signal")
	}
}

func TestBySymbol(t *testing.T) {
	f := BySymbol("AAPL", "MSFT")
	s := validSignal()
	if !f(s) {
		t.Error("expected AAPL to match")
	}
	s.Symbol = "TSLA"
	if f(s) {
		t.Error("expected TSLA to not match")
	}
}

func TestByStrategy(t *testing.T) {
	f := ByStrategy("rsi_momentum_v1")
	s := validSignal()
	if !f(s) {
		t.Error("expected strategy to match")
	}
	s.StrategyID = "macd_crossover_v1"
	if f(s) {
		t.Error("expected different strategy to not match")
	}
}

func TestByDirection(t *testing.T) {
	f := ByDirection(DirectionBuy)
	s := validSignal()
	if !f(s) {
		t.Error("expected buy direction to match")
	}
	s.Direction = DirectionSell
	if f(s) {
		t.Error("expected sell to not match buy filter")
	}
}

func TestByMinConfidence(t *testing.T) {
	f := ByMinConfidence(0.8)
	s := validSignal()
	s.Confidence = 0.82
	if !f(s) {
		t.Error("0.82 should pass min=0.8")
	}
	s.Confidence = 0.70
	if f(s) {
		t.Error("0.70 should not pass min=0.8")
	}
}

func TestByMinStrength(t *testing.T) {
	f := ByMinStrength(StrengthModerate)
	s := validSignal()
	s.Strength = StrengthStrong
	if !f(s) {
		t.Error("strong should pass moderate threshold")
	}
	s.Strength = StrengthModerate
	if !f(s) {
		t.Error("moderate should pass moderate threshold")
	}
	s.Strength = StrengthWeak
	if f(s) {
		t.Error("weak should not pass moderate threshold")
	}
}

func TestExcludeExpired(t *testing.T) {
	now := time.Now()
	f := ExcludeExpired(func() time.Time { return now })
	s := validSignal()

	// No expiry — allowed.
	if !f(s) {
		t.Error("non-expiring signal should pass")
	}

	// Expired.
	past := now.Add(-time.Second)
	s.ExpiresAt = &past
	if f(s) {
		t.Error("expired signal should be filtered out")
	}
}

func TestExcludeCancelled(t *testing.T) {
	f := ExcludeCancelled()
	s := validSignal()
	if !f(s) {
		t.Error("non-cancelled signal should pass")
	}
	s.Cancelled = true
	if f(s) {
		t.Error("cancelled signal should be filtered")
	}
}

func TestAllOf(t *testing.T) {
	f := AllOf(BySymbol("AAPL"), ByDirection(DirectionBuy))
	s := validSignal() // AAPL buy
	if !f(s) {
		t.Error("AAPL buy should pass AllOf")
	}
	s.Direction = DirectionSell
	if f(s) {
		t.Error("AAPL sell should not pass AllOf(buy)")
	}
}

func TestAnyOf(t *testing.T) {
	f := AnyOf(BySymbol("MSFT"), ByDirection(DirectionBuy))
	s := validSignal() // AAPL buy
	if !f(s) {
		t.Error("buy direction should satisfy AnyOf(MSFT or buy)")
	}
	s.Direction = DirectionSell
	s.Symbol = "TSLA"
	if f(s) {
		t.Error("TSLA sell should not pass AnyOf(MSFT or buy)")
	}
}

// ─── Publisher ───────────────────────────────────────────────────────────────

func TestPublisher_Subscribe_And_Publish(t *testing.T) {
	p := NewPublisher()
	var received []*SignalProduct
	var mu sync.Mutex

	unsub := p.Subscribe(nil, func(_ context.Context, sp *SignalProduct) {
		mu.Lock()
		received = append(received, sp)
		mu.Unlock()
	})
	defer unsub()

	s := validSignal()
	n, err := p.Publish(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 delivery, got %d", n)
	}
	mu.Lock()
	if len(received) != 1 {
		t.Errorf("expected 1 received signal, got %d", len(received))
	}
	mu.Unlock()
}

func TestPublisher_FilteredSubscription(t *testing.T) {
	p := NewPublisher()
	var buyCount, sellCount int

	p.Subscribe(ByDirection(DirectionBuy), func(_ context.Context, _ *SignalProduct) { buyCount++ })
	p.Subscribe(ByDirection(DirectionSell), func(_ context.Context, _ *SignalProduct) { sellCount++ })

	buy := validSignal()
	sell := validSignal()
	sell.Direction = DirectionSell

	p.Publish(context.Background(), buy)  //nolint:errcheck
	p.Publish(context.Background(), sell) //nolint:errcheck

	if buyCount != 1 {
		t.Errorf("expected buyCount=1, got %d", buyCount)
	}
	if sellCount != 1 {
		t.Errorf("expected sellCount=1, got %d", sellCount)
	}
}

func TestPublisher_Unsubscribe(t *testing.T) {
	p := NewPublisher()
	var count int
	unsub := p.Subscribe(nil, func(_ context.Context, _ *SignalProduct) { count++ })

	p.Publish(context.Background(), validSignal()) //nolint:errcheck
	unsub()
	p.Publish(context.Background(), validSignal()) //nolint:errcheck

	if count != 1 {
		t.Errorf("expected 1 delivery after unsubscribe, got %d", count)
	}
	if p.Len() != 0 {
		t.Errorf("expected 0 subscriptions after unsub, got %d", p.Len())
	}
}

func TestPublisher_InvalidSignalReturnsError(t *testing.T) {
	p := NewPublisher()
	bad := &SignalProduct{Symbol: "AAPL"} // missing ID, StrategyID
	_, err := p.Publish(context.Background(), bad)
	if err == nil {
		t.Error("expected validation error for invalid signal")
	}
}

func TestPublisher_Middleware_Drop(t *testing.T) {
	p := NewPublisher()
	var count int
	p.Subscribe(nil, func(_ context.Context, _ *SignalProduct) { count++ })

	// Middleware that drops all signals.
	p.Use(func(sp *SignalProduct) *SignalProduct { return nil })

	n, err := p.Publish(context.Background(), validSignal())
	if err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if n != 0 {
		t.Errorf("dropped signal should deliver to 0 subscribers, got %d", n)
	}
	if count != 0 {
		t.Errorf("handler should not be called for dropped signal")
	}
}

func TestPublisher_Middleware_Transform(t *testing.T) {
	p := NewPublisher()
	var got *SignalProduct

	p.Subscribe(nil, func(_ context.Context, sp *SignalProduct) { got = sp })

	// Middleware that downgrades confidence.
	p.Use(func(sp *SignalProduct) *SignalProduct {
		sp.Confidence = 0.10
		sp.Strength = StrengthWeak
		return sp
	})

	p.Publish(context.Background(), validSignal()) //nolint:errcheck

	if got == nil {
		t.Fatal("expected signal to be received")
	}
	if got.Confidence != 0.10 {
		t.Errorf("expected confidence=0.10, got %f", got.Confidence)
	}
}

func TestPublisher_ConcurrentPublish(t *testing.T) {
	p := NewPublisher()
	var mu sync.Mutex
	count := 0
	p.Subscribe(nil, func(_ context.Context, _ *SignalProduct) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := validSignal()
			s.ID = fmt.Sprintf("sig-%d", i)
			p.Publish(context.Background(), s) //nolint:errcheck
		}(i)
	}
	wg.Wait()

	mu.Lock()
	if count != n {
		t.Errorf("expected %d deliveries, got %d", n, count)
	}
	mu.Unlock()
}

// ─── StrengthFromConfidence ───────────────────────────────────────────────────

func TestStrengthFromConfidence(t *testing.T) {
	cases := []struct {
		confidence float64
		want       Strength
	}{
		{0.0, StrengthWeak},
		{0.49, StrengthWeak},
		{0.50, StrengthModerate},
		{0.74, StrengthModerate},
		{0.75, StrengthStrong},
		{1.0, StrengthStrong},
	}
	for _, tc := range cases {
		got := StrengthFromConfidence(tc.confidence)
		if got != tc.want {
			t.Errorf("StrengthFromConfidence(%.2f) = %s, want %s", tc.confidence, got, tc.want)
		}
	}
}
