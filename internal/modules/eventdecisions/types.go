package eventdecisions

import (
	"encoding/json"
	"time"

	"jax-trading-assistant/internal/modules/candidates"

	"github.com/google/uuid"
)

type Decision string

const (
	DecisionNoTrade   Decision = "NO_TRADE"
	DecisionWatch     Decision = "WATCH"
	DecisionCandidate Decision = "CANDIDATE"
)

const ProcessingModeDeterministic = "deterministic"

type Ruleset struct {
	Version                        string   `json:"version"`
	ProcessorIdentity              string   `json:"processor_identity"`
	WatchConfidenceMinimum         float64  `json:"watch_confidence_minimum"`
	CandidateEvidenceMinimum       float64  `json:"candidate_evidence_minimum"`
	AllowedCandidateInstrumentType string   `json:"allowed_candidate_instrument_type"`
	MaximumLeverage                float64  `json:"maximum_leverage"`
	MaterialSeverities             []string `json:"material_severities"`
}

type Event struct {
	InboxID                uuid.UUID
	NormalizedEventID      *uuid.UUID
	RawEventID             *uuid.UUID
	Source                 string
	SourceEventID          string
	Status                 string
	EventType              string
	Headline               string
	Summary                string
	SourceURLs             []string
	SourceCount            int
	PublicationAt          time.Time
	CollectionAt           *time.Time
	ReceiptAt              time.Time
	Severity               string
	SourceTier             string
	Confidence             float64
	ConfidenceReasons      []string
	AffectedAssets         []string
	MappingReason          string
	MappingMethods         []string
	ProvenanceAvailable    bool
	IsSynthetic            bool
	SyntheticReason        string
	DataSourceType         string
	SourceProvider         string
	DeterministicAnalysis  string
	AIAnalysisProvider     string
	Candidate              *candidates.Candidate
	CandidateEvidenceScore *candidates.EvidenceScoreSummary
}

type Result struct {
	Decision               Decision       `json:"decision"`
	EvidenceScore          float64        `json:"evidenceScore"`
	EvidenceScoreSource    string         `json:"evidenceScoreSource"`
	AffectedAssets         []string       `json:"affectedAssets"`
	UnknownAssets          bool           `json:"unknownAssets"`
	AssetMappingProvenance map[string]any `json:"assetMappingProvenance"`
	Reasons                []string       `json:"reasons"`
	BlockingReasons        []string       `json:"blockingReasons"`
	MissingEvidence        []string       `json:"missingEvidence"`
	TrustGateState         string         `json:"trustGateState"`
	RiskReviewState        string         `json:"riskReviewState"`
	CandidateID            *uuid.UUID     `json:"candidateId,omitempty"`
}

type PersistedDecision struct {
	ID                  uuid.UUID       `json:"decisionId"`
	SourceInboxEventID  uuid.UUID       `json:"sourceInboxEventId"`
	NormalizedEventID   *uuid.UUID      `json:"normalizedEventId,omitempty"`
	SourceEventIdentity string          `json:"sourceEventIdentity"`
	Decision            Decision        `json:"decision"`
	DecisionVersion     int             `json:"decisionVersion"`
	RulesetVersion      string          `json:"rulesetVersion"`
	ProcessorIdentity   string          `json:"processorIdentity"`
	ProcessingMode      string          `json:"processingMode"`
	DecisionAt          time.Time       `json:"decisionAt"`
	PublicationAt       time.Time       `json:"eventPublicationAt"`
	CollectionAt        *time.Time      `json:"eventCollectionAt,omitempty"`
	ReceiptAt           time.Time       `json:"eventReceiptAt"`
	Source              string          `json:"source"`
	SourceURL           string          `json:"sourceUrl"`
	EventType           string          `json:"eventType"`
	Severity            string          `json:"severity"`
	EvidenceScore       float64         `json:"evidenceScore"`
	EvidenceScoreSource string          `json:"evidenceScoreSource"`
	Confidence          float64         `json:"confidence"`
	AffectedAssets      []string        `json:"affectedAssets"`
	UnknownAssets       bool            `json:"unknownAssets"`
	MappingProvenance   map[string]any  `json:"assetMappingProvenance"`
	Reasons             []string        `json:"reasons"`
	BlockingReasons     []string        `json:"blockingReasons"`
	MissingEvidence     []string        `json:"missingEvidence"`
	TrustGateState      string          `json:"trustGateState"`
	RiskReviewState     string          `json:"riskReviewState"`
	CandidateID         *uuid.UUID      `json:"candidateId,omitempty"`
	ReplayIdentity      string          `json:"replayIdentity"`
	InputFingerprint    string          `json:"inputFingerprint"`
	ReplayMetadata      json.RawMessage `json:"replayMetadata"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}

type Exclusion struct {
	InboxID       uuid.UUID `json:"inboxId"`
	SourceEventID string    `json:"sourceEventId"`
	Reason        string    `json:"reason"`
}

type Failure struct {
	InboxID       uuid.UUID `json:"inboxId"`
	SourceEventID string    `json:"sourceEventId"`
	Error         string    `json:"error"`
}

type EventOutcome struct {
	InboxID       uuid.UUID          `json:"inboxId"`
	SourceEventID string             `json:"sourceEventId"`
	Proposed      Result             `json:"proposed"`
	Persisted     *PersistedDecision `json:"persisted,omitempty"`
	Reused        bool               `json:"reused"`
}

type Summary struct {
	DryRun            bool           `json:"dryRun"`
	RulesetVersion    string         `json:"rulesetVersion"`
	Selected          int            `json:"selected"`
	Eligible          int            `json:"eligible"`
	NoTrade           int            `json:"noTrade"`
	Watch             int            `json:"watch"`
	Candidate         int            `json:"candidate"`
	Excluded          []Exclusion    `json:"excluded"`
	Failures          []Failure      `json:"failures"`
	DecisionsCreated  int            `json:"decisionsCreated"`
	DecisionsReused   int            `json:"decisionsReused"`
	CandidatesCreated int            `json:"candidatesCreated"`
	CandidatesReused  int            `json:"candidatesReused"`
	Outcomes          []EventOutcome `json:"outcomes"`
	StartedAt         time.Time      `json:"startedAt"`
	CompletedAt       time.Time      `json:"completedAt"`
}
