package canonical

import (
	"encoding/json"
	"fmt"
	"time"
)

type AuditAction string

const (
	AuditActionRecordCreated         AuditAction = "record_created"
	AuditActionRecordSuperseded      AuditAction = "record_superseded"
	AuditActionProcessingCompleted   AuditAction = "processing_completed"
	AuditActionProcessingFailed      AuditAction = "processing_failed"
	AuditActionCompatibilityAssessed AuditAction = "compatibility_assessed"
	AuditActionCompatibilityApplied  AuditAction = "compatibility_applied"
)

type AuditOutcome string

const (
	AuditOutcomeSucceeded AuditOutcome = "SUCCEEDED"
	AuditOutcomeFailed    AuditOutcome = "FAILED"
	AuditOutcomeRejected  AuditOutcome = "REJECTED"
)

type AuditFailureCode string

const (
	AuditFailureValidationRejected                  AuditFailureCode = "validation_rejected"
	AuditFailureUnsupportedContractVersion          AuditFailureCode = "unsupported_contract_version"
	AuditFailureProvenanceMismatch                  AuditFailureCode = "provenance_mismatch"
	AuditFailureMissingImmutableInput               AuditFailureCode = "missing_immutable_input"
	AuditFailureComponent                           AuditFailureCode = "component_failure"
	AuditFailureModelProvider                       AuditFailureCode = "model_provider_failure"
	AuditFailureCompatibilityTranslationUnavailable AuditFailureCode = "compatibility_translation_unavailable"
	AuditFailureContentDigestMismatch               AuditFailureCode = "content_digest_mismatch"
	AuditFailureReplayNonReproducible               AuditFailureCode = "replay_non_reproducible"
)

// AuditFailure identifies a material failure without credentials, evidence
// payloads, provider response bodies, or stack traces.
type AuditFailure struct {
	ID     string           `json:"id"`
	Code   AuditFailureCode `json:"code"`
	Detail string           `json:"detail"`
}

// AuditSubjectRef identifies the canonical record or attempted record acted
// upon. Unsupported versions are allowed only on an explicit rejected/failed
// audit event whose failure code is unsupported_contract_version.
type AuditSubjectRef struct {
	Kind            ContractKind    `json:"kind"`
	ID              string          `json:"id"`
	ContractVersion ContractVersion `json:"contract_version"`
}

func (ref AuditSubjectRef) Validate() error {
	const contract = "audit_subject_ref"
	_, prefix, ok := contractIdentity(ref.Kind)
	if !ok {
		return invalid(contract, "kind", "must identify one of the eight canonical domain families")
	}
	if err := validateCanonicalID(contract, "id", ref.ID, prefix); err != nil {
		return err
	}
	if err := validateRequiredText(contract, "contract_version", string(ref.ContractVersion), maxShortText); err != nil {
		return err
	}
	return nil
}

// AuditEvent records something that happened inside Jax. It is structurally
// distinct from Event, which records a market/company/macro/geopolitical
// occurrence. AuditEvent contains immutable references, never mutable business
// objects or arbitrary payload maps.
type AuditEvent struct {
	ContractVersion  ContractVersion        `json:"contract_version"`
	ID               AuditEventID           `json:"id"`
	StreamID         AuditStreamID          `json:"stream_id"`
	Sequence         uint64                 `json:"sequence"`
	IdempotencyID    string                 `json:"idempotency_id"`
	Action           AuditAction            `json:"action"`
	Subject          AuditSubjectRef        `json:"subject"`
	Inputs           []LineageInput         `json:"inputs,omitempty"`
	InputFingerprint *ContentDigest         `json:"input_fingerprint,omitempty"`
	Output           *ImmutableContractRef  `json:"output,omitempty"`
	Supersedes       *ImmutableContractRef  `json:"supersedes,omitempty"`
	Producer         ComponentIdentity      `json:"producer"`
	Components       []ComponentIdentity    `json:"components,omitempty"`
	ProvenanceID     string                 `json:"provenance_id,omitempty"`
	Compatibility    *ContractCompatibility `json:"compatibility,omitempty"`
	Outcome          AuditOutcome           `json:"outcome"`
	Failure          *AuditFailure          `json:"failure,omitempty"`
	CorrelationID    string                 `json:"correlation_id"`
	CausationID      *AuditEventID          `json:"causation_id,omitempty"`
	KnowledgeCutoff  time.Time              `json:"knowledge_cutoff"`
	OccurredAt       time.Time              `json:"occurred_at"`
	RecordedAt       time.Time              `json:"recorded_at"`
}

