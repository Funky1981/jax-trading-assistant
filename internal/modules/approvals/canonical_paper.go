package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrCanonicalPaperDecisionConflict = errors.New("canonical PAPER candidate decision conflicts with durable state")

type CanonicalPaperDecision struct {
	CandidateID           string  `json:"candidateId"`
	WorkflowID            string  `json:"workflowId"`
	RiskID                string  `json:"riskDecisionId"`
	ConfirmID             string  `json:"confirmationId"`
	Instrument            string  `json:"instrumentId"`
	Direction             string  `json:"direction"`
	Value                 float64 `json:"resultingValue"`
	Actor                 string  `json:"actor"`
	EconomicInputIdentity string  `json:"economicInputIdentity"`
	Decision              string  `json:"decision"`
}

// PersistCanonicalWorldMonitorDecision writes the human decision into the
// established candidate approval/read-model tables. Callers must hold the
// candidate row lock and include this write in the same transaction as the
// durable workflow and paper queue changes.

func PersistCanonicalWorldMonitorDecision(ctx context.Context, tx pgx.Tx, candidateID uuid.UUID, actor, economicInputIdentity string, risk portfoliorisk.RiskDecision, wf workflow.Workflow, now time.Time) error {
	if tx == nil || candidateID == uuid.Nil || strings.TrimSpace(actor) == "" || strings.TrimSpace(economicInputIdentity) == "" {
		return fmt.Errorf("canonical candidate approval transaction, candidate, and authenticated actor are required")
	}
	if err := risk.Validate(); err != nil || (risk.Outcome != portfoliorisk.DecisionAccept && risk.Outcome != portfoliorisk.DecisionAmend) {
		return fmt.Errorf("canonical candidate decision requires an accepted portfolio risk decision")
	}
	if err := wf.Validate(); err != nil || wf.RecommendationID != candidateID.String() || wf.RiskDecisionID != risk.DecisionID || wf.PortfolioSnapshotID != risk.PortfolioSnapshotID || wf.PolicyID != risk.PolicyID {
		return fmt.Errorf("canonical workflow and risk identities do not match the candidate")
	}
	if wf.State != workflow.StatePaperIntentCreated && wf.State != workflow.StateHumanRejected || wf.Confirmation == nil {
		return fmt.Errorf("canonical candidate decision requires a final authenticated workflow confirmation")
	}
	confirmation := wf.Confirmation
	if confirmation.Actor != actor || confirmation.WorkflowID != wf.WorkflowID || confirmation.RiskDecisionID != risk.DecisionID || confirmation.RecommendationID != candidateID.String() || confirmation.ProposedValue != risk.ResultingValue {
		return fmt.Errorf("canonical confirmation actor or binding does not match authenticated risk workflow")
	}
	if err := confirmation.ValidateFor(wf, now.UTC()); err != nil {
		return fmt.Errorf("canonical confirmation is invalid: %w", err)
	}
	var source, status string
	if err := tx.QueryRow(ctx, `SELECT source,status FROM candidate_trades WHERE id=$1 FOR UPDATE`, candidateID).Scan(&source, &status); err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(source), "world-monitor") {
		return fmt.Errorf("canonical decision only accepts World Monitor candidates")
	}
	decision := "approved"
	approvalStatus := candidatesmod.ApprovalStatusHumanApprovedPaper
	if wf.State == workflow.StateHumanRejected {
		if confirmation.Decision != workflow.ConfirmationReject {
			return fmt.Errorf("rejected workflow has no rejection confirmation")
		}
		decision = "rejected"
		approvalStatus = candidatesmod.ApprovalStatusHumanRejected
	} else if confirmation.Decision != workflow.ConfirmationApprove {
		return fmt.Errorf("paper-intent workflow has no approval confirmation")
	}
	if wf.State == workflow.StatePaperIntentCreated && status != candidatesmod.StatusAwaitingApproval && status != candidatesmod.StatusApproved {
		return fmt.Errorf("candidate is not awaiting canonical human approval")
	}
	if wf.State == workflow.StateHumanRejected && status != candidatesmod.StatusAwaitingApproval && status != candidatesmod.StatusRejected {
		return fmt.Errorf("candidate is not awaiting canonical human rejection")
	}
	record := CanonicalPaperDecision{CandidateID: candidateID.String(), WorkflowID: wf.WorkflowID, RiskID: risk.DecisionID,
		ConfirmID: confirmation.ConfirmationID, Instrument: confirmation.InstrumentID, Direction: confirmation.Direction,
		Value: risk.ResultingValue, Actor: actor, EconomicInputIdentity: economicInputIdentity, Decision: decision}
	notesBytes, err := json.Marshal(record)
	if err != nil {
		return err
	}
	notes := string(notesBytes)
	var priorID uuid.NullUUID
	var priorDecision, priorNotes string
	err = tx.QueryRow(ctx, `SELECT id,decision,COALESCE(notes,'') FROM candidate_approvals WHERE candidate_id=$1 AND decision IN ('approved','rejected') ORDER BY decided_at DESC,id DESC LIMIT 1 FOR UPDATE`, candidateID).Scan(&priorID, &priorDecision, &priorNotes)
	if err == nil {
		if priorDecision == decision && priorNotes == notes {
			return nil
		}
		return ErrCanonicalPaperDecisionConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO candidate_approvals(candidate_id,decision,approved_by,notes,decided_at,created_at) VALUES($1,$2,$3,$4,$5,$5)`, candidateID, decision, actor, notes, now.UTC()); err != nil {
		return err
	}
	statusUpdate := candidatesmod.StatusApproved
	if decision == "rejected" {
		statusUpdate = candidatesmod.StatusRejected
	}
	result, err := tx.Exec(ctx, `UPDATE candidate_trades SET status=$2,approval_status=$3,updated_at=$4 WHERE id=$1 AND source='world-monitor' AND status=$5`, candidateID, statusUpdate, approvalStatus, now.UTC(), candidatesmod.StatusAwaitingApproval)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrCanonicalPaperDecisionConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO candidate_events(candidate_id,event_type,from_status,to_status,detail,occurred_at) VALUES($1,$2,$3,$4,$5::jsonb,$6)`, candidateID, "canonical_paper_"+decision, candidatesmod.StatusAwaitingApproval, statusUpdate, notes, now.UTC())
	if err != nil {
		return err
	}
	return nil
}
