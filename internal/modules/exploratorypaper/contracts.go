// Package exploratorypaper defines the bounded, versioned contracts for the
// event-driven exploratory paper-trader loop. It deliberately does not submit
// broker orders; papertrading remains the only execution venue.
package exploratorypaper

import (
	"fmt"
	"strings"
	"time"
)

const (
	ExploratoryPaperMode = "EXPLORATORY_PAPER"
	FormalForwardMode    = "FORMAL_FORWARD_PAPER"
	ContractVersion      = "paper-01-thesis-v1"
	TraderModelVersion   = "event-driven-short-horizon-swing-v1"
)

type Direction string

const (
	DirectionLong  Direction = "LONG"
	DirectionShort Direction = "SHORT"
)

type ThesisState string

const (
	StateValid                     ThesisState = "VALID"
	StateStrengthened              ThesisState = "STRENGTHENED"
	StateWeakened                  ThesisState = "WEAKENED"
	StateInvalidated               ThesisState = "INVALIDATED"
	StateReviewRequired            ThesisState = "REVIEW_REQUIRED"
	StateExitRecommended           ThesisState = "EXIT_RECOMMENDED"
	StateExitAtNextTradableSession ThesisState = "EXIT_AT_NEXT_TRADABLE_SESSION"
	StateClosed                    ThesisState = "CLOSED"
)

type EvidenceReference struct {
	EvidenceID string    `json:"evidenceId"`
	SourceID   string    `json:"sourceId"`
	SourceURL  string    `json:"sourceUrl"`
	Quality    string    `json:"quality"`
	ObservedAt time.Time `json:"observedAt"`
}

type TradeThesis struct {
	ThesisID                string              `json:"thesisId"`
	ContractVersion         string              `json:"contractVersion"`
	Mode                    string              `json:"mode"`
	EventID                 string              `json:"eventId"`
	IssuerID                string              `json:"issuerId"`
	InstrumentID            string              `json:"instrumentId"`
	Direction               Direction           `json:"direction"`
	Exposure                string              `json:"exposure"`
	EventCategory           string              `json:"eventCategory"`
	EventTimestamp          time.Time           `json:"eventTimestamp"`
	CandidateGeneratedAt    time.Time           `json:"candidateGeneratedAt"`
	CausalReason            string              `json:"causalReason"`
	Evidence                []EvidenceReference `json:"evidence"`
	ExpectedMechanism       string              `json:"expectedMechanism"`
	ExpectedHorizonSessions int                 `json:"expectedHorizonSessions"`
	EntryRationale          string              `json:"entryRationale"`
	QuantTechnicalContext   string              `json:"quantTechnicalContext"`
	RiskAssessment          string              `json:"riskAssessment"`
	EntryPolicyVersion      string              `json:"entryPolicyVersion"`
	ProtectiveStop          float64             `json:"protectiveStop"`
	Target                  float64             `json:"target"`
	InvalidationConditions  []string            `json:"invalidationConditions"`
	CounterEvidence         []EvidenceReference `json:"counterEvidence"`
	Confidence              float64             `json:"confidence"`
	Uncertainty             string              `json:"uncertainty"`
	PolicyVersion           string              `json:"policyVersion"`
	CreatedAt               time.Time           `json:"createdAt"`
}

// EntryBinding is the immutable bridge between an exploratory thesis and the
// existing approved paper workflow. The actual order/fill/ledger remains owned
// by internal/modules/papertrading.
type EntryBinding struct {
	Mode                   string         `json:"mode"`
	CandidateID            string         `json:"candidateId"`
	ThesisID               string         `json:"thesisId"`
	TraderModelVersion     string         `json:"traderModelVersion"`
	WorkflowID             string         `json:"workflowId"`
	PaperIntentID          string         `json:"paperIntentId"`
	Environment            string         `json:"environment"`
	ExecutionAuthority     string         `json:"executionAuthority"`
	BrokerExecutionAllowed bool           `json:"brokerExecutionAllowed"`
	MaximumLeverage        float64        `json:"maximumLeverage"`
	PolicyVersions         PolicyVersions `json:"policyVersions"`
	BoundAt                time.Time      `json:"boundAt"`
}

