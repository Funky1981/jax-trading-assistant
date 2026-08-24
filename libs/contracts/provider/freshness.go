package provider

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	FreshnessPolicyContractV1     canonical.ContractVersion = "jax.provider_freshness_policy/v1"
	LastKnownGoodPolicyContractV1 canonical.ContractVersion = "jax.provider_lkg_policy/v1"
	FreshnessEvaluationContractV1 canonical.ContractVersion = "jax.provider_freshness_evaluation/v1"
	FreshnessResolutionContractV1 canonical.ContractVersion = "jax.provider_freshness_resolution/v1"
)

// DataUseClass bounds the consumer intent that a temporal policy serves. It
// deliberately contains no approval, order, execution, or broker authority.
type DataUseClass string

const (
	DataUseResearch        DataUseClass = "RESEARCH"
	DataUseDisplay         DataUseClass = "DISPLAY"
	DataUseDecisionSupport DataUseClass = "DECISION_SUPPORT"
)

// FreshnessValidityMode describes how temporal validity ends. Age-bounded
// policies use explicit fresh and expiry horizons. Until-superseded policies
// never encode "forever" as a large duration. NOT_APPLICABLE is an explicit
// statement that age freshness is not the governing temporal dimension.
type FreshnessValidityMode string

const (
	FreshnessValidityAgeBounded      FreshnessValidityMode = "AGE_BOUNDED"
	FreshnessValidityUntilSuperseded FreshnessValidityMode = "UNTIL_SUPERSEDED"
	FreshnessValidityNotApplicable   FreshnessValidityMode = "NOT_APPLICABLE"
)

// AuthoritativeTimestampRole names the semantic clock used for age. CreatedAt
// and raw receipt time are intentionally absent. Collection time is available
// only when a policy explicitly declares it authoritative.
type AuthoritativeTimestampRole string

const (
	TimestampRoleNone          AuthoritativeTimestampRole = "NONE"
	TimestampRoleObservedAt    AuthoritativeTimestampRole = "OBSERVED_AT"
	TimestampRolePublishedAt   AuthoritativeTimestampRole = "PUBLISHED_AT"
	TimestampRoleCollectedAt   AuthoritativeTimestampRole = "COLLECTED_AT"
	TimestampRoleOccurredAt    AuthoritativeTimestampRole = "OCCURRED_AT"
	TimestampRoleEffectiveAt   AuthoritativeTimestampRole = "EFFECTIVE_AT"
	TimestampRoleEffectiveFrom AuthoritativeTimestampRole = "EFFECTIVE_FROM"
	TimestampRoleDatasetAsOf   AuthoritativeTimestampRole = "DATASET_AS_OF"
)

type MissingTimestampBehavior string

const (
	MissingTimestampFail    MissingTimestampBehavior = "FAIL"
	MissingTimestampUnknown MissingTimestampBehavior = "UNKNOWN"
)

type FallbackMode string

const (
	FallbackProhibited   FallbackMode = "PROHIBITED"
	FallbackFreshOnly    FallbackMode = "FRESH_ONLY"
	FallbackFreshOrStale FallbackMode = "FRESH_OR_STALE"
)

// LastKnownGoodPolicy makes fallback an independently identified policy. A
// positive MaximumAge is mandatory whenever fallback is allowed.
type LastKnownGoodPolicy struct {
	ContractVersion canonical.ContractVersion   `json:"contract_version"`
	Identity        canonical.ComponentIdentity `json:"identity"`
	Mode            FallbackMode                `json:"mode"`
	MaximumAge      time.Duration               `json:"maximum_age"`
}

