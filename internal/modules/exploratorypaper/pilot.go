package exploratorypaper

// This file defines the dormant PAPER-02 control plane. It records the
// prospective sample boundary and preserves non-trades, but it is not wired to
// admit observations until an external activation decision is made.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	PilotMode = ExploratoryPaperMode

	PilotStatusDraft                  = "DRAFT"
	PilotStatusReadyForExternalReview = "READY_FOR_EXTERNAL_REVIEW"
	PilotStatusActive                 = "ACTIVE"
	PilotStatusPaused                 = "PAUSED"
	PilotStatusAborted                = "ABORTED"
	PilotStatusCompleted              = "COMPLETED"

	PilotClassificationCandidate                = "CANDIDATE"
	PilotClassificationWatch                    = "WATCH"
	PilotClassificationNoTrade                  = "NO_TRADE"
	PilotClassificationUnresolved               = "UNRESOLVED"
	PilotClassificationRejectedEvidence         = "REJECTED_EVIDENCE"
	PilotClassificationRejectedRisk             = "REJECTED_RISK"
	PilotClassificationRejectedHuman            = "REJECTED_HUMAN"
	PilotClassificationApprovedExploratoryTrade = "APPROVED_EXPLORATORY_TRADE"

	EvidencePhaseEntry         = "ENTRY_EVIDENCE"
	EvidencePhaseSupporting    = "NEW_SUPPORTING_EVIDENCE"
	EvidencePhaseContradictory = "NEW_CONTRADICTORY_EVIDENCE"
	EvidencePhaseInvalidating  = "NEW_INVALIDATING_EVIDENCE"
	EvidencePhaseNoNewRelevant = "NO_NEW_RELEVANT_EVIDENCE"
	EvidencePhaseMissing       = "MISSING_EVIDENCE"

	PilotObservationKindEntry      = "ENTRY"
	PilotObservationKindReview     = "REVIEW"
	PilotObservationKindExit       = "EXIT"
	PilotObservationKindCheckpoint = "CHECKPOINT"
)

var (
	ErrPilotNotAdmitting     = errors.New("PAPER-02 pilot is not admitting new opportunities")
	ErrPilotIdentityConflict = errors.New("PAPER-02 pilot identity conflict")
	ErrPilotFormalPromotion  = errors.New("exploratory PAPER-02 records cannot become formal evidence")
)

// PilotProtocol is frozen before activation. The prose fields deliberately
// make the operational rules inspectable and hashable without inventing a
// second scientific policy engine.
type PilotProtocol struct {
	ProtocolVersion               string   `json:"protocolVersion"`
	Purpose                       string   `json:"purpose"`
	ExploratoryStatus             string   `json:"exploratoryStatus"`
	StartCondition                string   `json:"startCondition"`
	EndCondition                  string   `json:"endCondition"`
	EligibleEventUniverse         []string `json:"eligibleEventUniverse"`
	EligibleInstrumentUniverse    []string `json:"eligibleInstrumentUniverse"`
	DataEvidenceRequirements      []string `json:"dataEvidenceRequirements"`
	CandidateGate                 []string `json:"candidateGate"`
	HumanApprovalRequirements     []string `json:"humanApprovalRequirements"`
	RiskBoundaries                []string `json:"riskBoundaries"`
	SimulatedExecutionAssumptions []string `json:"simulatedExecutionAssumptions"`
	MaximumLeverage               float64  `json:"maximumLeverage"`
	PositionSizing                string   `json:"positionSizing"`
	ConcurrentPositionPolicy      string   `json:"concurrentPositionPolicy"`
	EntryRules                    []string `json:"entryRules"`
	ReviewCadence                 string   `json:"reviewCadence"`
	ExitRules                     []string `json:"exitRules"`
	FiveSessionRule               string   `json:"fiveSessionRule"`
	MissingDataTreatment          string   `json:"missingDataTreatment"`
	IncidentHandling              string   `json:"incidentHandling"`
	OperatorResponsibilities      []string `json:"operatorResponsibilities"`
	RequiredAuditFields           []string `json:"requiredAuditFields"`
	PilotMetrics                  []string `json:"pilotMetrics"`
	ProhibitedInterpretations     []string `json:"prohibitedInterpretations"`
	StopAbortConditions           []string `json:"stopAbortConditions"`
	PostPilotReview               []string `json:"postPilotReview"`
	TargetOpportunityCount        int      `json:"targetOpportunityCount"`
	MaximumDurationDays           int      `json:"maximumDurationDays"`
	EligibleUniverseVersion       string   `json:"eligibleUniverseVersion"`
}

