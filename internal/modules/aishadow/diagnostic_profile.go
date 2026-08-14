package aishadow

import "fmt"

const (
	DiagnosticProfileOriginal       = DiagnosticManifestVersion
	DiagnosticProfileGeneralization = "ai-shadow-issuer-generalization-holdout-v1"
	DiagnosticProfileBoundary       = "ai-shadow-issuer-boundary-challenge-v1"

	OpenAIGeneralizationExperimentID      = "WP-00.03C1D2-GENERALIZATION"
	OpenAIBoundaryExperimentID            = "WP-00.03C1D2-BOUNDARY"
	OpenAIGeneralizationEvidenceNamespace = "openai-hosted-c1d2-generalization-v1"
	OpenAIBoundaryEvidenceNamespace       = "openai-hosted-c1d2-boundary-v1"

	expectedAssetRulesetFileSHA256 = "5170cc7c8dede5f1a095dfae1ffa7f0060bb8358c85b6dbb0609dab8ef30ede0"
)

// DiagnosticEvaluationProfile binds an executable diagnostic to immutable,
// registered evidence. It deliberately does not permit caller-supplied hashes
// or case counts.
type DiagnosticEvaluationProfile struct {
	Identity                       string
	ManifestPath                   string
	ManifestVersion                string
	ManifestFileSHA256             string
	ManifestFingerprint            string
	FingerprintLockPath            string
	FingerprintLockVersion         string
	FingerprintLockFileSHA256      string
	FingerprintLockFingerprint     string
	FreezePath                     string
	FreezeVersion                  string
	FreezeFileSHA256               string
	CaseCount                      int
	DefaultRepetitions             int
	AllowedRepetitions             []int
	CategoryCounts                 map[string]int
	RequiredProvider               string
	RequiredModel                  string
	RequiredExperimentID           string
	RequiredOutputContractMode     OpenAIOutputContractMode
	EvidenceNamespace              string
	CredentiallessPreflightAllowed bool
	MaximumBudgetMicros            int64
}

