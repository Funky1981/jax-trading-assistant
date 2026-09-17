package exploratorypaper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/workflow"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPersistenceConflict = errors.New("exploratory paper persistence identity conflict")
	ErrFailedClosed        = errors.New("exploratory paper lifecycle failed closed")
)

type ApprovalSnapshot struct {
	Workflow    workflow.Workflow
	Events      []workflow.TransitionEvent
	PaperIntent workflow.PaperIntent
}

type EntrySnapshot struct {
	Order  papertrading.PaperOrder
	Fill   papertrading.PaperFill
	Ledger papertrading.PaperAccount
}

type ExitSnapshot struct {
	Order    papertrading.PaperOrder
	Fill     papertrading.PaperFill
	Ledger   papertrading.PaperAccount
	Approval *ApprovalSnapshot
}

type Review struct {
	ReviewID       string     `json:"reviewId"`
	PositionID     string     `json:"positionId"`
	SessionNumber  int        `json:"sessionNumber"`
	ScheduledAt    time.Time  `json:"scheduledAt"`
	Status         string     `json:"status"`
	EvidenceReview string     `json:"evidenceReviewId,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
}

type LifecycleRecord struct {
	LifecycleID string
	CandidateID string
	Thesis      FrozenThesis
	Binding     EntryBinding
	Position    Position
	Workflow    workflow.Workflow
	PaperIntent workflow.PaperIntent
	EntryOrder  papertrading.PaperOrder
	EntryFill   papertrading.PaperFill
	Reviews     []Review
	Checkpoints []Checkpoint
	Outcome     *Outcome
}

type PostgresStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PostgresStore) SaveApprovedEntry(ctx context.Context, candidateID string, thesis TradeThesis, binding EntryBinding, approval ApprovalSnapshot, position Position, entry EntrySnapshot) (string, error) {
	if s == nil || s.pool == nil {
		return "", fmt.Errorf("%w: database is unavailable", ErrFailedClosed)
	}
	if strings.TrimSpace(candidateID) == "" || binding.CandidateID != candidateID {
		return "", fmt.Errorf("%w: candidate identity is incomplete", ErrFailedClosed)
	}
	if err := validateApprovedEntry(thesis, binding, approval, position, entry); err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedClosed, err)
	}
	frozen, err := FreezeThesis(thesis, binding.BoundAt)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedClosed, err)
	}
	lifecycleID := lifecycleIdentity(thesis.ThesisID, frozen.ThesisHash)
	positionPayload, _ := json.Marshal(position)
	thesisPayload, _ := json.Marshal(thesis)
	bindingPayload, _ := json.Marshal(binding)
	policyPayload, _ := json.Marshal(binding.PolicyVersions)
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", fmt.Errorf("begin exploratory entry transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := persistWorkflowSnapshot(ctx, tx, approval); err != nil {
		return "", failClosed("workflow snapshot", err)
	}
	if err := persistPaperArtifacts(ctx, tx, entry.Order, entry.Fill, entry.Ledger); err != nil {
		return "", failClosed("paper artifacts", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO exploratory_paper_lifecycles (
			lifecycle_id,mode,environment,execution_authority,broker_execution_allowed,maximum_leverage,
			candidate_id,event_id,issuer_id,instrument_id,thesis_id,thesis_hash,evidence_set_hash,
			thesis_payload,binding_payload,workflow_id,paper_intent_id,position_id,position_payload,
			state,operational_state,entry_order_id,entry_fill_id,policy_versions,frozen_at,created_at,updated_at
		) VALUES ($1,'EXPLORATORY_PAPER','PAPER','NONE',FALSE,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$22)
		ON CONFLICT (lifecycle_id) DO NOTHING
	`, lifecycleID, binding.MaximumLeverage, candidateID, thesis.EventID, thesis.IssuerID, thesis.InstrumentID, thesis.ThesisID, frozen.ThesisHash, frozen.EvidenceSetHash, thesisPayload, bindingPayload, binding.WorkflowID, binding.PaperIntentID, position.PositionID, positionPayload, string(position.State), string(position.OperationalState), entry.Order.OrderID, entry.Fill.FillID, policyPayload, frozen.FrozenAt, now)
	if err != nil {
		return "", failClosed("exploratory lifecycle", err)
	}
	if err := ensureLifecycleIdentity(ctx, tx, lifecycleID, frozen.ThesisHash, binding.WorkflowID, position.PositionID); err != nil {
		return "", failClosed("exploratory lifecycle identity", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", failClosed("commit exploratory entry", err)
	}
	return lifecycleID, nil
}

