package provider

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

// FreshnessEvaluator consumes only accepted records, static registries,
// policies, immutable status inputs, and caller-supplied evaluation time. It
// performs no acquisition, provider health check, retry, persistence, cleanup,
// model call, or current-clock lookup.
type FreshnessEvaluator struct {
	providers *Registry
	policies  *FreshnessPolicyRegistry
}

func NewFreshnessEvaluator(providers *Registry, policies *FreshnessPolicyRegistry) (*FreshnessEvaluator, error) {
	if providers == nil || providers.ContractVersion() != RegistryContractV1 {
		return nil, freshnessError(FreshnessErrorInvalidPolicy, canonical.ComponentIdentity{}, "", canonical.ImmutableContractRef{}, "a valid provider registry is required", nil)
	}
	if policies == nil {
		return nil, freshnessError(FreshnessErrorInvalidPolicy, canonical.ComponentIdentity{}, "", canonical.ImmutableContractRef{}, "a freshness policy registry is required", nil)
	}
	return &FreshnessEvaluator{providers: providers, policies: policies}, nil
}

func (evaluator *FreshnessEvaluator) Evaluate(request FreshnessEvaluationRequest) (FreshnessEvaluation, error) {
	if evaluator == nil || evaluator.providers == nil || evaluator.policies == nil {
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorInvalidPolicy, request.Policy, request.Record.Key.CapabilityID, request.Record.Normalized.Output, "freshness evaluator is not initialized", nil)
	}
	policy, err := evaluator.policies.Lookup(request.Record.Key.CapabilityID, request.Record.Key.Target, request.UseClass, request.Policy)
	if err != nil {
		return FreshnessEvaluation{}, err
	}
	return evaluator.evaluateWithPolicy(policy, request.EvaluationTime, request.Context, request.Record)
}

