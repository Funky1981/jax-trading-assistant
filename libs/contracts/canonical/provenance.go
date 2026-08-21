package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	EvidenceRefContractV1     ContractVersion = "jax.evidence_ref/v1"
	ProvenanceContractV1      ContractVersion = "jax.provenance/v1"
	DatasetSnapshotContractV1 ContractVersion = "jax.dataset_snapshot_ref/v1"
)

// DigestAlgorithm names the cryptographic algorithm used by a content
// identity. The algorithm is part of the identity and unsupported algorithms
// fail closed.
type DigestAlgorithm string

const DigestAlgorithmSHA256 DigestAlgorithm = "sha256"

// ContentDigest is a lower-case hexadecimal cryptographic digest.
type ContentDigest struct {
	Algorithm DigestAlgorithm `json:"algorithm"`
	Value     string          `json:"value"`
}

func (digest ContentDigest) Validate() error {
	const contract = "content_digest"
	if digest.Algorithm != DigestAlgorithmSHA256 {
		return invalid(contract, "algorithm", "is not supported")
	}
	if len(digest.Value) != sha256.Size*2 {
		return invalid(contract, "value", fmt.Sprintf("must contain %d hexadecimal characters for sha256", sha256.Size*2))
	}
	if strings.ToLower(digest.Value) != digest.Value {
		return invalid(contract, "value", "must use lower-case hexadecimal encoding")
	}
	decoded, err := hex.DecodeString(digest.Value)
	if err != nil || len(decoded) != sha256.Size {
		return invalid(contract, "value", "must be valid sha256 hexadecimal encoding")
	}
	return nil
}

// DigestBytes identifies the exact supplied byte sequence. No decoding,
// newline normalization, JSON parsing, or other transformation is performed.
func DigestBytes(data []byte) ContentDigest {
	sum := sha256.Sum256(data)
	return ContentDigest{Algorithm: DigestAlgorithmSHA256, Value: hex.EncodeToString(sum[:])}
}

// DigestMismatchError reports an immutable-content verification failure.
type DigestMismatchError struct {
	Expected ContentDigest
	Actual   ContentDigest
}

func (err *DigestMismatchError) Error() string {
	return fmt.Sprintf("canonical content digest mismatch: expected %s:%s, got %s:%s", err.Expected.Algorithm, err.Expected.Value, err.Actual.Algorithm, err.Actual.Value)
}

// VerifyBytes verifies exact bytes against the digest and fails closed for an
// invalid or unsupported digest identity.
func (digest ContentDigest) VerifyBytes(data []byte) error {
	if err := digest.Validate(); err != nil {
		return err
	}
	actual := DigestBytes(data)
	if digest != actual {
		return &DigestMismatchError{Expected: digest, Actual: actual}
	}
	return nil
}

type ContentRepresentation string

const (
	ContentRepresentationRawBytes      ContentRepresentation = "raw_bytes"
	ContentRepresentationCanonicalJSON ContentRepresentation = "canonical_json"
	CanonicalJSONIdentityV1                                  = "jax.canonical-contract-json/v1"
)

// ContentIdentity states both the exact digest and what representation was
// hashed. Raw bytes are hashed byte-for-byte. Canonical JSON is produced only
// by CanonicalContractBytes using the named canonicalization identity.
type ContentIdentity struct {
	Representation   ContentRepresentation `json:"representation"`
	Digest           ContentDigest         `json:"digest"`
	Canonicalization string                `json:"canonicalization,omitempty"`
}

func (identity ContentIdentity) Validate() error {
	const contract = "content_identity"
	switch identity.Representation {
	case ContentRepresentationRawBytes:
		if identity.Canonicalization != "" {
			return invalid(contract, "canonicalization", "must be empty for raw bytes")
		}
	case ContentRepresentationCanonicalJSON:
		if identity.Canonicalization != CanonicalJSONIdentityV1 {
			return invalid(contract, "canonicalization", fmt.Sprintf("must be %q", CanonicalJSONIdentityV1))
		}
	default:
		return invalid(contract, "representation", "is not supported")
	}
	if err := identity.Digest.Validate(); err != nil {
		return invalid(contract, "digest", err.Error())
	}
	return nil
}

