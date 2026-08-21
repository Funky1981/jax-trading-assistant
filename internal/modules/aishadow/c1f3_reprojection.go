package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F3ReprojectionVersion         = "ai-shadow-c1f3-reprojection-v1"
	C1F3ReprojectionWorkPackage     = "WP-00.03C1F3A"
	C1F3GeneralizationSourceRunID   = "0a650e09-1c64-4349-bf5d-09bf4dd697d9"
	C1F3BoundarySourceRunID         = "9da5becc-41de-4946-8703-b1c9b620382e"
	C1F3GeneralizationArtifactIndex = "681db26c0d3e83e85537671e519db55c2b77397b03feb0c8a9395dd87b0ea0cb"
	C1F3BoundaryArtifactIndex       = "d551d69ced918432fa5e3fa0fa78e99c6009c8bf3030358a1cf8a8adbca0260d"
	c1f3ArtifactIndexVersion        = "ai-shadow-issuer-diagnostic-artifact-index-v1"
	c1f3FrozenInputPriceUSD         = "0.20"
	c1f3FrozenCachedInputPriceUSD   = "0.02"
	c1f3FrozenCacheWritePriceUSD    = "0.25"
	c1f3FrozenOutputPriceUSD        = "1.20"
)

type C1F3ReprojectionSourceSpec struct {
	ProfileID           string `json:"profile_id"`
	SourceNamespace     string `json:"source_namespace"`
	RunID               string `json:"run_id"`
	ArtifactIndexSHA256 string `json:"artifact_index_sha256"`
}

type C1F3ReprojectionRequest struct {
	RepositoryRoot string
	OutputRoot     string
	Sources        []C1F3ReprojectionSourceSpec
}

type C1F3ComponentBinding struct {
	Identity string `json:"identity"`
	SHA256   string `json:"sha256"`
}

type C1F3ReprojectionComponents struct {
	Prompt         C1F3ComponentBinding `json:"prompt"`
	OutputContract C1F3ComponentBinding `json:"output_contract"`
	Parser         C1F3ComponentBinding `json:"parser"`
	Validator      C1F3ComponentBinding `json:"validator"`
	Policy         C1F3ComponentBinding `json:"policy"`
	Resolver       C1F3ComponentBinding `json:"resolver"`
	Comparator     C1F3ComponentBinding `json:"comparator"`
	Scorer         C1F3ComponentBinding `json:"scorer"`
}

type C1F3Rate struct {
	Numerator   int     `json:"numerator"`
	Denominator int     `json:"denominator"`
	Percentage  float64 `json:"percentage"`
}

type C1F3ReprojectedAttempt struct {
	SourceArtifactPath   string                     `json:"source_artifact_path"`
	SourceArtifactSHA256 string                     `json:"source_artifact_sha256"`
	SourceRawSHA256      string                     `json:"source_raw_response_sha256"`
	CaseID               string                     `json:"case_id"`
	AttemptNumber        int                        `json:"attempt_number"`
	RequestID            string                     `json:"request_id"`
	ResponseID           string                     `json:"response_id"`
	OriginallyComplete   bool                       `json:"original_projection_complete"`
	HistoricalV5Errors   []string                   `json:"historical_v5_projection_errors"`
	RawModelOutput       *V5StructuredResult        `json:"raw_model_output"`
	TypedAttribution     *TypedCausalAttribution    `json:"typed_attribution"`
	PolicyDecision       *CausalAttributionDecision `json:"policy_decision"`
	EffectiveMapping     *AssetMapping              `json:"effective_mapping"`
	Resolution           *PolicyResolution          `json:"deterministic_resolution"`
}

type C1F3ProjectionRecovery struct {
	AcceptedRecords     int                      `json:"accepted_records"`
	OriginallyComplete  int                      `json:"originally_complete"`
	RecoveredRecords    int                      `json:"recovered_records"`
	RecoveryErrors      map[string][]string      `json:"recovery_errors"`
	HistoricalV5Errors  map[string]int           `json:"historical_v5_projection_error_histogram"`
	ReprojectedAttempts []C1F3ReprojectedAttempt `json:"reprojected_attempts"`
}

type C1F3RetryAccounting struct {
	InitialRequests           int            `json:"initial_requests"`
	FirstPassValidity         C1F3Rate       `json:"first_pass_validity"`
	CorrectiveRetries         int            `json:"corrective_retries"`
	CorrectiveRetryRate       C1F3Rate       `json:"corrective_retry_rate"`
	ValidationReasonHistogram map[string]int `json:"validation_reason_histogram"`
	FinalValidity             C1F3Rate       `json:"final_validity"`
	SemanticChanges           int            `json:"semantic_changes"`
	SemanticComparisons       int            `json:"semantic_comparisons"`
	SemanticIndeterminate     int            `json:"semantic_indeterminate"`
}

type C1F3CostAccounting struct {
	Usage                   ProviderUsage       `json:"usage"`
	InitialRequests         int                 `json:"initial_requests"`
	CorrectiveRetries       int                 `json:"corrective_retries"`
	TotalProviderRequests   int                 `json:"total_provider_requests"`
	Pricing                 HostedPricingPlan   `json:"pricing"`
	CostByCategory          HostedCostBreakdown `json:"cost_by_category"`
	LocallyAccountedCostUSD string              `json:"locally_accounted_cost_usd"`
	BudgetCeilingUSD        string              `json:"budget_ceiling_usd"`
	ProviderErrors          int                 `json:"provider_errors"`
	Timeouts                int                 `json:"timeouts"`
	BudgetRejections        int                 `json:"budget_rejections"`
}

type C1F3IdentityAnalysis struct {
	Exact                          int      `json:"exact"`
	Equivalent                     int      `json:"equivalent"`
	Distinct                       int      `json:"distinct"`
	Ambiguous                      int      `json:"ambiguous"`
	TotalComparisons               int      `json:"total_comparisons"`
	MappingEquivalentDependent     C1F3Rate `json:"mapping_successes_depending_on_equivalent"`
	AttributionEquivalentDependent C1F3Rate `json:"attribution_successes_depending_on_equivalent"`
}