func (evaluator *FreshnessEvaluator) evaluateWithPolicy(policy FreshnessPolicy, evaluationTime time.Time, context FreshnessEvaluationContext, record TemporalRecord) (FreshnessEvaluation, error) {
	if !validEvaluationTime(evaluationTime) {
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorInvalidEvaluationTime, policy.Identity, policy.CapabilityID, record.Normalized.Output, "evaluation time is required and must use UTC", nil)
	}
	switch context {
	case FreshnessContextCurrentState, FreshnessContextHistoricalReplay:
	default:
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorInvalidEvaluationTime, policy.Identity, policy.CapabilityID, record.Normalized.Output, "evaluation context is not supported", nil)
	}
	if err := record.Key.validate(policy); err != nil {
		return FreshnessEvaluation{}, err
	}
	if err := evaluator.validateAcceptedRecord(policy, record); err != nil {
		return FreshnessEvaluation{}, err
	}
	if record.Normalized.RawRef.ReceivedAt.After(evaluationTime) {
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorRecordNotYetAvailable, policy.Identity, policy.CapabilityID, record.Normalized.Output, "immutable raw acquisition completed after evaluation time", nil)
	}
	if err := ensureRecordActiveAt(policy, record, evaluationTime, context); err != nil {
		return FreshnessEvaluation{}, err
	}

	evaluation := FreshnessEvaluation{
		ContractVersion: FreshnessEvaluationContractV1,
		Record:          record.Normalized.Output, RawPayloadID: record.Normalized.RawRef.ID,
		Normalizer: cloneComponentIdentity(record.Normalized.Normalizer), Policy: cloneComponentIdentity(policy.Identity),
		CapabilityID: policy.CapabilityID, UseClass: policy.UseClass, Key: record.Key,
		EvaluationTime: evaluationTime, EvaluationContext: context, TimestampRole: policy.TimestampRole,
		CanonicalQuality: record.Normalized.Quality, Lifecycle: cloneTemporalLifecycle(record.Lifecycle),
	}
	if policy.ValidityMode == FreshnessValidityNotApplicable {
		evaluation.State = TemporalNotApplicable
		evaluation.ReasonCode = FreshnessReasonNotApplicable
		return evaluation, nil
	}
	if policy.ValidityMode == FreshnessValidityUntilSuperseded && policy.TimestampRole == TimestampRoleNone {
		evaluation.State = TemporalFresh
		evaluation.ReasonCode = FreshnessReasonUntilSuperseded
		return evaluation, nil
	}

	authoritative, err := authoritativeTimestamp(record.Normalized.Record, policy.TimestampRole)
	if err != nil {
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorAmbiguousAuthoritativeTimestamp, policy.Identity, policy.CapabilityID, record.Normalized.Output, "canonical record has ambiguous authoritative timestamp evidence", err)
	}
	if authoritative == nil {
		if policy.MissingTimestamp == MissingTimestampUnknown {
			evaluation.State = TemporalUnknown
			evaluation.ReasonCode = FreshnessReasonMissingTimestamp
			return evaluation, nil
		}
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorMissingAuthoritativeTimestamp, policy.Identity, policy.CapabilityID, record.Normalized.Output, "required canonical authoritative timestamp is absent", nil)
	}
	authoritativeCopy := *authoritative
	evaluation.AuthoritativeTimestamp = &authoritativeCopy
	age, err := boundedAge(evaluationTime, authoritativeCopy)
	if err != nil {
		return FreshnessEvaluation{}, freshnessError(FreshnessErrorAgeOutOfRange, policy.Identity, policy.CapabilityID, record.Normalized.Output, "evaluation time and authoritative timestamp exceed supported duration range", err)
	}
	if age < 0 {
		lead := -age
		if lead > policy.AllowedFutureSkew {
			return FreshnessEvaluation{}, freshnessError(FreshnessErrorFutureTimestampBeyondSkew, policy.Identity, policy.CapabilityID, record.Normalized.Output, "authoritative timestamp is after evaluation time beyond explicit skew tolerance", nil)
		}
		evaluation.WithinFutureSkew = true
	}
	ageCopy := age
	evaluation.Age = &ageCopy

	if policy.ValidityMode == FreshnessValidityUntilSuperseded {
		evaluation.State = TemporalFresh
		evaluation.ReasonCode = FreshnessReasonUntilSuperseded
		return evaluation, nil
	}
	switch {
	case age < policy.FreshFor:
		evaluation.State = TemporalFresh
		evaluation.ReasonCode = FreshnessReasonWithinHorizon
	case age >= policy.ExpireAfter:
		evaluation.State = TemporalExpired
		evaluation.ReasonCode = FreshnessReasonAtOrBeyondExpiry
	default:
		evaluation.State = TemporalStale
		evaluation.ReasonCode = FreshnessReasonAtOrBeyondTTL
	}
	return evaluation, nil
}

