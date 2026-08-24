package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type TimeSource interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
	WithTimeout(context.Context, time.Duration) (context.Context, context.CancelFunc)
}

type SystemTimeSource struct{}

func (SystemTimeSource) Now() time.Time { return time.Now().UTC() }

func (SystemTimeSource) Sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (SystemTimeSource) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

type AttemptContext struct {
	Operation Operation `json:"operation"`
	Attempt   int       `json:"attempt"`
	StartedAt time.Time `json:"started_at"`
	Deadline  time.Time `json:"deadline"`
}

type ProviderCall func(context.Context, AttemptContext) ProviderAttemptResult

type RetryDecision struct {
	AfterAttempt   int                `json:"after_attempt"`
	FailureClass   FailureClass       `json:"failure_class"`
	BackoffDelay   time.Duration      `json:"backoff_delay"`
	LocalDelay     time.Duration      `json:"local_delay"`
	ScheduledDelay time.Duration      `json:"scheduled_delay"`
	ScheduledAt    time.Time          `json:"scheduled_at"`
	RetryAt        time.Time          `json:"retry_at"`
	RetryAfter     RetryAfterDecision `json:"retry_after"`
}

type ExecutionStatus string

const (
	ExecutionSucceeded ExecutionStatus = "SUCCEEDED"
	ExecutionFailed    ExecutionStatus = "FAILED"
)

