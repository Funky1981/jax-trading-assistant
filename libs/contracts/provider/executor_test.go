package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeTimeSource struct {
	mu                   sync.Mutex
	now                  time.Time
	sleeps               []time.Duration
	forceAttemptDeadline bool
	cancelDuringSleep    context.CancelFunc
}

func (source *fakeTimeSource) Now() time.Time {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.now
}

func (source *fakeTimeSource) Sleep(ctx context.Context, delay time.Duration) error {
	source.mu.Lock()
	source.sleeps = append(source.sleeps, delay)
	cancel := source.cancelDuringSleep
	if cancel == nil {
		source.now = source.now.Add(delay)
	}
	source.mu.Unlock()
	if cancel != nil {
		cancel()
		return ctx.Err()
	}
	return ctx.Err()
}

func (source *fakeTimeSource) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	source.mu.Lock()
	deadline := source.now.Add(timeout)
	force := source.forceAttemptDeadline
	source.mu.Unlock()
	if force {
		return &fixedDeadlineContext{Context: ctx, deadline: deadline, expired: true}, func() {}
	}
	return &fixedDeadlineContext{Context: ctx, deadline: deadline}, func() {}
}

func (source *fakeTimeSource) Sleeps() []time.Duration {
	source.mu.Lock()
	defer source.mu.Unlock()
	return append([]time.Duration(nil), source.sleeps...)
}

type fixedDeadlineContext struct {
	context.Context
	deadline time.Time
	expired  bool
}

func (ctx *fixedDeadlineContext) Deadline() (time.Time, bool) { return ctx.deadline, true }
func (ctx *fixedDeadlineContext) Done() <-chan struct{} {
	if !ctx.expired {
		return ctx.Context.Done()
	}
	closed := make(chan struct{})
	close(closed)
	return closed
}
func (ctx *fixedDeadlineContext) Err() error {
	if ctx.expired {
		return context.DeadlineExceeded
	}
	return ctx.Context.Err()
}

func newOperationalExecutorFixture(t *testing.T, policy OperationalPolicy, source *fakeTimeSource) (*OperationalExecutor, *Registry, ProviderDefinition, *MemoryInstrumentation, Operation) {
	t.Helper()
	registry, definition := rawPayloadRegistry(t)
	sink := NewMemoryInstrumentation()
	executor, err := NewOperationalExecutor(registry, policy, source, nil, sink)
	if err != nil {
		t.Fatal(err)
	}
	return executor, registry, definition, sink, syntheticOperation(definition)
}

func requireOperationalCode(t *testing.T, err error, want OperationalErrorCode) {
	t.Helper()
	var operational *OperationalError
	if !errors.As(err, &operational) {
		t.Fatalf("error type = %T, want *OperationalError: %v", err, err)
	}
	if operational.Code != want {
		t.Fatalf("error code = %q, want %q: %v", operational.Code, want, err)
	}
}