// FreshnessPolicy is provider-neutral. Capability, canonical target, and use
// class are stable Jax semantic keys; provider-specific names and conditions
// do not belong in the deterministic evaluator.
type FreshnessPolicy struct {
	ContractVersion   canonical.ContractVersion   `json:"contract_version"`
	Identity          canonical.ComponentIdentity `json:"identity"`
	CapabilityID      CapabilityID                `json:"capability_id"`
	Target            canonical.ContractSchemaRef `json:"target"`
	UseClass          DataUseClass                `json:"use_class"`
	ValidityMode      FreshnessValidityMode       `json:"validity_mode"`
	TimestampRole     AuthoritativeTimestampRole  `json:"timestamp_role"`
	FreshFor          time.Duration               `json:"fresh_for"`
	ExpireAfter       time.Duration               `json:"expire_after"`
	AllowedFutureSkew time.Duration               `json:"allowed_future_skew"`
	MissingTimestamp  MissingTimestampBehavior    `json:"missing_timestamp"`
	LastKnownGood     LastKnownGoodPolicy         `json:"last_known_good"`
}

// TemporalQualityState is the state of one exact canonical record at one
// evaluation time. It is not provider health and does not replace canonical
// validation quality.
type TemporalQualityState string

const (
	TemporalFresh         TemporalQualityState = "FRESH"
	TemporalStale         TemporalQualityState = "STALE"
	TemporalExpired       TemporalQualityState = "EXPIRED"
	TemporalUnknown       TemporalQualityState = "UNKNOWN"
	TemporalNotApplicable TemporalQualityState = "NOT_APPLICABLE"
)

type TemporalRecordState string

const (
	TemporalRecordActive      TemporalRecordState = "ACTIVE"
	TemporalRecordSuperseded  TemporalRecordState = "SUPERSEDED"
	TemporalRecordRetracted   TemporalRecordState = "RETRACTED"
	TemporalRecordDisputed    TemporalRecordState = "DISPUTED"
	TemporalRecordInvalidated TemporalRecordState = "INVALIDATED"
)

// TemporalRecordLifecycle is supplied immutable status evidence. ChangedAt is
// required for every non-active state so historical replay can determine
// whether the status applied at its evaluation time.
type TemporalRecordLifecycle struct {
	State     TemporalRecordState `json:"state"`
	ChangedAt *time.Time          `json:"changed_at,omitempty"`
}

type FreshnessEvaluationContext string

const (
	FreshnessContextCurrentState     FreshnessEvaluationContext = "CURRENT_STATE"
	FreshnessContextHistoricalReplay FreshnessEvaluationContext = "HISTORICAL_REPLAY"
)

// FreshnessKey prevents last-known-good selection across different semantic
// subjects or values. Qualifier is a stable semantic value key (for an
// Observation it must equal metric), never a provider symbol or row key.
type FreshnessKey struct {
	CapabilityID CapabilityID                `json:"capability_id"`
	Target       canonical.ContractSchemaRef `json:"target"`
	Subject      canonical.ContractRef       `json:"subject"`
	Qualifier    string                      `json:"qualifier"`
}

// TemporalRecord binds the accepted WP-02.03 normalization envelope to its
// semantic key and immutable lifecycle status. PriorFreshEvaluation is needed
// only when a stale record is offered as last-known-good.
type TemporalRecord struct {
	Normalized           NormalizationResult
	Key                  FreshnessKey
	Lifecycle            TemporalRecordLifecycle
	PriorFreshEvaluation *FreshnessEvaluation
}

type FreshnessReasonCode string

const (
	FreshnessReasonWithinHorizon    FreshnessReasonCode = "within_freshness_horizon"
	FreshnessReasonAtOrBeyondTTL    FreshnessReasonCode = "at_or_beyond_freshness_ttl"
	FreshnessReasonAtOrBeyondExpiry FreshnessReasonCode = "at_or_beyond_expiry_horizon"
	FreshnessReasonMissingTimestamp FreshnessReasonCode = "authoritative_timestamp_missing"
	FreshnessReasonUntilSuperseded  FreshnessReasonCode = "valid_until_superseded"
	FreshnessReasonNotApplicable    FreshnessReasonCode = "freshness_not_applicable"
)

