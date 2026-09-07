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
	mu                 sync.RWMutex
	workflows          map[string]Workflow
	events             map[string][]TransitionEvent
	idempotency        map[string]idempotencyRecord
	paperIntents       map[string]PaperIntent
	breakers           map[string]Breaker
	breakerEvents      map[string][]BreakerEvent
	breakerIdempotency map[string]string
	failAuditOnce      bool
}

func NewStore() *Store {
	return &Store{
		workflows: make(map[string]Workflow), events: make(map[string][]TransitionEvent),
		idempotency: make(map[string]idempotencyRecord), paperIntents: make(map[string]PaperIntent),
		breakers: make(map[string]Breaker), breakerEvents: make(map[string][]BreakerEvent), breakerIdempotency: make(map[string]string),
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
	if !PermissionAllowed(ActorSystem, ActionWorkflowCreated, State(""), workflow.State) {
		return Workflow{}, ErrUnauthorized
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

type HumanConfirmationRequest struct {
	WorkflowID     string       `json:"workflow_id"`
	Confirmation   Confirmation `json:"confirmation"`
	Actor          string       `json:"actor"`
	ActorRole      ActorRole    `json:"actor_role"`
	IdempotencyKey string       `json:"idempotency_key"`
	Now            time.Time    `json:"now"`
}

func (s *Store) Confirm(_ context.Context, request HumanConfirmationRequest) (Workflow, error) {
	if request.ActorRole != ActorHuman || strings.TrimSpace(request.Actor) == "" || request.Actor != request.Confirmation.Actor {
		return Workflow{}, ErrUnauthorized
	}
	if strings.TrimSpace(request.WorkflowID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
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
	if workflow.State != StateAwaitingHumanConfirmation || !PermissionAllowed(request.ActorRole, request.ConfirmationAction(), StateAwaitingHumanConfirmation, confirmationTarget(request.Confirmation.Decision)) {
		return Workflow{}, fmt.Errorf("%w: confirmation requires awaiting human confirmation", ErrInvalidTransition)
	}
	if request.Confirmation.Decision == ConfirmationApprove && s.anyBreakerTrippedLocked() {
		return Workflow{}, ErrBreakerActive
	}
	if err := request.Confirmation.ValidateFor(workflow, now); err != nil {
		return Workflow{}, err
	}
	target, action := StateHumanRejected, ActionHumanReject
	if request.Confirmation.Decision == ConfirmationApprove {
		target, action = StateHumanApproved, ActionHumanApprove
	}
	if now.Before(workflow.UpdatedAt) {
		return Workflow{}, fmt.Errorf("%w: confirmation timestamp precedes workflow update", ErrInvalidTimestamp)
	}
	previous := workflow.State
	workflow.State = target
	workflow.Revision++
	workflow.UpdatedAt = now
	confirmation := request.Confirmation
	workflow.Confirmation = &confirmation
	if err := workflow.Validate(); err != nil {
		return Workflow{}, err
	}
	event, err := s.appendLocked(workflow, previous, target, action, ActorHuman, request.Actor, request.IdempotencyKey, request, now, "explicit human decision")
	if err != nil {
		return Workflow{}, err
	}
	s.workflows[workflow.WorkflowID] = workflow
	s.idempotency[request.IdempotencyKey] = idempotencyRecord{fingerprint: inputFingerprint(request), workflowID: workflow.WorkflowID, event: event}
	return workflow, nil
}

type PaperIntentRequest struct {
	WorkflowID     string    `json:"workflow_id"`
	Actor          string    `json:"actor"`
	ActorRole      ActorRole `json:"actor_role"`
	IdempotencyKey string    `json:"idempotency_key"`
	Now            time.Time `json:"now"`
}

func (s *Store) CreatePaperIntent(_ context.Context, request PaperIntentRequest) (Workflow, PaperIntent, error) {
	if request.ActorRole != ActorSystem || strings.TrimSpace(request.Actor) == "" || strings.TrimSpace(request.WorkflowID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return Workflow{}, PaperIntent{}, ErrUnauthorized
	}
	now, err := requestTime(request.Now)
	if err != nil {
		return Workflow{}, PaperIntent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idempotency[request.IdempotencyKey]; ok {
		if existing.fingerprint != inputFingerprint(request) {
			return Workflow{}, PaperIntent{}, ErrIdempotencyConflict
		}
		workflow := s.workflows[existing.workflowID]
		return workflow, s.paperIntents[workflow.WorkflowID], nil
	}
	workflow, ok := s.workflows[request.WorkflowID]
	if !ok {
		return Workflow{}, PaperIntent{}, ErrWorkflowNotFound
	}
	if workflow.State != StateHumanApproved || !PermissionAllowed(request.ActorRole, ActionCreatePaperIntent, workflow.State, StatePaperIntentCreated) || workflow.Confirmation == nil {
		return Workflow{}, PaperIntent{}, fmt.Errorf("%w: paper intent requires explicit human approval", ErrInvalidTransition)
	}
	if s.anyBreakerTrippedLocked() {
		return Workflow{}, PaperIntent{}, ErrBreakerActive
	}
	if err := workflow.Confirmation.ValidateFor(workflow, now); err != nil {
		return Workflow{}, PaperIntent{}, err
	}
	if now.Before(workflow.UpdatedAt) {
		return Workflow{}, PaperIntent{}, fmt.Errorf("%w: paper intent timestamp precedes workflow update", ErrInvalidTimestamp)
	}
	intent := PaperIntent{
		ContractVersion: PaperIntentContractVersion, WorkflowID: workflow.WorkflowID,
		RecommendationID: workflow.RecommendationID, RiskDecisionID: workflow.RiskDecisionID,
		InstrumentID: workflow.Confirmation.InstrumentID, Direction: workflow.Confirmation.Direction,
		DescriptiveValue: workflow.ResultingValue, ExecutionStatus: ExecutionStatusNotExecuted,
		PaperOnly: true, BrokerExecutionAllowed: false, PortfolioMutation: false, CreatedAt: now,
	}
	intent.IntentID = paperIntentIdentity(intent)
	if err := intent.Validate(); err != nil {
		return Workflow{}, PaperIntent{}, err
	}
	previous := workflow.State
	workflow.State = StatePaperIntentCreated
	workflow.PaperIntentID = intent.IntentID
	workflow.Revision++
	workflow.UpdatedAt = now
	if err := workflow.Validate(); err != nil {
		return Workflow{}, PaperIntent{}, err
	}
	event, err := s.appendLocked(workflow, previous, workflow.State, ActionCreatePaperIntent, ActorSystem, request.Actor, request.IdempotencyKey, request, now, "inert paper intent created")
	if err != nil {
		return Workflow{}, PaperIntent{}, err
	}
	s.workflows[workflow.WorkflowID] = workflow
	s.paperIntents[workflow.WorkflowID] = intent
	s.idempotency[request.IdempotencyKey] = idempotencyRecord{fingerprint: inputFingerprint(request), workflowID: workflow.WorkflowID, event: event}
	return workflow, intent, nil
}

func (request HumanConfirmationRequest) ConfirmationAction() Action {
	if request.Confirmation.Decision == ConfirmationApprove {
		return ActionHumanApprove
	}
	return ActionHumanReject
}

func confirmationTarget(decision ConfirmationDecision) State {
	if decision == ConfirmationApprove {
		return StateHumanApproved
	}
	return StateHumanRejected
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
	if !allowedTransition(workflow.State, target, request.Action) || !PermissionAllowed(request.ActorRole, request.Action, workflow.State, target) {
		return Workflow{}, fmt.Errorf("%w: %s -> %s via %s", ErrInvalidTransition, workflow.State, target, request.Action)
	}
	if isDangerousState(target) && s.anyBreakerTrippedLocked() {
		return Workflow{}, ErrBreakerActive
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
	event.ContentIdentity = transitionContentIdentity(event)
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

func isDangerousState(state State) bool {
	return state == StateAwaitingHumanConfirmation || state == StateHumanApproved || state == StatePaperIntentCreated
}

type BreakerEvent struct {
	EventID        string    `json:"event_id"`
	Name           string    `json:"name"`
	Sequence       uint64    `json:"sequence"`
	Tripped        bool      `json:"tripped"`
	Reason         string    `json:"reason,omitempty"`
	Actor          string    `json:"actor"`
	IdempotencyKey string    `json:"idempotency_key"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func (event BreakerEvent) Validate() error {
	if event.EventID == "" || event.Name == "" || event.Sequence == 0 || event.Actor == "" || event.IdempotencyKey == "" {
		return fmt.Errorf("breaker event identity is invalid")
	}
	if err := validateUTC(event.OccurredAt); err != nil {
		return err
	}
	if event.EventID != eventIdentity("breaker:"+event.Name, event.Sequence, breakerAction(event.Tripped), event.IdempotencyKey) {
		return fmt.Errorf("breaker event identity mismatch")
	}
	return nil
}

type BreakerRequest struct {
	Name           string    `json:"name"`
	Tripped        bool      `json:"tripped"`
	Reason         string    `json:"reason,omitempty"`
	Actor          string    `json:"actor"`
	ActorRole      ActorRole `json:"actor_role"`
	IdempotencyKey string    `json:"idempotency_key"`
	Now            time.Time `json:"now"`
}

func (s *Store) SetBreaker(_ context.Context, request BreakerRequest) (Breaker, error) {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Actor) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return Breaker{}, ErrUnauthorized
	}
	if request.Tripped && !(request.ActorRole == ActorOperator || request.ActorRole == ActorSystem) {
		return Breaker{}, ErrUnauthorized
	}
	if !request.Tripped && request.ActorRole != ActorOperator {
		return Breaker{}, ErrUnauthorized
	}
	now, err := requestTime(request.Now)
	if err != nil {
		return Breaker{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fingerprint := inputFingerprint(request)
	if priorFingerprint, ok := s.breakerIdempotency[request.IdempotencyKey]; ok {
		if priorFingerprint != fingerprint {
			return Breaker{}, ErrIdempotencyConflict
		}
		return s.breakers[request.Name], nil
	}
	current := s.breakers[request.Name]
	if !current.UpdatedAt.IsZero() && now.Before(current.UpdatedAt) {
		return Breaker{}, ErrInvalidTimestamp
	}
	breaker := Breaker{Name: request.Name, ContractVersion: BreakerContractVersion, Tripped: request.Tripped, Reason: strings.TrimSpace(request.Reason), Actor: request.Actor, UpdatedAt: now}
	if err := breaker.Validate(); err != nil {
		return Breaker{}, err
	}
	sequence := uint64(len(s.breakerEvents[request.Name]) + 1)
	event := BreakerEvent{EventID: eventIdentity("breaker:"+request.Name, sequence, breakerAction(request.Tripped), request.IdempotencyKey), Name: request.Name, Sequence: sequence, Tripped: request.Tripped, Reason: breaker.Reason, Actor: request.Actor, IdempotencyKey: request.IdempotencyKey, OccurredAt: now}
	s.breakers[request.Name] = breaker
	s.breakerEvents[request.Name] = append(s.breakerEvents[request.Name], event)
	s.breakerIdempotency[request.IdempotencyKey] = fingerprint
	return breaker, nil
}

func breakerAction(tripped bool) Action {
	if tripped {
		return ActionTripBreaker
	}
	return ActionResetBreaker
}

func (s *Store) BreakerEvents(_ context.Context, name string) []BreakerEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]BreakerEvent(nil), s.breakerEvents[name]...)
}

func (s *Store) anyBreakerTrippedLocked() bool {
	for _, breaker := range s.breakers {
		if breaker.Tripped {
			return true
		}
	}
	return false
}

func (s *Store) AnyBreakerTripped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.anyBreakerTrippedLocked()
}

func (s *Store) Replay(_ context.Context, workflowID string) (Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	workflow, ok := s.workflows[workflowID]
	if !ok {
		return Workflow{}, ErrWorkflowNotFound
	}
	events := append([]TransitionEvent(nil), s.events[workflowID]...)
	if err := ValidateAuditEvents(events); err != nil {
		return Workflow{}, fmt.Errorf("workflow audit is not replayable: %w", err)
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
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateFailed && to == StateCancelled && action == ActionCancel:
		return true
	case from != StatePaperIntentCreated && from != StateHumanRejected && from != StateCancelled && from != StateBlocked && from != StateFailed && to == StateBlocked && action == ActionBlock:
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
