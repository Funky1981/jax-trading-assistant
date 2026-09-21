package exploratorypaper

// runtime.go is the single PAPER-01 orchestration seam. It composes the
// canonical candidate/evidence projection, workflow approval, paper venue,
// existing paper ledger, and Postgres exploratory store. It intentionally does
// not import the broker execution package.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
)

var (
	ErrRuntimeMode             = errors.New("exploratory paper runtime requires PAPER mode")
	ErrHumanApprovalPending    = errors.New("exploratory paper exit requires explicit human approval")
	ErrReviewInputsUnavailable = errors.New("exploratory paper review inputs are unavailable")
	ErrCanonicalProjection     = errors.New("canonical evidence projection is unavailable or invalid")
	ErrRuntimeIdentityConflict = errors.New("exploratory paper runtime identity conflict")
)

// EntryRequest is produced by the existing accepted event-driven path. The
// evidence assessment must be loaded from candidate_evidence_scores by the
// source adapter; a generic caller cannot promote a candidate by setting a
// boolean or score on this request.
type EntryRequest struct {
	CandidateID  string
	Candidate    CandidateInput
	Evidence     EvidenceAssessment
	Thesis       TradeThesis
	RiskDecision portfoliorisk.RiskDecision
	Approval     ApprovalSnapshot
	PositionID   string
	Quantity     float64
	Tick         papertrading.MarketTick
	Venue        papertrading.CapabilityContract
	CostModel    papertrading.CostModel
	Ledger       papertrading.PaperAccount
	Calendar     SessionCalendar
	Now          time.Time
}

type EntrySource interface {
	LoadApprovedEntries(context.Context) ([]EntryRequest, error)
}

type ReviewObservation struct {
	Tick                 papertrading.MarketTick
	Price                float64
	PriceSource          string
	Evidence             []RelevantEvidence
	Path                 []PriceObservation
	Calendar             SessionCalendar
	Ledger               papertrading.PaperAccount
	Excursion            ExcursionCoverage
	QuoteMode            string
	LiquidityMode        string
	ActualQuoteAvailable bool
	ObservedAt           time.Time
	ReceivedAt           time.Time
	ThesisInvalidated    bool
	RiskKill             bool
	ManualOperator       bool
}

type ReviewSource interface {
	LoadReviewObservation(context.Context, LifecycleRecord, Review) (ReviewObservation, error)
}

type ExitApprovalSource interface {
	ApproveExit(context.Context, LifecycleRecord, ExitDecision, ReviewObservation) (ApprovalSnapshot, error)
}

type RuntimeStore interface {
	FindByCandidate(context.Context, string) (LifecycleRecord, bool, error)
	SaveApprovedEntry(context.Context, string, TradeThesis, EntryBinding, ApprovalSnapshot, Position, EntrySnapshot) (string, error)
	PersistReviewSchedule(context.Context, string, ReviewSchedule) error
	MarkEntryProcessed(context.Context, string) error
	ListDueReviews(context.Context, time.Time) ([]ReviewRecord, error)
	RecordReviewUnavailable(context.Context, string, int, time.Time, string) error
	ApplyEvidence(context.Context, string, RelevantEvidence, time.Time, string) (Position, error)
	PersistCheckpoint(context.Context, Checkpoint) error
	PersistExitRecommendation(context.Context, string, ExitDecision, Review) error
	PersistExitApproval(context.Context, string, Review, ApprovalSnapshot) error
	PersistExitRejection(context.Context, string, Review, ApprovalSnapshot) error
	PersistOutcome(context.Context, string, ExitSnapshot, Outcome) error
}

type Runtime struct {
	Mode         string
	Store        RuntimeStore
	Entries      EntrySource
	Reviews      ReviewSource
	ExitApprover ExitApprovalSource
	Venue        *papertrading.PaperVenue
	CostModel    papertrading.CostModel
	Now          func() time.Time
}

