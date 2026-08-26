package provider

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	QualificationRegistryContractV1 canonical.ContractVersion = "jax.source_qualification_registry/v1"
	QualificationPolicyContractV1   canonical.ContractVersion = "jax.source_qualification_policy/v1"
	QualificationDecisionContractV1 canonical.ContractVersion = "jax.source_qualification_decision/v1"
)

func QualificationDecisionVersionV1() canonical.VersionIdentity {
	return canonical.VersionIdentity{Namespace: "jax.qualification_decision", Value: "v1"}
}

type QualificationID string

type QualificationState string

const (
	QualificationQualified              QualificationState = "QUALIFIED"
	QualificationConditionallyQualified QualificationState = "CONDITIONALLY_QUALIFIED"
	QualificationUnqualified            QualificationState = "UNQUALIFIED"
	QualificationNotAssessed            QualificationState = "NOT_ASSESSED"
	QualificationDisabled               QualificationState = "DISABLED"
)

// SourceAuthorityClass is a characteristic of a logical source. It is not a
// qualification decision and says nothing about current provider health.
type SourceAuthorityClass string

const (
	SourceAuthorityOfficialPrimary       SourceAuthorityClass = "OFFICIAL_PRIMARY"
	SourceAuthorityRegulatedExchange     SourceAuthorityClass = "REGULATED_OR_EXCHANGE"
	SourceAuthorityDirectIssuer          SourceAuthorityClass = "DIRECT_ISSUER"
	SourceAuthorityProfessionalSecondary SourceAuthorityClass = "PROFESSIONAL_SECONDARY"
	SourceAuthorityAggregator            SourceAuthorityClass = "AGGREGATOR"
	SourceAuthorityCommunityUnverified   SourceAuthorityClass = "COMMUNITY_UNVERIFIED"
	SourceAuthorityModelDerived          SourceAuthorityClass = "MODEL_DERIVED"
	SourceAuthorityOther                 SourceAuthorityClass = "OTHER"
	SourceAuthorityUnknown               SourceAuthorityClass = "UNKNOWN"
)

type QualificationRole string

const (
	QualificationRoleAuthoritativeSource    QualificationRole = "AUTHORITATIVE_SOURCE"
	QualificationRolePrimaryEvidence        QualificationRole = "PRIMARY_EVIDENCE_SOURCE"
	QualificationRoleSecondaryCorroboration QualificationRole = "SECONDARY_CORROBORATION_SOURCE"
	QualificationRoleMarketObservation      QualificationRole = "MARKET_OBSERVATION_SOURCE"
	QualificationRoleReferenceData          QualificationRole = "REFERENCE_DATA_SOURCE"
	QualificationRoleFilingDocument         QualificationRole = "FILING_DOCUMENT_SOURCE"
	QualificationRoleMacroeconomicAuthority QualificationRole = "MACROECONOMIC_AUTHORITY"
	QualificationRoleNewsEvidence           QualificationRole = "NEWS_EVIDENCE_SOURCE"
	QualificationRoleEventDetection         QualificationRole = "EVENT_DETECTION_SOURCE"
	QualificationRoleResearchSupplementary  QualificationRole = "RESEARCH_SUPPLEMENTARY_SOURCE"
)

// QualificationUse is the bounded action being assessed. It deliberately
// excludes recommendation, approval, order, execution, and broker authority.
type QualificationUse string

const (
	QualificationUsePrimaryEvidence           QualificationUse = "PRIMARY_EVIDENCE"
	QualificationUseCorroboration             QualificationUse = "CORROBORATION"
	QualificationUseInvestigationTrigger      QualificationUse = "INVESTIGATION_TRIGGER"
	QualificationUseResearchDisplay           QualificationUse = "RESEARCH_DISPLAY"
	QualificationUseFallbackInput             QualificationUse = "FALLBACK_INPUT"
	QualificationUseCanonicalFactualAssertion QualificationUse = "CANONICAL_FACTUAL_ASSERTION"
)

type PermissionState string

const (
	PermissionAllowed     PermissionState = "ALLOWED"
	PermissionConditional PermissionState = "CONDITIONAL"
	PermissionProhibited  PermissionState = "PROHIBITED"
	PermissionNotAssessed PermissionState = "NOT_ASSESSED"
)

// QualificationPermissions makes corroborating-only sources explicit. A
// consumer must inspect the permission for its exact requested use.
type QualificationPermissions struct {
	PrimaryEvidence           PermissionState `json:"primary_evidence"`
	Corroboration             PermissionState `json:"corroboration"`
	InvestigationTrigger      PermissionState `json:"investigation_trigger"`
	ResearchDisplay           PermissionState `json:"research_display"`
	FallbackInput             PermissionState `json:"fallback_input"`
	CanonicalFactualAssertion PermissionState `json:"canonical_factual_assertion"`
}

func (permissions QualificationPermissions) For(use QualificationUse) PermissionState {
	switch use {
	case QualificationUsePrimaryEvidence:
		return permissions.PrimaryEvidence
	case QualificationUseCorroboration:
		return permissions.Corroboration
	case QualificationUseInvestigationTrigger:
		return permissions.InvestigationTrigger
	case QualificationUseResearchDisplay:
		return permissions.ResearchDisplay
	case QualificationUseFallbackInput:
		return permissions.FallbackInput
	case QualificationUseCanonicalFactualAssertion:
		return permissions.CanonicalFactualAssertion
	default:
		return ""
	}
}

type QualificationConditionType string

const (
	ConditionIndependentPrimaryRequired QualificationConditionType = "INDEPENDENT_PRIMARY_REQUIRED"
	ConditionResearchDisplayOnly        QualificationConditionType = "RESEARCH_DISPLAY_ONLY"
	ConditionCoverageRestricted         QualificationConditionType = "COVERAGE_RESTRICTED"
	ConditionUsageRestrictionsApply     QualificationConditionType = "USAGE_RESTRICTIONS_APPLY"
	ConditionRetentionLimitApplies      QualificationConditionType = "RETENTION_LIMIT_APPLIES"
	ConditionFreshnessPolicyRequired    QualificationConditionType = "FRESHNESS_POLICY_REQUIRED"
)

type QualificationCondition struct {
	Type      QualificationConditionType   `json:"type"`
	AppliesTo QualificationUse             `json:"applies_to"`
	Policy    *canonical.ComponentIdentity `json:"policy,omitempty"`
}

type CoverageStatus string
type CoverageScopeKind string
type CoverageDimension string

const (
	CoverageUnknown     CoverageStatus = "UNKNOWN"
	CoverageSupported   CoverageStatus = "SUPPORTED"
	CoverageUnsupported CoverageStatus = "UNSUPPORTED"

	CoverageScopeUnknown        CoverageScopeKind = "UNKNOWN"
	CoverageScopeCapabilityWide CoverageScopeKind = "CAPABILITY_WIDE"
	CoverageScopeBounded        CoverageScopeKind = "BOUNDED"

	CoverageDimensionGeography      CoverageDimension = "GEOGRAPHY"
	CoverageDimensionInstrumentType CoverageDimension = "INSTRUMENT_TYPE"
	CoverageDimensionVenue          CoverageDimension = "VENUE"
	CoverageDimensionSeries         CoverageDimension = "SERIES"
	CoverageDimensionDataset        CoverageDimension = "DATASET"
)

