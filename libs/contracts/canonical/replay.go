package canonical

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ReplayClaim string

const (
	ReplayClaimExact                      ReplayClaim = "EXACT"
	ReplayClaimCompatibleMigrated         ReplayClaim = "COMPATIBLE_MIGRATED"
	ReplayClaimUnavailableNonReproducible ReplayClaim = "UNAVAILABLE_NON_REPRODUCIBLE"
)

type ReplayStrategy string

const (
	ReplayStrategyDeterministicReexecution ReplayStrategy = "deterministic_reexecution"
	ReplayStrategyImmutableReconstruction  ReplayStrategy = "immutable_history_reconstruction"
	ReplayStrategyCompatibilityMigration   ReplayStrategy = "compatibility_migration"
	ReplayStrategyUnavailable              ReplayStrategy = "unavailable"
)

// ReplayManifest is the minimum portable identity for reconstructing or
// re-running one historical canonical output. It contains no machine path and
// grants no trading authority.
type ReplayManifest struct {
	ContractVersion      ContractVersion         `json:"contract_version"`
	ID                   ReplayManifestID        `json:"id"`
	Target               ImmutableContractRef    `json:"target"`
	Inputs               []LineageInput          `json:"inputs,omitempty"`
	InputFingerprint     *ContentDigest          `json:"input_fingerprint,omitempty"`
	Producer             ComponentIdentity       `json:"producer"`
	Components           []ComponentIdentity     `json:"components,omitempty"`
	RequiredComponentIDs []string                `json:"required_component_ids,omitempty"`
	AuditEvents          []AuditEventRef         `json:"audit_events"`
	StoredResponses      []ImmutableContractRef  `json:"stored_responses,omitempty"`
	Compatibility        []ContractCompatibility `json:"compatibility,omitempty"`
	Claim                ReplayClaim             `json:"claim"`
	Strategy             ReplayStrategy          `json:"strategy"`
	Unavailable          *AuditFailure           `json:"unavailable,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
}

func (manifest ReplayManifest) Validate() error {
	const contract = "replay_manifest"
	if err := validateVersion(contract, manifest.ContractVersion, ReplayManifestContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(manifest.ID), "rpl_"); err != nil {
		return err
	}
	if err := manifest.Target.Validate(); err != nil {
		return invalid(contract, "target", err.Error())
	}
	if manifest.Target.Contract.Kind == ContractKindAuditEvent || manifest.Target.Contract.Kind == ContractKindReplayManifest {
		return invalid(contract, "target.contract.kind", "must identify a canonical domain output")
	}
	if err := validateReplayInputs(manifest); err != nil {
		return err
	}
	if err := manifest.Producer.Validate(); err != nil {
		return invalid(contract, "producer", err.Error())
	}
	components := map[string]ComponentIdentity{manifest.Producer.ID: manifest.Producer}
	for i, component := range manifest.Components {
		if err := component.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("components[%d]", i), err.Error())
		}
		if _, exists := components[component.ID]; exists {
			return invalid(contract, fmt.Sprintf("components[%d].id", i), "duplicates producer or an earlier component")
		}
		components[component.ID] = component
	}
	seenRequired := map[string]struct{}{}
	for i, id := range manifest.RequiredComponentIDs {
		if err := validateCanonicalID(contract, fmt.Sprintf("required_component_ids[%d]", i), id, "cmp_"); err != nil {
			return err
		}
		if _, exists := seenRequired[id]; exists {
			return invalid(contract, fmt.Sprintf("required_component_ids[%d]", i), "duplicates an earlier required component")
		}
		if _, exists := components[id]; !exists {
			return invalid(contract, fmt.Sprintf("required_component_ids[%d]", i), "does not identify producer or a recorded component")
		}
		seenRequired[id] = struct{}{}
	}
	if manifest.Claim != ReplayClaimUnavailableNonReproducible {
		if len(manifest.RequiredComponentIDs) == 0 {
			return invalid(contract, "required_component_ids", "requires at least one replay component")
		}
		if _, required := seenRequired[manifest.Producer.ID]; !required {
			return invalid(contract, "required_component_ids", "must include the target producer")
		}
	}
	if len(manifest.AuditEvents) == 0 {
		return invalid(contract, "audit_events", "requires at least one immutable audit event")
	}
	seenAudit := map[AuditEventID]struct{}{}
	lastSequence := map[AuditStreamID]uint64{}
	for i, ref := range manifest.AuditEvents {
		if err := ref.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("audit_events[%d]", i), err.Error())
		}
		if _, exists := seenAudit[ref.ID]; exists {
			return invalid(contract, fmt.Sprintf("audit_events[%d].id", i), "duplicates an earlier audit event")
		}
		if previous := lastSequence[ref.StreamID]; previous != 0 && ref.Sequence <= previous {
			return invalid(contract, fmt.Sprintf("audit_events[%d].sequence", i), "must increase within its stream in manifest path order")
		}
		seenAudit[ref.ID] = struct{}{}
		lastSequence[ref.StreamID] = ref.Sequence
	}
	if err := validateStoredResponses(manifest); err != nil {
		return err
	}
	seenCompatibility := map[string]struct{}{}
	for i, assessment := range manifest.Compatibility {
		if err := assessment.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("compatibility[%d]", i), err.Error())
		}
		if _, exists := seenCompatibility[assessment.ID]; exists {
			return invalid(contract, fmt.Sprintf("compatibility[%d].id", i), "duplicates an earlier compatibility assessment")
		}
		seenCompatibility[assessment.ID] = struct{}{}
		if assessment.Translator != nil {
			if _, required := seenRequired[assessment.Translator.ID]; !required {
				return invalid(contract, fmt.Sprintf("compatibility[%d].translator.id", i), "must identify a required replay component")
			}
		}
	}
	if err := validateReplayClaim(manifest, components, seenRequired); err != nil {
		return err
	}
	return validateRequiredUTC(contract, "created_at", manifest.CreatedAt)
}

func validateReplayInputs(manifest ReplayManifest) error {
	const contract = "replay_manifest"
	seen := map[string]struct{}{}
	for i, input := range manifest.Inputs {
		if err := input.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("inputs[%d]", i), err.Error())
		}
		if _, exists := seen[input.identityKey()]; exists {
			return invalid(contract, fmt.Sprintf("inputs[%d]", i), "duplicates an earlier immutable input")
		}
		seen[input.identityKey()] = struct{}{}
	}
	if len(manifest.Inputs) == 0 {
		if manifest.InputFingerprint != nil {
			return invalid(contract, "input_fingerprint", "must be absent when inputs are unavailable")
		}
		return nil
	}
	if manifest.InputFingerprint == nil {
		return invalid(contract, "input_fingerprint", "is required when immutable inputs are present")
	}
	want, err := ComputeInputFingerprint(manifest.Inputs)
	if err != nil {
		return invalid(contract, "input_fingerprint", err.Error())
	}
	if err := manifest.InputFingerprint.Validate(); err != nil {
		return invalid(contract, "input_fingerprint", err.Error())
	}
	if *manifest.InputFingerprint != want {
		return invalid(contract, "input_fingerprint", "does not match the ordered immutable inputs")
	}
	return nil
}

func validateStoredResponses(manifest ReplayManifest) error {
	const contract = "replay_manifest"
	seen := map[string]struct{}{}
	for i, response := range manifest.StoredResponses {
		if err := response.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("stored_responses[%d]", i), err.Error())
		}
		if response.Contract.Kind != ContractKindEvidence || response.Contract.ContractVersion != EvidenceContractV2 {
			return invalid(contract, fmt.Sprintf("stored_responses[%d].contract", i), "must identify immutable model-output Evidence V2")
		}
		key := immutableContractRefKey(response)
		if _, exists := seen[key]; exists {
			return invalid(contract, fmt.Sprintf("stored_responses[%d]", i), "duplicates an earlier historical response")
		}
		seen[key] = struct{}{}
		covered := false
		for _, input := range manifest.Inputs {
			if input.Kind == LineageInputKindContract && immutableContractRefsEqual(*input.Contract, response) {
				covered = true
				break
			}
		}
		if !covered {
			return invalid(contract, fmt.Sprintf("stored_responses[%d]", i), "must also appear in the immutable input set")
		}
	}
	return nil
}

func validateReplayClaim(manifest ReplayManifest, components map[string]ComponentIdentity, required map[string]struct{}) error {
	const contract = "replay_manifest"
	hasModel := false
	requiredModel := false
	for id, component := range components {
		if component.Kind == ComponentKindModel {
			hasModel = true
			if _, ok := required[id]; ok {
				requiredModel = true
			}
		}
	}
	switch manifest.Claim {
	case ReplayClaimExact:
		if manifest.Strategy != ReplayStrategyDeterministicReexecution && manifest.Strategy != ReplayStrategyImmutableReconstruction {
			return invalid(contract, "strategy", "EXACT requires deterministic reexecution or immutable history reconstruction")
		}
		if manifest.Unavailable != nil || len(manifest.Inputs) == 0 {
			return invalid(contract, "claim", "EXACT requires all inputs and cannot carry an unavailable failure")
		}
		if len(manifest.Compatibility) != 0 {
			return invalid(contract, "compatibility", "EXACT cannot use compatibility translation")
		}
		if requiredModel {
			return invalid(contract, "required_component_ids", "EXACT cannot require stochastic model re-inference")
		}
		if hasModel && manifest.Strategy != ReplayStrategyImmutableReconstruction {
			return invalid(contract, "strategy", "historical model use requires immutable history reconstruction")
		}
		if hasModel && len(manifest.StoredResponses) == 0 {
			return invalid(contract, "stored_responses", "historical model use requires a stored immutable response")
		}
	case ReplayClaimCompatibleMigrated:
		if manifest.Strategy != ReplayStrategyCompatibilityMigration || manifest.Unavailable != nil || len(manifest.Inputs) == 0 || len(manifest.Compatibility) == 0 {
			return invalid(contract, "claim", "COMPATIBLE_MIGRATED requires inputs and explicit compatibility migration")
		}
		for i, assessment := range manifest.Compatibility {
			if assessment.Classification == CompatibilityExact || assessment.Classification == CompatibilityIncompatible {
				return invalid(contract, fmt.Sprintf("compatibility[%d].classification", i), "must be a declared lossless or lossy translation")
			}
		}
	case ReplayClaimUnavailableNonReproducible:
		if manifest.Strategy != ReplayStrategyUnavailable || manifest.Unavailable == nil {
			return invalid(contract, "claim", "UNAVAILABLE_NON_REPRODUCIBLE requires an explicit unavailable failure")
		}
		if err := validateReplayUnavailable(*manifest.Unavailable); err != nil {
			return err
		}
	default:
		return invalid(contract, "claim", "is not supported")
	}
	return nil
}

func validateReplayUnavailable(failure AuditFailure) error {
	if err := validateCanonicalID("replay_manifest", "unavailable.id", failure.ID, "fail_"); err != nil {
		return err
	}
	switch failure.Code {
	case AuditFailureUnsupportedContractVersion, AuditFailureMissingImmutableInput, AuditFailureComponent,
		AuditFailureModelProvider, AuditFailureCompatibilityTranslationUnavailable,
		AuditFailureContentDigestMismatch, AuditFailureReplayNonReproducible, AuditFailureProvenanceMismatch:
	default:
		return invalid("replay_manifest", "unavailable.code", "does not describe replay unavailability")
	}
	return validateRequiredText("replay_manifest", "unavailable.detail", failure.Detail, maxDescription)
}

// Replay materials are supplied by an in-memory fixture or a future storage
// adapter. They are verification inputs, not canonical serialized contracts.
type ContractMaterial struct {
	Reference ImmutableContractRef
	Value     Contract
}

type EvidenceMaterial struct {
	Reference EvidenceRef
	Bytes     []byte
}

type DatasetMaterial struct {
	Reference DatasetSnapshotRef
	Bytes     []byte
}

type ComponentMaterial struct {
	Identity ComponentIdentity
	Bytes    []byte
}

type ReplayMaterials struct {
	Contracts   []ContractMaterial
	Evidence    []EvidenceMaterial
	Datasets    []DatasetMaterial
	Components  []ComponentMaterial
	AuditEvents []AuditEvent
}

type ReplayVerification struct {
	Claim            ReplayClaim
	Target           ImmutableContractRef
	InputFingerprint ContentDigest
	AuditEventCount  int
}

type ReplayVerificationError struct {
	Code   string
	Field  string
	Detail string
	Cause  error
}

func (err *ReplayVerificationError) Error() string {
	return fmt.Sprintf("canonical replay verification %s: %s %s", err.Code, err.Field, err.Detail)
}

func (err *ReplayVerificationError) Unwrap() error { return err.Cause }

// VerifyReplayManifest verifies immutable inputs, the historical output,
// required components, and the complete referenced audit trail. It performs no
// database/provider call and never invokes a model.
func VerifyReplayManifest(manifest ReplayManifest, materials ReplayMaterials) (ReplayVerification, error) {
	if err := manifest.Validate(); err != nil {
		return ReplayVerification{}, replayError("invalid_manifest", "manifest", err.Error(), err)
	}
	if manifest.Claim == ReplayClaimUnavailableNonReproducible {
		return ReplayVerification{}, replayError("replay_unavailable", "manifest.claim", manifest.Unavailable.Detail, nil)
	}
	if err := verifyContractMaterial(manifest.Target, materials.Contracts); err != nil {
		return ReplayVerification{}, err
	}
	for i, response := range manifest.StoredResponses {
		if err := verifyContractMaterial(response, materials.Contracts); err != nil {
			return ReplayVerification{}, replayError("missing_historical_ai_response", fmt.Sprintf("stored_responses[%d]", i), err.Error(), err)
		}
	}
	for i, input := range manifest.Inputs {
		switch input.Kind {
		case LineageInputKindContract:
			if err := verifyContractMaterial(*input.Contract, materials.Contracts); err != nil {
				return ReplayVerification{}, annotateReplayError(err, fmt.Sprintf("inputs[%d]", i))
			}
		case LineageInputKindEvidence:
			if err := verifyEvidenceMaterial(*input.Evidence, materials.Evidence); err != nil {
				return ReplayVerification{}, annotateReplayError(err, fmt.Sprintf("inputs[%d]", i))
			}
		case LineageInputKindDataset:
			if err := verifyDatasetMaterial(*input.Dataset, materials.Datasets); err != nil {
				return ReplayVerification{}, annotateReplayError(err, fmt.Sprintf("inputs[%d]", i))
			}
		}
	}
	componentIndex := map[string]ComponentIdentity{manifest.Producer.ID: manifest.Producer}
	for _, component := range manifest.Components {
		componentIndex[component.ID] = component
	}
	for i, id := range manifest.RequiredComponentIDs {
		if err := verifyComponentMaterial(componentIndex[id], materials.Components); err != nil {
			return ReplayVerification{}, annotateReplayError(err, fmt.Sprintf("required_component_ids[%d]", i))
		}
	}
	if err := verifyAuditMaterials(manifest.AuditEvents, materials.AuditEvents); err != nil {
		return ReplayVerification{}, err
	}
	return ReplayVerification{Claim: manifest.Claim, Target: manifest.Target, InputFingerprint: *manifest.InputFingerprint, AuditEventCount: len(manifest.AuditEvents)}, nil
}

func verifyContractMaterial(want ImmutableContractRef, materials []ContractMaterial) error {
	for _, material := range materials {
		if !immutableContractRefsEqual(material.Reference, want) {
			continue
		}
		if material.Value == nil {
			return replayError("missing_immutable_input", "contract", "matching reference has no contract value", nil)
		}
		kind, version, err := canonicalContractMetadata(material.Value)
		if err != nil {
			return replayError("invalid_contract_material", "contract", err.Error(), err)
		}
		if kind != want.Contract.Kind || version != want.Contract.ContractVersion {
			return replayError("contract_identity_mismatch", "contract", "material kind/version differs from immutable reference", nil)
		}
		if err := want.Content.VerifyCanonicalContract(material.Value); err != nil {
			return replayError("content_digest_mismatch", "contract.content", err.Error(), err)
		}
		return nil
	}
	return replayError("missing_immutable_input", "contract", immutableContractRefKey(want), nil)
}

func verifyEvidenceMaterial(want EvidenceRef, materials []EvidenceMaterial) error {
	for _, material := range materials {
		if !jsonValuesEqual(material.Reference, want) {
			continue
		}
		if err := want.Content.Digest.VerifyBytes(material.Bytes); err != nil {
			return replayError("content_digest_mismatch", "evidence.content", err.Error(), err)
		}
		return nil
	}
	return replayError("missing_immutable_input", "evidence", want.Evidence.ID, nil)
}

func verifyDatasetMaterial(want DatasetSnapshotRef, materials []DatasetMaterial) error {
	for _, material := range materials {
		if !jsonValuesEqual(material.Reference, want) {
			continue
		}
		if err := want.Content.Digest.VerifyBytes(material.Bytes); err != nil {
			return replayError("content_digest_mismatch", "dataset_snapshot.content", err.Error(), err)
		}
		return nil
	}
	return replayError("missing_immutable_input", "dataset_snapshot", string(want.SnapshotID), nil)
}

func verifyComponentMaterial(want ComponentIdentity, materials []ComponentMaterial) error {
	for _, material := range materials {
		if !jsonValuesEqual(material.Identity, want) {
			continue
		}
		if want.Content != nil {
			if len(material.Bytes) == 0 {
				return replayError("missing_component_content", "component.content", want.ID, nil)
			}
			if err := want.Content.Digest.VerifyBytes(material.Bytes); err != nil {
				return replayError("component_content_mismatch", "component.content", err.Error(), err)
			}
		}
		return nil
	}
	return replayError("component_identity_mismatch", "component", want.ID, nil)
}

func verifyAuditMaterials(refs []AuditEventRef, events []AuditEvent) error {
	path := make([]AuditEvent, 0, len(refs))
	for i, ref := range refs {
		found := false
		for _, event := range events {
			if event.ID != ref.ID {
				continue
			}
			actual, err := NewAuditEventRef(event, ref.Revision)
			if err != nil {
				return replayError("invalid_audit_event", fmt.Sprintf("audit_events[%d]", i), err.Error(), err)
			}
			if !jsonValuesEqual(actual, ref) {
				return replayError("audit_event_mismatch", fmt.Sprintf("audit_events[%d]", i), string(ref.ID), nil)
			}
			path = append(path, event)
			found = true
			break
		}
		if !found {
			return replayError("missing_audit_event", fmt.Sprintf("audit_events[%d]", i), string(ref.ID), nil)
		}
	}
	if err := ValidateAuditTrail(path); err != nil {
		return replayError("invalid_audit_trail", "audit_events", err.Error(), err)
	}
	return nil
}

func annotateReplayError(err error, field string) error {
	var replayErr *ReplayVerificationError
	if errors.As(err, &replayErr) {
		copy := *replayErr
		copy.Field = field + "." + copy.Field
		return &copy
	}
	return replayError("verification_failed", field, err.Error(), err)
}

func replayError(code, field, detail string, cause error) error {
	return &ReplayVerificationError{Code: code, Field: field, Detail: detail, Cause: cause}
}

func immutableContractRefKey(ref ImmutableContractRef) string {
	return string(ref.Contract.Kind) + "/" + ref.Contract.ID + "/" + string(ref.Contract.ContractVersion) + "/" + ref.Revision.Namespace + "/" + ref.Revision.Value + "/" + ref.Content.Digest.Value
}

func jsonValuesEqual(left, right any) bool {
	leftRaw, _ := json.Marshal(left)
	rightRaw, _ := json.Marshal(right)
	return string(leftRaw) == string(rightRaw)
}