func (r *Runtime) validate() error {
	if r == nil || strings.ToUpper(strings.TrimSpace(r.Mode)) != "PAPER" {
		return ErrRuntimeMode
	}
	if r.Store == nil || r.Entries == nil || r.Reviews == nil || r.Venue == nil {
		return fmt.Errorf("%w: runtime dependencies are incomplete", ErrFailedClosed)
	}
	if r.Now == nil {
		r.Now = func() time.Time { return time.Now().UTC() }
	}
	return nil
}

// RunEntryCycle consumes approved, canonical event candidates. All venue and
// ledger effects happen only after candidate, risk, approval, session, and
// safety validation. A previously persisted candidate is a successful retry.
func (r *Runtime) RunEntryCycle(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}
	entries, err := r.Entries.LoadApprovedEntries(ctx)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := r.processEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) processEntry(ctx context.Context, entry EntryRequest) error {
	if strings.TrimSpace(entry.CandidateID) == "" {
		return fmt.Errorf("%w: candidate identity is incomplete", ErrCanonicalProjection)
	}
	if existing, found, err := r.Store.FindByCandidate(ctx, entry.CandidateID); err != nil {
		return err
	} else if found {
		if existing.Binding.WorkflowID != entry.Approval.Workflow.WorkflowID || existing.Binding.PaperIntentID != entry.Approval.PaperIntent.IntentID {
			return ErrRuntimeIdentityConflict
		}
		return r.Store.MarkEntryProcessed(ctx, entry.CandidateID)
	}

	input := entry.Candidate
	input.ReviewedEvidence = &entry.Evidence
	result, err := GenerateCandidate(input)
	if err != nil {
		return fmt.Errorf("candidate projection: %w", err)
	}
	if result.Decision != DecisionCandidate {
		return r.Store.MarkEntryProcessed(ctx, entry.CandidateID)
	}
	if entry.RiskDecision.Outcome != portfoliorisk.DecisionAccept && entry.RiskDecision.Outcome != portfoliorisk.DecisionAmend {
		return fmt.Errorf("%w: risk decision is not accepted", ErrFailedClosed)
	}
	if err := entry.RiskDecision.Validate(); err != nil {
		return fmt.Errorf("%w: risk decision: %v", ErrFailedClosed, err)
	}
	if entry.RiskDecision.ExecutionAuthority != "NONE" || entry.RiskDecision.ResultingValue <= 0 {
		return fmt.Errorf("%w: risk decision is not paper-safe", ErrFailedClosed)
	}
	if err := entry.Thesis.Validate(); err != nil {
		return fmt.Errorf("%w: thesis: %v", ErrFailedClosed, err)
	}
	if entry.Thesis.EventID != input.EventID || entry.Thesis.IssuerID != input.IssuerID || entry.Thesis.InstrumentID != input.InstrumentID {
		return ErrRuntimeIdentityConflict
	}
	if err := entry.Approval.Workflow.Validate(); err != nil {
		return fmt.Errorf("%w: workflow: %v", ErrFailedClosed, err)
	}
	if entry.Approval.Workflow.RiskDecisionID != entry.RiskDecision.DecisionID || entry.Approval.Workflow.State != workflow.StatePaperIntentCreated {
		return fmt.Errorf("%w: approval is not bound to the accepted risk decision", ErrFailedClosed)
	}
	if err := entry.Approval.PaperIntent.Validate(); err != nil {
		return fmt.Errorf("%w: paper intent: %v", ErrFailedClosed, err)
	}
	if entry.Now.IsZero() {
		entry.Now = r.Now().UTC()
	}
	if entry.Now.Location() != time.UTC {
		return fmt.Errorf("%w: entry time must be UTC", ErrFailedClosed)
	}
	if err := entry.Calendar.Validate(); err != nil {
		return fmt.Errorf("%w: session calendar: %v", ErrFailedClosed, err)
	}
	if state, err := entry.Calendar.SessionState(entry.Tick.Timestamp); err != nil || state != SessionOpen || string(entry.Tick.Session) != string(state) {
		if err != nil {
			return fmt.Errorf("%w: entry session: %v", ErrFailedClosed, err)
		}
		return fmt.Errorf("%w: entry is not in a known open session", ErrFailedClosed)
	}
	if err := entry.Tick.Validate(entry.Venue.MaxQuoteAge); err != nil {
		return fmt.Errorf("%w: entry market observation: %v", ErrFailedClosed, err)
	}
	if entry.Venue.Environment != papertrading.EnvironmentPaper || entry.Venue.ExecutionAuthority != papertrading.ExecutionAuthorityNone {
		return ErrRuntimeMode
	}
	ledger, err := papertrading.RestorePaperLedger(entry.Ledger)
	if err != nil {
		return fmt.Errorf("%w: restore paper ledger: %v", ErrFailedClosed, err)
	}
	createdAt := entry.Now
	if createdAt.After(entry.Tick.Timestamp) {
		createdAt = entry.Tick.Timestamp
	}
	order, err := r.Venue.Submit(papertrading.CreateOrderRequest{Workflow: entry.Approval.Workflow, PaperIntent: entry.Approval.PaperIntent, Venue: entry.Venue, CostModel: entry.CostModel, InstrumentID: entry.Thesis.InstrumentID, Quantity: entry.Quantity, ReferencePrice: entry.Tick.Last, OrderType: papertrading.OrderMarket, CreatedAt: createdAt, IdempotencyKey: entry.Approval.PaperIntent.IntentID})
	if err != nil {
		return err
	}
	fills, err := r.Venue.ProcessTickWithSafety(entry.Tick, false)
	if err != nil || len(fills) != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: entry did not produce exactly one simulated fill", ErrFailedClosed)
	}
	account, err := ledger.ApplyFill(fills[0])
	if err != nil {
		return err
	}
	if result := papertrading.Reconcile(papertrading.ReconciliationInput{VenueSnapshot: r.Venue.Snapshot(), Account: account}, entry.Now.UTC()); result.Status != papertrading.ReconciliationClean {
		return fmt.Errorf("%w: entry ledger reconciliation failed: %v", ErrFailedClosed, result.ReasonCodes)
	}
	position, err := OpenApprovedPosition(entry.PositionID, entry.Thesis, EntryBindingFrom(entry), fills[0].FilledAt, fills[0].Price)
	if err != nil {
		return err
	}
	lifecycleID, err := r.Store.SaveApprovedEntry(ctx, entry.CandidateID, entry.Thesis, EntryBindingFrom(entry), entry.Approval, position, EntrySnapshot{Order: order, Fill: fills[0], Ledger: account})
	if err != nil {
		return err
	}
	schedule, err := BuildReviewSchedule(position, entry.Calendar)
	if err != nil {
		return err
	}
	if err := r.Store.PersistReviewSchedule(ctx, position.PositionID, schedule); err != nil {
		return err
	}
	if err := r.Store.MarkEntryProcessed(ctx, entry.CandidateID); err != nil {
		return err
	}
	_ = lifecycleID
	return nil
}

