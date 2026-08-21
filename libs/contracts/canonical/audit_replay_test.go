package canonical

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRepresentativeHistoricalOutputReconstruction(t *testing.T) {
	proof := representativeReplayProof(t)
	if err := ValidateAuditTrail(proof.events); err != nil {
		t.Fatalf("ValidateAuditTrail() error = %v", err)
	}
	verification, err := VerifyReplayManifest(proof.manifest, proof.materials)
	if err != nil {
		t.Fatalf("VerifyReplayManifest() error = %v", err)
	}
	if verification.Claim != ReplayClaimExact || verification.AuditEventCount != 5 {
		t.Fatalf("verification = %#v", verification)
	}
	if verification.Target.Contract.ID != string(proof.contracts.recommendation.ID) {
		t.Fatalf("target = %q, want %q", verification.Target.Contract.ID, proof.contracts.recommendation.ID)
	}
	first, err := CanonicalContractBytes(proof.manifest)
	if err != nil {
		t.Fatalf("CanonicalContractBytes(manifest) error = %v", err)
	}
	second, err := CanonicalContractBytes(proof.manifest)
	if err != nil {
		t.Fatalf("second CanonicalContractBytes(manifest) error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("replay manifest canonical bytes are not deterministic")
	}
	if proof.contracts.evidence.ImmutableRef.Provider == nil || proof.contracts.evidence.ImmutableRef.Source.ID == "" {
		t.Fatal("proof does not identify both source and provider")
	}
	if proof.contracts.recommendation.ExecutionAuthority != ExecutionAuthorityNone {
		t.Fatal("proof escaped research-only execution authority")
	}
}

func TestAuditAndReplayContractsStrictRoundTrip(t *testing.T) {
	proof := representativeReplayProof(t)
	tests := []struct {
		name        string
		value       Contract
		destination Contract
	}{
		{"audit_event", proof.events[0], &AuditEvent{}},
		{"replay_manifest", proof.manifest, &ReplayManifest{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeJSON(test.value)
			if err != nil {
				t.Fatalf("EncodeJSON() error = %v", err)
			}
			if err := DecodeJSON(encoded, test.destination); err != nil {
				t.Fatalf("DecodeJSON() error = %v", err)
			}
			decoded := reflect.ValueOf(test.destination).Elem().Interface()
			if !reflect.DeepEqual(test.value, decoded) {
				t.Fatalf("round trip mismatch:\nwant: %#v\n got: %#v", test.value, decoded)
			}
		})
	}
}

func TestAuditEventValidationFailsClosed(t *testing.T) {
	proof := representativeReplayProof(t)
	base := proof.events[1]
	tests := []struct {
		name  string
		field string
		edit  func(*AuditEvent)
	}{
		{"empty_id", "id", func(event *AuditEvent) { event.ID = "" }},
		{"empty_version", "contract_version", func(event *AuditEvent) { event.ContractVersion = "" }},
		{"unsupported_version", "contract_version", func(event *AuditEvent) { event.ContractVersion = "jax.audit_event/v2" }},
		{"invalid_subject", "subject", func(event *AuditEvent) { event.Subject.ID = "not-canonical" }},
		{"missing_provenance", "action", func(event *AuditEvent) { event.ProvenanceID = "" }},
		{"missing_input", "action", func(event *AuditEvent) { event.Inputs = nil; event.InputFingerprint = nil }},
		{"input_fingerprint_mismatch", "input_fingerprint", func(event *AuditEvent) { wrong := DigestBytes([]byte("wrong")); event.InputFingerprint = &wrong }},
		{"self_causation", "causation_id", func(event *AuditEvent) { id := event.ID; event.CausationID = &id }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalidEvent := base
			invalidEvent.Inputs = append([]LineageInput(nil), base.Inputs...)
			test.edit(&invalidEvent)
			assertValidationField(t, invalidEvent.Validate(), test.field)
		})
	}
}

