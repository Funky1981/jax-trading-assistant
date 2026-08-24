package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	OperationContractV1            canonical.ContractVersion = "jax.provider_operation/v1"
	ClassificationPolicyContractV1 canonical.ContractVersion = "jax.provider_failure_classification_policy/v1"
	RetryPolicyContractV1          canonical.ContractVersion = "jax.provider_retry_policy/v1"
	RateLimitPolicyContractV1      canonical.ContractVersion = "jax.provider_rate_limit_policy/v1"
	HealthPolicyContractV1         canonical.ContractVersion = "jax.provider_health_policy/v1"
	OperationalPolicyContractV1    canonical.ContractVersion = "jax.provider_operational_policy/v1"
)

// OperationKind describes a bounded external-data acquisition operation. It
// deliberately excludes broker, order, trade, and recommendation operations.
type OperationKind string

const (
	OperationReadFetch       OperationKind = "READ_FETCH"
	OperationPaginatedRead   OperationKind = "PAGINATED_READ"
	OperationStreamEstablish OperationKind = "STREAM_ESTABLISH"
	OperationMetadataFetch   OperationKind = "METADATA_FETCH"
)

// RetrySafety makes repeatability an explicit caller assertion. A read-shaped
// operation is not silently assumed safe merely because it uses HTTP GET.
type RetrySafety string

const (
	RetrySafetyRepeatable    RetrySafety = "SAFE_REPEATABLE"
	RetrySafetySingleAttempt RetrySafety = "SINGLE_ATTEMPT"
)

// Operation identifies what is being attempted without carrying an endpoint,
// symbol, request ID, payload, or other unbounded/cardinality-sensitive value.
type Operation struct {
	ContractVersion canonical.ContractVersion  `json:"contract_version"`
	Provider        canonical.ProviderIdentity `json:"provider"`
	CapabilityID    CapabilityID               `json:"capability_id"`
	Kind            OperationKind              `json:"kind"`
	RetrySafety     RetrySafety                `json:"retry_safety"`
}

func (operation Operation) Validate() error {
	if operation.ContractVersion != OperationContractV1 {
		return operationalError(ErrorInvalidOperation, operation, FailureUnknown, 0, "unsupported_operation_version", nil)
	}
	if err := operation.Provider.Validate(); err != nil {
		return operationalError(ErrorInvalidOperation, operation, FailureUnknown, 0, "invalid_provider", err)
	}
	if _, _, ok := capabilitySpecification(operation.CapabilityID); !ok {
		return operationalError(ErrorInvalidOperation, operation, FailureUnsupportedCapability, 0, "unsupported_capability", nil)
	}
	switch operation.Kind {
	case OperationReadFetch, OperationPaginatedRead, OperationStreamEstablish, OperationMetadataFetch:
	default:
		return operationalError(ErrorInvalidOperation, operation, FailureUnknown, 0, "unsupported_operation_kind", nil)
	}
	switch operation.RetrySafety {
	case RetrySafetyRepeatable, RetrySafetySingleAttempt:
	default:
		return operationalError(ErrorInvalidOperation, operation, FailureUnknown, 0, "invalid_retry_safety", nil)
	}
	return nil
}

// FailureClass is a closed operational failure taxonomy. Data-quality and
// caller-control failures are intentionally distinguishable from upstream
// availability failures.
type FailureClass string

const (
	FailureUnknown               FailureClass = "UNKNOWN"
	FailureTransportTransient    FailureClass = "TRANSPORT_TRANSIENT"
	FailureProviderServer        FailureClass = "PROVIDER_SERVER_ERROR"
	FailureTemporaryUnavailable  FailureClass = "TEMPORARY_UNAVAILABLE"
	FailureRateLimited           FailureClass = "RATE_LIMITED"
	FailureAttemptDeadline       FailureClass = "ATTEMPT_DEADLINE"
	FailureAuthentication        FailureClass = "AUTHENTICATION"
	FailureAuthorization         FailureClass = "AUTHORIZATION"
	FailureMalformedRequest      FailureClass = "MALFORMED_REQUEST"
	FailureNotFound              FailureClass = "NOT_FOUND"
	FailureConflict              FailureClass = "CONFLICT"
	FailureUnsupportedCapability FailureClass = "UNSUPPORTED_CAPABILITY"
	FailureProviderPayloadParse  FailureClass = "PROVIDER_PAYLOAD_PARSE"
	FailureCanonicalValidation   FailureClass = "CANONICAL_VALIDATION"
	FailureProvenanceMismatch    FailureClass = "PROVENANCE_MISMATCH"
	FailurePermanentRejection    FailureClass = "PERMANENT_PROVIDER_REJECTION"
	FailureCallerCancelled       FailureClass = "CALLER_CANCELLED"
	FailureOverallDeadline       FailureClass = "OVERALL_DEADLINE"
)

