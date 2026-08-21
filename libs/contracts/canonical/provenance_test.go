package canonical

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDigestBytesIsDeterministicAndSensitive(t *testing.T) {
	first := DigestBytes([]byte("exact provider bytes\r\n"))
	second := DigestBytes([]byte("exact provider bytes\r\n"))
	changed := DigestBytes([]byte("exact provider bytes\n"))
	if first != second {
		t.Fatalf("same bytes produced different digest: %#v != %#v", first, second)
	}
	if first == changed {
		t.Fatal("changed bytes produced the same digest")
	}
	if err := first.VerifyBytes([]byte("exact provider bytes\r\n")); err != nil {
		t.Fatalf("VerifyBytes() error = %v", err)
	}
	if err := first.VerifyBytes([]byte("tampered")); err == nil {
		t.Fatal("VerifyBytes() accepted changed content")
	} else {
		var mismatch *DigestMismatchError
		if !errors.As(err, &mismatch) {
			t.Fatalf("VerifyBytes() error type = %T, want *DigestMismatchError", err)
		}
	}
}

func TestContentDigestValidationFailsClosed(t *testing.T) {
	valid := DigestBytes([]byte("content"))
	tests := []struct {
		name   string
		digest ContentDigest
	}{
		{"empty", ContentDigest{}},
		{"unsupported_algorithm", ContentDigest{Algorithm: "md5", Value: strings.Repeat("0", 32)}},
		{"wrong_length", ContentDigest{Algorithm: DigestAlgorithmSHA256, Value: "abcd"}},
		{"malformed_hex", ContentDigest{Algorithm: DigestAlgorithmSHA256, Value: strings.Repeat("z", 64)}},
		{"non_canonical_encoding", ContentDigest{Algorithm: DigestAlgorithmSHA256, Value: strings.ToUpper(valid.Value)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.digest.Validate(); err == nil {
				t.Fatal("Validate() accepted malformed digest")
			} else {
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("error type = %T, want *ValidationError", err)
				}
			}
		})
	}
}

