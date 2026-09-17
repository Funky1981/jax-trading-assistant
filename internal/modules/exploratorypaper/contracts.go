// Package exploratorypaper defines the bounded, versioned contracts for the
// event-driven exploratory paper-trader loop. It deliberately does not submit
// broker orders; papertrading remains the only execution venue.
package exploratorypaper

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
	ExploratoryPaperMode = "EXPLORATORY_PAPER"
	FormalForwardMode    = "FORMAL_FORWARD_PAPER"
	ContractVersion      = "paper-01-thesis-v1"
	TraderModelVersion   = "event-driven-short-horizon-swing-v1"
)

var ErrApprovalBindingRequired = errors.New("exploratory position requires a validated approval binding")

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

// EvidenceAssessment is the reviewed, canonical evidence result supplied by
// the event/evidence projection. GenerateCandidate deliberately does not use
// the legacy caller-declared EvidenceQuality or Corroborated fields.
type EvidenceAssessment struct {
	Provider                string    `json:"provider"`
	PolicyVersion           string    `json:"policyVersion"`
	EvidenceSetFingerprint  string    `json:"evidenceSetFingerprint"`
	ReviewedAt              time.Time `json:"reviewedAt"`
	SourceBacked            bool      `json:"sourceBacked"`
	QualityState            string    `json:"qualityState"`
	QualityScore            float64   `json:"qualityScore"`
	RequiredQualityScore    float64   `json:"requiredQualityScore"`
	EvidenceReady           bool      `json:"evidenceReady"`
	EvidenceGateReady       bool      `json:"evidenceGateReady"`
	Corroborated            bool      `json:"corroborated"`
	Contradictory           bool      `json:"contradictory"`
	Unknown                 bool      `json:"unknown"`
	Stale                   bool      `json:"stale"`
	IssuerRelevant          bool      `json:"issuerRelevant"`
	InstrumentRelevant      bool      `json:"instrumentRelevant"`
	IndependentSourceGroups int       `json:"independentSourceGroups"`
}

func (a EvidenceAssessment) ValidateFor(evidence []EvidenceReference, issuerID, instrumentID string) error {
	if a.Provider != "candidate_evidence_scores" || strings.TrimSpace(a.PolicyVersion) == "" {
		return fmt.Errorf("candidate evidence must come from the canonical reviewed score projection")
	}
	if a.ReviewedAt.IsZero() || a.ReviewedAt.Location() != time.UTC {
		return fmt.Errorf("candidate evidence review timestamp must be UTC")
	}
	if !a.SourceBacked || a.Unknown || a.Stale || a.Contradictory || !a.IssuerRelevant || !a.InstrumentRelevant {
		return fmt.Errorf("candidate evidence is unknown, stale, contradictory, or not relevant")
	}
	if !a.EvidenceReady || !a.EvidenceGateReady || !a.Corroborated || a.IndependentSourceGroups < 2 || independentSourceCount(evidence) < 2 {
		return fmt.Errorf("candidate evidence corroboration gate is not ready")
	}
	if a.QualityState != "sufficient" || a.QualityScore < a.RequiredQualityScore || a.RequiredQualityScore <= 0 {
		return fmt.Errorf("candidate evidence quality threshold is not met")
	}
	if len(evidence) == 0 || a.EvidenceSetFingerprint != EvidenceSetFingerprint(evidence) {
		return fmt.Errorf("candidate evidence identity does not match the reviewed evidence set")
	}
	for _, item := range evidence {
		if item.EvidenceID == "" || item.SourceID == "" || item.SourceURL == "" || item.ObservedAt.IsZero() || item.ObservedAt.Location() != time.UTC {
			return fmt.Errorf("candidate evidence must remain source-backed and timestamped")
		}
	}
	if strings.TrimSpace(issuerID) == "" || strings.TrimSpace(instrumentID) == "" {
		return fmt.Errorf("candidate evidence relevance requires issuer and instrument")
	}
	return nil
}

func independentSourceCount(evidence []EvidenceReference) int {
	sources := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		if item.SourceID != "" {
			sources[item.SourceID] = struct{}{}
		}
	}
	return len(sources)
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

type FrozenThesis struct {
	Thesis          TradeThesis `json:"thesis"`
	ThesisHash      string      `json:"thesisHash"`
	EvidenceSetHash string      `json:"evidenceSetHash"`
	FrozenAt        time.Time   `json:"frozenAt"`
}

