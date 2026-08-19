package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F2AWorkPackageIdentity             = "WP-00.03C1F2A"
	C1F3GeneralizationTypedLabelsVersion = "ai-shadow-causal-attribution-labels-generalization-v3-v1"
	C1F3BoundaryTypedLabelsVersion       = "ai-shadow-causal-attribution-labels-boundary-v3-v1"
	C1F2AAdjudicationRubricVersion       = "ai-shadow-causal-attribution-adjudication-rubric-c1f2a-v1"
	C1F3ScoringRubricVersion             = "ai-shadow-causal-attribution-scoring-c1f3-v1"

	frozenV6PromptSHA256             = "9a4ee7e3bcc5a2a7e1fdb8f5542ac158f2ae18f46fd457b21bcd643b150db3ca"
	frozenC1FValidatorSHA256         = "0590cf5582d46bd08a0c053be6d83232ac00d83a06a212583190b0a3c0edc207"
	frozenSemanticIdentitySHA256     = "75cedcc94bc8d5c5eb7f4fabba83067ac00857744a10c806be77cea1733e51c8"
	frozenC1FScoringSourceSHA256     = "f54ec91af32a9a11f3c0a7f27eefd3c52624d2360fce613820e7aa39380d8269"
	c1f2aVersionOnObservationRule    = "Once any C1F3 provider output exists, this sidecar is immutable forever; any change requires a new sidecar identity and a new evaluation cell."
	c1f2aHoldoutDisposition          = "CASE-BLIND / CATEGORY-AWARE until the authorized C1F2A opening; frozen before any C1F3 provider output."
	c1f2aAttributionCompletenessRule = "Include every issuer identity required to represent causal structure, exclude irrelevant lexical entities, and preserve meaningful empty, singleton, and multiple principal-proxy candidate sets."
)

type C1F2AQualityControl struct {
	Passes                     int      `json:"passes"`
	FirstPassClean             int      `json:"first_pass_clean"`
	RequiredReconsideration    int      `json:"required_reconsideration"`
	RemainingAmbiguous         int      `json:"remaining_ambiguous"`
	ContractConflicts          int      `json:"contract_conflicts"`
	ReconsideredCases          []string `json:"reconsidered_cases"`
	ReconsiderationResolutions []string `json:"reconsideration_resolutions"`
	Contradictions             []string `json:"contradictions"`
}

type C1F3TypedLabelSidecar struct {
	Version                     string                 `json:"version"`
	DatasetIdentity             string                 `json:"dataset_identity"`
	DatasetVersion              string                 `json:"dataset_version"`
	CreatedAt                   time.Time              `json:"created_at"`
	Fingerprint                 string                 `json:"fingerprint"`
	CaseCount                   int                    `json:"case_count"`
	SourceDataset               TypedLabelSourceLink   `json:"source_dataset"`
	SourceInputLock             TypedLabelSourceLink   `json:"source_input_lock"`
	SourceFreeze                TypedLabelContractLink `json:"source_freeze"`
	Prompt                      TypedLabelContractLink `json:"prompt"`
	OutputContract              TypedLabelContractLink `json:"output_contract"`
	Validator                   TypedLabelContractLink `json:"validator"`
	AttributionPolicy           TypedLabelContractLink `json:"attribution_policy"`
	SemanticIdentityComparator  TypedLabelContractLink `json:"semantic_identity_comparator"`
	Resolver                    TypedLabelContractLink `json:"resolver"`
	Scoring                     TypedLabelContractLink `json:"scoring"`
	AdjudicationRubric          TypedLabelContractLink `json:"adjudication_rubric"`
	AdjudicationMethod          string                 `json:"adjudication_method"`
	AttributionCompletenessRule string                 `json:"attribution_completeness_rule"`
	HoldoutDisposition          string                 `json:"holdout_disposition"`
	VersionOnObservationRule    string                 `json:"version_on_observation_rule"`
	QualityControl              C1F2AQualityControl    `json:"quality_control"`
	Cases                       []TypedExpectedCase    `json:"cases"`
}

