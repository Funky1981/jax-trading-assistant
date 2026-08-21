package aishadow

import (
	"fmt"
	"path/filepath"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F3RepeatabilityProfileIdentity           = "openai-hosted-c1f3-repeatability-generalization-v3-r2"
	C1F3RepeatabilityExperimentID              = "WP-00.03C1F3B-GENERALIZATION-R2"
	C1F3RepeatabilityEvidenceNamespace         = C1F3RepeatabilityProfileIdentity
	C1F3RepeatabilityScoringPath               = "config/ai-shadow-c1f3-repeatability-scoring-v1.json"
	C1F3RepeatabilityScoringFileSHA256         = "ff6fe9618ad8e89440e4d015724a9949c95f0542e6b0e0048bcb5d73f7e4c45d"
	C1F3RepeatabilityScoringFingerprint        = "70313e7f77f5f82b912c382d96aa513f8e67dc03f53abe09c26189494677d1f5"
	C1F3RepeatabilityProfileFingerprint        = "294433030c9e24ff695b0a6792e7039555de21cdcfce8cc86805a3c86d45964d"
	C1F3RepeatabilityComparisonSourceSHA256    = "7b5e36b96bda8456b20841c59ec759ddf8f31472b288668b6880c4e7deb73114"
	C1F3RepeatabilityBaselineRelativeDirectory = ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-generalization-v3/WP-00.03C1F3-GENERALIZATION/0a650e09-1c64-4349-bf5d-09bf4dd697d9"
	C1F3AReprojectionRelativeDirectory         = ".runtime/diagnostics/ai-shadow-c1f3-reprojection-v1/WP-00.03C1F3A/8f860c0a8abcdc755b737b59c6feeb17af796a00c97afc900443a663563eba05"
	C1F3AReprojectionJSONSHA256                = "3d1990d1a0da5b6c2890fad2231b06394a5b0a3f55e43a545a4f93155506db70"
	C1F3AReprojectionArtifactIndexSHA256       = "73396d5f1bd77106d256d2d2639f6e6e8006521dd0070a5eee36c0a082d5b9c1"
	C1F3AAcceptedReprojectionFingerprint       = "8f860c0a8abcdc755b737b59c6feeb17af796a00c97afc900443a663563eba05"
)

type C1F3RepeatabilityScoringBinding struct {
	Identity                 string                                 `json:"identity"`
	Path                     string                                 `json:"path"`
	FileSHA256               string                                 `json:"file_sha256"`
	SemanticFingerprint      string                                 `json:"semantic_fingerprint"`
	Implementation           C1F3RepeatabilityImplementationBinding `json:"implementation"`
	ComparisonImplementation C1F3RepeatabilityImplementationBinding `json:"comparison_implementation"`
}

type C1F3RepeatabilityProfile struct {
	Identity                   string                          `json:"identity"`
	SourceProfileIdentity      string                          `json:"source_profile_identity"`
	FrozenSemanticStack        C1F3EvaluationProfile           `json:"frozen_semantic_stack"`
	Baseline                   C1F3RepeatabilityBaseline       `json:"baseline"`
	ComparisonScoring          C1F3RepeatabilityScoringBinding `json:"comparison_scoring"`
	Provider                   string                          `json:"provider"`
	Model                      string                          `json:"model"`
	ReasoningEffort            string                          `json:"reasoning_effort"`
	CaseCount                  int                             `json:"case_count"`
	Repetitions                int                             `json:"repetitions"`
	ExperimentIdentity         string                          `json:"experiment_identity"`
	EvidenceNamespace          string                          `json:"evidence_namespace"`
	MaximumBudgetMicros        int64                           `json:"maximum_budget_micros"`
	ExpectedAnswerControlPlane bool                            `json:"expected_answer_control_plane_only"`
	DefaultDeny                bool                            `json:"default_deny"`
}

type C1F3AcceptedEvidenceBindings struct {
	GeneralizationRunID         string `json:"generalization_run_id"`
	GeneralizationArtifactIndex string `json:"generalization_artifact_index_sha256"`
	BoundaryRunID               string `json:"boundary_run_id"`
	BoundaryArtifactIndex       string `json:"boundary_artifact_index_sha256"`
	C1F3AIdentity               string `json:"c1f3a_identity"`
	C1F3AFingerprint            string `json:"c1f3a_fingerprint"`
	C1F3AJSONSHA256             string `json:"c1f3a_json_sha256"`
	C1F3AArtifactIndexSHA256    string `json:"c1f3a_artifact_index_sha256"`
}