func TestAuditTrailRejectsDuplicateIdentityAndInvalidOrder(t *testing.T) {
	proof := representativeReplayProof(t)
	duplicate := append([]AuditEvent(nil), proof.events...)
	duplicate[4].ID = duplicate[0].ID
	if err := ValidateAuditTrail(duplicate); err == nil {
		t.Fatal("ValidateAuditTrail() accepted duplicate event identity")
	} else {
		var trailErr *AuditTrailError
		if !errors.As(err, &trailErr) || trailErr.Code != "duplicate_event_identity" {
			t.Fatalf("error = %T %v, want duplicate_event_identity", err, err)
		}
	}

	badSequence := append([]AuditEvent(nil), proof.events...)
	badSequence[1].Sequence = 3
	if err := ValidateAuditTrail(badSequence); err == nil {
		t.Fatal("ValidateAuditTrail() accepted a sequence gap")
	} else {
		var trailErr *AuditTrailError
		if !errors.As(err, &trailErr) || trailErr.Code != "invalid_stream_sequence" {
			t.Fatalf("error = %T %v, want invalid_stream_sequence", err, err)
		}
	}

	duplicateIdempotency := append([]AuditEvent(nil), proof.events...)
	duplicateIdempotency[4].IdempotencyID = duplicateIdempotency[0].IdempotencyID
	if err := ValidateAuditTrail(duplicateIdempotency); err == nil {
		t.Fatal("ValidateAuditTrail() accepted duplicate idempotency identity")
	} else {
		var trailErr *AuditTrailError
		if !errors.As(err, &trailErr) || trailErr.Code != "duplicate_idempotency_identity" {
			t.Fatalf("error = %T %v, want duplicate_idempotency_identity", err, err)
		}
	}
}

func TestAuditFailureAndSupersessionRemainHistory(t *testing.T) {
	proof := representativeReplayProof(t)
	failed := proof.events[0]
	failed.ID = "aud_validation_failed"
	failed.IdempotencyID = "adi_validation_failed"
	failed.Action = AuditActionProcessingFailed
	failed.Inputs = nil
	failed.InputFingerprint = nil
	failed.Output = nil
	failed.ProvenanceID = ""
	failed.Outcome = AuditOutcomeRejected
	failed.Failure = &AuditFailure{ID: "fail_validation_rejected", Code: AuditFailureValidationRejected, Detail: "The attempted canonical record failed deterministic validation."}
	if err := failed.Validate(); err != nil {
		t.Fatalf("failed AuditEvent.Validate() error = %v", err)
	}
	unsupported := failed
	unsupported.ID = "aud_unsupported_contract"
	unsupported.IdempotencyID = "adi_unsupported_contract"
	unsupported.Subject.ContractVersion = "jax.evidence/v99"
	unsupported.Failure = &AuditFailure{ID: "fail_unsupported_contract", Code: AuditFailureUnsupportedContractVersion, Detail: "The attempted contract version is not supported."}
	if err := unsupported.Validate(); err != nil {
		t.Fatalf("unsupported-version AuditEvent.Validate() error = %v", err)
	}

	prior := *proof.events[0].Output
	correctedEvidence := proof.contracts.evidence
	correctedEvidence.Summary = "Corrected interpretation; original record remains immutable."
	current := immutableContract(t, prior.Contract, correctedEvidence)
	current.Revision.Value = "2"
	superseded := proof.events[0]
	superseded.ID = "aud_evidence_superseded"
	superseded.IdempotencyID = "adi_evidence_superseded"
	superseded.Action = AuditActionRecordSuperseded
	superseded.Output = &current
	superseded.Supersedes = &prior
	if err := superseded.Validate(); err != nil {
		t.Fatalf("supersession AuditEvent.Validate() error = %v", err)
	}
}