func supportedFailureClass(class FailureClass) bool {
	switch class {
	case FailureTransportTransient, FailureProviderServer, FailureTemporaryUnavailable, FailureRateLimited,
		FailureAttemptDeadline, FailureAuthentication, FailureAuthorization, FailureMalformedRequest,
		FailureNotFound, FailureConflict, FailureUnsupportedCapability, FailureProviderPayloadParse,
		FailureCanonicalValidation, FailureProvenanceMismatch, FailurePermanentRejection,
		FailureCallerCancelled, FailureOverallDeadline:
		return true
	default:
		return false
	}
}

func potentiallyRetryableFailure(class FailureClass) bool {
	switch class {
	case FailureTransportTransient, FailureProviderServer, FailureTemporaryUnavailable, FailureRateLimited, FailureAttemptDeadline:
		return true
	default:
		return false
	}
}

type RetryDisposition string

const (
	RetryDispositionRetryable    RetryDisposition = "RETRYABLE_TRANSIENT"
	RetryDispositionNonRetryable RetryDisposition = "NON_RETRYABLE"
	RetryDispositionCancelled    RetryDisposition = "CANCELLED_DEADLINE"
)

type RetryClassification struct {
	FailureClass FailureClass     `json:"failure_class"`
	Disposition  RetryDisposition `json:"disposition"`
	ReasonCode   string           `json:"reason_code"`
}

// HTTPStatusOverride is an explicit, versioned exception to the base HTTP
// mapping. The policy selected for a provider adapter records the exception;
// no provider-name conditional is hidden in classification code.
type HTTPStatusOverride struct {
	Status int          `json:"status"`
	Class  FailureClass `json:"class"`
}

type ClassificationPolicy struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	Identity        canonical.ComponentIdentity `json:"identity"`
	HTTPOverrides   []HTTPStatusOverride        `json:"http_overrides,omitempty"`
}

func (policy ClassificationPolicy) Validate() error {
	if policy.ContractVersion != ClassificationPolicyContractV1 {
		return fmt.Errorf("classification policy: unsupported contract version")
	}
	if err := validatePolicyIdentity("classification", policy.Identity, "jax.policy.failure_classification"); err != nil {
		return err
	}
	seen := make(map[int]struct{}, len(policy.HTTPOverrides))
	for _, override := range policy.HTTPOverrides {
		if override.Status < 100 || override.Status > 599 || (override.Status >= 200 && override.Status <= 299) {
			return fmt.Errorf("classification policy: HTTP override status is invalid")
		}
		if !supportedFailureClass(override.Class) || override.Class == FailureCallerCancelled || override.Class == FailureOverallDeadline || override.Class == FailureAttemptDeadline {
			return fmt.Errorf("classification policy: HTTP override class is invalid")
		}
		if _, ok := seen[override.Status]; ok {
			return fmt.Errorf("classification policy: duplicate HTTP override")
		}
		seen[override.Status] = struct{}{}
	}
	return nil
}

type JitterStrategy string

const (
	JitterNone         JitterStrategy = "NONE"
	JitterProportional JitterStrategy = "PROPORTIONAL"
)

type BackoffPolicy struct {
	InitialDelay time.Duration  `json:"initial_delay"`
	Multiplier   float64        `json:"multiplier"`
	MaximumDelay time.Duration  `json:"maximum_delay"`
	Jitter       JitterStrategy `json:"jitter"`
	JitterRatio  float64        `json:"jitter_ratio"`
}