func RawContentIdentity(data []byte) ContentIdentity {
	return ContentIdentity{Representation: ContentRepresentationRawBytes, Digest: DigestBytes(data)}
}

// VersionIdentity identifies a version in an explicit namespace such as
// semver, git.commit, provider.model_id, jax.prompt, or jax.policy.
type VersionIdentity struct {
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

func (version VersionIdentity) Validate() error {
	return validateVersionIdentity("version_identity", "version", version)
}

func validateVersionIdentity(contract, field string, version VersionIdentity) error {
	if err := validateCode(contract, field+".namespace", version.Namespace); err != nil {
		return err
	}
	return validateRequiredText(contract, field+".value", version.Value, 512)
}

// SourceIdentity identifies the logical information source. Provider and model
// are deliberately excluded because they are transport/producer identities.
type SourceIdentity struct {
	ID   string     `json:"id"`
	Kind SourceKind `json:"kind"`
}

func (source SourceIdentity) Validate() error {
	return validateSourceIdentity("source_identity", "source", source)
}

func validateSourceIdentity(contract, field string, source SourceIdentity) error {
	if err := validateCanonicalID(contract, field+".id", source.ID, "src_"); err != nil {
		return err
	}
	switch source.Kind {
	case SourceKindPublisher, SourceKindRegulator, SourceKindExchange, SourceKindIssuer, SourceKindDataset, SourceKindInternal:
		return nil
	case SourceKindProvider, SourceKindModel:
		return invalid(contract, field+".kind", "must identify a logical source, not a provider or producing model")
	default:
		return invalid(contract, field+".kind", "is not supported")
	}
}

// ProviderIdentity identifies the service/transport that supplied content. Its
// namespace is distinct from any provider-assigned external evidence ID.
type ProviderIdentity struct {
	ID         string      `json:"id"`
	Namespace  string      `json:"namespace"`
	ExternalID *ExternalID `json:"external_id,omitempty"`
}

func (provider ProviderIdentity) Validate() error {
	return validateProviderIdentity("provider_identity", "provider", provider)
}

func validateProviderIdentity(contract, field string, provider ProviderIdentity) error {
	if err := validateCanonicalID(contract, field+".id", provider.ID, "pvd_"); err != nil {
		return err
	}
	if err := validateCode(contract, field+".namespace", provider.Namespace); err != nil {
		return err
	}
	if provider.ExternalID != nil {
		return validateExternalID(contract, field+".external_id", *provider.ExternalID)
	}
	return nil
}

// RevisionIdentity distinguishes one immutable source/provider revision or
// snapshot from another logical revision.
type RevisionIdentity struct {
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

func (revision RevisionIdentity) Validate() error {
	return validateRevisionIdentity("revision_identity", "revision", revision)
}

func validateRevisionIdentity(contract, field string, revision RevisionIdentity) error {
	if err := validateCode(contract, field+".namespace", revision.Namespace); err != nil {
		return err
	}
	return validateRequiredText(contract, field+".value", revision.Value, 512)
}

// EvidenceRef is an immutable evidence identity. It identifies the canonical
// Evidence record, exact content bytes, logical source, optional supplying
// provider, source revision, and acquisition clock without copying Evidence.
type EvidenceRef struct {
	ContractVersion ContractVersion   `json:"contract_version"`
	Evidence        ContractRef       `json:"evidence"`
	Content         ContentIdentity   `json:"content"`
	Source          SourceIdentity    `json:"source"`
	Provider        *ProviderIdentity `json:"provider,omitempty"`
	Revision        RevisionIdentity  `json:"revision"`
	ObservedAt      *time.Time        `json:"observed_at,omitempty"`
	PublishedAt     *time.Time        `json:"published_at,omitempty"`
	CollectedAt     time.Time         `json:"collected_at"`
}

func (ref EvidenceRef) Validate() error {
	const contract = "evidence_ref"
	if err := validateVersion(contract, ref.ContractVersion, EvidenceRefContractV1); err != nil {
		return err
	}
	if err := validateSupportedContractRef(contract, "evidence", ref.Evidence); err != nil {
		return err
	}
	if ref.Evidence.Kind != ContractKindEvidence {
		return invalid(contract, "evidence.kind", "must identify canonical evidence")
	}
	if err := ref.Content.Validate(); err != nil {
		return invalid(contract, "content", err.Error())
	}
	if ref.Content.Representation != ContentRepresentationRawBytes {
		return invalid(contract, "content.representation", "must be raw_bytes for forensic evidence content")
	}
	if err := validateSourceIdentity(contract, "source", ref.Source); err != nil {
		return err
	}
	if ref.Provider != nil {
		if err := validateProviderIdentity(contract, "provider", *ref.Provider); err != nil {
			return err
		}
	}
	if err := validateRevisionIdentity(contract, "revision", ref.Revision); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "observed_at", ref.ObservedAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "published_at", ref.PublishedAt); err != nil {
		return err
	}
	if err := validateRequiredUTC(contract, "collected_at", ref.CollectedAt); err != nil {
		return err
	}
	if ref.ObservedAt != nil && ref.CollectedAt.Before(*ref.ObservedAt) {
		return invalid(contract, "collected_at", "must not precede observed_at")
	}
	if ref.PublishedAt != nil && ref.CollectedAt.Before(*ref.PublishedAt) {
		return invalid(contract, "collected_at", "must not precede published_at")
	}
	return nil
}

// ImmutableContractRef binds a canonical record identity and schema to one
// exact record revision and canonical content representation.
type ImmutableContractRef struct {
	Contract ContractRef      `json:"contract"`
	Revision RevisionIdentity `json:"revision"`
	Content  ContentIdentity  `json:"content"`
}

func (ref ImmutableContractRef) Validate() error {
	const contract = "immutable_contract_ref"
	if err := validateSupportedContractRef(contract, "contract", ref.Contract); err != nil {
		return err
	}
	if err := validateRevisionIdentity(contract, "revision", ref.Revision); err != nil {
		return err
	}
	if err := ref.Content.Validate(); err != nil {
		return invalid(contract, "content", err.Error())
	}
	if ref.Content.Representation != ContentRepresentationCanonicalJSON {
		return invalid(contract, "content.representation", "must be canonical_json for a canonical contract revision")
	}
	return nil
}

type DatasetID string
type DatasetSnapshotID string

// DatasetIdentity names the logical dataset independently from a mutable path
// or one snapshot. ExternalID is where an existing registry/provider ID maps.
type DatasetIdentity struct {
	ID            DatasetID       `json:"id"`
	ExternalID    ExternalID      `json:"external_id"`
	SchemaVersion VersionIdentity `json:"schema_version"`
}

// DatasetSnapshotRef binds a logical dataset to an exact immutable snapshot.
type DatasetSnapshotRef struct {
	ContractVersion ContractVersion   `json:"contract_version"`
	Dataset         DatasetIdentity   `json:"dataset"`
	SnapshotID      DatasetSnapshotID `json:"snapshot_id"`
	Revision        RevisionIdentity  `json:"revision"`
	Content         ContentIdentity   `json:"content"`
	AsOf            *time.Time        `json:"as_of,omitempty"`
	CollectedAt     time.Time         `json:"collected_at"`
}

func (ref DatasetSnapshotRef) Validate() error {
	const contract = "dataset_snapshot_ref"
	if err := validateVersion(contract, ref.ContractVersion, DatasetSnapshotContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "dataset.id", string(ref.Dataset.ID), "dset_"); err != nil {
		return err
	}
	if err := validateExternalID(contract, "dataset.external_id", ref.Dataset.ExternalID); err != nil {
		return err
	}
	if err := validateVersionIdentity(contract, "dataset.schema_version", ref.Dataset.SchemaVersion); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "snapshot_id", string(ref.SnapshotID), "dss_"); err != nil {
		return err
	}
	if err := validateRevisionIdentity(contract, "revision", ref.Revision); err != nil {
		return err
	}
	if err := ref.Content.Validate(); err != nil {
		return invalid(contract, "content", err.Error())
	}
	if err := validateOptionalUTC(contract, "as_of", ref.AsOf); err != nil {
		return err
	}
	if err := validateRequiredUTC(contract, "collected_at", ref.CollectedAt); err != nil {
		return err
	}
	if ref.AsOf != nil && ref.CollectedAt.Before(*ref.AsOf) {
		return invalid(contract, "collected_at", "must not precede as_of")
	}
	return nil
}

