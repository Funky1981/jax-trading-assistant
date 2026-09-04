// Package evidencequality provides deterministic, provider-neutral checks for
// semantic compatibility and consistency between accepted evidence inputs.
// It does not rank sources, infer meaning, or produce trading conclusions.
package evidencequality

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

const (
	MappingContractV1          canonical.ContractVersion = "jax.evidencequality/mapping/v1"
	ComparisonPolicyContractV1 canonical.ContractVersion = "jax.evidencequality/comparison_policy/v1"
	CrossSourceCheckContractV1 canonical.ContractVersion = "jax.evidencequality/cross_source_check/v1"
	RangeCheckContractV1       canonical.ContractVersion = "jax.evidencequality/range_check/v1"

	MappingIDPrefix = "eqm_"
	PolicyIDPrefix  = "eqp_"
	CheckIDPrefix   = "eqc_"
	RangeIDPrefix   = "eqr_"
)

type LineageRelationship string

const (
	LineageIndependent             LineageRelationship = "INDEPENDENT"
	LineageSameOriginRedistributed LineageRelationship = "SAME_ORIGIN_REDISTRIBUTED"
	LineageDerivedFrom             LineageRelationship = "DERIVED_FROM"
	LineageUnknown                 LineageRelationship = "UNKNOWN"
)

func (relationship LineageRelationship) Validate() error {
	switch relationship {
	case LineageIndependent, LineageSameOriginRedistributed, LineageDerivedFrom, LineageUnknown:
		return nil
	default:
		return fmt.Errorf("unsupported lineage relationship %q", relationship)
	}
}

type ComparisonOutcome string

const (
	OutcomeAgree         ComparisonOutcome = "AGREE"
	OutcomeDiverge       ComparisonOutcome = "DIVERGE"
	OutcomeLeftMissing   ComparisonOutcome = "LEFT_MISSING"
	OutcomeRightMissing  ComparisonOutcome = "RIGHT_MISSING"
	OutcomeBothMissing   ComparisonOutcome = "BOTH_MISSING"
	OutcomeNotComparable ComparisonOutcome = "NOT_COMPARABLE"
	OutcomeIndeterminate ComparisonOutcome = "INDETERMINATE"
)

func (outcome ComparisonOutcome) Validate() error {
	switch outcome {
	case OutcomeAgree, OutcomeDiverge, OutcomeLeftMissing, OutcomeRightMissing,
		OutcomeBothMissing, OutcomeNotComparable, OutcomeIndeterminate:
		return nil
	default:
		return fmt.Errorf("unsupported comparison outcome %q", outcome)
	}
}

type ReasonCode string

const (
	ReasonValuesEqual              ReasonCode = "VALUES_EQUAL"
	ReasonValuesDiffer             ReasonCode = "VALUES_DIFFER"
	ReasonLeftMissing              ReasonCode = "LEFT_VALUE_MISSING"
	ReasonRightMissing             ReasonCode = "RIGHT_VALUE_MISSING"
	ReasonBothMissing              ReasonCode = "BOTH_VALUES_MISSING"
	ReasonMappingNotApproved       ReasonCode = "MAPPING_NOT_APPROVED"
	ReasonSeriesIdentityMismatch   ReasonCode = "SERIES_IDENTITY_MISMATCH"
	ReasonMeasurementMismatch      ReasonCode = "MEASUREMENT_TYPE_MISMATCH"
	ReasonUnitMismatch             ReasonCode = "UNIT_MISMATCH"
	ReasonFrequencyMismatch        ReasonCode = "FREQUENCY_MISMATCH"
	ReasonObservationDateMismatch  ReasonCode = "OBSERVATION_DATE_MISMATCH"
	ReasonInformationStateMismatch ReasonCode = "INFORMATION_STATE_MISMATCH"
	ReasonRealtimeStateMismatch    ReasonCode = "REALTIME_STATE_MISMATCH"
	ReasonAdjustmentMismatch       ReasonCode = "ADJUSTMENT_MISMATCH"
	ReasonSemanticsMismatch        ReasonCode = "OBSERVATION_SEMANTICS_MISMATCH"
	ReasonReleaseSemanticsMismatch ReasonCode = "RELEASE_SEMANTICS_MISMATCH"
)

type SourceSeriesIdentity struct {
	Provider         canonical.ProviderIdentity `json:"provider"`
	Source           canonical.SourceIdentity   `json:"source"`
	ProviderSeriesID string                     `json:"provider_series_id"`
}

func (identity SourceSeriesIdentity) Validate() error {
	if err := identity.Provider.Validate(); err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	if err := identity.Source.Validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if strings.TrimSpace(identity.ProviderSeriesID) == "" || len(identity.ProviderSeriesID) > 128 {
		return fmt.Errorf("provider series identity is required")
	}
	return nil
}

// SeriesSemantics are explicit fields needed to establish a comparison. They
// are never inferred from a title or a provider-series name.
type SeriesSemantics struct {
	MeasurementType      string `json:"measurement_type"`
	Units                string `json:"units"`
	FrequencyCode        string `json:"frequency_code"`
	ObservationSemantics string `json:"observation_semantics"`
	AdjustmentMethod     string `json:"adjustment_method"`
	ReleaseSemantics     string `json:"release_semantics"`
}

func (semantics SeriesSemantics) Validate() error {
	for field, value := range map[string]string{
		"measurement_type":      semantics.MeasurementType,
		"units":                 semantics.Units,
		"frequency_code":        semantics.FrequencyCode,
		"observation_semantics": semantics.ObservationSemantics,
		"release_semantics":     semantics.ReleaseSemantics,
	} {
		if strings.TrimSpace(value) == "" || len(value) > 128 {
			return fmt.Errorf("%s is required", field)
		}
	}
	if len(semantics.AdjustmentMethod) > 128 {
		return fmt.Errorf("adjustment_method is too long")
	}
	return nil
}

type InformationState struct {
	Mode string `json:"mode"`
	Date string `json:"date,omitempty"`
}

