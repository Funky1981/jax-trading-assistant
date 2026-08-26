package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

type QualificationRegistryErrorCode string

const (
	QualificationErrorUnsupportedRegistryVersion QualificationRegistryErrorCode = "unsupported_qualification_registry_version"
	QualificationErrorUnsupportedDecisionVersion QualificationRegistryErrorCode = "unsupported_qualification_version"
	QualificationErrorInvalidPolicy              QualificationRegistryErrorCode = "invalid_qualification_policy"
	QualificationErrorDuplicatePolicy            QualificationRegistryErrorCode = "duplicate_qualification_policy"
	QualificationErrorUnknownPolicy              QualificationRegistryErrorCode = "unknown_qualification_policy"
	QualificationErrorPolicyVersionMismatch      QualificationRegistryErrorCode = "qualification_policy_version_mismatch"
	QualificationErrorInvalidDecision            QualificationRegistryErrorCode = "invalid_qualification_decision"
	QualificationErrorDuplicateDecision          QualificationRegistryErrorCode = "duplicate_qualification_decision"
	QualificationErrorAmbiguousDecision          QualificationRegistryErrorCode = "ambiguous_qualification_scope"
	QualificationErrorUnknownQualification       QualificationRegistryErrorCode = "unknown_qualification"
	QualificationErrorNoEffectiveDecision        QualificationRegistryErrorCode = "qualification_not_effective"
	QualificationErrorInvalidSourceIdentity      QualificationRegistryErrorCode = "invalid_source_identity"
	QualificationErrorAmbiguousSourceIdentity    QualificationRegistryErrorCode = "ambiguous_source_identity"
	QualificationErrorInvalidProviderIdentity    QualificationRegistryErrorCode = "invalid_provider_identity"
	QualificationErrorCapabilityMismatch         QualificationRegistryErrorCode = "qualification_capability_mismatch"
	QualificationErrorInvalidHistoricalInterval  QualificationRegistryErrorCode = "invalid_historical_effective_interval"
	QualificationErrorInvalidLookupTime          QualificationRegistryErrorCode = "invalid_qualification_lookup_time"
)

type QualificationRegistryError struct {
	Code            QualificationRegistryErrorCode
	QualificationID QualificationID
	SourceID        string
	ProviderID      string
	CapabilityID    CapabilityID
	Detail          string
	Cause           error
}

func (err *QualificationRegistryError) Error() string {
	return fmt.Sprintf("source qualification registry %s: qualification=%q source=%q provider=%q capability=%q %s", err.Code, err.QualificationID, err.SourceID, err.ProviderID, err.CapabilityID, err.Detail)
}

func (err *QualificationRegistryError) Unwrap() error { return err.Cause }

type QualificationEvaluation struct {
	Decision            QualificationDecision `json:"decision"`
	EvaluatedAt         time.Time             `json:"evaluated_at"`
	EffectiveState      QualificationState    `json:"effective_state"`
	ActiveDisqualifiers []HardDisqualifier    `json:"active_disqualifiers,omitempty"`
}

// QualificationRegistry is a deterministic in-memory governance catalogue.
// It stores immutable decisions and exact policy versions; it performs no
// provider selection, health/freshness polling, persistence, or inference.
type QualificationRegistry struct {
	contractVersion canonical.ContractVersion
	providers       *Registry
	policies        map[string]QualificationPolicy
	decisions       map[QualificationID]QualificationDecision
	byScope         map[string][]QualificationID
	sourceKinds     map[string]canonical.SourceKind
	supersededBy    map[QualificationID]QualificationID
}

func NewQualificationRegistry(version canonical.ContractVersion, providers *Registry) (*QualificationRegistry, error) {
	if version != QualificationRegistryContractV1 {
		return nil, &QualificationRegistryError{Code: QualificationErrorUnsupportedRegistryVersion, Detail: fmt.Sprintf("version must be %q", QualificationRegistryContractV1)}
	}
	if providers == nil || providers.ContractVersion() != RegistryContractV1 {
		return nil, &QualificationRegistryError{Code: QualificationErrorInvalidProviderIdentity, Detail: "an accepted provider registry is required"}
	}
	return &QualificationRegistry{
		contractVersion: version, providers: providers,
		policies: make(map[string]QualificationPolicy), decisions: make(map[QualificationID]QualificationDecision),
		byScope: make(map[string][]QualificationID), sourceKinds: make(map[string]canonical.SourceKind),
		supersededBy: make(map[QualificationID]QualificationID),
	}, nil
}