const (
	ComponentKindPrompt        ComponentKind = "prompt"
	ComponentKindMapping       ComponentKind = "mapping"
	ComponentKindNormalizer    ComponentKind = "normalizer"
	ComponentKindResolver      ComponentKind = "resolver"
	ComponentKindValidator     ComponentKind = "validator"
	ComponentKindSoftwareBuild ComponentKind = "software_build"
	ComponentKindConfiguration ComponentKind = "configuration"
)

// ComponentIdentity identifies a producing or supporting component. Version
// is namespaced; Content may bind exact prompt, policy, tool, config, source, or
// build bytes. Models additionally require their supplying provider identity.
type ComponentIdentity struct {
	ID       string            `json:"id"`
	Kind     ComponentKind     `json:"kind"`
	Name     string            `json:"name"`
	Version  VersionIdentity   `json:"version"`
	Provider *ProviderIdentity `json:"provider,omitempty"`
	Content  *ContentIdentity  `json:"content,omitempty"`
}

func (component ComponentIdentity) Validate() error {
	const contract = "component_identity"
	if err := validateCanonicalID(contract, "id", component.ID, "cmp_"); err != nil {
		return err
	}
	switch component.Kind {
	case ComponentKindMethod, ComponentKindAlgorithm, ComponentKindModel, ComponentKindTool, ComponentKindPolicy,
		ComponentKindPrompt, ComponentKindMapping, ComponentKindNormalizer, ComponentKindResolver,
		ComponentKindValidator, ComponentKindSoftwareBuild, ComponentKindConfiguration:
	default:
		return invalid(contract, "kind", "is not supported")
	}
	if err := validateRequiredText(contract, "name", component.Name, maxShortText); err != nil {
		return err
	}
	if err := validateVersionIdentity(contract, "version", component.Version); err != nil {
		return err
	}
	if component.Provider != nil {
		if err := validateProviderIdentity(contract, "provider", *component.Provider); err != nil {
			return err
		}
	}
	if component.Kind == ComponentKindModel && component.Provider == nil {
		return invalid(contract, "provider", "is required for a model component")
	}
	if component.Content != nil {
		if err := component.Content.Validate(); err != nil {
			return invalid(contract, "content", err.Error())
		}
	}
	if component.Kind == ComponentKindPrompt && component.Content == nil {
		return invalid(contract, "content", "is required for a prompt component")
	}
	return nil
}

