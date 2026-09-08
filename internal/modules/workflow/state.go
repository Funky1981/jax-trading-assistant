package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/portfoliorisk"
)

const (
	WorkflowContractVersion       = "jax.workflow/v1"
	PaperIntentContractVersion    = "jax.paper_intent/v1"
	WorkflowAlgorithmVersion      = "jax.workflow.state_machine/v1"
	ExecutionStatusNotExecuted    = "NOT_EXECUTED"
	BrokerExecutionAuthorityNone  = "NONE"
	WorkflowIdentityPrefix        = "wf_"
	PaperIntentIdentityPrefix     = "pint_"
	TransitionEventIdentityPrefix = "wfe_"
)

type State string

const (
	StateRiskAccepted              State = "RISK_ACCEPTED"
	StateRiskAmended               State = "RISK_AMENDED"
	StateAwaitingHumanConfirmation State = "AWAITING_HUMAN_CONFIRMATION"
	StateHumanRejected             State = "HUMAN_REJECTED"
	StateHumanApproved             State = "HUMAN_APPROVED"
	StatePaperIntentCreated        State = "PAPER_INTENT_CREATED"
	StateCancelled                 State = "CANCELLED"
	StateBlocked                   State = "BLOCKED"
	StateFailed                    State = "FAILED"
	StateReconciliationRequired    State = "RECONCILIATION_REQUIRED"
)

type ActorRole string

const (
	ActorSystem     ActorRole = "SYSTEM"
	ActorResearcher ActorRole = "RESEARCH_AGENT"
	ActorHuman      ActorRole = "HUMAN"
	ActorOperator   ActorRole = "OPERATOR"
	ActorRecovery   ActorRole = "RECOVERY"
)

type Action string

const (
	ActionWorkflowCreated       Action = "WORKFLOW_CREATED"
	ActionRequestConfirmation   Action = "REQUEST_HUMAN_CONFIRMATION"
	ActionHumanApprove          Action = "HUMAN_APPROVE"
	ActionHumanReject           Action = "HUMAN_REJECT"
	ActionCreatePaperIntent     Action = "CREATE_PAPER_INTENT"
	ActionCancel                Action = "CANCEL_WORKFLOW"
	ActionBlock                 Action = "BLOCK_WORKFLOW"
	ActionFail                  Action = "FAIL_WORKFLOW"
	ActionRequireReconciliation Action = "REQUIRE_RECONCILIATION"
	ActionTripBreaker           Action = "TRIP_BREAKER"
	ActionResetBreaker          Action = "RESET_BREAKER"
)

var (
	ErrInvalidTransition   = errors.New("workflow transition is invalid")
	ErrUnauthorized        = errors.New("workflow action is unauthorized")
	ErrRiskRejected        = errors.New("risk rejection cannot enter approval workflow")
	ErrBindingMismatch     = errors.New("workflow input binding mismatch")
	ErrIdempotencyConflict = errors.New("workflow idempotency key conflicts with existing action")
	ErrInvalidTimestamp    = errors.New("workflow timestamp is invalid")
	ErrSafetyInvariant     = errors.New("workflow safety invariant failed")
	ErrWorkflowNotFound    = errors.New("workflow not found")
	ErrWorkflowImmutable   = errors.New("workflow artifact is immutable")
	ErrBreakerActive       = errors.New("workflow safety breaker is active")
)

type RiskBinding struct {
	RecommendationID    string                        `json:"recommendation_id"`
	RiskDecisionID      string                        `json:"risk_decision_id"`
	PortfolioSnapshotID string                        `json:"portfolio_snapshot_id"`
	AnalyticsID         string                        `json:"analytics_id"`
	PolicyID            string                        `json:"policy_id"`
	ProposalID          string                        `json:"proposal_id,omitempty"`
	Outcome             portfoliorisk.DecisionOutcome `json:"outcome"`
	ResultingValue      float64                       `json:"resulting_value"`
}