func validateApprovedEntry(thesis TradeThesis, binding EntryBinding, approval ApprovalSnapshot, position Position, entry EntrySnapshot) error {
	if err := thesis.Validate(); err != nil {
		return err
	}
	if err := binding.Validate(); err != nil {
		return err
	}
	if binding.ThesisID != thesis.ThesisID || binding.WorkflowID != approval.Workflow.WorkflowID || binding.PaperIntentID != approval.PaperIntent.IntentID {
		return fmt.Errorf("approval identity chain is inconsistent")
	}
	if err := approval.Workflow.Validate(); err != nil {
		return err
	}
	if len(approval.Events) == 0 || workflow.ValidateAuditEvents(approval.Events) != nil {
		return fmt.Errorf("workflow audit is incomplete or divergent")
	}
	if approval.Workflow.State != workflow.StatePaperIntentCreated || approval.Workflow.PaperIntentID != approval.PaperIntent.IntentID {
		return fmt.Errorf("workflow is not in approved paper-intent state")
	}
	if err := approval.PaperIntent.Validate(); err != nil {
		return err
	}
	if approval.PaperIntent.WorkflowID != approval.Workflow.WorkflowID || approval.PaperIntent.InstrumentID != thesis.InstrumentID || approval.PaperIntent.Direction != string(thesis.Direction) {
		return fmt.Errorf("paper intent does not match thesis/workflow")
	}
	if err := entry.Order.Validate(); err != nil {
		return err
	}
	if err := entry.Fill.Validate(); err != nil {
		return err
	}
	if entry.Order.OrderID != entry.Fill.OrderID || entry.Order.PaperIntentID != binding.PaperIntentID || entry.Order.WorkflowID != binding.WorkflowID || entry.Fill.PaperIntentID != binding.PaperIntentID || entry.Fill.WorkflowID != binding.WorkflowID || entry.Fill.InstrumentID != thesis.InstrumentID || entry.Fill.Direction != string(thesis.Direction) {
		return fmt.Errorf("paper order/fill does not match the approval chain")
	}
	if _, err := papertrading.RestorePaperLedger(entry.Ledger); err != nil {
		return fmt.Errorf("paper ledger snapshot is invalid: %w", err)
	}
	if err := position.VerifyFrozenIdentity(); err != nil {
		return err
	}
	if err := position.VerifyApprovalBinding(binding); err != nil {
		return err
	}
	if position.PositionID == "" || position.Thesis.ThesisID != thesis.ThesisID || position.EntryPrice != entry.Fill.Price || position.EntryAt != entry.Fill.FilledAt {
		return fmt.Errorf("position does not match the entry fill")
	}
	return nil
}

func failClosed(stage string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrFailedClosed, stage, err)
}

func lifecycleIdentity(thesisID, thesisHash string) string {
	digest := sha256.Sum256([]byte(thesisID + "|" + thesisHash))
	return "epl_" + hex.EncodeToString(digest[:])
}