var diagnosticEvaluationProfiles = map[string]DiagnosticEvaluationProfile{
	DiagnosticProfileOriginal: {
		Identity:                   DiagnosticProfileOriginal,
		ManifestPath:               "config/ai-shadow-issuer-diagnostic-manifest-v1.json",
		ManifestVersion:            DiagnosticManifestVersion,
		ManifestFileSHA256:         ExpectedDiagnosticManifestFileSHA256,
		ManifestFingerprint:        ExpectedDiagnosticManifestFingerprint,
		FingerprintLockPath:        "config/ai-shadow-issuer-diagnostic-input-fingerprints-v1.json",
		FingerprintLockVersion:     DiagnosticFingerprintLockVersion,
		FingerprintLockFileSHA256:  "9de8c28b3fc100bdb77ed3baad912e0d3863050df50534f7c67483eb43e87739",
		FingerprintLockFingerprint: "8554be89314d3af29042737fc54f935e313b37e74f56ca56da7909d76b745556",
		CaseCount:                  diagnosticEventCount,
		DefaultRepetitions:         diagnosticRepetitionCount,
		AllowedRepetitions:         []int{1, diagnosticRepetitionCount},
		CategoryCounts: map[string]int{
			"clear_single_issuer_positive": 6,
			"clear_exposure_only_negative": 6,
			"ambiguous_company_reference":  6,
			"multi_issuer_event":           6,
			"less_famous_issuer":           6,
			"common_word_issuer_name":      6,
			"company_mentioned_not_causal": 6,
			"unsupported_unknown_issuer":   6,
		},
	},
	DiagnosticProfileGeneralization: {
		Identity:                   DiagnosticProfileGeneralization,
		ManifestPath:               "config/ai-shadow-issuer-generalization-holdout-v1.json",
		ManifestVersion:            DiagnosticProfileGeneralization,
		ManifestFileSHA256:         "bbad87247e3c532c780b21d052d13b5e5b5407379ec8f73e21ba18c6e9463c72",
		ManifestFingerprint:        "7dbd15ddcc9209c658ad56b1e175e83e2a67a6eadb8813cf36bb50e4531a6ad5",
		FingerprintLockPath:        "config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v1.json",
		FingerprintLockVersion:     "ai-shadow-issuer-generalization-holdout-input-fingerprints-v1",
		FingerprintLockFileSHA256:  "c4161bc7e80fb64984b180e5e5c710a3c1b5f27c92ffc940044e29b774ffb10e",
		FingerprintLockFingerprint: "512fb2a7ccb91c237aba3e65e4cd43a4408435ae25af843f2ba07c01529d3716",
		FreezePath:                 "config/ai-shadow-issuer-generalization-holdout-freeze-v1.json",
		FreezeVersion:              "ai-shadow-issuer-generalization-holdout-freeze-v1",
		FreezeFileSHA256:           "29b24744e917a391994dfed066233635d20b40a9b6939b2fd6a268ec6c17fc86",
		CaseCount:                  48,
		DefaultRepetitions:         1,
		AllowedRepetitions:         []int{1},
		CategoryCounts: map[string]int{
			"ambiguous_company_reference":   6,
			"clear_single_issuer":           6,
			"company_mentioned_not_causal":  6,
			"legitimate_proxy_exposure":     6,
			"less_famous_issuer":            6,
			"multi_issuer_event":            6,
			"supported_issuer_alias":        6,
			"unsupported_or_unknown_issuer": 6,
		},
		RequiredProvider:               OpenAIDiagnosticProvider,
		RequiredModel:                  OpenAIDiagnosticLunaModel,
		RequiredExperimentID:           OpenAIGeneralizationExperimentID,
		RequiredOutputContractMode:     OpenAIOutputContractStrictJSONSchema,
		EvidenceNamespace:              OpenAIGeneralizationEvidenceNamespace,
		CredentiallessPreflightAllowed: true,
		MaximumBudgetMicros:            200_000,
	},
	DiagnosticProfileBoundary: {
		Identity:                   DiagnosticProfileBoundary,
		ManifestPath:               "config/ai-shadow-issuer-boundary-challenge-v1.json",
		ManifestVersion:            DiagnosticProfileBoundary,
		ManifestFileSHA256:         "aa20ee9b37fbb200f21a90a79ca98fb3964ba56603f592cece55a5bd92e366a9",
		ManifestFingerprint:        "84bba8c605a35da96a7c97c15b8176f85e3f9fc8213994177696549630fd2ba7",
		FingerprintLockPath:        "config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v1.json",
		FingerprintLockVersion:     "ai-shadow-issuer-boundary-challenge-input-fingerprints-v1",
		FingerprintLockFileSHA256:  "feabd614dcdc5d853cc7dcb8c50070f4f25c986fec556a20ca68e9a4d08e1dc8",
		FingerprintLockFingerprint: "fc0ad093ed589373f32e9319b81319145595991af4d9fd25a59128c16ca498ed",
		FreezePath:                 "config/ai-shadow-issuer-boundary-challenge-freeze-v1.json",
		FreezeVersion:              "ai-shadow-issuer-boundary-challenge-freeze-v1",
		FreezeFileSHA256:           "310442dddf2f52557b53c275c36036bee443f25b8785b458133ded49bf189ae1",
		CaseCount:                  24,
		DefaultRepetitions:         1,
		AllowedRepetitions:         []int{1},
		CategoryCounts: map[string]int{
			"ambiguous_company_reference":       2,
			"incidental_identifier_language":    2,
			"legitimate_proxy_exposure":         4,
			"multi_issuer_principal_boundary":   2,
			"named_company_causality_boundary":  2,
			"proxy_term_boundary":               2,
			"strong_relevance_no_direct_issuer": 2,
			"supported_issuer_alias":            2,
			"tempting_incorrect_proxy":          2,
			"unsupported_unknown_issuer":        2,
			"weak_legitimate_causal_effect":     2,
		},
		RequiredProvider:               OpenAIDiagnosticProvider,
		RequiredModel:                  OpenAIDiagnosticLunaModel,
		RequiredExperimentID:           OpenAIBoundaryExperimentID,
		RequiredOutputContractMode:     OpenAIOutputContractStrictJSONSchema,
		EvidenceNamespace:              OpenAIBoundaryEvidenceNamespace,
		CredentiallessPreflightAllowed: true,
		MaximumBudgetMicros:            100_000,
	},
}

func LoadDiagnosticEvaluationProfile(identity string) (DiagnosticEvaluationProfile, error) {
	profile, ok := diagnosticEvaluationProfiles[identity]
	if !ok {
		return DiagnosticEvaluationProfile{}, fmt.Errorf("unknown frozen issuer diagnostic evaluation profile %q", identity)
	}
	return profile, nil
}

func (p DiagnosticEvaluationProfile) permitsRepetitions(value int) bool {
	for _, allowed := range p.AllowedRepetitions {
		if value == allowed {
			return true
		}
	}
	return false
}

func (p DiagnosticEvaluationProfile) isHoldout() bool {
	return p.Identity == DiagnosticProfileGeneralization || p.Identity == DiagnosticProfileBoundary
}