func (binding RiskBinding) Validate() error {
	for name, value := range map[string]string{
		"recommendation_id":     binding.RecommendationID,
		"risk_decision_id":      binding.RiskDecisionID,
		"portfolio_snapshot_id": binding.PortfolioSnapshotID,
		"analytics_id":          binding.AnalyticsID,
		"policy_id":             binding.PolicyID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", ErrBindingMismatch, name)
		}
	}
	if binding.Outcome != portfoliorisk.DecisionAccept && binding.Outcome != portfoliorisk.DecisionAmend {
		return ErrRiskRejected
	}
	if !finite(binding.ResultingValue) || binding.ResultingValue == 0 {
		return fmt.Errorf("%w: resulting value must be finite and non-zero", ErrBindingMismatch)
	}
	return nil
}

type Workflow struct {
	WorkflowID          string                        `json:"workflow_id"`
	ContractVersion     string                        `json:"contract_version"`
	Algorithm           string                        `json:"algorithm"`
	RecommendationID    string                        `json:"recommendation_id"`
	RiskDecisionID      string                        `json:"risk_decision_id"`
	PortfolioSnapshotID string                        `json:"portfolio_snapshot_id"`
	AnalyticsID         string                        `json:"analytics_id"`
	PolicyID            string                        `json:"policy_id"`
	ProposalID          string                        `json:"proposal_id,omitempty"`
	RiskOutcome         portfoliorisk.DecisionOutcome `json:"risk_outcome"`
	ResultingValue      float64                       `json:"resulting_value"`
	State               State                         `json:"state"`
	Revision            uint64                        `json:"revision"`
	CreatedAt           time.Time                     `json:"created_at"`
	UpdatedAt           time.Time                     `json:"updated_at"`
	LastError           string                        `json:"last_error,omitempty"`
	Confirmation        *Confirmation                 `json:"confirmation,omitempty"`
	PaperIntentID       string                        `json:"paper_intent_id,omitempty"`
}

func (workflow Workflow) Binding() RiskBinding {
	return RiskBinding{
		RecommendationID: workflow.RecommendationID, RiskDecisionID: workflow.RiskDecisionID,
		PortfolioSnapshotID: workflow.PortfolioSnapshotID, AnalyticsID: workflow.AnalyticsID,
		PolicyID: workflow.PolicyID, ProposalID: workflow.ProposalID,
		Outcome: workflow.RiskOutcome, ResultingValue: workflow.ResultingValue,
	}
}

func (workflow Workflow) Validate() error {
	if strings.TrimSpace(workflow.WorkflowID) == "" || !strings.HasPrefix(workflow.WorkflowID, WorkflowIdentityPrefix) {
		return fmt.Errorf("workflow id is invalid")
	}
	if workflow.ContractVersion != WorkflowContractVersion || workflow.Algorithm != WorkflowAlgorithmVersion {
		return fmt.Errorf("workflow contract or algorithm is unsupported")
	}
	if err := workflow.Binding().Validate(); err != nil {
		return err
	}
	if workflow.WorkflowID != workflowIdentity(workflow.Binding()) {
		return fmt.Errorf("workflow identity does not match binding")
	}
	if !validState(workflow.State) || workflow.Revision == 0 {
		return fmt.Errorf("workflow state or revision is invalid")
	}
	if err := validateUTC(workflow.CreatedAt); err != nil {
		return fmt.Errorf("created_at: %w", err)
	}
	if err := validateUTC(workflow.UpdatedAt); err != nil {
		return fmt.Errorf("updated_at: %w", err)
	}
	if workflow.UpdatedAt.Before(workflow.CreatedAt) {
		return fmt.Errorf("%w: updated_at precedes created_at", ErrInvalidTimestamp)
	}
	if (workflow.State == StateHumanApproved || workflow.State == StateHumanRejected) && workflow.Confirmation == nil {
		return fmt.Errorf("human decision requires an explicit confirmation")
	}
	if workflow.State == StatePaperIntentCreated && !strings.HasPrefix(workflow.PaperIntentID, PaperIntentIdentityPrefix) {
		return fmt.Errorf("paper intent state requires an immutable paper intent identity")
	}
	return nil
}