func TestContractCompatibilityRequiresExplicitDeterministicTranslation(t *testing.T) {
	source := ContractSchemaRef{Kind: ContractKindRecommendation, Version: RecommendationContractV1}
	target := ContractSchemaRef{Kind: ContractKindRecommendation, Version: RecommendationContractV2}
	if _, err := ResolveContractCompatibility(source, target, nil); err == nil {
		t.Fatal("ResolveContractCompatibility() silently upgraded V1 to V2")
	} else {
		var compatibilityErr *CompatibilityResolutionError
		if !errors.As(err, &compatibilityErr) || compatibilityErr.Code != "translation_not_declared" {
			t.Fatalf("error = %T %v", err, err)
		}
	}

	exact, err := ResolveContractCompatibility(source, source, nil)
	if err != nil || exact.Classification != CompatibilityExact {
		t.Fatalf("exact resolution = %#v, %v", exact, err)
	}

	incompatible, err := V1ToV2WithoutHistoricalProvenance(ContractKindRecommendation)
	if err != nil {
		t.Fatalf("V1ToV2WithoutHistoricalProvenance() error = %v", err)
	}
	resolved, err := ResolveContractCompatibility(source, target, []ContractCompatibility{incompatible})
	if err != nil || resolved.Classification != CompatibilityIncompatible || resolved.ReasonCode != "insufficient_historical_provenance" {
		t.Fatalf("V1/V2 resolution = %#v, %v", resolved, err)
	}

	future := ContractSchemaRef{Kind: ContractKindRecommendation, Version: "jax.recommendation/v99"}
	if _, err := ResolveContractCompatibility(source, future, nil); err == nil {
		t.Fatal("ResolveContractCompatibility() accepted a future version")
	} else {
		var compatibilityErr *CompatibilityResolutionError
		if !errors.As(err, &compatibilityErr) || compatibilityErr.Code != "unsupported_target_version" {
			t.Fatalf("error = %T %v", err, err)
		}
	}
}

func TestCompatibilityRejectsLossMasqueradingAsExact(t *testing.T) {
	source := ContractSchemaRef{Kind: ContractKindEvidence, Version: EvidenceContractV1}
	target := ContractSchemaRef{Kind: ContractKindEvidence, Version: EvidenceContractV2}
	assessment := ContractCompatibility{
		ContractVersion: ContractCompatibilityV1,
		ID:              "cpa_invalid_exact",
		Source:          source,
		Target:          target,
		Classification:  CompatibilityExact,
	}
	assertValidationField(t, assessment.Validate(), "classification")

	adapterBytes := []byte("evidence v1 to v2 mapping specification")
	adapterContent := RawContentIdentity(adapterBytes)
	assessment.Classification = CompatibilityLossyTranslation
	assessment.Translator = &ComponentIdentity{
		ID:      "cmp_evidence_v1_v2_mapping",
		Kind:    ComponentKindMapping,
		Name:    "evidence v1 to v2 mapping",
		Version: VersionIdentity{Namespace: "semver", Value: "1.0.0"},
		Content: &adapterContent,
	}
	assertValidationField(t, assessment.Validate(), "losses")
	assessment.Losses = []CompatibilityLoss{{Code: "historical_provenance_absent", Detail: "V1 carries no immutable provenance identity."}}
	if err := assessment.Validate(); err != nil {
		t.Fatalf("declared lossy assessment error = %v", err)
	}
	assessment.Classification = CompatibilityLosslessTranslation
	assessment.Losses = nil
	if err := assessment.Validate(); err != nil {
		t.Fatalf("declared lossless assessment error = %v", err)
	}
}

func TestReplayVerificationDetectsMissingAndTamperedMaterials(t *testing.T) {
	proof := representativeReplayProof(t)

	missing := proof.materials
	missing.Contracts = append([]ContractMaterial(nil), proof.materials.Contracts[1:]...)
	assertReplayErrorCode(t, verifyReplayError(proof.manifest, missing), "missing_immutable_input")

	tamperedEvidence := proof.materials
	tamperedEvidence.Evidence = append([]EvidenceMaterial(nil), proof.materials.Evidence...)
	tamperedEvidence.Evidence[0].Bytes = []byte("tampered provider response")
	assertReplayErrorCode(t, verifyReplayError(proof.manifest, tamperedEvidence), "content_digest_mismatch")

	tamperedContract := proof.materials
	tamperedContract.Contracts = append([]ContractMaterial(nil), proof.materials.Contracts...)
	changed := proof.contracts.recommendation
	changed.Reasons = append([]RecommendationReason(nil), changed.Reasons...)
	changed.Reasons[0].Summary = "Tampered historical conclusion."
	tamperedContract.Contracts[0].Value = changed
	assertReplayErrorCode(t, verifyReplayError(proof.manifest, tamperedContract), "content_digest_mismatch")

	differentComponent := proof.materials
	differentComponent.Components = append([]ComponentMaterial(nil), proof.materials.Components...)
	differentComponent.Components[0].Identity.Version.Value = "2.0.0"
	assertReplayErrorCode(t, verifyReplayError(proof.manifest, differentComponent), "component_identity_mismatch")
}