type CoverageBound struct {
	Dimension CoverageDimension `json:"dimension"`
	Value     string            `json:"value"`
}

type QualificationCoverage struct {
	Status CoverageStatus    `json:"status"`
	Scope  CoverageScopeKind `json:"scope"`
	Bounds []CoverageBound   `json:"bounds,omitempty"`
}

type IdentityCertainty string

const (
	IdentityCertaintyUnknown  IdentityCertainty = "UNKNOWN"
	IdentityCertaintyLow      IdentityCertainty = "LOW"
	IdentityCertaintyModerate IdentityCertainty = "MODERATE"
	IdentityCertaintyHigh     IdentityCertainty = "HIGH"
	IdentityCertaintyVerified IdentityCertainty = "VERIFIED"
)

type ReliabilityClass string
type OutageCharacteristic string
type SchemaChangeFrequency string

const (
	ReliabilityUnknown      ReliabilityClass = "UNKNOWN"
	ReliabilityInsufficient ReliabilityClass = "INSUFFICIENT"
	ReliabilityLimited      ReliabilityClass = "LIMITED"
	ReliabilityAcceptable   ReliabilityClass = "ACCEPTABLE"
	ReliabilityStrong       ReliabilityClass = "STRONG"

	OutageUnknown     OutageCharacteristic = "UNKNOWN"
	OutageRareBounded OutageCharacteristic = "RARE_BOUNDED"
	OutageOccasional  OutageCharacteristic = "OCCASIONAL"
	OutageFrequent    OutageCharacteristic = "FREQUENT"
	OutageUnbounded   OutageCharacteristic = "UNBOUNDED"

	SchemaChangeUnknown    SchemaChangeFrequency = "UNKNOWN"
	SchemaChangeRare       SchemaChangeFrequency = "RARE"
	SchemaChangeOccasional SchemaChangeFrequency = "OCCASIONAL"
	SchemaChangeFrequent   SchemaChangeFrequency = "FREQUENT"
)

// HistoricalReliability records reviewed historical evidence. It is not
// WP-02.05 current health and is never populated from ambient telemetry here.
type HistoricalReliability struct {
	Class                      ReliabilityClass      `json:"class"`
	WindowFrom                 *time.Time            `json:"window_from,omitempty"`
	WindowTo                   *time.Time            `json:"window_to,omitempty"`
	DeliverySuccessBasisPoints *int                  `json:"delivery_success_basis_points,omitempty"`
	MissingDataBasisPoints     *int                  `json:"missing_data_basis_points,omitempty"`
	CorrectionBasisPoints      *int                  `json:"correction_basis_points,omitempty"`
	Outages                    OutageCharacteristic  `json:"outages"`
	SchemaChanges              SchemaChangeFrequency `json:"schema_changes"`
}

type AvailabilityClass string
type UpdateLatencyClass string
type ProvenanceQualityClass string
type AccessStabilityClass string
type SchemaStabilityClass string

const (
	AvailabilityUnknown            AvailabilityClass = "UNKNOWN"
	AvailabilityContinuousExpected AvailabilityClass = "CONTINUOUS_EXPECTED"
	AvailabilityScheduled          AvailabilityClass = "SCHEDULED"
	AvailabilityBestEffort         AvailabilityClass = "BEST_EFFORT"

	UpdateLatencyUnknown     UpdateLatencyClass = "UNKNOWN"
	UpdateLatencyRealTime    UpdateLatencyClass = "REAL_TIME"
	UpdateLatencyDelayed     UpdateLatencyClass = "DELAYED"
	UpdateLatencyPeriodic    UpdateLatencyClass = "PERIODIC"
	UpdateLatencyEventDriven UpdateLatencyClass = "EVENT_DRIVEN"

	ProvenanceQualityUnknown                ProvenanceQualityClass = "UNKNOWN"
	ProvenanceQualityInsufficient           ProvenanceQualityClass = "INSUFFICIENT"
	ProvenanceQualitySourceIdentified       ProvenanceQualityClass = "SOURCE_IDENTIFIED"
	ProvenanceQualitySourceRevision         ProvenanceQualityClass = "SOURCE_AND_REVISION"
	ProvenanceQualityImmutableRawAndLineage ProvenanceQualityClass = "IMMUTABLE_RAW_AND_LINEAGE"

	AccessStabilityUnknown  AccessStabilityClass = "UNKNOWN"
	AccessStabilityUnstable AccessStabilityClass = "UNSTABLE"
	AccessStabilityVariable AccessStabilityClass = "VARIABLE"
	AccessStabilityStable   AccessStabilityClass = "STABLE"

	SchemaStabilityUnknown  SchemaStabilityClass = "UNKNOWN"
	SchemaStabilityUnstable SchemaStabilityClass = "UNSTABLE"
	SchemaStabilityManaged  SchemaStabilityClass = "MANAGED_CHANGE"
	SchemaStabilityStable   SchemaStabilityClass = "STABLE"
)

type UseRight string
type RetentionRight string

const (
	UseRightUnknown    UseRight = "UNKNOWN"
	UseRightAllowed    UseRight = "ALLOWED"
	UseRightRestricted UseRight = "RESTRICTED"
	UseRightProhibited UseRight = "PROHIBITED"

	RetentionRightUnknown     RetentionRight = "UNKNOWN"
	RetentionRightIndefinite  RetentionRight = "INDEFINITE_ALLOWED"
	RetentionRightBounded     RetentionRight = "BOUNDED"
	RetentionRightSessionOnly RetentionRight = "SESSION_ONLY"
	RetentionRightProhibited  RetentionRight = "PROHIBITED"
)

// UsageRights contains reviewed facts, not legal interpretation or vendor
// legal text. Restricted rights require an explicit typed condition.
type UsageRights struct {
	InternalProcessing UseRight       `json:"internal_processing"`
	PersistentStorage  UseRight       `json:"persistent_storage"`
	DerivedData        UseRight       `json:"derived_data"`
	ModelResearchUse   UseRight       `json:"model_research_use"`
	Redistribution     UseRight       `json:"redistribution"`
	CommercialUse      UseRight       `json:"commercial_use"`
	Retention          RetentionRight `json:"retention"`
}

type CostModel string
type CostClass string

const (
	CostModelFree              CostModel = "FREE"
	CostModelFixedSubscription CostModel = "FIXED_SUBSCRIPTION"
	CostModelUsageBased        CostModel = "USAGE_BASED"
	CostModelTiered            CostModel = "TIERED"
	CostModelInternal          CostModel = "INTERNAL"
	CostModelUnknown           CostModel = "UNKNOWN"

	CostClassNegligible CostClass = "NEGLIGIBLE"
	CostClassLow        CostClass = "LOW"
	CostClassModerate   CostClass = "MODERATE"
	CostClassHigh       CostClass = "HIGH"
	CostClassUnknown    CostClass = "UNKNOWN"
)

type CostCharacteristics struct {
	Model CostModel `json:"model"`
	Class CostClass `json:"class"`
}

