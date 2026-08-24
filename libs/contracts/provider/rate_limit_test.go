package provider

import (
	"testing"
	"time"
)

func TestRateLimiterSeparatesStaticPolicyFromCapabilityRuntimeState(t *testing.T) {
	policy := syntheticOperationalPolicy().RateLimit
	policy.RequestLimit = 2
	policy.ConcurrencyLimit = 2
	policy.Window = 10 * time.Second
	limiter, err := NewRateLimiter(policy)
	if err != nil {
		t.Fatal(err)
	}
	definition := validDefinition(CapabilityCorporateFiling)
	operation := syntheticOperation(definition)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	for attempt := 0; attempt < 2; attempt++ {
		decision, err := limiter.Acquire(operation, now)
		if err != nil || !decision.Allowed {
			t.Fatalf("Acquire(%d) = %+v, %v", attempt, decision, err)
		}
		limiter.Release(operation)
	}
	decision, err := limiter.MayAttempt(operation, now)
	if err != nil || decision.Allowed || decision.Reason != RateLimitLocalWindowExhausted || decision.RetryAt == nil || !decision.RetryAt.Equal(now.Add(10*time.Second)) {
		t.Fatalf("exhausted decision = %+v, %v", decision, err)
	}
	decision, err = limiter.MayAttempt(operation, now.Add(10*time.Second))
	if err != nil || !decision.Allowed || decision.RequestsRemaining != 2 {
		t.Fatalf("reset decision = %+v, %v", decision, err)
	}
	if policy.RequestLimit != 2 || policy.Window != 10*time.Second {
		t.Fatal("runtime limiter mutated static policy")
	}
}

func TestRateLimiterEnforcesConcurrencyAndCapabilityScope(t *testing.T) {
	policy := syntheticOperationalPolicy().RateLimit
	policy.ConcurrencyLimit = 1
	limiter, err := NewRateLimiter(policy)
	if err != nil {
		t.Fatal(err)
	}
	definition := validDefinition(CapabilityCorporateFiling)
	operation := syntheticOperation(definition)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if decision, err := limiter.Acquire(operation, now); err != nil || !decision.Allowed {
		t.Fatalf("first acquire = %+v, %v", decision, err)
	}
	if decision, err := limiter.Acquire(operation, now); err != nil || decision.Allowed || decision.Reason != RateLimitConcurrencyExhausted {
		t.Fatalf("concurrent acquire = %+v, %v", decision, err)
	}

	other := operation
	other.CapabilityID = CapabilityNewsArticle
	if decision, err := limiter.Acquire(other, now); err != nil || !decision.Allowed {
		t.Fatalf("separate capability acquire = %+v, %v", decision, err)
	}
	limiter.Release(other)
	limiter.Release(operation)
}

func TestRateLimiterHonoursProviderRemainingAndResetObservation(t *testing.T) {
	policy := syntheticOperationalPolicy().RateLimit
	limiter, err := NewRateLimiter(policy)
	if err != nil {
		t.Fatal(err)
	}
	operation := syntheticOperation(validDefinition(CapabilityCorporateFiling))
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	zero := uint64(0)
	reset := now.Add(8 * time.Second)
	if err := limiter.Observe(operation, now, ProviderRateLimitObservation{Remaining: &zero, ResetAt: &reset}); err != nil {
		t.Fatal(err)
	}
	decision, err := limiter.MayAttempt(operation, now)
	if err != nil || decision.Allowed || decision.Reason != RateLimitProviderReportedExhausted || decision.RetryAt == nil || !decision.RetryAt.Equal(reset) {
		t.Fatalf("provider exhaustion = %+v, %v", decision, err)
	}
	decision, err = limiter.MayAttempt(operation, reset)
	if err != nil || !decision.Allowed {
		t.Fatalf("provider reset = %+v, %v", decision, err)
	}

	if err := limiter.Observe(operation, reset, ProviderRateLimitObservation{}); err == nil {
		t.Fatal("Observe() accepted empty provider state")
	}
	nonUTC := reset.In(time.FixedZone("not-utc", 60))
	if err := limiter.Observe(operation, reset, ProviderRateLimitObservation{ResetAt: &nonUTC}); err == nil {
		t.Fatal("Observe() accepted non-UTC reset")
	}
}