type C1F3PolicyAnalysis struct {
	Accepted                   int      `json:"accepted"`
	Abstained                  int      `json:"abstained"`
	AcceptanceCorrectness      C1F3Rate `json:"acceptance_correctness"`
	AbstentionCorrectness      C1F3Rate `json:"abstention_correctness"`
	RawCorrectPolicyPreserved  int      `json:"raw_correct_policy_preserved"`
	RawWrongPolicyFixed        int      `json:"raw_wrong_policy_fixed"`
	RawCorrectPolicyBroke      int      `json:"raw_correct_policy_broke"`
	RawWrongPolicyFailedToFix  int      `json:"raw_wrong_policy_failed_to_fix"`
	PolicyInducedFalsePositive C1F3Rate `json:"policy_induced_false_positives"`
	PolicyInducedFalseNegative C1F3Rate `json:"policy_induced_false_negatives"`
}

type C1F3ResolverAnalysis struct {
	Correctness      C1F3Rate `json:"correctness"`
	IncorrectCount   int      `json:"incorrect_ticker_or_rule"`
	IncorrectCaseIDs []string `json:"incorrect_case_ids"`
}

type C1F3ProxyRoleAnalysis struct {
	FalseProxies               []string `json:"false_proxies"`
	MissedProxies              []string `json:"missed_proxies"`
	NearestTopicFallbackErrors []string `json:"nearest_topic_fallback_errors"`
	EntityExposureConfusion    []string `json:"entity_exposure_confusion"`
	RoleReversals              []string `json:"role_reversals"`
	PossiblePrincipalMisuse    []string `json:"possible_principal_misuse"`
	AttributionOmissions       []string `json:"attribution_omissions"`
	UnexpectedAttributions     []string `json:"unexpected_attributions"`
}

type C1F3TickerFinding struct {
	CaseID  string `json:"case_id"`
	Finding string `json:"finding"`
}

type C1F3FailureRecord struct {
	CaseID                  string              `json:"case_id"`
	Category                string              `json:"category"`
	ExpectedMapping         AssetMapping        `json:"expected_mapping"`
	RawResult               *V5StructuredResult `json:"raw_result"`
	IdentityOutcome         string              `json:"identity_outcome"`
	RoleProxyIssues         []string            `json:"role_proxy_issues"`
	PolicyEffect            string              `json:"policy_effect"`
	EffectiveMapping        *AssetMapping       `json:"effective_mapping"`
	ResolverResult          *PolicyResolution   `json:"resolver_result"`
	SemanticMappingCorrect  bool                `json:"semantic_mapping_correct"`
	WholeAttributionCorrect bool                `json:"whole_attribution_correct"`
	AttributionComplete     bool                `json:"attribution_complete"`
	ResolverCorrect         bool                `json:"resolver_correct"`
	RootCauseClassification []string            `json:"root_cause_classification"`
}

type C1F3GateResult struct {
	Gate     string `json:"gate"`
	Observed string `json:"observed"`
	Required string `json:"required"`
	Passed   bool   `json:"passed"`
}

type C1F3DatasetReprojection struct {
	ProfileID                   string                 `json:"profile_id"`
	ProfileFingerprint          string                 `json:"profile_fingerprint"`
	FrozenProfile               C1F3EvaluationProfile  `json:"frozen_profile"`
	SourceRunID                 string                 `json:"source_run_id"`
	SourceArtifactIndexSHA256   string                 `json:"source_artifact_index_sha256"`
	SourceNamespace             string                 `json:"source_namespace"`
	CaseCount                   int                    `json:"case_count"`
	Projection                  C1F3ProjectionRecovery `json:"projection_recovery"`
	Score                       C1FDualScore           `json:"score"`
	Identity                    C1F3IdentityAnalysis   `json:"identity_analysis"`
	Policy                      C1F3PolicyAnalysis     `json:"policy_analysis"`
	Retry                       C1F3RetryAccounting    `json:"retry_analysis"`
	Resolver                    C1F3ResolverAnalysis   `json:"resolver_analysis"`
	ProxyRole                   C1F3ProxyRoleAnalysis  `json:"proxy_role_analysis"`
	TickerFindings              []C1F3TickerFinding    `json:"ticker_token_findings"`
	MappingFailures             []C1F3FailureRecord    `json:"mapping_failure_inventory"`
	TypedFailures               []C1F3FailureRecord    `json:"typed_attribution_failure_inventory"`
	AllFailures                 []C1F3FailureRecord    `json:"complete_failure_inventory"`
	Cost                        C1F3CostAccounting     `json:"cost_accounting"`
	SafetyPersistenceViolations int                    `json:"safety_persistence_violations"`
	PrimaryGates                []C1F3GateResult       `json:"generalization_primary_gates,omitempty"`
}

type C1F3CombinedAccounting struct {
	Cases int                 `json:"cases"`
	Retry C1F3RetryAccounting `json:"retry"`
	Cost  C1F3CostAccounting  `json:"cost"`
}

type C1F3ReprojectionEvidence struct {
	Version                    string                     `json:"version"`
	WorkPackage                string                     `json:"work_package"`
	Fingerprint                string                     `json:"fingerprint"`
	ProviderContact            bool                       `json:"provider_contact"`
	Inference                  bool                       `json:"inference"`
	SourceArtifactsModified    bool                       `json:"source_artifacts_modified"`
	SemanticComponentsModified bool                       `json:"semantic_components_modified"`
	Components                 C1F3ReprojectionComponents `json:"components"`
	Datasets                   []C1F3DatasetReprojection  `json:"datasets"`
	Combined                   C1F3CombinedAccounting     `json:"combined"`
}

type c1f3ArtifactHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type c1f3ArtifactIndex struct {
	Version   string             `json:"version"`
	Artifacts []c1f3ArtifactHash `json:"artifacts"`
}