func (evaluator *FreshnessEvaluator) validateAcceptedRecord(policy FreshnessPolicy, record TemporalRecord) error {
	result := record.Normalized
	if result.Status != NormalizationStatusAccepted || result.Quality != NormalizationQualityValidated ||
		result.Validation != (NormalizationValidation{RawVerified: true, Parsed: true, Mapped: true, CanonicalValidated: true, ProvenanceValidated: true}) {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "record was not fully accepted by the normalization pipeline", nil)
	}
	if result.Record == nil || (reflect.ValueOf(result.Record).Kind() == reflect.Ptr && reflect.ValueOf(result.Record).IsNil()) {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "accepted normalization envelope has no canonical record", nil)
	}
	if result.RawRef.CapabilityID != policy.CapabilityID || result.Target != policy.Target {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, policy.CapabilityID, result.Output, "normalized capability/target does not match freshness policy", nil)
	}
	capability, err := evaluator.providers.Capability(result.RawRef.Provider, policy.CapabilityID)
	if err != nil || capability.Support != SupportSupported {
		return freshnessError(FreshnessErrorPolicyCapabilityMismatch, policy.Identity, policy.CapabilityID, result.Output, "normalized provider capability is not registered and supported", err)
	}
	if err := result.RawRef.Validate(); err != nil {
		return freshnessError(FreshnessErrorInvalidProvenanceInput, policy.Identity, policy.CapabilityID, result.Output, "raw payload reference is invalid", err)
	}
	if err := result.Normalizer.Validate(); err != nil {
		return freshnessError(FreshnessErrorInvalidProvenanceInput, policy.Identity, policy.CapabilityID, result.Output, "normalizer identity is invalid", err)
	}
	if err := result.Record.Validate(); err != nil {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "canonical validation no longer succeeds", err)
	}
	contractRef, err := canonicalRecordRef(result.Record)
	if err != nil || contractRef != result.Output.Contract {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "immutable output identity does not identify the supplied canonical record", err)
	}
	if err := result.Output.Validate(); err != nil {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "immutable canonical output reference is invalid", err)
	}
	if err := result.Output.Content.VerifyCanonicalContract(result.Record); err != nil {
		return freshnessError(FreshnessErrorInvalidCanonicalInput, policy.Identity, policy.CapabilityID, result.Output, "canonical record bytes do not match immutable output identity", err)
	}
	if err := validateRawProvenanceLink(result.Record, result.RawRef, result.Normalizer); err != nil {
		return freshnessError(FreshnessErrorInvalidProvenanceInput, policy.Identity, policy.CapabilityID, result.Output, "canonical provenance does not retain the exact raw input and normalizer", err)
	}
	if err := validateRecordSemanticKey(record.Key, result.Record, result.Output.Contract); err != nil {
		return freshnessError(FreshnessErrorSemanticKeyMismatch, policy.Identity, policy.CapabilityID, result.Output, "canonical record does not match the supplied semantic key", err)
	}
	return record.Lifecycle.validate(policy, result.Output)
}

func ensureRecordActiveAt(policy FreshnessPolicy, record TemporalRecord, evaluationTime time.Time, context FreshnessEvaluationContext) error {
	if record.Lifecycle.State == TemporalRecordActive {
		return nil
	}
	if context == FreshnessContextHistoricalReplay && record.Lifecycle.ChangedAt != nil && record.Lifecycle.ChangedAt.After(evaluationTime) {
		return nil
	}
	code := FreshnessErrorRecordInvalidated
	switch record.Lifecycle.State {
	case TemporalRecordSuperseded:
		code = FreshnessErrorRecordSuperseded
	case TemporalRecordRetracted:
		code = FreshnessErrorRecordRetracted
	case TemporalRecordDisputed:
		code = FreshnessErrorRecordDisputed
	case TemporalRecordInvalidated:
		code = FreshnessErrorRecordInvalidated
	}
	return freshnessError(code, policy.Identity, policy.CapabilityID, record.Normalized.Output, "record lifecycle state prohibits selection in this evaluation context", nil)
}

