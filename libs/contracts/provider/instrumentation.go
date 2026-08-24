package provider

import (
	"fmt"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type OperationalEventKind string

const (
	OperationalEventAttempt           OperationalEventKind = "ATTEMPT"
	OperationalEventRetryScheduled    OperationalEventKind = "RETRY_SCHEDULED"
	OperationalEventThrottled         OperationalEventKind = "THROTTLED"
	OperationalEventRateLimitResponse OperationalEventKind = "RATE_LIMIT_RESPONSE"
	OperationalEventSuccess           OperationalEventKind = "SUCCESS"
	OperationalEventTerminalFailure   OperationalEventKind = "TERMINAL_FAILURE"
	OperationalEventHealthTransition  OperationalEventKind = "HEALTH_TRANSITION"
)

type OperationalIdentitySet struct {
	ClassificationPolicy canonical.ComponentIdentity `json:"classification_policy"`
	RetryPolicy          canonical.ComponentIdentity `json:"retry_policy"`
	RateLimitPolicy      canonical.ComponentIdentity `json:"rate_limit_policy"`
	HealthPolicy         canonical.ComponentIdentity `json:"health_policy"`
	Component            canonical.ComponentIdentity `json:"component"`
}

// OperationalEvent has only bounded dimensions. It cannot carry a URL,
// symbol, raw payload, request ID, arbitrary error string, headers, or secret.
type OperationalEvent struct {
	Kind               OperationalEventKind       `json:"kind"`
	Provider           canonical.ProviderIdentity `json:"provider"`
	CapabilityID       CapabilityID               `json:"capability_id"`
	OperationKind      OperationKind              `json:"operation_kind"`
	ObservedAt         time.Time                  `json:"observed_at"`
	Attempt            int                        `json:"attempt,omitempty"`
	FailureClass       FailureClass               `json:"failure_class,omitempty"`
	Disposition        RetryDisposition           `json:"disposition,omitempty"`
	Delay              time.Duration              `json:"delay,omitempty"`
	RateLimitReason    RateLimitReason            `json:"rate_limit_reason,omitempty"`
	RetryAfterKind     RetryAfterKind             `json:"retry_after_kind,omitempty"`
	RetryAfterHonoured bool                       `json:"retry_after_honoured,omitempty"`
	HealthFrom         RuntimeStatus              `json:"health_from,omitempty"`
	HealthTo           RuntimeStatus              `json:"health_to,omitempty"`
	HealthReason       HealthReasonCode           `json:"health_reason,omitempty"`
	TerminalCode       OperationalErrorCode       `json:"terminal_code,omitempty"`
	Identities         OperationalIdentitySet     `json:"identities"`
}

func (event OperationalEvent) Validate() error {
	switch event.Kind {
	case OperationalEventAttempt, OperationalEventRetryScheduled, OperationalEventThrottled,
		OperationalEventRateLimitResponse, OperationalEventSuccess, OperationalEventTerminalFailure,
		OperationalEventHealthTransition:
	default:
		return fmt.Errorf("operational event: unsupported kind")
	}
	if err := event.Provider.Validate(); err != nil {
		return fmt.Errorf("operational event: invalid provider")
	}
	if _, _, ok := capabilitySpecification(event.CapabilityID); !ok {
		return fmt.Errorf("operational event: unsupported capability")
	}
	switch event.OperationKind {
	case OperationReadFetch, OperationPaginatedRead, OperationStreamEstablish, OperationMetadataFetch:
	default:
		return fmt.Errorf("operational event: unsupported operation kind")
	}
	if !validOperationalTime(event.ObservedAt) {
		return fmt.Errorf("operational event: observed_at must use UTC")
	}
	if event.Attempt < 0 || event.Delay < 0 {
		return fmt.Errorf("operational event: attempt and delay cannot be negative")
	}
	if event.FailureClass != "" && event.FailureClass != FailureUnknown && !supportedFailureClass(event.FailureClass) {
		return fmt.Errorf("operational event: unsupported failure class")
	}
	if event.Disposition != "" {
		switch event.Disposition {
		case RetryDispositionRetryable, RetryDispositionNonRetryable, RetryDispositionCancelled:
		default:
			return fmt.Errorf("operational event: unsupported retry disposition")
		}
	}
	if event.RateLimitReason != "" {
		switch event.RateLimitReason {
		case RateLimitAllowed, RateLimitLocalWindowExhausted, RateLimitConcurrencyExhausted, RateLimitProviderReportedExhausted:
		default:
			return fmt.Errorf("operational event: unsupported rate-limit reason")
		}
	}
	if event.RetryAfterKind != "" && event.RetryAfterKind != RetryAfterDeltaSeconds && event.RetryAfterKind != RetryAfterHTTPDate {
		return fmt.Errorf("operational event: unsupported Retry-After kind")
	}
	if event.Kind == OperationalEventHealthTransition {
		if !supportedRuntimeStatus(event.HealthFrom) || !supportedRuntimeStatus(event.HealthTo) || event.HealthFrom == event.HealthTo {
			return fmt.Errorf("operational event: invalid health transition")
		}
		if event.HealthReason != "" && !supportedHealthReason(event.HealthReason) {
			return fmt.Errorf("operational event: unsupported health reason")
		}
	}
	if event.TerminalCode != "" && !supportedOperationalErrorCode(event.TerminalCode) {
		return fmt.Errorf("operational event: unsupported terminal code")
	}
	if err := event.Identities.Validate(); err != nil {
		return err
	}
	return nil
}

func supportedRuntimeStatus(status RuntimeStatus) bool {
	switch status {
	case RuntimeUnknown, RuntimeHealthy, RuntimeDegraded, RuntimeUnavailable, RuntimeDisabled:
		return true
	default:
		return false
	}
}

func (identities OperationalIdentitySet) Validate() error {
	if err := validatePolicyIdentity("classification", identities.ClassificationPolicy, "jax.policy.failure_classification"); err != nil {
		return err
	}
	if err := validatePolicyIdentity("retry", identities.RetryPolicy, "jax.policy.retry"); err != nil {
		return err
	}
	if err := validatePolicyIdentity("rate-limit", identities.RateLimitPolicy, "jax.policy.rate_limit"); err != nil {
		return err
	}
	if err := validatePolicyIdentity("health", identities.HealthPolicy, "jax.policy.health"); err != nil {
		return err
	}
	if err := identities.Component.Validate(); err != nil || identities.Component.Kind != canonical.ComponentKindSoftwareBuild {
		return fmt.Errorf("operational identities: invalid component")
	}
	return nil
}

type InstrumentationSink interface {
	Record(OperationalEvent)
}

type discardInstrumentation struct{}

func (discardInstrumentation) Record(OperationalEvent) {}

type MemoryInstrumentation struct {
	mu     sync.Mutex
	events []OperationalEvent
}

func NewMemoryInstrumentation() *MemoryInstrumentation { return &MemoryInstrumentation{} }

func (sink *MemoryInstrumentation) Record(event OperationalEvent) {
	if sink == nil {
		return
	}
	sink.mu.Lock()
	sink.events = append(sink.events, cloneOperationalEvent(event))
	sink.mu.Unlock()
}

func (sink *MemoryInstrumentation) Events() []OperationalEvent {
	if sink == nil {
		return nil
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	events := make([]OperationalEvent, len(sink.events))
	for i, event := range sink.events {
		events[i] = cloneOperationalEvent(event)
	}
	return events
}

func operationalIdentities(policy OperationalPolicy) OperationalIdentitySet {
	return OperationalIdentitySet{
		ClassificationPolicy: cloneComponentIdentity(policy.Classification.Identity),
		RetryPolicy:          cloneComponentIdentity(policy.Retry.Identity),
		RateLimitPolicy:      cloneComponentIdentity(policy.RateLimit.Identity),
		HealthPolicy:         cloneComponentIdentity(policy.Health.Identity),
		Component:            cloneComponentIdentity(policy.Component),
	}
}

func cloneOperationalEvent(event OperationalEvent) OperationalEvent {
	event.Provider = cloneProviderIdentity(event.Provider)
	event.Identities.ClassificationPolicy = cloneComponentIdentity(event.Identities.ClassificationPolicy)
	event.Identities.RetryPolicy = cloneComponentIdentity(event.Identities.RetryPolicy)
	event.Identities.RateLimitPolicy = cloneComponentIdentity(event.Identities.RateLimitPolicy)
	event.Identities.HealthPolicy = cloneComponentIdentity(event.Identities.HealthPolicy)
	event.Identities.Component = cloneComponentIdentity(event.Identities.Component)
	return event
}
