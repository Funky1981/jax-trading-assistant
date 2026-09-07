package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"jax-trading-assistant/internal/modules/portfoliorisk"
)

type CreateRequest struct {
	RiskDecision   portfoliorisk.RiskDecision `json:"risk_decision"`
	Now            time.Time                  `json:"now"`
	IdempotencyKey string                     `json:"idempotency_key"`
}

type TransitionRequest struct {
	WorkflowID     string    `json:"workflow_id"`
	Action         Action    `json:"action"`
	Actor          string    `json:"actor"`
	ActorRole      ActorRole `json:"actor_role"`
	IdempotencyKey string    `json:"idempotency_key"`
	Now            time.Time `json:"now"`
	Reason         string    `json:"reason,omitempty"`
}

type idempotencyRecord struct {
	fingerprint string
	workflowID  string
	event       TransitionEvent
}

// Store is the bounded authoritative in-memory workflow store used by the
// deterministic Phase-10 contract and harness. Its mutex models the atomic
// state/audit transaction; the persistence adapter can apply the same command
// contract to Postgres without changing the state machine.
type Store struct {
	mu            sync.RWMutex
	workflows     map[string]Workflow
	events        map[string][]TransitionEvent
	idempotency   map[string]idempotencyRecord
	paperIntents  map[string]PaperIntent
	breakers      map[string]Breaker
	failAuditOnce bool
}

func NewStore() *Store {
	return &Store{
		workflows: make(map[string]Workflow), events: make(map[string][]TransitionEvent),
		idempotency: make(map[string]idempotencyRecord), paperIntents: make(map[string]PaperIntent),
		breakers: make(map[string]Breaker),
	}
}

func (s *Store) Create(_ context.Context, request CreateRequest) (Workflow, error) {
	if request.RiskDecision.Outcome == portfoliorisk.DecisionReject {
		return Workflow{}, ErrRiskRejected
	}
	if err := request.RiskDecision.Validate(); err != nil {
		return Workflow{}, fmt.Errorf("risk decision: %w", err)
	}
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return Workflow{}, fmt.Errorf("idempotency key is required")
	}
	now, err := requestTime(request.Now)
	if err != nil {
		return Workflow{}, err
	}
	binding := RiskBinding{
		RecommendationID:    request.RiskDecision.RecommendationID,
		RiskDecisionID:      request.RiskDecision.DecisionID,
		PortfolioSnapshotID: request.RiskDecision.PortfolioSnapshotID,
		AnalyticsID:         request.RiskDecision.AnalyticsID,
		PolicyID:            request.RiskDecision.PolicyID,
		ProposalID:          request.RiskDecision.ProposalID,
		Outcome:             request.RiskDecision.Outcome,
		ResultingValue:      request.RiskDecision.ResultingValue,
	}
	if err := binding.Validate(); err != nil {
		return Workflow{}, err
	}
	fingerprint := inputFingerprint(request)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idempotency[request.IdempotencyKey]; ok {
		if existing.fingerprint != fingerprint {
			return Workflow{}, ErrIdempotencyConflict
		}
		return s.workflows[existing.workflowID], nil
	}
	workflow := Workflow{
		WorkflowID: workflowIdentity(binding), ContractVersion: WorkflowContractVersion,
		Algorithm: WorkflowAlgorithmVersion, RecommendationID: binding.RecommendationID,
		RiskDecisionID: binding.RiskDecisionID, PortfolioSnapshotID: binding.PortfolioSnapshotID,
		AnalyticsID: binding.AnalyticsID, PolicyID: binding.PolicyID, ProposalID: binding.ProposalID,
		RiskOutcome: binding.Outcome, ResultingValue: binding.ResultingValue,
		State: stateForRisk(binding.Outcome), Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := workflow.Validate(); err != nil {
		return Workflow{}, err
	}
	event, err := s.appendLocked(workflow, State(""), workflow.State, ActionWorkflowCreated, ActorSystem, "workflow-system", request.IdempotencyKey, request, now, "risk decision bound")
	if err != nil {
		return Workflow{}, err
	}
	s.workflows[workflow.WorkflowID] = workflow
	s.idempotency[request.IdempotencyKey] = idempotencyRecord{fingerprint: fingerprint, workflowID: workflow.WorkflowID, event: event}
	return workflow, nil
}

func (s *Store) RequestHumanConfirmation(_ context.Context, request TransitionRequest) (Workflow, error) {
	if request.Action == "" {
		request.Action = ActionRequestConfirmation
	}
	return s.transition(request, StateAwaitingHumanConfirmation, ActorSystem)
}