func (evaluator *FreshnessEvaluator) Resolve(request FreshnessResolutionRequest) (FreshnessResolution, error) {
	if evaluator == nil || evaluator.providers == nil || evaluator.policies == nil {
		return FreshnessResolution{}, freshnessError(FreshnessErrorInvalidPolicy, request.Policy, request.Key.CapabilityID, canonical.ImmutableContractRef{}, "freshness evaluator is not initialized", nil)
	}
	policy, err := evaluator.policies.Lookup(request.Key.CapabilityID, request.Key.Target, request.UseClass, request.Policy)
	if err != nil {
		return FreshnessResolution{}, err
	}
	if !samePolicyIdentity(policy.LastKnownGood.Identity, request.ExpectedFallbackPolicy) {
		return FreshnessResolution{}, freshnessError(FreshnessErrorPolicyVersionMismatch, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "expected last-known-good policy identity/version does not match registered policy", nil)
	}
	if !validEvaluationTime(request.EvaluationTime) {
		return FreshnessResolution{}, freshnessError(FreshnessErrorInvalidEvaluationTime, policy.Identity, policy.CapabilityID, canonical.ImmutableContractRef{}, "evaluation time is required and must use UTC", nil)
	}
	if err := request.Key.validate(policy); err != nil {
		return FreshnessResolution{}, err
	}

	reason := ResolutionCurrentMissing
	var currentRecord *canonical.ImmutableContractRef
	candidates := append([]TemporalRecord(nil), request.Historical...)
	if request.Current != nil {
		currentRef := request.Current.Normalized.Output
		currentRecord = &currentRef
		if request.Current.Key != request.Key {
			reason = ResolutionCurrentInvalid
		} else {
			evaluated, evalErr := evaluator.evaluateWithPolicy(policy, request.EvaluationTime, request.Context, *request.Current)
			if evalErr == nil {
				switch evaluated.State {
				case TemporalFresh, TemporalNotApplicable:
					return directFreshnessResolution(policy, request.EvaluationTime, request.Key, currentRef, evaluated), nil
				case TemporalStale:
					reason = ResolutionCurrentStale
					candidates = append(candidates, *request.Current)
				case TemporalExpired:
					reason = ResolutionCurrentExpired
				case TemporalUnknown:
					reason = ResolutionCurrentUnknown
				}
			} else if tolerableCandidateFailure(evalErr) {
				reason = ResolutionCurrentInvalid
			} else {
				return FreshnessResolution{}, evalErr
			}
		}
	}
	if policy.LastKnownGood.Mode == FallbackProhibited {
		return FreshnessResolution{}, freshnessError(FreshnessErrorFallbackProhibited, policy.Identity, policy.CapabilityID, dereferenceImmutableRef(currentRecord), "current record is not temporally acceptable and policy prohibits fallback", nil)
	}

	type qualifiedCandidate struct {
		record        TemporalRecord
		evaluation    FreshnessEvaluation
		qualification LKGQualification
		prior         *FreshnessEvaluation
	}
	qualified := make([]qualifiedCandidate, 0, len(candidates))
	seen := make(map[string]qualifiedCandidate)
	exceededAge := false
	for _, candidate := range candidates {
		if candidate.Key != request.Key {
			continue
		}
		evaluation, evalErr := evaluator.evaluateWithPolicy(policy, request.EvaluationTime, request.Context, candidate)
		if evalErr != nil {
			if tolerableCandidateFailure(evalErr) {
				continue
			}
			return FreshnessResolution{}, evalErr
		}
		if evaluation.Age == nil || evaluation.State == TemporalExpired || evaluation.State == TemporalUnknown || evaluation.State == TemporalNotApplicable {
			continue
		}
		if *evaluation.Age > policy.LastKnownGood.MaximumAge {
			exceededAge = true
			continue
		}
		item := qualifiedCandidate{record: candidate, evaluation: evaluation, qualification: LKGQualificationCurrentlyFresh}
		switch evaluation.State {
		case TemporalFresh:
			// A recent immutable record may remain fresh while the current
			// acquisition is absent or invalid. It is still visibly fallback.
		case TemporalStale:
			if policy.LastKnownGood.Mode != FallbackFreshOrStale {
				continue
			}
			if err := evaluator.validatePriorFreshEvaluation(policy, candidate, evaluation, request.EvaluationTime); err != nil {
				continue
			}
			prior := cloneFreshnessEvaluation(*candidate.PriorFreshEvaluation)
			item.qualification = LKGQualificationPriorFresh
			item.prior = &prior
		default:
			continue
		}
		key := immutableFreshnessRecordKey(evaluation.Record)
		if existing, ok := seen[key]; ok {
			if !reflect.DeepEqual(existing.evaluation, item.evaluation) || existing.qualification != item.qualification {
				return FreshnessResolution{}, freshnessError(FreshnessErrorAmbiguousLKG, policy.Identity, policy.CapabilityID, evaluation.Record, "same immutable record has conflicting temporal qualification", nil)
			}
			continue
		}
		seen[key] = item
		qualified = append(qualified, item)
	}
	if len(qualified) == 0 {
		code := FreshnessErrorNoAcceptableLKG
		detail := "no valid same-key historical candidate satisfies the fallback policy"
		if exceededAge {
			code = FreshnessErrorFallbackAgeExceeded
			detail = "otherwise eligible candidate exceeds the explicit maximum fallback age"
		}
		return FreshnessResolution{}, freshnessError(code, policy.Identity, policy.CapabilityID, dereferenceImmutableRef(currentRecord), detail, nil)
	}
	sort.Slice(qualified, func(i, j int) bool {
		left := *qualified[i].evaluation.AuthoritativeTimestamp
		right := *qualified[j].evaluation.AuthoritativeTimestamp
		if !left.Equal(right) {
			return left.After(right)
		}
		return immutableFreshnessRecordKey(qualified[i].evaluation.Record) < immutableFreshnessRecordKey(qualified[j].evaluation.Record)
	})
	if len(qualified) > 1 && qualified[0].evaluation.AuthoritativeTimestamp.Equal(*qualified[1].evaluation.AuthoritativeTimestamp) {
		return FreshnessResolution{}, freshnessError(FreshnessErrorAmbiguousLKG, policy.Identity, policy.CapabilityID, qualified[0].evaluation.Record, "multiple different immutable records share the newest authoritative timestamp", nil)
	}
	selected := qualified[0]
	age := *selected.evaluation.Age
	return FreshnessResolution{
		ContractVersion: FreshnessResolutionContractV1,
		Policy:          cloneComponentIdentity(policy.Identity), FallbackPolicy: cloneComponentIdentity(policy.LastKnownGood.Identity),
		EvaluationTime: request.EvaluationTime, Key: request.Key, CurrentRecord: cloneImmutableRefPointer(currentRecord),
		Selected: cloneFreshnessEvaluation(selected.evaluation), FallbackStatus: FallbackUsed, FallbackAge: &age,
		Qualification: selected.qualification, PriorFreshEvaluation: selected.prior, ReasonCode: reason,
	}, nil
}