type LineageInputKind string

const (
	LineageInputKindContract LineageInputKind = "contract"
	LineageInputKindEvidence LineageInputKind = "evidence"
	LineageInputKindDataset  LineageInputKind = "dataset_snapshot"
)

// LineageInput is a closed union of immutable canonical, evidence, or dataset
// snapshot inputs. It never embeds the complete input object.
type LineageInput struct {
	Kind     LineageInputKind      `json:"kind"`
	Contract *ImmutableContractRef `json:"contract,omitempty"`
	Evidence *EvidenceRef          `json:"evidence,omitempty"`
	Dataset  *DatasetSnapshotRef   `json:"dataset_snapshot,omitempty"`
}

func (input LineageInput) Validate() error {
	const contract = "lineage_input"
	set := 0
	if input.Contract != nil {
		set++
	}
	if input.Evidence != nil {
		set++
	}
	if input.Dataset != nil {
		set++
	}
	if set != 1 {
		return invalid(contract, "kind", "must set exactly one immutable input reference")
	}
	switch input.Kind {
	case LineageInputKindContract:
		if input.Contract == nil {
			return invalid(contract, "contract", "is required for contract input")
		}
		return input.Contract.Validate()
	case LineageInputKindEvidence:
		if input.Evidence == nil {
			return invalid(contract, "evidence", "is required for evidence input")
		}
		return input.Evidence.Validate()
	case LineageInputKindDataset:
		if input.Dataset == nil {
			return invalid(contract, "dataset_snapshot", "is required for dataset input")
		}
		return input.Dataset.Validate()
	default:
		return invalid(contract, "kind", "is not supported")
	}
}

