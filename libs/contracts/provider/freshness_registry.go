package provider

import (
	"fmt"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

func (policy FreshnessPolicy) Validate() error {
	if policy.ContractVersion != FreshnessPolicyContractV1 {
		return freshnessError(FreshnessErrorUnsupportedPolicyVersion, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, fmt.Sprintf("contract_version must be %q", FreshnessPolicyContractV1), nil)
	}
	if err := validateFreshnessPolicyIdentity(policy.Identity); err != nil {
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "freshness policy identity is invalid", err)
	}
	outputs, ok := CanonicalOutputsFor(policy.CapabilityID)
	if !ok {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "capability is not a supported Jax data capability", nil)
	}
	if err := policy.Target.Validate(); err != nil {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "canonical target is invalid", err)
	}
	targetAllowed := false
	for _, output := range outputs {
		if output == policy.Target {
			targetAllowed = true
			break
		}
	}
	if !targetAllowed {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "canonical target is not an authoritative output for the capability", nil)
	}
	switch policy.UseClass {
	case DataUseResearch, DataUseDisplay, DataUseDecisionSupport:
	default:
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "data use class is not supported", nil)
	}
	if !supportedTimestampRole(policy.TimestampRole) {
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "authoritative timestamp role is not supported", nil)
	}
	switch policy.MissingTimestamp {
	case MissingTimestampFail, MissingTimestampUnknown:
	default:
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "missing timestamp behavior is not supported", nil)
	}
	if policy.AllowedFutureSkew < 0 {
		return freshnessError(FreshnessErrorInvalidTTL, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "allowed future skew must not be negative", nil)
	}
	switch policy.ValidityMode {
	case FreshnessValidityAgeBounded:
		if policy.TimestampRole == TimestampRoleNone {
			return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "age-bounded policy requires an authoritative timestamp role", nil)
		}
		if policy.FreshFor <= 0 || policy.ExpireAfter <= policy.FreshFor {
			return freshnessError(FreshnessErrorInvalidTTL, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "age-bounded policy requires fresh_for > 0 and expire_after > fresh_for", nil)
		}
	case FreshnessValidityUntilSuperseded:
		if policy.FreshFor != 0 || policy.ExpireAfter != 0 {
			return freshnessError(FreshnessErrorInvalidTTL, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "until-superseded policy must not encode an age TTL or expiry horizon", nil)
		}
		if policy.TimestampRole == TimestampRoleNone && policy.AllowedFutureSkew != 0 {
			return freshnessError(FreshnessErrorInvalidTTL, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "future skew has no meaning without an authoritative timestamp", nil)
		}
	case FreshnessValidityNotApplicable:
		if policy.TimestampRole != TimestampRoleNone || policy.FreshFor != 0 || policy.ExpireAfter != 0 || policy.AllowedFutureSkew != 0 {
			return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "not-applicable policy must not declare timestamp or duration semantics", nil)
		}
	default:
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "validity mode is not supported", nil)
	}
	if err := policy.LastKnownGood.validateAgainst(policy); err != nil {
		return err
	}
	return nil
}

func (policy LastKnownGoodPolicy) validateAgainst(freshness FreshnessPolicy) error {
	if policy.ContractVersion != LastKnownGoodPolicyContractV1 {
		return freshnessError(FreshnessErrorUnsupportedPolicyVersion, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, fmt.Sprintf("last-known-good contract_version must be %q", LastKnownGoodPolicyContractV1), nil)
	}
	if err := validateFreshnessPolicyIdentity(policy.Identity); err != nil {
		return freshnessError(FreshnessErrorInvalidPolicy, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "last-known-good policy identity is invalid", err)
	}
	switch policy.Mode {
	case FallbackProhibited:
		if policy.MaximumAge != 0 {
			return freshnessError(FreshnessErrorInvalidTTL, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "prohibited fallback must not declare maximum age", nil)
		}
	case FallbackFreshOnly:
		if policy.MaximumAge <= 0 {
			return freshnessError(FreshnessErrorInvalidTTL, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "allowed fallback requires a positive maximum age", nil)
		}
	case FallbackFreshOrStale:
		if policy.MaximumAge <= 0 {
			return freshnessError(FreshnessErrorInvalidTTL, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "allowed stale fallback requires a positive maximum age", nil)
		}
		if freshness.ValidityMode != FreshnessValidityAgeBounded {
			return freshnessError(FreshnessErrorInvalidPolicy, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "stale fallback requires an age-bounded freshness policy", nil)
		}
		if policy.MaximumAge < freshness.FreshFor {
			return freshnessError(FreshnessErrorInvalidTTL, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "stale fallback maximum age must not be shorter than the fresh horizon", nil)
		}
	default:
		return freshnessError(FreshnessErrorInvalidPolicy, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "fallback mode is not supported", nil)
	}
	if freshness.ValidityMode != FreshnessValidityAgeBounded && policy.Mode != FallbackProhibited {
		return freshnessError(FreshnessErrorInvalidPolicy, freshness.Identity, freshness.CapabilityID, canonical.ImmutableContractRef{}, "fallback is only defined for age-bounded data", nil)
	}
	return nil
}