type C1F3RepeatabilityFrozenBindingPlan struct {
	ProfileIdentity    string                          `json:"profile_identity"`
	ProfileFingerprint string                          `json:"profile_fingerprint"`
	SemanticStack      C1F3FrozenBindingPlan           `json:"semantic_stack"`
	Baseline           C1F3RepeatabilityBaseline       `json:"baseline"`
	ComparisonScoring  C1F3RepeatabilityScoringBinding `json:"comparison_scoring"`
	AcceptedEvidence   C1F3AcceptedEvidenceBindings    `json:"accepted_evidence"`
	CaseCount          int                             `json:"case_count"`
	Repetitions        int                             `json:"repetitions"`
}

func frozenC1F3RepeatabilityBaseline() C1F3RepeatabilityBaseline {
	return C1F3RepeatabilityBaseline{Profile: C1F3ProfileGeneralization, RunID: C1F3GeneralizationSourceRunID, ArtifactIndexSHA256: C1F3GeneralizationArtifactIndex}
}

func FrozenC1F3RepeatabilityProfile() (C1F3RepeatabilityProfile, error) {
	base, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		return C1F3RepeatabilityProfile{}, err
	}
	return C1F3RepeatabilityProfile{
		Identity: C1F3RepeatabilityProfileIdentity, SourceProfileIdentity: C1F3ProfileGeneralization, FrozenSemanticStack: base,
		Baseline: frozenC1F3RepeatabilityBaseline(), ComparisonScoring: C1F3RepeatabilityScoringBinding{
			Identity: C1F3RepeatabilityScoringVersion, Path: C1F3RepeatabilityScoringPath, FileSHA256: C1F3RepeatabilityScoringFileSHA256,
			SemanticFingerprint:      C1F3RepeatabilityScoringFingerprint,
			Implementation:           C1F3RepeatabilityImplementationBinding{Identity: C1FScoringVersion, SHA256: frozenC1FScoringSourceSHA256, SourcePath: "internal/modules/aishadow/scoring_c1f.go"},
			ComparisonImplementation: C1F3RepeatabilityImplementationBinding{Identity: C1F3RepeatabilityScoringVersion, SHA256: C1F3RepeatabilityComparisonSourceSHA256, SourcePath: "internal/modules/aishadow/repeatability_scoring.go"},
		},
		Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel, ReasoningEffort: OpenAIDiagnosticReasoningEffort,
		CaseCount: 48, Repetitions: 1, ExperimentIdentity: C1F3RepeatabilityExperimentID, EvidenceNamespace: C1F3RepeatabilityEvidenceNamespace,
		MaximumBudgetMicros: 300_000, ExpectedAnswerControlPlane: true, DefaultDeny: true,
	}, nil
}

func (p C1F3RepeatabilityProfile) Fingerprint() (string, error) { return fingerprint(p) }

func loadC1F3RepeatabilityDiagnosticProfile(identity string) (DiagnosticEvaluationProfile, bool) {
	if identity != C1F3RepeatabilityProfileIdentity {
		return DiagnosticEvaluationProfile{}, false
	}
	base, ok := loadC1F3AuthorizedDiagnosticProfile(C1F3ProfileGeneralization)
	if !ok {
		return DiagnosticEvaluationProfile{}, false
	}
	base.Identity = C1F3RepeatabilityProfileIdentity
	base.RequiredExperimentID = C1F3RepeatabilityExperimentID
	base.EvidenceNamespace = C1F3RepeatabilityEvidenceNamespace
	base.MaximumBudgetMicros = 300_000
	return base, true
}

func isC1F3RepeatabilityProfile(profile DiagnosticEvaluationProfile) bool {
	return profile.Identity == C1F3RepeatabilityProfileIdentity
}

func usesC1F3SemanticStack(profile DiagnosticEvaluationProfile) bool {
	return isC1F3Profile(profile) || isC1F3RepeatabilityProfile(profile)
}

