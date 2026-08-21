package aishadow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F3TerraChallengerProfileIdentity          = "openai-hosted-c1f3-challenger-generalization-v3-terra-t1"
	C1F3TerraChallengerExperimentID             = "WP-00.03T1"
	C1F3TerraChallengerEvidenceNamespace        = C1F3TerraChallengerProfileIdentity
	C1F3TerraChallengerRubricPath               = "config/ai-shadow-c1f3-terra-challenger-rubric-v1.json"
	C1F3TerraChallengerRubricFileSHA256         = "94ad6df14a58cc605d6498818c43dc585eaff84ae59f63528a12d5f105916ff2"
	C1F3TerraChallengerRubricFingerprint        = "abd02e30a62f93516aa90488ef9e2b28f1611b9d75dfe5b4bc974c1367024ee0"
	C1F3TerraChallengerProfileFingerprint       = "d58e9af61c2ac482a4641933ed422535c55400d862a571367693d42a7e20a915"
	C1F3TerraChallengerScoringSourceSHA256      = "de1d263c51e925df5b2c0f6016e0a817c8b82eac134256c22d8f10cc35915121"
	C1F3TerraAcceptedLunaRunID                  = "77bf44ba-2d3e-49bb-b7b1-a3006def8c5c"
	C1F3TerraAcceptedLunaArtifactIndexSHA256    = "ee6baeed27770d0f77581e49c6b99e3b9d1d974f90123272d6fad9d116f65439"
	C1F3TerraAcceptedLunaAnalysisSHA256         = "6acc2711056bc20f598784ceed98a69546db0fb73a1a38923f76aa7019b47a4f"
	C1F3TerraAcceptedLunaComparisonScoreSHA256  = "3788c31a5ada41c4a96227136a7dcb343c0a8285483a152f21942479423a80d0"
	C1F3TerraAcceptedLunaRelativeDirectory      = ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-repeatability-generalization-v3-r3/WP-00.03C1F3D-GENERALIZATION-R3/77bf44ba-2d3e-49bb-b7b1-a3006def8c5c"
	C1F3TerraAcceptedLunaAnalysisRelativePath   = ".runtime/c1f3-r3-analysis.json"
	C1F3TerraAcceptedLunaComparisonRelativePath = ".runtime/c1f3-repeatability-r3-score.json"
)