// ExecutionResult never contains fallback/canonical data. RawBytes are set
// only for a successful provider acquisition and remain exact for the
// WP-02.02 persistence handoff.
type ExecutionResult struct {
	Status       ExecutionStatus        `json:"status"`
	RawBytes     []byte                 `json:"-"`
	Attempts     int                    `json:"attempts"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  time.Time              `json:"completed_at"`
	FailureClass FailureClass           `json:"failure_class,omitempty"`
	TerminalCode OperationalErrorCode   `json:"terminal_code,omitempty"`
	Retries      []RetryDecision        `json:"retries,omitempty"`
	Health       HealthAssessment       `json:"health"`
	Identities   OperationalIdentitySet `json:"identities"`
}

type OperationalExecutor struct {
	providers *Registry
	policy    OperationalPolicy
	time      TimeSource
	jitter    JitterSource
	sink      InstrumentationSink
	limiter   *RateLimiter

	healthMu sync.Mutex
	health   map[string]*HealthTracker
}

func NewOperationalExecutor(providers *Registry, policy OperationalPolicy, source TimeSource, jitter JitterSource, sink InstrumentationSink) (*OperationalExecutor, error) {
	if providers == nil || providers.ContractVersion() != RegistryContractV1 {
		return nil, fmt.Errorf("operational executor: valid provider registry is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("operational executor: time source is required")
	}
	if policy.Retry.Backoff.Jitter == JitterProportional && jitter == nil {
		return nil, fmt.Errorf("operational executor: jitter source is required by policy")
	}
	limiter, err := NewRateLimiter(policy.RateLimit)
	if err != nil {
		return nil, err
	}
	if sink == nil {
		sink = discardInstrumentation{}
	}
	return &OperationalExecutor{
		providers: providers, policy: policy, time: source, jitter: jitter, sink: sink, limiter: limiter,
		health: make(map[string]*HealthTracker),
	}, nil
}

func (executor *OperationalExecutor) Policy() OperationalPolicy {
	return cloneOperationalPolicy(executor.policy)
}

func (executor *OperationalExecutor) MayAttempt(ctx context.Context, operation Operation) (RateLimitDecision, error) {
	if err := executor.validateOperation(ctx, operation); err != nil {
		return RateLimitDecision{}, err
	}
	return executor.limiter.MayAttempt(operation, executor.time.Now())
}

func (executor *OperationalExecutor) Health(operation Operation) (HealthAssessment, error) {
	_, _, tracker, err := executor.operationState(operation)
	if err != nil {
		return HealthAssessment{}, err
	}
	return tracker.Snapshot(executor.time.Now())
}

func (executor *OperationalExecutor) Execute(ctx context.Context, operation Operation, call ProviderCall) (ExecutionResult, error) {
	started := executor.time.Now()
	result := ExecutionResult{Status: ExecutionFailed, StartedAt: started, Identities: operationalIdentities(executor.policy)}
	if call == nil {
		return executor.fail(result, operation, FailureUnknown, ErrorInvalidOperation, 0, "provider_call_required", nil, nil)
	}
	definition, capability, tracker, err := executor.operationState(operation)
	if err != nil {
		var operational *OperationalError
		if errors.As(err, &operational) {
			return executor.fail(result, operation, operational.FailureClass, operational.Code, operational.Attempt, operational.ReasonCode, operational.Cause, nil)
		}
		return executor.fail(result, operation, FailureUnknown, ErrorInvalidOperation, 0, "operation_validation_failed", err, nil)
	}
	_ = definition
	if capability.Support != SupportSupported {
		return executor.fail(result, operation, FailureUnsupportedCapability, ErrorStaticCapabilityUnavailable, 0, "static_capability_not_supported", nil, tracker)
	}
	if err := contextFailure(ctx); err != nil {
		class, code := contextFailureClass(err)
		return executor.fail(result, operation, class, code, 0, "context_stopped_before_attempt", err, tracker)
	}
	if !validOperationalTime(started) {
		return executor.fail(result, operation, FailureUnknown, ErrorInvalidPolicy, 0, "time_source_not_utc", nil, tracker)
	}

	for result.Attempts < executor.policy.Retry.MaximumAttempts {
		now := executor.time.Now()
		if now.Sub(started) >= executor.policy.Retry.MaximumElapsed && result.Attempts > 0 {
			return executor.fail(result, operation, result.FailureClass, ErrorElapsedBudgetExhausted, result.Attempts, "maximum_elapsed_reached", nil, tracker)
		}
		if err := contextFailure(ctx); err != nil {
			class, code := contextFailureClass(err)
			return executor.fail(result, operation, class, code, result.Attempts, "context_stopped_before_attempt", err, tracker)
		}

		rateDecision, err := executor.limiter.Acquire(operation, now)
		if err != nil {
			return executor.fail(result, operation, FailureUnknown, ErrorInvalidOperation, result.Attempts, "rate_limit_evaluation_failed", err, tracker)
		}
		if !rateDecision.Allowed {
			executor.emit(operation, OperationalEvent{Kind: OperationalEventThrottled, ObservedAt: now, Attempt: result.Attempts + 1, RateLimitReason: rateDecision.Reason})
			return executor.fail(result, operation, FailureRateLimited, ErrorLocalThrottled, result.Attempts, string(rateDecision.Reason), nil, tracker)
		}

		result.Attempts++
		attempt := result.Attempts
		attemptStarted := executor.time.Now()
		attemptTimeout := executor.attemptTimeout(started, attemptStarted)
		attemptCtx, cancel := executor.time.WithTimeout(ctx, attemptTimeout)
		executor.emit(operation, OperationalEvent{Kind: OperationalEventAttempt, ObservedAt: attemptStarted, Attempt: attempt})
		attemptResult := call(attemptCtx, AttemptContext{Operation: operation, Attempt: attempt, StartedAt: attemptStarted, Deadline: attemptStarted.Add(attemptTimeout)})
		attemptContextError := attemptCtx.Err()
		cancel()
		executor.limiter.Release(operation)
		completed := executor.time.Now()

		if ctx.Err() != nil {
			class, code := contextFailureClass(ctx.Err())
			return executor.fail(result, operation, class, code, attempt, "caller_context_stopped", ctx.Err(), tracker)
		}
		if errors.Is(attemptContextError, context.DeadlineExceeded) {
			attemptResult = ProviderAttemptResult{Failure: &ProviderFailure{Class: FailureAttemptDeadline, Cause: attemptContextError}}
		}
		if err := attemptResult.validate(); err != nil {
			return executor.fail(result, operation, FailureUnknown, ErrorInvalidAttemptResult, attempt, "invalid_provider_attempt_envelope", err, tracker)
		}
		if attemptResult.RateLimitObservation != nil {
			if err := executor.limiter.Observe(operation, completed, *attemptResult.RateLimitObservation); err != nil {
				return executor.fail(result, operation, FailureUnknown, ErrorInvalidAttemptResult, attempt, "invalid_rate_limit_observation", err, tracker)
			}
		}

		if attemptResult.Failure == nil {
			assessment, transition, err := tracker.ObserveSuccess(completed)
			if err != nil {
				return executor.fail(result, operation, FailureUnknown, ErrorInvalidAttemptResult, attempt, "health_observation_failed", err, tracker)
			}
			executor.emitHealthTransition(operation, transition)
			executor.emit(operation, OperationalEvent{Kind: OperationalEventSuccess, ObservedAt: completed, Attempt: attempt})
			result.Status = ExecutionSucceeded
			result.RawBytes = append([]byte(nil), attemptResult.RawBytes...)
			result.CompletedAt = completed
			result.FailureClass = FailureUnknown
			result.TerminalCode = ""
			result.Health = assessment
			return result, nil
		}

		class, err := classifyProviderFailure(executor.policy.Classification, *attemptResult.Failure)
		if err != nil {
			return executor.fail(result, operation, FailureUnknown, ErrorInvalidAttemptResult, attempt, "failure_classification_failed", err, tracker)
		}
		result.FailureClass = class
		assessment, transition, healthErr := tracker.ObserveFailure(completed, class)
		if healthErr != nil {
			return executor.fail(result, operation, class, ErrorInvalidAttemptResult, attempt, "health_observation_failed", healthErr, tracker)
		}
		result.Health = assessment
		executor.emitHealthTransition(operation, transition)
		if class == FailureRateLimited {
			executor.emit(operation, OperationalEvent{Kind: OperationalEventRateLimitResponse, ObservedAt: completed, Attempt: attempt, FailureClass: class})
		}

		classification := ClassifyRetry(operation, executor.policy.Retry, class)
		if classification.Disposition == RetryDispositionCancelled {
			code := ErrorCancelled
			if class == FailureOverallDeadline {
				code = ErrorDeadlineExceeded
			}
			return executor.fail(result, operation, class, code, attempt, classification.ReasonCode, attemptResult.Failure.Cause, tracker)
		}
		if classification.Disposition == RetryDispositionNonRetryable {
			return executor.fail(result, operation, class, ErrorNonRetryableFailure, attempt, classification.ReasonCode, attemptResult.Failure.Cause, tracker)
		}
		if attempt >= executor.policy.Retry.MaximumAttempts {
			return executor.fail(result, operation, class, ErrorRetryBudgetExhausted, attempt, "maximum_attempts_reached", attemptResult.Failure.Cause, tracker)
		}

		decision, decisionErr := executor.retryDecision(ctx, operation, started, completed, attempt, class, attemptResult.Failure.RetryAfter)
		if decisionErr != nil {
			var operational *OperationalError
			if errors.As(decisionErr, &operational) {
				return executor.fail(result, operation, class, operational.Code, attempt, operational.ReasonCode, operational.Cause, tracker)
			}
			return executor.fail(result, operation, class, ErrorInvalidPolicy, attempt, "retry_decision_failed", decisionErr, tracker)
		}
		result.Retries = append(result.Retries, decision)
		beforeSleep := executor.time.Now()
		if err := executor.time.Sleep(ctx, decision.ScheduledDelay); err != nil {
			class, code := contextFailureClass(err)
			return executor.fail(result, operation, class, code, attempt, "context_stopped_during_backoff", err, tracker)
		}
		afterSleep := executor.time.Now()
		if decision.ScheduledDelay > 0 && afterSleep.Before(beforeSleep.Add(decision.ScheduledDelay)) {
			return executor.fail(result, operation, class, ErrorClockDidNotAdvance, attempt, "time_source_did_not_advance_through_backoff", nil, tracker)
		}
		if decision.RetryAfter.Present {
			result.Retries[len(result.Retries)-1].RetryAfter.Honoured = true
			decision.RetryAfter.Honoured = true
		}
		executor.emit(operation, OperationalEvent{
			Kind: OperationalEventRetryScheduled, ObservedAt: afterSleep, Attempt: attempt,
			FailureClass: class, Disposition: classification.Disposition, Delay: decision.ScheduledDelay,
			RetryAfterKind: decision.RetryAfter.Kind, RetryAfterHonoured: decision.RetryAfter.Honoured,
		})
	}
	return executor.fail(result, operation, result.FailureClass, ErrorRetryBudgetExhausted, result.Attempts, "maximum_attempts_reached", nil, tracker)
}

func (executor *OperationalExecutor) retryDecision(ctx context.Context, operation Operation, started, now time.Time, attempt int, class FailureClass, retryAfterValue string) (RetryDecision, error) {
	backoff, err := CalculateBackoff(executor.policy.Retry.Backoff, attempt, executor.jitter)
	if err != nil {
		return RetryDecision{}, err
	}
	decision := RetryDecision{AfterAttempt: attempt, FailureClass: class, BackoffDelay: backoff, ScheduledDelay: backoff, ScheduledAt: now}
	if retryAfterValue != "" {
		delay, kind, err := ParseRetryAfter(retryAfterValue, now)
		if err != nil {
			return RetryDecision{}, operationalError(ErrorMalformedRetryAfter, operation, class, attempt, "provider_retry_after_malformed", err)
		}
		decision.RetryAfter = RetryAfterDecision{Present: true, Kind: kind, Delay: delay}
		if delay > executor.policy.RateLimit.MaximumProviderDelay {
			return RetryDecision{}, operationalError(ErrorExcessiveRetryAfter, operation, class, attempt, "provider_retry_after_exceeds_policy", nil)
		}
		if delay > decision.ScheduledDelay {
			decision.ScheduledDelay = delay
		}
		if err := executor.limiter.ObserveExhaustedUntil(operation, now, now.Add(delay)); err != nil {
			return RetryDecision{}, err
		}
	}
	local, err := executor.limiter.MayAttempt(operation, now)
	if err != nil {
		return RetryDecision{}, err
	}
	if !local.Allowed && local.RetryAt != nil && local.RetryAt.After(now) {
		decision.LocalDelay = local.RetryAt.Sub(now)
		if decision.LocalDelay > decision.ScheduledDelay {
			decision.ScheduledDelay = decision.LocalDelay
		}
	}
	decision.RetryAt = now.Add(decision.ScheduledDelay)
	if decision.RetryAt.Sub(started) > executor.policy.Retry.MaximumElapsed {
		code := ErrorElapsedBudgetExhausted
		reason := "scheduled_retry_exceeds_elapsed_budget"
		if decision.RetryAfter.Present {
			code, reason = ErrorUnhonourableRetryAfter, "provider_retry_after_exceeds_remaining_budget"
		}
		return RetryDecision{}, operationalError(code, operation, class, attempt, reason, nil)
	}
	if deadline, ok := ctx.Deadline(); ok && !deadline.After(decision.RetryAt) {
		code := ErrorDeadlineExceeded
		reason := "scheduled_retry_reaches_caller_deadline"
		if decision.RetryAfter.Present {
			code, reason = ErrorUnhonourableRetryAfter, "provider_retry_after_reaches_caller_deadline"
		}
		return RetryDecision{}, operationalError(code, operation, class, attempt, reason, context.DeadlineExceeded)
	}
	return decision, nil
}

func (executor *OperationalExecutor) attemptTimeout(started, now time.Time) time.Duration {
	remaining := executor.policy.Retry.MaximumElapsed - now.Sub(started)
	if remaining < executor.policy.Retry.PerAttemptTimeout {
		return remaining
	}
	return executor.policy.Retry.PerAttemptTimeout
}

func (executor *OperationalExecutor) validateOperation(ctx context.Context, operation Operation) error {
	if err := operation.Validate(); err != nil {
		return err
	}
	if err := contextFailure(ctx); err != nil {
		class, code := contextFailureClass(err)
		return operationalError(code, operation, class, 0, "caller_context_stopped", err)
	}
	_, capability, _, err := executor.operationState(operation)
	if err != nil {
		return err
	}
	if capability.Support != SupportSupported {
		return operationalError(ErrorStaticCapabilityUnavailable, operation, FailureUnsupportedCapability, 0, "static_capability_not_supported", nil)
	}
	return nil
}

func (executor *OperationalExecutor) operationState(operation Operation) (ProviderDefinition, Capability, *HealthTracker, error) {
	if err := operation.Validate(); err != nil {
		return ProviderDefinition{}, Capability{}, nil, err
	}
	definition, err := executor.providers.Lookup(operation.Provider)
	if err != nil {
		return ProviderDefinition{}, Capability{}, nil, operationalError(ErrorUnsupportedProvider, operation, FailureUnknown, 0, "provider_not_registered", err)
	}
	capability, err := executor.providers.Capability(operation.Provider, operation.CapabilityID)
	if err != nil {
		return ProviderDefinition{}, Capability{}, nil, operationalError(ErrorUnsupportedCapability, operation, FailureUnsupportedCapability, 0, "capability_not_registered", err)
	}
	tracker, err := executor.healthTracker(definition, operation.CapabilityID)
	if err != nil {
		return ProviderDefinition{}, Capability{}, nil, err
	}
	return definition, capability, tracker, nil
}

func (executor *OperationalExecutor) healthTracker(definition ProviderDefinition, capability CapabilityID) (*HealthTracker, error) {
	key := definition.Identity.ID + "\x00" + string(capability)
	executor.healthMu.Lock()
	defer executor.healthMu.Unlock()
	if tracker := executor.health[key]; tracker != nil {
		return tracker, nil
	}
	tracker, err := NewHealthTracker(executor.policy.Health, executor.policy.Component, definition, capability)
	if err != nil {
		return nil, err
	}
	executor.health[key] = tracker
	return tracker, nil
}

func (executor *OperationalExecutor) fail(result ExecutionResult, operation Operation, class FailureClass, code OperationalErrorCode, attempt int, reason string, cause error, tracker *HealthTracker) (ExecutionResult, error) {
	completed := executor.time.Now()
	result.Status = ExecutionFailed
	result.RawBytes = nil
	result.CompletedAt = completed
	result.FailureClass = class
	result.TerminalCode = code
	if tracker != nil {
		if assessment, err := tracker.Snapshot(completed); err == nil {
			result.Health = assessment
		}
	}
	executor.emit(operation, OperationalEvent{Kind: OperationalEventTerminalFailure, ObservedAt: completed, Attempt: attempt, FailureClass: class, TerminalCode: code})
	return result, operationalError(code, operation, class, attempt, reason, cause)
}

func (executor *OperationalExecutor) emit(operation Operation, event OperationalEvent) {
	event.Provider = cloneProviderIdentity(operation.Provider)
	event.CapabilityID = operation.CapabilityID
	event.OperationKind = operation.Kind
	event.Identities = operationalIdentities(executor.policy)
	if event.Validate() == nil {
		executor.sink.Record(event)
	}
}

func (executor *OperationalExecutor) emitHealthTransition(operation Operation, transition *HealthTransition) {
	if transition == nil {
		return
	}
	executor.emit(operation, OperationalEvent{
		Kind: OperationalEventHealthTransition, ObservedAt: transition.At,
		HealthFrom: transition.From, HealthTo: transition.To, HealthReason: transition.ReasonCode,
	})
}

func contextFailure(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}

func contextFailureClass(err error) (FailureClass, OperationalErrorCode) {
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureOverallDeadline, ErrorDeadlineExceeded
	}
	return FailureCallerCancelled, ErrorCancelled
}

func cloneOperationalPolicy(policy OperationalPolicy) OperationalPolicy {
	copyPolicy := policy
	copyPolicy.Classification.Identity = cloneComponentIdentity(policy.Classification.Identity)
	copyPolicy.Classification.HTTPOverrides = append([]HTTPStatusOverride(nil), policy.Classification.HTTPOverrides...)
	copyPolicy.Retry.Identity = cloneComponentIdentity(policy.Retry.Identity)
	copyPolicy.Retry.RetryableFailures = append([]FailureClass(nil), policy.Retry.RetryableFailures...)
	copyPolicy.RateLimit.Identity = cloneComponentIdentity(policy.RateLimit.Identity)
	copyPolicy.Health.Identity = cloneComponentIdentity(policy.Health.Identity)
	copyPolicy.Component = cloneComponentIdentity(policy.Component)
	return copyPolicy
}

var _ canonical.Contract = Operation{}