type TransitionEvent struct {
	EventID           string    `json:"event_id"`
	WorkflowID        string    `json:"workflow_id"`
	Sequence          uint64    `json:"sequence"`
	PreviousState     State     `json:"previous_state"`
	NextState         State     `json:"next_state"`
	Action            Action    `json:"action"`
	Actor             string    `json:"actor"`
	ActorRole         ActorRole `json:"actor_role"`
	IdempotencyKey    string    `json:"idempotency_key"`
	InputFingerprint  string    `json:"input_fingerprint"`
	ContentIdentity   string    `json:"content_identity"`
	Reason            string    `json:"reason,omitempty"`
	OccurredAt        time.Time `json:"occurred_at"`
	TransitionVersion string    `json:"transition_version"`
}

func (event TransitionEvent) Validate() error {
	if !strings.HasPrefix(event.EventID, TransitionEventIdentityPrefix) || event.WorkflowID == "" || event.Sequence == 0 || event.IdempotencyKey == "" {
		return fmt.Errorf("invalid workflow audit identity")
	}
	if !validState(event.NextState) || event.Actor == "" || event.ActorRole == "" || event.Action == "" {
		return fmt.Errorf("invalid workflow audit event")
	}
	if err := validateUTC(event.OccurredAt); err != nil {
		return err
	}
	if event.TransitionVersion != WorkflowAlgorithmVersion {
		return fmt.Errorf("unsupported transition version")
	}
	if event.ContentIdentity == "" || event.ContentIdentity != transitionContentIdentity(event) {
		return fmt.Errorf("workflow audit content identity mismatch")
	}
	return nil
}

func workflowIdentity(binding RiskBinding) string {
	data, _ := json.Marshal(struct {
		Contract string      `json:"contract"`
		Binding  RiskBinding `json:"binding"`
	}{WorkflowContractVersion, binding})
	digest := sha256.Sum256(data)
	return WorkflowIdentityPrefix + hex.EncodeToString(digest[:])
}

func paperIntentIdentity(intent PaperIntent) string {
	copyIntent := intent
	copyIntent.IntentID = ""
	data, _ := json.Marshal(copyIntent)
	digest := sha256.Sum256(data)
	return PaperIntentIdentityPrefix + hex.EncodeToString(digest[:])
}

func eventIdentity(workflowID string, sequence uint64, action Action, idempotency string) string {
	data := []byte(fmt.Sprintf("%s|%d|%s|%s", workflowID, sequence, action, idempotency))
	digest := sha256.Sum256(data)
	return TransitionEventIdentityPrefix + hex.EncodeToString(digest[:])
}

func transitionContentIdentity(event TransitionEvent) string {
	copyEvent := event
	copyEvent.EventID = ""
	copyEvent.ContentIdentity = ""
	data, _ := json.Marshal(copyEvent)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func inputFingerprint(value any) string {
	data, _ := json.Marshal(value)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func validState(state State) bool {
	switch state {
	case StateRiskAccepted, StateRiskAmended, StateAwaitingHumanConfirmation, StateHumanRejected,
		StateHumanApproved, StatePaperIntentCreated, StateCancelled, StateBlocked, StateFailed,
		StateReconciliationRequired:
		return true
	default:
		return false
	}
}

// ValidateAuditEvents validates a complete workflow stream as it would be
// reconstructed from durable audit storage. It rejects gaps, rewrites and
// state divergence before replay can be trusted.
func ValidateAuditEvents(events []TransitionEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("workflow audit is empty")
	}
	for i, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		if event.Sequence != uint64(i+1) {
			return fmt.Errorf("workflow audit sequence gap")
		}
		if i > 0 && event.PreviousState != events[i-1].NextState {
			return fmt.Errorf("workflow audit state divergence")
		}
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func validateUTC(value time.Time) error {
	if value.IsZero() || value.Location() != time.UTC || value.Nanosecond() < 0 {
		return ErrInvalidTimestamp
	}
	return nil
}