func (p PilotProtocol) Validate() error {
	if p.ProtocolVersion == "" || p.Purpose == "" || p.ExploratoryStatus != "EXPLORATORY_ONLY" || p.StartCondition == "" || p.EndCondition == "" {
		return fmt.Errorf("pilot protocol identity and exploratory status are incomplete")
	}
	if p.MaximumLeverage <= 0 || p.MaximumLeverage > 1 || math.IsNaN(p.MaximumLeverage) || math.IsInf(p.MaximumLeverage, 0) {
		return fmt.Errorf("pilot maximum leverage must be >0 and <=1")
	}
	if p.TargetOpportunityCount < 50 || p.MaximumDurationDays <= 0 || p.EligibleUniverseVersion == "" {
		return fmt.Errorf("pilot sample target, duration, and universe version are incomplete")
	}
	for name, values := range map[string][]string{
		"eligible event universe":      p.EligibleEventUniverse,
		"eligible instrument universe": p.EligibleInstrumentUniverse,
		"evidence requirements":        p.DataEvidenceRequirements,
		"candidate gate":               p.CandidateGate,
		"human approval":               p.HumanApprovalRequirements,
		"risk boundaries":              p.RiskBoundaries,
		"execution assumptions":        p.SimulatedExecutionAssumptions,
		"entry rules":                  p.EntryRules,
		"exit rules":                   p.ExitRules,
		"operator responsibilities":    p.OperatorResponsibilities,
		"audit fields":                 p.RequiredAuditFields,
		"metrics":                      p.PilotMetrics,
		"prohibited interpretations":   p.ProhibitedInterpretations,
		"abort conditions":             p.StopAbortConditions,
		"post-pilot review":            p.PostPilotReview,
	} {
		if len(values) == 0 {
			return fmt.Errorf("pilot %s are incomplete", name)
		}
	}
	for name, value := range map[string]string{"position sizing": p.PositionSizing, "concurrent position policy": p.ConcurrentPositionPolicy, "review cadence": p.ReviewCadence, "five-session rule": p.FiveSessionRule, "missing-data treatment": p.MissingDataTreatment, "incident handling": p.IncidentHandling} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("pilot %s is incomplete", name)
		}
	}
	return nil
}