func TestReplayManifestRejectsFingerprintAndMissingProducerVersion(t *testing.T) {
	proof := representativeReplayProof(t)
	wrongFingerprint := proof.manifest
	wrong := DigestBytes([]byte("wrong"))
	wrongFingerprint.InputFingerprint = &wrong
	assertValidationField(t, wrongFingerprint.Validate(), "input_fingerprint")

	missingVersion := proof.manifest
	missingVersion.Producer.Version = VersionIdentity{}
	assertValidationField(t, missingVersion.Validate(), "producer")

	missingRequiredProducer := proof.manifest
	missingRequiredProducer.RequiredComponentIDs = missingRequiredProducer.RequiredComponentIDs[1:]
	assertValidationField(t, missingRequiredProducer.Validate(), "required_component_ids")
}

func TestExactReplayWithHistoricalModelUseRequiresStoredResponse(t *testing.T) {
	proof := representativeReplayProof(t)
	modelResponse, responseRef, model := historicalModelResponse(t, proof)
	manifest := proof.manifest
	manifest.Components = append(manifest.Components, model)
	manifest.Inputs = append(manifest.Inputs, LineageInput{Kind: LineageInputKindContract, Contract: &responseRef})
	fingerprint := mustFingerprint(t, manifest.Inputs)
	manifest.InputFingerprint = &fingerprint
	manifest.StoredResponses = []ImmutableContractRef{responseRef}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("model history manifest Validate() error = %v", err)
	}

	// The provider/model is intentionally not required or invoked. Omitting the
	// stored immutable response still makes exact historical reconstruction fail.
	materials := proof.materials
	materials.Contracts = append([]ContractMaterial(nil), materials.Contracts...)
	_ = modelResponse
	assertReplayErrorCode(t, verifyReplayError(manifest, materials), "missing_historical_ai_response")

	reinference := manifest
	reinference.RequiredComponentIDs = append(reinference.RequiredComponentIDs, model.ID)
	assertValidationField(t, reinference.Validate(), "required_component_ids")
}

func verifyReplayError(manifest ReplayManifest, materials ReplayMaterials) error {
	_, err := VerifyReplayManifest(manifest, materials)
	return err
}

func assertReplayErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("verification accepted failure case, want %s", code)
	}
	var replayErr *ReplayVerificationError
	if !errors.As(err, &replayErr) {
		t.Fatalf("error type = %T, want *ReplayVerificationError (%v)", err, err)
	}
	if replayErr.Code != code {
		t.Fatalf("error code = %q, want %q (%v)", replayErr.Code, code, err)
	}
}

type replayProof struct {
	contracts provenanceFixtures
	events    []AuditEvent
	manifest  ReplayManifest
	materials ReplayMaterials
}