func validateFreshnessPolicyIdentity(identity canonical.ComponentIdentity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	if identity.Kind != canonical.ComponentKindPolicy {
		return fmt.Errorf("component kind must be policy")
	}
	if identity.Provider != nil {
		return fmt.Errorf("generic freshness policy must not be bound to a provider")
	}
	return nil
}

func supportedTimestampRole(role AuthoritativeTimestampRole) bool {
	switch role {
	case TimestampRoleNone, TimestampRoleObservedAt, TimestampRolePublishedAt, TimestampRoleCollectedAt,
		TimestampRoleOccurredAt, TimestampRoleEffectiveAt, TimestampRoleEffectiveFrom, TimestampRoleDatasetAsOf:
		return true
	default:
		return false
	}
}

func (key FreshnessKey) validate(policy FreshnessPolicy) error {
	if key.CapabilityID != policy.CapabilityID || key.Target != policy.Target {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, key.CapabilityID, canonical.ImmutableContractRef{}, "semantic key capability/target does not match policy applicability", nil)
	}
	if key.Subject.Kind == "" || key.Subject.ID == "" || key.Subject.ContractVersion == "" {
		return freshnessError(FreshnessErrorSemanticKeyMismatch, policy.Identity, key.CapabilityID, canonical.ImmutableContractRef{}, "semantic key requires a typed canonical subject", nil)
	}
	if err := validateCode("freshness_key", "qualifier", key.Qualifier); err != nil {
		return freshnessError(FreshnessErrorSemanticKeyMismatch, policy.Identity, key.CapabilityID, canonical.ImmutableContractRef{}, "semantic qualifier is invalid", err)
	}
	return nil
}

func (lifecycle TemporalRecordLifecycle) validate(policy FreshnessPolicy, record canonical.ImmutableContractRef) error {
	switch lifecycle.State {
	case TemporalRecordActive:
		if lifecycle.ChangedAt != nil {
			return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, record, "active lifecycle must not declare changed_at", nil)
		}
	case TemporalRecordSuperseded, TemporalRecordRetracted, TemporalRecordDisputed, TemporalRecordInvalidated:
		if lifecycle.ChangedAt == nil || !validEvaluationTime(*lifecycle.ChangedAt) {
			return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, record, "inactive lifecycle requires a UTC changed_at", nil)
		}
	default:
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, record, "record lifecycle state is not supported", nil)
	}
	return nil
}

func validEvaluationTime(value time.Time) bool {
	_, offset := value.Zone()
	return !value.IsZero() && offset == 0 && value.Year() >= 0 && value.Year() <= 9999
}

type freshnessPolicyRoute struct {
	capability      CapabilityID
	targetKind      canonical.ContractKind
	targetVersion   canonical.ContractVersion
	useClass        DataUseClass
	policyNamespace string
	policyVersion   string
}

// FreshnessPolicyRegistry is a deterministic process-local catalogue. It
// performs exact version lookup and has no configuration service, persistence,
// provider calls, or dynamic policy loading.
type FreshnessPolicyRegistry struct {
	routes map[freshnessPolicyRoute]FreshnessPolicy
	byID   map[string]FreshnessPolicy
}

func NewFreshnessPolicyRegistry() *FreshnessPolicyRegistry {
	return &FreshnessPolicyRegistry{routes: make(map[freshnessPolicyRoute]FreshnessPolicy), byID: make(map[string]FreshnessPolicy)}
}

