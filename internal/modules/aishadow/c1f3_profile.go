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
	Identity             string                    `json:"identity"`
	Dataset              C1F3FrozenDatasetMetadata `json:"dataset"`
	Provider             string                    `json:"provider"`
	Model                string                    `json:"model"`
	Prompt               string                    `json:"prompt"`
	OutputContract       string                    `json:"output_contract"`
	Validator            string                    `json:"validator"`
	AttributionPolicy    string                    `json:"attribution_policy"`
	SemanticIdentity     string                    `json:"semantic_identity"`
	Scoring              string                    `json:"scoring"`
	Resolver             string                    `json:"resolver"`
	TypedSidecarIdentity string                    `json:"typed_sidecar_identity"`
	TypedSidecarSHA256   string                    `json:"typed_sidecar_sha256"`
	ScoringRubricSHA256  string                    `json:"scoring_rubric_sha256"`
	Executable           bool                      `json:"executable"`
	DefaultDenyReason    string                    `json:"default_deny_reason"`
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
	return C1F3EvaluationProfile{Identity: identity, Dataset: dataset, Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticLunaModel,
		Prompt: V6PromptVersion, OutputContract: V5SchemaVersion, Validator: C1FValidatorVersion, AttributionPolicy: CausalAttributionPolicyVersion,
		SemanticIdentity: IssuerSemanticIdentityVersion, Scoring: C1FScoringVersion, Resolver: "event-asset-resolution-v1", Executable: false,
		DefaultDenyReason: "C1F2A typed sidecar and scoring-rubric bindings are not frozen"}
}

func LoadC1F3EvaluationProfile(identity string) (C1F3EvaluationProfile, error) {
	p, ok := c1f3Profiles[identity]
	if !ok {
		return C1F3EvaluationProfile{}, fmt.Errorf("unknown C1F3 evaluation profile %q", identity)
	}
	return p, nil
}

func (p C1F3EvaluationProfile) Fingerprint() (string, error) { return fingerprint(p) }

func ValidateC1F3ExecutionReadiness(p C1F3EvaluationProfile) error {
	if err := ValidateC1FContractRoute(p.Prompt, p.OutputContract, p.Validator, p.AttributionPolicy, p.SemanticIdentity, p.Scoring); err != nil {
		return err
	}
	if p.Provider != OpenAIDiagnosticProvider || p.Model != OpenAIDiagnosticLunaModel || p.Resolver != "event-asset-resolution-v1" {
		return fmt.Errorf("C1F3 profile has an unsupported immutable binding")
	}
	if p.Executable || p.TypedSidecarIdentity == "" || p.TypedSidecarSHA256 == "" || p.ScoringRubricSHA256 == "" {
		return fmt.Errorf("C1F3 execution denied: %s", p.DefaultDenyReason)
	}
	return fmt.Errorf("C1F3 execution denied: C1F2 profiles are non-executable")
}

type C1F3QualityGates struct {
	FinalValidity, DirectPrecision, DirectRecall               float64
	SemanticFalseDirect                                        float64
	MaximumIncorrectTickerResolutions, MaximumSafetyViolations int
}

func FrozenC1F3QualityGates() C1F3QualityGates { return C1F3QualityGates{98, 95, 90, 5, 0, 0} }