func TestOperationalExecutorRepresentativeRetryAfterRawHandoffAndInstrumentation(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	source := &fakeTimeSource{now: start}
	executor, registry, definition, sink, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
	exact := []byte("{\r\n  \"filing\": \"exact provider bytes\"\r\n}\r\n")
	calls := 0
	result, err := executor.Execute(context.Background(), operation, func(_ context.Context, attempt AttemptContext) ProviderAttemptResult {
		calls++
		if attempt.Attempt != calls || !attempt.Deadline.Equal(attempt.StartedAt.Add(5*time.Second)) {
			t.Fatalf("attempt context = %+v calls=%d", attempt, calls)
		}
		switch calls {
		case 1:
			return ProviderAttemptResult{Failure: &ProviderFailure{Class: FailureTransportTransient}}
		case 2:
			return ProviderAttemptResult{Failure: &ProviderFailure{HTTPStatus: 429, RetryAfter: "2"}}
		default:
			return ProviderAttemptResult{RawBytes: exact}
		}
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != ExecutionSucceeded || result.Attempts != 3 || calls != 3 || string(result.RawBytes) != string(exact) {
		t.Fatalf("execution result = %+v calls=%d", result, calls)
	}
	if got := source.Sleeps(); len(got) != 2 || got[0] != time.Second || got[1] != 2*time.Second {
		t.Fatalf("deterministic sleeps = %v", got)
	}
	if len(result.Retries) != 2 || result.Retries[0].RetryAfter.Present || !result.Retries[1].RetryAfter.Present || !result.Retries[1].RetryAfter.Honoured || result.Retries[1].RetryAfter.Kind != RetryAfterDeltaSeconds {
		t.Fatalf("retry decisions = %+v", result.Retries)
	}
	if result.Health.Status != RuntimeHealthy || result.Identities.RetryPolicy.ID != syntheticOperationalPolicy().Retry.Identity.ID {
		t.Fatalf("health/identities = %+v / %+v", result.Health, result.Identities)
	}

	store := NewMemoryRawPayloadStore()
	descriptor, err := PersistRawPayload(context.Background(), registry, store, validRawPayloadRequest(definition, "rpa_operational_success", result.CompletedAt), result.RawBytes)
	if err != nil {
		t.Fatalf("PersistRawPayload() error = %v", err)
	}
	verified, err := RetrieveRawPayload(context.Background(), store, descriptor.Ref)
	if err != nil || string(verified) != string(exact) || descriptor.Ref.Content.Digest.VerifyBytes(exact) != nil {
		t.Fatalf("raw handoff bytes=%q descriptor=%+v err=%v", verified, descriptor, err)
	}

	counts := map[OperationalEventKind]int{}
	for _, event := range sink.Events() {
		counts[event.Kind]++
		if err := event.Validate(); err != nil {
			t.Fatalf("invalid instrumentation event: %+v: %v", event, err)
		}
	}
	if counts[OperationalEventAttempt] != 3 || counts[OperationalEventRetryScheduled] != 2 || counts[OperationalEventRateLimitResponse] != 1 || counts[OperationalEventSuccess] != 1 || counts[OperationalEventTerminalFailure] != 0 || counts[OperationalEventHealthTransition] != 2 {
		t.Fatalf("instrumentation counts = %+v events=%+v", counts, sink.Events())
	}
}

func TestOperationalExecutorDoesNotRetryNonRetryableFailures(t *testing.T) {
	cases := []struct {
		name       string
		failure    ProviderFailure
		wantClass  FailureClass
		wantHealth RuntimeStatus
	}{
		{name: "authentication", failure: ProviderFailure{HTTPStatus: 401}, wantClass: FailureAuthentication, wantHealth: RuntimeUnavailable},
		{name: "malformed request", failure: ProviderFailure{HTTPStatus: 400}, wantClass: FailureMalformedRequest, wantHealth: RuntimeUnknown},
		{name: "unsupported capability", failure: ProviderFailure{Class: FailureUnsupportedCapability}, wantClass: FailureUnsupportedCapability, wantHealth: RuntimeUnavailable},
		{name: "payload parse", failure: ProviderFailure{Class: FailureProviderPayloadParse}, wantClass: FailureProviderPayloadParse, wantHealth: RuntimeUnknown},
		{name: "canonical validation", failure: ProviderFailure{Class: FailureCanonicalValidation}, wantClass: FailureCanonicalValidation, wantHealth: RuntimeUnknown},
		{name: "provenance", failure: ProviderFailure{Class: FailureProvenanceMismatch}, wantClass: FailureProvenanceMismatch, wantHealth: RuntimeUnknown},
		{name: "permanent rejection", failure: ProviderFailure{Class: FailurePermanentRejection}, wantClass: FailurePermanentRejection, wantHealth: RuntimeUnknown},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			source := &fakeTimeSource{now: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)}
			executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
			calls := 0
			result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
				calls++
				failure := test.failure
				return ProviderAttemptResult{Failure: &failure}
			})
			requireOperationalCode(t, err, ErrorNonRetryableFailure)
			if calls != 1 || result.Attempts != 1 || len(result.RawBytes) != 0 || result.FailureClass != test.wantClass || result.Health.Status != test.wantHealth || len(source.Sleeps()) != 0 {
				t.Fatalf("result=%+v calls=%d sleeps=%v", result, calls, source.Sleeps())
			}
		})
	}
}

func TestOperationalExecutorRejectsEmptyOrMixedSuccessEnvelopes(t *testing.T) {
	cases := []ProviderAttemptResult{
		{},
		{RawBytes: []byte("fake success"), Failure: &ProviderFailure{Class: FailureTransportTransient}},
	}
	for index, attemptResult := range cases {
		source := &fakeTimeSource{now: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)}
		executor, _, _, sink, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
		result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			return attemptResult
		})
		requireOperationalCode(t, err, ErrorInvalidAttemptResult)
		if result.Status != ExecutionFailed || len(result.RawBytes) != 0 || eventCount(sink.Events(), OperationalEventSuccess) != 0 {
			t.Fatalf("case %d published fake success: result=%+v events=%+v", index, result, sink.Events())
		}
	}
}

