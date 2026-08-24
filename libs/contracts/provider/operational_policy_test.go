package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type fixedJitter float64

func (value fixedJitter) Float64() float64 { return float64(value) }

func syntheticOperationalPolicy() OperationalPolicy {
	policyIdentity := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "synthetic/v1"}}
	}
	return OperationalPolicy{
		ContractVersion: OperationalPolicyContractV1,
		Classification:  ClassificationPolicy{ContractVersion: ClassificationPolicyContractV1, Identity: policyIdentity("cmp_test_failure_classification", "synthetic failure classification", "jax.policy.failure_classification")},
		Retry: RetryPolicy{
			ContractVersion: RetryPolicyContractV1, Identity: policyIdentity("cmp_test_retry", "synthetic retry", "jax.policy.retry"),
			MaximumAttempts: 3, MaximumElapsed: 30 * time.Second, PerAttemptTimeout: 5 * time.Second,
			RetryableFailures: []FailureClass{FailureTransportTransient, FailureProviderServer, FailureTemporaryUnavailable, FailureRateLimited, FailureAttemptDeadline},
			Backoff:           BackoffPolicy{InitialDelay: time.Second, Multiplier: 2, MaximumDelay: 4 * time.Second, Jitter: JitterNone},
		},
		RateLimit: RateLimitPolicy{
			ContractVersion: RateLimitPolicyContractV1, Identity: policyIdentity("cmp_test_rate_limit", "synthetic rate limit", "jax.policy.rate_limit"),
			RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 2, MaximumProviderDelay: 10 * time.Second,
		},
		Health: HealthPolicy{
			ContractVersion: HealthPolicyContractV1, Identity: policyIdentity("cmp_test_health", "synthetic health", "jax.policy.health"),
			DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour,
		},
		Component: canonical.ComponentIdentity{ID: "cmp_test_operational_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "synthetic operational executor", Version: canonical.VersionIdentity{Namespace: "git.commit", Value: "synthetic-wp-02.05"}},
	}
}

func syntheticOperation(definition ProviderDefinition) Operation {
	return Operation{ContractVersion: OperationContractV1, Provider: definition.Identity, CapabilityID: definition.Capabilities[0].ID, Kind: OperationReadFetch, RetrySafety: RetrySafetyRepeatable}
}

func TestHTTPStatusClassificationIsExplicitAndNarrow(t *testing.T) {
	policy := syntheticOperationalPolicy().Classification
	cases := map[int]FailureClass{
		400: FailureMalformedRequest,
		401: FailureAuthentication,
		403: FailureAuthorization,
		404: FailureNotFound,
		408: FailureTransportTransient,
		409: FailureConflict,
		425: FailureTemporaryUnavailable,
		429: FailureRateLimited,
		500: FailureProviderServer,
		502: FailureProviderServer,
		503: FailureTemporaryUnavailable,
		504: FailureTemporaryUnavailable,
		501: FailureUnsupportedCapability,
		418: FailurePermanentRejection,
		505: FailurePermanentRejection,
	}
	for status, want := range cases {
		got, err := ClassifyHTTPStatus(policy, status)
		if err != nil || got != want {
			t.Fatalf("ClassifyHTTPStatus(%d) = %q, %v; want %q", status, got, err, want)
		}
	}
	if _, err := ClassifyHTTPStatus(policy, http.StatusOK); err == nil {
		t.Fatal("ClassifyHTTPStatus() accepted success status")
	}

	policy.HTTPOverrides = []HTTPStatusOverride{{Status: http.StatusConflict, Class: FailureTemporaryUnavailable}}
	got, err := ClassifyHTTPStatus(policy, http.StatusConflict)
	if err != nil || got != FailureTemporaryUnavailable {
		t.Fatalf("versioned override = %q, %v", got, err)
	}
}

func TestRetryClassificationRequiresSafeOperationAndExplicitClass(t *testing.T) {
	policy := syntheticOperationalPolicy().Retry
	definition := validDefinition(CapabilityCorporateFiling)
	operation := syntheticOperation(definition)
	if got := ClassifyRetry(operation, policy, FailureTransportTransient); got.Disposition != RetryDispositionRetryable {
		t.Fatalf("safe transient disposition = %q", got.Disposition)
	}
	operation.RetrySafety = RetrySafetySingleAttempt
	if got := ClassifyRetry(operation, policy, FailureTransportTransient); got.Disposition != RetryDispositionNonRetryable {
		t.Fatalf("single-attempt disposition = %q", got.Disposition)
	}
	operation.RetrySafety = RetrySafetyRepeatable
	for _, class := range []FailureClass{FailureAuthentication, FailureMalformedRequest, FailureProviderPayloadParse, FailureCanonicalValidation, FailureProvenanceMismatch, FailurePermanentRejection} {
		if got := ClassifyRetry(operation, policy, class); got.Disposition != RetryDispositionNonRetryable {
			t.Fatalf("class %q disposition = %q", class, got.Disposition)
		}
	}
	for _, class := range []FailureClass{FailureCallerCancelled, FailureOverallDeadline} {
		if got := ClassifyRetry(operation, policy, class); got.Disposition != RetryDispositionCancelled {
			t.Fatalf("class %q disposition = %q", class, got.Disposition)
		}
	}
}

