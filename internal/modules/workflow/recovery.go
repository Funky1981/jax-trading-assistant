package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DurableSnapshotVersion = "jax.workflow.durable_snapshot/v1"

type PersistedIdempotency struct {
	Fingerprint string          `json:"fingerprint"`
	WorkflowID  string          `json:"workflow_id"`
	Event       TransitionEvent `json:"event"`
}

type DurableSnapshot struct {
	Version       string                          `json:"version"`
	Workflows     map[string]Workflow             `json:"workflows"`
	Events        map[string][]TransitionEvent    `json:"events"`
	Idempotency   map[string]PersistedIdempotency `json:"idempotency"`
	PaperIntents  map[string]PaperIntent          `json:"paper_intents"`
	Breakers      map[string]Breaker              `json:"breakers"`
	BreakerEvents map[string][]BreakerEvent       `json:"breaker_events"`
}

type ReconciliationRequest struct {
	WorkflowID     string    `json:"workflow_id"`
	Actor          string    `json:"actor"`
	ActorRole      ActorRole `json:"actor_role"`
	IdempotencyKey string    `json:"idempotency_key"`
	Now            time.Time `json:"now"`
	Cancel         bool      `json:"cancel"`
}

func (s *Store) MarkReconciliationRequired(_ context.Context, request TransitionRequest) (Workflow, error) {
	if request.Action == "" {
		request.Action = ActionRequireReconciliation
	}
	return s.transition(request, StateReconciliationRequired, ActorRecovery)
}

func (s *Store) ResolveReconciliation(_ context.Context, request ReconciliationRequest) (Workflow, error) {
	if request.ActorRole != ActorOperator || strings.TrimSpace(request.Actor) == "" || strings.TrimSpace(request.WorkflowID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return Workflow{}, ErrUnauthorized
	}
	if request.Cancel {
		return s.transition(TransitionRequest{WorkflowID: request.WorkflowID, Action: ActionCancel, Actor: request.Actor, ActorRole: request.ActorRole, IdempotencyKey: request.IdempotencyKey, Now: request.Now, Reason: "operator resolved reconciliation by cancellation"}, StateCancelled, ActorOperator)
	}
	return s.transition(TransitionRequest{WorkflowID: request.WorkflowID, Action: ActionBlock, Actor: request.Actor, ActorRole: request.ActorRole, IdempotencyKey: request.IdempotencyKey, Now: request.Now, Reason: "operator resolved reconciliation by blocking"}, StateBlocked, ActorOperator)
}

func (s *Store) ExportSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := DurableSnapshot{Version: DurableSnapshotVersion, Workflows: cloneWorkflows(s.workflows), Events: cloneEvents(s.events), Idempotency: make(map[string]PersistedIdempotency, len(s.idempotency)), PaperIntents: cloneIntents(s.paperIntents), Breakers: cloneBreakers(s.breakers), BreakerEvents: cloneBreakerEvents(s.breakerEvents)}
	for key, record := range s.idempotency {
		snapshot.Idempotency[key] = PersistedIdempotency{Fingerprint: record.fingerprint, WorkflowID: record.workflowID, Event: record.event}
	}
	if err := validateSnapshot(snapshot); err != nil {
		return nil, err
	}
	return json.Marshal(snapshot)
}

func RestoreSnapshot(data []byte) (*Store, error) {
	var snapshot DurableSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode durable workflow snapshot: %w", err)
	}
	if err := validateSnapshot(snapshot); err != nil {
		return nil, err
	}
	store := NewStore()
	store.workflows = snapshot.Workflows
	store.events = snapshot.Events
	store.paperIntents = snapshot.PaperIntents
	store.breakers = snapshot.Breakers
	store.breakerEvents = snapshot.BreakerEvents
	for key, record := range snapshot.Idempotency {
		store.idempotency[key] = idempotencyRecord{fingerprint: record.Fingerprint, workflowID: record.WorkflowID, event: record.Event}
	}
	return store, nil
}

func validateSnapshot(snapshot DurableSnapshot) error {
	if snapshot.Version != DurableSnapshotVersion || snapshot.Workflows == nil || snapshot.Events == nil || snapshot.Idempotency == nil || snapshot.PaperIntents == nil || snapshot.Breakers == nil || snapshot.BreakerEvents == nil {
		return fmt.Errorf("unsupported or incomplete durable workflow snapshot")
	}
	for workflowID, workflow := range snapshot.Workflows {
		if workflow.WorkflowID != workflowID {
			return fmt.Errorf("workflow snapshot identity mismatch")
		}
		if err := workflow.Validate(); err != nil {
			return fmt.Errorf("workflow %s: %w", workflowID, err)
		}
		events := snapshot.Events[workflowID]
		if err := ValidateAuditEvents(events); err != nil || events[len(events)-1].NextState != workflow.State || events[len(events)-1].Sequence != workflow.Revision {
			return fmt.Errorf("workflow %s audit cannot be resumed", workflowID)
		}
		if workflow.State == StatePaperIntentCreated {
			intent, ok := snapshot.PaperIntents[workflowID]
			if !ok || intent.IntentID != workflow.PaperIntentID || intent.WorkflowID != workflowID {
				return fmt.Errorf("workflow %s paper intent is missing or mismatched", workflowID)
			}
			if err := intent.Validate(); err != nil {
				return err
			}
		}
	}
	for key, record := range snapshot.Idempotency {
		if key == "" || record.WorkflowID == "" || record.Fingerprint == "" || record.Event.IdempotencyKey != key {
			return fmt.Errorf("durable idempotency record is invalid")
		}
	}
	for name, breaker := range snapshot.Breakers {
		if breaker.Name != name {
			return fmt.Errorf("breaker snapshot identity mismatch")
		}
		if err := breaker.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func cloneWorkflows(input map[string]Workflow) map[string]Workflow {
	out := make(map[string]Workflow, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
func cloneEvents(input map[string][]TransitionEvent) map[string][]TransitionEvent {
	out := make(map[string][]TransitionEvent, len(input))
	for key, value := range input {
		out[key] = append([]TransitionEvent(nil), value...)
	}
	return out
}
func cloneIntents(input map[string]PaperIntent) map[string]PaperIntent {
	out := make(map[string]PaperIntent, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
func cloneBreakers(input map[string]Breaker) map[string]Breaker {
	out := make(map[string]Breaker, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
func cloneBreakerEvents(input map[string][]BreakerEvent) map[string][]BreakerEvent {
	out := make(map[string][]BreakerEvent, len(input))
	for key, value := range input {
		out[key] = append([]BreakerEvent(nil), value...)
	}
	return out
}