func (policy BackoffPolicy) Validate() error {
	if policy.InitialDelay <= 0 || policy.MaximumDelay <= 0 || policy.InitialDelay > policy.MaximumDelay {
		return fmt.Errorf("backoff policy: delays must be positive and initial must not exceed maximum")
	}
	if math.IsNaN(policy.Multiplier) || math.IsInf(policy.Multiplier, 0) || policy.Multiplier < 1 || policy.Multiplier > 10 {
		return fmt.Errorf("backoff policy: multiplier must be finite and between 1 and 10")
	}
	switch policy.Jitter {
	case JitterNone:
		if policy.JitterRatio != 0 {
			return fmt.Errorf("backoff policy: NONE jitter requires zero ratio")
		}
	case JitterProportional:
		if math.IsNaN(policy.JitterRatio) || math.IsInf(policy.JitterRatio, 0) || policy.JitterRatio <= 0 || policy.JitterRatio > 1 {
			return fmt.Errorf("backoff policy: proportional jitter ratio must be in (0,1]")
		}
	default:
		return fmt.Errorf("backoff policy: unsupported jitter strategy")
	}
	return nil
}

type RetryPolicy struct {
	ContractVersion   canonical.ContractVersion   `json:"contract_version"`
	Identity          canonical.ComponentIdentity `json:"identity"`
	MaximumAttempts   int                         `json:"maximum_attempts"`
	MaximumElapsed    time.Duration               `json:"maximum_elapsed"`
	PerAttemptTimeout time.Duration               `json:"per_attempt_timeout"`
	RetryableFailures []FailureClass              `json:"retryable_failures"`
	Backoff           BackoffPolicy               `json:"backoff"`
}

func (policy RetryPolicy) Validate() error {
	if policy.ContractVersion != RetryPolicyContractV1 {
		return fmt.Errorf("retry policy: unsupported contract version")
	}
	if err := validatePolicyIdentity("retry", policy.Identity, "jax.policy.retry"); err != nil {
		return err
	}
	if policy.MaximumAttempts < 1 || policy.MaximumAttempts > 20 {
		return fmt.Errorf("retry policy: maximum attempts must be between 1 and 20")
	}
	if policy.MaximumElapsed <= 0 || policy.MaximumElapsed > 24*time.Hour {
		return fmt.Errorf("retry policy: maximum elapsed must be in (0,24h]")
	}
	if policy.PerAttemptTimeout <= 0 || policy.PerAttemptTimeout > policy.MaximumElapsed {
		return fmt.Errorf("retry policy: per-attempt timeout must be positive and not exceed maximum elapsed")
	}
	if err := policy.Backoff.Validate(); err != nil {
		return err
	}
	if policy.Backoff.MaximumDelay > policy.MaximumElapsed {
		return fmt.Errorf("retry policy: maximum backoff exceeds maximum elapsed")
	}
	if len(policy.RetryableFailures) == 0 {
		return fmt.Errorf("retry policy: requires explicit retryable failures")
	}
	seen := make(map[FailureClass]struct{}, len(policy.RetryableFailures))
	for _, class := range policy.RetryableFailures {
		if !potentiallyRetryableFailure(class) {
			return fmt.Errorf("retry policy: failure class %q cannot be retryable", class)
		}
		if _, ok := seen[class]; ok {
			return fmt.Errorf("retry policy: duplicate retryable failure class")
		}
		seen[class] = struct{}{}
	}
	return nil
}

func (policy RetryPolicy) retries(class FailureClass) bool {
	for _, candidate := range policy.RetryableFailures {
		if candidate == class {
			return true
		}
	}
	return false
}

// OperationalPolicy binds the exact policies and Jax software component that
// produced an acquisition decision. It contains no vendor quota defaults.
type OperationalPolicy struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	Classification  ClassificationPolicy        `json:"classification"`
	Retry           RetryPolicy                 `json:"retry"`
	RateLimit       RateLimitPolicy             `json:"rate_limit"`
	Health          HealthPolicy                `json:"health"`
	Component       canonical.ComponentIdentity `json:"component"`
}