func (registry *QualificationRegistry) ContractVersion() canonical.ContractVersion {
	if registry == nil {
		return ""
	}
	return registry.contractVersion
}

func (registry *QualificationRegistry) RegisterPolicy(policy QualificationPolicy) error {
	if registry == nil || registry.contractVersion != QualificationRegistryContractV1 {
		return &QualificationRegistryError{Code: QualificationErrorUnsupportedRegistryVersion, Detail: "registry is nil or invalid"}
	}
	if err := policy.Validate(); err != nil {
		return &QualificationRegistryError{Code: QualificationErrorInvalidPolicy, Detail: "policy validation failed", Cause: err}
	}
	key := qualificationPolicyKey(policy.Identity)
	if existing, ok := registry.policies[key]; ok {
		code := QualificationErrorDuplicatePolicy
		if !reflect.DeepEqual(existing, policy) {
			code = QualificationErrorInvalidPolicy
		}
		return &QualificationRegistryError{Code: code, Detail: "the exact policy identity/version is already registered"}
	}
	registry.policies[key] = cloneQualificationPolicy(policy)
	return nil
}

func (registry *QualificationRegistry) Register(decision QualificationDecision) error {
	baseErr := func(code QualificationRegistryErrorCode, detail string, cause error) error {
		providerID := ""
		if decision.Scope.ProviderPath != nil {
			providerID = decision.Scope.ProviderPath.ID
		}
		return &QualificationRegistryError{Code: code, QualificationID: decision.ID, SourceID: decision.Scope.Source.ID, ProviderID: providerID, CapabilityID: decision.Scope.CapabilityID, Detail: detail, Cause: cause}
	}
	if registry == nil || registry.contractVersion != QualificationRegistryContractV1 {
		return baseErr(QualificationErrorUnsupportedRegistryVersion, "registry is nil or invalid", nil)
	}
	if decision.DecisionVersion != QualificationDecisionVersionV1() {
		return baseErr(QualificationErrorUnsupportedDecisionVersion, "decision version is not supported", nil)
	}
	if err := decision.Validate(); err != nil {
		return baseErr(QualificationErrorInvalidDecision, "decision validation failed", err)
	}
	policy, err := registry.lookupPolicy(decision.Policy)
	if err != nil {
		return baseErr(QualificationErrorUnknownPolicy, "decision policy is not registered at the exact version", err)
	}
	if decision.DecisionVersion != policy.DecisionVersion {
		return baseErr(QualificationErrorPolicyVersionMismatch, "decision version does not match policy output version", nil)
	}
	if decision.ReviewDueAt == nil || decision.ReviewDueAt.After(decision.AssessedAt.Add(policy.MaximumReviewInterval)) {
		return baseErr(QualificationErrorInvalidDecision, "review_due_at is required within the policy maximum interval", nil)
	}
	derived := deriveQualificationDisqualifiers(policy, decision)
	for _, required := range derived {
		if !containsDisqualifier(decision.Disqualifiers, required) {
			return baseErr(QualificationErrorInvalidDecision, "decision omits a policy-derived hard disqualifier", nil)
		}
	}
	if _, exists := registry.decisions[decision.ID]; exists {
		return baseErr(QualificationErrorDuplicateDecision, "qualification identity is already registered", nil)
	}
	if kind, exists := registry.sourceKinds[decision.Scope.Source.ID]; exists && kind != decision.Scope.Source.Kind {
		return baseErr(QualificationErrorAmbiguousSourceIdentity, "source ID is already associated with a different logical source kind", nil)
	}
	if decision.Scope.ProviderPath != nil {
		capability, providerErr := registry.providers.Capability(*decision.Scope.ProviderPath, decision.Scope.CapabilityID)
		if providerErr != nil {
			return baseErr(QualificationErrorInvalidProviderIdentity, "provider path/capability is not registered", providerErr)
		}
		if capability.Support != SupportSupported {
			return baseErr(QualificationErrorCapabilityMismatch, "provider path does not statically support the qualified capability", nil)
		}
	}
	scopeKey, err := qualificationScopeKey(decision.Scope)
	if err != nil {
		return baseErr(QualificationErrorInvalidDecision, "scope cannot be encoded deterministically", err)
	}
	var superseded QualificationID
	if decision.Supersedes != nil {
		superseded = *decision.Supersedes
		prior, ok := registry.decisions[superseded]
		if !ok {
			return baseErr(QualificationErrorInvalidHistoricalInterval, "superseded qualification is not registered", nil)
		}
		if !sameQualificationScope(prior.Scope, decision.Scope) || !decision.EffectiveFrom.After(prior.EffectiveFrom) {
			return baseErr(QualificationErrorInvalidHistoricalInterval, "supersession must preserve exact scope and advance effective time", nil)
		}
		if _, already := registry.supersededBy[superseded]; already {
			return baseErr(QualificationErrorAmbiguousDecision, "superseded qualification already has a successor", nil)
		}
	}
	for _, existingID := range registry.byScope[scopeKey] {
		existing := registry.decisions[existingID]
		if existingID == superseded {
			continue
		}
		if qualificationIntervalsOverlap(existing, decision, registry) {
			return baseErr(QualificationErrorAmbiguousDecision, "effective interval overlaps another decision for the exact scope", nil)
		}
	}
	copyDecision := cloneQualificationDecision(decision)
	registry.decisions[decision.ID] = copyDecision
	registry.byScope[scopeKey] = append(registry.byScope[scopeKey], decision.ID)
	sort.Slice(registry.byScope[scopeKey], func(i, j int) bool {
		left := registry.decisions[registry.byScope[scopeKey][i]]
		right := registry.decisions[registry.byScope[scopeKey][j]]
		if left.EffectiveFrom.Equal(right.EffectiveFrom) {
			return left.ID < right.ID
		}
		return left.EffectiveFrom.Before(right.EffectiveFrom)
	})
	registry.sourceKinds[decision.Scope.Source.ID] = decision.Scope.Source.Kind
	if superseded != "" {
		registry.supersededBy[superseded] = decision.ID
	}
	return nil
}