func TestTransportClassificationDoesNotRetryUnknownErrors(t *testing.T) {
	if got := ClassifyTransportError(context.Background(), io.ErrUnexpectedEOF); got.Class != FailureTransportTransient {
		t.Fatalf("unexpected EOF class = %q", got.Class)
	}
	if got := ClassifyTransportError(context.Background(), errors.New("opaque failure")); got.Class != FailurePermanentRejection {
		t.Fatalf("unknown error class = %q", got.Class)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got := ClassifyTransportError(cancelled, cancelled.Err()); got.Class != FailureCallerCancelled {
		t.Fatalf("cancelled class = %q", got.Class)
	}
}

func TestBackoffIsBoundedAndDeterministic(t *testing.T) {
	policy := syntheticOperationalPolicy().Retry.Backoff
	for ordinal, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second} {
		got, err := CalculateBackoff(policy, ordinal+1, nil)
		if err != nil || got != want {
			t.Fatalf("CalculateBackoff(%d) = %s, %v; want %s", ordinal+1, got, err, want)
		}
	}
	jittered := BackoffPolicy{InitialDelay: 4 * time.Second, Multiplier: 2, MaximumDelay: 10 * time.Second, Jitter: JitterProportional, JitterRatio: 0.25}
	low, err := CalculateBackoff(jittered, 1, fixedJitter(0))
	if err != nil || low != 3*time.Second {
		t.Fatalf("low jitter = %s, %v", low, err)
	}
	high, err := CalculateBackoff(jittered, 1, fixedJitter(1))
	if err != nil || high != 5*time.Second {
		t.Fatalf("high jitter = %s, %v", high, err)
	}
	if _, err := CalculateBackoff(jittered, 1, fixedJitter(1.1)); err == nil {
		t.Fatal("CalculateBackoff() accepted out-of-range jitter")
	}
}

func TestRetryAfterSupportsSecondsAndAbsoluteTime(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	delay, kind, err := ParseRetryAfter("7", now)
	if err != nil || delay != 7*time.Second || kind != RetryAfterDeltaSeconds {
		t.Fatalf("delta Retry-After = %s, %q, %v", delay, kind, err)
	}
	delay, kind, err = ParseRetryAfter(now.Add(5*time.Second).Format(http.TimeFormat), now)
	if err != nil || delay != 5*time.Second || kind != RetryAfterHTTPDate {
		t.Fatalf("date Retry-After = %s, %q, %v", delay, kind, err)
	}
	delay, _, err = ParseRetryAfter(now.Add(-time.Minute).Format(http.TimeFormat), now)
	if err != nil || delay != 0 {
		t.Fatalf("past Retry-After = %s, %v", delay, err)
	}
	for _, malformed := range []string{"", "later", "-1", "1\r\nAuthorization: secret"} {
		if _, _, err := ParseRetryAfter(malformed, now); err == nil {
			t.Fatalf("ParseRetryAfter(%q) accepted malformed value", malformed)
		}
	}
}

func TestOperationalPoliciesRejectInvalidAndContradictoryParameters(t *testing.T) {
	base := syntheticOperationalPolicy()
	cases := []func(*OperationalPolicy){
		func(policy *OperationalPolicy) { policy.Retry.MaximumAttempts = 0 },
		func(policy *OperationalPolicy) { policy.Retry.MaximumElapsed = 0 },
		func(policy *OperationalPolicy) {
			policy.Retry.PerAttemptTimeout = policy.Retry.MaximumElapsed + time.Second
		},
		func(policy *OperationalPolicy) { policy.Retry.Backoff.InitialDelay = 0 },
		func(policy *OperationalPolicy) { policy.Retry.Backoff.Multiplier = 0.5 },
		func(policy *OperationalPolicy) {
			policy.Retry.RetryableFailures = append(policy.Retry.RetryableFailures, FailureAuthentication)
		},
		func(policy *OperationalPolicy) { policy.RateLimit.RequestLimit = 0 },
		func(policy *OperationalPolicy) { policy.RateLimit.Window = 0 },
		func(policy *OperationalPolicy) { policy.RateLimit.ConcurrencyLimit = 0 },
		func(policy *OperationalPolicy) { policy.Health.DegradedAfterFailures = 0 },
		func(policy *OperationalPolicy) {
			policy.Health.UnavailableAfterFailures = policy.Health.DegradedAfterFailures - 1
		},
		func(policy *OperationalPolicy) { policy.Health.AssessmentHorizon = 0 },
		func(policy *OperationalPolicy) {
			policy.Classification.HTTPOverrides = []HTTPStatusOverride{{Status: 409, Class: FailureConflict}, {Status: 409, Class: FailureTemporaryUnavailable}}
		},
		func(policy *OperationalPolicy) { policy.RateLimit.Identity.ID = policy.Retry.Identity.ID },
		func(policy *OperationalPolicy) {
			policy.RateLimit.MaximumProviderDelay = policy.Retry.MaximumElapsed + time.Second
		},
	}
	for index, mutate := range cases {
		policy := cloneOperationalPolicy(base)
		mutate(&policy)
		if err := policy.Validate(); err == nil {
			t.Fatalf("invalid policy case %d passed validation", index)
		}
	}
}