type QualificationDimensions struct {
	IdentityCertainty IdentityCertainty      `json:"identity_certainty"`
	Reliability       HistoricalReliability  `json:"historical_reliability"`
	Availability      AvailabilityClass      `json:"availability"`
	UpdateLatency     UpdateLatencyClass     `json:"update_latency"`
	ProvenanceQuality ProvenanceQualityClass `json:"provenance_quality"`
	Rights            UsageRights            `json:"usage_rights"`
	Cost              CostCharacteristics    `json:"cost"`
	AccessStability   AccessStabilityClass   `json:"access_stability"`
	SchemaStability   SchemaStabilityClass   `json:"schema_stability"`
	Coverage          QualificationCoverage  `json:"coverage"`
}

type QualificationEvidenceKind string

const (
	EvidenceSourceIdentity          QualificationEvidenceKind = "SOURCE_IDENTITY"
	EvidenceAuthorityClassification QualificationEvidenceKind = "AUTHORITY_CLASSIFICATION"
	EvidenceSourceDocumentation     QualificationEvidenceKind = "SOURCE_DOCUMENTATION_SNAPSHOT"
	EvidenceLicensingReview         QualificationEvidenceKind = "LICENSING_REVIEW"
	EvidenceHistoricalReliability   QualificationEvidenceKind = "HISTORICAL_RELIABILITY_SNAPSHOT"
	EvidenceCoverageReview          QualificationEvidenceKind = "COVERAGE_REVIEW"
	EvidenceApprovedInternalReview  QualificationEvidenceKind = "APPROVED_INTERNAL_REVIEW"
)

type QualificationEvidence struct {
	Kind QualificationEvidenceKind `json:"kind"`
	Ref  canonical.EvidenceRef     `json:"ref"`
}

type HardDisqualifier string

const (
	DisqualifierUnknownSourceIdentity   HardDisqualifier = "UNKNOWN_SOURCE_IDENTITY"
	DisqualifierInvalidProvenance       HardDisqualifier = "INVALID_OR_MISSING_PROVENANCE"
	DisqualifierLicensingProhibitsUse   HardDisqualifier = "LICENSING_PROHIBITS_INTENDED_USE"
	DisqualifierCapabilityMismatch      HardDisqualifier = "CAPABILITY_ROLE_MISMATCH"
	DisqualifierInsufficientIdentity    HardDisqualifier = "INSUFFICIENT_IDENTITY_CERTAINTY"
	DisqualifierUnsupportedCoverage     HardDisqualifier = "UNSUPPORTED_OR_UNKNOWN_COVERAGE"
	DisqualifierModelDerivedAuthority   HardDisqualifier = "MODEL_DERIVED_AUTHORITY_PROHIBITED"
	DisqualifierRequiredEvidenceMissing HardDisqualifier = "REQUIRED_QUALIFICATION_EVIDENCE_MISSING"
	DisqualifierInsufficientReliability HardDisqualifier = "INSUFFICIENT_HISTORICAL_RELIABILITY"
	DisqualifierUnstableAccess          HardDisqualifier = "UNSTABLE_ACCESS_CHARACTERISTICS"
	DisqualifierUnstableSchema          HardDisqualifier = "UNSTABLE_SCHEMA_CHARACTERISTICS"
	DisqualifierQualificationExpired    HardDisqualifier = "QUALIFICATION_EXPIRED"
	DisqualifierReviewOverdue           HardDisqualifier = "QUALIFICATION_REVIEW_OVERDUE"
	DisqualifierSourceDisabled          HardDisqualifier = "EXPLICIT_SOURCE_DISABLEMENT"
	DisqualifierIntendedUseProhibited   HardDisqualifier = "INTENDED_USE_PROHIBITED"
	DisqualifierConflictingConditions   HardDisqualifier = "CONFLICTING_CONDITIONS"
)

type QualificationScope struct {
	Source       canonical.SourceIdentity    `json:"source"`
	ProviderPath *canonical.ProviderIdentity `json:"provider_path,omitempty"`
	CapabilityID CapabilityID                `json:"capability_id"`
	Role         QualificationRole           `json:"role"`
	IntendedUse  QualificationUse            `json:"intended_use"`
	Coverage     QualificationCoverage       `json:"coverage"`
}

type QualificationReasonCode string

const (
	ReasonPolicyRequirementsSatisfied QualificationReasonCode = "policy_requirements_satisfied"
	ReasonConditionalRestrictions     QualificationReasonCode = "conditional_restrictions_apply"
	ReasonHardDisqualifier            QualificationReasonCode = "hard_disqualifier_present"
	ReasonEvidenceIncomplete          QualificationReasonCode = "qualification_evidence_incomplete"
	ReasonExplicitlyDisabled          QualificationReasonCode = "source_explicitly_disabled"
)

type QualificationReason struct {
	Code        QualificationReasonCode `json:"code"`
	EvidenceIDs []string                `json:"evidence_ids,omitempty"`
}

type QualificationPolicy struct {
	ContractVersion           canonical.ContractVersion   `json:"contract_version"`
	Identity                  canonical.ComponentIdentity `json:"identity"`
	DecisionVersion           canonical.VersionIdentity   `json:"decision_version"`
	RequiredEvidence          []QualificationEvidenceKind `json:"required_evidence"`
	MinimumIdentity           IdentityCertainty           `json:"minimum_identity_certainty"`
	MinimumReliability        ReliabilityClass            `json:"minimum_reliability"`
	RequireKnownCoverage      bool                        `json:"require_known_coverage"`
	RequireKnownCost          bool                        `json:"require_known_cost"`
	RequireKnownAvailability  bool                        `json:"require_known_availability"`
	RequireKnownUpdateLatency bool                        `json:"require_known_update_latency"`
	RequireStableAccess       bool                        `json:"require_stable_access"`
	RequireManagedSchema      bool                        `json:"require_managed_schema"`
	RequirePersistentStore    bool                        `json:"require_persistent_storage"`
	RequireDerivedData        bool                        `json:"require_derived_data"`
	RequireResearchUse        bool                        `json:"require_research_use"`
	MaximumReviewInterval     time.Duration               `json:"maximum_review_interval"`
}

type QualificationAssessmentInput struct {
	ID                    QualificationID
	Scope                 QualificationScope
	AuthorityClass        SourceAuthorityClass
	Dimensions            QualificationDimensions
	Permissions           QualificationPermissions
	Conditions            []QualificationCondition
	Evidence              []QualificationEvidence
	ExplicitDisqualifiers []HardDisqualifier
	Disabled              bool
	Assessor              canonical.ComponentIdentity
	AssessedAt            time.Time
	EffectiveFrom         time.Time
	EffectiveTo           *time.Time
	ReviewDueAt           *time.Time
	Supersedes            *QualificationID
}