// LookupAt performs exact typed-scope lookup at an explicit UTC time. Unknown
// scope, explicit UNQUALIFIED, expiry, and review overdue remain distinguishable.
func (registry *QualificationRegistry) LookupAt(scope QualificationScope, at time.Time) (QualificationEvaluation, error) {
	baseErr := func(code QualificationRegistryErrorCode, detail string) error {
		providerID := ""
		if scope.ProviderPath != nil {
			providerID = scope.ProviderPath.ID
		}
		return &QualificationRegistryError{Code: code, SourceID: scope.Source.ID, ProviderID: providerID, CapabilityID: scope.CapabilityID, Detail: detail}
	}
	if registry == nil {
		return QualificationEvaluation{}, baseErr(QualificationErrorUnknownQualification, "registry is nil")
	}
	if err := scope.Validate(); err != nil {
		return QualificationEvaluation{}, baseErr(QualificationErrorInvalidSourceIdentity, "scope is invalid")
	}
	if !validOperationalTime(at) {
		return QualificationEvaluation{}, baseErr(QualificationErrorInvalidLookupTime, "lookup time is required and must use UTC")
	}
	key, err := qualificationScopeKey(scope)
	if err != nil {
		return QualificationEvaluation{}, baseErr(QualificationErrorUnknownQualification, "scope cannot be encoded")
	}
	ids := registry.byScope[key]
	if len(ids) == 0 {
		return QualificationEvaluation{}, baseErr(QualificationErrorUnknownQualification, "no qualification has been assessed for the exact scope")
	}
	active := []QualificationDecision{}
	var latestPrior *QualificationDecision
	for _, id := range ids {
		decision := registry.decisions[id]
		if !decision.EffectiveFrom.After(at) {
			copyDecision := decision
			if latestPrior == nil || copyDecision.EffectiveFrom.After(latestPrior.EffectiveFrom) {
				latestPrior = &copyDecision
			}
		}
		if qualificationEffectiveAt(decision, at, registry) {
			active = append(active, decision)
		}
	}
	if len(active) > 1 {
		return QualificationEvaluation{}, baseErr(QualificationErrorAmbiguousDecision, "multiple qualification decisions are effective")
	}
	if len(active) == 1 {
		return qualificationEvaluationAt(active[0], at, false), nil
	}
	if latestPrior != nil {
		return qualificationEvaluationAt(*latestPrior, at, true), nil
	}
	return QualificationEvaluation{}, baseErr(QualificationErrorNoEffectiveDecision, "qualification exists but was not yet effective")
}