func ProtocolContentHash(protocol PilotProtocol) string {
	data, _ := json.Marshal(protocol)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type PilotIdentity struct {
	PilotID                    string     `json:"pilotId"`
	Mode                       string     `json:"mode"`
	ProtocolVersion            string     `json:"protocolVersion"`
	ProtocolContentHash        string     `json:"protocolContentHash"`
	CreatedAt                  time.Time  `json:"createdAt"`
	StartTimestamp             *time.Time `json:"startTimestamp,omitempty"`
	FirstEligibleObservationAt *time.Time `json:"firstEligibleObservationAt,omitempty"`
	TargetOpportunityCount     int        `json:"targetOpportunityCount"`
	MaximumDurationDays        int        `json:"maximumDurationDays"`
	EligibleUniverseVersion    string     `json:"eligibleUniverseVersion"`
	EvidencePolicyVersion      string     `json:"evidencePolicyVersion"`
	TraderModelVersion         string     `json:"traderModelVersion"`
	RiskPolicyVersion          string     `json:"riskPolicyVersion"`
	EntryPolicyVersion         string     `json:"entryPolicyVersion"`
	ExitPolicyVersion          string     `json:"exitPolicyVersion"`
	CostModelVersion           string     `json:"costModelVersion"`
	CalendarVersion            string     `json:"calendarVersion"`
	CodeSHA                    string     `json:"codeSha"`
}

func (i PilotIdentity) Validate(protocol PilotProtocol) error {
	if err := protocol.Validate(); err != nil {
		return err
	}
	if i.PilotID == "" || i.Mode != PilotMode || i.ProtocolVersion != protocol.ProtocolVersion || i.ProtocolContentHash != ProtocolContentHash(protocol) || i.CreatedAt.IsZero() || i.CreatedAt.Location() != time.UTC || i.TargetOpportunityCount != protocol.TargetOpportunityCount || i.MaximumDurationDays != protocol.MaximumDurationDays || i.EligibleUniverseVersion != protocol.EligibleUniverseVersion || i.EvidencePolicyVersion == "" || i.TraderModelVersion == "" || i.RiskPolicyVersion == "" || i.EntryPolicyVersion == "" || i.ExitPolicyVersion == "" || i.CostModelVersion == "" || i.CalendarVersion == "" || i.CodeSHA == "" {
		return fmt.Errorf("pilot identity is incomplete or does not match the frozen protocol")
	}
	if i.StartTimestamp != nil && (i.StartTimestamp.IsZero() || i.StartTimestamp.Location() != time.UTC) {
		return fmt.Errorf("pilot start timestamp must be UTC")
	}
	return nil
}

type PilotRecord struct {
	Identity PilotIdentity `json:"identity"`
	Protocol PilotProtocol `json:"protocol"`
	Status   string        `json:"status"`
}

func NewPilotDraft(pilotID string, protocol PilotProtocol, createdAt time.Time, versions PolicyVersions, costModelVersion, calendarVersion, codeSHA string) (PilotRecord, error) {
	if err := protocol.Validate(); err != nil {
		return PilotRecord{}, err
	}
	if pilotID == "" || createdAt.IsZero() || createdAt.Location() != time.UTC || costModelVersion == "" || calendarVersion == "" || codeSHA == "" {
		return PilotRecord{}, fmt.Errorf("pilot draft identity is incomplete")
	}
	identity := PilotIdentity{PilotID: pilotID, Mode: PilotMode, ProtocolVersion: protocol.ProtocolVersion, ProtocolContentHash: ProtocolContentHash(protocol), CreatedAt: createdAt, TargetOpportunityCount: protocol.TargetOpportunityCount, MaximumDurationDays: protocol.MaximumDurationDays, EligibleUniverseVersion: protocol.EligibleUniverseVersion, EvidencePolicyVersion: versions.CandidatePolicy, TraderModelVersion: TraderModelVersion, RiskPolicyVersion: versions.RiskPolicy, EntryPolicyVersion: versions.EntryPolicy, ExitPolicyVersion: versions.ExitPolicy, CostModelVersion: costModelVersion, CalendarVersion: calendarVersion, CodeSHA: codeSHA}
	if err := identity.Validate(protocol); err != nil {
		return PilotRecord{}, err
	}
	return PilotRecord{Identity: identity, Protocol: protocol, Status: PilotStatusDraft}, nil
}

func MarkPilotReady(record PilotRecord) (PilotRecord, error) {
	if record.Status != PilotStatusDraft {
		return PilotRecord{}, fmt.Errorf("pilot can become READY only from DRAFT")
	}
	if err := record.Identity.Validate(record.Protocol); err != nil {
		return PilotRecord{}, err
	}
	record.Status = PilotStatusReadyForExternalReview
	return record, nil
}

// ActivatePilotForExternalReview is a pure transition used by the future
// external operator boundary. No current runtime calls it.
func ActivatePilotForExternalReview(record PilotRecord, auth ExternalActivationAuthorization) (PilotRecord, error) {
	if record.Status != PilotStatusReadyForExternalReview {
		return PilotRecord{}, fmt.Errorf("pilot activation requires READY_FOR_EXTERNAL_REVIEW")
	}
	if err := auth.Validate(); err != nil {
		return PilotRecord{}, err
	}
	start := auth.AuthorizedAt.UTC()
	record.Identity.StartTimestamp = &start
	if err := record.Identity.Validate(record.Protocol); err != nil {
		return PilotRecord{}, err
	}
	record.Status = PilotStatusActive
	return record, nil
}

type ExternalActivationAuthorization struct {
	Reviewer     string    `json:"reviewer"`
	Decision     string    `json:"decision"`
	Acknowledged string    `json:"acknowledged"`
	AuthorizedAt time.Time `json:"authorizedAt"`
}

func (a ExternalActivationAuthorization) Validate() error {
	if strings.TrimSpace(a.Reviewer) == "" || a.Decision != "ACTIVATE_PAPER_02" || a.Acknowledged != "EXPLORATORY_PAPER_ONLY_NO_LIVE_NO_FORMAL" || a.AuthorizedAt.IsZero() || a.AuthorizedAt.Location() != time.UTC {
		return fmt.Errorf("explicit external PAPER-02 activation authorization is incomplete")
	}
	return nil
}

type OpportunityClassification string

type PilotOpportunity struct {
	PilotID                string                    `json:"pilotId"`
	OpportunityID          string                    `json:"opportunityId"`
	EventID                string                    `json:"eventId"`
	SourceEventIdentity    string                    `json:"sourceEventIdentity"`
	EventCategory          string                    `json:"eventCategory"`
	FirstSeenAt            time.Time                 `json:"firstSeenAt"`
	PublicationAt          *time.Time                `json:"publicationAt,omitempty"`
	IngestionAt            time.Time                 `json:"ingestionAt"`
	SourceProvenance       []EvidenceReference       `json:"sourceProvenance"`
	IssuerID               string                    `json:"issuerId"`
	InstrumentID           string                    `json:"instrumentId"`
	ResolutionState        string                    `json:"resolutionState"`
	EvidenceState          string                    `json:"evidenceState"`
	CandidateDecision      CandidateDecision         `json:"candidateDecision"`
	CandidateDecisionAt    *time.Time                `json:"candidateDecisionAt,omitempty"`
	RiskDecision           string                    `json:"riskDecision"`
	RiskDecisionAt         *time.Time                `json:"riskDecisionAt,omitempty"`
	HumanDecision          string                    `json:"humanDecision"`
	HumanReviewer          string                    `json:"humanReviewer"`
	HumanDecisionAt        *time.Time                `json:"humanDecisionAt,omitempty"`
	HumanRationale         string                    `json:"humanRationale"`
	ThesisHash             string                    `json:"thesisHash"`
	EvidenceSnapshotHash   string                    `json:"evidenceSnapshotHash"`
	Direction              Direction                 `json:"direction"`
	ProtectiveStop         float64                   `json:"protectiveStop"`
	Target                 float64                   `json:"target"`
	HorizonSessions        int                       `json:"horizonSessions"`
	Confidence             float64                   `json:"confidence"`
	Uncertainty            string                    `json:"uncertainty"`
	PaperOnlyAcknowledged  bool                      `json:"paperOnlyAcknowledged"`
	TradeLifecycleID       string                    `json:"tradeLifecycleId"`
	CandidateID            string                    `json:"candidateId"`
	ThesisID               string                    `json:"thesisId"`
	EntryEvidenceIDs       []string                  `json:"entryEvidenceIds"`
	EntryAt                *time.Time                `json:"entryAt,omitempty"`
	FinalClassification    OpportunityClassification `json:"finalClassification"`
	MissingDataFlags       []string                  `json:"missingDataFlags"`
	PolicyModelVersions    PolicyVersions            `json:"policyModelVersions"`
	EvidenceLatency        LatencyChain              `json:"evidenceLatency"`
	NewEvidenceCount       int                       `json:"newEvidenceCount"`
	ReconciliationIncident bool                      `json:"reconciliationIncident"`
	Outcome                *PilotOutcomeSummary      `json:"outcome,omitempty"`
	CreatedAt              time.Time                 `json:"createdAt"`
	UpdatedAt              time.Time                 `json:"updatedAt"`
}

func (o PilotOpportunity) ValidateForPilot(pilot PilotRecord) error {
	if o.PilotID != pilot.Identity.PilotID || o.OpportunityID == "" || o.EventID == "" || o.SourceEventIdentity == "" || o.EventCategory == "" || o.FirstSeenAt.IsZero() || o.FirstSeenAt.Location() != time.UTC || o.IngestionAt.IsZero() || o.IngestionAt.Location() != time.UTC || o.ResolutionState == "" || o.EvidenceState == "" || o.FinalClassification == "" {
		return fmt.Errorf("pilot opportunity identity or provenance is incomplete")
	}
	if pilot.Identity.StartTimestamp != nil && o.FirstSeenAt.Before(*pilot.Identity.StartTimestamp) {
		return fmt.Errorf("retrospective opportunity cannot enter a prospective pilot")
	}
	if o.PublicationAt != nil && (o.PublicationAt.IsZero() || o.PublicationAt.Location() != time.UTC) {
		return fmt.Errorf("opportunity publication timestamp must be UTC")
	}
	return nil
}

type OpportunityIntake struct {
	OpportunityID       string              `json:"opportunityId"`
	EventID             string              `json:"eventId"`
	SourceEventIdentity string              `json:"sourceEventIdentity"`
	EventCategory       string              `json:"eventCategory"`
	FirstSeenAt         time.Time           `json:"firstSeenAt"`
	PublicationAt       *time.Time          `json:"publicationAt,omitempty"`
	IngestionAt         time.Time           `json:"ingestionAt"`
	SourceProvenance    []EvidenceReference `json:"sourceProvenance"`
	IssuerID            string              `json:"issuerId"`
	InstrumentID        string              `json:"instrumentId"`
	ResolutionState     string              `json:"resolutionState"`
	EvidenceState       string              `json:"evidenceState"`
	MissingDataFlags    []string            `json:"missingDataFlags"`
}

type OpportunityDecision struct {
	CandidateDecision     CandidateDecision `json:"candidateDecision"`
	DecisionAt            time.Time         `json:"decisionAt"`
	Reason                string            `json:"reason"`
	RiskDecision          string            `json:"riskDecision"`
	RiskDecisionAt        *time.Time        `json:"riskDecisionAt,omitempty"`
	HumanDecision         string            `json:"humanDecision"`
	HumanReviewer         string            `json:"humanReviewer"`
	HumanDecisionAt       *time.Time        `json:"humanDecisionAt,omitempty"`
	HumanRationale        string            `json:"humanRationale"`
	ThesisHash            string            `json:"thesisHash"`
	EvidenceSnapshotHash  string            `json:"evidenceSnapshotHash"`
	Direction             Direction         `json:"direction"`
	ProtectiveStop        float64           `json:"protectiveStop"`
	Target                float64           `json:"target"`
	HorizonSessions       int               `json:"horizonSessions"`
	Confidence            float64           `json:"confidence"`
	Uncertainty           string            `json:"uncertainty"`
	PaperOnlyAcknowledged bool              `json:"paperOnlyAcknowledged"`
	EvidenceRejected      bool              `json:"evidenceRejected"`
	CandidateID           string            `json:"candidateId"`
	ThesisID              string            `json:"thesisId"`
	TradeLifecycleID      string            `json:"tradeLifecycleId"`
	EntryEvidenceIDs      []string          `json:"entryEvidenceIds"`
	EntryAt               *time.Time        `json:"entryAt,omitempty"`
}

type LatencyValue struct {
	Known         bool   `json:"known"`
	DurationMS    *int64 `json:"durationMs,omitempty"`
	UnknownReason string `json:"unknownReason,omitempty"`
}

func knownLatency(from, to time.Time) LatencyValue {
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return LatencyValue{Known: false, UnknownReason: "timestamp unavailable or ordering invalid"}
	}
	ms := to.Sub(from).Milliseconds()
	return LatencyValue{Known: true, DurationMS: &ms}
}