// QualificationDecision is immutable governance evidence. Runtime health and
// datum freshness are intentionally absent.
type QualificationDecision struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	ID              QualificationID             `json:"id"`
	DecisionVersion canonical.VersionIdentity   `json:"decision_version"`
	Scope           QualificationScope          `json:"scope"`
	AuthorityClass  SourceAuthorityClass        `json:"authority_class"`
	State           QualificationState          `json:"state"`
	Dimensions      QualificationDimensions     `json:"dimensions"`
	Permissions     QualificationPermissions    `json:"permissions"`
	Conditions      []QualificationCondition    `json:"conditions,omitempty"`
	Evidence        []QualificationEvidence     `json:"evidence"`
	Disqualifiers   []HardDisqualifier          `json:"disqualifiers,omitempty"`
	Reasons         []QualificationReason       `json:"reasons"`
	Policy          canonical.ComponentIdentity `json:"policy"`
	Assessor        canonical.ComponentIdentity `json:"assessor"`
	AssessedAt      time.Time                   `json:"assessed_at"`
	EffectiveFrom   time.Time                   `json:"effective_from"`
	EffectiveTo     *time.Time                  `json:"effective_to,omitempty"`
	ReviewDueAt     *time.Time                  `json:"review_due_at,omitempty"`
	Supersedes      *QualificationID            `json:"supersedes,omitempty"`
}

// AssessQualification deterministically applies one exact policy version. It
// does not call a provider, clock, model, database, or ranking mechanism.
func AssessQualification(policy QualificationPolicy, input QualificationAssessmentInput) (QualificationDecision, error) {
	if err := policy.Validate(); err != nil {
		return QualificationDecision{}, err
	}
	decision := QualificationDecision{
		ContractVersion: QualificationDecisionContractV1, ID: input.ID, DecisionVersion: policy.DecisionVersion,
		Scope: input.Scope, AuthorityClass: input.AuthorityClass, Dimensions: input.Dimensions,
		Permissions: input.Permissions, Conditions: cloneQualificationConditions(input.Conditions),
		Evidence: cloneQualificationEvidence(input.Evidence), Policy: cloneComponentIdentity(policy.Identity),
		Assessor: cloneComponentIdentity(input.Assessor), AssessedAt: input.AssessedAt,
		EffectiveFrom: input.EffectiveFrom, EffectiveTo: cloneTime(input.EffectiveTo), ReviewDueAt: cloneTime(input.ReviewDueAt),
		Supersedes: cloneQualificationID(input.Supersedes),
	}
	disqualifiers := append([]HardDisqualifier(nil), input.ExplicitDisqualifiers...)
	if input.Disabled {
		disqualifiers = append(disqualifiers, DisqualifierSourceDisabled)
	}
	disqualifiers = append(disqualifiers, deriveQualificationDisqualifiers(policy, decision)...)
	decision.Disqualifiers = uniqueSortedDisqualifiers(disqualifiers)
	decision.State = qualificationState(decision.Permissions.For(decision.Scope.IntendedUse), decision.Conditions, decision.Scope.IntendedUse, decision.Disqualifiers)
	decision.Reasons = qualificationReasons(decision)
	if err := decision.Validate(); err != nil {
		return QualificationDecision{}, err
	}
	return decision, nil
}

func (policy QualificationPolicy) Validate() error {
	if policy.ContractVersion != QualificationPolicyContractV1 {
		return fmt.Errorf("qualification policy: unsupported contract version")
	}
	if err := policy.Identity.Validate(); err != nil || policy.Identity.Kind != canonical.ComponentKindPolicy || policy.Identity.Provider != nil || policy.Identity.Version.Namespace != "jax.policy.qualification" {
		return fmt.Errorf("qualification policy: invalid policy identity")
	}
	if err := policy.DecisionVersion.Validate(); err != nil || policy.DecisionVersion != QualificationDecisionVersionV1() {
		return fmt.Errorf("qualification policy: unsupported decision version")
	}
	if identityCertaintyRank(policy.MinimumIdentity) < 0 {
		return fmt.Errorf("qualification policy: unsupported minimum identity certainty")
	}
	if reliabilityRank(policy.MinimumReliability) < 0 {
		return fmt.Errorf("qualification policy: unsupported minimum reliability")
	}
	if len(policy.RequiredEvidence) == 0 {
		return fmt.Errorf("qualification policy: required evidence must not be empty")
	}
	seen := map[QualificationEvidenceKind]bool{}
	for _, kind := range policy.RequiredEvidence {
		if !supportedQualificationEvidenceKind(kind) || seen[kind] {
			return fmt.Errorf("qualification policy: invalid or duplicate required evidence")
		}
		seen[kind] = true
	}
	if policy.MaximumReviewInterval <= 0 || policy.MaximumReviewInterval > 5*365*24*time.Hour {
		return fmt.Errorf("qualification policy: maximum review interval must be in (0,5y]")
	}
	return nil
}

func (decision QualificationDecision) Validate() error {
	if decision.ContractVersion != QualificationDecisionContractV1 {
		return fmt.Errorf("qualification decision: unsupported contract version")
	}
	if err := validateQualificationID(decision.ID); err != nil {
		return err
	}
	if err := decision.DecisionVersion.Validate(); err != nil || decision.DecisionVersion != QualificationDecisionVersionV1() {
		return fmt.Errorf("qualification decision: unsupported decision version")
	}
	if err := decision.Scope.Validate(); err != nil {
		return err
	}
	if !supportedAuthorityClass(decision.AuthorityClass) {
		return fmt.Errorf("qualification decision: unsupported authority class")
	}
	if err := decision.Dimensions.Validate(); err != nil {
		return err
	}
	if err := validatePermissions(decision.Permissions); err != nil {
		return err
	}
	if err := validateConditions(decision.Conditions); err != nil {
		return err
	}
	if err := validateQualificationEvidence(decision.Evidence); err != nil {
		return err
	}
	if err := validateDisqualifiers(decision.Disqualifiers); err != nil {
		return err
	}
	if err := decision.Policy.Validate(); err != nil || decision.Policy.Kind != canonical.ComponentKindPolicy || decision.Policy.Version.Namespace != "jax.policy.qualification" {
		return fmt.Errorf("qualification decision: invalid policy identity")
	}
	if err := decision.Assessor.Validate(); err != nil || (decision.Assessor.Kind != canonical.ComponentKindValidator && decision.Assessor.Kind != canonical.ComponentKindMethod) || decision.Assessor.Version.Namespace != "jax.assessment.qualification" {
		return fmt.Errorf("qualification decision: invalid assessor identity")
	}
	if !validOperationalTime(decision.AssessedAt) || !validOperationalTime(decision.EffectiveFrom) || decision.EffectiveFrom.Before(decision.AssessedAt) {
		return fmt.Errorf("qualification decision: assessment/effective times must use UTC and effective_from must not precede assessed_at")
	}
	if decision.EffectiveTo != nil && (!validOperationalTime(*decision.EffectiveTo) || !decision.EffectiveTo.After(decision.EffectiveFrom)) {
		return fmt.Errorf("qualification decision: effective_to must use UTC and follow effective_from")
	}
	if decision.ReviewDueAt != nil && (!validOperationalTime(*decision.ReviewDueAt) || !decision.ReviewDueAt.After(decision.AssessedAt)) {
		return fmt.Errorf("qualification decision: review_due_at must use UTC and follow assessed_at")
	}
	if decision.Supersedes != nil {
		if err := validateQualificationID(*decision.Supersedes); err != nil || *decision.Supersedes == decision.ID {
			return fmt.Errorf("qualification decision: supersedes must identify another qualification")
		}
	}
	wantState := qualificationState(decision.Permissions.For(decision.Scope.IntendedUse), decision.Conditions, decision.Scope.IntendedUse, decision.Disqualifiers)
	if decision.State != wantState {
		return fmt.Errorf("qualification decision: state %q does not match deterministic result %q", decision.State, wantState)
	}
	if err := validateQualificationReasons(decision); err != nil {
		return err
	}
	return nil
}

