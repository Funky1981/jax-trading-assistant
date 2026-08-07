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
	ModelOutput             StructuredResult `json:"model_output"`
	DeterministicResolution PolicyResolution `json:"deterministic_resolution"`
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
}

type EventResult struct {
	Attempt
	ManifestVersion string
	RetryCount      int
	Parsed          *StructuredResult
	Resolution      *PolicyResolution
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
	System string
	User   string
	Schema map[string]any
}

type ProviderResponse struct {
	Content         string
	ModelIdentifier string
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