func directFreshnessResolution(policy FreshnessPolicy, evaluationTime time.Time, key FreshnessKey, current canonical.ImmutableContractRef, evaluation FreshnessEvaluation) FreshnessResolution {
	return FreshnessResolution{
		ContractVersion: FreshnessResolutionContractV1,
		Policy:          cloneComponentIdentity(policy.Identity), FallbackPolicy: cloneComponentIdentity(policy.LastKnownGood.Identity),
		EvaluationTime: evaluationTime, Key: key, CurrentRecord: &current, Selected: cloneFreshnessEvaluation(evaluation),
		FallbackStatus: FallbackNotUsed, Qualification: LKGQualificationNotApplicable, ReasonCode: ResolutionCurrentAccepted,
	}
}

func (evaluator *FreshnessEvaluator) validatePriorFreshEvaluation(policy FreshnessPolicy, candidate TemporalRecord, current FreshnessEvaluation, evaluationTime time.Time) error {
	prior := candidate.PriorFreshEvaluation
	if prior == nil || prior.ContractVersion != FreshnessEvaluationContractV1 || prior.State != TemporalFresh ||
		prior.CanonicalQuality != NormalizationQualityValidated || prior.Record != current.Record || prior.Key != current.Key ||
		prior.CapabilityID != policy.CapabilityID || prior.UseClass != policy.UseClass || !samePolicyIdentity(prior.Policy, policy.Identity) ||
		prior.AuthoritativeTimestamp == nil || current.AuthoritativeTimestamp == nil || !prior.AuthoritativeTimestamp.Equal(*current.AuthoritativeTimestamp) ||
		prior.EvaluationTime.After(evaluationTime) {
		return freshnessError(FreshnessErrorPriorAcceptanceMismatch, policy.Identity, policy.CapabilityID, current.Record, "stale candidate lacks a matching earlier FRESH evaluation", nil)
	}
	recomputed, err := evaluator.evaluateWithPolicy(policy, prior.EvaluationTime, prior.EvaluationContext, candidate)
	if err != nil || !reflect.DeepEqual(recomputed, *prior) {
		return freshnessError(FreshnessErrorPriorAcceptanceMismatch, policy.Identity, policy.CapabilityID, current.Record, "prior FRESH evaluation is not reproducible from the candidate, time, context, and policy", err)
	}
	return nil
}