func (event AuditEvent) Validate() error {
	const contract = "audit_event"
	if err := validateVersion(contract, event.ContractVersion, AuditEventContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(event.ID), "aud_"); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "stream_id", string(event.StreamID), "ast_"); err != nil {
		return err
	}
	if event.Sequence == 0 {
		return invalid(contract, "sequence", "must be greater than zero")
	}
	if err := validateCanonicalID(contract, "idempotency_id", event.IdempotencyID, "adi_"); err != nil {
		return err
	}
	switch event.Action {
	case AuditActionRecordCreated, AuditActionRecordSuperseded, AuditActionProcessingCompleted,
		AuditActionProcessingFailed, AuditActionCompatibilityAssessed, AuditActionCompatibilityApplied:
	default:
		return invalid(contract, "action", "is not supported")
	}
	if err := event.Subject.Validate(); err != nil {
		return invalid(contract, "subject", err.Error())
	}
	unsupportedSubject := !isSupportedContractVersion(event.Subject.Kind, event.Subject.ContractVersion)
	if unsupportedSubject && (event.Failure == nil || event.Failure.Code != AuditFailureUnsupportedContractVersion) {
		return invalid(contract, "subject.contract_version", "unsupported versions require an explicit unsupported_contract_version failure")
	}

	seenInputs := map[string]struct{}{}
	for i, input := range event.Inputs {
		if err := input.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("inputs[%d]", i), err.Error())
		}
		if _, exists := seenInputs[input.identityKey()]; exists {
			return invalid(contract, fmt.Sprintf("inputs[%d]", i), "duplicates an earlier immutable input")
		}
		seenInputs[input.identityKey()] = struct{}{}
		if input.Kind == LineageInputKindEvidence && input.Evidence.CollectedAt.After(event.KnowledgeCutoff) {
			return invalid(contract, fmt.Sprintf("inputs[%d].evidence.collected_at", i), "must not exceed knowledge_cutoff")
		}
		if input.Kind == LineageInputKindDataset && input.Dataset.CollectedAt.After(event.KnowledgeCutoff) {
			return invalid(contract, fmt.Sprintf("inputs[%d].dataset_snapshot.collected_at", i), "must not exceed knowledge_cutoff")
		}
	}
	if len(event.Inputs) == 0 {
		if event.InputFingerprint != nil {
			return invalid(contract, "input_fingerprint", "must be absent when inputs are unavailable")
		}
	} else {
		if event.InputFingerprint == nil {
			return invalid(contract, "input_fingerprint", "is required when immutable inputs are present")
		}
		want, err := ComputeInputFingerprint(event.Inputs)
		if err != nil {
			return invalid(contract, "input_fingerprint", err.Error())
		}
		if err := event.InputFingerprint.Validate(); err != nil {
			return invalid(contract, "input_fingerprint", err.Error())
		}
		if *event.InputFingerprint != want {
			return invalid(contract, "input_fingerprint", "does not match the ordered immutable inputs")
		}
	}
	if event.Output != nil {
		if err := event.Output.Validate(); err != nil {
			return invalid(contract, "output", err.Error())
		}
		if event.Output.Contract.Kind != event.Subject.Kind || event.Output.Contract.ID != event.Subject.ID || event.Output.Contract.ContractVersion != event.Subject.ContractVersion {
			return invalid(contract, "output.contract", "must identify the exact audit subject")
		}
	}
	if event.Supersedes != nil {
		if err := event.Supersedes.Validate(); err != nil {
			return invalid(contract, "supersedes", err.Error())
		}
		if event.Output == nil {
			return invalid(contract, "supersedes", "requires a replacement output")
		}
		if event.Supersedes.Contract.Kind != event.Output.Contract.Kind || event.Supersedes.Contract.ID != event.Output.Contract.ID {
			return invalid(contract, "supersedes.contract", "must identify the same logical record as output")
		}
		if immutableContractRefsEqual(*event.Supersedes, *event.Output) {
			return invalid(contract, "supersedes", "must identify a different immutable revision")
		}
	}
	if err := event.Producer.Validate(); err != nil {
		return invalid(contract, "producer", err.Error())
	}
	seenComponents := map[string]struct{}{event.Producer.ID: {}}
	for i, component := range event.Components {
		if err := component.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("components[%d]", i), err.Error())
		}
		if _, exists := seenComponents[component.ID]; exists {
			return invalid(contract, fmt.Sprintf("components[%d].id", i), "duplicates producer or an earlier component")
		}
		seenComponents[component.ID] = struct{}{}
	}
	if event.ProvenanceID != "" {
		if err := validateCanonicalID(contract, "provenance_id", event.ProvenanceID, "pvn_"); err != nil {
			return err
		}
	}
	if event.Compatibility != nil {
		if err := event.Compatibility.Validate(); err != nil {
			return invalid(contract, "compatibility", err.Error())
		}
	}
	if err := validateAuditOutcome(event); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "correlation_id", event.CorrelationID, "cor_"); err != nil {
		return err
	}
	if event.CausationID != nil {
		if err := validateCanonicalID(contract, "causation_id", string(*event.CausationID), "aud_"); err != nil {
			return err
		}
		if *event.CausationID == event.ID {
			return invalid(contract, "causation_id", "must not refer to the event itself")
		}
	}
	if err := validateRequiredUTC(contract, "knowledge_cutoff", event.KnowledgeCutoff); err != nil {
		return err
	}
	if err := validateRequiredUTC(contract, "occurred_at", event.OccurredAt); err != nil {
		return err
	}
	if event.KnowledgeCutoff.After(event.OccurredAt) {
		return invalid(contract, "knowledge_cutoff", "must not follow occurred_at")
	}
	if err := validateRequiredUTC(contract, "recorded_at", event.RecordedAt); err != nil {
		return err
	}
	if event.RecordedAt.Before(event.OccurredAt) {
		return invalid(contract, "recorded_at", "must not precede occurred_at")
	}
	return nil
}

