package provider

import (
	"fmt"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

// RateLimitPolicy is static configuration. Values are supplied by an adopting
// adapter/configuration and are not vendor quotas embedded by this package.
type RateLimitPolicy struct {
	ContractVersion      canonical.ContractVersion   `json:"contract_version"`
	Identity             canonical.ComponentIdentity `json:"identity"`
	RequestLimit         uint64                      `json:"request_limit"`
	Window               time.Duration               `json:"window"`
	ConcurrencyLimit     uint64                      `json:"concurrency_limit"`
	MaximumProviderDelay time.Duration               `json:"maximum_provider_delay"`
}

func (policy RateLimitPolicy) Validate() error {
	if policy.ContractVersion != RateLimitPolicyContractV1 {
		return fmt.Errorf("rate-limit policy: unsupported contract version")
	}
	if err := validatePolicyIdentity("rate-limit", policy.Identity, "jax.policy.rate_limit"); err != nil {
		return err
	}
	if policy.RequestLimit == 0 || policy.RequestLimit > 1_000_000 {
		return fmt.Errorf("rate-limit policy: request limit must be between 1 and 1000000")
	}
	if policy.Window <= 0 || policy.Window > 24*time.Hour {
		return fmt.Errorf("rate-limit policy: window must be in (0,24h]")
	}
	if policy.ConcurrencyLimit == 0 || policy.ConcurrencyLimit > 10_000 {
		return fmt.Errorf("rate-limit policy: concurrency limit must be between 1 and 10000")
	}
	if policy.MaximumProviderDelay <= 0 || policy.MaximumProviderDelay > 24*time.Hour {
		return fmt.Errorf("rate-limit policy: maximum provider delay must be in (0,24h]")
	}
	return nil
}

type ProviderRateLimitObservation struct {
	Remaining *uint64    `json:"remaining,omitempty"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

func (observation ProviderRateLimitObservation) Validate() error {
	if observation.Remaining == nil && observation.ResetAt == nil {
		return fmt.Errorf("provider rate-limit observation: remaining or reset_at is required")
	}
	if observation.ResetAt != nil {
		_, offset := observation.ResetAt.Zone()
		if observation.ResetAt.IsZero() || offset != 0 || observation.ResetAt.Year() < 0 || observation.ResetAt.Year() > 9999 {
			return fmt.Errorf("provider rate-limit observation: reset_at must use UTC")
		}
	}
	return nil
}

type RateLimitReason string

const (
	RateLimitAllowed                   RateLimitReason = "allowed"
	RateLimitLocalWindowExhausted      RateLimitReason = "local_window_exhausted"
	RateLimitConcurrencyExhausted      RateLimitReason = "concurrency_exhausted"
	RateLimitProviderReportedExhausted RateLimitReason = "provider_reported_exhausted"
)

type RateLimitDecision struct {
	Allowed           bool                        `json:"allowed"`
	Reason            RateLimitReason             `json:"reason"`
	AssessedAt        time.Time                   `json:"assessed_at"`
	RetryAt           *time.Time                  `json:"retry_at,omitempty"`
	RequestsRemaining uint64                      `json:"requests_remaining"`
	ConcurrencyInUse  uint64                      `json:"concurrency_in_use"`
	Policy            canonical.ComponentIdentity `json:"policy"`
}

type rateLimitRuntimeState struct {
	windowStarted     time.Time
	windowReset       time.Time
	requests          uint64
	concurrency       uint64
	providerExhausted bool
	providerReset     *time.Time
	providerRemaining *uint64
}

// RateLimiter is a small process-local proof/state machine. The provider
// registry remains immutable and no counters are persisted.
type RateLimiter struct {
	policy RateLimitPolicy
	mu     sync.Mutex
	states map[string]*rateLimitRuntimeState
}

func NewRateLimiter(policy RateLimitPolicy) (*RateLimiter, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &RateLimiter{policy: policy, states: make(map[string]*rateLimitRuntimeState)}, nil
}

func (limiter *RateLimiter) MayAttempt(operation Operation, now time.Time) (RateLimitDecision, error) {
	if err := operation.Validate(); err != nil {
		return RateLimitDecision{}, err
	}
	if !validOperationalTime(now) {
		return RateLimitDecision{}, fmt.Errorf("rate limiter: assessment time must use UTC")
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	state := limiter.state(operation)
	limiter.advance(state, now)
	return limiter.decision(state, now), nil
}

func (limiter *RateLimiter) Acquire(operation Operation, now time.Time) (RateLimitDecision, error) {
	if err := operation.Validate(); err != nil {
		return RateLimitDecision{}, err
	}
	if !validOperationalTime(now) {
		return RateLimitDecision{}, fmt.Errorf("rate limiter: acquisition time must use UTC")
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	state := limiter.state(operation)
	limiter.advance(state, now)
	decision := limiter.decision(state, now)
	if !decision.Allowed {
		return decision, nil
	}
	if state.windowStarted.IsZero() {
		state.windowStarted = now
		state.windowReset = now.Add(limiter.policy.Window)
	}
	state.requests++
	state.concurrency++
	decision.RequestsRemaining = limiter.policy.RequestLimit - state.requests
	decision.ConcurrencyInUse = state.concurrency
	return decision, nil
}

func (limiter *RateLimiter) Release(operation Operation) {
	if limiter == nil {
		return
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if state := limiter.states[operationalStateKey(operation)]; state != nil && state.concurrency > 0 {
		state.concurrency--
	}
}

func (limiter *RateLimiter) Observe(operation Operation, now time.Time, observation ProviderRateLimitObservation) error {
	if err := operation.Validate(); err != nil {
		return err
	}
	if !validOperationalTime(now) {
		return fmt.Errorf("rate limiter: observation time must use UTC")
	}
	if err := observation.Validate(); err != nil {
		return err
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	state := limiter.state(operation)
	limiter.advance(state, now)
	state.providerRemaining = cloneUint64Pointer(observation.Remaining)
	state.providerReset = cloneTimePointer(observation.ResetAt)
	if observation.Remaining != nil {
		state.providerExhausted = *observation.Remaining == 0
	}
	if state.providerReset != nil && !state.providerReset.After(now) {
		state.providerExhausted = false
		state.providerReset = nil
	}
	return nil
}

func (limiter *RateLimiter) ObserveExhaustedUntil(operation Operation, now, retryAt time.Time) error {
	zero := uint64(0)
	return limiter.Observe(operation, now, ProviderRateLimitObservation{Remaining: &zero, ResetAt: &retryAt})
}

func (limiter *RateLimiter) state(operation Operation) *rateLimitRuntimeState {
	key := operationalStateKey(operation)
	state := limiter.states[key]
	if state == nil {
		state = &rateLimitRuntimeState{}
		limiter.states[key] = state
	}
	return state
}

func (limiter *RateLimiter) advance(state *rateLimitRuntimeState, now time.Time) {
	if !state.windowReset.IsZero() && !now.Before(state.windowReset) {
		state.windowStarted = time.Time{}
		state.windowReset = time.Time{}
		state.requests = 0
	}
	if state.providerExhausted && state.providerReset != nil && !now.Before(*state.providerReset) {
		state.providerExhausted = false
		state.providerReset = nil
		state.providerRemaining = nil
	}
}

func (limiter *RateLimiter) decision(state *rateLimitRuntimeState, now time.Time) RateLimitDecision {
	decision := RateLimitDecision{
		Allowed:           true,
		Reason:            RateLimitAllowed,
		AssessedAt:        now,
		RequestsRemaining: limiter.policy.RequestLimit - state.requests,
		ConcurrencyInUse:  state.concurrency,
		Policy:            cloneComponentIdentity(limiter.policy.Identity),
	}
	if state.providerExhausted {
		decision.Allowed = false
		decision.Reason = RateLimitProviderReportedExhausted
		decision.RetryAt = cloneTimePointer(state.providerReset)
		decision.RequestsRemaining = 0
		return decision
	}
	if state.requests >= limiter.policy.RequestLimit {
		decision.Allowed = false
		decision.Reason = RateLimitLocalWindowExhausted
		decision.RetryAt = cloneTimePointer(&state.windowReset)
		decision.RequestsRemaining = 0
		return decision
	}
	if state.concurrency >= limiter.policy.ConcurrencyLimit {
		decision.Allowed = false
		decision.Reason = RateLimitConcurrencyExhausted
		return decision
	}
	return decision
}

func operationalStateKey(operation Operation) string {
	return operation.Provider.ID + "\x00" + string(operation.CapabilityID)
}

func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func validOperationalTime(value time.Time) bool {
	_, offset := value.Zone()
	return !value.IsZero() && offset == 0 && value.Year() >= 0 && value.Year() <= 9999
}