func DefaultC1F3ReprojectionRequest(repositoryRoot string) C1F3ReprojectionRequest {
	return C1F3ReprojectionRequest{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     filepath.Join(repositoryRoot, ".runtime", "diagnostics", C1F3ReprojectionVersion, C1F3ReprojectionWorkPackage),
		Sources: []C1F3ReprojectionSourceSpec{
			{ProfileID: C1F3ProfileGeneralization, SourceNamespace: ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-generalization-v3/WP-00.03C1F3-GENERALIZATION/" + C1F3GeneralizationSourceRunID, RunID: C1F3GeneralizationSourceRunID, ArtifactIndexSHA256: C1F3GeneralizationArtifactIndex},
			{ProfileID: C1F3ProfileBoundary, SourceNamespace: ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-boundary-v3/WP-00.03C1F3-BOUNDARY/" + C1F3BoundarySourceRunID, RunID: C1F3BoundarySourceRunID, ArtifactIndexSHA256: C1F3BoundaryArtifactIndex},
		},
	}
}

func BuildC1F3Reprojection(request C1F3ReprojectionRequest) (C1F3ReprojectionEvidence, error) {
	if len(request.Sources) != 2 {
		return C1F3ReprojectionEvidence{}, fmt.Errorf("C1F3 reprojection requires exactly the two immutable source cells")
	}
	components, err := validateC1F3ReprojectionComponents(request.RepositoryRoot)
	if err != nil {
		return C1F3ReprojectionEvidence{}, err
	}
	evidence := C1F3ReprojectionEvidence{
		Version: C1F3ReprojectionVersion, WorkPackage: C1F3ReprojectionWorkPackage,
		ProviderContact: false, Inference: false, SourceArtifactsModified: false, SemanticComponentsModified: false,
		Components: components,
	}
	seenProfiles := map[string]bool{}
	for _, source := range request.Sources {
		if seenProfiles[source.ProfileID] {
			return C1F3ReprojectionEvidence{}, fmt.Errorf("duplicate C1F3 reprojection profile %q", source.ProfileID)
		}
		seenProfiles[source.ProfileID] = true
		dataset, loadErr := reprojectC1F3Dataset(request.RepositoryRoot, source)
		if loadErr != nil {
			return C1F3ReprojectionEvidence{}, loadErr
		}
		evidence.Datasets = append(evidence.Datasets, dataset)
	}
	if !seenProfiles[C1F3ProfileGeneralization] || !seenProfiles[C1F3ProfileBoundary] {
		return C1F3ReprojectionEvidence{}, fmt.Errorf("C1F3 reprojection source set is incomplete")
	}
	sort.Slice(evidence.Datasets, func(i, j int) bool { return evidence.Datasets[i].ProfileID < evidence.Datasets[j].ProfileID })
	evidence.Combined = combineC1F3Accounting(evidence.Datasets)
	fingerprint, err := c1f3ReprojectionFingerprint(evidence)
	if err != nil {
		return C1F3ReprojectionEvidence{}, err
	}
	evidence.Fingerprint = fingerprint
	return evidence, nil
}

func WriteC1F3Reprojection(request C1F3ReprojectionRequest) (string, string, C1F3ReprojectionEvidence, error) {
	evidence, err := BuildC1F3Reprojection(request)
	if err != nil {
		return "", "", C1F3ReprojectionEvidence{}, err
	}
	dir := filepath.Join(request.OutputRoot, evidence.Fingerprint)
	if _, err = writeExclusiveJSON(filepath.Join(dir, "reprojection.json"), evidence); err != nil {
		return "", "", C1F3ReprojectionEvidence{}, err
	}
	indexPath, indexSHA, err := writeDiagnosticArtifactIndex(dir)
	if err != nil {
		return "", "", C1F3ReprojectionEvidence{}, err
	}
	return indexPath, indexSHA, evidence, nil
}

func c1f3ReprojectionFingerprint(evidence C1F3ReprojectionEvidence) (string, error) {
	copy := evidence
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func validateC1F3ReprojectionComponents(root string) (C1F3ReprojectionComponents, error) {
	specs := []struct {
		path string
		want string
	}{
		{"internal/modules/aishadow/validation_c1f.go", frozenC1FValidatorSHA256},
		{"internal/modules/aishadow/causal_attribution.go", frozenC1EPolicySHA256},
		{"config/event-asset-resolution-v1.json", expectedAssetRulesetFileSHA256},
		{"internal/modules/aishadow/semantic_identity.go", frozenSemanticIdentitySHA256},
		{"internal/modules/aishadow/scoring_c1f.go", frozenC1FScoringSourceSHA256},
	}
	for _, spec := range specs {
		got, err := hashOpaqueFile(filepath.Join(root, filepath.FromSlash(spec.path)))
		if err != nil || got != spec.want {
			return C1F3ReprojectionComponents{}, fmt.Errorf("frozen C1F3 component %s changed: got %s want %s: %v", spec.path, got, spec.want, err)
		}
	}
	rules, err := assetresolution.LoadRuleset(filepath.Join(root, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		return C1F3ReprojectionComponents{}, err
	}
	exposures, err := (assetresolution.Resolver{Rules: rules}).ProxyExposures()
	if err != nil {
		return C1F3ReprojectionComponents{}, err
	}
	schemaSHA, err := fingerprint(V5OutputSchema(exposures))
	if err != nil || schemaSHA != frozenV5SchemaSHA256 || V6PromptSHA256() != frozenV6PromptSHA256 {
		return C1F3ReprojectionComponents{}, fmt.Errorf("frozen C1F3 prompt/output contract changed")
	}
	return C1F3ReprojectionComponents{
		Prompt:         C1F3ComponentBinding{V6PromptVersion, frozenV6PromptSHA256},
		OutputContract: C1F3ComponentBinding{V5SchemaVersion, frozenV5SchemaSHA256},
		Parser:         C1F3ComponentBinding{"ParseValidateAndApplyC1F", frozenC1FValidatorSHA256},
		Validator:      C1F3ComponentBinding{C1FValidatorVersion, frozenC1FValidatorSHA256},
		Policy:         C1F3ComponentBinding{CausalAttributionPolicyVersion, frozenC1EPolicySHA256},
		Resolver:       C1F3ComponentBinding{"event-asset-resolution-v1", expectedAssetRulesetFileSHA256},
		Comparator:     C1F3ComponentBinding{IssuerSemanticIdentityVersion, frozenSemanticIdentitySHA256},
		Scorer:         C1F3ComponentBinding{C1FScoringVersion, frozenC1FScoringSourceSHA256},
	}, nil
}

func reprojectC1F3Dataset(root string, source C1F3ReprojectionSourceSpec) (C1F3DatasetReprojection, error) {
	profile, err := LoadC1F3EvaluationProfile(source.ProfileID)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	if (source.ProfileID == C1F3ProfileGeneralization && (source.RunID != C1F3GeneralizationSourceRunID || source.ArtifactIndexSHA256 != C1F3GeneralizationArtifactIndex)) ||
		(source.ProfileID == C1F3ProfileBoundary && (source.RunID != C1F3BoundarySourceRunID || source.ArtifactIndexSHA256 != C1F3BoundaryArtifactIndex)) {
		return C1F3DatasetReprojection{}, fmt.Errorf("source binding for %s does not match the accepted immutable C1F3 cell", source.ProfileID)
	}
	sourceDir, err := resolveC1F3SourceDirectory(root, source.SourceNamespace)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	index, artifactHashes, err := verifyC1F3ArtifactIndex(sourceDir, source.ArtifactIndexSHA256)
	if err != nil {
		return C1F3DatasetReprojection{}, fmt.Errorf("verify source %s: %w", source.RunID, err)
	}
	_ = index

	rulesPath := filepath.Join(root, "config", "event-asset-resolution-v1.json")
	rules, err := assetresolution.LoadRuleset(rulesPath)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	manifest, err := LoadFrozenC1F3Manifest(profile, filepath.Join(sourceDir, "manifest.json"), exposures)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	if err = ValidateFrozenC1F3InputLock(profile, filepath.Join(sourceDir, "input-fingerprints.json"), manifest); err != nil {
		return C1F3DatasetReprojection{}, err
	}
	sidecar, err := LoadFrozenC1F3TypedLabelSidecar(profile, filepath.Join(root, filepath.FromSlash(profile.TypedSidecarPath)))
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	if err = ValidateC1F3TypedLabelSidecar(profile, sidecar, manifest, resolver); err != nil {
		return C1F3DatasetReprojection{}, err
	}
	if _, err = LoadFrozenC1F3ScoringFreeze(profile, filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath))); err != nil {
		return C1F3DatasetReprojection{}, err
	}

	plan, report, err := loadC1F3SourcePlanReport(sourceDir, source, profile)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	inputs := make(map[string]EventInput, len(manifest.Events))
	inputFingerprints := make(map[string]string, len(manifest.Events))
	for _, event := range manifest.Events {
		inputs[event.ID] = event.Input
		inputFingerprints[event.ID], _ = EventInputFingerprint(event.Input)
	}
	audits, projection, err := reprojectC1F3Attempts(sourceDir, source, profile, inputs, inputFingerprints, artifactHashes, resolver)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	identity := NewIssuerSemanticIdentity(rules)
	profileFingerprint, err := profile.Fingerprint()
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	dataset := C1F3DatasetReprojection{
		ProfileID: source.ProfileID, ProfileFingerprint: profileFingerprint, FrozenProfile: profile,
		SourceRunID: source.RunID, SourceArtifactIndexSHA256: source.ArtifactIndexSHA256,
		SourceNamespace: source.SourceNamespace, CaseCount: profile.Dataset.CaseCount, Projection: projection,
		Score: ScoreC1FDataset(source.ProfileID, sidecar.Cases, audits, identity),
	}
	dataset.Retry = reproduceC1F3RetryAccounting(sidecar.Cases, audits)
	dataset.Cost, err = reproduceC1F3CostAccounting(plan, report, audits, dataset.Retry)
	if err != nil {
		return C1F3DatasetReprojection{}, err
	}
	dataset.SafetyPersistenceViolations = c1f3SafetyViolations(plan.Plan.Safety)
	analyzeC1F3Dataset(&dataset, sidecar.Cases, manifest, audits, identity, resolver)
	if source.ProfileID == C1F3ProfileGeneralization {
		dataset.PrimaryGates = c1f3GeneralizationGates(dataset)
	}
	return dataset, nil
}

func resolveC1F3SourceDirectory(root, namespace string) (string, error) {
	if filepath.IsAbs(namespace) || strings.TrimSpace(namespace) == "" {
		return "", fmt.Errorf("C1F3 source namespace must be a repository-relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(namespace))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("C1F3 source namespace escapes the repository")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", err
	}
	if resolved != rootAbs && !strings.HasPrefix(strings.ToLower(resolved), strings.ToLower(rootAbs+string(filepath.Separator))) {
		return "", fmt.Errorf("C1F3 source namespace escapes the repository")
	}
	return resolved, nil
}

func verifyC1F3ArtifactIndex(sourceDir, expectedIndexSHA string) (c1f3ArtifactIndex, map[string]string, error) {
	indexPath := filepath.Join(sourceDir, "artifact-index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return c1f3ArtifactIndex{}, nil, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != expectedIndexSHA {
		return c1f3ArtifactIndex{}, nil, fmt.Errorf("artifact-index SHA-256 mismatch: got %s want %s", got, expectedIndexSHA)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var index c1f3ArtifactIndex
	if err = decoder.Decode(&index); err != nil {
		return c1f3ArtifactIndex{}, nil, err
	}
	if err = ensureEOF(decoder); err != nil {
		return c1f3ArtifactIndex{}, nil, err
	}
	if index.Version != c1f3ArtifactIndexVersion || len(index.Artifacts) == 0 {
		return c1f3ArtifactIndex{}, nil, fmt.Errorf("unsupported or empty artifact index")
	}
	hashes := map[string]string{}
	for _, artifact := range index.Artifacts {
		clean := filepath.Clean(filepath.FromSlash(artifact.Path))
		if artifact.Path == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == "artifact-index.json" || hashes[artifact.Path] != "" {
			return c1f3ArtifactIndex{}, nil, fmt.Errorf("unsafe or duplicate indexed artifact %q", artifact.Path)
		}
		fileRaw, readErr := os.ReadFile(filepath.Join(sourceDir, clean))
		if readErr != nil {
			return c1f3ArtifactIndex{}, nil, readErr
		}
		fileDigest := sha256.Sum256(fileRaw)
		if got := hex.EncodeToString(fileDigest[:]); got != artifact.SHA256 {
			return c1f3ArtifactIndex{}, nil, fmt.Errorf("indexed artifact %s SHA-256 mismatch: got %s want %s", artifact.Path, got, artifact.SHA256)
		}
		hashes[artifact.Path] = artifact.SHA256
	}
	actual := map[string]bool{}
	err = filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "artifact-index.json" {
			return nil
		}
		relative, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return relErr
		}
		actual[filepath.ToSlash(relative)] = true
		return nil
	})
	if err != nil {
		return c1f3ArtifactIndex{}, nil, err
	}
	if len(actual) != len(hashes) {
		return c1f3ArtifactIndex{}, nil, fmt.Errorf("artifact index coverage mismatch: indexed=%d actual=%d", len(hashes), len(actual))
	}
	for path := range actual {
		if hashes[path] == "" {
			return c1f3ArtifactIndex{}, nil, fmt.Errorf("unindexed source artifact %s", path)
		}
	}
	return index, hashes, nil
}

type c1f3PlanAudit struct {
	RunID         string                  `json:"run_id"`
	StartedAt     time.Time               `json:"started_at"`
	ModelIdentity DiagnosticModelIdentity `json:"model_identity"`
	Plan          DiagnosticPlan          `json:"plan"`
}

func loadC1F3SourcePlanReport(sourceDir string, source C1F3ReprojectionSourceSpec, profile C1F3EvaluationProfile) (c1f3PlanAudit, DiagnosticRunReport, error) {
	var plan c1f3PlanAudit
	if err := decodeStrictJSONFile(filepath.Join(sourceDir, "plan.json"), &plan); err != nil {
		return plan, DiagnosticRunReport{}, err
	}
	var report DiagnosticRunReport
	if err := decodeStrictJSONFile(filepath.Join(sourceDir, "report.json"), &report); err != nil {
		return plan, report, err
	}
	if plan.RunID != source.RunID || report.RunID != source.RunID || plan.StartedAt.IsZero() || plan.ModelIdentity.Name != OpenAIDiagnosticLunaModel ||
		plan.Plan.EvaluationProfile != source.ProfileID || report.ManifestFingerprint != profile.Dataset.SemanticFingerprint ||
		plan.Plan.PromptVersion != V6PromptVersion || report.PromptVersion != V6PromptVersion || plan.Plan.OutputContract != V5SchemaVersion || report.OutputContract != V5SchemaVersion ||
		plan.Plan.CausalAttributionPolicy != CausalAttributionPolicyVersion || report.ModelIdentity.Name != OpenAIDiagnosticLunaModel || report.HostedExperiment == nil {
		return plan, report, fmt.Errorf("source plan/report identity does not match frozen C1F3 cell %s", source.RunID)
	}
	if plan.Plan.C1F3FrozenBindings == nil || plan.Plan.C1F3FrozenBindings.ValidatorIdentity != C1FValidatorVersion || plan.Plan.C1F3FrozenBindings.ScoringIdentity != C1FScoringVersion ||
		plan.Plan.C1F3FrozenBindings.SemanticComparatorIdentity != IssuerSemanticIdentityVersion || plan.Plan.C1F3FrozenBindings.ProfileIdentity != source.ProfileID {
		return plan, report, fmt.Errorf("source plan lost frozen C1F3 bindings")
	}
	return plan, report, nil
}

func decodeStrictJSONFile(path string, value any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(value); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func reprojectC1F3Attempts(sourceDir string, source C1F3ReprojectionSourceSpec, profile C1F3EvaluationProfile, inputs map[string]EventInput, inputFingerprints, artifactHashes map[string]string, resolver assetresolution.Resolver) ([]DiagnosticAttemptAudit, C1F3ProjectionRecovery, error) {
	paths := make([]string, 0)
	for path := range artifactHashes {
		if strings.HasPrefix(path, "repetition-01/") && strings.Contains(path, "-attempt-") && strings.HasSuffix(path, ".json") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	recovery := C1F3ProjectionRecovery{RecoveryErrors: map[string][]string{}, HistoricalV5Errors: map[string]int{}, ReprojectedAttempts: []C1F3ReprojectedAttempt{}}
	audits := make([]DiagnosticAttemptAudit, 0, len(paths))
	requestIDs := map[string]bool{}
	responseIDs := map[string]bool{}
	caseAttempts := map[string][]int{}
	for _, relative := range paths {
		var sourceAudit DiagnosticAttemptAudit
		if err := decodeStrictJSONFile(filepath.Join(sourceDir, filepath.FromSlash(relative)), &sourceAudit); err != nil {
			return nil, recovery, fmt.Errorf("decode source attempt %s: %w", relative, err)
		}
		input, ok := inputs[sourceAudit.CaseID]
		if !ok || sourceAudit.RunID != source.RunID || sourceAudit.Repetition != 1 || sourceAudit.InputFingerprint != inputFingerprints[sourceAudit.CaseID] ||
			sourceAudit.Provider != OpenAIDiagnosticProvider || sourceAudit.ConfiguredModel != OpenAIDiagnosticLunaModel || sourceAudit.ModelReportedIdentifier != OpenAIDiagnosticLunaModel ||
			sourceAudit.PromptVersion != V6PromptVersion || sourceAudit.OutputContract != V5SchemaVersion || strings.TrimSpace(sourceAudit.RequestID) == "" || strings.TrimSpace(sourceAudit.ResponseID) == "" {
			return nil, recovery, fmt.Errorf("source attempt identity mismatch in %s", relative)
		}
		expectedPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("repetition-%02d", sourceAudit.Repetition), fmt.Sprintf("%s-attempt-%02d.json", sourceAudit.CaseID, sourceAudit.AttemptNumber)))
		if relative != expectedPath {
			return nil, recovery, fmt.Errorf("source attempt path identity mismatch in %s", relative)
		}
		if requestIDs[sourceAudit.RequestID] || responseIDs[sourceAudit.ResponseID] {
			return nil, recovery, fmt.Errorf("duplicate source request/response identity in %s", relative)
		}
		requestIDs[sourceAudit.RequestID] = true
		responseIDs[sourceAudit.ResponseID] = true
		if got := rawHash(sourceAudit.RawResponseBody); got != sourceAudit.RawResponseHash {
			return nil, recovery, fmt.Errorf("source raw-response hash mismatch in %s: got %s want %s", relative, got, sourceAudit.RawResponseHash)
		}
		caseAttempts[sourceAudit.CaseID] = append(caseAttempts[sourceAudit.CaseID], sourceAudit.AttemptNumber)
		parsed, decision, resolution, validationErrors := ParseValidateAndApplyC1F(sourceAudit.RawResponseBody, input, resolver)
		if sourceAudit.ValidationStatus == "accepted" {
			recovery.AcceptedRecords++
			if len(validationErrors) > 0 || parsed == nil || decision == nil || resolution == nil {
				recovery.RecoveryErrors[sourceAudit.CaseID] = validationErrors
				return nil, recovery, fmt.Errorf("accepted source attempt %s does not revalidate through frozen C1F route: %s", relative, strings.Join(validationErrors, "; "))
			}
			attribution := TypedAttributionFromV5(*parsed)
			effective := decision.EffectiveMapping
			complete := sourceAudit.V5RawModelOutput != nil && sourceAudit.TypedAttribution != nil && sourceAudit.CausalAttributionPolicy != nil && sourceAudit.EffectiveSemanticMapping != nil && sourceAudit.DeterministicResolution != nil
			_, _, _, historicalV5Errors := ParseValidateAndApplyV5(sourceAudit.RawResponseBody, input, resolver)
			if complete {
				if len(historicalV5Errors) != 0 {
					return nil, recovery, fmt.Errorf("stored complete source projection does not reproduce the historical v5 route in %s", relative)
				}
				recovery.OriginallyComplete++
				if !reflect.DeepEqual(sourceAudit.V5RawModelOutput, parsed) || !reflect.DeepEqual(sourceAudit.TypedAttribution, &attribution) || !reflect.DeepEqual(sourceAudit.CausalAttributionPolicy, decision) || !reflect.DeepEqual(sourceAudit.EffectiveSemanticMapping, &effective) || !reflect.DeepEqual(sourceAudit.DeterministicResolution, resolution) {
					return nil, recovery, fmt.Errorf("stored complete source projection diverges from frozen C1F route in %s", relative)
				}
			} else {
				if len(historicalV5Errors) == 0 {
					return nil, recovery, fmt.Errorf("incomplete source projection is not explained by the historical v5 route in %s", relative)
				}
				recovery.RecoveredRecords++
				for _, validationError := range historicalV5Errors {
					recovery.HistoricalV5Errors[validationError]++
				}
			}
			sourceAudit.V5RawModelOutput = parsed
			sourceAudit.TypedAttribution = &attribution
			sourceAudit.CausalAttributionPolicy = decision
			sourceAudit.EffectiveSemanticMapping = &effective
			sourceAudit.DeterministicResolution = resolution
			recovery.ReprojectedAttempts = append(recovery.ReprojectedAttempts, C1F3ReprojectedAttempt{
				SourceArtifactPath: relative, SourceArtifactSHA256: artifactHashes[relative], SourceRawSHA256: sourceAudit.RawResponseHash,
				CaseID: sourceAudit.CaseID, AttemptNumber: sourceAudit.AttemptNumber, RequestID: sourceAudit.RequestID, ResponseID: sourceAudit.ResponseID,
				OriginallyComplete: complete, HistoricalV5Errors: nonNilStrings(historicalV5Errors),
				RawModelOutput: parsed, TypedAttribution: &attribution, PolicyDecision: decision, EffectiveMapping: &effective, Resolution: resolution,
			})
		} else {
			if len(validationErrors) == 0 || parsed != nil || !reflect.DeepEqual(nonNilStrings(validationErrors), nonNilStrings(sourceAudit.ValidationErrors)) {
				return nil, recovery, fmt.Errorf("rejected source attempt %s does not reproduce its frozen validation outcome", relative)
			}
		}
		audits = append(audits, sourceAudit)
	}
	if len(caseAttempts) != profile.Dataset.CaseCount || recovery.AcceptedRecords != profile.Dataset.CaseCount || len(recovery.RecoveryErrors) != 0 {
		return nil, recovery, fmt.Errorf("source attempt coverage mismatch: cases=%d accepted=%d recovery_errors=%d", len(caseAttempts), recovery.AcceptedRecords, len(recovery.RecoveryErrors))
	}
	for caseID, numbers := range caseAttempts {
		sort.Ints(numbers)
		for index, number := range numbers {
			if number != index+1 {
				return nil, recovery, fmt.Errorf("non-contiguous attempt identity for %s", caseID)
			}
		}
	}
	return audits, recovery, nil
}

func reproduceC1F3RetryAccounting(labels []TypedExpectedCase, audits []DiagnosticAttemptAudit) C1F3RetryAccounting {
	byCase := map[string][]DiagnosticAttemptAudit{}
	for _, audit := range audits {
		byCase[audit.CaseID] = append(byCase[audit.CaseID], audit)
	}
	result := C1F3RetryAccounting{InitialRequests: len(labels), ValidationReasonHistogram: map[string]int{}}
	firstValid, finalValid := 0, 0
	for _, label := range labels {
		attempts := byCase[label.CaseID]
		sort.Slice(attempts, func(i, j int) bool { return attempts[i].AttemptNumber < attempts[j].AttemptNumber })
		if len(attempts) > 0 && attempts[0].ValidationStatus == "accepted" {
			firstValid++
		}
		if len(attempts) > 0 && attempts[len(attempts)-1].ValidationStatus == "accepted" {
			finalValid++
		}
		if len(attempts) > 1 {
			result.CorrectiveRetries += len(attempts) - 1
			for _, reason := range attempts[0].ValidationErrors {
				result.ValidationReasonHistogram[reason]++
			}
			if attempts[0].V5RawModelOutput != nil && attempts[len(attempts)-1].V5RawModelOutput != nil {
				result.SemanticComparisons++
				if !reflect.DeepEqual(attempts[0].V5RawModelOutput, attempts[len(attempts)-1].V5RawModelOutput) {
					result.SemanticChanges++
				}
			} else {
				result.SemanticIndeterminate++
			}
		}
	}
	result.FirstPassValidity = c1f3Rate(firstValid, len(labels))
	result.CorrectiveRetryRate = c1f3Rate(result.CorrectiveRetries, len(labels))
	result.FinalValidity = c1f3Rate(finalValid, len(labels))
	return result
}

func reproduceC1F3CostAccounting(plan c1f3PlanAudit, report DiagnosticRunReport, audits []DiagnosticAttemptAudit, retry C1F3RetryAccounting) (C1F3CostAccounting, error) {
	if plan.Plan.HostedExperiment == nil {
		return C1F3CostAccounting{}, fmt.Errorf("source plan has no hosted accounting")
	}
	pricing := plan.Plan.HostedExperiment.Pricing
	if pricing.InputUSDPerMillionTokens != c1f3FrozenInputPriceUSD || pricing.CachedInputUSDPerMillionTokens != c1f3FrozenCachedInputPriceUSD ||
		pricing.CacheWriteUSDPerMillionTokens != c1f3FrozenCacheWritePriceUSD || pricing.OutputUSDPerMillionTokens != c1f3FrozenOutputPriceUSD || pricing != report.HostedExperiment.Pricing {
		return C1F3CostAccounting{}, fmt.Errorf("source pricing does not match the frozen C1F3 accounting constants")
	}
	inputPrice, _ := parseUSDMicros(pricing.InputUSDPerMillionTokens)
	cachedPrice, _ := parseUSDMicros(pricing.CachedInputUSDPerMillionTokens)
	writePrice, _ := parseUSDMicros(pricing.CacheWriteUSDPerMillionTokens)
	outputPrice, _ := parseUSDMicros(pricing.OutputUSDPerMillionTokens)
	var usage ProviderUsage
	var uncachedCost, cachedCost, writeCost, outputCost int64
	for _, audit := range audits {
		u := audit.Usage
		base := u.InputTokens - u.CachedTokens - u.CacheWriteTokens
		if u.CacheMissTokens > 0 || u.InputTokens == u.CachedTokens+u.CacheMissTokens {
			base = u.CacheMissTokens
		}
		if base < 0 {
			base = 0
		}
		uncachedCost += tokenCostMicros(base, inputPrice)
		cachedCost += tokenCostMicros(u.CachedTokens, cachedPrice)
		writeCost += tokenCostMicros(u.CacheWriteTokens, writePrice)
		outputCost += tokenCostMicros(u.OutputTokens, outputPrice)
		usage.InputTokens += u.InputTokens
		usage.CachedTokens += u.CachedTokens
		usage.CacheMissTokens += u.CacheMissTokens
		usage.CacheWriteTokens += u.CacheWriteTokens
		usage.OutputTokens += u.OutputTokens
		usage.ReasoningTokens += u.ReasoningTokens
		usage.TotalTokens += u.TotalTokens
	}
	total := uncachedCost + cachedCost + writeCost + outputCost
	costs := HostedCostBreakdown{UncachedInputUSD: formatUSDMicros(uncachedCost), CachedInputUSD: formatUSDMicros(cachedCost), CacheWriteUSD: formatUSDMicros(writeCost), OutputUSD: formatUSDMicros(outputCost), TotalUSD: formatUSDMicros(total)}
	if !reflect.DeepEqual(usage, report.HostedExperiment.Usage) || costs != report.HostedExperiment.CostByCategory || formatUSDMicros(total) != report.HostedExperiment.AccountedCostUSD || len(audits) != report.HostedExperiment.RequestCount || retry.CorrectiveRetries != report.HostedExperiment.RetryCount {
		return C1F3CostAccounting{}, fmt.Errorf("independent source usage/cost accounting does not match immutable provider metadata")
	}
	return C1F3CostAccounting{
		Usage: usage, InitialRequests: retry.InitialRequests, CorrectiveRetries: retry.CorrectiveRetries, TotalProviderRequests: len(audits), Pricing: pricing,
		CostByCategory: costs, LocallyAccountedCostUSD: formatUSDMicros(total), BudgetCeilingUSD: report.HostedExperiment.BudgetCeilingUSD,
		ProviderErrors: report.HostedExperiment.ProviderErrorCount, Timeouts: report.HostedExperiment.TimeoutCount, BudgetRejections: report.HostedExperiment.BudgetRejectionCount,
	}, nil
}

func c1f3SafetyViolations(safety DiagnosticSafetyState) int {
	violations := 0
	if safety.AllowLiveTrading || safety.ExecutionEnabled || safety.ExecutionWorker || safety.BrokerExecution || safety.MaximumLeverage > 1 {
		violations++
	}
	return violations
}

func combineC1F3Accounting(datasets []C1F3DatasetReprojection) C1F3CombinedAccounting {
	combined := C1F3CombinedAccounting{Retry: C1F3RetryAccounting{ValidationReasonHistogram: map[string]int{}}, Cost: C1F3CostAccounting{}}
	var uncachedCost, cachedCost, writeCost, outputCost, locallyAccounted, budgetCeiling int64
	for _, dataset := range datasets {
		combined.Cases += dataset.CaseCount
		combined.Retry.InitialRequests += dataset.Retry.InitialRequests
		combined.Retry.FirstPassValidity.Numerator += dataset.Retry.FirstPassValidity.Numerator
		combined.Retry.CorrectiveRetries += dataset.Retry.CorrectiveRetries
		combined.Retry.FinalValidity.Numerator += dataset.Retry.FinalValidity.Numerator
		combined.Retry.SemanticChanges += dataset.Retry.SemanticChanges
		combined.Retry.SemanticComparisons += dataset.Retry.SemanticComparisons
		combined.Retry.SemanticIndeterminate += dataset.Retry.SemanticIndeterminate
		for reason, count := range dataset.Retry.ValidationReasonHistogram {
			combined.Retry.ValidationReasonHistogram[reason] += count
		}
		combined.Cost.InitialRequests += dataset.Cost.InitialRequests
		combined.Cost.CorrectiveRetries += dataset.Cost.CorrectiveRetries
		combined.Cost.TotalProviderRequests += dataset.Cost.TotalProviderRequests
		combined.Cost.ProviderErrors += dataset.Cost.ProviderErrors
		combined.Cost.Timeouts += dataset.Cost.Timeouts
		combined.Cost.BudgetRejections += dataset.Cost.BudgetRejections
		combined.Cost.Usage.InputTokens += dataset.Cost.Usage.InputTokens
		combined.Cost.Usage.CachedTokens += dataset.Cost.Usage.CachedTokens
		combined.Cost.Usage.CacheMissTokens += dataset.Cost.Usage.CacheMissTokens
		combined.Cost.Usage.CacheWriteTokens += dataset.Cost.Usage.CacheWriteTokens
		combined.Cost.Usage.OutputTokens += dataset.Cost.Usage.OutputTokens
		combined.Cost.Usage.ReasoningTokens += dataset.Cost.Usage.ReasoningTokens
		combined.Cost.Usage.TotalTokens += dataset.Cost.Usage.TotalTokens
		cost, _ := parseUSDMicros(dataset.Cost.LocallyAccountedCostUSD)
		locallyAccounted += cost
		cost, _ = parseUSDMicros(dataset.Cost.CostByCategory.UncachedInputUSD)
		uncachedCost += cost
		cost, _ = parseUSDMicros(dataset.Cost.CostByCategory.CachedInputUSD)
		cachedCost += cost
		cost, _ = parseUSDMicros(dataset.Cost.CostByCategory.CacheWriteUSD)
		writeCost += cost
		cost, _ = parseUSDMicros(dataset.Cost.CostByCategory.OutputUSD)
		outputCost += cost
		cost, _ = parseUSDMicros(dataset.Cost.BudgetCeilingUSD)
		budgetCeiling += cost
	}
	combined.Retry.FirstPassValidity = c1f3Rate(combined.Retry.FirstPassValidity.Numerator, combined.Cases)
	combined.Retry.CorrectiveRetryRate = c1f3Rate(combined.Retry.CorrectiveRetries, combined.Cases)
	combined.Retry.FinalValidity = c1f3Rate(combined.Retry.FinalValidity.Numerator, combined.Cases)
	combined.Cost.LocallyAccountedCostUSD = formatUSDMicros(locallyAccounted)
	combined.Cost.CostByCategory = HostedCostBreakdown{
		UncachedInputUSD: formatUSDMicros(uncachedCost), CachedInputUSD: formatUSDMicros(cachedCost),
		CacheWriteUSD: formatUSDMicros(writeCost), OutputUSD: formatUSDMicros(outputCost), TotalUSD: formatUSDMicros(locallyAccounted),
	}
	combined.Cost.Pricing = HostedPricingPlan{InputUSDPerMillionTokens: c1f3FrozenInputPriceUSD, CachedInputUSDPerMillionTokens: c1f3FrozenCachedInputPriceUSD, CacheWriteUSDPerMillionTokens: c1f3FrozenCacheWritePriceUSD, OutputUSDPerMillionTokens: c1f3FrozenOutputPriceUSD, Source: OpenAIDiagnosticPricingSource}
	combined.Cost.BudgetCeilingUSD = formatUSDMicros(budgetCeiling)
	return combined
}

func c1f3Rate(numerator, denominator int) C1F3Rate {
	percentage := 0.0
	if denominator > 0 {
		percentage = float64(numerator) * 100 / float64(denominator)
	}
	return C1F3Rate{Numerator: numerator, Denominator: denominator, Percentage: percentage}
}

func c1f3GeneralizationGates(dataset C1F3DatasetReprojection) []C1F3GateResult {
	gates := FrozenC1F3QualityGates()
	semantic := dataset.Score.Semantic
	return []C1F3GateResult{
		{"final validity", fmt.Sprintf("%.2f%%", semantic.FinalValidity.Percentage), fmt.Sprintf(">= %.2f%%", gates.FinalValidity), semantic.FinalValidity.Percentage >= gates.FinalValidity},
		{"semantic DIRECT precision", fmt.Sprintf("%.2f%%", semantic.DirectPrecision.Percentage), fmt.Sprintf(">= %.2f%%", gates.DirectPrecision), semantic.DirectPrecision.Percentage >= gates.DirectPrecision},
		{"semantic DIRECT recall", fmt.Sprintf("%.2f%%", semantic.DirectRecall.Percentage), fmt.Sprintf(">= %.2f%%", gates.DirectRecall), semantic.DirectRecall.Percentage >= gates.DirectRecall},
		{"semantic false-DIRECT", fmt.Sprintf("%.2f%%", semantic.FalseDirect.Percentage), fmt.Sprintf("<= %.2f%%", gates.SemanticFalseDirect), semantic.FalseDirect.Percentage <= gates.SemanticFalseDirect},
		{"incorrect deterministic ticker resolutions", fmt.Sprint(dataset.Resolver.IncorrectCount), fmt.Sprintf("<= %d", gates.MaximumIncorrectTickerResolutions), dataset.Resolver.IncorrectCount <= gates.MaximumIncorrectTickerResolutions},
		{"safety/persistence violations", fmt.Sprint(dataset.SafetyPersistenceViolations), fmt.Sprintf("<= %d", gates.MaximumSafetyViolations), dataset.SafetyPersistenceViolations <= gates.MaximumSafetyViolations},
	}
}