type LatencyChain struct {
	PublicationToIngestion LatencyValue `json:"publicationToIngestion"`
	IngestionToAssessment  LatencyValue `json:"ingestionToAssessment"`
	AssessmentToCandidate  LatencyValue `json:"assessmentToCandidate"`
	CandidateToHuman       LatencyValue `json:"candidateToHuman"`
	HumanToEntry           LatencyValue `json:"humanToEntry"`
}

func BuildLatencyChain(publication *time.Time, ingestion, assessment, candidate, human, entry time.Time) LatencyChain {
	var pub time.Time
	if publication != nil {
		pub = publication.UTC()
	}
	return LatencyChain{PublicationToIngestion: knownLatency(pub, ingestion), IngestionToAssessment: knownLatency(ingestion, assessment), AssessmentToCandidate: knownLatency(assessment, candidate), CandidateToHuman: knownLatency(candidate, human), HumanToEntry: knownLatency(human, entry)}
}

type PilotEvidenceRecord struct {
	PilotID       string       `json:"pilotId"`
	OpportunityID string       `json:"opportunityId"`
	EvidenceID    string       `json:"evidenceId"`
	FirstSeenAt   time.Time    `json:"firstSeenAt"`
	PublicationAt *time.Time   `json:"publicationAt,omitempty"`
	IngestionAt   time.Time    `json:"ingestionAt"`
	SourceID      string       `json:"sourceId"`
	SourceURL     string       `json:"sourceUrl"`
	Relevance     string       `json:"relevance"`
	Signal        string       `json:"signal"`
	PolicyVersion string       `json:"policyVersion"`
	Phase         string       `json:"phase"`
	Latency       LatencyValue `json:"publicationToIngestion"`
}