func TestCanonicalContractBytesAreStableAndVersionBound(t *testing.T) {
	event := validContracts().event
	first, err := CanonicalContractBytes(event)
	if err != nil {
		t.Fatalf("CanonicalContractBytes() error = %v", err)
	}
	second, err := CanonicalContractBytes(event)
	if err != nil {
		t.Fatalf("second CanonicalContractBytes() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("canonical bytes are unstable:\n%s\n%s", first, second)
	}
	if !bytes.Contains(first, []byte(`"canonicalization":"jax.canonical-contract-json/v1"`)) ||
		!bytes.Contains(first, []byte(`"contract_version":"jax.event/v1"`)) {
		t.Fatalf("canonical envelope is missing identity: %s", first)
	}
	identity, err := CanonicalContractContentIdentity(event)
	if err != nil {
		t.Fatalf("CanonicalContractContentIdentity() error = %v", err)
	}
	if err := identity.VerifyCanonicalContract(event); err != nil {
		t.Fatalf("VerifyCanonicalContract() error = %v", err)
	}
	changed := event
	changed.Title = "Changed title"
	if err := identity.VerifyCanonicalContract(changed); err == nil {
		t.Fatal("VerifyCanonicalContract() accepted changed canonical content")
	}
}

func TestProvenanceV2ContractsValidateAndRoundTrip(t *testing.T) {
	fixtures := validProvenanceContracts(t)
	tests := []struct {
		name        string
		value       Contract
		destination func() Contract
	}{
		{"evidence", fixtures.evidence, func() Contract { return &Evidence{} }},
		{"observation", fixtures.observation, func() Contract { return &Observation{} }},
		{"research_run", fixtures.run, func() Contract { return &ResearchRun{} }},
		{"quant_result", fixtures.quant, func() Contract { return &QuantResult{} }},
		{"recommendation", fixtures.recommendation, func() Contract { return &Recommendation{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			encoded, err := EncodeJSON(test.value)
			if err != nil {
				t.Fatalf("EncodeJSON() error = %v", err)
			}
			destination := test.destination()
			if err := DecodeJSON(encoded, destination); err != nil {
				t.Fatalf("DecodeJSON() error = %v", err)
			}
			decoded := reflect.ValueOf(destination).Elem().Interface()
			if !reflect.DeepEqual(test.value, decoded) {
				t.Fatalf("round trip mismatch:\nwant: %#v\n got: %#v", test.value, decoded)
			}
		})
	}
}

func TestV1RejectsV2ProvenanceFields(t *testing.T) {
	observation := validContracts().observation
	observation.Provenance = validProvenanceContracts(t).observation.Provenance
	assertValidationField(t, observation.Validate(), "contract_version")
}

func TestEvidenceRefValidationAndVerification(t *testing.T) {
	ref := validProvenanceContracts(t).evidence.ImmutableRef
	if err := ref.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := ref.Content.Digest.VerifyBytes([]byte("raw SEC filing bytes\n")); err != nil {
		t.Fatalf("VerifyBytes() error = %v", err)
	}

	tests := []struct {
		name string
		edit func(*EvidenceRef)
	}{
		{"malformed_evidence_id", func(ref *EvidenceRef) { ref.Evidence.ID = "not-evidence" }},
		{"incompatible_contract_version", func(ref *EvidenceRef) { ref.Evidence.ContractVersion = "jax.evidence/v99" }},
		{"missing_content_identity", func(ref *EvidenceRef) { ref.Content = ContentIdentity{} }},
		{"normalized_content_masquerading_as_raw", func(ref *EvidenceRef) {
			ref.Content.Representation = ContentRepresentationCanonicalJSON
			ref.Content.Canonicalization = CanonicalJSONIdentityV1
		}},
		{"empty_source_identity", func(ref *EvidenceRef) { ref.Source.ID = "" }},
		{"source_provider_confusion", func(ref *EvidenceRef) { ref.Source.Kind = SourceKindProvider }},
		{"invalid_provider_namespace", func(ref *EvidenceRef) { ref.Provider.Namespace = "SEC API" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalidRef := *ref
			if ref.Provider != nil {
				provider := *ref.Provider
				invalidRef.Provider = &provider
			}
			test.edit(&invalidRef)
			if err := invalidRef.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid EvidenceRef")
			}
		})
	}
}

func TestProvenanceValidationRejectsInvalidLineage(t *testing.T) {
	fixtures := validProvenanceContracts(t)
	base := *fixtures.quant.Provenance
	tests := []struct {
		name string
		edit func(*Provenance)
	}{
		{"empty_identity", func(provenance *Provenance) { provenance.ID = "" }},
		{"missing_component_version", func(provenance *Provenance) { provenance.Producer.Version = VersionIdentity{} }},
		{"duplicate_lineage_input", func(provenance *Provenance) {
			provenance.Inputs = append(provenance.Inputs, provenance.Inputs[0])
			provenance.InputFingerprint = mustFingerprint(t, provenance.Inputs)
		}},
		{"fingerprint_mismatch", func(provenance *Provenance) { provenance.InputFingerprint = DigestBytes([]byte("wrong")) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalidProvenance := base
			invalidProvenance.Inputs = append([]LineageInput(nil), base.Inputs...)
			test.edit(&invalidProvenance)
			if err := invalidProvenance.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid provenance")
			}
		})
	}
}

func TestDerivedContractRejectsSelfReferentialLineage(t *testing.T) {
	fixtures := validProvenanceContracts(t)
	result := fixtures.quant
	selfContent := mustCanonicalIdentity(t, result)
	self := ImmutableContractRef{
		Contract: ContractRef{Kind: ContractKindQuantResult, ID: string(result.ID), ContractVersion: result.ContractVersion},
		Revision: RevisionIdentity{Namespace: "jax.record_revision", Value: "1"},
		Content:  selfContent,
	}
	result.Provenance.Inputs = []LineageInput{{Kind: LineageInputKindContract, Contract: &self}}
	result.Provenance.InputFingerprint = mustFingerprint(t, result.Provenance.Inputs)
	assertValidationField(t, result.Validate(), "provenance.inputs[0]")
}

func TestQuantResultRequiresImmutableProducingRun(t *testing.T) {
	result := validProvenanceContracts(t).quant
	result.Provenance.Inputs = result.Provenance.Inputs[:1]
	result.Provenance.InputFingerprint = mustFingerprint(t, result.Provenance.Inputs)
	assertValidationField(t, result.Validate(), "provenance.inputs")
}

func TestDatasetSnapshotIdentityDistinguishesRevisions(t *testing.T) {
	fixtures := validProvenanceContracts(t)
	first := fixtures.dataset
	if err := first.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	second := first
	second.SnapshotID = "dss_prices_20260822"
	second.Revision.Value = "2026-08-22"
	second.Content = RawContentIdentity([]byte("new snapshot bytes"))
	if first.SnapshotID == second.SnapshotID || first.Content == second.Content {
		t.Fatal("later dataset revision is not distinguishable")
	}
	inputsA := []LineageInput{{Kind: LineageInputKindDataset, Dataset: &first}}
	inputsB := []LineageInput{{Kind: LineageInputKindDataset, Dataset: &second}}
	if mustFingerprint(t, inputsA) == mustFingerprint(t, inputsB) {
		t.Fatal("dataset revision change did not change input fingerprint")
	}
}