func (scope QualificationScope) Validate() error {
	if err := scope.Source.Validate(); err != nil {
		return fmt.Errorf("qualification scope: invalid source identity")
	}
	if scope.ProviderPath != nil {
		if err := scope.ProviderPath.Validate(); err != nil {
			return fmt.Errorf("qualification scope: invalid provider identity")
		}
	}
	if _, _, ok := capabilitySpecification(scope.CapabilityID); !ok {
		return fmt.Errorf("qualification scope: unsupported capability")
	}
	if !supportedQualificationRole(scope.Role) || !supportedQualificationUse(scope.IntendedUse) {
		return fmt.Errorf("qualification scope: unsupported role or intended use")
	}
	if err := scope.Coverage.Validate(); err != nil {
		return err
	}
	return nil
}

func (coverage QualificationCoverage) Validate() error {
	switch coverage.Status {
	case CoverageUnknown:
		if coverage.Scope != CoverageScopeUnknown || len(coverage.Bounds) != 0 {
			return fmt.Errorf("qualification coverage: UNKNOWN requires unknown scope and no bounds")
		}
	case CoverageUnsupported:
		if coverage.Scope == CoverageScopeUnknown || len(coverage.Bounds) != 0 && coverage.Scope != CoverageScopeBounded {
			return fmt.Errorf("qualification coverage: unsupported coverage scope is inconsistent")
		}
	case CoverageSupported:
		switch coverage.Scope {
		case CoverageScopeCapabilityWide:
			if len(coverage.Bounds) != 0 {
				return fmt.Errorf("qualification coverage: capability-wide scope must not have bounds")
			}
		case CoverageScopeBounded:
			if len(coverage.Bounds) == 0 {
				return fmt.Errorf("qualification coverage: bounded scope requires bounds")
			}
		default:
			return fmt.Errorf("qualification coverage: supported scope is invalid")
		}
	default:
		return fmt.Errorf("qualification coverage: unsupported status")
	}
	previous := ""
	for _, bound := range coverage.Bounds {
		if !supportedCoverageDimension(bound.Dimension) || validateCode("qualification_coverage", "bound.value", bound.Value) != nil {
			return fmt.Errorf("qualification coverage: invalid bound")
		}
		key := string(bound.Dimension) + "\x00" + bound.Value
		if previous != "" && key <= previous {
			return fmt.Errorf("qualification coverage: bounds must be unique and sorted")
		}
		previous = key
	}
	return nil
}

func (dimensions QualificationDimensions) Validate() error {
	if identityCertaintyRank(dimensions.IdentityCertainty) < 0 {
		return fmt.Errorf("qualification dimensions: unsupported identity certainty")
	}
	if err := dimensions.Reliability.Validate(); err != nil {
		return err
	}
	switch dimensions.Availability {
	case AvailabilityUnknown, AvailabilityContinuousExpected, AvailabilityScheduled, AvailabilityBestEffort:
	default:
		return fmt.Errorf("qualification dimensions: unsupported availability")
	}
	switch dimensions.UpdateLatency {
	case UpdateLatencyUnknown, UpdateLatencyRealTime, UpdateLatencyDelayed, UpdateLatencyPeriodic, UpdateLatencyEventDriven:
	default:
		return fmt.Errorf("qualification dimensions: unsupported update latency")
	}
	switch dimensions.ProvenanceQuality {
	case ProvenanceQualityUnknown, ProvenanceQualityInsufficient, ProvenanceQualitySourceIdentified, ProvenanceQualitySourceRevision, ProvenanceQualityImmutableRawAndLineage:
	default:
		return fmt.Errorf("qualification dimensions: unsupported provenance quality")
	}
	if err := dimensions.Rights.Validate(); err != nil {
		return err
	}
	if err := dimensions.Cost.Validate(); err != nil {
		return err
	}
	switch dimensions.AccessStability {
	case AccessStabilityUnknown, AccessStabilityUnstable, AccessStabilityVariable, AccessStabilityStable:
	default:
		return fmt.Errorf("qualification dimensions: unsupported access stability")
	}
	switch dimensions.SchemaStability {
	case SchemaStabilityUnknown, SchemaStabilityUnstable, SchemaStabilityManaged, SchemaStabilityStable:
	default:
		return fmt.Errorf("qualification dimensions: unsupported schema stability")
	}
	return dimensions.Coverage.Validate()
}

func (reliability HistoricalReliability) Validate() error {
	switch reliability.Class {
	case ReliabilityUnknown:
		if reliability.WindowFrom != nil || reliability.WindowTo != nil || reliability.DeliverySuccessBasisPoints != nil || reliability.MissingDataBasisPoints != nil || reliability.CorrectionBasisPoints != nil || reliability.Outages != OutageUnknown || reliability.SchemaChanges != SchemaChangeUnknown {
			return fmt.Errorf("historical reliability: UNKNOWN must not claim observations")
		}
	case ReliabilityInsufficient, ReliabilityLimited, ReliabilityAcceptable, ReliabilityStrong:
		if reliability.WindowFrom == nil || reliability.WindowTo == nil || !validOperationalTime(*reliability.WindowFrom) || !validOperationalTime(*reliability.WindowTo) || !reliability.WindowTo.After(*reliability.WindowFrom) {
			return fmt.Errorf("historical reliability: assessed class requires a valid UTC observation window")
		}
		for _, value := range []*int{reliability.DeliverySuccessBasisPoints, reliability.MissingDataBasisPoints, reliability.CorrectionBasisPoints} {
			if value != nil && (*value < 0 || *value > 10000) {
				return fmt.Errorf("historical reliability: rate must be in [0,10000] basis points")
			}
		}
		switch reliability.Outages {
		case OutageUnknown, OutageRareBounded, OutageOccasional, OutageFrequent, OutageUnbounded:
		default:
			return fmt.Errorf("historical reliability: unsupported outage characteristic")
		}
		switch reliability.SchemaChanges {
		case SchemaChangeUnknown, SchemaChangeRare, SchemaChangeOccasional, SchemaChangeFrequent:
		default:
			return fmt.Errorf("historical reliability: unsupported schema-change frequency")
		}
	default:
		return fmt.Errorf("historical reliability: unsupported class")
	}
	return nil
}

func (rights UsageRights) Validate() error {
	for _, right := range []UseRight{rights.InternalProcessing, rights.PersistentStorage, rights.DerivedData, rights.ModelResearchUse, rights.Redistribution, rights.CommercialUse} {
		switch right {
		case UseRightUnknown, UseRightAllowed, UseRightRestricted, UseRightProhibited:
		default:
			return fmt.Errorf("qualification rights: unsupported use right")
		}
	}
	switch rights.Retention {
	case RetentionRightUnknown, RetentionRightIndefinite, RetentionRightBounded, RetentionRightSessionOnly, RetentionRightProhibited:
	default:
		return fmt.Errorf("qualification rights: unsupported retention right")
	}
	return nil
}