func TestOperationalExecutorRetryAndElapsedBudgetsTerminateExplicitly(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	t.Run("maximum attempts", func(t *testing.T) {
		source := &fakeTimeSource{now: start}
		executor, _, _, sink, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
		calls := 0
		result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			calls++
			return ProviderAttemptResult{Failure: &ProviderFailure{HTTPStatus: 503}}
		})
		requireOperationalCode(t, err, ErrorRetryBudgetExhausted)
		if calls != 3 || result.Attempts != 3 || result.Status != ExecutionFailed || len(result.RawBytes) != 0 || result.Health.Status != RuntimeUnavailable {
			t.Fatalf("exhausted result=%+v calls=%d", result, calls)
		}
		if eventCount(sink.Events(), OperationalEventTerminalFailure) != 1 || eventCount(sink.Events(), OperationalEventSuccess) != 0 {
			t.Fatalf("terminal instrumentation = %+v", sink.Events())
		}
	})

	t.Run("maximum elapsed", func(t *testing.T) {
		policy := syntheticOperationalPolicy()
		policy.Retry.MaximumElapsed = 2500 * time.Millisecond
		policy.Retry.PerAttemptTimeout = time.Second
		policy.Retry.Backoff.MaximumDelay = 2 * time.Second
		policy.RateLimit.MaximumProviderDelay = 2 * time.Second
		source := &fakeTimeSource{now: start}
		executor, _, _, _, operation := newOperationalExecutorFixture(t, policy, source)
		calls := 0
		result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			calls++
			return ProviderAttemptResult{Failure: &ProviderFailure{Class: FailureTransportTransient}}
		})
		requireOperationalCode(t, err, ErrorElapsedBudgetExhausted)
		if calls != 2 || result.Attempts != 2 || len(result.RawBytes) != 0 {
			t.Fatalf("elapsed result=%+v calls=%d", result, calls)
		}
	})
}

func TestOperationalExecutorRetryAfterFailsVisiblyWhenInvalidOrUnhonourable(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		value string
		code  OperationalErrorCode
	}{
		{name: "malformed", value: "later", code: ErrorMalformedRetryAfter},
		{name: "excessive", value: "11", code: ErrorExcessiveRetryAfter},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			source := &fakeTimeSource{now: start}
			executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
			result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
				return ProviderAttemptResult{Failure: &ProviderFailure{HTTPStatus: 429, RetryAfter: test.value}}
			})
			requireOperationalCode(t, err, test.code)
			if result.Attempts != 1 || len(result.RawBytes) != 0 || len(source.Sleeps()) != 0 {
				t.Fatalf("Retry-After failure result=%+v sleeps=%v", result, source.Sleeps())
			}
		})
	}

	source := &fakeTimeSource{now: start}
	executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
	deadlineContext := &fixedDeadlineContext{Context: context.Background(), deadline: start.Add(1500 * time.Millisecond)}
	result, err := executor.Execute(deadlineContext, operation, func(context.Context, AttemptContext) ProviderAttemptResult {
		return ProviderAttemptResult{Failure: &ProviderFailure{HTTPStatus: 429, RetryAfter: "2"}}
	})
	requireOperationalCode(t, err, ErrorUnhonourableRetryAfter)
	if result.Attempts != 1 || len(source.Sleeps()) != 0 {
		t.Fatalf("deadline Retry-After result=%+v sleeps=%v", result, source.Sleeps())
	}
}

