package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ConfirmationDecision string

const (
	ConfirmationApprove ConfirmationDecision = "APPROVE_PAPER_INTENT"
	ConfirmationReject  ConfirmationDecision = "REJECT_WORKFLOW"
)

type Confirmation struct {
	ConfirmationID         string               `json:"confirmation_id"`
	ContractVersion        string               `json:"contract_version"`
	WorkflowID             string               `json:"workflow_id"`
	RecommendationID       string               `json:"recommendation_id"`
	RiskDecisionID         string               `json:"risk_decision_id"`
	PortfolioSnapshotID    string               `json:"portfolio_snapshot_id"`
	PolicyID               string               `json:"policy_id"`
	ProposalID             string               `json:"proposal_id,omitempty"`
	InstrumentID           string               `json:"instrument_id"`
	Direction              string               `json:"direction"`
	ProposedValue          float64              `json:"proposed_value"`
	RiskOutcome            string               `json:"risk_outcome"`
	Decision               ConfirmationDecision `json:"decision"`
	Actor                  string               `json:"actor"`
	ConfirmedAt            time.Time            `json:"confirmed_at"`
	ExpiresAt              time.Time            `json:"expires_at"`
	PaperOnly              bool                 `json:"paper_only"`
	BrokerExecutionAllowed bool                 `json:"broker_execution_allowed"`
	LiveTradingAllowed     bool                 `json:"live_trading_allowed"`
}

func NewConfirmation(workflow Workflow, instrumentID, direction string, now, expiresAt time.Time) (Confirmation, error) {
	if err := workflow.Validate(); err != nil {
		return Confirmation{}, err
	}
	if strings.TrimSpace(instrumentID) == "" || (strings.ToUpper(direction) != "LONG" && strings.ToUpper(direction) != "SHORT") {
		return Confirmation{}, fmt.Errorf("instrument and LONG/SHORT direction are required")
	}
	if err := validateUTC(now); err != nil {
		return Confirmation{}, err
	}
	if err := validateUTC(expiresAt); err != nil || !expiresAt.After(now) {
		return Confirmation{}, fmt.Errorf("confirmation expiry must be a later UTC timestamp")
	}
	confirmation := Confirmation{
		ContractVersion: WorkflowContractVersion, WorkflowID: workflow.WorkflowID,
		RecommendationID: workflow.RecommendationID, RiskDecisionID: workflow.RiskDecisionID,
		PortfolioSnapshotID: workflow.PortfolioSnapshotID, PolicyID: workflow.PolicyID,
		ProposalID: workflow.ProposalID, InstrumentID: strings.TrimSpace(instrumentID),
		Direction: strings.ToUpper(direction), ProposedValue: workflow.ResultingValue,
		RiskOutcome: string(workflow.RiskOutcome), ConfirmedAt: now, ExpiresAt: expiresAt,
		PaperOnly: true, BrokerExecutionAllowed: false, LiveTradingAllowed: false,
	}
	confirmation.ConfirmationID = confirmationIdentity(confirmation)
	return confirmation, nil
}

// WithDecision binds the explicit human actor and decision to the exact facts
// presented for review. Callers cannot construct a valid approval by changing
// a field without also producing a new content identity.
func (confirmation Confirmation) WithDecision(actor string, decision ConfirmationDecision) (Confirmation, error) {
	if strings.TrimSpace(actor) == "" {
		return Confirmation{}, ErrUnauthorized
	}
	if decision != ConfirmationApprove && decision != ConfirmationReject {
		return Confirmation{}, fmt.Errorf("confirmation decision must be explicit")
	}
	confirmation.Actor = strings.TrimSpace(actor)
	confirmation.Decision = decision
	confirmation.ConfirmationID = confirmationIdentity(confirmation)
	return confirmation, nil
}

func (confirmation Confirmation) ValidateFor(workflow Workflow, now time.Time) error {
	if confirmation.ContractVersion != WorkflowContractVersion || confirmation.WorkflowID != workflow.WorkflowID {
		return ErrBindingMismatch
	}
	if confirmation.RecommendationID != workflow.RecommendationID || confirmation.RiskDecisionID != workflow.RiskDecisionID || confirmation.PortfolioSnapshotID != workflow.PortfolioSnapshotID || confirmation.PolicyID != workflow.PolicyID || confirmation.ProposalID != workflow.ProposalID {
		return ErrBindingMismatch
	}
	if confirmation.ProposedValue != workflow.ResultingValue || confirmation.RiskOutcome != string(workflow.RiskOutcome) {
		return ErrBindingMismatch
	}
	if confirmation.Decision != ConfirmationApprove && confirmation.Decision != ConfirmationReject {
		return fmt.Errorf("confirmation decision must be explicit")
	}
	if strings.TrimSpace(confirmation.Actor) == "" || strings.TrimSpace(confirmation.InstrumentID) == "" || (confirmation.Direction != "LONG" && confirmation.Direction != "SHORT") {
		return fmt.Errorf("confirmation actor, instrument and direction are required")
	}
	if !confirmation.PaperOnly || confirmation.BrokerExecutionAllowed || confirmation.LiveTradingAllowed {
		return ErrSafetyInvariant
	}
	if err := validateUTC(confirmation.ConfirmedAt); err != nil {
		return err
	}
	if err := validateUTC(confirmation.ExpiresAt); err != nil || !confirmation.ExpiresAt.After(confirmation.ConfirmedAt) {
		return fmt.Errorf("confirmation expiry is invalid")
	}
	if err := validateUTC(now); err != nil || !now.Before(confirmation.ExpiresAt) {
		return fmt.Errorf("confirmation is expired")
	}
	if confirmation.ConfirmationID != confirmationIdentity(confirmation) {
		return ErrBindingMismatch
	}
	return nil
}

func confirmationIdentity(confirmation Confirmation) string {
	copyConfirmation := confirmation
	copyConfirmation.ConfirmationID = ""
	data, _ := json.Marshal(copyConfirmation)
	digest := sha256.Sum256(data)
	return "cnf_" + hex.EncodeToString(digest[:])
}