// EntryBindingFrom binds only the canonical approved-entry facts. It is kept
// here so all runtime-created positions use the same bridge as direct tests.
func EntryBindingFrom(entry EntryRequest) EntryBinding {
	frozen, _ := FreezeThesis(entry.Thesis, entry.Approval.Workflow.UpdatedAt)
	return EntryBinding{Mode: ExploratoryPaperMode, CandidateID: entry.CandidateID, ThesisID: entry.Thesis.ThesisID, TraderModelVersion: TraderModelVersion, WorkflowID: entry.Approval.Workflow.WorkflowID, PaperIntentID: entry.Approval.PaperIntent.IntentID, Environment: "PAPER", ExecutionAuthority: "NONE", BrokerExecutionAllowed: false, MaximumLeverage: 1, PolicyVersions: PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: entry.Thesis.ContractVersion, CandidatePolicy: entry.CandidatePolicyVersion(), RiskPolicy: entry.Thesis.PolicyVersion, EntryPolicy: entry.Thesis.EntryPolicyVersion, ExitPolicy: "exit-policy-v1", CostModel: entry.CostModel.ModelID}, ThesisContentHash: frozen.ThesisHash, EvidenceSetHash: frozen.EvidenceSetHash, BoundAt: entry.Approval.Workflow.UpdatedAt}
}