type EvidenceInput struct {
	EvidenceID    string     `json:"evidenceId"`
	FirstSeenAt   time.Time  `json:"firstSeenAt"`
	PublicationAt *time.Time `json:"publicationAt,omitempty"`
	IngestionAt   time.Time  `json:"ingestionAt"`
	SourceID      string     `json:"sourceId"`
	SourceURL     string     `json:"sourceUrl"`
	Relevance     string     `json:"relevance"`
	Signal        string     `json:"signal"`
	PolicyVersion string     `json:"policyVersion"`
}

func (e EvidenceInput) Validate() error {
	if e.EvidenceID == "" || e.FirstSeenAt.IsZero() || e.FirstSeenAt.Location() != time.UTC || e.IngestionAt.IsZero() || e.IngestionAt.Location() != time.UTC || e.SourceID == "" || !strings.HasPrefix(strings.ToLower(e.SourceURL), "http") || e.Relevance == "" || e.Signal == "" || e.PolicyVersion == "" {
		return fmt.Errorf("pilot evidence provenance is incomplete")
	}
	if e.PublicationAt != nil && (e.PublicationAt.IsZero() || e.PublicationAt.Location() != time.UTC) {
		return fmt.Errorf("pilot evidence publication timestamp must be UTC")
	}
	return nil
}

