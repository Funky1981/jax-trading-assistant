package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"jax-trading-assistant/internal/modules/evidencequality"
)

const (
	ManifestVersion     = "ai-shadow-manifest-v1"
	LegacyPromptVersion = "ai-shadow-prompt-v1"
	LegacySchemaVersion = "ai-shadow-output-v1"
	V2PromptVersion     = "ai-shadow-prompt-v2-flat-mapping"
	V2SchemaVersion     = "ai-shadow-output-v2-flat-mapping"
	V3PromptVersion     = "ai-shadow-prompt-v3-bounded-exposure"
	V3SchemaVersion     = "ai-shadow-output-v3-bounded-exposure"
	PromptVersion       = "ai-shadow-prompt-v4-issuer-resolution"
	SchemaVersion       = "ai-shadow-output-v4-issuer-resolution"
	V5PromptVersion     = "ai-shadow-prompt-v5-causal-attribution"
	V5SchemaVersion     = "ai-shadow-output-v5-causal-attribution"
	V6PromptVersion     = "ai-shadow-prompt-v6-causal-attribution-boundaries"
	NoProxyExposure     = "NONE"
)

type EventInput struct {
	Title                string    `json:"title"`
	Summary              string    `json:"summary"`
	Source               string    `json:"source"`
	PublicationTimestamp time.Time `json:"publication_timestamp"`
	ReceiptTimestamp     time.Time `json:"receipt_timestamp"`
	EventCategory        string    `json:"event_category"`
	Entities             []string  `json:"entities"`
	ReceiptEvidence      []string  `json:"receipt_evidence"`
}

type StructuredResult struct {
	MarketRelevance   string   `json:"market_relevance"`
	MappingStatus     string   `json:"mapping_status"`
	DirectIssuer      string   `json:"direct_issuer"`
	ProxyExposure     string   `json:"proxy_exposure"`
	MappingConfidence string   `json:"mapping_confidence"`
	ExpectedHorizon   string   `json:"expected_horizon"`
	LikelyDirection   string   `json:"likely_direction"`
	CatalystType      string   `json:"catalyst_type"`
	Reason            string   `json:"reason"`
	MissingEvidence   []string `json:"missing_evidence"`
}

// CausalRole is the model's typed causal relationship between a named issuer
// and the event. These values are contract data, not resolver classifications.
type CausalRole string

const (
	CausalRolePrincipal         CausalRole = "PRINCIPAL"
	CausalRoleEqualPrincipal    CausalRole = "EQUAL_PRINCIPAL"
	CausalRoleSecondaryAffected CausalRole = "SECONDARY_AFFECTED"
	CausalRoleContextOnly       CausalRole = "CONTEXT_ONLY"
	CausalRolePossiblePrincipal CausalRole = "POSSIBLE_PRINCIPAL"
)

type IssuerAttribution struct {
	Issuer     string     `json:"issuer"`
	CausalRole CausalRole `json:"causal_role"`
}

// V5StructuredResult preserves the v4 fields while adding the typed causal
// information used by the C1E policy. It is deliberately a distinct type so
// historical v4 output can never be silently interpreted as v5.
type V5StructuredResult struct {
	MarketRelevance          string              `json:"market_relevance"`
	MappingStatus            string              `json:"mapping_status"`
	DirectIssuer             string              `json:"direct_issuer"`
	ProxyExposure            string              `json:"proxy_exposure"`
	MappingConfidence        string              `json:"mapping_confidence"`
	ExpectedHorizon          string              `json:"expected_horizon"`
	LikelyDirection          string              `json:"likely_direction"`
	CatalystType             string              `json:"catalyst_type"`
	Reason                   string              `json:"reason"`
	MissingEvidence          []string            `json:"missing_evidence"`
	IssuerAttributions       []IssuerAttribution `json:"issuer_attributions"`
	PrincipalProxyCandidates []string            `json:"principal_proxy_candidates"`
}