func (e EntryRequest) CandidatePolicyVersion() string {
	if e.Candidate.CandidatePolicyVersion != "" {
		return e.Candidate.CandidatePolicyVersion
	}
	return e.Evidence.PolicyVersion
}

func (r *Runtime) RunDueReviews(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}
	reviews, err := r.Store.ListDueReviews(ctx, r.Now().UTC())
	if err != nil {
		return err
	}
	for _, item := range reviews {
		if err := r.processReview(ctx, item); err != nil {
			if errors.Is(err, ErrReviewInputsUnavailable) || errors.Is(err, ErrHumanApprovalPending) {
				continue
			}
			return err
		}
	}
	return nil
}

func (r *Runtime) processReview(ctx context.Context, item ReviewRecord) error {
	observation, err := r.Reviews.LoadReviewObservation(ctx, item.Lifecycle, item.Review)
	if err != nil {
		_ = r.Store.RecordReviewUnavailable(ctx, item.Review.PositionID, item.Review.SessionNumber, r.Now().UTC(), err.Error())
		return ErrReviewInputsUnavailable
	}
	if observation.Price <= 0 || observation.PriceSource == "" || len(observation.Evidence) == 0 {
		_ = r.Store.RecordReviewUnavailable(ctx, item.Review.PositionID, item.Review.SessionNumber, r.Now().UTC(), "required evidence or market observation was unavailable")
		return ErrReviewInputsUnavailable
	}
	position := item.Lifecycle.Position
	for _, evidence := range observation.Evidence {
		position, err = r.Store.ApplyEvidence(ctx, position.PositionID, evidence, observation.Tick.Timestamp, item.Lifecycle.Binding.PolicyVersions.ExitPolicy)
		if err != nil {
			return err
		}
	}
	entryCosts := item.Lifecycle.EntryFill.Costs.Commission
	direction := 1.0
	if position.Thesis.Direction == DirectionShort {
		direction = -1
	}
	gross := (observation.Price - position.EntryPrice) * item.Lifecycle.EntryFill.Quantity * direction
	denominator := position.EntryPrice * item.Lifecycle.EntryFill.Quantity
	checkpoint := Checkpoint{PositionID: position.PositionID, ThesisID: position.Thesis.ThesisID, Sessions: item.Review.SessionNumber, At: observation.Tick.Timestamp, Price: observation.Price, PriceSource: observation.PriceSource, EvidenceReviewID: item.Review.ReviewID, GrossPnL: gross, NetPnL: gross - entryCosts, NetReturn: (gross - entryCosts) / denominator, DataQuality: "COMPLETE", CommissionCost: entryCosts, SpreadCost: item.Lifecycle.EntryFill.Costs.SpreadCost, SlippageCost: item.Lifecycle.EntryFill.Costs.SlippageCost, AccountingVersion: EconomicAccountingVersion, QuoteMode: observation.QuoteMode, LiquidityMode: observation.LiquidityMode, ActualQuoteAvailable: observation.ActualQuoteAvailable, ObservedAt: observation.ObservedAt, ReceivedAt: observation.ReceivedAt}
	if err := r.Store.PersistCheckpoint(ctx, checkpoint); err != nil {
		return err
	}
	decision, err := EvaluateExit(position, ExitInput{Now: observation.Tick.Timestamp, Session: MarketSession(observation.Tick.Session), Bid: observation.Tick.Bid, Ask: observation.Tick.Ask, ThesisInvalidated: observation.ThesisInvalidated, RiskKill: observation.RiskKill, ManualOperator: observation.ManualOperator, Calendar: observation.Calendar})
	if err != nil || decision.Action == NoExit {
		return err
	}
	if item.Review.ExitApproval == nil {
		if err := r.Store.PersistExitRecommendation(ctx, item.Lifecycle.Position.PositionID, decision, item.Review); err != nil {
			return err
		}
		if r.ExitApprover == nil {
			return ErrHumanApprovalPending
		}
		approval, approvalErr := r.ExitApprover.ApproveExit(ctx, item.Lifecycle, decision, observation)
		if approvalErr != nil {
			return ErrHumanApprovalPending
		}
		if approval.Workflow.State == workflow.StateHumanRejected {
			item.Review.ExitRecommendationID = exitRecommendationIdentity(item.Review, decision)
			item.Review.ExitAction = string(decision.Action)
			item.Review.ExitReason = string(decision.Reason)
			if err := r.Store.PersistExitRejection(ctx, item.Lifecycle.Position.PositionID, item.Review, approval); err != nil {
				return err
			}
			return nil
		}
		if err := approval.Workflow.Validate(); err != nil || approval.Workflow.State != workflow.StatePaperIntentCreated {
			return fmt.Errorf("%w: exit approval is not a human-approved paper intent", ErrFailedClosed)
		}
		item.Review.ExitRecommendationID = exitRecommendationIdentity(item.Review, decision)
		item.Review.ExitAction = string(decision.Action)
		item.Review.ExitReason = string(decision.Reason)
		if err := r.Store.PersistExitApproval(ctx, item.Lifecycle.Position.PositionID, item.Review, approval); err != nil {
			return err
		}
		item.Review.ExitApproval = &approval
	}
	approval := *item.Review.ExitApproval
	if err := approval.Workflow.Validate(); err != nil || approval.Workflow.State != workflow.StatePaperIntentCreated {
		return fmt.Errorf("%w: exit approval is not a human-approved paper intent", ErrFailedClosed)
	}
	ledger, err := papertrading.RestorePaperLedger(observation.Ledger)
	if err != nil {
		return err
	}
	costModel := r.CostModel
	if costModel.ModelID == "" {
		costModel = papertrading.DefaultCostModel()
	}
	createdAt := observation.Tick.Timestamp.Add(-costModel.Latency)
	if !createdAt.Before(observation.Tick.Timestamp) {
		createdAt = observation.Tick.Timestamp
	}
	order, err := r.Venue.Submit(papertrading.CreateOrderRequest{Workflow: approval.Workflow, PaperIntent: approval.PaperIntent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costModel, InstrumentID: position.Thesis.InstrumentID, Quantity: item.Lifecycle.EntryFill.Quantity, ReferencePrice: observation.Price, OrderType: papertrading.OrderMarket, CreatedAt: createdAt, IdempotencyKey: approval.PaperIntent.IntentID})
	if err != nil {
		return err
	}
	fills, err := r.Venue.ProcessTickWithSafety(observation.Tick, false)
	if err != nil || len(fills) != 1 {
		return fmt.Errorf("%w: exit did not produce exactly one simulated fill", ErrFailedClosed)
	}
	account, err := ledger.ApplyFill(fills[0])
	if err != nil {
		return err
	}
	reconciliationAt := observation.ReceivedAt
	if reconciliationAt.IsZero() {
		reconciliationAt = observation.Tick.ReceivedAt
	}
	if reconciliationAt.IsZero() {
		reconciliationAt = observation.Tick.Timestamp
	}
	if result := papertrading.Reconcile(papertrading.ReconciliationInput{VenueSnapshot: r.Venue.Snapshot(), Account: account}, reconciliationAt.UTC()); result.Status != papertrading.ReconciliationClean {
		return fmt.Errorf("%w: exit ledger reconciliation failed: %v", ErrFailedClosed, result.ReasonCodes)
	}
	outcome, err := BuildOutcomeFromFillsWithCoverage(position, item.Lifecycle.Binding, item.Lifecycle.EntryFill, fills[0], decision.Reason, item.Review.SessionNumber, observation.Path, item.Lifecycle.Checkpoints, observation.Excursion)
	if err != nil {
		return err
	}
	return r.Store.PersistOutcome(ctx, position.PositionID, ExitSnapshot{Order: order, Fill: fills[0], Ledger: account, Approval: &approval}, outcome)
}