func (input LineageInput) identityKey() string {
	switch input.Kind {
	case LineageInputKindContract:
		ref := input.Contract
		return string(input.Kind) + "\x00" + string(ref.Contract.Kind) + "\x00" + ref.Contract.ID + "\x00" + string(ref.Contract.ContractVersion) + "\x00" + ref.Revision.Namespace + "\x00" + ref.Revision.Value + "\x00" + ref.Content.Digest.Value
	case LineageInputKindEvidence:
		ref := input.Evidence
		return string(input.Kind) + "\x00" + ref.Evidence.ID + "\x00" + string(ref.Evidence.ContractVersion) + "\x00" + ref.Revision.Namespace + "\x00" + ref.Revision.Value + "\x00" + ref.Content.Digest.Value
	case LineageInputKindDataset:
		ref := input.Dataset
		return string(input.Kind) + "\x00" + string(ref.Dataset.ID) + "\x00" + string(ref.SnapshotID) + "\x00" + ref.Revision.Namespace + "\x00" + ref.Revision.Value + "\x00" + ref.Content.Digest.Value
	default:
		return string(input.Kind)
	}
}

// Provenance binds exact inputs to the primary producing component and all
// materially relevant supporting component identities. InputFingerprint is a
// verified digest of the ordered input-reference envelope, not runtime memory.
type Provenance struct {
	ContractVersion  ContractVersion     `json:"contract_version"`
	ID               string              `json:"id"`
	Inputs           []LineageInput      `json:"inputs"`
	InputFingerprint ContentDigest       `json:"input_fingerprint"`
	Producer         ComponentIdentity   `json:"producer"`
	Components       []ComponentIdentity `json:"components,omitempty"`
}

func (provenance Provenance) Validate() error {
	return validateProvenance("provenance", "", provenance, nil)
}

func validateProvenance(contract, field string, provenance Provenance, self *ContractRef) error {
	prefix := field
	if prefix != "" {
		prefix += "."
	}
	if provenance.ContractVersion != ProvenanceContractV1 {
		return invalid(contract, prefix+"contract_version", fmt.Sprintf("must be %q", ProvenanceContractV1))
	}
	if err := validateCanonicalID(contract, prefix+"id", provenance.ID, "pvn_"); err != nil {
		return err
	}
	if len(provenance.Inputs) == 0 {
		return invalid(contract, prefix+"inputs", "requires at least one immutable input")
	}
	seenInputs := make(map[string]struct{}, len(provenance.Inputs))
	for i, input := range provenance.Inputs {
		inputField := fmt.Sprintf("%sinputs[%d]", prefix, i)
		if err := input.Validate(); err != nil {
			return invalid(contract, inputField, err.Error())
		}
		key := input.identityKey()
		if _, exists := seenInputs[key]; exists {
			return invalid(contract, inputField, "duplicates an earlier immutable input")
		}
		seenInputs[key] = struct{}{}
		if self != nil {
			var ref *ContractRef
			switch input.Kind {
			case LineageInputKindContract:
				ref = &input.Contract.Contract
			case LineageInputKindEvidence:
				ref = &input.Evidence.Evidence
			}
			if ref != nil && ref.Kind == self.Kind && ref.ID == self.ID {
				return invalid(contract, inputField, "must not refer to the output itself")
			}
		}
	}
	want, err := ComputeInputFingerprint(provenance.Inputs)
	if err != nil {
		return invalid(contract, prefix+"input_fingerprint", err.Error())
	}
	if err := provenance.InputFingerprint.Validate(); err != nil {
		return invalid(contract, prefix+"input_fingerprint", err.Error())
	}
	if provenance.InputFingerprint != want {
		return invalid(contract, prefix+"input_fingerprint", "does not match the ordered immutable inputs")
	}
	if err := provenance.Producer.Validate(); err != nil {
		return invalid(contract, prefix+"producer", err.Error())
	}
	seenComponents := map[string]struct{}{provenance.Producer.ID: {}}
	for i, component := range provenance.Components {
		componentField := fmt.Sprintf("%scomponents[%d]", prefix, i)
		if err := component.Validate(); err != nil {
			return invalid(contract, componentField, err.Error())
		}
		if _, exists := seenComponents[component.ID]; exists {
			return invalid(contract, componentField+".id", "duplicates a producer or earlier component identity")
		}
		seenComponents[component.ID] = struct{}{}
	}
	return nil
}