func tolerableCandidateFailure(err error) bool {
	typed, ok := asFreshnessError(err)
	if !ok {
		return false
	}
	switch typed.Code {
	case FreshnessErrorInvalidCanonicalInput, FreshnessErrorInvalidProvenanceInput, FreshnessErrorSemanticKeyMismatch,
		FreshnessErrorMissingAuthoritativeTimestamp, FreshnessErrorFutureTimestampBeyondSkew,
		FreshnessErrorRecordNotYetAvailable,
		FreshnessErrorRecordSuperseded, FreshnessErrorRecordRetracted, FreshnessErrorRecordDisputed, FreshnessErrorRecordInvalidated:
		return true
	default:
		return false
	}
}

func validateRecordSemanticKey(key FreshnessKey, record canonical.Contract, output canonical.ContractRef) error {
	matchesSubject := func(ref canonical.ContractRef) bool { return ref == key.Subject }
	switch value := record.(type) {
	case canonical.Observation:
		if !matchesSubject(value.Subject) || value.Metric != key.Qualifier {
			return fmt.Errorf("observation subject/metric differs from semantic key")
		}
	case *canonical.Observation:
		return validateRecordSemanticKey(key, *value, output)
	case canonical.Evidence:
		for _, link := range value.Links {
			if matchesSubject(link.Target) {
				return nil
			}
		}
		return fmt.Errorf("evidence does not link the semantic subject")
	case *canonical.Evidence:
		return validateRecordSemanticKey(key, *value, output)
	case canonical.Event:
		if key.Subject == output {
			return nil
		}
		for _, subject := range value.Subjects {
			if matchesSubject(subject) {
				return nil
			}
		}
		return fmt.Errorf("event does not identify the semantic subject")
	case *canonical.Event:
		return validateRecordSemanticKey(key, *value, output)
	case canonical.Instrument, *canonical.Instrument, canonical.Issuer, *canonical.Issuer:
		if key.Subject != output {
			return fmt.Errorf("reference-data semantic subject must identify the canonical record")
		}
	default:
		return fmt.Errorf("canonical family %T is not provider freshness data", record)
	}
	return nil
}

func authoritativeTimestamp(record canonical.Contract, role AuthoritativeTimestampRole) (*time.Time, error) {
	copyTime := func(value time.Time) *time.Time { copy := value; return &copy }
	switch role {
	case TimestampRoleObservedAt:
		switch value := record.(type) {
		case canonical.Observation:
			return copyTime(value.ObservedAt), nil
		case *canonical.Observation:
			return copyTime(value.ObservedAt), nil
		}
	case TimestampRolePublishedAt:
		switch value := record.(type) {
		case canonical.Observation:
			return cloneTimePointer(value.PublishedAt), nil
		case *canonical.Observation:
			return cloneTimePointer(value.PublishedAt), nil
		case canonical.Evidence:
			return cloneTimePointer(value.PublishedAt), nil
		case *canonical.Evidence:
			return cloneTimePointer(value.PublishedAt), nil
		}
	case TimestampRoleCollectedAt:
		switch value := record.(type) {
		case canonical.Observation:
			return copyTime(value.CollectedAt), nil
		case *canonical.Observation:
			return copyTime(value.CollectedAt), nil
		case canonical.Evidence:
			return copyTime(value.CollectedAt), nil
		case *canonical.Evidence:
			return copyTime(value.CollectedAt), nil
		}
	case TimestampRoleOccurredAt:
		switch value := record.(type) {
		case canonical.Event:
			return cloneTimePointer(value.OccurredAt), nil
		case *canonical.Event:
			return cloneTimePointer(value.OccurredAt), nil
		}
	case TimestampRoleEffectiveAt:
		switch value := record.(type) {
		case canonical.Event:
			return cloneTimePointer(value.EffectiveAt), nil
		case *canonical.Event:
			return cloneTimePointer(value.EffectiveAt), nil
		}
	case TimestampRoleEffectiveFrom:
		switch value := record.(type) {
		case canonical.Instrument:
			return cloneTimePointer(value.Effective.From), nil
		case *canonical.Instrument:
			return cloneTimePointer(value.Effective.From), nil
		case canonical.Issuer:
			return cloneTimePointer(value.Effective.From), nil
		case *canonical.Issuer:
			return cloneTimePointer(value.Effective.From), nil
		}
	case TimestampRoleDatasetAsOf:
		return datasetAsOfTimestamp(record)
	case TimestampRoleNone:
		return nil, nil
	}
	return nil, nil
}