// FreshnessEvaluation records the complete deterministic decision for one
// canonical record. Age remains negative inside an allowed skew window; it is
// never silently clamped to zero.
type FreshnessEvaluation struct {
	ContractVersion        canonical.ContractVersion      `json:"contract_version"`
	Record                 canonical.ImmutableContractRef `json:"record"`
	RawPayloadID           RawPayloadID                   `json:"raw_payload_id"`
	Normalizer             canonical.ComponentIdentity    `json:"normalizer"`
	Policy                 canonical.ComponentIdentity    `json:"policy"`
	CapabilityID           CapabilityID                   `json:"capability_id"`
	UseClass               DataUseClass                   `json:"use_class"`
	Key                    FreshnessKey                   `json:"key"`
	EvaluationTime         time.Time                      `json:"evaluation_time"`
	EvaluationContext      FreshnessEvaluationContext     `json:"evaluation_context"`
	TimestampRole          AuthoritativeTimestampRole     `json:"timestamp_role"`
	AuthoritativeTimestamp *time.Time                     `json:"authoritative_timestamp,omitempty"`
	Age                    *time.Duration                 `json:"age,omitempty"`
	WithinFutureSkew       bool                           `json:"within_future_skew"`
	State                  TemporalQualityState           `json:"state"`
	CanonicalQuality       NormalizationQuality           `json:"canonical_quality"`
	Lifecycle              TemporalRecordLifecycle        `json:"lifecycle"`
	ReasonCode             FreshnessReasonCode            `json:"reason_code"`
}

type FallbackStatus string

const (
	FallbackNotUsed FallbackStatus = "NOT_USED"
	FallbackUsed    FallbackStatus = "USED"
)

type LKGQualification string

const (
	LKGQualificationNotApplicable  LKGQualification = "NOT_APPLICABLE"
	LKGQualificationCurrentlyFresh LKGQualification = "CURRENTLY_FRESH"
	LKGQualificationPriorFresh     LKGQualification = "PRIOR_FRESH_EVALUATION"
)

type FreshnessResolutionReason string

const (
	ResolutionCurrentAccepted FreshnessResolutionReason = "current_record_temporally_accepted"
	ResolutionCurrentMissing  FreshnessResolutionReason = "current_record_missing_lkg_selected"
	ResolutionCurrentInvalid  FreshnessResolutionReason = "current_record_invalid_lkg_selected"
	ResolutionCurrentStale    FreshnessResolutionReason = "current_record_stale_lkg_selected"
	ResolutionCurrentExpired  FreshnessResolutionReason = "current_record_expired_lkg_selected"
	ResolutionCurrentUnknown  FreshnessResolutionReason = "current_record_unknown_lkg_selected"
)

// FreshnessResolution preserves all three dimensions: normalization quality,
// the selected record's original temporal state, and whether fallback was
// used. Selecting stale data never upgrades its state to FRESH.
type FreshnessResolution struct {
	ContractVersion      canonical.ContractVersion       `json:"contract_version"`
	Policy               canonical.ComponentIdentity     `json:"policy"`
	FallbackPolicy       canonical.ComponentIdentity     `json:"fallback_policy"`
	EvaluationTime       time.Time                       `json:"evaluation_time"`
	Key                  FreshnessKey                    `json:"key"`
	CurrentRecord        *canonical.ImmutableContractRef `json:"current_record,omitempty"`
	Selected             FreshnessEvaluation             `json:"selected"`
	FallbackStatus       FallbackStatus                  `json:"fallback_status"`
	FallbackAge          *time.Duration                  `json:"fallback_age,omitempty"`
	Qualification        LKGQualification                `json:"qualification"`
	PriorFreshEvaluation *FreshnessEvaluation            `json:"prior_fresh_evaluation,omitempty"`
	ReasonCode           FreshnessResolutionReason       `json:"reason_code"`
}

type FreshnessEvaluationRequest struct {
	Policy         canonical.ComponentIdentity
	UseClass       DataUseClass
	EvaluationTime time.Time
	Context        FreshnessEvaluationContext
	Record         TemporalRecord
}

type FreshnessResolutionRequest struct {
	Policy                 canonical.ComponentIdentity
	ExpectedFallbackPolicy canonical.ComponentIdentity
	UseClass               DataUseClass
	EvaluationTime         time.Time
	Context                FreshnessEvaluationContext
	Key                    FreshnessKey
	Current                *TemporalRecord
	Historical             []TemporalRecord
}

type FreshnessErrorCode string

