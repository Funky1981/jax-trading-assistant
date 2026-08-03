package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"jax-trading-assistant/internal/modules/evidencequality"
)

const (
	ManifestVersion = "ai-shadow-manifest-v1"
	PromptVersion   = "ai-shadow-prompt-v1"
	SchemaVersion   = "ai-shadow-output-v1"
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