func (policy OperationalPolicy) Validate() error {
	if policy.ContractVersion != OperationalPolicyContractV1 {
		return fmt.Errorf("operational policy: unsupported contract version")
	}
	if err := policy.Classification.Validate(); err != nil {
		return err
	}
	if err := policy.Retry.Validate(); err != nil {
		return err
	}
	if err := policy.RateLimit.Validate(); err != nil {
		return err
	}
	if err := policy.Health.Validate(); err != nil {
		return err
	}
	if err := policy.Component.Validate(); err != nil || policy.Component.Kind != canonical.ComponentKindSoftwareBuild {
		return fmt.Errorf("operational policy: component must be a valid software_build identity")
	}
	ids := []string{policy.Classification.Identity.ID, policy.Retry.Identity.ID, policy.RateLimit.Identity.ID, policy.Health.Identity.ID, policy.Component.ID}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("operational policy: component identities must be distinct")
		}
		seen[id] = struct{}{}
	}
	if policy.RateLimit.MaximumProviderDelay > policy.Retry.MaximumElapsed {
		return fmt.Errorf("operational policy: provider delay maximum exceeds retry elapsed budget")
	}
	return nil
}

func validatePolicyIdentity(label string, identity canonical.ComponentIdentity, namespace string) error {
	if err := identity.Validate(); err != nil {
		return fmt.Errorf("%s policy: invalid identity", label)
	}
	if identity.Kind != canonical.ComponentKindPolicy || identity.Version.Namespace != namespace || identity.Provider != nil {
		return fmt.Errorf("%s policy: identity must be an unbound policy with version namespace %q", label, namespace)
	}
	return nil
}

// ProviderFailure is the adapter-to-operational-boundary failure envelope. An
// HTTP failure supplies status and lets the selected classification policy
// decide its class. A non-HTTP failure supplies an explicit class. Cause is
// unwrap-able but never rendered into instrumentation or Error text.
type ProviderFailure struct {
	Class      FailureClass `json:"class,omitempty"`
	HTTPStatus int          `json:"http_status,omitempty"`
	RetryAfter string       `json:"retry_after,omitempty"`
	Cause      error        `json:"-"`
}

func (failure ProviderFailure) Validate() error {
	if failure.HTTPStatus != 0 {
		if failure.HTTPStatus < 100 || failure.HTTPStatus > 599 || (failure.HTTPStatus >= 200 && failure.HTTPStatus <= 299) || (failure.Class != "" && failure.Class != FailureUnknown) {
			return fmt.Errorf("provider failure: invalid HTTP failure")
		}
	} else if !supportedFailureClass(failure.Class) || failure.Class == FailureUnknown {
		return fmt.Errorf("provider failure: explicit non-HTTP class is required")
	}
	if len(failure.RetryAfter) > 128 || strings.ContainsAny(failure.RetryAfter, "\r\n") {
		return fmt.Errorf("provider failure: invalid Retry-After value")
	}
	return nil
}

func (failure ProviderFailure) Unwrap() error { return failure.Cause }

type ProviderAttemptResult struct {
	RawBytes             []byte                        `json:"-"`
	Failure              *ProviderFailure              `json:"failure,omitempty"`
	RateLimitObservation *ProviderRateLimitObservation `json:"rate_limit_observation,omitempty"`
}