const (
	FreshnessErrorUnsupportedPolicyVersion        FreshnessErrorCode = "unsupported_policy_version"
	FreshnessErrorInvalidPolicy                   FreshnessErrorCode = "invalid_freshness_policy"
	FreshnessErrorInvalidTTL                      FreshnessErrorCode = "invalid_ttl"
	FreshnessErrorInvalidEvaluationTime           FreshnessErrorCode = "invalid_evaluation_time"
	FreshnessErrorMissingAuthoritativeTimestamp   FreshnessErrorCode = "missing_authoritative_timestamp"
	FreshnessErrorAmbiguousAuthoritativeTimestamp FreshnessErrorCode = "ambiguous_authoritative_timestamp"
	FreshnessErrorFutureTimestampBeyondSkew       FreshnessErrorCode = "future_timestamp_beyond_allowed_skew"
	FreshnessErrorRecordNotYetAvailable           FreshnessErrorCode = "record_not_yet_available"
	FreshnessErrorAgeOutOfRange                   FreshnessErrorCode = "age_out_of_range"
	FreshnessErrorSemanticKeyMismatch             FreshnessErrorCode = "semantic_key_mismatch"
	FreshnessErrorInvalidCanonicalInput           FreshnessErrorCode = "invalid_canonical_input"
	FreshnessErrorInvalidProvenanceInput          FreshnessErrorCode = "invalid_provenance_input"
	FreshnessErrorNoAcceptableLKG                 FreshnessErrorCode = "no_acceptable_lkg_candidate"
	FreshnessErrorAmbiguousLKG                    FreshnessErrorCode = "ambiguous_lkg_selection"
	FreshnessErrorFallbackAgeExceeded             FreshnessErrorCode = "candidate_exceeds_fallback_age"
	FreshnessErrorFallbackProhibited              FreshnessErrorCode = "fallback_prohibited"
	FreshnessErrorRecordSuperseded                FreshnessErrorCode = "record_superseded"
	FreshnessErrorRecordRetracted                 FreshnessErrorCode = "record_retracted"
	FreshnessErrorRecordDisputed                  FreshnessErrorCode = "record_disputed"
	FreshnessErrorRecordInvalidated               FreshnessErrorCode = "record_invalidated"
	FreshnessErrorPolicyCapabilityMismatch        FreshnessErrorCode = "policy_capability_mismatch"
	FreshnessErrorPolicyVersionMismatch           FreshnessErrorCode = "policy_version_mismatch"
	FreshnessErrorDuplicatePolicy                 FreshnessErrorCode = "duplicate_freshness_policy"
	FreshnessErrorAmbiguousPolicy                 FreshnessErrorCode = "ambiguous_freshness_policy"
	FreshnessErrorUnknownPolicy                   FreshnessErrorCode = "unknown_freshness_policy"
	FreshnessErrorPriorAcceptanceMismatch         FreshnessErrorCode = "prior_fresh_evaluation_mismatch"
)

// FreshnessError is safe for operator-visible handling and contains no raw
// payload bytes, credentials, provider response, or network error text.
type FreshnessError struct {
	Code         FreshnessErrorCode
	PolicyID     string
	CapabilityID CapabilityID
	RecordID     string
	Detail       string
	Cause        error
}

func (err *FreshnessError) Error() string {
	return fmt.Sprintf("provider freshness %s: policy=%q capability=%q record=%q %s", err.Code, err.PolicyID, err.CapabilityID, err.RecordID, err.Detail)
}

func (err *FreshnessError) Unwrap() error { return err.Cause }

func freshnessError(code FreshnessErrorCode, policy canonical.ComponentIdentity, capability CapabilityID, record canonical.ImmutableContractRef, detail string, cause error) error {
	return &FreshnessError{Code: code, PolicyID: policy.ID, CapabilityID: capability, RecordID: record.Contract.ID, Detail: detail, Cause: cause}
}

func asFreshnessError(err error) (*FreshnessError, bool) {
	var target *FreshnessError
	ok := errors.As(err, &target)
	return target, ok
}

func samePolicyIdentity(left, right canonical.ComponentIdentity) bool {
	return reflect.DeepEqual(left, right)
}