func ValidateC1F3AcceptedEvidenceBindings(repositoryRoot string) (C1F3AcceptedEvidenceBindings, error) {
	generalization := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3RepeatabilityBaselineRelativeDirectory))
	if _, _, err := verifyC1F3ArtifactIndex(generalization, C1F3GeneralizationArtifactIndex); err != nil {
		return C1F3AcceptedEvidenceBindings{}, fmt.Errorf("verify accepted C1F3 Generalization baseline: %w", err)
	}
	boundary := filepath.Join(repositoryRoot, filepath.FromSlash(".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-boundary-v3/WP-00.03C1F3-BOUNDARY/"+C1F3BoundarySourceRunID))
	if _, _, err := verifyC1F3ArtifactIndex(boundary, C1F3BoundaryArtifactIndex); err != nil {
		return C1F3AcceptedEvidenceBindings{}, fmt.Errorf("verify accepted C1F3 Boundary evidence: %w", err)
	}
	reprojection := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3AReprojectionRelativeDirectory))
	if got, err := hashOpaqueFile(filepath.Join(reprojection, "reprojection.json")); err != nil || got != C1F3AReprojectionJSONSHA256 {
		return C1F3AcceptedEvidenceBindings{}, fmt.Errorf("accepted C1F3A reprojection JSON hash changed")
	}
	if got, err := hashOpaqueFile(filepath.Join(reprojection, "artifact-index.json")); err != nil || got != C1F3AReprojectionArtifactIndexSHA256 {
		return C1F3AcceptedEvidenceBindings{}, fmt.Errorf("accepted C1F3A artifact-index hash changed")
	}
	return C1F3AcceptedEvidenceBindings{
		GeneralizationRunID: C1F3GeneralizationSourceRunID, GeneralizationArtifactIndex: C1F3GeneralizationArtifactIndex,
		BoundaryRunID: C1F3BoundarySourceRunID, BoundaryArtifactIndex: C1F3BoundaryArtifactIndex,
		C1F3AIdentity: C1F3ReprojectionVersion, C1F3AFingerprint: C1F3AAcceptedReprojectionFingerprint,
		C1F3AJSONSHA256: C1F3AReprojectionJSONSHA256, C1F3AArtifactIndexSHA256: C1F3AReprojectionArtifactIndexSHA256,
	}, nil
}

func loadC1F3RepeatabilityExecutionInputs(paths DiagnosticPaths, profile DiagnosticEvaluationProfile, resolver assetresolution.Resolver, exposures []string) (DiagnosticManifest, DiagnosticFingerprintLock, DiagnosticFreezeRecord, C1F3FrozenBindingPlan, C1F3RepeatabilityFrozenBindingPlan, error) {
	baseDiagnostic, ok := loadC1F3AuthorizedDiagnosticProfile(C1F3ProfileGeneralization)
	if !ok {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, fmt.Errorf("load frozen C1F3 Generalization profile")
	}
	manifest, lock, freeze, semanticBindings, err := loadC1F3ExecutionInputs(paths, baseDiagnostic, resolver, exposures)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, err
	}
	repeatProfile, err := FrozenC1F3RepeatabilityProfile()
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, err
	}
	profileFingerprint, err := repeatProfile.Fingerprint()
	if err != nil || profileFingerprint != C1F3RepeatabilityProfileFingerprint || profile.Identity != repeatProfile.Identity {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, fmt.Errorf("frozen C1F3 repeatability profile identity changed")
	}
	root := c1e3RepositoryRoot(paths.AssetRulesetPath)
	if _, err := LoadFrozenC1F3RepeatabilityScoring(filepath.Join(root, filepath.FromSlash(C1F3RepeatabilityScoringPath)), C1F3RepeatabilityScoringFileSHA256, C1F3RepeatabilityScoringFingerprint); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, err
	}
	evidence, err := ValidateC1F3AcceptedEvidenceBindings(root)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3RepeatabilityFrozenBindingPlan{}, err
	}
	binding := C1F3RepeatabilityFrozenBindingPlan{
		ProfileIdentity: repeatProfile.Identity, ProfileFingerprint: profileFingerprint, SemanticStack: semanticBindings,
		Baseline: repeatProfile.Baseline, ComparisonScoring: repeatProfile.ComparisonScoring, AcceptedEvidence: evidence,
		CaseCount: repeatProfile.CaseCount, Repetitions: repeatProfile.Repetitions,
	}
	return manifest, lock, freeze, semanticBindings, binding, nil
}