func (result ProviderAttemptResult) validate() error {
	if result.Failure == nil {
		if len(result.RawBytes) == 0 {
			return fmt.Errorf("provider attempt: successful acquisition requires exact non-empty raw bytes")
		}
	} else {
		if len(result.RawBytes) != 0 {
			return fmt.Errorf("provider attempt: failure must not return successful raw bytes")
		}
		if err := result.Failure.Validate(); err != nil {
			return err
		}
	}
	if result.RateLimitObservation != nil {
		if err := result.RateLimitObservation.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ClassifyHTTPStatus applies explicit overrides before the deliberately narrow
// V1 base mapping. Unlisted 4xx and 5xx responses are permanent rejections;
// the classifier does not retry whole status families.
func ClassifyHTTPStatus(policy ClassificationPolicy, status int) (FailureClass, error) {
	if err := policy.Validate(); err != nil {
		return FailureUnknown, err
	}
	if status < 100 || status > 599 || (status >= 200 && status <= 299) {
		return FailureUnknown, fmt.Errorf("HTTP classification: status is not a failure")
	}
	for _, override := range policy.HTTPOverrides {
		if override.Status == status {
			return override.Class, nil
		}
	}
	switch status {
	case http.StatusBadRequest:
		return FailureMalformedRequest, nil
	case http.StatusUnauthorized:
		return FailureAuthentication, nil
	case http.StatusForbidden:
		return FailureAuthorization, nil
	case http.StatusNotFound:
		return FailureNotFound, nil
	case http.StatusRequestTimeout:
		return FailureTransportTransient, nil
	case http.StatusConflict:
		return FailureConflict, nil
	case http.StatusTooEarly:
		return FailureTemporaryUnavailable, nil
	case http.StatusTooManyRequests:
		return FailureRateLimited, nil
	case http.StatusInternalServerError, http.StatusBadGateway:
		return FailureProviderServer, nil
	case http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return FailureTemporaryUnavailable, nil
	case http.StatusNotImplemented:
		return FailureUnsupportedCapability, nil
	default:
		return FailurePermanentRejection, nil
	}
}

func classifyProviderFailure(policy ClassificationPolicy, failure ProviderFailure) (FailureClass, error) {
	if err := failure.Validate(); err != nil {
		return FailureUnknown, err
	}
	if failure.HTTPStatus != 0 {
		return ClassifyHTTPStatus(policy, failure.HTTPStatus)
	}
	return failure.Class, nil
}

// ClassifyTransportError is a bounded adapter helper. Unknown errors are not
// retried merely because they are errors.
func ClassifyTransportError(ctx context.Context, err error) ProviderFailure {
	var contextErr error
	if ctx != nil {
		contextErr = ctx.Err()
	}
	if errors.Is(contextErr, context.Canceled) || errors.Is(err, context.Canceled) {
		return ProviderFailure{Class: FailureCallerCancelled, Cause: err}
	}
	if errors.Is(contextErr, context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return ProviderFailure{Class: FailureOverallDeadline, Cause: err}
	}
	var networkError net.Error
	if errors.As(err, &networkError) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) {
		return ProviderFailure{Class: FailureTransportTransient, Cause: err}
	}
	return ProviderFailure{Class: FailurePermanentRejection, Cause: err}
}

func ClassifyRetry(operation Operation, policy RetryPolicy, class FailureClass) RetryClassification {
	if class == FailureCallerCancelled || class == FailureOverallDeadline {
		return RetryClassification{FailureClass: class, Disposition: RetryDispositionCancelled, ReasonCode: "caller_controlled_stop"}
	}
	if operation.RetrySafety != RetrySafetyRepeatable {
		return RetryClassification{FailureClass: class, Disposition: RetryDispositionNonRetryable, ReasonCode: "operation_not_repeatable"}
	}
	if policy.retries(class) {
		return RetryClassification{FailureClass: class, Disposition: RetryDispositionRetryable, ReasonCode: "policy_allows_retry"}
	}
	return RetryClassification{FailureClass: class, Disposition: RetryDispositionNonRetryable, ReasonCode: "policy_forbids_retry"}
}

type RetryAfterKind string

const (
	RetryAfterDeltaSeconds RetryAfterKind = "DELTA_SECONDS"
	RetryAfterHTTPDate     RetryAfterKind = "HTTP_DATE"
)

type RetryAfterDecision struct {
	Present  bool
	Kind     RetryAfterKind
	Delay    time.Duration
	Honoured bool
}

func ParseRetryAfter(value string, now time.Time) (time.Duration, RetryAfterKind, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n") {
		return 0, "", fmt.Errorf("malformed Retry-After")
	}
	if seconds, err := strconv.ParseUint(value, 10, 31); err == nil {
		return time.Duration(seconds) * time.Second, RetryAfterDeltaSeconds, nil
	}
	absolute, err := http.ParseTime(value)
	if err != nil {
		return 0, "", fmt.Errorf("malformed Retry-After")
	}
	delay := absolute.Sub(now)
	if delay < 0 {
		delay = 0
	}
	return delay, RetryAfterHTTPDate, nil
}

type JitterSource interface {
	Float64() float64
}