func (b EntryBinding) Validate() error {
	if b.Mode != ExploratoryPaperMode || b.CandidateID == "" || b.ThesisID == "" || b.TraderModelVersion == "" || b.WorkflowID == "" || b.PaperIntentID == "" {
		return fmt.Errorf("entry binding identity is incomplete")
	}
	if b.Environment != "PAPER" || b.ExecutionAuthority != "NONE" || b.BrokerExecutionAllowed || b.MaximumLeverage <= 0 || b.MaximumLeverage > 1 || b.BoundAt.IsZero() {
		return fmt.Errorf("entry binding violates paper safety boundary")
	}
	return b.PolicyVersions.Validate()
}

func (t TradeThesis) Validate() error {
	if t.Mode != ExploratoryPaperMode {
		return fmt.Errorf("thesis mode must be %s", ExploratoryPaperMode)
	}
	if t.ContractVersion == "" || t.ThesisID == "" || t.EventID == "" || t.IssuerID == "" || t.InstrumentID == "" {
		return fmt.Errorf("thesis identity and contract version are required")
	}
	if t.Direction != DirectionLong && t.Direction != DirectionShort {
		return fmt.Errorf("unsupported direction %q", t.Direction)
	}
	if strings.TrimSpace(t.Exposure) == "" || strings.TrimSpace(t.EventCategory) == "" || strings.TrimSpace(t.CausalReason) == "" || strings.TrimSpace(t.ExpectedMechanism) == "" {
		return fmt.Errorf("event, causal reason, and mechanism are required")
	}
	if t.EventTimestamp.IsZero() || t.CandidateGeneratedAt.IsZero() || t.CreatedAt.IsZero() {
		return fmt.Errorf("event, candidate, and creation timestamps are required")
	}
	if t.ExpectedHorizonSessions < 1 || t.ExpectedHorizonSessions > 5 {
		return fmt.Errorf("expected horizon must be 1-5 trading sessions")
	}
	if len(t.Evidence) == 0 {
		return fmt.Errorf("at least one evidence reference is required")
	}
	for _, e := range append(append([]EvidenceReference{}, t.Evidence...), t.CounterEvidence...) {
		if e.EvidenceID == "" || e.SourceID == "" || e.SourceURL == "" || e.Quality == "" || e.ObservedAt.IsZero() {
			return fmt.Errorf("evidence references must be source-backed and timestamped")
		}
	}
	if strings.TrimSpace(t.EntryRationale) == "" || strings.TrimSpace(t.QuantTechnicalContext) == "" || strings.TrimSpace(t.RiskAssessment) == "" || t.EntryPolicyVersion == "" || t.PolicyVersion == "" {
		return fmt.Errorf("entry, quant, risk, and policy provenance are required")
	}
	if t.ProtectiveStop <= 0 || t.Target <= 0 {
		return fmt.Errorf("protective stop and target must be positive")
	}
	if len(t.InvalidationConditions) == 0 || strings.TrimSpace(t.Uncertainty) == "" {
		return fmt.Errorf("invalidation conditions and uncertainty are required")
	}
	if t.Confidence < 0 || t.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

type CandidateDecision string

const (
	DecisionWatch     CandidateDecision = "WATCH"
	DecisionNoTrade   CandidateDecision = "NO_TRADE"
	DecisionCandidate CandidateDecision = "CANDIDATE"
)

type CandidateInput struct {
	EventID               string
	IssuerID              string
	InstrumentID          string
	EventCategory         string
	EventTimestamp        time.Time
	GeneratedAt           time.Time
	SourceURL             string
	Evidence              []EvidenceReference
	EvidenceQuality       float64
	Corroborated          bool
	CausalMechanism       string
	QuantContext          string
	RiskAssessment        string
	TechnicalConfirmation string
	TechnicalOnly         bool
	Direction             Direction
}

type CandidateResult struct {
	Decision   CandidateDecision `json:"decision"`
	Reason     string            `json:"reason"`
	Provenance []string          `json:"provenance"`
}

func GenerateCandidate(in CandidateInput) (CandidateResult, error) {
	if in.EventID == "" || in.IssuerID == "" || in.InstrumentID == "" || in.EventCategory == "" || in.SourceURL == "" || in.EventTimestamp.IsZero() || in.GeneratedAt.IsZero() {
		return CandidateResult{}, fmt.Errorf("event, resolved asset, source, and timestamps are required")
	}
	if in.TechnicalOnly || strings.TrimSpace(in.CausalMechanism) == "" {
		return CandidateResult{Decision: DecisionNoTrade, Reason: "technical confirmation cannot be the sole catalyst"}, nil
	}
	if len(in.Evidence) == 0 || !in.Corroborated || in.EvidenceQuality <= 0 {
		return CandidateResult{Decision: DecisionWatch, Reason: "candidate requires material corroborated evidence with quality"}, nil
	}
	if strings.TrimSpace(in.QuantContext) == "" || strings.TrimSpace(in.RiskAssessment) == "" || strings.TrimSpace(in.TechnicalConfirmation) == "" {
		return CandidateResult{Decision: DecisionWatch, Reason: "candidate requires quant, risk, and technical context"}, nil
	}
	if in.Direction != DirectionLong && in.Direction != DirectionShort {
		return CandidateResult{Decision: DecisionNoTrade, Reason: "direction is unresolved"}, nil
	}
	return CandidateResult{Decision: DecisionCandidate, Reason: "event, evidence, resolved asset, causal mechanism, quant, technical, and risk gates passed", Provenance: []string{in.EventID, in.IssuerID, in.InstrumentID, in.SourceURL}}, nil
}

type Transition struct {
	At              time.Time           `json:"at"`
	TriggerEvidence []EvidenceReference `json:"triggerEvidence"`
	Previous        ThesisState         `json:"previous"`
	Next            ThesisState         `json:"next"`
	Reason          string              `json:"reason"`
	Provenance      []string            `json:"provenance"`
	PolicyVersion   string              `json:"policyVersion"`
}

type Position struct {
	PositionID       string       `json:"positionId"`
	Thesis           TradeThesis  `json:"thesis"`
	EntryAt          time.Time    `json:"entryAt"`
	EntryPrice       float64      `json:"entryPrice"`
	State            ThesisState  `json:"state"`
	OperationalState ThesisState  `json:"operationalState"`
	Transitions      []Transition `json:"transitions"`
	ClosedAt         *time.Time   `json:"closedAt,omitempty"`
}

type EvidenceSignal string

const (
	EvidenceIrrelevant  EvidenceSignal = "IRRELEVANT"
	EvidenceStrengthens EvidenceSignal = "STRENGTHENS"
	EvidenceContradicts EvidenceSignal = "CONTRADICTS"
	EvidenceInvalidates EvidenceSignal = "INVALIDATES"
)

type RelevantEvidence struct {
	Reference    EvidenceReference
	IssuerID     string
	InstrumentID string
	Signal       EvidenceSignal
	Reason       string
}

func OpenPosition(id string, thesis TradeThesis, at time.Time, price float64) (Position, error) {
	if err := thesis.Validate(); err != nil {
		return Position{}, err
	}
	if id == "" || at.IsZero() || price <= 0 {
		return Position{}, fmt.Errorf("position identity, entry time, and positive price are required")
	}
	return Position{PositionID: id, Thesis: thesis, EntryAt: at, EntryPrice: price, State: StateValid, OperationalState: StateValid}, nil
}

func (p *Position) Reassess(e RelevantEvidence, at time.Time, policyVersion string) error {
	if p.State == StateClosed {
		return fmt.Errorf("closed position cannot be reassessed")
	}
	if e.Reference.EvidenceID == "" || e.Reference.SourceID == "" || e.Reference.SourceURL == "" || at.IsZero() {
		return fmt.Errorf("reassessment requires timestamped source evidence")
	}
	if e.IssuerID != p.Thesis.IssuerID || e.InstrumentID != p.Thesis.InstrumentID {
		return nil
	}
	if e.Signal == EvidenceIrrelevant {
		return nil
	}
	next := p.State
	switch e.Signal {
	case EvidenceStrengthens:
		next = StateStrengthened
	case EvidenceContradicts:
		next = StateWeakened
	case EvidenceInvalidates:
		next = StateInvalidated
	default:
		return fmt.Errorf("unsupported evidence signal %q", e.Signal)
	}
	if next == p.State && next != StateInvalidated {
		return nil
	}
	if e.Reason == "" {
		return fmt.Errorf("state transition reason is required")
	}
	tr := Transition{At: at, TriggerEvidence: []EvidenceReference{e.Reference}, Previous: p.State, Next: next, Reason: e.Reason, Provenance: []string{e.Reference.EvidenceID, e.Reference.SourceID}, PolicyVersion: policyVersion}
	p.Transitions = append(p.Transitions, tr)
	p.State = next
	if next == StateInvalidated {
		p.OperationalState = StateExitRecommended
	}
	return nil
}

func (p *Position) Close(at time.Time) error {
	if p.State == StateClosed {
		return fmt.Errorf("position already closed")
	}
	if at.IsZero() {
		return fmt.Errorf("close time is required")
	}
	p.State, p.OperationalState, p.ClosedAt = StateClosed, StateClosed, &at
	return nil
}