func validateAuditOutcome(event AuditEvent) error {
	const contract = "audit_event"
	switch event.Outcome {
	case AuditOutcomeSucceeded:
		if event.Failure != nil {
			return invalid(contract, "failure", "must be absent for a succeeded audit event")
		}
	case AuditOutcomeFailed, AuditOutcomeRejected:
		if event.Failure == nil {
			return invalid(contract, "failure", "is required for failed or rejected audit history")
		}
	default:
		return invalid(contract, "outcome", "is not supported")
	}
	if event.Failure != nil {
		if err := validateCanonicalID(contract, "failure.id", event.Failure.ID, "fail_"); err != nil {
			return err
		}
		switch event.Failure.Code {
		case AuditFailureValidationRejected, AuditFailureUnsupportedContractVersion, AuditFailureProvenanceMismatch,
			AuditFailureMissingImmutableInput, AuditFailureComponent, AuditFailureModelProvider,
			AuditFailureCompatibilityTranslationUnavailable, AuditFailureContentDigestMismatch,
			AuditFailureReplayNonReproducible:
		default:
			return invalid(contract, "failure.code", "is not supported")
		}
		if err := validateRequiredText(contract, "failure.detail", event.Failure.Detail, maxDescription); err != nil {
			return err
		}
	}

	switch event.Action {
	case AuditActionRecordCreated, AuditActionProcessingCompleted:
		if event.Outcome != AuditOutcomeSucceeded || event.Output == nil || event.Supersedes != nil || event.ProvenanceID == "" || len(event.Inputs) == 0 {
			return invalid(contract, "action", "successful creation/completion requires inputs, provenance, output, and no supersedes/failure")
		}
	case AuditActionRecordSuperseded:
		if event.Outcome != AuditOutcomeSucceeded || event.Output == nil || event.Supersedes == nil || event.ProvenanceID == "" || len(event.Inputs) == 0 {
			return invalid(contract, "action", "successful supersession requires inputs, provenance, prior revision, and replacement output")
		}
	case AuditActionProcessingFailed:
		if event.Outcome == AuditOutcomeSucceeded || event.Output != nil || event.Supersedes != nil || event.Failure == nil {
			return invalid(contract, "action", "processing_failed requires failure history and cannot claim an output")
		}
	case AuditActionCompatibilityAssessed:
		if event.Compatibility == nil || event.Output != nil || event.Supersedes != nil {
			return invalid(contract, "action", "compatibility_assessed requires an assessment and cannot claim an output")
		}
		if event.Compatibility.Classification == CompatibilityIncompatible && event.Outcome != AuditOutcomeRejected {
			return invalid(contract, "outcome", "an incompatible assessment must be REJECTED")
		}
		if event.Compatibility.Classification != CompatibilityIncompatible && event.Outcome != AuditOutcomeSucceeded {
			return invalid(contract, "outcome", "an available compatibility assessment must be SUCCEEDED")
		}
	case AuditActionCompatibilityApplied:
		if event.Outcome != AuditOutcomeSucceeded || event.Compatibility == nil || event.Output == nil || len(event.Inputs) == 0 {
			return invalid(contract, "action", "compatibility_applied requires inputs, assessment, and translated output")
		}
		if event.Compatibility.Classification != CompatibilityLosslessTranslation && event.Compatibility.Classification != CompatibilityLossyTranslation {
			return invalid(contract, "compatibility.classification", "must be an explicit translation")
		}
	}
	return nil
}