// V3StructuredResult preserves the bounded-exposure contract in which the
// model still produced a ticker for DIRECT mappings. It is historical only.
type V3StructuredResult struct {
	MarketRelevance   string   `json:"market_relevance"`
	MappingStatus     string   `json:"mapping_status"`
	DirectTicker      string   `json:"direct_ticker"`
	ProxyExposure     string   `json:"proxy_exposure"`
	MappingConfidence string   `json:"mapping_confidence"`
	ExpectedHorizon   string   `json:"expected_horizon"`
	LikelyDirection   string   `json:"likely_direction"`
	CatalystType      string   `json:"catalyst_type"`
	Reason            string   `json:"reason"`
	MissingEvidence   []string `json:"missing_evidence"`
}

// V2StructuredResult preserves the free-form ticker contract. It is decoded
// only as v2 and is never reinterpreted as a v3 bounded exposure.
type V2StructuredResult struct {
	MarketRelevance   string   `json:"market_relevance"`
	MappingStatus     string   `json:"mapping_status"`
	Ticker            string   `json:"ticker"`
	MappingConfidence string   `json:"mapping_confidence"`
	ExpectedHorizon   string   `json:"expected_horizon"`
	LikelyDirection   string   `json:"likely_direction"`
	CatalystType      string   `json:"catalyst_type"`
	Reason            string   `json:"reason"`
	MissingEvidence   []string `json:"missing_evidence"`
}

// PolicyResolution records the deterministic Jax result separately from the
// model output so policy behaviour is never attributed to AI behaviour.
type PolicyResolution struct {
	Status           string `json:"status"`
	PolicyVersion    string `json:"policy_version"`
	RawDirectIssuer  string `json:"raw_direct_issuer,omitempty"`
	NormalizedIssuer string `json:"normalized_issuer,omitempty"`
	CanonicalIssuer  string `json:"canonical_issuer,omitempty"`
	MatchedAlias     string `json:"matched_alias,omitempty"`
	MatchedRule      string `json:"matched_rule,omitempty"`
	ResolvedTicker   string `json:"resolved_ticker,omitempty"`
	MappingType      string `json:"mapping_type"`
	Relationship     string `json:"relationship"`
	Reason           string `json:"reason"`
}

type V3PersistedResult struct {
	ModelOutput             V3StructuredResult `json:"model_output"`
	DeterministicResolution PolicyResolution   `json:"deterministic_resolution"`
}

type V4PersistedResult struct {
	ModelOutput             StructuredResult           `json:"model_output"`
	CausalConsistencyGuard  *CausalConsistencyDecision `json:"causal_consistency_guard,omitempty"`
	DeterministicResolution PolicyResolution           `json:"deterministic_resolution"`
}

type V5PersistedResult struct {
	RawModelOutput            V5StructuredResult        `json:"raw_model_output"`
	TypedAttribution          TypedCausalAttribution    `json:"typed_causal_attribution"`
	CausalAttributionDecision CausalAttributionDecision `json:"causal_attribution_policy_decision"`
	EffectiveSemanticMapping  AssetMapping              `json:"effective_semantic_mapping"`
	DeterministicResolution   PolicyResolution          `json:"deterministic_resolution"`
}

// LegacyStructuredResult preserves the v1 representation for append-only
// historical result reads. Numeric confidence is intentionally not converted
// into the categorical v2 mapping confidence.
type LegacyStructuredResult struct {
	MarketRelevance  string   `json:"market_relevance"`
	ResolvedAsset    *string  `json:"resolved_asset"`
	AssetMappingType string   `json:"asset_mapping_type"`
	ExpectedHorizon  string   `json:"expected_horizon"`
	LikelyDirection  string   `json:"likely_direction"`
	Confidence       int      `json:"confidence"`
	CatalystType     string   `json:"catalyst_type"`
	Reason           string   `json:"reason"`
	MissingEvidence  []string `json:"missing_evidence"`
}