func classifyPilotEvidence(opportunity PilotOpportunity, evidence EvidenceInput) (string, error) {
	if err := evidence.Validate(); err != nil {
		return "", err
	}
	for _, id := range opportunity.EntryEvidenceIDs {
		if id == evidence.EvidenceID {
			return EvidencePhaseEntry, nil
		}
	}
	if opportunity.EntryAt == nil {
		return EvidencePhaseEntry, nil
	}
	if !evidence.FirstSeenAt.After(*opportunity.EntryAt) {
		return EvidencePhaseEntry, nil
	}
	if strings.EqualFold(evidence.Relevance, "irrelevant") {
		return EvidencePhaseNoNewRelevant, nil
	}
	switch strings.ToUpper(evidence.Signal) {
	case "SUPPORTS", "STRENGTHENS", "SUPPORTING":
		return EvidencePhaseSupporting, nil
	case "CONTRADICTS", "CONTRADICTORY":
		return EvidencePhaseContradictory, nil
	case "INVALIDATES", "INVALIDATING":
		return EvidencePhaseInvalidating, nil
	default:
		return EvidencePhaseMissing, nil
	}
}

type MarketObservationQuality struct {
	ObservationID    string     `json:"observationId"`
	Kind             string     `json:"kind"`
	Source           string     `json:"source"`
	ObservedAt       time.Time  `json:"observedAt"`
	ReceivedAt       *time.Time `json:"receivedAt,omitempty"`
	AgeSeconds       *int64     `json:"ageSeconds,omitempty"`
	BidAvailable     bool       `json:"bidAvailable"`
	AskAvailable     bool       `json:"askAvailable"`
	Bid              float64    `json:"bid"`
	Ask              float64    `json:"ask"`
	Spread           float64    `json:"spread"`
	PriceUsed        float64    `json:"priceUsed"`
	CostModelVersion string     `json:"costModelVersion"`
	Stale            bool       `json:"stale"`
	Missing          bool       `json:"missing"`
	MissingReason    string     `json:"missingReason,omitempty"`
}