func (cost CostCharacteristics) Validate() error {
	switch cost.Model {
	case CostModelFree, CostModelFixedSubscription, CostModelUsageBased, CostModelTiered, CostModelInternal, CostModelUnknown:
	default:
		return fmt.Errorf("qualification cost: unsupported model")
	}
	switch cost.Class {
	case CostClassNegligible, CostClassLow, CostClassModerate, CostClassHigh, CostClassUnknown:
	default:
		return fmt.Errorf("qualification cost: unsupported class")
	}
	if (cost.Model == CostModelUnknown) != (cost.Class == CostClassUnknown) {
		return fmt.Errorf("qualification cost: model and class must both be known or both unknown")
	}
	return nil
}

func deriveQualificationDisqualifiers(policy QualificationPolicy, decision QualificationDecision) []HardDisqualifier {
	result := []HardDisqualifier{}
	evidenceKinds := map[QualificationEvidenceKind]bool{}
	for _, evidence := range decision.Evidence {
		evidenceKinds[evidence.Kind] = true
	}
	for _, required := range policy.RequiredEvidence {
		if !evidenceKinds[required] {
			result = append(result, DisqualifierRequiredEvidenceMissing)
		}
	}
	if decision.Dimensions.Reliability.Class != ReliabilityUnknown && !evidenceKinds[EvidenceHistoricalReliability] {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	}
	if identityCertaintyRank(decision.Dimensions.IdentityCertainty) < identityCertaintyRank(policy.MinimumIdentity) {
		result = append(result, DisqualifierInsufficientIdentity)
	}
	if decision.Dimensions.Reliability.Class == ReliabilityUnknown && policy.MinimumReliability != ReliabilityUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	} else if reliabilityRank(decision.Dimensions.Reliability.Class) < reliabilityRank(policy.MinimumReliability) {
		result = append(result, DisqualifierInsufficientReliability)
	}
	if decision.AuthorityClass == SourceAuthorityUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	}
	if decision.Dimensions.ProvenanceQuality == ProvenanceQualityUnknown || decision.Dimensions.ProvenanceQuality == ProvenanceQualityInsufficient {
		result = append(result, DisqualifierInvalidProvenance)
	}
	if policy.RequireKnownCoverage && decision.Scope.Coverage.Status != CoverageSupported {
		result = append(result, DisqualifierUnsupportedCoverage)
	}
	if !reflect.DeepEqual(decision.Scope.Coverage, decision.Dimensions.Coverage) {
		result = append(result, DisqualifierUnsupportedCoverage)
	}
	if policy.RequireKnownCost && decision.Dimensions.Cost.Model == CostModelUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	}
	if policy.RequireKnownAvailability && decision.Dimensions.Availability == AvailabilityUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	}
	if policy.RequireKnownUpdateLatency && decision.Dimensions.UpdateLatency == UpdateLatencyUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	}
	if policy.RequireStableAccess && decision.Dimensions.AccessStability == AccessStabilityUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	} else if policy.RequireStableAccess && decision.Dimensions.AccessStability == AccessStabilityUnstable {
		result = append(result, DisqualifierUnstableAccess)
	}
	if policy.RequireManagedSchema && decision.Dimensions.SchemaStability == SchemaStabilityUnknown {
		result = append(result, DisqualifierRequiredEvidenceMissing)
	} else if policy.RequireManagedSchema && decision.Dimensions.SchemaStability == SchemaStabilityUnstable {
		result = append(result, DisqualifierUnstableSchema)
	}
	if !roleCapabilityCompatible(decision.Scope.Role, decision.Scope.CapabilityID, decision.Scope.IntendedUse) {
		result = append(result, DisqualifierCapabilityMismatch)
	}
	if authorityUseProhibited(decision.AuthorityClass, decision.Scope.Role, decision.Scope.IntendedUse) {
		if decision.AuthorityClass == SourceAuthorityModelDerived {
			result = append(result, DisqualifierModelDerivedAuthority)
		} else {
			result = append(result, DisqualifierCapabilityMismatch)
		}
	}
	rights := decision.Dimensions.Rights
	if rightBlocks(rights.InternalProcessing) || policy.RequirePersistentStore && (rightBlocks(rights.PersistentStorage) || rights.Retention == RetentionRightUnknown || rights.Retention == RetentionRightSessionOnly || rights.Retention == RetentionRightProhibited) || policy.RequireDerivedData && rightBlocks(rights.DerivedData) || policy.RequireResearchUse && rightBlocks(rights.ModelResearchUse) {
		result = append(result, DisqualifierLicensingProhibitsUse)
	}
	permission := decision.Permissions.For(decision.Scope.IntendedUse)
	if requiredRightsRestricted(policy, rights) && (permission != PermissionConditional || !hasConditionTypeFor(decision.Conditions, decision.Scope.IntendedUse, ConditionUsageRestrictionsApply)) {
		result = append(result, DisqualifierConflictingConditions)
	}
	if policy.RequirePersistentStore && rights.Retention == RetentionRightBounded && (permission != PermissionConditional || !hasConditionTypeFor(decision.Conditions, decision.Scope.IntendedUse, ConditionRetentionLimitApplies)) {
		result = append(result, DisqualifierConflictingConditions)
	}
	if permission == PermissionProhibited {
		result = append(result, DisqualifierIntendedUseProhibited)
	}
	if conditionsConflict(decision.Conditions) || permission == PermissionConditional && !hasConditionFor(decision.Conditions, decision.Scope.IntendedUse) || permission == PermissionAllowed && hasConditionFor(decision.Conditions, decision.Scope.IntendedUse) {
		result = append(result, DisqualifierConflictingConditions)
	}
	return result
}

func qualificationState(permission PermissionState, conditions []QualificationCondition, use QualificationUse, disqualifiers []HardDisqualifier) QualificationState {
	if containsDisqualifier(disqualifiers, DisqualifierSourceDisabled) {
		return QualificationDisabled
	}
	if len(disqualifiers) > 0 {
		onlyMissing := true
		for _, item := range disqualifiers {
			if item != DisqualifierRequiredEvidenceMissing {
				onlyMissing = false
				break
			}
		}
		if onlyMissing {
			return QualificationNotAssessed
		}
		return QualificationUnqualified
	}
	switch permission {
	case PermissionAllowed:
		return QualificationQualified
	case PermissionConditional:
		if hasConditionFor(conditions, use) {
			return QualificationConditionallyQualified
		}
		return QualificationUnqualified
	case PermissionProhibited:
		return QualificationUnqualified
	default:
		return QualificationNotAssessed
	}
}

