package aishadow

import "fmt"

const (
	DiagnosticProfileOriginal         = DiagnosticManifestVersion
	DiagnosticProfileGeneralization   = "ai-shadow-issuer-generalization-holdout-v1"
	DiagnosticProfileBoundary         = "ai-shadow-issuer-boundary-challenge-v1"
	DiagnosticProfileGeneralizationV2 = "ai-shadow-issuer-generalization-holdout-v2"
	DiagnosticProfileBoundaryV2       = "ai-shadow-issuer-boundary-challenge-v2"
	CausalAttributionScoringVersion   = "ai-shadow-causal-attribution-metrics-v1"

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
	ExecutionPromptVersion         string
	ExecutionOutputContract        string
	ExecutionCausalPolicy          string
	ScoringVersion                 string
	ScoringRubricVersion           string
	RequiresTypedAttributionLabels bool
	TypedLabelPath                 string
	TypedLabelVersion              string
	TypedLabelFileSHA256           string
	TypedLabelFingerprint          string
	ScoringRubricPath              string
	ScoringRubricFileSHA256        string
	ScoringRubricFingerprint       string
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
	DiagnosticProfileGeneralizationV2: {
		Identity:                   DiagnosticProfileGeneralizationV2,
		ManifestPath:               "config/ai-shadow-issuer-generalization-holdout-v2.json",
		ManifestVersion:            DiagnosticProfileGeneralizationV2,
		ManifestFileSHA256:         "7b22c4c6d72d53d9976df17463bd6116a50ac305008c6d71c5a36f6971091c04",
		ManifestFingerprint:        "1686f2fa9494dd07887763e0bf55b37a8f0fa2cdb98214a6a23a03156e1f9114",
		FingerprintLockPath:        "config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v2.json",
		FingerprintLockVersion:     "ai-shadow-issuer-generalization-holdout-input-fingerprints-v2",
		FingerprintLockFileSHA256:  "c3dc9715e5c7bcc1f8e0cb1020d95e7979a79f4c5748abf0124df5fa76e1cf88",
		FingerprintLockFingerprint: "81d369388e071e7b84f6adeae997b3f8ccb9d9bdcd2e9e4ae2426c75c4e5b514",
		FreezePath:                 "config/ai-shadow-issuer-generalization-holdout-freeze-v2.json",
		FreezeVersion:              "ai-shadow-issuer-generalization-holdout-freeze-v2",
		FreezeFileSHA256:           "e32eb3ef76a234b5b53db2cea9c011a4e1d85571c6d4b865e1498cd48761d878",
		CaseCount:                  48,
		CategoryCounts: map[string]int{
			"ambiguous_or_invalid_exposure": 4, "clear_causal_direct": 5, "commentator_beneficiary_boundary": 4,
			"contextual_historical_mention": 4, "equal_causality_multi_issuer": 4, "legitimate_principal_proxy": 6,
			"low_medium_relevance_direct": 5, "principal_issuer_among_many": 4, "supported_alias_direct": 4,
			"unsupported_causal_issuer": 3, "weak_causal_direct": 5,
		},
		DefaultRepetitions:             1,
		AllowedRepetitions:             []int{1},
		RequiredProvider:               OpenAIDiagnosticProvider,
		RequiredModel:                  OpenAIDiagnosticLunaModel,
		RequiredExperimentID:           OpenAIC1E3GeneralizationExperimentID,
		RequiredOutputContractMode:     OpenAIOutputContractStrictJSONSchema,
		EvidenceNamespace:              OpenAIC1E3GeneralizationEvidenceNamespace,
		CredentiallessPreflightAllowed: true,
		MaximumBudgetMicros:            300_000,
		ExecutionPromptVersion:         V5PromptVersion,
		ExecutionOutputContract:        V5SchemaVersion,
		ExecutionCausalPolicy:          CausalAttributionPolicyVersion,
		ScoringVersion:                 CausalAttributionScoringVersion,
		ScoringRubricVersion:           C1E2AScoringRubricVersion,
		RequiresTypedAttributionLabels: true,
		TypedLabelPath:                 "config/ai-shadow-causal-attribution-labels-generalization-v2-v1.json",
		TypedLabelVersion:              GeneralizationV2TypedLabelsVersion,
		TypedLabelFileSHA256:           "7092398e52df79c375b850e2d408e689ab879fecf1b2d37a63520e2cd98d2855",
		TypedLabelFingerprint:          "1fceef630e5e5519a7bfe09e2d807cbbfd5c524ee4058515335b26bea2616db9",
		ScoringRubricPath:              "config/ai-shadow-causal-attribution-scoring-c1e3-v1.json",
		ScoringRubricFileSHA256:        "d9b415d0201657ad7c322e5b3dc9064d4ae4e0f572e5b3232b1b1d66b367066b",
		ScoringRubricFingerprint:       "349ca731a4b95d1d0046f0d8eacf7ee771b1c0db1e7f127910b63ece4e5d11df",
	},
	DiagnosticProfileBoundaryV2: {
		Identity:                   DiagnosticProfileBoundaryV2,
		ManifestPath:               "config/ai-shadow-issuer-boundary-challenge-v2.json",
		ManifestVersion:            DiagnosticProfileBoundaryV2,
		ManifestFileSHA256:         "ae2e15a18e28094c44663bd94bc8f40145e3fd1358ae46e525fca85166ce7578",
		ManifestFingerprint:        "4567c371d1a376596911c97a76a8b8cb24efc6ccee376ff78485f0a79d09fdd9",
		FingerprintLockPath:        "config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v2.json",
		FingerprintLockVersion:     "ai-shadow-issuer-boundary-challenge-input-fingerprints-v2",
		FingerprintLockFileSHA256:  "3cced77fbc0d2d229143f379d22365981a668dce6e93174027b0dcfe7a137112",
		FingerprintLockFingerprint: "0c48bf1456a107d2996676157416ca9b51b23d59e60cecfccfa26d19c3271953",
		FreezePath:                 "config/ai-shadow-issuer-boundary-challenge-freeze-v2.json",
		FreezeVersion:              "ai-shadow-issuer-boundary-challenge-freeze-v2",
		FreezeFileSHA256:           "0123286e7e0862961368e85ebea81d3474b49008bd70f85b5c21f7ad75f80dc2",
		CaseCount:                  32,
		CategoryCounts: map[string]int{
			"ambiguous_unknown_identity": 2, "beneficiary_subject_distinction": 2, "clear_causal_direct": 2,
			"direct_plus_sector_effect": 2, "equal_causality_multi_issuer": 2, "high_relevance_noncausal_mention": 2,
			"historical_contextual_mention": 2, "incidental_entity_extraction": 2, "low_medium_relevance_causal_issuer": 2,
			"multiple_plausible_proxies": 2, "principal_proxy": 2, "principal_vs_related_exposure": 2,
			"single_principal_among_many": 2, "supplier_customer_causal_chain": 2, "victim_commentator_distinction": 2,
			"weak_causal_direct": 2,
		},
		DefaultRepetitions:             1,
		AllowedRepetitions:             []int{1},
		RequiredProvider:               OpenAIDiagnosticProvider,
		RequiredModel:                  OpenAIDiagnosticLunaModel,
		RequiredExperimentID:           OpenAIC1E3BoundaryExperimentID,
		RequiredOutputContractMode:     OpenAIOutputContractStrictJSONSchema,
		EvidenceNamespace:              OpenAIC1E3BoundaryEvidenceNamespace,
		CredentiallessPreflightAllowed: true,
		MaximumBudgetMicros:            200_000,
		ExecutionPromptVersion:         V5PromptVersion,
		ExecutionOutputContract:        V5SchemaVersion,
		ExecutionCausalPolicy:          CausalAttributionPolicyVersion,
		ScoringVersion:                 CausalAttributionScoringVersion,
		ScoringRubricVersion:           C1E2AScoringRubricVersion,
		RequiresTypedAttributionLabels: true,
		TypedLabelPath:                 "config/ai-shadow-causal-attribution-labels-boundary-v2-v1.json",
		TypedLabelVersion:              BoundaryV2TypedLabelsVersion,
		TypedLabelFileSHA256:           "54be48060bc4c430f19824ada39db4054bbdb93ac46bb698d939ddbf8a61c5d5",
		TypedLabelFingerprint:          "db5490be5c99675e1a27ebd39706c184538f3663694806c16814bad3b0f191a1",
		ScoringRubricPath:              "config/ai-shadow-causal-attribution-scoring-c1e3-v1.json",
		ScoringRubricFileSHA256:        "d9b415d0201657ad7c322e5b3dc9064d4ae4e0f572e5b3232b1b1d66b367066b",
		ScoringRubricFingerprint:       "349ca731a4b95d1d0046f0d8eacf7ee771b1c0db1e7f127910b63ece4e5d11df",
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
	return p.Identity == DiagnosticProfileGeneralization || p.Identity == DiagnosticProfileBoundary ||
		p.Identity == DiagnosticProfileGeneralizationV2 || p.Identity == DiagnosticProfileBoundaryV2
}

func (p DiagnosticEvaluationProfile) executionVersions() (prompt, output, policy string) {
	if p.ExecutionOutputContract == "" {
		return PromptVersion, SchemaVersion, CausalConsistencyPolicyVersion
	}
	return p.ExecutionPromptVersion, p.ExecutionOutputContract, p.ExecutionCausalPolicy
}