func (m MarketObservationQuality) Validate() error {
	if m.ObservationID == "" || m.Kind == "" || m.CostModelVersion == "" {
		return fmt.Errorf("market observation identity and cost model are required")
	}
	if m.Missing {
		if strings.TrimSpace(m.MissingReason) == "" {
			return fmt.Errorf("missing market data requires an explicit reason")
		}
		return nil
	}
	if m.Source == "" || m.ObservedAt.IsZero() || m.ObservedAt.Location() != time.UTC || m.PriceUsed <= 0 || m.Stale || !m.BidAvailable || !m.AskAvailable || m.Bid <= 0 || m.Ask <= 0 || m.Ask < m.Bid || m.Spread < 0 {
		return fmt.Errorf("market observation is stale, incomplete, or invalid")
	}
	if m.ReceivedAt != nil && (m.ReceivedAt.IsZero() || m.ReceivedAt.Location() != time.UTC || m.ReceivedAt.Before(m.ObservedAt)) {
		return fmt.Errorf("market received timestamp is invalid")
	}
	return nil
}

type PilotOutcomeSummary struct {
	OutcomeID        string  `json:"outcomeId"`
	GrossPnL         float64 `json:"grossPnl"`
	NetPnL           float64 `json:"netPnl"`
	NetReturn        float64 `json:"netReturn"`
	SessionsHeld     int     `json:"sessionsHeld"`
	ExitReason       string  `json:"exitReason"`
	CostModelVersion string  `json:"costModelVersion"`
}

type PilotMetrics struct {
	EligibleOpportunities         int     `json:"eligibleOpportunities"`
	ResolutionRate                float64 `json:"resolutionRate"`
	UnresolvedRate                float64 `json:"unresolvedRate"`
	EvidenceReadyRate             float64 `json:"evidenceReadyRate"`
	WatchRate                     float64 `json:"watchRate"`
	NoTradeRate                   float64 `json:"noTradeRate"`
	CandidateRate                 float64 `json:"candidateRate"`
	RiskRejectionRate             float64 `json:"riskRejectionRate"`
	HumanRejectionRate            float64 `json:"humanRejectionRate"`
	ExploratoryEntryRate          float64 `json:"exploratoryEntryRate"`
	MissingDataRate               float64 `json:"missingDataRate"`
	ReviewCompletionRate          float64 `json:"reviewCompletionRate"`
	KnownEvidenceLatencyCount     int     `json:"knownEvidenceLatencyCount"`
	DuplicateIdempotencyIncidents int     `json:"duplicateIdempotencyIncidents"`
	ReconciliationIncidents       int     `json:"reconciliationIncidents"`
	FailClosedIncidents           int     `json:"failClosedIncidents"`
	DescriptiveGrossPnL           float64 `json:"descriptiveGrossPnl"`
	DescriptiveNetPnL             float64 `json:"descriptiveNetPnl"`
	DescriptiveOutcomeCount       int     `json:"descriptiveOutcomeCount"`
	FormalEvidenceEligible        bool    `json:"formalEvidenceEligible"`
	DemonstratedEdge              bool    `json:"demonstratedEdge"`
}