// ComputeInputFingerprint hashes a versioned deterministic JSON envelope of
// validated, ordered immutable input references.
func ComputeInputFingerprint(inputs []LineageInput) (ContentDigest, error) {
	if len(inputs) == 0 {
		return ContentDigest{}, invalid("provenance", "inputs", "requires at least one immutable input")
	}
	for i, input := range inputs {
		if err := input.Validate(); err != nil {
			return ContentDigest{}, fmt.Errorf("lineage input %d: %w", i, err)
		}
	}
	envelope := struct {
		Version string         `json:"version"`
		Inputs  []LineageInput `json:"inputs"`
	}{Version: "jax.lineage-inputs/v1", Inputs: inputs}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return ContentDigest{}, fmt.Errorf("marshal lineage input envelope: %w", err)
	}
	return DigestBytes(raw), nil
}

func provenanceCoversContractRefs(provenance Provenance, refs []ContractRef) bool {
	covered := make(map[string]bool)
	for _, input := range provenance.Inputs {
		switch input.Kind {
		case LineageInputKindContract:
			ref := input.Contract.Contract
			covered[string(ref.Kind)+"\x00"+ref.ID+"\x00"+string(ref.ContractVersion)] = true
		case LineageInputKindEvidence:
			ref := input.Evidence.Evidence
			covered[string(ref.Kind)+"\x00"+ref.ID+"\x00"+string(ref.ContractVersion)] = true
		}
	}
	for _, ref := range refs {
		if !covered[string(ref.Kind)+"\x00"+ref.ID+"\x00"+string(ref.ContractVersion)] {
			return false
		}
	}
	return true
}

func provenanceCoversEvidenceIDs(provenance Provenance, ids []EvidenceID) bool {
	covered := make(map[string]bool)
	for _, input := range provenance.Inputs {
		if input.Kind == LineageInputKindEvidence {
			covered[input.Evidence.Evidence.ID] = true
		} else if input.Kind == LineageInputKindContract && input.Contract.Contract.Kind == ContractKindEvidence {
			covered[input.Contract.Contract.ID] = true
		}
	}
	for _, id := range ids {
		if !covered[string(id)] {
			return false
		}
	}
	return true
}

func provenanceCoversContractRefsFlexibleRun(provenance Provenance, refs []ContractRef, runID ResearchRunID) bool {
	basis := make([]ContractRef, 0, len(refs))
	for _, ref := range refs {
		if ref.Kind != ContractKindResearchRun || ref.ID != string(runID) {
			basis = append(basis, ref)
		}
	}
	if !provenanceCoversContractRefs(provenance, basis) {
		return false
	}
	return provenanceCoversResearchRun(provenance, runID)
}

func provenanceCoversResearchRun(provenance Provenance, runID ResearchRunID) bool {
	for _, input := range provenance.Inputs {
		if input.Kind == LineageInputKindContract && input.Contract.Contract.Kind == ContractKindResearchRun && input.Contract.Contract.ID == string(runID) {
			return true
		}
	}
	return false
}

func componentMatchesRef(component ComponentIdentity, ref ComponentRef) bool {
	return component.Kind == ref.Kind && component.Name == ref.Name && component.Version.Value == ref.Version
}