func datasetAsOfTimestamp(record canonical.Contract) (*time.Time, error) {
	values := make([]time.Time, 0, 2)
	appendValue := func(value *time.Time) {
		if value == nil {
			return
		}
		for _, existing := range values {
			if existing.Equal(*value) {
				return
			}
		}
		values = append(values, *value)
	}
	appendProvenance := func(provenance *canonical.Provenance) {
		if provenance == nil {
			return
		}
		for _, input := range provenance.Inputs {
			if input.Kind == canonical.LineageInputKindDataset && input.Dataset != nil {
				appendValue(input.Dataset.AsOf)
			}
		}
	}
	switch value := record.(type) {
	case canonical.Observation:
		appendProvenance(value.Provenance)
	case *canonical.Observation:
		appendProvenance(value.Provenance)
	case canonical.Evidence:
		if value.DatasetSnapshot != nil {
			appendValue(value.DatasetSnapshot.AsOf)
		}
		appendProvenance(value.Provenance)
	case *canonical.Evidence:
		if value.DatasetSnapshot != nil {
			appendValue(value.DatasetSnapshot.AsOf)
		}
		appendProvenance(value.Provenance)
	}
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > 1 {
		return nil, fmt.Errorf("multiple distinct dataset as-of timestamps are present")
	}
	value := values[0]
	return &value, nil
}

func boundedAge(evaluationTime, authoritative time.Time) (time.Duration, error) {
	age := evaluationTime.Sub(authoritative)
	if age == time.Duration(math.MaxInt64) || age == time.Duration(math.MinInt64) {
		return 0, fmt.Errorf("duration overflow")
	}
	return age, nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTemporalLifecycle(value TemporalRecordLifecycle) TemporalRecordLifecycle {
	return TemporalRecordLifecycle{State: value.State, ChangedAt: cloneTimePointer(value.ChangedAt)}
}

func cloneFreshnessEvaluation(value FreshnessEvaluation) FreshnessEvaluation {
	copy := value
	copy.Normalizer = cloneComponentIdentity(value.Normalizer)
	copy.Policy = cloneComponentIdentity(value.Policy)
	copy.AuthoritativeTimestamp = cloneTimePointer(value.AuthoritativeTimestamp)
	if value.Age != nil {
		age := *value.Age
		copy.Age = &age
	}
	copy.Lifecycle = cloneTemporalLifecycle(value.Lifecycle)
	return copy
}

func cloneImmutableRefPointer(value *canonical.ImmutableContractRef) *canonical.ImmutableContractRef {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func dereferenceImmutableRef(value *canonical.ImmutableContractRef) canonical.ImmutableContractRef {
	if value == nil {
		return canonical.ImmutableContractRef{}
	}
	return *value
}

func immutableFreshnessRecordKey(ref canonical.ImmutableContractRef) string {
	return string(ref.Contract.Kind) + "\x00" + ref.Contract.ID + "\x00" + string(ref.Contract.ContractVersion) + "\x00" + ref.Revision.Namespace + "\x00" + ref.Revision.Value + "\x00" + ref.Content.Digest.Value
}
