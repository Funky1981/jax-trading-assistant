package aishadow

import "fmt"

const (
	C1F3ProfileGeneralization = "openai-hosted-c1f3-generalization-v3"
	C1F3ProfileBoundary       = "openai-hosted-c1f3-boundary-v3"
)

type C1F3FrozenDatasetMetadata struct {
	Identity            string `json:"identity"`
	CaseCount           int    `json:"case_count"`
	ManifestPath        string `json:"manifest_path"`
	ManifestSHA256      string `json:"manifest_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
	InputLockPath       string `json:"input_lock_path"`
	InputLockSHA256     string `json:"input_lock_sha256"`
	InputFingerprint    string `json:"input_fingerprint"`
	FreezePath          string `json:"freeze_path"`
	FreezeSHA256        string `json:"freeze_sha256"`
}

type C1F3EvaluationProfile struct {
	Identity                   string                    `json:"identity"`
	Dataset                    C1F3FrozenDatasetMetadata `json:"dataset"`
	Provider                   string                    `json:"provider"`
	Model                      string                    `json:"model"`
	Prompt                     string                    `json:"prompt"`
	PromptSHA256               string                    `json:"prompt_sha256"`
	OutputContract             string                    `json:"output_contract"`
	OutputContractSHA256       string                    `json:"output_contract_sha256"`
	Validator                  string                    `json:"validator"`
	ValidatorSHA256            string                    `json:"validator_sha256"`
	AttributionPolicy          string                    `json:"attribution_policy"`
	AttributionPolicySHA256    string                    `json:"attribution_policy_sha256"`
	SemanticIdentity           string                    `json:"semantic_identity"`
	SemanticIdentitySHA256     string                    `json:"semantic_identity_sha256"`
	Scoring                    string                    `json:"scoring"`
	ScoringSHA256              string                    `json:"scoring_sha256"`
	Resolver                   string                    `json:"resolver"`
	ResolverSHA256             string                    `json:"resolver_sha256"`
	TypedSidecarIdentity       string                    `json:"typed_sidecar_identity"`
	TypedSidecarPath           string                    `json:"typed_sidecar_path"`
	TypedSidecarSHA256         string                    `json:"typed_sidecar_sha256"`
	TypedSidecarFingerprint    string                    `json:"typed_sidecar_fingerprint"`
	ScoringRubricPath          string                    `json:"scoring_rubric_path"`
	ScoringRubricSHA256        string                    `json:"scoring_rubric_sha256"`
	ScoringRubricFingerprint   string                    `json:"scoring_rubric_fingerprint"`
	AdjudicationRubricIdentity string                    `json:"adjudication_rubric_identity"`
	AdjudicationRubricPath     string                    `json:"adjudication_rubric_path"`
	AdjudicationRubricSHA256   string                    `json:"adjudication_rubric_sha256"`
	DefaultRepetitions         int                       `json:"default_repetitions"`
	AllowedRepetitions         []int                     `json:"allowed_repetitions"`
	Executable                 bool                      `json:"executable"`
	DefaultDenyReason          string                    `json:"default_deny_reason"`
}

var c1f3Profiles = map[string]C1F3EvaluationProfile{
	C1F3ProfileGeneralization: c1f3Profile(C1F3ProfileGeneralization, C1F3FrozenDatasetMetadata{
		Identity: "ai-shadow-issuer-generalization-holdout-v3", CaseCount: 48,
		ManifestPath: "config/ai-shadow-issuer-generalization-holdout-v3.json", ManifestSHA256: "6abd7767e0031945e71f2a1d3ef49536adc2d9e9b7d4a78bd938f1f469d27502", SemanticFingerprint: "6e4cdd0133b1a12e980650e18786070586a522375c01d0ed0f2e61823a42f86c",
		InputLockPath: "config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v3.json", InputLockSHA256: "0b6ab35b963c99dae44d1c43fa657dc23c202a65f6b78f17dcc73af0a145ee8a", InputFingerprint: "890dd084bc65828b839b250d5c0a7663016d11ee6bfade7a9b1dc7cd61e7d501",
		FreezePath: "config/ai-shadow-issuer-generalization-holdout-freeze-v3.json", FreezeSHA256: "747beb6e52bbd8d3710674bd79014eee5bfb9d801e902e5a4e2e9e385d9cee6b"}),
	C1F3ProfileBoundary: c1f3Profile(C1F3ProfileBoundary, C1F3FrozenDatasetMetadata{
		Identity: "ai-shadow-issuer-boundary-challenge-v3", CaseCount: 32,
		ManifestPath: "config/ai-shadow-issuer-boundary-challenge-v3.json", ManifestSHA256: "cb2c93afa18cd889790664e3642fadd57015506af23873942115c29d8be27a56", SemanticFingerprint: "47dac15c2cbc3684d1d1f07bc1cd3188778326f51ea0198fa5cd176f8cd6dda9",
		InputLockPath: "config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v3.json", InputLockSHA256: "dcf485e1a56ecc88e66bba0a1b91ca86530b1bb815d39407380ada92f8641d37", InputFingerprint: "7707a32766748fce1e1f564dc2bd18d0e4042853fecf55115ffb09f770e0196e",
		FreezePath: "config/ai-shadow-issuer-boundary-challenge-freeze-v3.json", FreezeSHA256: "46a915f41536872da4e153386eb166612a356b65a6e3aebc197b533e2ae5131c"}),
}

func c1f3Profile(identity string, dataset C1F3FrozenDatasetMetadata) C1F3EvaluationProfile {
	profile := C1F3EvaluationProfile{Identity: identity, Dataset: dataset, Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel,
		Prompt: V6PromptVersion, PromptSHA256: frozenV6PromptSHA256, OutputContract: V5SchemaVersion, OutputContractSHA256: frozenV5SchemaSHA256,
		Validator: C1FValidatorVersion, ValidatorSHA256: frozenC1FValidatorSHA256, AttributionPolicy: CausalAttributionPolicyVersion, AttributionPolicySHA256: frozenC1EPolicySHA256,
		SemanticIdentity: IssuerSemanticIdentityVersion, SemanticIdentitySHA256: frozenSemanticIdentitySHA256, Scoring: C1FScoringVersion, ScoringSHA256: frozenC1FScoringSourceSHA256,
		Resolver: "event-asset-resolution-v1", ResolverSHA256: expectedAssetRulesetFileSHA256, AdjudicationRubricIdentity: C1F2AAdjudicationRubricVersion,
		AdjudicationRubricPath: "config/ai-shadow-causal-attribution-adjudication-rubric-c1f2a-v1.json", AdjudicationRubricSHA256: "bf5548387b1627e1dedc92198705e555721c913df9a316e9b469fd69598dbac3",
		ScoringRubricPath: "config/ai-shadow-causal-attribution-scoring-c1f3-v1.json", DefaultRepetitions: 1, AllowedRepetitions: []int{1}, Executable: false,
		DefaultDenyReason: "C1F3 execution is not authorized; C1F2A freezes metadata only"}
	if identity == C1F3ProfileGeneralization {
		profile.TypedSidecarIdentity = C1F3GeneralizationTypedLabelsVersion
		profile.TypedSidecarPath = "config/ai-shadow-causal-attribution-labels-generalization-v3-v1.json"
		profile.TypedSidecarSHA256 = "702fc698ee0d17af97321074f49bd5e0e414260248aff77eca40e7e15e920bc4"
		profile.TypedSidecarFingerprint = "46d5e6b09bea1b2c24b513052973ac776865e9a307bd13bf00443b0e2a1ff739"
	} else {
		profile.TypedSidecarIdentity = C1F3BoundaryTypedLabelsVersion
		profile.TypedSidecarPath = "config/ai-shadow-causal-attribution-labels-boundary-v3-v1.json"
		profile.TypedSidecarSHA256 = "d0d1604811f32b3323f9bad20d58e68044eb97c3829050fdb87f8bb94d3ef19f"
		profile.TypedSidecarFingerprint = "3a591381686aae2125228cbe5623d4c95dcac6ef4ae8f9f6f7b87d018548dce6"
	}
	profile.ScoringRubricSHA256 = "7f795302a4c33c1a7c103294c1cab590a548c9e2cd746ca7908a10dc2a4a65a9"
	profile.ScoringRubricFingerprint = "744a197191d2226005dd823b5793bafc86bf5e52164fd515efbe1efdc514162d"
	return profile
}

func LoadC1F3EvaluationProfile(identity string) (C1F3EvaluationProfile, error) {
	p, ok := c1f3Profiles[identity]
	if !ok {
		return C1F3EvaluationProfile{}, fmt.Errorf("unknown C1F3 evaluation profile %q", identity)
	}
	return p, nil
}

func (p C1F3EvaluationProfile) Fingerprint() (string, error) { return fingerprint(p) }

func (p C1F3EvaluationProfile) SourceInputLockIdentity() string {
	if p.Identity == C1F3ProfileGeneralization {
		return "ai-shadow-issuer-generalization-holdout-input-fingerprints-v3"
	}
	return "ai-shadow-issuer-boundary-challenge-input-fingerprints-v3"
}

func ValidateC1F3ExecutionReadiness(p C1F3EvaluationProfile) error {
	if err := ValidateC1FContractRoute(p.Prompt, p.OutputContract, p.Validator, p.AttributionPolicy, p.SemanticIdentity, p.Scoring); err != nil {
		return err
	}
	if p.Provider != OpenAIDiagnosticProvider || p.Model != OpenAIDiagnosticLunaModel || p.Resolver != "event-asset-resolution-v1" || p.DefaultRepetitions != 1 || len(p.AllowedRepetitions) != 1 || p.AllowedRepetitions[0] != 1 {
		return fmt.Errorf("C1F3 profile has an unsupported immutable binding")
	}
	if p.Executable || p.TypedSidecarIdentity == "" || p.TypedSidecarPath == "" || p.TypedSidecarSHA256 == "" || p.TypedSidecarFingerprint == "" || p.ScoringRubricSHA256 == "" || p.ScoringRubricFingerprint == "" {
		return fmt.Errorf("C1F3 execution denied: %s", p.DefaultDenyReason)
	}
	return fmt.Errorf("C1F3 execution denied: C1F2A profiles are frozen non-executable metadata")
}

type C1F3QualityGates struct {
	FinalValidity                     float64 `json:"final_validity"`
	DirectPrecision                   float64 `json:"direct_precision"`
	DirectRecall                      float64 `json:"direct_recall"`
	SemanticFalseDirect               float64 `json:"semantic_false_direct"`
	MaximumIncorrectTickerResolutions int     `json:"maximum_incorrect_ticker_resolutions"`
	MaximumSafetyViolations           int     `json:"maximum_safety_violations"`
}

func FrozenC1F3QualityGates() C1F3QualityGates { return C1F3QualityGates{98, 95, 90, 5, 0, 0} }