func FreezeThesis(thesis TradeThesis, at time.Time) (FrozenThesis, error) {
	if err := thesis.Validate(); err != nil {
		return FrozenThesis{}, err
	}
	if at.IsZero() || at.Location() != time.UTC {
		return FrozenThesis{}, fmt.Errorf("frozen thesis time must be UTC")
	}
	return FrozenThesis{Thesis: thesis, ThesisHash: ThesisContentHash(thesis), EvidenceSetHash: EvidenceSetFingerprint(append(append([]EvidenceReference{}, thesis.Evidence...), thesis.CounterEvidence...)), FrozenAt: at}, nil
}

func ThesisContentHash(thesis TradeThesis) string {
	data, _ := json.Marshal(thesis)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func EvidenceSetFingerprint(evidence []EvidenceReference) string {
	data, _ := json.Marshal(evidence)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
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
	ThesisContentHash      string         `json:"thesisContentHash"`
	EvidenceSetHash        string         `json:"evidenceSetHash"`
	BoundAt                time.Time      `json:"boundAt"`
}

func (b EntryBinding) Validate() error {
	if b.Mode != ExploratoryPaperMode || b.CandidateID == "" || b.ThesisID == "" || b.TraderModelVersion == "" || b.WorkflowID == "" || b.PaperIntentID == "" || b.ThesisContentHash == "" || b.EvidenceSetHash == "" {
		return fmt.Errorf("entry binding identity is incomplete")
	}
	if b.TraderModelVersion != TraderModelVersion {
		return fmt.Errorf("entry binding trader model version is unsupported")
	}
	if b.Environment != "PAPER" || b.ExecutionAuthority != "NONE" || b.BrokerExecutionAllowed || math.IsNaN(b.MaximumLeverage) || math.IsInf(b.MaximumLeverage, 0) || b.MaximumLeverage <= 0 || b.MaximumLeverage > 1 || b.BoundAt.IsZero() || b.BoundAt.Location() != time.UTC {
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
	EventID                string
	IssuerID               string
	InstrumentID           string
	EventCategory          string
	EventTimestamp         time.Time
	GeneratedAt            time.Time
	SourceURL              string
	Evidence               []EvidenceReference
	EvidenceQuality        float64
	Corroborated           bool
	ReviewedEvidence       *EvidenceAssessment
	CandidatePolicyVersion string
	CausalMechanism        string
	QuantContext           string
	RiskAssessment         string
	TechnicalConfirmation  string
	TechnicalOnly          bool
	Direction              Direction
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
	if in.ReviewedEvidence == nil {
		return CandidateResult{Decision: DecisionWatch, Reason: "candidate requires a canonical reviewed evidence assessment"}, nil
	}
	if err := in.ReviewedEvidence.ValidateFor(in.Evidence, in.IssuerID, in.InstrumentID); err != nil {
		return CandidateResult{Decision: DecisionWatch, Reason: err.Error()}, nil
	}
	if strings.TrimSpace(in.CandidatePolicyVersion) == "" {
		return CandidateResult{Decision: DecisionWatch, Reason: "candidate evidence policy version is required"}, nil
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
	PositionID        string            `json:"positionId"`
	Thesis            TradeThesis       `json:"thesis"`
	EntryAt           time.Time         `json:"entryAt"`
	EntryPrice        float64           `json:"entryPrice"`
	State             ThesisState       `json:"state"`
	OperationalState  ThesisState       `json:"operationalState"`
	Transitions       []Transition      `json:"transitions"`
	FrozenThesisHash  string            `json:"frozenThesisHash"`
	EvidenceSetHash   string            `json:"evidenceSetHash"`
	BindingHash       string            `json:"bindingHash"`
	ProcessedEvidence map[string]string `json:"processedEvidence"`
	ClosedAt          *time.Time        `json:"closedAt,omitempty"`
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

// OpenApprovedPosition is the only constructor that can create an exploratory
// position. The binding is the immutable bridge from candidate approval and
// the existing paper workflow to this domain position.
func OpenApprovedPosition(id string, thesis TradeThesis, binding EntryBinding, at time.Time, price float64) (Position, error) {
	if err := thesis.Validate(); err != nil {
		return Position{}, err
	}
	if err := binding.Validate(); err != nil {
		return Position{}, fmt.Errorf("entry binding: %w", err)
	}
	if binding.ThesisID != thesis.ThesisID {
		return Position{}, fmt.Errorf("entry binding thesis does not match position thesis")
	}
	frozen, err := FreezeThesis(thesis, at)
	if err != nil {
		return Position{}, err
	}
	if binding.ThesisContentHash != frozen.ThesisHash || binding.EvidenceSetHash != frozen.EvidenceSetHash {
		return Position{}, fmt.Errorf("entry binding does not match frozen thesis provenance")
	}
	if binding.PolicyVersions.ThesisContract != thesis.ContractVersion || binding.PolicyVersions.EntryPolicy != thesis.EntryPolicyVersion {
		return Position{}, fmt.Errorf("entry binding policy versions do not match thesis")
	}
	if binding.BoundAt.After(at) {
		return Position{}, fmt.Errorf("entry binding must precede or equal entry time")
	}
	if strings.TrimSpace(id) == "" || at.IsZero() || at.Location() != time.UTC || math.IsNaN(price) || math.IsInf(price, 0) || price <= 0 {
		return Position{}, fmt.Errorf("position identity, entry time, and positive price are required")
	}
	return Position{PositionID: id, Thesis: thesis, EntryAt: at, EntryPrice: price, State: StateValid, OperationalState: StateValid, FrozenThesisHash: frozen.ThesisHash, EvidenceSetHash: frozen.EvidenceSetHash, BindingHash: bindingIdentity(binding), ProcessedEvidence: map[string]string{}}, nil
}

func (p Position) VerifyFrozenIdentity() error {
	if p.FrozenThesisHash == "" || p.FrozenThesisHash != ThesisContentHash(p.Thesis) {
		return fmt.Errorf("frozen thesis identity mismatch")
	}
	evidenceHash := EvidenceSetFingerprint(append(append([]EvidenceReference{}, p.Thesis.Evidence...), p.Thesis.CounterEvidence...))
	if p.EvidenceSetHash == "" || p.EvidenceSetHash != evidenceHash {
		return fmt.Errorf("frozen evidence identity mismatch")
	}
	return nil
}

func (p Position) VerifyApprovalBinding(binding EntryBinding) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	if p.BindingHash == "" || p.BindingHash != bindingIdentity(binding) {
		return fmt.Errorf("frozen approval binding identity mismatch")
	}
	if binding.ThesisID != p.Thesis.ThesisID || binding.ThesisContentHash != p.FrozenThesisHash || binding.EvidenceSetHash != p.EvidenceSetHash {
		return fmt.Errorf("approval binding does not match frozen position")
	}
	return nil
}

// OpenPosition is retained only as a safe compatibility guard. It cannot
// create an exploratory position without the candidate/workflow/paper-intent
// approval binding; callers must use OpenApprovedPosition.
func OpenPosition(id string, thesis TradeThesis, at time.Time, price float64) (Position, error) {
	return Position{}, fmt.Errorf("%w: use OpenApprovedPosition", ErrApprovalBindingRequired)
}

func (p *Position) Reassess(e RelevantEvidence, at time.Time, policyVersion string) error {
	if p.State == StateClosed {
		return fmt.Errorf("closed position cannot be reassessed")
	}
	if e.Reference.EvidenceID == "" || e.Reference.SourceID == "" || e.Reference.SourceURL == "" || at.IsZero() || at.Location() != time.UTC || strings.TrimSpace(policyVersion) == "" {
		return fmt.Errorf("reassessment requires timestamped source evidence")
	}
	if e.Signal != EvidenceIrrelevant && e.Signal != EvidenceStrengthens && e.Signal != EvidenceContradicts && e.Signal != EvidenceInvalidates {
		return fmt.Errorf("unsupported evidence signal %q", e.Signal)
	}
	if err := p.VerifyFrozenIdentity(); err != nil {
		return err
	}
	if p.ProcessedEvidence == nil {
		p.ProcessedEvidence = map[string]string{}
	}
	fingerprint := relevantEvidenceFingerprint(e)
	if prior, ok := p.ProcessedEvidence[e.Reference.EvidenceID]; ok {
		if prior != fingerprint {
			return fmt.Errorf("processed evidence identity conflicts with prior content")
		}
		return nil
	}
	p.ProcessedEvidence[e.Reference.EvidenceID] = fingerprint
	if e.IssuerID != p.Thesis.IssuerID || e.InstrumentID != p.Thesis.InstrumentID {
		return nil
	}
	if e.Signal == EvidenceIrrelevant {
		return nil
	}
	var next ThesisState
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
	if at.IsZero() || at.Location() != time.UTC {
		return fmt.Errorf("close time is required")
	}
	p.State, p.OperationalState, p.ClosedAt = StateClosed, StateClosed, &at
	return nil
}

func relevantEvidenceFingerprint(e RelevantEvidence) string {
	data, _ := json.Marshal(struct {
		Reference    EvidenceReference
		IssuerID     string
		InstrumentID string
		Signal       EvidenceSignal
		Reason       string
	}{e.Reference, e.IssuerID, e.InstrumentID, e.Signal, e.Reason})
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
