package provider

import (
	"fmt"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

// HealthPolicy contains synthetic/adopter-supplied thresholds. It assesses
// operational capability reliability only; it has no TTL, LKG, trust, ranking,
// licensing, or cost semantics.
type HealthPolicy struct {
	ContractVersion          canonical.ContractVersion   `json:"contract_version"`
	Identity                 canonical.ComponentIdentity `json:"identity"`
	DegradedAfterFailures    int                         `json:"degraded_after_failures"`
	UnavailableAfterFailures int                         `json:"unavailable_after_failures"`
	RecoverySuccesses        int                         `json:"recovery_successes"`
	AssessmentHorizon        time.Duration               `json:"assessment_horizon"`
	CountAttemptDeadline     bool                        `json:"count_attempt_deadline"`
}

func (policy HealthPolicy) Validate() error {
	if policy.ContractVersion != HealthPolicyContractV1 {
		return fmt.Errorf("health policy: unsupported contract version")
	}
	if err := validatePolicyIdentity("health", policy.Identity, "jax.policy.health"); err != nil {
		return err
	}
	if policy.DegradedAfterFailures < 1 || policy.DegradedAfterFailures > 1000 {
		return fmt.Errorf("health policy: degraded threshold must be between 1 and 1000")
	}
	if policy.UnavailableAfterFailures < policy.DegradedAfterFailures || policy.UnavailableAfterFailures > 1000 {
		return fmt.Errorf("health policy: unavailable threshold must be at least degraded threshold and at most 1000")
	}
	if policy.RecoverySuccesses < 1 || policy.RecoverySuccesses > 1000 {
		return fmt.Errorf("health policy: recovery threshold must be between 1 and 1000")
	}
	if policy.AssessmentHorizon <= 0 || policy.AssessmentHorizon > 30*24*time.Hour {
		return fmt.Errorf("health policy: assessment horizon must be in (0,30d]")
	}
	return nil
}

type HealthReasonCode string

const (
	HealthReasonInsufficientEvidence HealthReasonCode = "insufficient_operational_evidence"
	HealthReasonTransientFailures    HealthReasonCode = "transient_failure_threshold"
	HealthReasonUnavailableFailures  HealthReasonCode = "unavailable_failure_threshold"
	HealthReasonRateLimitExhausted   HealthReasonCode = "rate_limit_exhausted"
	HealthReasonAuthentication       HealthReasonCode = "authentication_or_authorization_failure"
	HealthReasonUnsupported          HealthReasonCode = "unsupported_capability"
	HealthReasonRecovering           HealthReasonCode = "recovery_threshold_pending"
	HealthReasonHorizonElapsed       HealthReasonCode = "assessment_horizon_elapsed"
	HealthReasonStaticUnavailable    HealthReasonCode = "static_capability_unavailable"
	HealthReasonStaticDisabled       HealthReasonCode = "static_capability_disabled"
)

func supportedHealthReason(reason HealthReasonCode) bool {
	switch reason {
	case HealthReasonInsufficientEvidence, HealthReasonTransientFailures, HealthReasonUnavailableFailures,
		HealthReasonRateLimitExhausted, HealthReasonAuthentication, HealthReasonUnsupported,
		HealthReasonRecovering, HealthReasonHorizonElapsed, HealthReasonStaticUnavailable, HealthReasonStaticDisabled:
		return true
	default:
		return false
	}
}

type HealthAssessment struct {
	Provider             canonical.ProviderIdentity  `json:"provider"`
	CapabilityID         CapabilityID                `json:"capability_id"`
	Status               RuntimeStatus               `json:"status"`
	ReasonCode           HealthReasonCode            `json:"reason_code,omitempty"`
	AssessedAt           time.Time                   `json:"assessed_at"`
	LastSuccessAt        *time.Time                  `json:"last_success_at,omitempty"`
	LastFailureAt        *time.Time                  `json:"last_failure_at,omitempty"`
	ConsecutiveFailures  int                         `json:"consecutive_failures"`
	ConsecutiveSuccesses int                         `json:"consecutive_successes"`
	Policy               canonical.ComponentIdentity `json:"policy"`
	Component            canonical.ComponentIdentity `json:"component"`
}

func (assessment HealthAssessment) Validate() error {
	if err := assessment.Provider.Validate(); err != nil {
		return fmt.Errorf("health assessment: invalid provider")
	}
	if _, _, ok := capabilitySpecification(assessment.CapabilityID); !ok {
		return fmt.Errorf("health assessment: unsupported capability")
	}
	switch assessment.Status {
	case RuntimeUnknown, RuntimeHealthy, RuntimeDegraded, RuntimeUnavailable, RuntimeDisabled:
	default:
		return fmt.Errorf("health assessment: unsupported status")
	}
	if !validOperationalTime(assessment.AssessedAt) {
		return fmt.Errorf("health assessment: assessed_at must use UTC")
	}
	if assessment.LastSuccessAt != nil && !validOperationalTime(*assessment.LastSuccessAt) {
		return fmt.Errorf("health assessment: last_success_at must use UTC")
	}
	if assessment.LastFailureAt != nil && !validOperationalTime(*assessment.LastFailureAt) {
		return fmt.Errorf("health assessment: last_failure_at must use UTC")
	}
	if assessment.ConsecutiveFailures < 0 || assessment.ConsecutiveSuccesses < 0 {
		return fmt.Errorf("health assessment: consecutive counters cannot be negative")
	}
	if assessment.Status == RuntimeHealthy && assessment.ReasonCode != "" {
		return fmt.Errorf("health assessment: healthy status has no failure reason")
	}
	if assessment.Status != RuntimeHealthy && assessment.ReasonCode == "" {
		return fmt.Errorf("health assessment: non-healthy status requires a bounded reason")
	}
	if assessment.ReasonCode != "" && !supportedHealthReason(assessment.ReasonCode) {
		return fmt.Errorf("health assessment: unsupported reason")
	}
	if err := validatePolicyIdentity("health", assessment.Policy, "jax.policy.health"); err != nil {
		return err
	}
	if err := assessment.Component.Validate(); err != nil || assessment.Component.Kind != canonical.ComponentKindSoftwareBuild {
		return fmt.Errorf("health assessment: invalid operational component")
	}
	return nil
}

// AsRuntimeState integrates with the accepted WP-02.01 aggregate attachment.
// Freshness and quality are mandatory caller inputs: health never calculates,
// mutates, or infers either dimension.
func (assessment HealthAssessment) AsRuntimeState(definition ProviderDefinition, freshness FreshnessState, quality QualityState) (CapabilityRuntimeState, error) {
	if err := assessment.Validate(); err != nil {
		return CapabilityRuntimeState{}, err
	}
	state := CapabilityRuntimeState{
		ContractVersion: CapabilityRuntimeStateV1,
		Provider:        cloneProviderIdentity(assessment.Provider),
		CapabilityID:    assessment.CapabilityID,
		Status:          assessment.Status,
		Freshness:       freshness,
		Quality:         quality,
		ObservedAt:      assessment.AssessedAt,
	}
	if assessment.Status != RuntimeHealthy {
		state.ReasonCode = string(assessment.ReasonCode)
	}
	if err := ValidateRuntimeState(definition, state); err != nil {
		return CapabilityRuntimeState{}, err
	}
	return state, nil
}

type HealthTransition struct {
	From       RuntimeStatus    `json:"from"`
	To         RuntimeStatus    `json:"to"`
	ReasonCode HealthReasonCode `json:"reason_code,omitempty"`
	At         time.Time        `json:"at"`
}

type HealthTracker struct {
	mu           sync.Mutex
	policy       HealthPolicy
	component    canonical.ComponentIdentity
	definition   ProviderDefinition
	capability   CapabilityID
	status       RuntimeStatus
	reason       HealthReasonCode
	lastEvidence time.Time
	lastSuccess  *time.Time
	lastFailure  *time.Time
	failures     int
	successes    int
}

func NewHealthTracker(policy HealthPolicy, component canonical.ComponentIdentity, definition ProviderDefinition, capability CapabilityID) (*HealthTracker, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if err := component.Validate(); err != nil || component.Kind != canonical.ComponentKindSoftwareBuild {
		return nil, fmt.Errorf("health tracker: component must be a software_build identity")
	}
	if err := definition.Validate(); err != nil {
		return nil, err
	}
	declared, ok := definitionCapability(definition, capability)
	if !ok {
		return nil, fmt.Errorf("health tracker: capability is not declared")
	}
	tracker := &HealthTracker{
		policy: policy, component: cloneComponentIdentity(component), definition: cloneDefinition(definition), capability: capability,
		status: RuntimeUnknown, reason: HealthReasonInsufficientEvidence,
	}
	switch declared.Support {
	case SupportUnavailable:
		tracker.status, tracker.reason = RuntimeUnavailable, HealthReasonStaticUnavailable
	case SupportDisabled:
		tracker.status, tracker.reason = RuntimeDisabled, HealthReasonStaticDisabled
	}
	return tracker, nil
}

func (tracker *HealthTracker) ObserveSuccess(at time.Time) (HealthAssessment, *HealthTransition, error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if err := tracker.validateObservationTime(at); err != nil {
		return HealthAssessment{}, nil, err
	}
	if tracker.status == RuntimeDisabled || tracker.reason == HealthReasonStaticUnavailable {
		return tracker.assessment(at), nil, fmt.Errorf("contradictory health transition: static support cannot observe success")
	}
	tracker.expireEvidence(at)
	from := tracker.status
	tracker.lastEvidence = at
	tracker.lastSuccess = cloneTimePointer(&at)
	tracker.failures = 0
	tracker.successes++
	if tracker.successes >= tracker.policy.RecoverySuccesses {
		tracker.status, tracker.reason = RuntimeHealthy, ""
	} else {
		tracker.reason = HealthReasonRecovering
	}
	assessment := tracker.assessment(at)
	return assessment, transitionIfChanged(from, assessment), nil
}

func (tracker *HealthTracker) ObserveFailure(at time.Time, class FailureClass) (HealthAssessment, *HealthTransition, error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if err := tracker.validateObservationTime(at); err != nil {
		return HealthAssessment{}, nil, err
	}
	if !supportedFailureClass(class) {
		return HealthAssessment{}, nil, fmt.Errorf("health tracker: unsupported failure class")
	}
	if tracker.status == RuntimeDisabled || tracker.reason == HealthReasonStaticUnavailable {
		return tracker.assessment(at), nil, nil
	}
	tracker.expireEvidence(at)
	if (!tracker.policy.CountAttemptDeadline && class == FailureAttemptDeadline) || healthNeutralFailure(class) {
		return tracker.assessment(at), nil, nil
	}
	from := tracker.status
	tracker.lastEvidence = at
	tracker.lastFailure = cloneTimePointer(&at)
	tracker.successes = 0
	tracker.failures++
	switch class {
	case FailureAuthentication, FailureAuthorization:
		tracker.status, tracker.reason = RuntimeUnavailable, HealthReasonAuthentication
	case FailureUnsupportedCapability:
		tracker.status, tracker.reason = RuntimeUnavailable, HealthReasonUnsupported
	case FailureRateLimited:
		tracker.status, tracker.reason = RuntimeDegraded, HealthReasonRateLimitExhausted
	default:
		switch {
		case tracker.failures >= tracker.policy.UnavailableAfterFailures:
			tracker.status, tracker.reason = RuntimeUnavailable, HealthReasonUnavailableFailures
		case tracker.failures >= tracker.policy.DegradedAfterFailures:
			tracker.status, tracker.reason = RuntimeDegraded, HealthReasonTransientFailures
		}
	}
	assessment := tracker.assessment(at)
	return assessment, transitionIfChanged(from, assessment), nil
}

func (tracker *HealthTracker) Snapshot(at time.Time) (HealthAssessment, error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !validOperationalTime(at) {
		return HealthAssessment{}, fmt.Errorf("health tracker: assessment time must use UTC")
	}
	assessment := tracker.assessment(at)
	if !tracker.lastEvidence.IsZero() && at.Sub(tracker.lastEvidence) > tracker.policy.AssessmentHorizon && tracker.status != RuntimeDisabled {
		assessment.Status = RuntimeUnknown
		assessment.ReasonCode = HealthReasonHorizonElapsed
	}
	return assessment, assessment.Validate()
}

func (tracker *HealthTracker) validateObservationTime(at time.Time) error {
	if !validOperationalTime(at) {
		return fmt.Errorf("health tracker: observation time must use UTC")
	}
	if !tracker.lastEvidence.IsZero() && at.Before(tracker.lastEvidence) {
		return fmt.Errorf("contradictory health transition: observation precedes existing evidence")
	}
	return nil
}

func (tracker *HealthTracker) assessment(at time.Time) HealthAssessment {
	return HealthAssessment{
		Provider: cloneProviderIdentity(tracker.definition.Identity), CapabilityID: tracker.capability,
		Status: tracker.status, ReasonCode: tracker.reason, AssessedAt: at,
		LastSuccessAt: cloneTimePointer(tracker.lastSuccess), LastFailureAt: cloneTimePointer(tracker.lastFailure),
		ConsecutiveFailures: tracker.failures, ConsecutiveSuccesses: tracker.successes,
		Policy: cloneComponentIdentity(tracker.policy.Identity), Component: cloneComponentIdentity(tracker.component),
	}
}

func (tracker *HealthTracker) expireEvidence(at time.Time) {
	if tracker.lastEvidence.IsZero() || at.Sub(tracker.lastEvidence) <= tracker.policy.AssessmentHorizon {
		return
	}
	tracker.status = RuntimeUnknown
	tracker.reason = HealthReasonInsufficientEvidence
	tracker.failures = 0
	tracker.successes = 0
}

func transitionIfChanged(from RuntimeStatus, assessment HealthAssessment) *HealthTransition {
	if from == assessment.Status {
		return nil
	}
	return &HealthTransition{From: from, To: assessment.Status, ReasonCode: assessment.ReasonCode, At: assessment.AssessedAt}
}

func healthNeutralFailure(class FailureClass) bool {
	switch class {
	case FailureCallerCancelled, FailureOverallDeadline, FailureMalformedRequest, FailureNotFound, FailureConflict,
		FailureProviderPayloadParse, FailureCanonicalValidation, FailureProvenanceMismatch, FailurePermanentRejection:
		return true
	default:
		return false
	}
}

func definitionCapability(definition ProviderDefinition, capability CapabilityID) (Capability, bool) {
	for _, declared := range definition.Capabilities {
		if declared.ID == capability {
			return declared, true
		}
	}
	return Capability{}, false
}