func c1f3TypedLabelFingerprint(sidecar C1F3TypedLabelSidecar) (string, error) {
	copy := sidecar
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func loadStrictC1F3TypedLabels(raw []byte) (C1F3TypedLabelSidecar, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var sidecar C1F3TypedLabelSidecar
	if err := decoder.Decode(&sidecar); err != nil {
		return C1F3TypedLabelSidecar{}, fmt.Errorf("decode C1F3 typed-attribution sidecar: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return C1F3TypedLabelSidecar{}, err
	}
	return sidecar, nil
}

func LoadFrozenC1F3TypedLabelSidecar(profile C1F3EvaluationProfile, path string) (C1F3TypedLabelSidecar, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return C1F3TypedLabelSidecar{}, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.TypedSidecarSHA256 {
		return C1F3TypedLabelSidecar{}, fmt.Errorf("C1F3 sidecar file hash changed: got %s want %s", got, profile.TypedSidecarSHA256)
	}
	sidecar, err := loadStrictC1F3TypedLabels(raw)
	if err != nil {
		return C1F3TypedLabelSidecar{}, err
	}
	computed, err := c1f3TypedLabelFingerprint(sidecar)
	if err != nil || sidecar.Fingerprint != computed || sidecar.Fingerprint != profile.TypedSidecarFingerprint {
		return C1F3TypedLabelSidecar{}, fmt.Errorf("C1F3 sidecar fingerprint changed: got %s computed %s want %s", sidecar.Fingerprint, computed, profile.TypedSidecarFingerprint)
	}
	return sidecar, nil
}

func LoadFrozenC1F3Manifest(profile C1F3EvaluationProfile, path string, proxyExposures []string) (DiagnosticManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticManifest{}, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.Dataset.ManifestSHA256 {
		return DiagnosticManifest{}, fmt.Errorf("C1F3 manifest file hash changed: got %s want %s", got, profile.Dataset.ManifestSHA256)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest DiagnosticManifest
	if err = decoder.Decode(&manifest); err != nil {
		return DiagnosticManifest{}, err
	}
	if err = ensureEOF(decoder); err != nil {
		return DiagnosticManifest{}, err
	}
	if manifest.Version != profile.Dataset.Identity || manifest.OutputContract != V5SchemaVersion || manifest.PolicyVersion != "event-asset-resolution-v1" || manifest.LabelVersion != DiagnosticLabelVersion || len(manifest.Events) != profile.Dataset.CaseCount {
		return DiagnosticManifest{}, fmt.Errorf("C1F3 manifest metadata changed")
	}
	computed, err := diagnosticManifestFingerprint(manifest)
	if err != nil || manifest.Fingerprint != computed || manifest.Fingerprint != profile.Dataset.SemanticFingerprint {
		return DiagnosticManifest{}, fmt.Errorf("C1F3 manifest fingerprint changed: got %s computed %s want %s", manifest.Fingerprint, computed, profile.Dataset.SemanticFingerprint)
	}
	allowed := map[string]bool{NoProxyExposure: true}
	for _, exposure := range proxyExposures {
		allowed[exposure] = true
	}
	seen := map[string]bool{}
	for _, event := range manifest.Events {
		if event.ID == "" || event.Category == "" || seen[event.ID] || event.Input.Title == "" || event.Input.PublicationTimestamp.IsZero() || event.Input.ReceiptTimestamp.IsZero() || event.Input.PublicationTimestamp.After(event.Input.ReceiptTimestamp) {
			return DiagnosticManifest{}, fmt.Errorf("C1F3 manifest event %q is invalid", event.ID)
		}
		seen[event.ID] = true
		if err = validateDiagnosticLabel(event.Label, allowed); err != nil {
			return DiagnosticManifest{}, fmt.Errorf("C1F3 manifest event %s: %w", event.ID, err)
		}
	}
	return manifest, nil
}

func ValidateFrozenC1F3InputLock(profile C1F3EvaluationProfile, path string, manifest DiagnosticManifest) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.Dataset.InputLockSHA256 {
		return fmt.Errorf("C1F3 input lock file hash changed: got %s want %s", got, profile.Dataset.InputLockSHA256)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var lock DiagnosticFingerprintLock
	if err = decoder.Decode(&lock); err != nil {
		return err
	}
	if err = ensureEOF(decoder); err != nil {
		return err
	}
	if lock.Version != profile.SourceInputLockIdentity() || lock.ManifestFingerprint != profile.Dataset.SemanticFingerprint || lock.PromptVersion != V5PromptVersion || lock.OutputContract != V5SchemaVersion || lock.PolicyVersion != "event-asset-resolution-v1" || len(lock.Events) != len(manifest.Events) {
		return fmt.Errorf("C1F3 input lock metadata changed")
	}
	computed, err := diagnosticFingerprintLockFingerprint(lock)
	if err != nil || lock.Fingerprint != computed || lock.Fingerprint != profile.Dataset.InputFingerprint {
		return fmt.Errorf("C1F3 input lock fingerprint changed: got %s computed %s want %s", lock.Fingerprint, computed, profile.Dataset.InputFingerprint)
	}
	for index, event := range manifest.Events {
		inputFingerprint, fingerprintErr := EventInputFingerprint(event.Input)
		if fingerprintErr != nil || lock.Events[index].ID != event.ID || lock.Events[index].InputFingerprint != inputFingerprint {
			return fmt.Errorf("C1F3 input lock mismatch at case %s", event.ID)
		}
	}
	return nil
}

func ValidateC1F3TypedLabelSidecar(profile C1F3EvaluationProfile, sidecar C1F3TypedLabelSidecar, manifest DiagnosticManifest, resolver assetresolution.Resolver) error {
	if sidecar.Version != profile.TypedSidecarIdentity || sidecar.DatasetIdentity != profile.Dataset.Identity || sidecar.DatasetVersion != profile.Dataset.Identity ||
		sidecar.CaseCount != profile.Dataset.CaseCount || len(sidecar.Cases) != profile.Dataset.CaseCount || sidecar.QualityControl.Passes != 2 ||
		sidecar.QualityControl.FirstPassClean+sidecar.QualityControl.RequiredReconsideration != sidecar.CaseCount || sidecar.QualityControl.RemainingAmbiguous != 0 || sidecar.QualityControl.ContractConflicts != 0 {
		return fmt.Errorf("C1F3 sidecar identity, count, or two-pass quality control changed")
	}
	if manifest.Version != profile.Dataset.Identity || len(manifest.Events) != len(sidecar.Cases) {
		return fmt.Errorf("C1F3 sidecar source dataset mismatch")
	}
	if sidecar.SourceDataset.Identity != profile.Dataset.Identity || sidecar.SourceDataset.Path != profile.Dataset.ManifestPath || sidecar.SourceDataset.FileSHA256 != profile.Dataset.ManifestSHA256 || sidecar.SourceDataset.SemanticFingerprint != profile.Dataset.SemanticFingerprint ||
		sidecar.SourceInputLock.Path != profile.Dataset.InputLockPath || sidecar.SourceInputLock.FileSHA256 != profile.Dataset.InputLockSHA256 || sidecar.SourceInputLock.SemanticFingerprint != profile.Dataset.InputFingerprint ||
		sidecar.SourceFreeze.SourcePath != profile.Dataset.FreezePath || sidecar.SourceFreeze.SHA256 != profile.Dataset.FreezeSHA256 {
		return fmt.Errorf("C1F3 sidecar source linkage changed")
	}
	if sidecar.Prompt.Identity != V6PromptVersion || sidecar.Prompt.SHA256 != frozenV6PromptSHA256 ||
		sidecar.OutputContract.Identity != V5SchemaVersion || sidecar.OutputContract.SHA256 != frozenV5SchemaSHA256 ||
		sidecar.Validator.Identity != C1FValidatorVersion || sidecar.Validator.SHA256 != frozenC1FValidatorSHA256 ||
		sidecar.AttributionPolicy.Identity != CausalAttributionPolicyVersion || sidecar.AttributionPolicy.SHA256 != frozenC1EPolicySHA256 ||
		sidecar.SemanticIdentityComparator.Identity != IssuerSemanticIdentityVersion || sidecar.SemanticIdentityComparator.SHA256 != frozenSemanticIdentitySHA256 ||
		sidecar.Resolver.Identity != "event-asset-resolution-v1" || sidecar.Resolver.SHA256 != expectedAssetRulesetFileSHA256 ||
		sidecar.Scoring.Identity != C1FScoringVersion || sidecar.Scoring.SHA256 != frozenC1FScoringSourceSHA256 ||
		sidecar.AdjudicationRubric.Identity != C1F2AAdjudicationRubricVersion || sidecar.AdjudicationRubric.SHA256 != profile.AdjudicationRubricSHA256 {
		return fmt.Errorf("C1F3 sidecar frozen semantic binding changed")
	}
	if sidecar.VersionOnObservationRule != c1f2aVersionOnObservationRule || sidecar.HoldoutDisposition != c1f2aHoldoutDisposition || sidecar.AttributionCompletenessRule != c1f2aAttributionCompletenessRule || strings.TrimSpace(sidecar.AdjudicationMethod) == "" {
		return fmt.Errorf("C1F3 sidecar freeze policy changed")
	}
	seen := map[string]bool{}
	for index, expected := range sidecar.Cases {
		source := manifest.Events[index]
		if seen[expected.CaseID] || expected.CaseID != source.ID || expected.DatasetIdentity != profile.Dataset.Identity || expected.DatasetVersion != profile.Dataset.Identity {
			return fmt.Errorf("C1F3 sidecar case identity/order mismatch at position %d", index+1)
		}
		seen[expected.CaseID] = true
		if expected.ExpectedMappingStatus != source.Label.MappingStatus || expected.ExpectedDirectIssuer != source.Label.DirectIssuer || expected.ExpectedProxyExposure != source.Label.ProxyExposure || expected.ExpectedDeterministicResolutionStatus != source.Label.ExpectedResolutionStatus {
			return fmt.Errorf("C1F3 sidecar case %s contradicts frozen v3 mapping", expected.CaseID)
		}
		if expected.AdjudicationStatus != "FROZEN" || expected.ExpectedIssuerAttributions == nil || expected.ExpectedPrincipalProxyCandidates == nil || strings.TrimSpace(expected.TypedAttributionRationale) == "" {
			return fmt.Errorf("C1F3 sidecar case %s is incomplete", expected.CaseID)
		}
		projection := V5StructuredResult{MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure, IssuerAttributions: expected.ExpectedIssuerAttributions, PrincipalProxyCandidates: expected.ExpectedPrincipalProxyCandidates}
		if validationErrors := validateV5Attribution(projection, resolver); len(validationErrors) > 0 {
			return fmt.Errorf("C1F3 sidecar case %s violates C1F/v5 attribution invariants: %s", expected.CaseID, strings.Join(validationErrors, "; "))
		}
		decision, err := ApplyCausalAttributionPolicy(projection)
		if err != nil {
			return fmt.Errorf("C1F3 sidecar case %s policy projection: %w", expected.CaseID, err)
		}
		resolution := ResolveCausalAttributionDecision(decision, source.Input, resolver)
		if resolution.Status != expected.ExpectedDeterministicResolutionStatus {
			return fmt.Errorf("C1F3 sidecar case %s deterministic resolution changed: got %s want %s", expected.CaseID, resolution.Status, expected.ExpectedDeterministicResolutionStatus)
		}
	}
	return nil
}

type C1F3ScoringFreeze struct {
	Version                  string                   `json:"version"`
	CreatedAt                time.Time                `json:"created_at"`
	Fingerprint              string                   `json:"fingerprint"`
	Implementation           TypedLabelContractLink   `json:"implementation"`
	EvaluationScope          string                   `json:"evaluation_scope"`
	InvalidOutputRule        string                   `json:"invalid_output_rule"`
	StrictIdentityRule       string                   `json:"strict_identity_rule"`
	SemanticIdentityRule     string                   `json:"semantic_identity_rule"`
	PrimaryLens              string                   `json:"primary_lens"`
	IdentityOutcomeRule      string                   `json:"identity_outcome_rule"`
	EquivalentDependencyRule string                   `json:"equivalent_dependency_rule"`
	CandidateComparisonRule  string                   `json:"candidate_comparison_rule"`
	LegacyDenominatorRule    string                   `json:"legacy_denominator_rule"`
	CorrectiveRetryRule      string                   `json:"corrective_retry_rule"`
	RepeatabilityRule        string                   `json:"repeatability_rule"`
	PrimaryGates             C1F3QualityGates         `json:"generalization_primary_gates"`
	Metrics                  []FrozenMetricDefinition `json:"metrics"`
}

func c1f3ScoringFreezeFingerprint(rubric C1F3ScoringFreeze) (string, error) {
	copy := rubric
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func LoadFrozenC1F3ScoringFreeze(profile C1F3EvaluationProfile, path string) (C1F3ScoringFreeze, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return C1F3ScoringFreeze{}, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.ScoringRubricSHA256 {
		return C1F3ScoringFreeze{}, fmt.Errorf("C1F3 scoring freeze file hash changed: got %s want %s", got, profile.ScoringRubricSHA256)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var rubric C1F3ScoringFreeze
	if err = decoder.Decode(&rubric); err != nil {
		return C1F3ScoringFreeze{}, err
	}
	if err = ensureEOF(decoder); err != nil {
		return C1F3ScoringFreeze{}, err
	}
	computed, err := c1f3ScoringFreezeFingerprint(rubric)
	if err != nil || rubric.Fingerprint != computed || rubric.Fingerprint != profile.ScoringRubricFingerprint || rubric.Version != C1F3ScoringRubricVersion {
		return C1F3ScoringFreeze{}, fmt.Errorf("C1F3 scoring freeze identity/fingerprint changed: got %s computed %s want %s", rubric.Fingerprint, computed, profile.ScoringRubricFingerprint)
	}
	if rubric.Implementation.Identity != C1FScoringVersion || rubric.Implementation.SHA256 != frozenC1FScoringSourceSHA256 || rubric.PrimaryLens != IssuerSemanticIdentityVersion || rubric.PrimaryGates != FrozenC1F3QualityGates() {
		return C1F3ScoringFreeze{}, fmt.Errorf("C1F3 scoring implementation, primary lens, or gates changed")
	}
	required := []string{"final_validity", "whole_mapping_correctness", "direct_precision", "direct_recall", "semantic_false_direct", "proxy_precision", "proxy_recall", "unresolved_correctness", "whole_typed_attribution", "attribution_completeness", "principal_correctness", "equal_principal_correctness", "secondary_affected_correctness", "context_only_correctness", "possible_principal_correctness", "principal_proxy_accuracy", "policy_false_positives", "policy_false_negatives", "incorrect_deterministic_ticker_resolutions", "identity_exact", "identity_equivalent", "identity_distinct", "identity_ambiguous", "semantic_successes_depending_on_equivalent", "first_pass_validity", "corrective_retry_count", "corrective_retry_rate", "retry_validation_reasons", "final_validity_after_retry", "semantic_change_after_retry", "safety_persistence_violations"}
	seen := map[string]bool{}
	for _, metric := range rubric.Metrics {
		if seen[metric.Identity] || strings.TrimSpace(metric.Numerator) == "" || strings.TrimSpace(metric.Denominator) == "" {
			return C1F3ScoringFreeze{}, fmt.Errorf("duplicate or incomplete C1F3 metric %q", metric.Identity)
		}
		seen[metric.Identity] = true
	}
	for _, metric := range required {
		if !seen[metric] {
			return C1F3ScoringFreeze{}, fmt.Errorf("missing C1F3 metric %s", metric)
		}
	}
	if len(seen) != len(required) || !strings.Contains(rubric.LegacyDenominatorRule, "not reused") || !strings.Contains(rubric.IdentityOutcomeRule, "EXACT + EQUIVALENT") || !strings.Contains(rubric.CorrectiveRetryRule, "not repeatability") {
		return C1F3ScoringFreeze{}, fmt.Errorf("C1F3 scoring rules changed")
	}
	return rubric, nil
}