func bindingIdentity(binding EntryBinding) string {
	data, _ := json.Marshal(binding)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func persistWorkflowSnapshot(ctx context.Context, tx pgx.Tx, approval ApprovalSnapshot) error {
	payload, err := json.Marshal(struct {
		Workflow    workflow.Workflow    `json:"workflow"`
		PaperIntent workflow.PaperIntent `json:"paperIntent"`
	}{approval.Workflow, approval.PaperIntent})
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		INSERT INTO workflow_instances (workflow_id,contract_version,algorithm,state,revision,recommendation_id,risk_decision_id,portfolio_snapshot_id,analytics_id,policy_id,proposal_id,payload,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (workflow_id) DO UPDATE SET payload=EXCLUDED.payload,updated_at=EXCLUDED.updated_at
		WHERE workflow_instances.payload = EXCLUDED.payload
	`, approval.Workflow.WorkflowID, approval.Workflow.ContractVersion, approval.Workflow.Algorithm, string(approval.Workflow.State), approval.Workflow.Revision, approval.Workflow.RecommendationID, approval.Workflow.RiskDecisionID, approval.Workflow.PortfolioSnapshotID, approval.Workflow.AnalyticsID, approval.Workflow.PolicyID, nullString(approval.Workflow.ProposalID), payload, approval.Workflow.CreatedAt, approval.Workflow.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		var existing []byte
		if err := tx.QueryRow(ctx, `SELECT payload FROM workflow_instances WHERE workflow_id=$1`, approval.Workflow.WorkflowID).Scan(&existing); err != nil {
			return err
		}
		if string(existing) != string(payload) {
			return ErrPersistenceConflict
		}
	}
	for _, event := range approval.Events {
		_, err = tx.Exec(ctx, `
			INSERT INTO workflow_audit_events (event_id,workflow_id,sequence,previous_state,next_state,action,actor,actor_role,idempotency_key,input_fingerprint,content_identity,reason,occurred_at,transition_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT (event_id) DO NOTHING
		`, event.EventID, event.WorkflowID, event.Sequence, string(event.PreviousState), string(event.NextState), string(event.Action), event.Actor, string(event.ActorRole), event.IdempotencyKey, event.InputFingerprint, event.ContentIdentity, nullString(event.Reason), event.OccurredAt, event.TransitionVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func persistPaperArtifacts(ctx context.Context, tx pgx.Tx, order papertrading.PaperOrder, fill papertrading.PaperFill, account papertrading.PaperAccount) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO paper_orders (order_id,contract_version,venue_id,environment,paper_intent_id,workflow_id,recommendation_id,risk_decision_id,confirmation_id,instrument_id,direction,quantity,remaining_quantity,filled_quantity,order_type,limit_price,cost_model_id,created_at,activates_at,status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		ON CONFLICT (order_id) DO NOTHING
	`, order.OrderID, order.ContractVersion, order.VenueID, string(order.Environment), order.PaperIntentID, order.WorkflowID, order.RecommendationID, order.RiskDecisionID, order.ConfirmationID, order.InstrumentID, order.Direction, order.Quantity, order.RemainingQuantity, order.FilledQuantity, string(order.OrderType), nullableFloat(order.LimitPrice), order.CostModelID, order.CreatedAt, order.ActivatesAt, string(order.Status))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO paper_fills (fill_id,contract_version,order_id,paper_intent_id,workflow_id,instrument_id,direction,quantity,price,cost_model_id,mid_price,spread_cost,slippage_cost,commission,executed_price,filled_at,tick_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (fill_id) DO NOTHING
	`, fill.FillID, fill.ContractVersion, fill.OrderID, fill.PaperIntentID, fill.WorkflowID, fill.InstrumentID, fill.Direction, fill.Quantity, fill.Price, fill.Costs.ModelID, fill.Costs.MidPrice, fill.Costs.SpreadCost, fill.Costs.SlippageCost, fill.Costs.Commission, fill.Costs.ExecutedPrice, fill.FilledAt, fill.TickID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO paper_accounts (account_id,contract_version,environment,currency,initial_cash,cash,equity,realized_pnl,fees,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (account_id) DO UPDATE SET cash=EXCLUDED.cash,equity=EXCLUDED.equity,realized_pnl=EXCLUDED.realized_pnl,fees=EXCLUDED.fees,updated_at=EXCLUDED.updated_at
	`, account.AccountID, account.ContractVersion, string(account.Environment), account.Currency, account.InitialCash, account.Cash, account.Equity, account.RealizedPnL, account.Fees, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, event := range account.Events {
		if err := event.Validate(); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO paper_ledger_events (event_id,contract_version,account_id,fill_id,order_id,workflow_id,instrument_id,direction,quantity,price,fee,cash_delta,realized_pnl,occurred_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT (event_id) DO NOTHING
		`, event.EventID, event.ContractVersion, account.AccountID, event.FillID, event.OrderID, event.WorkflowID, event.InstrumentID, event.Direction, event.Quantity, event.Price, event.Fee, event.CashDelta, event.RealizedPnL, event.OccurredAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureLifecycleIdentity(ctx context.Context, tx pgx.Tx, lifecycleID, thesisHash, workflowID, positionID string) error {
	var gotThesis, gotWorkflow, gotPosition string
	err := tx.QueryRow(ctx, `SELECT thesis_hash,workflow_id,position_id FROM exploratory_paper_lifecycles WHERE lifecycle_id=$1`, lifecycleID).Scan(&gotThesis, &gotWorkflow, &gotPosition)
	if err != nil {
		return err
	}
	if gotThesis != thesisHash || gotWorkflow != workflowID || gotPosition != positionID {
		return ErrPersistenceConflict
	}
	return nil
}

func (s *PostgresStore) PersistReviewSchedule(ctx context.Context, positionID string, schedule ReviewSchedule) error {
	if positionID == "" || schedule.PositionID != positionID || len(schedule.Sessions) == 0 {
		return fmt.Errorf("review schedule identity is invalid")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, scheduled := range schedule.Sessions {
		session := i + 1
		reviewID := reviewIdentity(positionID, session)
		payload, _ := json.Marshal(Review{ReviewID: reviewID, PositionID: positionID, SessionNumber: session, ScheduledAt: scheduled.UTC(), Status: "PENDING"})
		_, err = tx.Exec(ctx, `INSERT INTO exploratory_paper_reviews(review_id,position_id,session_number,scheduled_at,status,idempotency_identity,payload) VALUES($1,$2,$3,$4,'PENDING',$5,$6) ON CONFLICT(position_id,session_number) DO NOTHING`, reviewID, positionID, session, scheduled.UTC(), reviewID, payload)
		if err != nil {
			return failClosed("review schedule", err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE exploratory_paper_lifecycles SET next_review_at=(SELECT MIN(scheduled_at) FROM exploratory_paper_reviews WHERE position_id=$1 AND status='PENDING'),updated_at=$2 WHERE position_id=$1`, positionID, s.now().UTC())
	if err != nil {
		return failClosed("next review", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return failClosed("commit review schedule", err)
	}
	return nil
}

func reviewIdentity(positionID string, session int) string {
	digest := sha256.Sum256([]byte(positionID + "|review|" + fmt.Sprint(session)))
	return "epr_" + hex.EncodeToString(digest[:])
}

func (s *PostgresStore) RecordReviewUnavailable(ctx context.Context, positionID string, session int, at time.Time, reason string) error {
	if at.IsZero() || at.Location() != time.UTC || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("review-unavailable record is invalid")
	}
	reviewID := reviewIdentity(positionID, session)
	payload, _ := json.Marshal(Review{ReviewID: reviewID, PositionID: positionID, SessionNumber: session, Status: "MISSING_DATA", Reason: reason, ProcessedAt: &at})
	_, err := s.pool.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='MISSING_DATA',payload=$4,processed_at=$3 WHERE review_id=$1 AND position_id=$2 AND status='PENDING'`, reviewID, positionID, at, payload)
	if err != nil {
		return failClosed("record unavailable review", err)
	}
	return nil
}

func (s *PostgresStore) ApplyEvidence(ctx context.Context, positionID string, evidence RelevantEvidence, at time.Time, policyVersion string) (Position, error) {
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return Position{}, err
	}
	position := record.Position
	if err := position.Reassess(evidence, at, policyVersion); err != nil {
		return Position{}, err
	}
	fingerprint := relevantEvidenceFingerprint(evidence)
	payload, _ := json.Marshal(evidence)
	positionPayload, _ := json.Marshal(position)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Position{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var prior string
	err = tx.QueryRow(ctx, `SELECT evidence_fingerprint FROM exploratory_paper_evidence_reassessments WHERE position_id=$1 AND evidence_id=$2`, positionID, evidence.Reference.EvidenceID).Scan(&prior)
	if err == nil {
		if prior != fingerprint {
			return Position{}, fmt.Errorf("%w: evidence %s", ErrPersistenceConflict, evidence.Reference.EvidenceID)
		}
		return record.Position, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Position{}, failClosed("load processed evidence", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO exploratory_paper_evidence_reassessments(position_id,evidence_id,evidence_fingerprint,payload,processed_at) VALUES($1,$2,$3,$4,$5)`, positionID, evidence.Reference.EvidenceID, fingerprint, payload, at)
	if err != nil {
		return Position{}, failClosed("persist processed evidence", err)
	}
	_, err = tx.Exec(ctx, `UPDATE exploratory_paper_lifecycles SET position_payload=$2,state=$3,operational_state=$4,updated_at=$5 WHERE position_id=$1`, positionID, positionPayload, string(position.State), string(position.OperationalState), s.now().UTC())
	if err != nil {
		return Position{}, failClosed("persist reassessed position", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Position{}, failClosed("commit reassessment", err)
	}
	return position, nil
}

func (s *PostgresStore) PersistCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	if checkpoint.PositionID == "" || checkpoint.ThesisID == "" || checkpoint.Sessions < 1 || checkpoint.Sessions > 5 || checkpoint.At.IsZero() || checkpoint.At.Location() != time.UTC || checkpoint.Price <= 0 || checkpoint.PriceSource == "" || checkpoint.DataQuality == "" {
		return fmt.Errorf("checkpoint is invalid")
	}
	id := checkpointIdentity(checkpoint)
	payload, _ := json.Marshal(checkpoint)
	_, err := s.pool.Exec(ctx, `INSERT INTO exploratory_paper_checkpoints(checkpoint_id,position_id,thesis_id,session_number,observed_at,payload) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(position_id,session_number) DO NOTHING`, id, checkpoint.PositionID, checkpoint.ThesisID, checkpoint.Sessions, checkpoint.At, payload)
	if err != nil {
		return failClosed("persist checkpoint", err)
	}
	_, err = s.pool.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='COMPLETED',payload=$3,processed_at=$2 WHERE position_id=$1 AND session_number=$4 AND status='PENDING'`, checkpoint.PositionID, checkpoint.At, payload, checkpoint.Sessions)
	return err
}

func checkpointIdentity(c Checkpoint) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s", c.PositionID, c.ThesisID, c.Sessions, c.At.UTC().Format(time.RFC3339Nano))))
	return "epc_" + hex.EncodeToString(digest[:])
}

func (s *PostgresStore) PersistOutcome(ctx context.Context, positionID string, exit ExitSnapshot, outcome Outcome) error {
	if err := outcome.Validate(); err != nil {
		return err
	}
	if positionID != outcome.PositionID || exit.Fill.FillID != outcome.ExitFillID || exit.Order.OrderID != outcome.ExitOrderID {
		return fmt.Errorf("outcome exit identity is inconsistent")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load position for outcome close", err)
	}
	closedPosition := record.Position
	if closedPosition.State != StateClosed {
		if err := closedPosition.Close(outcome.ExitAt); err != nil {
			return failClosed("close position for outcome", err)
		}
	}
	positionPayload, err := json.Marshal(closedPosition)
	if err != nil {
		return failClosed("encode closed position", err)
	}
	if err := exit.Order.Validate(); err != nil {
		return err
	}
	if err := exit.Fill.Validate(); err != nil {
		return err
	}
	payload, _ := json.Marshal(outcome)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if exit.Approval != nil {
		if err := persistWorkflowSnapshot(ctx, tx, *exit.Approval); err != nil {
			return failClosed("persist exit workflow", err)
		}
	}
	if err := persistPaperArtifacts(ctx, tx, exit.Order, exit.Fill, exit.Ledger); err != nil {
		return failClosed("persist exit artifacts", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO exploratory_paper_outcomes(outcome_id,position_id,thesis_id,mode,entry_fill_id,exit_fill_id,payload,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(outcome_id) DO NOTHING`, outcome.OutcomeID, positionID, outcome.ThesisID, outcome.Mode, outcome.EntryFillID, outcome.ExitFillID, payload, s.now().UTC())
	if err != nil {
		return failClosed("persist outcome", err)
	}
	_, err = tx.Exec(ctx, `UPDATE exploratory_paper_lifecycles SET exit_order_id=$2,exit_fill_id=$3,outcome_id=$4,outcome_payload=$5,position_payload=$6,state='CLOSED',operational_state='CLOSED',next_review_at=NULL,updated_at=$7 WHERE position_id=$1`, positionID, outcome.ExitOrderID, outcome.ExitFillID, outcome.OutcomeID, payload, positionPayload, s.now().UTC())
	if err != nil {
		return failClosed("close exploratory lifecycle", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return failClosed("commit outcome", err)
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, positionID string) (LifecycleRecord, error) {
	var record LifecycleRecord
	var thesisPayload, bindingPayload, positionPayload []byte
	var outcomePayload []byte
	var frozenAt time.Time
	var lifecycleID, candidateID, thesisHash, evidenceHash, workflowID, paperIntentID, entryOrderID, entryFillID string
	err := s.pool.QueryRow(ctx, `SELECT lifecycle_id,candidate_id,thesis_hash,evidence_set_hash,thesis_payload,binding_payload,workflow_id,paper_intent_id,position_id,position_payload,entry_order_id,entry_fill_id,outcome_payload,frozen_at FROM exploratory_paper_lifecycles WHERE position_id=$1`, positionID).Scan(&lifecycleID, &candidateID, &thesisHash, &evidenceHash, &thesisPayload, &bindingPayload, &workflowID, &paperIntentID, &record.Position.PositionID, &positionPayload, &entryOrderID, &entryFillID, &outcomePayload, &frozenAt)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(thesisPayload, &record.Thesis.Thesis); err != nil {
		return record, failClosed("decode frozen thesis", err)
	}
	if err := json.Unmarshal(bindingPayload, &record.Binding); err != nil {
		return record, failClosed("decode exploratory lifecycle", err)
	}
	if err := json.Unmarshal(positionPayload, &record.Position); err != nil {
		return record, failClosed("decode exploratory position", err)
	}
	record.Thesis.FrozenAt = frozenAt.UTC()
	record.LifecycleID, record.CandidateID, record.Thesis.ThesisHash, record.Thesis.EvidenceSetHash = lifecycleID, candidateID, thesisHash, evidenceHash
	if record.Binding.WorkflowID != workflowID || record.Binding.PaperIntentID != paperIntentID || record.Position.PositionID != positionID || record.Position.FrozenThesisHash != thesisHash || record.Position.EvidenceSetHash != evidenceHash || record.Position.BindingHash != bindingIdentity(record.Binding) {
		return record, failClosed("frozen lifecycle identity", ErrPersistenceConflict)
	}
	if err := record.Position.VerifyFrozenIdentity(); err != nil {
		return record, failClosed("verify frozen position", err)
	}
	if err := record.Binding.Validate(); err != nil {
		return record, failClosed("verify entry binding", err)
	}
	if err := s.loadWorkflowAndPaperArtifacts(ctx, &record, workflowID, paperIntentID, entryOrderID, entryFillID); err != nil {
		return record, failClosed("restore approval and paper artifacts", err)
	}
	if len(outcomePayload) > 0 {
		var outcome Outcome
		if err := json.Unmarshal(outcomePayload, &outcome); err != nil {
			return record, failClosed("decode outcome", err)
		}
		if err := outcome.Validate(); err != nil {
			return record, failClosed("verify outcome", err)
		}
		record.Outcome = &outcome
	}
	rows, err := s.pool.Query(ctx, `SELECT review_id,position_id,session_number,scheduled_at,status,payload,processed_at FROM exploratory_paper_reviews WHERE position_id=$1 ORDER BY session_number`, positionID)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	for rows.Next() {
		var review Review
		var payload []byte
		if err := rows.Scan(&review.ReviewID, &review.PositionID, &review.SessionNumber, &review.ScheduledAt, &review.Status, &payload, &review.ProcessedAt); err != nil {
			return record, err
		}
		if err := json.Unmarshal(payload, &review); err != nil {
			return record, failClosed("decode review", err)
		}
		record.Reviews = append(record.Reviews, review)
	}
	rows, err = s.pool.Query(ctx, `SELECT payload FROM exploratory_paper_checkpoints WHERE position_id=$1 ORDER BY session_number`, positionID)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	for rows.Next() {
		var payload []byte
		var checkpoint Checkpoint
		if err := rows.Scan(&payload); err != nil || json.Unmarshal(payload, &checkpoint) != nil {
			return record, failClosed("decode checkpoint", err)
		}
		record.Checkpoints = append(record.Checkpoints, checkpoint)
	}
	return record, rows.Err()
}

func (s *PostgresStore) loadWorkflowAndPaperArtifacts(ctx context.Context, record *LifecycleRecord, workflowID, paperIntentID, orderID, fillID string) error {
	var workflowPayload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM workflow_instances WHERE workflow_id=$1`, workflowID).Scan(&workflowPayload); err != nil {
		return err
	}
	var snapshot struct {
		Workflow    workflow.Workflow    `json:"workflow"`
		PaperIntent workflow.PaperIntent `json:"paperIntent"`
	}
	if err := json.Unmarshal(workflowPayload, &snapshot); err != nil {
		return err
	}
	if err := snapshot.Workflow.Validate(); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if err := snapshot.PaperIntent.Validate(); err != nil {
		return fmt.Errorf("paper intent: %w", err)
	}
	if snapshot.Workflow.WorkflowID != workflowID || snapshot.Workflow.PaperIntentID != paperIntentID || snapshot.PaperIntent.IntentID != paperIntentID {
		return fmt.Errorf("workflow/paper intent identity: %w", ErrPersistenceConflict)
	}
	record.Workflow, record.PaperIntent = snapshot.Workflow, snapshot.PaperIntent
	var order papertrading.PaperOrder
	var environment, direction, orderType, status string
	var limitPrice *float64
	err := s.pool.QueryRow(ctx, `SELECT order_id,contract_version,venue_id,environment,paper_intent_id,workflow_id,recommendation_id,risk_decision_id,confirmation_id,instrument_id,direction,quantity,remaining_quantity,filled_quantity,order_type,limit_price,cost_model_id,created_at,activates_at,status FROM paper_orders WHERE order_id=$1`, orderID).Scan(&order.OrderID, &order.ContractVersion, &order.VenueID, &environment, &order.PaperIntentID, &order.WorkflowID, &order.RecommendationID, &order.RiskDecisionID, &order.ConfirmationID, &order.InstrumentID, &direction, &order.Quantity, &order.RemainingQuantity, &order.FilledQuantity, &orderType, &limitPrice, &order.CostModelID, &order.CreatedAt, &order.ActivatesAt, &status)
	if err != nil {
		return err
	}
	order.Environment, order.Direction, order.OrderType, order.Status = papertrading.Environment(environment), direction, papertrading.OrderType(orderType), papertrading.PaperOrderStatus(status)
	order.CreatedAt, order.ActivatesAt = order.CreatedAt.UTC(), order.ActivatesAt.UTC()
	if limitPrice != nil {
		order.LimitPrice = *limitPrice
	}
	if err := order.Validate(); err != nil {
		return fmt.Errorf("paper order: %w (id=%s env=%s qty=%v remaining=%v filled=%v type=%s created=%s activates=%s)", err, order.OrderID, order.Environment, order.Quantity, order.RemainingQuantity, order.FilledQuantity, order.OrderType, order.CreatedAt.Format(time.RFC3339Nano), order.ActivatesAt.Format(time.RFC3339Nano))
	}
	if order.OrderID != orderID || order.PaperIntentID != paperIntentID || order.WorkflowID != workflowID {
		return fmt.Errorf("paper order identity: %w", ErrPersistenceConflict)
	}
	var fill papertrading.PaperFill
	var fillDirection string
	var executedPrice *float64
	err = s.pool.QueryRow(ctx, `SELECT fill_id,contract_version,order_id,paper_intent_id,workflow_id,instrument_id,direction,quantity,price,cost_model_id,mid_price,spread_cost,slippage_cost,commission,executed_price,filled_at,tick_id FROM paper_fills WHERE fill_id=$1`, fillID).Scan(&fill.FillID, &fill.ContractVersion, &fill.OrderID, &fill.PaperIntentID, &fill.WorkflowID, &fill.InstrumentID, &fillDirection, &fill.Quantity, &fill.Price, &fill.Costs.ModelID, &fill.Costs.MidPrice, &fill.Costs.SpreadCost, &fill.Costs.SlippageCost, &fill.Costs.Commission, &executedPrice, &fill.FilledAt, &fill.TickID)
	if err != nil {
		return err
	}
	fill.Direction = fillDirection
	fill.FilledAt = fill.FilledAt.UTC()
	if executedPrice == nil {
		return fmt.Errorf("paper fill executed price is missing")
	}
	fill.Costs.ExecutedPrice = *executedPrice
	if err := fill.Validate(); err != nil {
		return fmt.Errorf("paper fill: %w (id=%s order=%s direction=%s qty=%v price=%v model=%s mid=%v spread=%v slip=%v commission=%v filled=%s tick=%s)", err, fill.FillID, fill.OrderID, fill.Direction, fill.Quantity, fill.Price, fill.Costs.ModelID, fill.Costs.MidPrice, fill.Costs.SpreadCost, fill.Costs.SlippageCost, fill.Costs.Commission, fill.FilledAt.Format(time.RFC3339Nano), fill.TickID)
	}
	if fill.FillID != fillID || fill.OrderID != orderID || fill.PaperIntentID != paperIntentID || fill.WorkflowID != workflowID {
		return fmt.Errorf("paper fill identity: %w", ErrPersistenceConflict)
	}
	record.EntryOrder, record.EntryFill = order, fill
	return nil
}

func (s *PostgresStore) ListActive(ctx context.Context, limit int) ([]LifecycleRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `SELECT position_id FROM exploratory_paper_lifecycles WHERE state='ACTIVE' ORDER BY updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []LifecycleRecord
	for rows.Next() {
		var positionID string
		if err := rows.Scan(&positionID); err != nil {
			return nil, err
		}
		record, err := s.Get(ctx, positionID)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableFloat(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}