type C1F3TerraChallengerRubricBinding struct {
	Identity            string `json:"identity"`
	Path                string `json:"path"`
	FileSHA256          string `json:"file_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
}

type C1F3TerraChallengerProfile struct {
	Identity                   string                                 `json:"identity"`
	SourceProfileIdentity      string                                 `json:"source_profile_identity"`
	FrozenSemanticStack        C1F3EvaluationProfile                  `json:"frozen_semantic_stack"`
	AcceptedLunaBaseline       C1F3TerraChallengerBaseline            `json:"accepted_luna_baseline"`
	DecisionRubric             C1F3TerraChallengerRubricBinding       `json:"decision_rubric"`
	ComparisonScoring          C1F3RepeatabilityImplementationBinding `json:"comparison_scoring"`
	EvidenceScoring            C1F3RepeatabilityImplementationBinding `json:"evidence_scoring"`
	Provider                   string                                 `json:"provider"`
	Model                      string                                 `json:"model"`
	ReasoningEffort            string                                 `json:"reasoning_effort"`
	CaseCount                  int                                    `json:"case_count"`
	Repetitions                int                                    `json:"repetitions"`
	ExperimentIdentity         string                                 `json:"experiment_identity"`
	EvidenceNamespace          string                                 `json:"evidence_namespace"`
	MaximumBudgetMicros        int64                                  `json:"maximum_budget_micros"`
	ExpectedAnswerControlPlane bool                                   `json:"expected_answer_control_plane_only"`
	BoundaryExcluded           bool                                   `json:"boundary_excluded"`
	DefaultDeny                bool                                   `json:"default_deny"`
}

type C1F3TerraLunaPreservationBinding struct {
	RunID                  string `json:"run_id"`
	ArtifactIndexSHA256    string `json:"artifact_index_sha256"`
	IndexedArtifactCount   int    `json:"indexed_artifact_count"`
	RawResponseCount       int    `json:"raw_response_count"`
	SemanticAnalysisSHA256 string `json:"semantic_analysis_sha256"`
	ComparisonScoreSHA256  string `json:"comparison_score_sha256"`
	EvidenceUntouched      bool   `json:"evidence_untouched"`
}

type C1F3TerraChallengerFrozenBindingPlan struct {
	ProfileIdentity    string                                 `json:"profile_identity"`
	ProfileFingerprint string                                 `json:"profile_fingerprint"`
	SemanticStack      C1F3FrozenBindingPlan                  `json:"semantic_stack"`
	AcceptedLuna       C1F3TerraLunaPreservationBinding       `json:"accepted_luna"`
	DecisionRubric     C1F3TerraChallengerRubricBinding       `json:"decision_rubric"`
	ComparisonScoring  C1F3RepeatabilityImplementationBinding `json:"comparison_scoring"`
	EvidenceScoring    C1F3RepeatabilityImplementationBinding `json:"evidence_scoring"`
	CaseCount          int                                    `json:"case_count"`
	Repetitions        int                                    `json:"repetitions"`
	BoundaryExcluded   bool                                   `json:"boundary_excluded"`
}

func FrozenC1F3TerraChallengerProfile() (C1F3TerraChallengerProfile, error) {
	base, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		return C1F3TerraChallengerProfile{}, err
	}
	return C1F3TerraChallengerProfile{
		Identity: C1F3TerraChallengerProfileIdentity, SourceProfileIdentity: C1F3ProfileGeneralization,
		FrozenSemanticStack: base, AcceptedLunaBaseline: frozenC1F3TerraChallengerBaseline(),
		DecisionRubric: C1F3TerraChallengerRubricBinding{
			Identity: C1F3TerraChallengerRubricVersion, Path: C1F3TerraChallengerRubricPath,
			FileSHA256: C1F3TerraChallengerRubricFileSHA256, SemanticFingerprint: C1F3TerraChallengerRubricFingerprint,
		},
		ComparisonScoring: C1F3RepeatabilityImplementationBinding{Identity: C1F3RepeatabilityScoringVersion, SHA256: C1F3RepeatabilityComparisonSourceSHA256, SourcePath: "internal/modules/aishadow/repeatability_scoring.go"},
		EvidenceScoring:   C1F3RepeatabilityImplementationBinding{Identity: C1F3TerraChallengerRubricVersion, SHA256: C1F3TerraChallengerScoringSourceSHA256, SourcePath: "internal/modules/aishadow/terra_challenger_scoring.go"},
		Provider:          OpenAIDiagnosticProvider, Model: OpenAIDiagnosticTerraModel, ReasoningEffort: OpenAIDiagnosticReasoningEffort,
		CaseCount: 48, Repetitions: 1, ExperimentIdentity: C1F3TerraChallengerExperimentID,
		EvidenceNamespace: C1F3TerraChallengerEvidenceNamespace, MaximumBudgetMicros: 300_000,
		ExpectedAnswerControlPlane: true, BoundaryExcluded: true, DefaultDeny: true,
	}, nil
}

func (p C1F3TerraChallengerProfile) Fingerprint() (string, error) { return fingerprint(p) }

func loadC1F3TerraChallengerDiagnosticProfile(identity string) (DiagnosticEvaluationProfile, bool) {
	if identity != C1F3TerraChallengerProfileIdentity {
		return DiagnosticEvaluationProfile{}, false
	}
	base, ok := loadC1F3AuthorizedDiagnosticProfile(C1F3ProfileGeneralization)
	if !ok {
		return DiagnosticEvaluationProfile{}, false
	}
	base.Identity = C1F3TerraChallengerProfileIdentity
	base.RequiredModel = OpenAIDiagnosticTerraModel
	base.RequiredExperimentID = C1F3TerraChallengerExperimentID
	base.EvidenceNamespace = C1F3TerraChallengerEvidenceNamespace
	base.MaximumBudgetMicros = 300_000
	return base, true
}

func isC1F3TerraChallengerProfile(profile DiagnosticEvaluationProfile) bool {
	return profile.Identity == C1F3TerraChallengerProfileIdentity
}

func ValidateC1F3TerraAcceptedLunaEvidence(repositoryRoot string) (C1F3TerraLunaPreservationBinding, error) {
	runDirectory := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3TerraAcceptedLunaRelativeDirectory))
	index, hashes, err := verifyC1F3ArtifactIndex(runDirectory, C1F3TerraAcceptedLunaArtifactIndexSHA256)
	if err != nil {
		return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("verify accepted Luna r3 evidence: %w", err)
	}
	rawCount := 0
	for path := range hashes {
		if strings.HasPrefix(path, "repetition-01/") && strings.HasSuffix(path, ".json") {
			raw, readErr := os.ReadFile(filepath.Join(runDirectory, filepath.FromSlash(path)))
			if readErr != nil {
				return C1F3TerraLunaPreservationBinding{}, readErr
			}
			var audit DiagnosticAttemptAudit
			if err := json.Unmarshal(raw, &audit); err != nil || audit.RunID != C1F3TerraAcceptedLunaRunID || strings.TrimSpace(audit.RawResponseBody) == "" {
				return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("accepted Luna raw response %s is missing or invalid", path)
			}
			rawCount++
		}
	}
	if rawCount != 48 {
		return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("accepted Luna raw-response count changed: got %d want 48", rawCount)
	}
	var report DiagnosticRunReport
	reportRaw, err := os.ReadFile(filepath.Join(runDirectory, "report.json"))
	if err != nil || json.Unmarshal(reportRaw, &report) != nil || report.RunID != C1F3TerraAcceptedLunaRunID ||
		report.ModelIdentity.Name != OpenAIDiagnosticLunaModel || len(report.Repetitions) != 1 || report.Repetitions[0].Contract.FinalValidOutputs != 48 {
		return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("accepted Luna report binding changed")
	}
	analysisPath := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3TerraAcceptedLunaAnalysisRelativePath))
	analysisHash, err := hashOpaqueFile(analysisPath)
	if err != nil || analysisHash != C1F3TerraAcceptedLunaAnalysisSHA256 {
		return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("accepted Luna semantic scoring evidence changed")
	}
	comparisonPath := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3TerraAcceptedLunaComparisonRelativePath))
	comparisonHash, err := hashOpaqueFile(comparisonPath)
	if err != nil || comparisonHash != C1F3TerraAcceptedLunaComparisonScoreSHA256 {
		return C1F3TerraLunaPreservationBinding{}, fmt.Errorf("accepted Luna comparison scoring evidence changed")
	}
	return C1F3TerraLunaPreservationBinding{
		RunID: C1F3TerraAcceptedLunaRunID, ArtifactIndexSHA256: C1F3TerraAcceptedLunaArtifactIndexSHA256,
		IndexedArtifactCount: len(index.Artifacts), RawResponseCount: rawCount,
		SemanticAnalysisSHA256: analysisHash, ComparisonScoreSHA256: comparisonHash, EvidenceUntouched: true,
	}, nil
}

func loadC1F3TerraChallengerExecutionInputs(paths DiagnosticPaths, profile DiagnosticEvaluationProfile, resolver assetresolution.Resolver, exposures []string) (DiagnosticManifest, DiagnosticFingerprintLock, DiagnosticFreezeRecord, C1F3FrozenBindingPlan, C1F3TerraChallengerFrozenBindingPlan, error) {
	baseDiagnostic, ok := loadC1F3AuthorizedDiagnosticProfile(C1F3ProfileGeneralization)
	if !ok {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, fmt.Errorf("load frozen C1F3 Generalization profile")
	}
	manifest, lock, freeze, semantic, err := loadC1F3ExecutionInputs(paths, baseDiagnostic, resolver, exposures)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, err
	}
	frozen, err := FrozenC1F3TerraChallengerProfile()
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, err
	}
	profileFingerprint, err := frozen.Fingerprint()
	if err != nil || profile.Identity != frozen.Identity || profileFingerprint != C1F3TerraChallengerProfileFingerprint {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, fmt.Errorf("frozen Terra challenger profile identity changed")
	}
	root := c1e3RepositoryRoot(paths.AssetRulesetPath)
	if _, err := LoadFrozenC1F3TerraChallengerRubric(filepath.Join(root, filepath.FromSlash(C1F3TerraChallengerRubricPath)), C1F3TerraChallengerRubricFileSHA256, C1F3TerraChallengerRubricFingerprint); err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, err
	}
	if got, err := hashOpaqueFile(filepath.Join(root, filepath.FromSlash(frozen.ComparisonScoring.SourcePath))); err != nil || got != frozen.ComparisonScoring.SHA256 {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, fmt.Errorf("frozen Terra comparison scorer hash changed")
	}
	if got, err := hashOpaqueFile(filepath.Join(root, filepath.FromSlash(frozen.EvidenceScoring.SourcePath))); err != nil || got != frozen.EvidenceScoring.SHA256 {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, fmt.Errorf("frozen Terra evidence scorer hash changed")
	}
	luna, err := ValidateC1F3TerraAcceptedLunaEvidence(root)
	if err != nil {
		return DiagnosticManifest{}, DiagnosticFingerprintLock{}, DiagnosticFreezeRecord{}, C1F3FrozenBindingPlan{}, C1F3TerraChallengerFrozenBindingPlan{}, err
	}
	semantic.Model = OpenAIDiagnosticTerraModel
	semantic.ExperimentIdentity = C1F3TerraChallengerExperimentID
	binding := C1F3TerraChallengerFrozenBindingPlan{
		ProfileIdentity: frozen.Identity, ProfileFingerprint: profileFingerprint, SemanticStack: semantic,
		AcceptedLuna: luna, DecisionRubric: frozen.DecisionRubric, ComparisonScoring: frozen.ComparisonScoring, EvidenceScoring: frozen.EvidenceScoring,
		CaseCount: 48, Repetitions: 1, BoundaryExcluded: true,
	}
	return manifest, lock, freeze, semantic, binding, nil
}