func representativeReplayProof(t *testing.T) replayProof {
	t.Helper()
	contracts := validProvenanceContracts(t)
	v1 := validContracts()
	evidenceImmutable := immutableContract(t, ContractRef{Kind: ContractKindEvidence, ID: string(contracts.evidence.ID), ContractVersion: EvidenceContractV2}, contracts.evidence)
	observationImmutable := immutableContract(t, ContractRef{Kind: ContractKindObservation, ID: string(contracts.observation.ID), ContractVersion: ObservationContractV2}, contracts.observation)
	runImmutable := immutableContract(t, ContractRef{Kind: ContractKindResearchRun, ID: string(contracts.run.ID), ContractVersion: ResearchRunContractV2}, contracts.run)
	quantImmutable := immutableContract(t, ContractRef{Kind: ContractKindQuantResult, ID: string(contracts.quant.ID), ContractVersion: QuantResultContractV2}, contracts.quant)
	recommendationImmutable := immutableContract(t, ContractRef{Kind: ContractKindRecommendation, ID: string(contracts.recommendation.ID), ContractVersion: RecommendationContractV2}, contracts.recommendation)

	evidenceProducer := component(ComponentKindNormalizer, "evidence normalizer", "1.0.0")
	steps := []struct {
		id         AuditEventID
		action     AuditAction
		subject    ImmutableContractRef
		inputs     []LineageInput
		producer   ComponentIdentity
		provenance string
	}{
		{"aud_evidence_created", AuditActionRecordCreated, evidenceImmutable, []LineageInput{{Kind: LineageInputKindEvidence, Evidence: contracts.evidence.ImmutableRef}}, evidenceProducer, "pvn_evidence_audit"},
		{"aud_observation_created", AuditActionRecordCreated, observationImmutable, contracts.observation.Provenance.Inputs, contracts.observation.Provenance.Producer, contracts.observation.Provenance.ID},
		{"aud_research_completed", AuditActionProcessingCompleted, runImmutable, contracts.run.Provenance.Inputs, contracts.run.Provenance.Producer, contracts.run.Provenance.ID},
		{"aud_quant_completed", AuditActionProcessingCompleted, quantImmutable, contracts.quant.Provenance.Inputs, contracts.quant.Provenance.Producer, contracts.quant.Provenance.ID},
		{"aud_recommendation_completed", AuditActionProcessingCompleted, recommendationImmutable, contracts.recommendation.Provenance.Inputs, contracts.recommendation.Provenance.Producer, contracts.recommendation.Provenance.ID},
	}
	base := contracts.recommendation.CreatedAt.Add(time.Minute)
	events := make([]AuditEvent, 0, len(steps))
	for i, step := range steps {
		fingerprint := mustFingerprint(t, step.inputs)
		occurred := base.Add(time.Duration(i) * time.Minute)
		event := AuditEvent{
			ContractVersion:  AuditEventContractV1,
			ID:               step.id,
			StreamID:         "ast_aapl_research_history",
			Sequence:         uint64(i + 1),
			IdempotencyID:    "adi_" + string(step.id[4:]),
			Action:           step.action,
			Subject:          AuditSubjectRef{Kind: step.subject.Contract.Kind, ID: step.subject.Contract.ID, ContractVersion: step.subject.Contract.ContractVersion},
			Inputs:           step.inputs,
			InputFingerprint: &fingerprint,
			Output:           &step.subject,
			Producer:         step.producer,
			ProvenanceID:     step.provenance,
			Outcome:          AuditOutcomeSucceeded,
			CorrelationID:    "cor_aapl_research_history",
			KnowledgeCutoff:  occurred,
			OccurredAt:       occurred,
			RecordedAt:       occurred.Add(time.Second),
		}
		if i > 0 {
			parent := events[i-1].ID
			event.CausationID = &parent
		}
		if err := event.Validate(); err != nil {
			t.Fatalf("audit event %d Validate() error = %v", i, err)
		}
		events = append(events, event)
	}
	auditRefs := make([]AuditEventRef, len(events))
	for i, event := range events {
		ref, err := NewAuditEventRef(event, RevisionIdentity{Namespace: "jax.audit_revision", Value: "1"})
		if err != nil {
			t.Fatalf("NewAuditEventRef(%d) error = %v", i, err)
		}
		auditRefs[i] = ref
	}

	manifestInputs := append([]LineageInput(nil), contracts.recommendation.Provenance.Inputs...)
	manifestInputs = append(manifestInputs, LineageInput{Kind: LineageInputKindContract, Contract: &evidenceImmutable})
	manifestFingerprint := mustFingerprint(t, manifestInputs)
	components := []ComponentIdentity{
		contracts.observation.Provenance.Producer,
		contracts.run.Provenance.Producer,
		contracts.quant.Provenance.Producer,
	}
	manifest := ReplayManifest{
		ContractVersion:  ReplayManifestContractV1,
		ID:               "rpl_aapl_recommendation_history",
		Target:           recommendationImmutable,
		Inputs:           manifestInputs,
		InputFingerprint: &manifestFingerprint,
		Producer:         contracts.recommendation.Provenance.Producer,
		Components:       components,
		RequiredComponentIDs: []string{
			contracts.recommendation.Provenance.Producer.ID,
			contracts.observation.Provenance.Producer.ID,
			contracts.run.Provenance.Producer.ID,
			contracts.quant.Provenance.Producer.ID,
		},
		AuditEvents: auditRefs,
		Claim:       ReplayClaimExact,
		Strategy:    ReplayStrategyImmutableReconstruction,
		CreatedAt:   base.Add(10 * time.Minute),
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("ReplayManifest.Validate() error = %v", err)
	}

	materials := ReplayMaterials{
		Contracts: []ContractMaterial{
			{Reference: recommendationImmutable, Value: contracts.recommendation},
			{Reference: evidenceImmutable, Value: contracts.evidence},
			{Reference: observationImmutable, Value: contracts.observation},
			{Reference: runImmutable, Value: contracts.run},
			{Reference: quantImmutable, Value: contracts.quant},
			{Reference: immutableContract(t, ContractRef{Kind: ContractKindEvent, ID: string(v1.event.ID), ContractVersion: EventContractV1}, v1.event), Value: v1.event},
		},
		Evidence: []EvidenceMaterial{{Reference: *contracts.evidence.ImmutableRef, Bytes: []byte("raw SEC filing bytes\n")}},
		Components: []ComponentMaterial{
			{Identity: contracts.recommendation.Provenance.Producer},
			{Identity: contracts.observation.Provenance.Producer},
			{Identity: contracts.run.Provenance.Producer},
			{Identity: contracts.quant.Provenance.Producer},
		},
		AuditEvents: events,
	}
	return replayProof{contracts: contracts, events: events, manifest: manifest, materials: materials}
}