func TestAIProvenanceIdentifiesProviderModelPromptPolicyToolAndBuild(t *testing.T) {
	fixtures := validProvenanceContracts(t)
	provider := ProviderIdentity{ID: "pvd_openai", Namespace: "openai.responses"}
	model := ComponentIdentity{
		ID:       "cmp_model_luna",
		Kind:     ComponentKindModel,
		Name:     "gpt-5.6-luna",
		Version:  VersionIdentity{Namespace: "provider.model_id", Value: "gpt-5.6-luna-2026-08-01"},
		Provider: &provider,
	}
	promptContent := RawContentIdentity([]byte("exact prompt template bytes"))
	prompt := ComponentIdentity{
		ID:      "cmp_prompt_research",
		Kind:    ComponentKindPrompt,
		Name:    "research synthesis prompt",
		Version: VersionIdentity{Namespace: "jax.prompt", Value: "research-synthesis-v3"},
		Content: &promptContent,
	}
	policy := component(ComponentKindPolicy, "recommendation validator", "2.1.0")
	tool := component(ComponentKindTool, "evidence lookup", "1.4.0")
	build := component(ComponentKindSoftwareBuild, "jax research runtime", "fb24f13")
	provenance := provenance(t, "pvn_ai_output", model, []LineageInput{{Kind: LineageInputKindEvidence, Evidence: fixtures.evidence.ImmutableRef}})
	provenance.Components = []ComponentIdentity{prompt, policy, tool, build}
	if err := provenance.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	withoutProvider := model
	withoutProvider.Provider = nil
	if err := withoutProvider.Validate(); err == nil {
		t.Fatal("model component accepted missing provider identity")
	}
	withoutPromptDigest := prompt
	withoutPromptDigest.Content = nil
	if err := withoutPromptDigest.Validate(); err == nil {
		t.Fatal("prompt component accepted missing content identity")
	}
}

type provenanceFixtures struct {
	evidence       Evidence
	observation    Observation
	run            ResearchRun
	quant          QuantResult
	recommendation Recommendation
	dataset        DatasetSnapshotRef
}