func qualificationReasons(decision QualificationDecision) []QualificationReason {
	evidenceIDs := make([]string, 0, len(decision.Evidence))
	for _, evidence := range decision.Evidence {
		evidenceIDs = append(evidenceIDs, evidence.Ref.Evidence.ID)
	}
	sort.Strings(evidenceIDs)
	code := ReasonPolicyRequirementsSatisfied
	switch decision.State {
	case QualificationConditionallyQualified:
		code = ReasonConditionalRestrictions
	case QualificationNotAssessed:
		code = ReasonEvidenceIncomplete
	case QualificationDisabled:
		code = ReasonExplicitlyDisabled
	case QualificationUnqualified:
		code = ReasonHardDisqualifier
	}
	return []QualificationReason{{Code: code, EvidenceIDs: evidenceIDs}}
}

func validateQualificationReasons(decision QualificationDecision) error {
	if len(decision.Reasons) == 0 {
		return fmt.Errorf("qualification decision: at least one typed reason is required")
	}
	knownEvidence := map[string]bool{}
	for _, evidence := range decision.Evidence {
		knownEvidence[evidence.Ref.Evidence.ID] = true
	}
	for _, reason := range decision.Reasons {
		switch reason.Code {
		case ReasonPolicyRequirementsSatisfied, ReasonConditionalRestrictions, ReasonHardDisqualifier, ReasonEvidenceIncomplete, ReasonExplicitlyDisabled:
		default:
			return fmt.Errorf("qualification decision: unsupported reason code")
		}
		for _, id := range reason.EvidenceIDs {
			if !knownEvidence[id] {
				return fmt.Errorf("qualification decision: reason cites unknown evidence")
			}
		}
	}
	return nil
}

func validateQualificationEvidence(values []QualificationEvidence) error {
	seen := map[string]bool{}
	for _, evidence := range values {
		if !supportedQualificationEvidenceKind(evidence.Kind) || evidence.Ref.Validate() != nil {
			return fmt.Errorf("qualification decision: invalid immutable evidence")
		}
		key := string(evidence.Kind) + "\x00" + evidence.Ref.Evidence.ID
		if seen[key] {
			return fmt.Errorf("qualification decision: duplicate evidence")
		}
		seen[key] = true
	}
	return nil
}

func validatePermissions(permissions QualificationPermissions) error {
	for _, value := range []PermissionState{permissions.PrimaryEvidence, permissions.Corroboration, permissions.InvestigationTrigger, permissions.ResearchDisplay, permissions.FallbackInput, permissions.CanonicalFactualAssertion} {
		switch value {
		case PermissionAllowed, PermissionConditional, PermissionProhibited, PermissionNotAssessed:
		default:
			return fmt.Errorf("qualification decision: unsupported permission state")
		}
	}
	return nil
}

func validateConditions(values []QualificationCondition) error {
	seen := map[string]bool{}
	for _, condition := range values {
		switch condition.Type {
		case ConditionIndependentPrimaryRequired, ConditionResearchDisplayOnly, ConditionCoverageRestricted, ConditionUsageRestrictionsApply, ConditionRetentionLimitApplies, ConditionFreshnessPolicyRequired:
		default:
			return fmt.Errorf("qualification decision: unsupported condition")
		}
		if !supportedQualificationUse(condition.AppliesTo) {
			return fmt.Errorf("qualification decision: condition has unsupported use")
		}
		if condition.Policy != nil {
			if err := condition.Policy.Validate(); err != nil || condition.Policy.Kind != canonical.ComponentKindPolicy {
				return fmt.Errorf("qualification decision: condition policy is invalid")
			}
		}
		key := string(condition.Type) + "\x00" + string(condition.AppliesTo)
		if seen[key] {
			return fmt.Errorf("qualification decision: duplicate condition")
		}
		seen[key] = true
	}
	return nil
}

func validateDisqualifiers(values []HardDisqualifier) error {
	previous := HardDisqualifier("")
	for _, value := range values {
		if !supportedDisqualifier(value) || previous != "" && value <= previous {
			return fmt.Errorf("qualification decision: disqualifiers must be supported, unique, and sorted")
		}
		previous = value
	}
	return nil
}

func validateQualificationID(id QualificationID) error {
	value := string(id)
	if len(value) <= len("qlf_") || value[:len("qlf_")] != "qlf_" || validateCode("qualification_decision", "id", value) != nil {
		return fmt.Errorf("qualification decision: id must use the qlf_ prefix")
	}
	return nil
}

func roleCapabilityCompatible(role QualificationRole, capability CapabilityID, use QualificationUse) bool {
	switch role {
	case QualificationRoleMacroeconomicAuthority:
		return capability == CapabilityMacroObservation || capability == CapabilityEconomicCalendar
	case QualificationRoleFilingDocument:
		return capability == CapabilityCorporateFiling
	case QualificationRoleNewsEvidence:
		return capability == CapabilityNewsArticle
	case QualificationRoleEventDetection:
		return capability == CapabilityEventFeed || capability == CapabilityNewsArticle
	case QualificationRoleReferenceData:
		return capability == CapabilityInstrumentReference
	case QualificationRoleMarketObservation:
		return capability == CapabilityMarketQuote || capability == CapabilityMarketBars || capability == CapabilityMarketTrades || capability == CapabilityFundamentalObservation
	case QualificationRoleSecondaryCorroboration:
		return use != QualificationUsePrimaryEvidence && use != QualificationUseCanonicalFactualAssertion
	default:
		return true
	}
}

func authorityUseProhibited(authority SourceAuthorityClass, role QualificationRole, use QualificationUse) bool {
	primaryUse := use == QualificationUsePrimaryEvidence || use == QualificationUseCanonicalFactualAssertion || role == QualificationRoleAuthoritativeSource || role == QualificationRolePrimaryEvidence
	if !primaryUse {
		return false
	}
	switch authority {
	case SourceAuthorityOfficialPrimary, SourceAuthorityRegulatedExchange, SourceAuthorityDirectIssuer, SourceAuthorityProfessionalSecondary:
		return false
	default:
		return true
	}
}

func rightBlocks(right UseRight) bool { return right == UseRightUnknown || right == UseRightProhibited }

func requiredRightsRestricted(policy QualificationPolicy, rights UsageRights) bool {
	return rights.InternalProcessing == UseRightRestricted ||
		policy.RequirePersistentStore && rights.PersistentStorage == UseRightRestricted ||
		policy.RequireDerivedData && rights.DerivedData == UseRightRestricted ||
		policy.RequireResearchUse && rights.ModelResearchUse == UseRightRestricted
}

func hasConditionFor(values []QualificationCondition, use QualificationUse) bool {
	for _, value := range values {
		if value.AppliesTo == use {
			return true
		}
	}
	return false
}

func hasConditionTypeFor(values []QualificationCondition, use QualificationUse, kind QualificationConditionType) bool {
	for _, value := range values {
		if value.AppliesTo == use && value.Type == kind {
			return true
		}
	}
	return false
}

func conditionsConflict(values []QualificationCondition) bool {
	for _, left := range values {
		for _, right := range values {
			if left.AppliesTo == right.AppliesTo && left.Type == ConditionResearchDisplayOnly && right.Type == ConditionIndependentPrimaryRequired {
				return true
			}
		}
	}
	return false
}