func (registry *FreshnessPolicyRegistry) Register(policy FreshnessPolicy) error {
	if registry == nil {
		return freshnessError(FreshnessErrorInvalidPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "freshness policy registry is not initialized", nil)
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	if existing, ok := registry.byID[policy.Identity.ID]; ok {
		code := FreshnessErrorDuplicatePolicy
		if !samePolicyIdentity(existing.Identity, policy.Identity) || !freshnessPoliciesEqual(existing, policy) {
			code = FreshnessErrorAmbiguousPolicy
		}
		return freshnessError(code, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "policy identity is already registered", nil)
	}
	route := routeForFreshnessPolicy(policy)
	if _, exists := registry.routes[route]; exists {
		return freshnessError(FreshnessErrorAmbiguousPolicy, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "capability/target/use/version route already has a policy", nil)
	}
	copyPolicy := cloneFreshnessPolicy(policy)
	registry.routes[route] = copyPolicy
	registry.byID[policy.Identity.ID] = copyPolicy
	return nil
}

func (registry *FreshnessPolicyRegistry) Lookup(capability CapabilityID, target canonical.ContractSchemaRef, useClass DataUseClass, identity canonical.ComponentIdentity) (FreshnessPolicy, error) {
	if registry == nil {
		return FreshnessPolicy{}, freshnessError(FreshnessErrorUnknownPolicy, identity, capability, canonical.ImmutableContractRef{}, "freshness policy registry is not initialized", nil)
	}
	known, exists := registry.byID[identity.ID]
	if !exists {
		return FreshnessPolicy{}, freshnessError(FreshnessErrorUnknownPolicy, identity, capability, canonical.ImmutableContractRef{}, "freshness policy identity is not registered", nil)
	}
	if !samePolicyIdentity(known.Identity, identity) {
		return FreshnessPolicy{}, freshnessError(FreshnessErrorPolicyVersionMismatch, identity, capability, canonical.ImmutableContractRef{}, "requested policy identity/version does not match the registered policy", nil)
	}
	route := freshnessPolicyRoute{capability: capability, targetKind: target.Kind, targetVersion: target.Version, useClass: useClass, policyNamespace: identity.Version.Namespace, policyVersion: identity.Version.Value}
	policy, ok := registry.routes[route]
	if !ok || policy.Identity.ID != identity.ID {
		return FreshnessPolicy{}, freshnessError(FreshnessErrorPolicyCapabilityMismatch, identity, capability, canonical.ImmutableContractRef{}, "requested policy is not applicable to the capability/target/use route", nil)
	}
	return cloneFreshnessPolicy(policy), nil
}

func routeForFreshnessPolicy(policy FreshnessPolicy) freshnessPolicyRoute {
	return freshnessPolicyRoute{capability: policy.CapabilityID, targetKind: policy.Target.Kind, targetVersion: policy.Target.Version, useClass: policy.UseClass, policyNamespace: policy.Identity.Version.Namespace, policyVersion: policy.Identity.Version.Value}
}

func cloneFreshnessPolicy(policy FreshnessPolicy) FreshnessPolicy {
	copyPolicy := policy
	copyPolicy.Identity = cloneComponentIdentity(policy.Identity)
	copyPolicy.LastKnownGood.Identity = cloneComponentIdentity(policy.LastKnownGood.Identity)
	return copyPolicy
}

func freshnessPoliciesEqual(left, right FreshnessPolicy) bool {
	return left.ContractVersion == right.ContractVersion && samePolicyIdentity(left.Identity, right.Identity) &&
		left.CapabilityID == right.CapabilityID && left.Target == right.Target && left.UseClass == right.UseClass &&
		left.ValidityMode == right.ValidityMode && left.TimestampRole == right.TimestampRole && left.FreshFor == right.FreshFor &&
		left.ExpireAfter == right.ExpireAfter && left.AllowedFutureSkew == right.AllowedFutureSkew &&
		left.MissingTimestamp == right.MissingTimestamp && left.LastKnownGood.ContractVersion == right.LastKnownGood.ContractVersion &&
		samePolicyIdentity(left.LastKnownGood.Identity, right.LastKnownGood.Identity) && left.LastKnownGood.Mode == right.LastKnownGood.Mode &&
		left.LastKnownGood.MaximumAge == right.LastKnownGood.MaximumAge
}
