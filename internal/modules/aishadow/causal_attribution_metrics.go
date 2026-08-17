package aishadow

// CausalAttributionMetrics is a versioned C1E metric family. It intentionally
// does not reuse v4 counters whose labels and denominators have different
// semantics. C1E3 will populate it from separately frozen typed labels.
type CausalAttributionMetrics struct {
	Version                          string          `json:"version"`
	FinalSchemaValidity              DiagnosticRate  `json:"final_schema_validity"`
	WholeCaseSemanticCorrectness     DiagnosticRate  `json:"whole_case_semantic_correctness"`
	DirectPrecision                  DiagnosticRate  `json:"direct_precision"`
	DirectRecall                     DiagnosticRate  `json:"direct_recall"`
	ProxyPrecision                   DiagnosticRate  `json:"proxy_precision"`
	ProxyRecall                      DiagnosticRate  `json:"proxy_recall"`
	UnresolvedCorrectness            DiagnosticRate  `json:"unresolved_correctness"`
	FalseDirect                      int             `json:"false_direct"`
	FalseProxy                       int             `json:"false_proxy"`
	PrincipalIssuerCorrectness       DiagnosticRate  `json:"principal_issuer_correctness"`
	EqualCausalityCorrectness        DiagnosticRate  `json:"equal_causality_correctness"`
	PossiblePrincipalCorrectness     DiagnosticRate  `json:"possible_principal_correctness"`
	SecondaryAffectedCorrectness     DiagnosticRate  `json:"secondary_affected_correctness"`
	ContextOnlyCorrectness           DiagnosticRate  `json:"context_only_correctness"`
	PrincipalProxyCorrectness        DiagnosticRate  `json:"principal_proxy_correctness"`
	WholeCaseAttributionCorrectness  DiagnosticRate  `json:"whole_case_causal_attribution_correctness"`
	AttributionCompleteness          DiagnosticRate  `json:"attribution_completeness"`
	AttributionPolicyCorrectness     DiagnosticRate  `json:"deterministic_c1e_policy_correctness"`
	ResolverCorrectness              DiagnosticRate  `json:"resolver_correctness"`
	PolicyInducedFalseNegatives      int             `json:"policy_induced_false_negatives"`
	PolicyInducedFalsePositives      int             `json:"policy_induced_false_positives"`
	PossiblePrincipalOccurrences     int             `json:"possible_principal_occurrences"`
	PossiblePrincipalAbstentions     int             `json:"possible_principal_abstentions"`
	PossiblePrincipalFalseNegatives  int             `json:"possible_principal_false_negatives"`
	PossiblePrincipalTrueAmbiguities int             `json:"possible_principal_true_ambiguities"`
	CorrectiveRetries                int             `json:"corrective_retries"`
	TickerTokenMetricVersion         string          `json:"ticker_token_metric_version"`
	SemanticRepeatability            *DiagnosticRate `json:"semantic_repeatability,omitempty"`
}

func NewCausalAttributionMetrics() CausalAttributionMetrics {
	return CausalAttributionMetrics{
		Version:                  CausalAttributionScoringVersion,
		TickerTokenMetricVersion: "historical-ticker-token-lexical-v1",
	}
}