// AuditEventRef binds one stream position to exact canonical AuditEvent bytes.
type AuditEventRef struct {
	ID              AuditEventID     `json:"id"`
	ContractVersion ContractVersion  `json:"contract_version"`
	StreamID        AuditStreamID    `json:"stream_id"`
	Sequence        uint64           `json:"sequence"`
	Revision        RevisionIdentity `json:"revision"`
	Content         ContentIdentity  `json:"content"`
}

func (ref AuditEventRef) Validate() error {
	const contract = "audit_event_ref"
	if err := validateCanonicalID(contract, "id", string(ref.ID), "aud_"); err != nil {
		return err
	}
	if err := validateVersion(contract, ref.ContractVersion, AuditEventContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "stream_id", string(ref.StreamID), "ast_"); err != nil {
		return err
	}
	if ref.Sequence == 0 {
		return invalid(contract, "sequence", "must be greater than zero")
	}
	if err := ref.Revision.Validate(); err != nil {
		return invalid(contract, "revision", err.Error())
	}
	if err := ref.Content.Validate(); err != nil {
		return invalid(contract, "content", err.Error())
	}
	if ref.Content.Representation != ContentRepresentationCanonicalJSON {
		return invalid(contract, "content.representation", "must be canonical_json")
	}
	return nil
}

func NewAuditEventRef(event AuditEvent, revision RevisionIdentity) (AuditEventRef, error) {
	if err := event.Validate(); err != nil {
		return AuditEventRef{}, err
	}
	if err := revision.Validate(); err != nil {
		return AuditEventRef{}, err
	}
	content, err := CanonicalContractContentIdentity(event)
	if err != nil {
		return AuditEventRef{}, err
	}
	return AuditEventRef{ID: event.ID, ContractVersion: event.ContractVersion, StreamID: event.StreamID, Sequence: event.Sequence, Revision: revision, Content: content}, nil
}

// AuditTrailError reports a duplicate, gap, or causation/order violation in a
// complete in-memory trail. Persistence can enforce the same per-stream rules
// transactionally without requiring a global sequence.
type AuditTrailError struct {
	Code   string
	Index  int
	Detail string
}

func (err *AuditTrailError) Error() string {
	return fmt.Sprintf("canonical audit trail %s at index %d: %s", err.Code, err.Index, err.Detail)
}

func ValidateAuditTrail(events []AuditEvent) error {
	if len(events) == 0 {
		return &AuditTrailError{Code: "empty_trail", Index: -1, Detail: "requires at least one audit event"}
	}
	seenIDs := map[AuditEventID]struct{}{}
	seenIdempotency := map[string]struct{}{}
	nextSequence := map[AuditStreamID]uint64{}
	for i, event := range events {
		if err := event.Validate(); err != nil {
			return &AuditTrailError{Code: "invalid_event", Index: i, Detail: err.Error()}
		}
		if _, exists := seenIDs[event.ID]; exists {
			return &AuditTrailError{Code: "duplicate_event_identity", Index: i, Detail: string(event.ID)}
		}
		if _, exists := seenIdempotency[event.IdempotencyID]; exists {
			return &AuditTrailError{Code: "duplicate_idempotency_identity", Index: i, Detail: event.IdempotencyID}
		}
		want := nextSequence[event.StreamID] + 1
		if event.Sequence != want {
			return &AuditTrailError{Code: "invalid_stream_sequence", Index: i, Detail: fmt.Sprintf("stream %s requires sequence %d, got %d", event.StreamID, want, event.Sequence)}
		}
		if event.CausationID != nil {
			if _, exists := seenIDs[*event.CausationID]; !exists {
				return &AuditTrailError{Code: "causation_not_earlier", Index: i, Detail: string(*event.CausationID)}
			}
		}
		seenIDs[event.ID] = struct{}{}
		seenIdempotency[event.IdempotencyID] = struct{}{}
		nextSequence[event.StreamID] = event.Sequence
	}
	return nil
}

func immutableContractRefsEqual(left, right ImmutableContractRef) bool {
	leftRaw, _ := json.Marshal(left)
	rightRaw, _ := json.Marshal(right)
	return string(leftRaw) == string(rightRaw)
}