// CalculateBackoff returns the bounded delay before retryOrdinal, where one is
// the first retry after the initial attempt.
func CalculateBackoff(policy BackoffPolicy, retryOrdinal int, jitter JitterSource) (time.Duration, error) {
	if err := policy.Validate(); err != nil {
		return 0, err
	}
	if retryOrdinal < 1 {
		return 0, fmt.Errorf("backoff: retry ordinal must be positive")
	}
	delay := float64(policy.InitialDelay)
	maximum := float64(policy.MaximumDelay)
	for ordinal := 1; ordinal < retryOrdinal && delay < maximum; ordinal++ {
		delay *= policy.Multiplier
		if math.IsInf(delay, 0) || delay > maximum {
			delay = maximum
		}
	}
	if policy.Jitter == JitterProportional {
		if jitter == nil {
			return 0, fmt.Errorf("backoff: proportional jitter requires a source")
		}
		fraction := jitter.Float64()
		if math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction < 0 || fraction > 1 {
			return 0, fmt.Errorf("backoff: jitter source must return [0,1]")
		}
		factor := (1 - policy.JitterRatio) + (2 * policy.JitterRatio * fraction)
		delay *= factor
	}
	if delay < 0 {
		delay = 0
	}
	if delay > maximum {
		delay = maximum
	}
	return time.Duration(delay), nil
}

type OperationalErrorCode string

const (
	ErrorInvalidOperation            OperationalErrorCode = "invalid_operation"
	ErrorInvalidPolicy               OperationalErrorCode = "invalid_policy"
	ErrorUnsupportedProvider         OperationalErrorCode = "unsupported_provider"
	ErrorUnsupportedCapability       OperationalErrorCode = "unsupported_capability"
	ErrorStaticCapabilityUnavailable OperationalErrorCode = "static_capability_unavailable"
	ErrorLocalThrottled              OperationalErrorCode = "local_throttled"
	ErrorInvalidAttemptResult        OperationalErrorCode = "invalid_attempt_result"
	ErrorNonRetryableFailure         OperationalErrorCode = "non_retryable_failure"
	ErrorRetryBudgetExhausted        OperationalErrorCode = "retry_budget_exhausted"
	ErrorElapsedBudgetExhausted      OperationalErrorCode = "elapsed_budget_exhausted"
	ErrorMalformedRetryAfter         OperationalErrorCode = "malformed_retry_after"
	ErrorExcessiveRetryAfter         OperationalErrorCode = "retry_after_exceeds_maximum"
	ErrorUnhonourableRetryAfter      OperationalErrorCode = "retry_after_cannot_be_honoured"
	ErrorCancelled                   OperationalErrorCode = "cancelled"
	ErrorDeadlineExceeded            OperationalErrorCode = "deadline_exceeded"
	ErrorClockDidNotAdvance          OperationalErrorCode = "clock_did_not_advance"
)

func supportedOperationalErrorCode(code OperationalErrorCode) bool {
	switch code {
	case ErrorInvalidOperation, ErrorInvalidPolicy, ErrorUnsupportedProvider, ErrorUnsupportedCapability,
		ErrorStaticCapabilityUnavailable, ErrorLocalThrottled, ErrorInvalidAttemptResult,
		ErrorNonRetryableFailure, ErrorRetryBudgetExhausted, ErrorElapsedBudgetExhausted,
		ErrorMalformedRetryAfter, ErrorExcessiveRetryAfter, ErrorUnhonourableRetryAfter,
		ErrorCancelled, ErrorDeadlineExceeded, ErrorClockDidNotAdvance:
		return true
	default:
		return false
	}
}

type OperationalError struct {
	Code         OperationalErrorCode `json:"code"`
	ProviderID   string               `json:"provider_id"`
	CapabilityID CapabilityID         `json:"capability_id"`
	FailureClass FailureClass         `json:"failure_class"`
	Attempt      int                  `json:"attempt"`
	ReasonCode   string               `json:"reason_code"`
	Cause        error                `json:"-"`
}

func (err *OperationalError) Error() string {
	return fmt.Sprintf("provider operation %s: provider=%q capability=%q class=%q attempt=%d reason=%q", err.Code, err.ProviderID, err.CapabilityID, err.FailureClass, err.Attempt, err.ReasonCode)
}

func (err *OperationalError) Unwrap() error { return err.Cause }

func operationalError(code OperationalErrorCode, operation Operation, class FailureClass, attempt int, reason string, cause error) error {
	return &OperationalError{Code: code, ProviderID: operation.Provider.ID, CapabilityID: operation.CapabilityID, FailureClass: class, Attempt: attempt, ReasonCode: reason, Cause: cause}
}