func TestOperationalExecutorCancellationAndDeadlinesDoNotBlameHealth(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	t.Run("caller cancellation before attempt", func(t *testing.T) {
		source := &fakeTimeSource{now: start}
		executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		result, err := executor.Execute(ctx, operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			calls++
			return ProviderAttemptResult{RawBytes: []byte("unexpected")}
		})
		requireOperationalCode(t, err, ErrorCancelled)
		if calls != 0 || result.Attempts != 0 || result.Health.Status != RuntimeUnknown || result.Health.ConsecutiveFailures != 0 {
			t.Fatalf("cancelled result=%+v calls=%d", result, calls)
		}
	})

	t.Run("caller deadline before attempt", func(t *testing.T) {
		source := &fakeTimeSource{now: start}
		executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
		ctx := &fixedDeadlineContext{Context: context.Background(), deadline: start, expired: true}
		result, err := executor.Execute(ctx, operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			t.Fatal("provider call ran after caller deadline")
			return ProviderAttemptResult{}
		})
		requireOperationalCode(t, err, ErrorDeadlineExceeded)
		if result.Health.Status != RuntimeUnknown || result.Health.ConsecutiveFailures != 0 {
			t.Fatalf("deadline blamed health: %+v", result.Health)
		}
	})

	t.Run("per-attempt deadline policy ignored by health", func(t *testing.T) {
		source := &fakeTimeSource{now: start, forceAttemptDeadline: true}
		policy := syntheticOperationalPolicy()
		policy.Retry.MaximumAttempts = 2
		executor, _, _, _, operation := newOperationalExecutorFixture(t, policy, source)
		result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			return ProviderAttemptResult{RawBytes: []byte("late bytes must be rejected")}
		})
		requireOperationalCode(t, err, ErrorRetryBudgetExhausted)
		if result.Attempts != 2 || result.Health.Status != RuntimeUnknown || result.Health.ConsecutiveFailures != 0 || len(result.RawBytes) != 0 {
			t.Fatalf("attempt deadline result=%+v", result)
		}
	})

	t.Run("cancellation interrupts backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		source := &fakeTimeSource{now: start, cancelDuringSleep: cancel}
		executor, _, _, _, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
		result, err := executor.Execute(ctx, operation, func(context.Context, AttemptContext) ProviderAttemptResult {
			return ProviderAttemptResult{Failure: &ProviderFailure{Class: FailureTransportTransient}}
		})
		requireOperationalCode(t, err, ErrorCancelled)
		if result.Attempts != 1 || result.Health.ConsecutiveFailures != 1 || len(result.RawBytes) != 0 {
			t.Fatalf("backoff cancellation result=%+v", result)
		}
	})
}

func TestOperationalExecutorLocalThrottleAndStaticSupportFailBeforeProviderCall(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	policy := syntheticOperationalPolicy()
	policy.RateLimit.RequestLimit = 1
	policy.Retry.MaximumAttempts = 1
	source := &fakeTimeSource{now: start}
	executor, _, _, sink, operation := newOperationalExecutorFixture(t, policy, source)
	if _, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
		return ProviderAttemptResult{Failure: &ProviderFailure{Class: FailureTransportTransient}}
	}); err == nil {
		t.Fatal("first call unexpectedly succeeded")
	}
	calls := 0
	result, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
		calls++
		return ProviderAttemptResult{RawBytes: []byte("must not run")}
	})
	requireOperationalCode(t, err, ErrorLocalThrottled)
	if calls != 0 || result.Attempts != 0 || eventCount(sink.Events(), OperationalEventThrottled) != 1 {
		t.Fatalf("local throttle result=%+v calls=%d events=%+v", result, calls, sink.Events())
	}

	registry := mustRegistry(t)
	definition := validDefinition(CapabilityCorporateFiling)
	definition.Capabilities[0].Support = SupportUnavailable
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	executor, err = NewOperationalExecutor(registry, syntheticOperationalPolicy(), &fakeTimeSource{now: start}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err = executor.Execute(context.Background(), syntheticOperation(definition), func(context.Context, AttemptContext) ProviderAttemptResult {
		calls++
		return ProviderAttemptResult{RawBytes: []byte("must not run")}
	})
	requireOperationalCode(t, err, ErrorStaticCapabilityUnavailable)
	if result.Attempts != 0 || result.Health.Status != RuntimeUnavailable {
		t.Fatalf("static unavailable attempted provider: %+v", result)
	}
}

func TestOperationalInstrumentationCannotCarrySecretsOrUnboundedLabels(t *testing.T) {
	start := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	source := &fakeTimeSource{now: start}
	executor, _, _, sink, operation := newOperationalExecutorFixture(t, syntheticOperationalPolicy(), source)
	secret := "Authorization=Bearer top-secret api_key=never-log"
	_, err := executor.Execute(context.Background(), operation, func(context.Context, AttemptContext) ProviderAttemptResult {
		return ProviderAttemptResult{Failure: &ProviderFailure{HTTPStatus: 401, Cause: errors.New(secret)}}
	})
	if err == nil || strings.Contains(err.Error(), "top-secret") || strings.Contains(err.Error(), "api_key") {
		t.Fatalf("safe error leaked secret or was nil: %v", err)
	}
	raw, marshalErr := json.Marshal(sink.Events())
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	text := string(raw)
	for _, forbidden := range []string{"top-secret", "api_key", "bearer", "raw_payload", "request_id", "symbol", "url"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("instrumentation contains forbidden value/label %q: %s", forbidden, text)
		}
	}
}

func eventCount(events []OperationalEvent, kind OperationalEventKind) int {
	count := 0
	for _, event := range events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}