func (s *Store) transition(request TransitionRequest, target State, requiredRole ActorRole) (Workflow, error) {
	if request.Action == "" || strings.TrimSpace(request.WorkflowID) == "" || strings.TrimSpace(request.Actor) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return Workflow{}, ErrUnauthorized
	}
	if request.ActorRole != requiredRole {
		return Workflow{}, ErrUnauthorized
	}
	now, err := requestTime(request.Now)
	if err != nil {
		return Workflow{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idempotency[request.IdempotencyKey]; ok {
		if existing.fingerprint != inputFingerprint(request) {
			return Workflow{}, ErrIdempotencyConflict
		}
		return s.workflows[existing.workflowID], nil
	}
	workflow, ok := s.workflows[request.WorkflowID]
	if !ok {
		return Workflow{}, ErrWorkflowNotFound
	}
	if !allowedTransition(workflow.State, target, request.Action) {
		return Workflow{}, fmt.Errorf("%w: %s -> %s via %s", ErrInvalidTransition, workflow.State, target, request.Action)
	}
	if now.Before(workflow.UpdatedAt) {
		return Workflow{}, fmt.Errorf("%w: transition timestamp precedes workflow update", ErrInvalidTimestamp)
	}
	previous := workflow.State
	workflow.State = target
	workflow.Revision++
	workflow.UpdatedAt = now
	if err := workflow.Validate(); err != nil {
		return Workflow{}, err
	}
	event, err := s.appendLocked(workflow, previous, target, request.Action, request.ActorRole, request.Actor, request.IdempotencyKey, request, now, request.Reason)
	if err != nil {
		return Workflow{}, err
	}
	s.workflows[workflow.WorkflowID] = workflow
	s.idempotency[request.IdempotencyKey] = idempotencyRecord{fingerprint: inputFingerprint(request), workflowID: workflow.WorkflowID, event: event}
	return workflow, nil
}

func (s *Store) Get(_ context.Context, workflowID string) (Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	workflow, ok := s.workflows[strings.TrimSpace(workflowID)]
	if !ok {
		return Workflow{}, ErrWorkflowNotFound
	}
	return workflow, nil
}

func (s *Store) Events(_ context.Context, workflowID string) ([]TransitionEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.workflows[workflowID]; !ok {
		return nil, ErrWorkflowNotFound
	}
	events := append([]TransitionEvent(nil), s.events[workflowID]...)
	return events, nil
}

func (s *Store) appendLocked(workflow Workflow, previous, next State, action Action, role ActorRole, actor, idem string, input any, now time.Time, reason string) (TransitionEvent, error) {
	if s.failAuditOnce {
		s.failAuditOnce = false
		return TransitionEvent{}, fmt.Errorf("audit persistence failed")
	}
	sequence := uint64(len(s.events[workflow.WorkflowID]) + 1)
	event := TransitionEvent{
		EventID: eventIdentity(workflow.WorkflowID, sequence, action, idem), WorkflowID: workflow.WorkflowID,
		Sequence: sequence, PreviousState: previous, NextState: next, Action: action,
		Actor: actor, ActorRole: role, IdempotencyKey: idem, InputFingerprint: inputFingerprint(input),
		Reason: reason, OccurredAt: now, TransitionVersion: WorkflowAlgorithmVersion,
	}
	if err := event.Validate(); err != nil {
		return TransitionEvent{}, err
	}
	s.events[workflow.WorkflowID] = append(s.events[workflow.WorkflowID], event)
	return event, nil
}

func (s *Store) FailNextAuditPersistence() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failAuditOnce = true
}

func (s *Store) Replay(_ context.Context, workflowID string) (Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	workflow, ok := s.workflows[workflowID]
	if !ok {
		return Workflow{}, ErrWorkflowNotFound
	}
	events := append([]TransitionEvent(nil), s.events[workflowID]...)
	if len(events) == 0 {
		return Workflow{}, fmt.Errorf("workflow has no audit history")
	}
	for i, event := range events {
		if err := event.Validate(); err != nil || event.Sequence != uint64(i+1) {
			return Workflow{}, fmt.Errorf("workflow audit is not replayable")
		}
		if i > 0 && event.PreviousState != events[i-1].NextState {
			return Workflow{}, fmt.Errorf("workflow audit state divergence")
		}
	}
	if events[len(events)-1].NextState != workflow.State || events[len(events)-1].Sequence != workflow.Revision {
		return Workflow{}, fmt.Errorf("workflow state does not match audit")
	}
	return workflow, nil
}

func stateForRisk(outcome portfoliorisk.DecisionOutcome) State {
	if outcome == portfoliorisk.DecisionAmend {
		return StateRiskAmended
	}
	return StateRiskAccepted
}

func allowedTransition(from, to State, action Action) bool {
	switch {
	case (from == StateRiskAccepted || from == StateRiskAmended) && to == StateAwaitingHumanConfirmation && action == ActionRequestConfirmation:
		return true
	case from == StateAwaitingHumanConfirmation && to == StateHumanRejected && action == ActionHumanReject:
		return true
	case from == StateAwaitingHumanConfirmation && to == StateHumanApproved && action == ActionHumanApprove:
		return true
	case from == StateHumanApproved && to == StatePaperIntentCreated && action == ActionCreatePaperIntent:
		return true
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateFailed && from != StateReconciliationRequired && to == StateCancelled && action == ActionCancel:
		return true
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateReconciliationRequired && to == StateBlocked && action == ActionBlock:
		return true
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateReconciliationRequired && to == StateFailed && action == ActionFail:
		return true
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateReconciliationRequired && to == StateReconciliationRequired && action == ActionRequireReconciliation:
		return true
	default:
		return false
	}
}

func requestTime(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Now().UTC(), nil
	}
	if err := validateUTC(value); err != nil {
		return time.Time{}, err
	}
	return value, nil
}

func SortedStates() []State {
	states := []State{StateRiskAccepted, StateRiskAmended, StateAwaitingHumanConfirmation, StateHumanRejected, StateHumanApproved, StatePaperIntentCreated, StateCancelled, StateBlocked, StateFailed, StateReconciliationRequired}
	sort.Slice(states, func(i, j int) bool { return states[i] < states[j] })
	return states
}
