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
	CausalAttributionLabelRubricVersion = "ai-shadow-causal-attribution-label-rubric-v1"
	C1E2AScoringRubricVersion           = "ai-shadow-causal-attribution-scoring-c1e3-v1"
	GeneralizationV2TypedLabelsVersion  = "ai-shadow-causal-attribution-labels-generalization-v2-v1"
	BoundaryV2TypedLabelsVersion        = "ai-shadow-causal-attribution-labels-boundary-v2-v1"

	frozenV5PromptSHA256  = "60fa5a4c5e1156e8715a779d506c58af5e9dbf4474bff0c25d6fbe46ecd15eff"
	frozenV5SchemaSHA256  = "8dc2a787bd7a33ec768a570d0d3588243561d85b06051d9e01bf2435fd88f960"
	frozenC1EPolicySHA256 = "e319b3ceca80e9c2c43ab2edabb60402d8c9e534645b0dcb0e81baa953349c2e"
)

type TypedLabelSourceLink struct {
	Identity            string `json:"identity"`
	Path                string `json:"path"`
	FileSHA256          string `json:"file_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
}

type TypedLabelContractLink struct {
	Identity   string `json:"identity"`
	SHA256     string `json:"sha256"`
	SourcePath string `json:"source_path,omitempty"`
}

type TypedLabelAdjudicationQC struct {
	Passes                     int      `json:"passes"`
	FirstPassClean             int      `json:"first_pass_clean"`
	RequiredReconsideration    int      `json:"required_reconsideration"`
	RemainingAmbiguous         int      `json:"remaining_ambiguous"`
	ReconsideredCases          []string `json:"reconsidered_cases"`
	ReconsiderationResolutions []string `json:"reconsideration_resolutions"`
}

type TypedExpectedCase struct {
	DatasetIdentity                       string              `json:"dataset_identity"`
	DatasetVersion                        string              `json:"dataset_version"`
	CaseID                                string              `json:"case_id"`
	ExpectedMappingStatus                 string              `json:"expected_mapping_status"`
	ExpectedDirectIssuer                  string              `json:"expected_direct_issuer"`
	ExpectedProxyExposure                 string              `json:"expected_proxy_exposure"`
	ExpectedIssuerAttributions            []IssuerAttribution `json:"expected_issuer_attributions"`
	ExpectedPrincipalProxyCandidates      []string            `json:"expected_principal_proxy_candidates"`
	ExpectedDeterministicResolutionStatus string              `json:"expected_deterministic_resolution_status"`
	TypedAttributionRationale             string              `json:"typed_attribution_rationale"`
	AdjudicationStatus                    string              `json:"adjudication_status"`
}

type TypedLabelSidecar struct {
	Version                     string                   `json:"version"`
	DatasetIdentity             string                   `json:"dataset_identity"`
	DatasetVersion              string                   `json:"dataset_version"`
	CreatedAt                   time.Time                `json:"created_at"`
	Fingerprint                 string                   `json:"fingerprint"`
	CaseCount                   int                      `json:"case_count"`
	SourceDataset               TypedLabelSourceLink     `json:"source_dataset"`
	SourceInputLock             TypedLabelSourceLink     `json:"source_input_lock"`
	Prompt                      TypedLabelContractLink   `json:"prompt"`
	OutputContract              TypedLabelContractLink   `json:"output_contract"`
	AttributionPolicy           TypedLabelContractLink   `json:"attribution_policy"`
	Resolver                    TypedLabelContractLink   `json:"resolver"`
	AdjudicationRubric          string                   `json:"adjudication_rubric"`
	ScoringRubric               string                   `json:"scoring_rubric"`
	AdjudicationMethod          string                   `json:"adjudication_method"`
	AttributionCompletenessRule string                   `json:"attribution_completeness_rule"`
	HoldoutDisposition          string                   `json:"holdout_disposition"`
	VersionOnObservationRule    string                   `json:"version_on_observation_rule"`
	QualityControl              TypedLabelAdjudicationQC `json:"quality_control"`
	Cases                       []TypedExpectedCase      `json:"cases"`
}

type FrozenMetricDefinition struct {
	Identity    string `json:"identity"`
	Scope       string `json:"scope"`
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}

type CausalAttributionScoringRubric struct {
	Version                 string                   `json:"version"`
	CreatedAt               time.Time                `json:"created_at"`
	Fingerprint             string                   `json:"fingerprint"`
	EvaluationScope         string                   `json:"evaluation_scope"`
	InvalidOutputRule       string                   `json:"invalid_output_rule"`
	IssuerComparisonRule    string                   `json:"issuer_comparison_rule"`
	CandidateComparisonRule string                   `json:"candidate_comparison_rule"`
	LegacyDenominatorRule   string                   `json:"legacy_denominator_rule"`
	Metrics                 []FrozenMetricDefinition `json:"metrics"`
}

func loadStrictTypedLabels(raw []byte) (TypedLabelSidecar, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var sidecar TypedLabelSidecar
	if err := decoder.Decode(&sidecar); err != nil {
		return TypedLabelSidecar{}, fmt.Errorf("decode typed-attribution label sidecar: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return TypedLabelSidecar{}, fmt.Errorf("decode typed-attribution label sidecar: %w", err)
	}
	return sidecar, nil
}

func typedLabelSidecarFingerprint(sidecar TypedLabelSidecar) (string, error) {
	copy := sidecar
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func LoadFrozenTypedLabelSidecarForProfile(profile DiagnosticEvaluationProfile, path string) (TypedLabelSidecar, error) {
	if !profile.RequiresTypedAttributionLabels || profile.TypedLabelVersion == "" {
		return TypedLabelSidecar{}, fmt.Errorf("issuer diagnostic profile %s does not accept typed-attribution labels", profile.Identity)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return TypedLabelSidecar{}, fmt.Errorf("read frozen typed-attribution label sidecar: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.TypedLabelFileSHA256 {
		return TypedLabelSidecar{}, fmt.Errorf("frozen typed-attribution label sidecar file hash changed for profile %s: got %s want %s", profile.Identity, got, profile.TypedLabelFileSHA256)
	}
	sidecar, err := loadStrictTypedLabels(raw)
	if err != nil {
		return TypedLabelSidecar{}, err
	}
	wantFingerprint, err := typedLabelSidecarFingerprint(sidecar)
	if err != nil || sidecar.Fingerprint != wantFingerprint || sidecar.Fingerprint != profile.TypedLabelFingerprint {
		return TypedLabelSidecar{}, fmt.Errorf("frozen typed-attribution label sidecar fingerprint changed for profile %s: got %s computed %s want %s", profile.Identity, sidecar.Fingerprint, wantFingerprint, profile.TypedLabelFingerprint)
	}
	if err := validateTypedLabelMetadata(profile, sidecar); err != nil {
		return TypedLabelSidecar{}, err
	}
	return sidecar, nil
}

func validateTypedLabelMetadata(profile DiagnosticEvaluationProfile, sidecar TypedLabelSidecar) error {
	if sidecar.Version != profile.TypedLabelVersion || sidecar.DatasetIdentity != profile.Identity || sidecar.DatasetVersion != profile.ManifestVersion ||
		sidecar.CaseCount != profile.CaseCount || len(sidecar.Cases) != profile.CaseCount || sidecar.AdjudicationRubric != CausalAttributionLabelRubricVersion ||
		sidecar.ScoringRubric != profile.ScoringRubricVersion || sidecar.QualityControl.Passes != 2 || sidecar.QualityControl.RemainingAmbiguous != 0 ||
		sidecar.QualityControl.FirstPassClean+sidecar.QualityControl.RequiredReconsideration != profile.CaseCount {
		return fmt.Errorf("typed-attribution label sidecar has incompatible identity, count, rubric, or review status")
	}
	if sidecar.SourceDataset.Identity != profile.ManifestVersion || sidecar.SourceDataset.Path != profile.ManifestPath ||
		sidecar.SourceDataset.FileSHA256 != profile.ManifestFileSHA256 || sidecar.SourceDataset.SemanticFingerprint != profile.ManifestFingerprint ||
		sidecar.SourceInputLock.Identity != profile.FingerprintLockVersion || sidecar.SourceInputLock.Path != profile.FingerprintLockPath ||
		sidecar.SourceInputLock.FileSHA256 != profile.FingerprintLockFileSHA256 || sidecar.SourceInputLock.SemanticFingerprint != profile.FingerprintLockFingerprint {
		return fmt.Errorf("typed-attribution label sidecar source linkage changed")
	}
	if sidecar.Prompt.Identity != V5PromptVersion || sidecar.Prompt.SHA256 != frozenV5PromptSHA256 ||
		sidecar.OutputContract.Identity != V5SchemaVersion || sidecar.OutputContract.SHA256 != frozenV5SchemaSHA256 ||
		sidecar.AttributionPolicy.Identity != CausalAttributionPolicyVersion || sidecar.AttributionPolicy.SHA256 != frozenC1EPolicySHA256 ||
		sidecar.Resolver.Identity != "event-asset-resolution-v1" || sidecar.Resolver.SHA256 != expectedAssetRulesetFileSHA256 {
		return fmt.Errorf("typed-attribution label sidecar frozen contract linkage changed")
	}
	if strings.TrimSpace(sidecar.AdjudicationMethod) == "" || strings.TrimSpace(sidecar.AttributionCompletenessRule) == "" ||
		!strings.Contains(sidecar.HoldoutDisposition, "CASE-BLIND / CATEGORY-AWARE") || strings.TrimSpace(sidecar.VersionOnObservationRule) == "" {
		return fmt.Errorf("typed-attribution label sidecar lacks required adjudication or observation policy")
	}
	return nil
}

func ValidateTypedLabelSidecarAgainstSource(profile DiagnosticEvaluationProfile, sidecar TypedLabelSidecar, manifest DiagnosticManifest, resolver assetresolution.Resolver) error {
	if manifest.Version != profile.ManifestVersion || len(manifest.Events) != len(sidecar.Cases) {
		return fmt.Errorf("typed-attribution label sidecar source dataset mismatch")
	}
	seen := map[string]bool{}
	for index, expected := range sidecar.Cases {
		source := manifest.Events[index]
		if expected.CaseID != source.ID || seen[expected.CaseID] || expected.DatasetIdentity != profile.Identity || expected.DatasetVersion != profile.ManifestVersion {
			return fmt.Errorf("typed-attribution label case identity/order mismatch at position %d", index+1)
		}
		seen[expected.CaseID] = true
		if expected.ExpectedMappingStatus != source.Label.MappingStatus || expected.ExpectedDirectIssuer != source.Label.DirectIssuer ||
			expected.ExpectedProxyExposure != source.Label.ProxyExposure || expected.ExpectedDeterministicResolutionStatus != source.Label.ExpectedResolutionStatus {
			return fmt.Errorf("typed-attribution label case %s contradicts frozen v2 mapping", expected.CaseID)
		}
		if expected.AdjudicationStatus != "FROZEN" || strings.TrimSpace(expected.TypedAttributionRationale) == "" ||
			expected.ExpectedIssuerAttributions == nil || expected.ExpectedPrincipalProxyCandidates == nil {
			return fmt.Errorf("typed-attribution label case %s is incomplete or not frozen", expected.CaseID)
		}
		projection := V5StructuredResult{
			MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure,
			IssuerAttributions: expected.ExpectedIssuerAttributions, PrincipalProxyCandidates: expected.ExpectedPrincipalProxyCandidates,
		}
		if errors := validateV5Attribution(projection, resolver); len(errors) > 0 {
			return fmt.Errorf("typed-attribution label case %s violates v5 invariants: %s", expected.CaseID, strings.Join(errors, "; "))
		}
		decision, err := ApplyCausalAttributionPolicy(projection)
		if err != nil {
			return fmt.Errorf("typed-attribution label case %s policy projection: %w", expected.CaseID, err)
		}
		resolution := ResolveCausalAttributionDecision(decision, source.Input, resolver)
		if resolution.Status != expected.ExpectedDeterministicResolutionStatus {
			return fmt.Errorf("typed-attribution label case %s deterministic resolution changed: got %s want %s", expected.CaseID, resolution.Status, expected.ExpectedDeterministicResolutionStatus)
		}
	}
	return nil
}

func scoringRubricFingerprint(rubric CausalAttributionScoringRubric) (string, error) {
	copy := rubric
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func LoadFrozenC1E3ScoringRubric(profile DiagnosticEvaluationProfile, path string) (CausalAttributionScoringRubric, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CausalAttributionScoringRubric{}, fmt.Errorf("read frozen C1E3 scoring rubric: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.ScoringRubricFileSHA256 {
		return CausalAttributionScoringRubric{}, fmt.Errorf("frozen C1E3 scoring rubric file hash changed: got %s want %s", got, profile.ScoringRubricFileSHA256)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var rubric CausalAttributionScoringRubric
	if err := decoder.Decode(&rubric); err != nil {
		return CausalAttributionScoringRubric{}, fmt.Errorf("decode frozen C1E3 scoring rubric: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return CausalAttributionScoringRubric{}, err
	}
	wantFingerprint, err := scoringRubricFingerprint(rubric)
	if err != nil || rubric.Fingerprint != wantFingerprint || rubric.Fingerprint != profile.ScoringRubricFingerprint || rubric.Version != profile.ScoringRubricVersion {
		return CausalAttributionScoringRubric{}, fmt.Errorf("frozen C1E3 scoring rubric identity or fingerprint changed: got %s computed %s want %s", rubric.Fingerprint, wantFingerprint, profile.ScoringRubricFingerprint)
	}
	if err := validateScoringDenominators(rubric); err != nil {
		return CausalAttributionScoringRubric{}, err
	}
	return rubric, nil
}

func validateScoringDenominators(rubric CausalAttributionScoringRubric) error {
	required := []string{
		"direct_recall", "direct_precision", "proxy_recall", "proxy_precision", "unresolved_correctness", "false_direct", "false_proxy",
		"whole_case_attribution_correctness", "principal_correctness", "equal_principal_correctness", "secondary_affected_correctness",
		"context_only_correctness", "possible_principal_correctness", "principal_proxy_candidate_correctness", "attribution_completeness",
		"typed_policy_acceptance_correctness", "typed_policy_abstention_correctness", "policy_induced_false_negatives", "policy_induced_false_positives",
		"deterministic_resolution_correctness", "ambiguous_unresolved_resolver_behavior", "possible_principal_frequency", "possible_principal_correct_uses",
		"possible_principal_incorrect_uses", "possible_principal_abstentions_caused", "possible_principal_false_negatives_caused",
	}
	seen := map[string]bool{}
	for _, metric := range rubric.Metrics {
		if seen[metric.Identity] || strings.TrimSpace(metric.Numerator) == "" || strings.TrimSpace(metric.Denominator) == "" {
			return fmt.Errorf("frozen C1E3 scoring rubric has duplicate or incomplete metric %q", metric.Identity)
		}
		seen[metric.Identity] = true
	}
	for _, identity := range required {
		if !seen[identity] {
			return fmt.Errorf("frozen C1E3 scoring rubric is missing metric %s", identity)
		}
	}
	if len(seen) != len(required) || !strings.Contains(rubric.LegacyDenominatorRule, "not reused") {
		return fmt.Errorf("frozen C1E3 scoring rubric contains an incompatible metric set or legacy denominator rule")
	}
	return nil
}