func identityCertaintyRank(value IdentityCertainty) int {
	switch value {
	case IdentityCertaintyUnknown:
		return 0
	case IdentityCertaintyLow:
		return 1
	case IdentityCertaintyModerate:
		return 2
	case IdentityCertaintyHigh:
		return 3
	case IdentityCertaintyVerified:
		return 4
	default:
		return -1
	}
}

func reliabilityRank(value ReliabilityClass) int {
	switch value {
	case ReliabilityUnknown:
		return 0
	case ReliabilityInsufficient:
		return 1
	case ReliabilityLimited:
		return 2
	case ReliabilityAcceptable:
		return 3
	case ReliabilityStrong:
		return 4
	default:
		return -1
	}
}

func supportedAuthorityClass(value SourceAuthorityClass) bool {
	switch value {
	case SourceAuthorityOfficialPrimary, SourceAuthorityRegulatedExchange, SourceAuthorityDirectIssuer, SourceAuthorityProfessionalSecondary, SourceAuthorityAggregator, SourceAuthorityCommunityUnverified, SourceAuthorityModelDerived, SourceAuthorityOther, SourceAuthorityUnknown:
		return true
	default:
		return false
	}
}

func supportedQualificationRole(value QualificationRole) bool {
	switch value {
	case QualificationRoleAuthoritativeSource, QualificationRolePrimaryEvidence, QualificationRoleSecondaryCorroboration, QualificationRoleMarketObservation, QualificationRoleReferenceData, QualificationRoleFilingDocument, QualificationRoleMacroeconomicAuthority, QualificationRoleNewsEvidence, QualificationRoleEventDetection, QualificationRoleResearchSupplementary:
		return true
	default:
		return false
	}
}

func supportedQualificationUse(value QualificationUse) bool {
	switch value {
	case QualificationUsePrimaryEvidence, QualificationUseCorroboration, QualificationUseInvestigationTrigger, QualificationUseResearchDisplay, QualificationUseFallbackInput, QualificationUseCanonicalFactualAssertion:
		return true
	default:
		return false
	}
}

func supportedCoverageDimension(value CoverageDimension) bool {
	switch value {
	case CoverageDimensionGeography, CoverageDimensionInstrumentType, CoverageDimensionVenue, CoverageDimensionSeries, CoverageDimensionDataset:
		return true
	default:
		return false
	}
}

func supportedQualificationEvidenceKind(value QualificationEvidenceKind) bool {
	switch value {
	case EvidenceSourceIdentity, EvidenceAuthorityClassification, EvidenceSourceDocumentation, EvidenceLicensingReview, EvidenceHistoricalReliability, EvidenceCoverageReview, EvidenceApprovedInternalReview:
		return true
	default:
		return false
	}
}

func supportedDisqualifier(value HardDisqualifier) bool {
	switch value {
	case DisqualifierUnknownSourceIdentity, DisqualifierInvalidProvenance, DisqualifierLicensingProhibitsUse, DisqualifierCapabilityMismatch, DisqualifierInsufficientIdentity, DisqualifierUnsupportedCoverage, DisqualifierModelDerivedAuthority, DisqualifierRequiredEvidenceMissing, DisqualifierInsufficientReliability, DisqualifierUnstableAccess, DisqualifierUnstableSchema, DisqualifierQualificationExpired, DisqualifierReviewOverdue, DisqualifierSourceDisabled, DisqualifierIntendedUseProhibited, DisqualifierConflictingConditions:
		return true
	default:
		return false
	}
}

func uniqueSortedDisqualifiers(values []HardDisqualifier) []HardDisqualifier {
	if len(values) == 0 {
		return nil
	}
	seen := map[HardDisqualifier]bool{}
	out := make([]HardDisqualifier, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func containsDisqualifier(values []HardDisqualifier, target HardDisqualifier) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneQualificationDecision(value QualificationDecision) QualificationDecision {
	copyValue := value
	copyValue.Scope = cloneQualificationScope(value.Scope)
	copyValue.Dimensions = cloneQualificationDimensions(value.Dimensions)
	copyValue.Conditions = cloneQualificationConditions(value.Conditions)
	copyValue.Evidence = cloneQualificationEvidence(value.Evidence)
	copyValue.Disqualifiers = append([]HardDisqualifier(nil), value.Disqualifiers...)
	copyValue.Reasons = make([]QualificationReason, len(value.Reasons))
	for i, reason := range value.Reasons {
		copyValue.Reasons[i] = reason
		copyValue.Reasons[i].EvidenceIDs = append([]string(nil), reason.EvidenceIDs...)
	}
	copyValue.Policy = cloneComponentIdentity(value.Policy)
	copyValue.Assessor = cloneComponentIdentity(value.Assessor)
	copyValue.EffectiveTo = cloneTime(value.EffectiveTo)
	copyValue.ReviewDueAt = cloneTime(value.ReviewDueAt)
	copyValue.Supersedes = cloneQualificationID(value.Supersedes)
	return copyValue
}

func cloneQualificationScope(value QualificationScope) QualificationScope {
	copyValue := value
	if value.ProviderPath != nil {
		provider := cloneProviderIdentity(*value.ProviderPath)
		copyValue.ProviderPath = &provider
	}
	copyValue.Coverage.Bounds = append([]CoverageBound(nil), value.Coverage.Bounds...)
	return copyValue
}

func cloneQualificationDimensions(value QualificationDimensions) QualificationDimensions {
	copyValue := value
	copyValue.Coverage.Bounds = append([]CoverageBound(nil), value.Coverage.Bounds...)
	copyValue.Reliability.WindowFrom = cloneTime(value.Reliability.WindowFrom)
	copyValue.Reliability.WindowTo = cloneTime(value.Reliability.WindowTo)
	copyValue.Reliability.DeliverySuccessBasisPoints = cloneInt(value.Reliability.DeliverySuccessBasisPoints)
	copyValue.Reliability.MissingDataBasisPoints = cloneInt(value.Reliability.MissingDataBasisPoints)
	copyValue.Reliability.CorrectionBasisPoints = cloneInt(value.Reliability.CorrectionBasisPoints)
	return copyValue
}

func cloneQualificationConditions(values []QualificationCondition) []QualificationCondition {
	if len(values) == 0 {
		return nil
	}
	out := make([]QualificationCondition, len(values))
	for i, value := range values {
		out[i] = value
		if value.Policy != nil {
			policy := cloneComponentIdentity(*value.Policy)
			out[i].Policy = &policy
		}
	}
	return out
}

func cloneQualificationEvidence(values []QualificationEvidence) []QualificationEvidence {
	out := make([]QualificationEvidence, len(values))
	copy(out, values)
	for i := range out {
		if values[i].Ref.Provider != nil {
			provider := cloneProviderIdentity(*values[i].Ref.Provider)
			out[i].Ref.Provider = &provider
		}
		out[i].Ref.ObservedAt = cloneTime(values[i].Ref.ObservedAt)
		out[i].Ref.PublishedAt = cloneTime(values[i].Ref.PublishedAt)
	}
	return out
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneQualificationID(value *QualificationID) *QualificationID {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func sameQualificationScope(left, right QualificationScope) bool {
	return reflect.DeepEqual(left, right)
}