func ComputePilotMetrics(opportunities []PilotOpportunity) PilotMetrics {
	m := PilotMetrics{EligibleOpportunities: len(opportunities), FormalEvidenceEligible: false, DemonstratedEdge: false}
	if len(opportunities) == 0 {
		return m
	}
	var resolved, evidenceReady, watch, noTrade, candidate, riskRejected, humanRejected, entries, missing, completed, knownLatency int
	for _, o := range opportunities {
		if o.IssuerID != "" && o.InstrumentID != "" {
			resolved++
		}
		if strings.EqualFold(o.EvidenceState, "sufficient") || strings.EqualFold(o.EvidenceState, "ready") {
			evidenceReady++
		}
		switch o.FinalClassification {
		case PilotClassificationWatch:
			watch++
		case PilotClassificationNoTrade:
			noTrade++
		case PilotClassificationCandidate:
			candidate++
		case PilotClassificationRejectedRisk:
			riskRejected++
		case PilotClassificationRejectedHuman:
			humanRejected++
		case PilotClassificationApprovedExploratoryTrade:
			entries++
		}
		if o.FinalClassification == PilotClassificationUnresolved || len(o.MissingDataFlags) > 0 {
			missing++
		}
		if o.TradeLifecycleID != "" && o.Outcome != nil {
			completed++
			m.DescriptiveGrossPnL += o.Outcome.GrossPnL
			m.DescriptiveNetPnL += o.Outcome.NetPnL
			m.DescriptiveOutcomeCount++
		}
		if o.EvidenceLatency.PublicationToIngestion.Known {
			knownLatency++
		}
	}
	n := float64(len(opportunities))
	m.ResolutionRate, m.UnresolvedRate, m.EvidenceReadyRate = float64(resolved)/n, float64(len(opportunities)-resolved)/n, float64(evidenceReady)/n
	m.WatchRate, m.NoTradeRate, m.CandidateRate = float64(watch)/n, float64(noTrade)/n, float64(candidate)/n
	m.RiskRejectionRate, m.HumanRejectionRate, m.ExploratoryEntryRate = float64(riskRejected)/n, float64(humanRejected)/n, float64(entries)/n
	m.MissingDataRate, m.ReviewCompletionRate = float64(missing)/n, float64(completed)/n
	m.KnownEvidenceLatencyCount = knownLatency
	return m
}

type PilotIncident struct {
	IncidentID       string    `json:"incidentId"`
	PilotID          string    `json:"pilotId"`
	Severity         string    `json:"severity"`
	Code             string    `json:"code"`
	Details          string    `json:"details"`
	OccurredAt       time.Time `json:"occurredAt"`
	AdmissionBlocked bool      `json:"admissionBlocked"`
}

func (i PilotIncident) Validate() error {
	if i.IncidentID == "" || i.PilotID == "" || i.Severity == "" || i.Code == "" || i.Details == "" || i.OccurredAt.IsZero() || i.OccurredAt.Location() != time.UTC {
		return fmt.Errorf("pilot incident is incomplete")
	}
	return nil
}

type PilotReadModel struct {
	Pilot             PilotRecord        `json:"pilot"`
	Opportunities     []PilotOpportunity `json:"opportunities"`
	Metrics           PilotMetrics       `json:"metrics"`
	Incidents         []PilotIncident    `json:"incidents"`
	ActivePositions   []LifecycleRecord  `json:"activePositions,omitempty"`
	Exploratory       bool               `json:"exploratory"`
	NotFormalEvidence bool               `json:"notFormalEvidence"`
}