func (registry *QualificationRegistry) History(scope QualificationScope) ([]QualificationDecision, error) {
	if registry == nil {
		return nil, &QualificationRegistryError{Code: QualificationErrorUnknownQualification, Detail: "registry is nil"}
	}
	if err := scope.Validate(); err != nil {
		return nil, &QualificationRegistryError{Code: QualificationErrorInvalidSourceIdentity, SourceID: scope.Source.ID, Detail: "scope is invalid", Cause: err}
	}
	key, err := qualificationScopeKey(scope)
	if err != nil || len(registry.byScope[key]) == 0 {
		return nil, &QualificationRegistryError{Code: QualificationErrorUnknownQualification, SourceID: scope.Source.ID, CapabilityID: scope.CapabilityID, Detail: "no qualification history exists for the exact scope"}
	}
	result := make([]QualificationDecision, 0, len(registry.byScope[key]))
	for _, id := range registry.byScope[key] {
		result = append(result, cloneQualificationDecision(registry.decisions[id]))
	}
	return result, nil
}

func (registry *QualificationRegistry) lookupPolicy(identity canonical.ComponentIdentity) (QualificationPolicy, error) {
	if err := identity.Validate(); err != nil {
		return QualificationPolicy{}, err
	}
	policy, ok := registry.policies[qualificationPolicyKey(identity)]
	if !ok {
		return QualificationPolicy{}, fmt.Errorf("policy is not registered")
	}
	if !sameComponentIdentity(policy.Identity, identity) {
		return QualificationPolicy{}, fmt.Errorf("policy identity/version mismatch")
	}
	return cloneQualificationPolicy(policy), nil
}

func qualificationEvaluationAt(decision QualificationDecision, at time.Time, expired bool) QualificationEvaluation {
	disqualifiers := append([]HardDisqualifier(nil), decision.Disqualifiers...)
	if expired {
		disqualifiers = append(disqualifiers, DisqualifierQualificationExpired)
	}
	if decision.ReviewDueAt != nil && !at.Before(*decision.ReviewDueAt) {
		disqualifiers = append(disqualifiers, DisqualifierReviewOverdue)
	}
	disqualifiers = uniqueSortedDisqualifiers(disqualifiers)
	state := decision.State
	if (expired || containsDisqualifier(disqualifiers, DisqualifierReviewOverdue)) && state != QualificationDisabled {
		state = QualificationUnqualified
	}
	return QualificationEvaluation{Decision: cloneQualificationDecision(decision), EvaluatedAt: at, EffectiveState: state, ActiveDisqualifiers: disqualifiers}
}

func qualificationEffectiveAt(decision QualificationDecision, at time.Time, registry *QualificationRegistry) bool {
	if decision.EffectiveFrom.After(at) {
		return false
	}
	if decision.EffectiveTo != nil && !at.Before(*decision.EffectiveTo) {
		return false
	}
	if successorID, ok := registry.supersededBy[decision.ID]; ok {
		successor := registry.decisions[successorID]
		if !at.Before(successor.EffectiveFrom) {
			return false
		}
	}
	return true
}

func qualificationIntervalsOverlap(left, right QualificationDecision, registry *QualificationRegistry) bool {
	leftEnd := left.EffectiveTo
	if successorID, ok := registry.supersededBy[left.ID]; ok {
		successor := registry.decisions[successorID]
		if leftEnd == nil || successor.EffectiveFrom.Before(*leftEnd) {
			leftEnd = &successor.EffectiveFrom
		}
	}
	rightEnd := right.EffectiveTo
	leftBeforeRightEnd := rightEnd == nil || left.EffectiveFrom.Before(*rightEnd)
	rightBeforeLeftEnd := leftEnd == nil || right.EffectiveFrom.Before(*leftEnd)
	return leftBeforeRightEnd && rightBeforeLeftEnd
}

func qualificationScopeKey(scope QualificationScope) (string, error) {
	raw, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func qualificationPolicyKey(identity canonical.ComponentIdentity) string {
	return identity.ID + "\x00" + identity.Version.Namespace + "\x00" + identity.Version.Value
}

func cloneQualificationPolicy(policy QualificationPolicy) QualificationPolicy {
	copyPolicy := policy
	copyPolicy.Identity = cloneComponentIdentity(policy.Identity)
	copyPolicy.RequiredEvidence = append([]QualificationEvidenceKind(nil), policy.RequiredEvidence...)
	return copyPolicy
}