func historicalModelResponse(t *testing.T, proof replayProof) (Evidence, ImmutableContractRef, ComponentIdentity) {
	t.Helper()
	provider := ProviderIdentity{ID: "pvd_model_provider", Namespace: "historical.model_provider"}
	model := ComponentIdentity{
		ID:       "cmp_historical_model",
		Kind:     ComponentKindModel,
		Name:     "historical advisory model",
		Version:  VersionIdentity{Namespace: "provider.model_id", Value: "historical-model-2026-08-21"},
		Provider: &provider,
	}
	raw := []byte(`{"decision":"WATCH","confidence":0.82}`)
	evidence := proof.contracts.evidence
	evidence.ID = "evd_historical_model_response"
	evidence.Type = EvidenceTypeModelOutput
	evidence.Title = "Stored historical model response"
	evidence.Source = SourceReference{ID: "src_jax_model_archive", Kind: SourceKindInternal, URI: "urn:jax:model-response:historical"}
	evidence.ImmutableRef = &EvidenceRef{
		ContractVersion: EvidenceRefContractV1,
		Evidence:        ContractRef{Kind: ContractKindEvidence, ID: string(evidence.ID), ContractVersion: EvidenceContractV2},
		Content:         RawContentIdentity(raw),
		Source:          SourceIdentity{ID: "src_jax_model_archive", Kind: SourceKindInternal},
		Provider:        &provider,
		Revision:        RevisionIdentity{Namespace: "provider.response_id", Value: "response-20260821-001"},
		PublishedAt:     evidence.PublishedAt,
		CollectedAt:     evidence.CollectedAt,
	}
	inputs := []LineageInput{{Kind: LineageInputKindEvidence, Evidence: proof.contracts.evidence.ImmutableRef}}
	evidence.Provenance = provenance(t, "pvn_historical_model_response", model, inputs)
	if err := evidence.Validate(); err != nil {
		t.Fatalf("historical model Evidence.Validate() error = %v", err)
	}
	ref := immutableContract(t, ContractRef{Kind: ContractKindEvidence, ID: string(evidence.ID), ContractVersion: EvidenceContractV2}, evidence)
	return evidence, ref, model
}