func InformationStateFromMacro(state macroevidence.InformationState) InformationState {
	result := InformationState{Mode: string(state.Mode)}
	if state.Date != nil {
		result.Date = string(*state.Date)
	}
	return result
}

func (state InformationState) Validate() error {
	switch state.Mode {
	case string(macroevidence.InformationStateCurrent), string(macroevidence.InformationStateInitialRelease):
		if state.Date != "" {
			return fmt.Errorf("%s information state must not carry a date", state.Mode)
		}
	case string(macroevidence.InformationStateAsOf), string(macroevidence.InformationStateVintage):
		if state.Date == "" {
			return fmt.Errorf("%s information state requires a date", state.Mode)
		}
		if err := validateDate(state.Date); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported information state mode %q", state.Mode)
	}
	return nil
}

type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func (period DateRange) Validate() error {
	if err := validateDate(period.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := validateDate(period.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if period.End < period.Start {
		return fmt.Errorf("end precedes start")
	}
	return nil
}

type ComparisonPolicy struct {
	ContractVersion  canonical.ContractVersion `json:"contract_version"`
	ID               string                    `json:"id"`
	Version          canonical.VersionIdentity `json:"version"`
	NumericMode      string                    `json:"numeric_mode"`
	Tolerance        string                    `json:"tolerance"`
	RequireDate      bool                      `json:"require_date"`
	RequireInfoState bool                      `json:"require_information_state"`
	RequireRealtime  bool                      `json:"require_realtime_alignment"`
}

const NumericExactDecimal = "EXACT_DECIMAL"

func (policy ComparisonPolicy) Validate() error {
	if policy.ContractVersion != ComparisonPolicyContractV1 {
		return fmt.Errorf("comparison policy contract version must be %q", ComparisonPolicyContractV1)
	}
	if !strings.HasPrefix(policy.ID, PolicyIDPrefix) || len(policy.ID) <= len(PolicyIDPrefix) {
		return fmt.Errorf("comparison policy ID must use %s prefix", PolicyIDPrefix)
	}
	if err := policy.Version.Validate(); err != nil {
		return err
	}
	if policy.NumericMode != NumericExactDecimal {
		return fmt.Errorf("unsupported numeric comparison mode")
	}
	if strings.TrimSpace(policy.Tolerance) == "" {
		return fmt.Errorf("comparison tolerance is required")
	}
	if _, err := parseDecimal(policy.Tolerance); err != nil {
		return fmt.Errorf("comparison tolerance: %w", err)
	}
	if policy.Tolerance != "0" {
		return fmt.Errorf("exact decimal policy requires zero tolerance")
	}
	return nil
}

type Mapping struct {
	ContractVersion     canonical.ContractVersion `json:"contract_version"`
	ID                  string                    `json:"id"`
	Version             canonical.VersionIdentity `json:"version"`
	LeftSeries          SourceSeriesIdentity      `json:"left_series"`
	RightSeries         SourceSeriesIdentity      `json:"right_series"`
	LeftSemantics       SeriesSemantics           `json:"left_semantics"`
	RightSemantics      SeriesSemantics           `json:"right_semantics"`
	MeasurementIdentity string                    `json:"measurement_identity"`
	CanonicalUnits      string                    `json:"canonical_units"`
	Lineage             LineageRelationship       `json:"lineage"`
	Policy              ComparisonPolicy          `json:"policy"`
	AuthorityNotes      string                    `json:"authority_notes"`
}

func (mapping Mapping) Validate() error {
	if mapping.ContractVersion != MappingContractV1 {
		return fmt.Errorf("mapping contract version must be %q", MappingContractV1)
	}
	if !strings.HasPrefix(mapping.ID, MappingIDPrefix) || len(mapping.ID) <= len(MappingIDPrefix) {
		return fmt.Errorf("mapping ID must use %s prefix", MappingIDPrefix)
	}
	if err := mapping.Version.Validate(); err != nil {
		return err
	}
	if err := mapping.LeftSeries.Validate(); err != nil {
		return fmt.Errorf("left series: %w", err)
	}
	if err := mapping.RightSeries.Validate(); err != nil {
		return fmt.Errorf("right series: %w", err)
	}
	if err := mapping.LeftSemantics.Validate(); err != nil {
		return fmt.Errorf("left semantics: %w", err)
	}
	if err := mapping.RightSemantics.Validate(); err != nil {
		return fmt.Errorf("right semantics: %w", err)
	}
	if strings.TrimSpace(mapping.MeasurementIdentity) == "" || strings.TrimSpace(mapping.CanonicalUnits) == "" || strings.TrimSpace(mapping.AuthorityNotes) == "" {
		return fmt.Errorf("measurement identity, canonical units, and authority notes are required")
	}
	if err := mapping.Lineage.Validate(); err != nil {
		return err
	}
	if err := mapping.Policy.Validate(); err != nil {
		return err
	}
	return nil
}

type Registry struct {
	mappings map[string]Mapping
	pairs    map[string]string
}

func NewRegistry(mappings []Mapping) (*Registry, error) {
	registry := &Registry{mappings: make(map[string]Mapping, len(mappings)), pairs: make(map[string]string, len(mappings))}
	for _, mapping := range mappings {
		if err := mapping.Validate(); err != nil {
			return nil, fmt.Errorf("mapping %s: %w", mapping.ID, err)
		}
		if _, exists := registry.mappings[mapping.ID]; exists {
			return nil, fmt.Errorf("duplicate mapping ID %q", mapping.ID)
		}
		pair := seriesPairKey(mapping.LeftSeries, mapping.RightSeries)
		if existing, exists := registry.pairs[pair]; exists {
			return nil, fmt.Errorf("ambiguous mapping pair: %q and %q", existing, mapping.ID)
		}
		registry.mappings[mapping.ID] = mapping
		registry.pairs[pair] = mapping.ID
	}
	return registry, nil
}

func (registry *Registry) Get(id string) (Mapping, bool) {
	if registry == nil {
		return Mapping{}, false
	}
	mapping, ok := registry.mappings[id]
	return mapping, ok
}

func (registry *Registry) Lookup(left, right SourceSeriesIdentity) (Mapping, bool) {
	if registry == nil {
		return Mapping{}, false
	}
	for _, mapping := range registry.mappings {
		if sameSeries(left, mapping.LeftSeries) && sameSeries(right, mapping.RightSeries) {
			return mapping, true
		}
	}
	return Mapping{}, false
}

type SourceValue struct {
	Present     bool   `json:"present"`
	SourceValue string `json:"source_value"`
}

func (value SourceValue) Validate() error {
	if !value.Present {
		if value.SourceValue == "" {
			return nil
		}
		switch strings.ToUpper(strings.TrimSpace(value.SourceValue)) {
		case ".", "N/A", "NA", "NULL":
			return nil
		default:
			return fmt.Errorf("missing value must preserve a recognized source marker")
		}
	}
	if strings.TrimSpace(value.SourceValue) == "" {
		return fmt.Errorf("present value requires source_value")
	}
	if _, err := parseDecimal(value.SourceValue); err != nil {
		return err
	}
	return nil
}

// ObservationInput is the small provider-neutral projection consumed by the
// checker. Adapters retain their own DTOs and use this only after validation.
type ObservationInput struct {
	ID                     string                         `json:"id"`
	Series                 SourceSeriesIdentity           `json:"series"`
	Semantics              SeriesSemantics                `json:"semantics"`
	ObservationDate        string                         `json:"observation_date"`
	Value                  SourceValue                    `json:"value"`
	InformationState       InformationState               `json:"information_state"`
	RealtimePeriod         *DateRange                     `json:"realtime_period,omitempty"`
	AcquiredAt             time.Time                      `json:"acquired_at"`
	PublicationTime        *time.Time                     `json:"publication_time,omitempty"`
	PublicAvailabilityTime *time.Time                     `json:"public_availability_time,omitempty"`
	SourcePayload          providercontract.RawPayloadRef `json:"source_payload"`
	Provenance             canonical.Provenance           `json:"provenance"`
}

func FromMacroObservation(series macroevidence.MacroSeries, observation macroevidence.MacroObservation, measurementType, observationSemantics string) (ObservationInput, error) {
	if err := series.Validate(); err != nil {
		return ObservationInput{}, fmt.Errorf("series: %w", err)
	}
	if err := observation.Validate(); err != nil {
		return ObservationInput{}, fmt.Errorf("observation: %w", err)
	}
	if series.ID != observation.Series || series.ProviderSeriesID != observation.ProviderSeriesID {
		return ObservationInput{}, fmt.Errorf("observation does not belong to the supplied series")
	}
	if observation.SourcePayload.Source == nil {
		return ObservationInput{}, fmt.Errorf("source identity is required")
	}
	input := ObservationInput{
		ID:               observation.ID,
		Series:           SourceSeriesIdentity{Provider: observation.SourcePayload.Provider, Source: *observation.SourcePayload.Source, ProviderSeriesID: observation.ProviderSeriesID},
		Semantics:        SeriesSemantics{MeasurementType: measurementType, Units: series.Units, FrequencyCode: series.FrequencyCode, ObservationSemantics: observationSemantics, ReleaseSemantics: VIXReleaseSemantics},
		ObservationDate:  string(observation.ObservationDate),
		Value:            SourceValue{Present: observation.Value.Present, SourceValue: observation.Value.SourceValue},
		InformationState: InformationStateFromMacro(observation.RequestedInformation),
		AcquiredAt:       observation.AcquiredAt,
		SourcePayload:    observation.SourcePayload,
		Provenance:       observation.Provenance,
	}
	if observation.RealtimePeriod != nil {
		input.RealtimePeriod = &DateRange{Start: string(observation.RealtimePeriod.Start), End: string(observation.RealtimePeriod.End)}
	}
	if err := input.Validate(); err != nil {
		return ObservationInput{}, err
	}
	return input, nil
}

func (input ObservationInput) Validate() error {
	if strings.TrimSpace(input.ID) == "" {
		return fmt.Errorf("observation ID is required")
	}
	if err := input.Series.Validate(); err != nil {
		return err
	}
	if err := input.Semantics.Validate(); err != nil {
		return err
	}
	if err := validateDate(input.ObservationDate); err != nil {
		return fmt.Errorf("observation date: %w", err)
	}
	if err := input.Value.Validate(); err != nil {
		return err
	}
	if err := input.InformationState.Validate(); err != nil {
		return err
	}
	if input.RealtimePeriod != nil {
		if err := input.RealtimePeriod.Validate(); err != nil {
			return fmt.Errorf("realtime period: %w", err)
		}
	}
	if input.AcquiredAt.IsZero() || input.AcquiredAt.Location() != time.UTC {
		return fmt.Errorf("acquired_at must be a non-zero UTC timestamp")
	}
	for field, value := range map[string]*time.Time{"publication_time": input.PublicationTime, "public_availability_time": input.PublicAvailabilityTime} {
		if value != nil && (value.IsZero() || value.Location() != time.UTC || value.After(input.AcquiredAt)) {
			return fmt.Errorf("%s must be a non-zero UTC timestamp no later than acquired_at", field)
		}
	}
	if err := input.SourcePayload.Validate(); err != nil {
		return fmt.Errorf("source payload: %w", err)
	}
	if input.SourcePayload.Source == nil || *input.SourcePayload.Source != input.Series.Source || !sameProvider(input.SourcePayload.Provider, input.Series.Provider) {
		return fmt.Errorf("source payload identity does not match observation series")
	}
	if err := input.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	return nil
}

type EvidenceObservationReference struct {
	ID              string                         `json:"id"`
	Series          SourceSeriesIdentity           `json:"series"`
	ObservationDate string                         `json:"observation_date"`
	SourcePayload   providercontract.RawPayloadRef `json:"source_payload"`
}

type PolicyReference struct {
	ID        string                    `json:"id"`
	Version   canonical.VersionIdentity `json:"version"`
	Tolerance string                    `json:"tolerance"`
}

type CrossSourceCheck struct {
	ContractVersion        canonical.ContractVersion    `json:"contract_version"`
	ID                     string                       `json:"id"`
	MappingID              string                       `json:"mapping_id,omitempty"`
	MappingVersion         *canonical.VersionIdentity   `json:"mapping_version,omitempty"`
	Left                   EvidenceObservationReference `json:"left"`
	Right                  EvidenceObservationReference `json:"right"`
	MeasurementIdentity    string                       `json:"measurement_identity"`
	ObservationDate        string                       `json:"observation_date"`
	Outcome                ComparisonOutcome            `json:"outcome"`
	Lineage                LineageRelationship          `json:"lineage"`
	IndependentSourceCount int                          `json:"independent_source_count"`
	LeftValue              string                       `json:"left_value,omitempty"`
	RightValue             string                       `json:"right_value,omitempty"`
	AbsoluteDifference     string                       `json:"absolute_difference,omitempty"`
	TolerancePolicy        PolicyReference              `json:"tolerance_policy"`
	ReasonCode             ReasonCode                   `json:"reason_code"`
	EvaluatedAt            time.Time                    `json:"evaluated_at"`
	LeftEvidence           []canonical.EvidenceRef      `json:"left_evidence"`
	RightEvidence          []canonical.EvidenceRef      `json:"right_evidence"`
	Provenance             canonical.Provenance         `json:"provenance"`
}

func (check CrossSourceCheck) Validate() error {
	if check.ContractVersion != CrossSourceCheckContractV1 {
		return fmt.Errorf("cross-source check contract version must be %q", CrossSourceCheckContractV1)
	}
	if !strings.HasPrefix(check.ID, CheckIDPrefix) || len(check.ID) <= len(CheckIDPrefix) {
		return fmt.Errorf("check ID must use %s prefix", CheckIDPrefix)
	}
	if err := check.Left.validate(); err != nil {
		return fmt.Errorf("left: %w", err)
	}
	if err := check.Right.validate(); err != nil {
		return fmt.Errorf("right: %w", err)
	}
	if err := validateDate(check.ObservationDate); err != nil {
		return err
	}
	if err := check.Outcome.Validate(); err != nil {
		return err
	}
	if err := check.Lineage.Validate(); err != nil {
		return err
	}
	if check.IndependentSourceCount < 0 || check.IndependentSourceCount > 2 {
		return fmt.Errorf("independent source count is invalid")
	}
	if err := check.TolerancePolicy.Version.Validate(); err != nil {
		return err
	}
	if check.TolerancePolicy.ID == "" || check.TolerancePolicy.Tolerance == "" {
		return fmt.Errorf("tolerance policy reference is required")
	}
	if _, err := parseDecimal(check.TolerancePolicy.Tolerance); err != nil {
		return err
	}
	switch check.ReasonCode {
	case ReasonValuesEqual, ReasonValuesDiffer, ReasonLeftMissing, ReasonRightMissing, ReasonBothMissing,
		ReasonMappingNotApproved, ReasonSeriesIdentityMismatch, ReasonMeasurementMismatch, ReasonUnitMismatch,
		ReasonFrequencyMismatch, ReasonObservationDateMismatch, ReasonInformationStateMismatch,
		ReasonRealtimeStateMismatch, ReasonAdjustmentMismatch, ReasonSemanticsMismatch, ReasonReleaseSemanticsMismatch:
	default:
		return fmt.Errorf("unsupported reason code %q", check.ReasonCode)
	}
	if check.EvaluatedAt.IsZero() || check.EvaluatedAt.Location() != time.UTC {
		return fmt.Errorf("evaluated_at must be a non-zero UTC timestamp")
	}
	if len(check.LeftEvidence) == 0 || len(check.RightEvidence) == 0 {
		return fmt.Errorf("both evidence sides are required")
	}
	for _, ref := range append(append([]canonical.EvidenceRef{}, check.LeftEvidence...), check.RightEvidence...) {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("evidence reference: %w", err)
		}
	}
	if err := check.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if !provenanceIncludes(check.Provenance, append(append([]canonical.EvidenceRef{}, check.LeftEvidence...), check.RightEvidence...)) {
		return fmt.Errorf("provenance must cover both source evidence sides")
	}
	return nil
}

func (reference EvidenceObservationReference) validate() error {
	if strings.TrimSpace(reference.ID) == "" {
		return fmt.Errorf("observation ID is required")
	}
	if err := reference.Series.Validate(); err != nil {
		return err
	}
	if err := validateDate(reference.ObservationDate); err != nil {
		return err
	}
	if err := reference.SourcePayload.Validate(); err != nil {
		return err
	}
	return nil
}

func independentSourceCount(lineage LineageRelationship) int {
	if lineage == LineageIndependent {
		return 2
	}
	if lineage == LineageUnknown {
		return 0
	}
	return 1
}

func sameSeries(left, right SourceSeriesIdentity) bool {
	return sameProvider(left.Provider, right.Provider) && left.Source == right.Source && left.ProviderSeriesID == right.ProviderSeriesID
}

func seriesPairKey(left, right SourceSeriesIdentity) string {
	return left.Provider.ID + "\x00" + left.Provider.Namespace + "\x00" + sourceKey(left.Source) + "\x00" + left.ProviderSeriesID + "\x00" + right.Provider.ID + "\x00" + right.Provider.Namespace + "\x00" + sourceKey(right.Source) + "\x00" + right.ProviderSeriesID
}

func sourceKey(source canonical.SourceIdentity) string {
	return source.ID + "\x00" + string(source.Kind)
}

func sameProvider(left, right canonical.ProviderIdentity) bool {
	if left.ID != right.ID || left.Namespace != right.Namespace {
		return false
	}
	if left.ExternalID == nil || right.ExternalID == nil {
		return left.ExternalID == nil && right.ExternalID == nil
	}
	return *left.ExternalID == *right.ExternalID
}

func effectiveLineage(mapping Mapping, left, right ObservationInput) LineageRelationship {
	if left.Series.Source == right.Series.Source || sameProvider(left.Series.Provider, right.Series.Provider) && left.Series.ProviderSeriesID == right.Series.ProviderSeriesID {
		return LineageSameOriginRedistributed
	}
	if left.SourcePayload.Content.Digest == right.SourcePayload.Content.Digest {
		return LineageUnknown
	}
	return mapping.Lineage
}

var decimalPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$`)

func parseDecimal(raw string) (*big.Rat, error) {
	value := strings.TrimSpace(raw)
	if !decimalPattern.MatchString(value) {
		return nil, fmt.Errorf("value %q is not a finite decimal; NaN and Inf are not accepted", raw)
	}
	result, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("value %q is not a valid decimal", raw)
	}
	return result, nil
}

func decimalDifference(left, right string) (string, error) {
	a, err := parseDecimal(left)
	if err != nil {
		return "", err
	}
	b, err := parseDecimal(right)
	if err != nil {
		return "", err
	}
	difference := new(big.Rat).Sub(a, b)
	if difference.Sign() < 0 {
		difference.Neg(difference)
	}
	return finiteDecimal(difference), nil
}

func finiteDecimal(value *big.Rat) string {
	// Decimal source inputs produce finite decimals. Increase precision until
	// the exact decimal is recovered, without using float arithmetic.
	for scale := 0; ; scale++ {
		candidate := value.FloatString(scale)
		parsed, ok := new(big.Rat).SetString(candidate)
		if ok && parsed.Cmp(value) == 0 {
			trimmed := strings.TrimRight(strings.TrimRight(candidate, "0"), ".")
			if trimmed == "" || trimmed == "-" {
				return "0"
			}
			return trimmed
		}
		if scale > 10000 {
			return value.RatString()
		}
	}
}

func validateDate(value string) error {
	if len(value) != len("2006-01-02") {
		return fmt.Errorf("date %q must use YYYY-MM-DD", value)
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return fmt.Errorf("date %q is invalid", value)
	}
	return nil
}

func evidenceReferences(provenance canonical.Provenance) ([]canonical.EvidenceRef, error) {
	refs := make([]canonical.EvidenceRef, 0)
	for _, input := range provenance.Inputs {
		if input.Kind == canonical.LineageInputKindEvidence && input.Evidence != nil {
			refs = append(refs, *input.Evidence)
		}
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("provenance contains no evidence reference")
	}
	return refs, nil
}

func provenanceIncludes(provenance canonical.Provenance, refs []canonical.EvidenceRef) bool {
	for _, want := range refs {
		found := false
		for _, input := range provenance.Inputs {
			if input.Kind == canonical.LineageInputKindEvidence && input.Evidence != nil && input.Evidence.Evidence == want.Evidence && input.Evidence.Content == want.Content {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func combinedProvenance(left, right ObservationInput, mapping Mapping) (canonical.Provenance, []canonical.EvidenceRef, []canonical.EvidenceRef, error) {
	leftRefs, err := evidenceReferences(left.Provenance)
	if err != nil {
		return canonical.Provenance{}, nil, nil, err
	}
	rightRefs, err := evidenceReferences(right.Provenance)
	if err != nil {
		return canonical.Provenance{}, nil, nil, err
	}
	provenance, err := mergeProvenances([]canonical.Provenance{left.Provenance, right.Provenance}, mapping, "cmp_evidence_quality_checker", "Deterministic cross-source evidence quality checker", canonical.ComponentKindValidator, canonical.VersionIdentity{Namespace: "jax.evidencequality", Value: "1.0.0"})
	if err != nil {
		return canonical.Provenance{}, nil, nil, err
	}
	return provenance, leftRefs, rightRefs, nil
}

func mergeProvenances(sources []canonical.Provenance, mapping Mapping, producerID, producerName string, producerKind canonical.ComponentKind, producerVersion canonical.VersionIdentity) (canonical.Provenance, error) {
	inputs := make([]canonical.LineageInput, 0)
	seen := map[string]bool{}
	for _, source := range sources {
		for _, input := range source.Inputs {
			encoded, encodeErr := json.Marshal(input)
			if encodeErr != nil {
				return canonical.Provenance{}, encodeErr
			}
			key := string(encoded)
			if !seen[key] {
				seen[key] = true
				inputs = append(inputs, input)
			}
		}
	}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		return canonical.Provenance{}, err
	}
	mappingComponent := canonical.ComponentIdentity{ID: "cmp_" + canonical.DigestBytes([]byte(mapping.ID + "\x00" + mapping.Version.Value)).Value[:24], Kind: canonical.ComponentKindMapping, Name: "Approved evidence equivalence mapping", Version: mapping.Version}
	producer := canonical.ComponentIdentity{ID: producerID, Kind: producerKind, Name: producerName, Version: producerVersion}
	provenance := canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte(fingerprint.Value)).Value[:24], Inputs: inputs, InputFingerprint: fingerprint, Producer: producer, Components: []canonical.ComponentIdentity{mappingComponent}}
	if err := provenance.Validate(); err != nil {
		return canonical.Provenance{}, err
	}
	return provenance, nil
}

func inputReference(input ObservationInput) EvidenceObservationReference {
	return EvidenceObservationReference{ID: input.ID, Series: input.Series, ObservationDate: input.ObservationDate, SourcePayload: input.SourcePayload}
}

func stableCheckID(mappingID string, mappingVersion canonical.VersionIdentity, left, right ObservationInput, outcome ComparisonOutcome, reason ReasonCode) string {
	envelope := struct {
		MappingID      string
		MappingVersion canonical.VersionIdentity
		Left, Right    ObservationInput
		Outcome        ComparisonOutcome
		Reason         ReasonCode
	}{mappingID, mappingVersion, left, right, outcome, reason}
	encoded, _ := json.Marshal(envelope)
	return CheckIDPrefix + canonical.DigestBytes(encoded).Value[:32]
}

func mappingCompatibility(mapping Mapping, left, right ObservationInput) (ReasonCode, bool) {
	if !sameSeries(left.Series, mapping.LeftSeries) || !sameSeries(right.Series, mapping.RightSeries) {
		return ReasonSeriesIdentityMismatch, false
	}
	if left.Semantics.MeasurementType != mapping.LeftSemantics.MeasurementType || right.Semantics.MeasurementType != mapping.RightSemantics.MeasurementType || mapping.LeftSemantics.MeasurementType != mapping.RightSemantics.MeasurementType {
		return ReasonMeasurementMismatch, false
	}
	if left.Semantics.Units != mapping.LeftSemantics.Units || right.Semantics.Units != mapping.RightSemantics.Units {
		return ReasonUnitMismatch, false
	}
	if left.Semantics.FrequencyCode != mapping.LeftSemantics.FrequencyCode || right.Semantics.FrequencyCode != mapping.RightSemantics.FrequencyCode || mapping.LeftSemantics.FrequencyCode != mapping.RightSemantics.FrequencyCode {
		return ReasonFrequencyMismatch, false
	}
	if left.Semantics.ObservationSemantics != mapping.LeftSemantics.ObservationSemantics || right.Semantics.ObservationSemantics != mapping.RightSemantics.ObservationSemantics || mapping.LeftSemantics.ObservationSemantics != mapping.RightSemantics.ObservationSemantics {
		return ReasonSemanticsMismatch, false
	}
	if left.Semantics.ReleaseSemantics != mapping.LeftSemantics.ReleaseSemantics || right.Semantics.ReleaseSemantics != mapping.RightSemantics.ReleaseSemantics || mapping.LeftSemantics.ReleaseSemantics != mapping.RightSemantics.ReleaseSemantics {
		return ReasonReleaseSemanticsMismatch, false
	}
	if left.Semantics.AdjustmentMethod != mapping.LeftSemantics.AdjustmentMethod || right.Semantics.AdjustmentMethod != mapping.RightSemantics.AdjustmentMethod {
		return ReasonAdjustmentMismatch, false
	}
	if mapping.Policy.RequireDate && left.ObservationDate != right.ObservationDate {
		return ReasonObservationDateMismatch, false
	}
	if mapping.Policy.RequireInfoState && (left.InformationState.Mode != right.InformationState.Mode || left.InformationState.Date != right.InformationState.Date) {
		return ReasonInformationStateMismatch, false
	}
	if mapping.Policy.RequireRealtime {
		if !sameDateRange(left.RealtimePeriod, right.RealtimePeriod) {
			return ReasonRealtimeStateMismatch, false
		}
	}
	if left.ObservationDate != right.ObservationDate {
		return ReasonObservationDateMismatch, false
	}
	return "", true
}

func sameDateRange(left, right *DateRange) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func makeResult(mapping *Mapping, left, right ObservationInput, outcome ComparisonOutcome, reason ReasonCode, evaluatedAt time.Time) (CrossSourceCheck, error) {
	if evaluatedAt.Location() != time.UTC || evaluatedAt.IsZero() {
		return CrossSourceCheck{}, fmt.Errorf("evaluated_at must be a non-zero UTC timestamp")
	}
	mappingID := ""
	var mappingVersion *canonical.VersionIdentity
	measurement := ""
	policy := ComparisonPolicy{ContractVersion: ComparisonPolicyContractV1, ID: PolicyIDPrefix + "unmapped", Version: canonical.VersionIdentity{Namespace: "jax.evidencequality", Value: "unmapped-v1"}, NumericMode: NumericExactDecimal, Tolerance: "0"}
	if mapping != nil {
		mappingID = mapping.ID
		version := mapping.Version
		mappingVersion = &version
		measurement = mapping.MeasurementIdentity
		policy = mapping.Policy
	}
	lineage := LineageUnknown
	if mapping != nil {
		lineage = effectiveLineage(*mapping, left, right)
	}
	provenance, leftEvidence, rightEvidence, err := combinedProvenance(left, right, mappingValue(mapping))
	if err != nil {
		return CrossSourceCheck{}, err
	}
	check := CrossSourceCheck{ContractVersion: CrossSourceCheckContractV1, MappingID: mappingID, MappingVersion: mappingVersion, Left: inputReference(left), Right: inputReference(right), MeasurementIdentity: measurement, ObservationDate: left.ObservationDate, Outcome: outcome, Lineage: lineage, IndependentSourceCount: independentSourceCount(lineage), LeftValue: left.Value.SourceValue, RightValue: right.Value.SourceValue, TolerancePolicy: PolicyReference{ID: policy.ID, Version: policy.Version, Tolerance: policy.Tolerance}, ReasonCode: reason, EvaluatedAt: evaluatedAt.UTC(), LeftEvidence: leftEvidence, RightEvidence: rightEvidence, Provenance: provenance}
	if outcome == OutcomeAgree || outcome == OutcomeDiverge {
		check.AbsoluteDifference, err = decimalDifference(left.Value.SourceValue, right.Value.SourceValue)
		if err != nil {
			return CrossSourceCheck{}, err
		}
	}
	check.ID = stableCheckID(mappingID, policy.Version, left, right, outcome, reason)
	if err := check.Validate(); err != nil {
		return CrossSourceCheck{}, err
	}
	return check, nil
}

func mappingValue(mapping *Mapping) Mapping {
	if mapping == nil {
		return Mapping{ID: MappingIDPrefix + "unmapped", Version: canonical.VersionIdentity{Namespace: "jax.evidencequality", Value: "unmapped-v1"}}
	}
	return *mapping
}

// Check applies an approved mapping. Missing registry mappings are explicit
// NOT_COMPARABLE results; malformed inputs and malformed mappings return an
// error so an operational failure cannot masquerade as a quality result.
func Check(registry *Registry, left, right ObservationInput, evaluatedAt time.Time) (CrossSourceCheck, error) {
	if err := left.Validate(); err != nil {
		return CrossSourceCheck{}, err
	}
	if err := right.Validate(); err != nil {
		return CrossSourceCheck{}, err
	}
	mapping, found := registry.Lookup(left.Series, right.Series)
	if !found {
		return makeResult(nil, left, right, OutcomeNotComparable, ReasonMappingNotApproved, evaluatedAt)
	}
	if err := mapping.Validate(); err != nil {
		return CrossSourceCheck{}, err
	}
	if reason, compatible := mappingCompatibility(mapping, left, right); !compatible {
		return makeResult(&mapping, left, right, OutcomeNotComparable, reason, evaluatedAt)
	}
	if !left.Value.Present && !right.Value.Present {
		return makeResult(&mapping, left, right, OutcomeBothMissing, ReasonBothMissing, evaluatedAt)
	}
	if !left.Value.Present {
		return makeResult(&mapping, left, right, OutcomeLeftMissing, ReasonLeftMissing, evaluatedAt)
	}
	if !right.Value.Present {
		return makeResult(&mapping, left, right, OutcomeRightMissing, ReasonRightMissing, evaluatedAt)
	}
	a, _ := parseDecimal(left.Value.SourceValue)
	b, _ := parseDecimal(right.Value.SourceValue)
	if a.Cmp(b) == 0 {
		return makeResult(&mapping, left, right, OutcomeAgree, ReasonValuesEqual, evaluatedAt)
	}
	return makeResult(&mapping, left, right, OutcomeDiverge, ReasonValuesDiffer, evaluatedAt)
}

type RangeCompleteness string

const (
	RangeCompletenessComplete   RangeCompleteness = "COMPLETE"
	RangeCompletenessIncomplete RangeCompleteness = "INCOMPLETE"
	RangeCompletenessUnknown    RangeCompleteness = "UNKNOWN"
)

type RangeRequest struct {
	MappingID     string
	Start         string
	End           string
	Left          []ObservationInput
	Right         []ObservationInput
	LeftComplete  RangeCompleteness
	RightComplete RangeCompleteness
	EvaluatedAt   time.Time
}

type RangeCheck struct {
	ContractVersion  canonical.ContractVersion `json:"contract_version"`
	ID               string                    `json:"id"`
	MappingID        string                    `json:"mapping_id"`
	OverlapStart     string                    `json:"overlap_start"`
	OverlapEnd       string                    `json:"overlap_end"`
	CandidateDates   []string                  `json:"candidate_dates"`
	ComparedRows     int                       `json:"compared_rows"`
	AgreementCount   int                       `json:"agreement_count"`
	DivergenceCount  int                       `json:"divergence_count"`
	LeftOnly         []string                  `json:"left_only"`
	RightOnly        []string                  `json:"right_only"`
	BothMissingCount int                       `json:"both_missing_count"`
	Completeness     RangeCompleteness         `json:"completeness"`
	Diagnostics      []string                  `json:"diagnostics"`
	EvaluatedAt      time.Time                 `json:"evaluated_at"`
	Provenance       canonical.Provenance      `json:"provenance"`
}

func (check RangeCheck) Validate() error {
	if check.ContractVersion != RangeCheckContractV1 {
		return fmt.Errorf("range check contract version must be %q", RangeCheckContractV1)
	}
	if !strings.HasPrefix(check.ID, RangeIDPrefix) || len(check.ID) <= len(RangeIDPrefix) {
		return fmt.Errorf("range check ID must use %s prefix", RangeIDPrefix)
	}
	if check.MappingID == "" {
		return fmt.Errorf("mapping ID is required")
	}
	if (check.OverlapStart == "") != (check.OverlapEnd == "") {
		return fmt.Errorf("overlap start and end must be provided together")
	}
	if check.OverlapStart != "" {
		if check.OverlapEnd == "" {
			return fmt.Errorf("overlap end is required when overlap start is present")
		}
		if err := validateDate(check.OverlapStart); err != nil {
			return err
		}
		if err := validateDate(check.OverlapEnd); err != nil {
			return err
		}
		if check.OverlapEnd < check.OverlapStart {
			return fmt.Errorf("overlap end precedes start")
		}
	}
	if check.ComparedRows < 0 || check.AgreementCount < 0 || check.DivergenceCount < 0 || check.BothMissingCount < 0 {
		return fmt.Errorf("range counts must not be negative")
	}
	if check.AgreementCount+check.DivergenceCount+check.BothMissingCount > check.ComparedRows {
		return fmt.Errorf("range outcome counts exceed compared rows")
	}
	if check.Completeness != RangeCompletenessComplete && check.Completeness != RangeCompletenessIncomplete && check.Completeness != RangeCompletenessUnknown {
		return fmt.Errorf("range completeness is invalid")
	}
	for i := 1; i < len(check.CandidateDates); i++ {
		if check.CandidateDates[i] <= check.CandidateDates[i-1] {
			return fmt.Errorf("candidate dates must be strictly ascending")
		}
		if err := validateDate(check.CandidateDates[i]); err != nil {
			return err
		}
	}
	if len(check.CandidateDates) > 0 {
		if err := validateDate(check.CandidateDates[0]); err != nil {
			return err
		}
	}
	if check.EvaluatedAt.IsZero() || check.EvaluatedAt.Location() != time.UTC {
		return fmt.Errorf("evaluated_at must be a non-zero UTC timestamp")
	}
	if err := check.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	return nil
}

func (request RangeRequest) Validate() error {
	if err := validateDate(request.Start); err != nil {
		return err
	}
	if err := validateDate(request.End); err != nil {
		return err
	}
	if request.End < request.Start {
		return fmt.Errorf("range end precedes start")
	}
	if request.MappingID == "" {
		return fmt.Errorf("mapping ID is required")
	}
	if request.LeftComplete != RangeCompletenessComplete && request.LeftComplete != RangeCompletenessIncomplete && request.LeftComplete != RangeCompletenessUnknown {
		return fmt.Errorf("left completeness is invalid")
	}
	if request.RightComplete != RangeCompletenessComplete && request.RightComplete != RangeCompletenessIncomplete && request.RightComplete != RangeCompletenessUnknown {
		return fmt.Errorf("right completeness is invalid")
	}
	if request.EvaluatedAt.IsZero() || request.EvaluatedAt.Location() != time.UTC {
		return fmt.Errorf("evaluated_at must be a non-zero UTC timestamp")
	}
	return nil
}

// CheckRange compares exact observation-date keys. It performs no resampling,
// aggregation, correlation, or statistical analysis.
func CheckRange(registry *Registry, request RangeRequest) (RangeCheck, []CrossSourceCheck, error) {
	if err := request.Validate(); err != nil {
		return RangeCheck{}, nil, err
	}
	mapping, ok := registry.Get(request.MappingID)
	if !ok {
		return RangeCheck{}, nil, fmt.Errorf("mapping %q is not approved", request.MappingID)
	}
	if err := mapping.Validate(); err != nil {
		return RangeCheck{}, nil, err
	}
	left := map[string]ObservationInput{}
	right := map[string]ObservationInput{}
	for _, input := range request.Left {
		if err := input.Validate(); err != nil {
			return RangeCheck{}, nil, err
		}
		if input.ObservationDate >= request.Start && input.ObservationDate <= request.End {
			if _, exists := left[input.ObservationDate]; exists {
				return RangeCheck{}, nil, fmt.Errorf("duplicate left observation date %s", input.ObservationDate)
			}
			left[input.ObservationDate] = input
		}
	}
	for _, input := range request.Right {
		if err := input.Validate(); err != nil {
			return RangeCheck{}, nil, err
		}
		if input.ObservationDate >= request.Start && input.ObservationDate <= request.End {
			if _, exists := right[input.ObservationDate]; exists {
				return RangeCheck{}, nil, fmt.Errorf("duplicate right observation date %s", input.ObservationDate)
			}
			right[input.ObservationDate] = input
		}
	}
	dates := map[string]bool{}
	for date := range left {
		dates[date] = true
	}
	for date := range right {
		dates[date] = true
	}
	candidateDates := make([]string, 0, len(dates))
	for date := range dates {
		candidateDates = append(candidateDates, date)
	}
	sort.Strings(candidateDates)
	checks := make([]CrossSourceCheck, 0, len(candidateDates))
	result := RangeCheck{ContractVersion: RangeCheckContractV1, MappingID: mapping.ID, CandidateDates: candidateDates, Completeness: RangeCompletenessComplete, EvaluatedAt: request.EvaluatedAt}
	if request.LeftComplete != RangeCompletenessComplete || request.RightComplete != RangeCompletenessComplete {
		result.Completeness = RangeCompletenessIncomplete
	}
	for _, date := range candidateDates {
		l, lok := left[date]
		r, rok := right[date]
		if !lok {
			result.RightOnly = append(result.RightOnly, date)
			continue
		}
		if !rok {
			result.LeftOnly = append(result.LeftOnly, date)
			continue
		}
		check, err := Check(registry, l, r, request.EvaluatedAt)
		if err != nil {
			return RangeCheck{}, nil, err
		}
		checks = append(checks, check)
		result.ComparedRows++
		switch check.Outcome {
		case OutcomeAgree:
			result.AgreementCount++
		case OutcomeDiverge:
			result.DivergenceCount++
		case OutcomeBothMissing:
			result.BothMissingCount++
		}
	}
	commonDates := make([]string, 0)
	for _, date := range candidateDates {
		if _, leftPresent := left[date]; leftPresent {
			if _, rightPresent := right[date]; rightPresent {
				commonDates = append(commonDates, date)
			}
		}
	}
	if len(commonDates) > 0 {
		result.OverlapStart, result.OverlapEnd = commonDates[0], commonDates[len(commonDates)-1]
	}
	if len(result.LeftOnly) > 0 || len(result.RightOnly) > 0 {
		result.Diagnostics = append(result.Diagnostics, "DATE_SET_DIFFERENCE")
	}
	if result.Completeness == RangeCompletenessComplete && len(result.LeftOnly) == 0 && len(result.RightOnly) == 0 {
		result.Diagnostics = append(result.Diagnostics, "BOTH_DECLARED_COMPLETE")
	}
	provenanceSources := make([]canonical.Provenance, 0, len(candidateDates)*2)
	for _, date := range candidateDates {
		if input, exists := left[date]; exists {
			provenanceSources = append(provenanceSources, input.Provenance)
		}
		if input, exists := right[date]; exists {
			provenanceSources = append(provenanceSources, input.Provenance)
		}
	}
	provenance, err := mergeProvenances(provenanceSources, mapping, "cmp_evidence_quality_range_checker", "Deterministic cross-source range quality checker", canonical.ComponentKindValidator, canonical.VersionIdentity{Namespace: "jax.evidencequality", Value: "1.0.0"})
	if err != nil {
		return RangeCheck{}, nil, err
	}
	result.Provenance = provenance
	envelope := struct {
		MappingID, Start, End string
		Dates                 []string
		Checks                []string
		Completeness          RangeCompleteness
	}{result.MappingID, request.Start, request.End, candidateDates, nil, result.Completeness}
	for _, check := range checks {
		envelope.Checks = append(envelope.Checks, check.ID)
	}
	encoded, _ := json.Marshal(envelope)
	result.ID = RangeIDPrefix + canonical.DigestBytes(encoded).Value[:32]
	if err := result.Validate(); err != nil {
		return RangeCheck{}, nil, err
	}
	return result, checks, nil
}