type PersistedStructuredResult struct {
	SchemaVersion string
	V5            *V5PersistedResult
	Current       *V4PersistedResult
	V3            *V3PersistedResult
	V2            *V2StructuredResult
	Legacy        *LegacyStructuredResult
}

type Manifest struct {
	Version     string          `json:"version"`
	Fingerprint string          `json:"fingerprint"`
	Events      []ManifestEvent `json:"events"`
}

type ManifestEvent struct {
	EventID          string `json:"event_id"`
	InputFingerprint string `json:"input_fingerprint"`
}

type BenchmarkEvent struct {
	ID               string
	Input            EventInput
	InputFingerprint string
	Decision         string
	Mapping          evidencequality.Mapping
	Outcome1H        float64
	Outcome1D        float64
}

type SafetyCounts = evidencequality.SafetyCounts

type Attempt struct {
	RunID                   string
	EventID                 string
	AttemptNumber           int
	InputFingerprint        string
	Provider                string
	Model                   string
	ModelReportedIdentifier string
	PromptVersion           string
	SchemaVersion           string
	Seed                    int64
	Temperature             float64
	RequestTimestamp        time.Time
	ResponseTimestamp       time.Time
	Duration                time.Duration
	RawResponseHash         string
	ValidationStatus        string
	ValidationErrors        []string
	FailureReason           string
	projection              *diagnosticAttemptProjection
}

type EventResult struct {
	Attempt
	ManifestVersion   string
	RetryCount        int
	Parsed            *StructuredResult
	V5Parsed          *V5StructuredResult
	CausalGuard       *CausalConsistencyDecision
	CausalAttribution *CausalAttributionDecision
	Resolution        *PolicyResolution
}

type RunRecord struct {
	ID                  string
	ManifestVersion     string
	ManifestFingerprint string
	Provider            string
	Model               string
	PromptVersion       string
	SchemaVersion       string
	Seed                int64
	Temperature         float64
	EventLimit          int
	StartedAt           time.Time
	SafetyBefore        SafetyCounts
}

type FinishRecord struct {
	RunID         string
	CompletedAt   time.Time
	Status        string
	FailureReason string
	SafetyAfter   SafetyCounts
	ReportPaths   ArtifactPaths
}

type Repository interface {
	SafetyCounts() (SafetyCounts, error)
	StartRun(RunRecord) error
	SaveAttempt(Attempt) error
	SaveResult(EventResult) error
	FinishRun(FinishRecord) error
}

type ProviderRequest struct {
	System         string
	User           string
	Schema         map[string]any
	SchemaContract string
	SchemaSHA256   string
	EventID        string
	AttemptNumber  int
	RequestKind    string
}

type ProviderResponse struct {
	Content           string
	ModelIdentifier   string
	RequestID         string
	ResponseID        string
	Status            string
	SystemFingerprint string
	FinishReason      string
	Usage             ProviderUsage
}

type ProviderUsage struct {
	InputTokens      int `json:"input_tokens"`
	CachedTokens     int `json:"cached_tokens"`
	CacheMissTokens  int `json:"cache_miss_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
	OutputTokens     int `json:"output_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderTrace is retained only by the isolated diagnostic file audit. The
// operational benchmark persistence continues to store hashes, not bodies.
type ProviderTrace struct {
	AttemptNumber     int           `json:"attempt_number"`
	Content           string        `json:"raw_response_body"`
	ModelIdentifier   string        `json:"model_identifier,omitempty"`
	RequestID         string        `json:"request_id,omitempty"`
	ResponseID        string        `json:"response_id,omitempty"`
	Status            string        `json:"status,omitempty"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
	FinishReason      string        `json:"finish_reason,omitempty"`
	Usage             ProviderUsage `json:"usage"`
	ProviderError     string        `json:"provider_error,omitempty"`
}

type Provider interface {
	Complete(ProviderRequest) (ProviderResponse, error)
}

func fingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func rawHash(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}