func validProvenanceContracts(t *testing.T) provenanceFixtures {
	t.Helper()
	v1 := validContracts()
	base := v1.event.CreatedAt.Add(-3 * time.Minute)
	published := v1.evidence.PublishedAt
	provider := ProviderIdentity{ID: "pvd_sec_data", Namespace: "sec.data_api"}
	source := SourceIdentity{ID: "src_sec_edgar", Kind: SourceKindRegulator}
	evidenceContractRef := ContractRef{Kind: ContractKindEvidence, ID: string(v1.evidence.ID), ContractVersion: EvidenceContractV2}
	evidenceRef := EvidenceRef{
		ContractVersion: EvidenceRefContractV1,
		Evidence:        evidenceContractRef,
		Content:         RawContentIdentity([]byte("raw SEC filing bytes\n")),
		Source:          source,
		Provider:        &provider,
		Revision:        RevisionIdentity{Namespace: "sec.accession", Value: "0000320193-26-000001"},
		PublishedAt:     published,
		CollectedAt:     v1.evidence.CollectedAt,
	}
	evidence := v1.evidence
	evidence.ContractVersion = EvidenceContractV2
	evidence.ImmutableRef = &evidenceRef

	evidenceLineage := LineageInput{Kind: LineageInputKindEvidence, Evidence: &evidenceRef}
	observationProducer := component(ComponentKindNormalizer, "market observation normalizer", "1.0.0")
	observation := v1.observation
	observation.ContractVersion = ObservationContractV2
	observation.Source.Kind = SourceKindExchange
	observation.Provenance = provenance(t, "pvn_observation", observationProducer, []LineageInput{evidenceLineage})

	eventImmutable := immutableContract(t, ContractRef{Kind: ContractKindEvent, ID: string(v1.event.ID), ContractVersion: EventContractV1}, v1.event)
	observationImmutable := immutableContract(t, ContractRef{Kind: ContractKindObservation, ID: string(observation.ID), ContractVersion: ObservationContractV2}, observation)
	runProducer := component(ComponentKindMethod, v1.run.Method.Name, v1.run.Method.Version)
	run := v1.run
	run.ContractVersion = ResearchRunContractV2
	run.InputRefs[1].ContractVersion = ObservationContractV2
	run.OutputRefs[0].ContractVersion = QuantResultContractV2
	run.OutputRefs[1].ContractVersion = RecommendationContractV2
	run.Provenance = provenance(t, "pvn_run", runProducer, []LineageInput{
		{Kind: LineageInputKindContract, Contract: &eventImmutable},
		{Kind: LineageInputKindContract, Contract: &observationImmutable},
	})

	quantProducer := component(ComponentKindAlgorithm, v1.quant.Method.Name, v1.quant.Method.Version)
	quant := v1.quant
	quant.ContractVersion = QuantResultContractV2
	quant.InputRefs[0].ContractVersion = ObservationContractV2
	runImmutable := immutableContract(t, ContractRef{Kind: ContractKindResearchRun, ID: string(run.ID), ContractVersion: ResearchRunContractV2}, run)
	quant.Provenance = provenance(t, "pvn_quant", quantProducer, []LineageInput{
		{Kind: LineageInputKindContract, Contract: &observationImmutable},
		{Kind: LineageInputKindContract, Contract: &runImmutable},
	})

	evidenceImmutableInput := evidenceLineage
	quantImmutable := immutableContract(t, ContractRef{Kind: ContractKindQuantResult, ID: string(quant.ID), ContractVersion: QuantResultContractV2}, quant)
	recommendationProducer := component(ComponentKindPolicy, "research recommendation policy", "1.0.0")
	recommendation := v1.recommendation
	recommendation.ContractVersion = RecommendationContractV2
	recommendation.Basis[1].ContractVersion = EvidenceContractV2
	recommendation.Basis[2].ContractVersion = ObservationContractV2
	recommendation.Basis[3].ContractVersion = QuantResultContractV2
	recommendation.Provenance = provenance(t, "pvn_recommendation", recommendationProducer, []LineageInput{
		{Kind: LineageInputKindContract, Contract: &eventImmutable},
		evidenceImmutableInput,
		{Kind: LineageInputKindContract, Contract: &observationImmutable},
		{Kind: LineageInputKindContract, Contract: &quantImmutable},
		{Kind: LineageInputKindContract, Contract: &runImmutable},
	})

	asOf := base
	dataset := DatasetSnapshotRef{
		ContractVersion: DatasetSnapshotContractV1,
		Dataset: DatasetIdentity{
			ID:            "dset_prices_aapl",
			ExternalID:    ExternalID{Namespace: "jax.dataset_registry", Value: "4ca87d11-78c0-4e27-a92d-090f0b0cf019"},
			SchemaVersion: VersionIdentity{Namespace: "jax.dataset_schema", Value: "ohlcv_v1"},
		},
		SnapshotID:  "dss_prices_20260821",
		Revision:    RevisionIdentity{Namespace: "dataset.snapshot_date", Value: "2026-08-21"},
		Content:     RawContentIdentity([]byte("snapshot bytes")),
		AsOf:        &asOf,
		CollectedAt: v1.evidence.CollectedAt,
	}

	return provenanceFixtures{evidence, observation, run, quant, recommendation, dataset}
}

func component(kind ComponentKind, name, version string) ComponentIdentity {
	return ComponentIdentity{
		ID:      "cmp_" + strings.ReplaceAll(strings.ReplaceAll(name, " ", "_"), ".", "_"),
		Kind:    kind,
		Name:    name,
		Version: VersionIdentity{Namespace: "semver", Value: version},
	}
}

func provenance(t *testing.T, id string, producer ComponentIdentity, inputs []LineageInput) *Provenance {
	t.Helper()
	return &Provenance{
		ContractVersion:  ProvenanceContractV1,
		ID:               id,
		Inputs:           inputs,
		InputFingerprint: mustFingerprint(t, inputs),
		Producer:         producer,
	}
}

func mustFingerprint(t *testing.T, inputs []LineageInput) ContentDigest {
	t.Helper()
	fingerprint, err := ComputeInputFingerprint(inputs)
	if err != nil {
		t.Fatalf("ComputeInputFingerprint() error = %v", err)
	}
	return fingerprint
}

func immutableContract(t *testing.T, ref ContractRef, value Contract) ImmutableContractRef {
	t.Helper()
	return ImmutableContractRef{
		Contract: ref,
		Revision: RevisionIdentity{Namespace: "jax.record_revision", Value: "1"},
		Content:  mustCanonicalIdentity(t, value),
	}
}

func mustCanonicalIdentity(t *testing.T, value Contract) ContentIdentity {
	t.Helper()
	identity, err := CanonicalContractContentIdentity(value)
	if err != nil {
		t.Fatalf("CanonicalContractContentIdentity() error = %v", err)
	}
	return identity
}
