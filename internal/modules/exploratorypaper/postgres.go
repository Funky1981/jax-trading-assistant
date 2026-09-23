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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPersistenceConflict = errors.New("exploratory paper persistence identity conflict")
	ErrFailedClosed        = errors.New("exploratory paper lifecycle failed closed")
)

type ApprovalSnapshot struct {
	Workflow    workflow.Workflow          `json:"workflow"`
	Events      []workflow.TransitionEvent `json:"events"`
	PaperIntent workflow.PaperIntent       `json:"paperIntent"`
	ExitBinding *ExitApprovalBinding       `json:"exitBinding,omitempty"`
}

// ExitApprovalBinding links an operator's existing workflow approval artifact
// to the exact durable recommendation and lifecycle that requested it. The
// workflow and paper intent remain owned by the existing workflow architecture;
// this binding prevents an otherwise valid paper workflow from being replayed
// against a different position or review.
type ExitApprovalBinding struct {
	PositionID         string `json:"positionId"`
	ReviewID           string `json:"reviewId"`
	RecommendationID   string `json:"recommendationId"`
	EntryWorkflowID    string `json:"entryWorkflowId"`
	EntryPaperIntentID string `json:"entryPaperIntentId"`
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
	ReviewID               string            `json:"reviewId"`
	PositionID             string            `json:"positionId"`
	SessionNumber          int               `json:"sessionNumber"`
	ScheduledAt            time.Time         `json:"scheduledAt"`
	Status                 string            `json:"status"`
	EvidenceReview         string            `json:"evidenceReviewId,omitempty"`
	Reason                 string            `json:"reason,omitempty"`
	ExitAction             string            `json:"exitAction,omitempty"`
	ExitReason             string            `json:"exitReason,omitempty"`
	ExitApprovalWorkflowID string            `json:"exitApprovalWorkflowId,omitempty"`
	ExitRecommendationID   string            `json:"exitRecommendationId,omitempty"`
	ExitApproval           *ApprovalSnapshot `json:"exitApproval,omitempty"`
	ProcessedAt            *time.Time        `json:"processedAt,omitempty"`
}

type ReviewRecord struct {
	Lifecycle LifecycleRecord
	Review    Review
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
	pool      *pgxpool.Pool
	now       func() time.Time
	accountID string
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func NewPostgresStoreForAccount(pool *pgxpool.Pool, accountID string) *PostgresStore {
	return &PostgresStore{pool: pool, accountID: strings.TrimSpace(accountID), now: func() time.Time { return time.Now().UTC() }}
}

func (s *PostgresStore) requireAccount(accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return fmt.Errorf("paper account identity is missing")
	}
	if s.accountID != "" && accountID != s.accountID {
		return fmt.Errorf("paper account identity %q does not match runtime account %q", accountID, s.accountID)
	}
	return nil
}

func (s *PostgresStore) requireFillAccount(ctx context.Context, fillID string) error {
	if s.accountID == "" {
		return nil
	}
	var accountID string
	if err := s.pool.QueryRow(ctx, `SELECT account_id FROM paper_ledger_events WHERE fill_id=$1`, fillID).Scan(&accountID); err != nil {
		return fmt.Errorf("fill %s account ownership is unavailable: %w", fillID, err)
	}
	if accountID != s.accountID {
		return fmt.Errorf("fill %s belongs to paper account %q, runtime account is %q", fillID, accountID, s.accountID)
	}
	return nil
}

// RestorePaperVenue is intentionally unavailable without an explicit account
// scope. A runtime restart must not silently combine economic artifacts from
// multiple paper accounts.
func (s *PostgresStore) RestorePaperVenue(ctx context.Context, contract papertrading.CapabilityContract, costModel papertrading.CostModel) (*papertrading.PaperVenue, error) {
	return nil, fmt.Errorf("paper venue restore requires an explicit paper account")
}

// RestorePaperVenueForAccount rebuilds the paper-only venue from durable
// artifacts owned by accountID. Ownership is derived only from the durable
// ledger event for each fill; missing or conflicting ownership fails closed.
func (s *PostgresStore) RestorePaperVenueForAccount(ctx context.Context, accountID string, contract papertrading.CapabilityContract, costModel papertrading.CostModel) (*papertrading.PaperVenue, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("paper venue restore requires a database")
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("paper venue restore requires a non-empty paper account")
	}
	if err := s.requireAccount(accountID); err != nil {
		return nil, fmt.Errorf("paper venue restore account scope: %w", err)
	}
	orderOwners, fillOwners, err := s.loadPaperArtifactOwnership(ctx)
	if err != nil {
		return nil, failClosed("restore paper artifact ownership", err)
	}
	orders := map[string]papertrading.PaperOrder{}
	orderRows, err := s.pool.Query(ctx, `
		SELECT order_id,contract_version,venue_id,environment,paper_intent_id,workflow_id,recommendation_id,risk_decision_id,confirmation_id,
		       instrument_id,direction,quantity,remaining_quantity,filled_quantity,order_type,limit_price,cost_model_id,created_at,activates_at,status
		FROM paper_orders ORDER BY created_at,order_id
	`)
	if err != nil {
		return nil, fmt.Errorf("load paper orders for restore: %w", err)
	}
	for orderRows.Next() {
		var order papertrading.PaperOrder
		var environment, direction, orderType, status string
		var limitPrice *float64
		if err := orderRows.Scan(&order.OrderID, &order.ContractVersion, &order.VenueID, &environment, &order.PaperIntentID, &order.WorkflowID, &order.RecommendationID, &order.RiskDecisionID, &order.ConfirmationID, &order.InstrumentID, &direction, &order.Quantity, &order.RemainingQuantity, &order.FilledQuantity, &orderType, &limitPrice, &order.CostModelID, &order.CreatedAt, &order.ActivatesAt, &status); err != nil {
			orderRows.Close()
			return nil, fmt.Errorf("scan paper order for restore: %w", err)
		}
		order.Environment = papertrading.Environment(environment)
		order.Direction = direction
		order.OrderType = papertrading.OrderType(orderType)
		order.Status = papertrading.PaperOrderStatus(status)
		order.CreatedAt = order.CreatedAt.UTC()
		order.ActivatesAt = order.ActivatesAt.UTC()
		if limitPrice != nil {
			order.LimitPrice = *limitPrice
		}
		if orderOwners[order.OrderID] == accountID {
			orders[order.OrderID] = order
		}
	}
	if err := orderRows.Err(); err != nil {
		orderRows.Close()
		return nil, fmt.Errorf("read paper orders for restore: %w", err)
	}
	orderRows.Close()

	fills := map[string]papertrading.PaperFill{}
	lastTick := time.Time{}
	fillRows, err := s.pool.Query(ctx, `
		SELECT fill_id,contract_version,order_id,paper_intent_id,workflow_id,instrument_id,direction,quantity,price,
		       cost_model_id,mid_price,spread_cost,slippage_cost,commission,executed_price,filled_at,tick_id
		FROM paper_fills ORDER BY filled_at,fill_id
	`)
	if err != nil {
		return nil, fmt.Errorf("load paper fills for restore: %w", err)
	}
	for fillRows.Next() {
		var fill papertrading.PaperFill
		var direction string
		var executedPrice *float64
		if err := fillRows.Scan(&fill.FillID, &fill.ContractVersion, &fill.OrderID, &fill.PaperIntentID, &fill.WorkflowID, &fill.InstrumentID, &direction, &fill.Quantity, &fill.Price, &fill.Costs.ModelID, &fill.Costs.MidPrice, &fill.Costs.SpreadCost, &fill.Costs.SlippageCost, &fill.Costs.Commission, &executedPrice, &fill.FilledAt, &fill.TickID); err != nil {
			fillRows.Close()
			return nil, fmt.Errorf("scan paper fill for restore: %w", err)
		}
		fill.Direction = direction
		fill.FilledAt = fill.FilledAt.UTC()
		if executedPrice != nil {
			fill.Costs.ExecutedPrice = *executedPrice
		} else {
			fill.Costs.ExecutedPrice = fill.Price
		}
		if fillOwners[fill.FillID] == accountID {
			fills[fill.FillID] = fill
			if lastTick.IsZero() || fill.FilledAt.After(lastTick) {
				lastTick = fill.FilledAt
			}
		}
	}
	if err := fillRows.Err(); err != nil {
		fillRows.Close()
		return nil, fmt.Errorf("read paper fills for restore: %w", err)
	}
	fillRows.Close()
	return papertrading.RestorePaperVenue(papertrading.VenueSnapshot{
		Contract: contract, CostModel: costModel, Orders: orders, Fills: fills,
		ProcessedTicks: map[string]string{}, CommandKeys: map[string]string{}, LastTick: lastTick,
	})
}

func (s *PostgresStore) loadPaperArtifactOwnership(ctx context.Context) (map[string]string, map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.fill_id,f.order_id,le.account_id,le.order_id
		FROM paper_fills f
		LEFT JOIN paper_ledger_events le ON le.fill_id=f.fill_id
		ORDER BY f.fill_id
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	orderOwners := map[string]string{}
	fillOwners := map[string]string{}
	for rows.Next() {
		var fillID, orderID string
		var accountID, ledgerOrderID *string
		if err := rows.Scan(&fillID, &orderID, &accountID, &ledgerOrderID); err != nil {
			return nil, nil, err
		}
		if accountID == nil || ledgerOrderID == nil || strings.TrimSpace(*accountID) == "" {
			return nil, nil, fmt.Errorf("fill %s has no provable paper account owner", fillID)
		}
		if *ledgerOrderID != orderID {
			return nil, nil, fmt.Errorf("fill %s ledger order %s conflicts with fill order %s", fillID, *ledgerOrderID, orderID)
		}
		fillOwners[fillID] = *accountID
		if prior, ok := orderOwners[orderID]; ok && prior != *accountID {
			return nil, nil, fmt.Errorf("order %s has conflicting paper account owners %s and %s", orderID, prior, *accountID)
		}
		orderOwners[orderID] = *accountID
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	orderRows, err := s.pool.Query(ctx, `SELECT order_id FROM paper_orders ORDER BY order_id`)
	if err != nil {
		return nil, nil, err
	}
	defer orderRows.Close()
	for orderRows.Next() {
		var orderID string
		if err := orderRows.Scan(&orderID); err != nil {
			return nil, nil, err
		}
		if _, ok := orderOwners[orderID]; !ok {
			return nil, nil, fmt.Errorf("order %s has no provable paper account owner", orderID)
		}
	}
	if err := orderRows.Err(); err != nil {
		return nil, nil, err
	}
	return orderOwners, fillOwners, nil
}

func (s *PostgresStore) SaveApprovedEntry(ctx context.Context, candidateID string, thesis TradeThesis, binding EntryBinding, approval ApprovalSnapshot, position Position, entry EntrySnapshot) (string, error) {
	if s == nil || s.pool == nil {
		return "", fmt.Errorf("%w: database is unavailable", ErrFailedClosed)
	}
	if err := s.requireAccount(entry.Ledger.AccountID); err != nil {
		return "", failClosed("approved entry account binding", err)
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

// QueueApprovedEntry is the narrow handoff from an already approved canonical
// workflow to the exploratory runtime. The payload is treated as untrusted;
// LoadApprovedEntries replaces its evidence with the persisted projection.
func (s *PostgresStore) QueueApprovedEntry(ctx context.Context, entry EntryRequest) error {
	if s == nil || s.pool == nil || strings.TrimSpace(entry.CandidateID) == "" || strings.TrimSpace(entry.Approval.Workflow.WorkflowID) == "" || strings.TrimSpace(entry.Approval.PaperIntent.IntentID) == "" {
		return fmt.Errorf("%w: approved entry queue identity is incomplete", ErrFailedClosed)
	}
	if err := s.requireAccount(entry.Ledger.AccountID); err != nil {
		return failClosed("queue approved entry account binding", err)
	}
	if err := entry.Approval.Workflow.Validate(); err != nil || entry.Approval.Workflow.State != workflow.StatePaperIntentCreated || entry.Approval.Workflow.Confirmation == nil {
		return fmt.Errorf("%w: queue requires a human-approved paper intent", ErrFailedClosed)
	}
	if err := entry.Approval.Workflow.Confirmation.ValidateFor(entry.Approval.Workflow, s.now().UTC()); err != nil {
		return fmt.Errorf("%w: human approval is stale or mismatched", ErrFailedClosed)
	}
	if err := entry.Approval.PaperIntent.Validate(); err != nil {
		return fmt.Errorf("%w: paper intent: %v", ErrFailedClosed, err)
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	requestID := queueIdentity(entry.CandidateID, entry.Approval.Workflow.WorkflowID, entry.Approval.PaperIntent.IntentID)
	now := s.now().UTC()
	result, err := s.pool.Exec(ctx, `INSERT INTO exploratory_paper_entry_queue(request_id,candidate_id,payload,status,created_at,updated_at) VALUES($1,$2,$3,'PENDING',$4,$4) ON CONFLICT(candidate_id) DO UPDATE SET payload=EXCLUDED.payload,updated_at=EXCLUDED.updated_at,status='PENDING' WHERE exploratory_paper_entry_queue.payload=EXCLUDED.payload`, requestID, entry.CandidateID, payload, now)
	if err != nil {
		return failClosed("queue approved entry", err)
	}
	if result.RowsAffected() == 0 {
		var existing []byte
		if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_entry_queue WHERE candidate_id=$1`, entry.CandidateID).Scan(&existing); err != nil {
			return failClosed("verify queued entry identity", err)
		}
		if string(existing) != string(payload) {
			return failClosed("queued entry identity", ErrPersistenceConflict)
		}
	}
	return nil
}

func queueIdentity(candidateID, workflowID, intentID string) string {
	digest := sha256.Sum256([]byte(candidateID + "|" + workflowID + "|" + intentID))
	return "epq_" + hex.EncodeToString(digest[:])
}

// LoadApprovedEntries is the production EntrySource. It loads the approved
// workflow handoff, then replaces caller-provided evidence with the canonical
// candidate_evidence_scores/items projection before the runtime can act.
func (s *PostgresStore) LoadApprovedEntries(ctx context.Context) ([]EntryRequest, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM exploratory_paper_entry_queue WHERE status='PENDING' ORDER BY created_at FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []EntryRequest
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var entry EntryRequest
		if err := json.Unmarshal(payload, &entry); err != nil {
			return nil, failClosed("decode approved entry queue", err)
		}
		if s.accountID != "" {
			if strings.TrimSpace(entry.Ledger.AccountID) == "" {
				return nil, failClosed("approved entry account binding", fmt.Errorf("queued entry has no paper account identity"))
			}
			if entry.Ledger.AccountID != s.accountID {
				continue
			}
		}
		if err := s.loadCanonicalEvidence(ctx, &entry); err != nil {
			return nil, failClosed("load canonical evidence projection", err)
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *PostgresStore) loadCanonicalEvidence(ctx context.Context, entry *EntryRequest) error {
	id, err := uuid.Parse(entry.CandidateID)
	if err != nil {
		return err
	}
	var decision string
	if err := s.pool.QueryRow(ctx, `SELECT decision FROM genuine_event_decisions WHERE candidate_id=$1 AND is_current ORDER BY decision_at DESC LIMIT 1`, id).Scan(&decision); err != nil {
		return err
	}
	if decision != string(DecisionCandidate) {
		return fmt.Errorf("canonical event decision is not CANDIDATE")
	}
	var assessment EvidenceAssessment
	var quality, overall float64
	var status string
	var ready, gate, broker, execution bool
	var contradictoryItems, staleItems int
	var scoredAt time.Time
	err = s.pool.QueryRow(ctx, `SELECT quality_score::float8,overall_evidence_score::float8,evidence_status,evidence_ready,evidence_gate_ready,broker_execution_allowed,execution_instruction_created,contradictory_item_count,stale_item_count,scored_at FROM candidate_evidence_scores WHERE candidate_id=$1 ORDER BY scored_at DESC LIMIT 1`, id).Scan(&assessment.QualityScore, &overall, &status, &ready, &gate, &broker, &execution, &contradictoryItems, &staleItems, &scoredAt)
	if err != nil {
		return err
	}
	quality = assessment.QualityScore
	if quality <= 0 || overall <= 0 {
		return fmt.Errorf("canonical evidence quality is not sufficient")
	}
	items, err := s.pool.Query(ctx, `SELECT evidence_id::text,source_ref,observed_at,quality_score::float8,freshness_status,supports_candidate,contradicts_candidate FROM candidate_evidence_items WHERE candidate_id=$1 ORDER BY observed_at,evidence_id`, id)
	if err != nil {
		return err
	}
	defer items.Close()
	var evidence []EvidenceReference
	independent := map[string]struct{}{}
	for items.Next() {
		var item EvidenceReference
		var sourceRef, freshness string
		var qualityScore float64
		var supports, contradicts bool
		if err := items.Scan(&item.EvidenceID, &sourceRef, &item.ObservedAt, &qualityScore, &freshness, &supports, &contradicts); err != nil {
			return err
		}
		item.ObservedAt = item.ObservedAt.UTC()
		if !validCanonicalEvidenceSourceRef(sourceRef) {
			return fmt.Errorf("canonical evidence source URL is unavailable")
		}
		item.SourceID, item.SourceURL = sourceRef, sourceRef
		item.Quality = fmt.Sprintf("%.3f/%s", qualityScore, freshness)
		evidence = append(evidence, item)
		independent[sourceRef] = struct{}{}
		if supports && contradicts {
			assessment.Contradictory = true
		}
	}
	if err := items.Err(); err != nil {
		return err
	}
	assessment.Provider = "candidate_evidence_scores"
	assessment.PolicyVersion = "candidate-evidence-scoring-v1"
	assessment.EvidenceSetFingerprint = EvidenceSetFingerprint(evidence)
	assessment.ReviewedAt = scoredAt.UTC()
	assessment.SourceBacked = len(evidence) > 0
	assessment.QualityState = status
	assessment.RequiredQualityScore = .70
	assessment.EvidenceReady, assessment.EvidenceGateReady = ready, gate
	assessment.Corroborated = len(independent) >= 2
	assessment.IndependentSourceGroups = len(independent)
	assessment.IssuerRelevant = strings.TrimSpace(entry.Candidate.IssuerID) != ""
	assessment.InstrumentRelevant = strings.TrimSpace(entry.Candidate.InstrumentID) != ""
	assessment.Contradictory = assessment.Contradictory || contradictoryItems > 0
	assessment.Stale = staleItems > 0
	assessment.Unknown = !assessment.SourceBacked || broker || execution
	if assessment.QualityScore < assessment.RequiredQualityScore || overall < assessment.RequiredQualityScore {
		assessment.QualityState = "insufficient"
	}
	entry.Candidate.Evidence = evidence
	entry.Candidate.ReviewedEvidence = &assessment
	entry.Candidate.CandidatePolicyVersion = assessment.PolicyVersion
	return nil
}

func validCanonicalEvidenceSourceRef(sourceRef string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sourceRef))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		return true
	}
	for _, prefix := range []string{"event_normalized:", "world_monitor_research_inbox:", "candles:"} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func (s *PostgresStore) MarkEntryProcessed(ctx context.Context, candidateID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE exploratory_paper_entry_queue SET status='PROCESSED',updated_at=$2 WHERE candidate_id=$1 AND status='PENDING'`, candidateID, s.now().UTC())
	return err
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

func (s *PostgresStore) FindByCandidate(ctx context.Context, candidateID string) (LifecycleRecord, bool, error) {
	var positionID string
	query := `SELECT l.position_id FROM exploratory_paper_lifecycles l WHERE l.candidate_id=$1 ORDER BY l.created_at LIMIT 1`
	args := []any{candidateID}
	if s.accountID != "" {
		query = `SELECT l.position_id FROM exploratory_paper_lifecycles l JOIN paper_ledger_events e ON e.fill_id=l.entry_fill_id WHERE l.candidate_id=$1 AND e.account_id=$2 ORDER BY l.created_at LIMIT 1`
		args = append(args, s.accountID)
	}
	err := s.pool.QueryRow(ctx, query, args...).Scan(&positionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleRecord{}, false, nil
	}
	if err != nil {
		return LifecycleRecord{}, false, err
	}
	record, err := s.Get(ctx, positionID)
	return record, true, err
}

func (s *PostgresStore) ListDueReviews(ctx context.Context, now time.Time) ([]ReviewRecord, error) {
	if now.IsZero() || now.Location() != time.UTC {
		return nil, fmt.Errorf("due-review time must be UTC")
	}
	query := `SELECT r.position_id,r.session_number FROM exploratory_paper_reviews r JOIN exploratory_paper_lifecycles l ON l.position_id=r.position_id WHERE l.state <> 'CLOSED' AND r.status IN ('PENDING','EXIT_RECOMMENDED','EXIT_APPROVED') AND r.scheduled_at <= $1 ORDER BY r.scheduled_at,r.position_id FOR UPDATE OF r SKIP LOCKED`
	args := []any{now}
	if s.accountID != "" {
		query = `SELECT r.position_id,r.session_number FROM exploratory_paper_reviews r JOIN exploratory_paper_lifecycles l ON l.position_id=r.position_id JOIN paper_ledger_events e ON e.fill_id=l.entry_fill_id WHERE l.state <> 'CLOSED' AND e.account_id=$2 AND r.status IN ('PENDING','EXIT_RECOMMENDED','EXIT_APPROVED') AND r.scheduled_at <= $1 ORDER BY r.scheduled_at,r.position_id FOR UPDATE OF r SKIP LOCKED`
		args = append(args, s.accountID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ReviewRecord
	for rows.Next() {
		var positionID string
		var session int
		if err := rows.Scan(&positionID, &session); err != nil {
			return nil, err
		}
		record, err := s.Get(ctx, positionID)
		if err != nil {
			return nil, failClosed("restore due lifecycle", err)
		}
		var review Review
		for _, candidate := range record.Reviews {
			if candidate.SessionNumber == session {
				review = candidate
				break
			}
		}
		if review.ReviewID == "" {
			return nil, failClosed("restore due review", ErrPersistenceConflict)
		}
		result = append(result, ReviewRecord{Lifecycle: record, Review: review})
	}
	return result, rows.Err()
}

func (s *PostgresStore) PersistExitRecommendation(ctx context.Context, positionID string, decision ExitDecision, review Review) error {
	if positionID == "" || review.PositionID != positionID || decision.Reason == "" {
		return fmt.Errorf("exit recommendation identity is invalid")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load exit recommendation lifecycle", err)
	}
	record.Position.OperationalState = StateExitRecommended
	review.Status = "EXIT_RECOMMENDED"
	review.ExitAction = string(decision.Action)
	review.ExitReason = string(decision.Reason)
	review.ExitRecommendationID = exitRecommendationIdentity(review, decision)
	now := s.now().UTC()
	review.ProcessedAt = &now
	payload, err := json.Marshal(review)
	if err != nil {
		return err
	}
	positionPayload, err := json.Marshal(record.Position)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return failClosed("begin exit recommendation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE exploratory_paper_lifecycles SET position_payload=$2,operational_state=$3,next_review_at=NULL,updated_at=$4 WHERE position_id=$1`, positionID, positionPayload, string(StateExitRecommended), now)
	if err != nil {
		return failClosed("persist exit recommendation position", err)
	}
	_, err = tx.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='EXIT_RECOMMENDED',payload=$3,processed_at=$2 WHERE review_id=$1 AND position_id=$4 AND status IN ('PENDING','COMPLETED','EXIT_RECOMMENDED')`, review.ReviewID, now, payload, positionID)
	if err != nil {
		return failClosed("persist exit recommendation review", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return failClosed("commit exit recommendation", err)
	}
	return nil
}

func exitRecommendationIdentity(review Review, decision ExitDecision) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", review.PositionID, review.ReviewID, decision.Action, decision.Reason)))
	return "exitrec_" + hex.EncodeToString(digest[:])
}

// ApproveExit is the production exit-approval source. It reads the durable
// operator decision written through the protected exploratory-paper API and
// never infers approval from an exit recommendation alone.
func (s *PostgresStore) ApproveExit(ctx context.Context, lifecycle LifecycleRecord, decision ExitDecision, _ ReviewObservation) (ApprovalSnapshot, error) {
	if s == nil || s.pool == nil {
		return ApprovalSnapshot{}, fmt.Errorf("%w: database is unavailable", ErrFailedClosed)
	}
	for _, candidate := range lifecycle.Reviews {
		if candidate.PositionID != lifecycle.Position.PositionID {
			continue
		}
		var storedStatus string
		var payload []byte
		err := s.pool.QueryRow(ctx, `SELECT status,payload FROM exploratory_paper_reviews WHERE review_id=$1 AND position_id=$2`, candidate.ReviewID, lifecycle.Position.PositionID).Scan(&storedStatus, &payload)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return ApprovalSnapshot{}, failClosed("load durable exit approval", err)
		}
		var review Review
		if err := json.Unmarshal(payload, &review); err != nil {
			return ApprovalSnapshot{}, failClosed("decode durable exit approval review", err)
		}
		review.Status = storedStatus
		if review.ExitRecommendationID != exitRecommendationIdentity(candidate, decision) {
			continue
		}
		switch storedStatus {
		case "EXIT_APPROVED", "EXIT_REJECTED":
			if review.ExitApproval == nil {
				return ApprovalSnapshot{}, failClosed("durable exit decision is missing its workflow snapshot", ErrPersistenceConflict)
			}
			if err := validateExitApproval(lifecycle, review, *review.ExitApproval, s.now().UTC()); err != nil {
				return ApprovalSnapshot{}, failClosed("validate durable exit decision", err)
			}
			return *review.ExitApproval, nil
		case "EXIT_RECOMMENDED":
			return ApprovalSnapshot{}, ErrHumanApprovalPending
		}
	}
	return ApprovalSnapshot{}, ErrHumanApprovalPending
}

func (s *PostgresStore) PersistExitApproval(ctx context.Context, positionID string, review Review, approval ApprovalSnapshot) error {
	if positionID == "" || review.PositionID != positionID || approval.Workflow.State != workflow.StatePaperIntentCreated {
		return fmt.Errorf("exit approval identity is invalid")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load exit approval lifecycle", err)
	}
	canonical, err := canonicalExitReview(record, review.ReviewID)
	if err != nil {
		return err
	}
	if err := validateExitApproval(record, canonical, approval, s.now().UTC()); err != nil {
		return failClosed("validate exit approval", err)
	}
	review = canonical
	review.Status = "EXIT_APPROVED"
	review.ExitApprovalWorkflowID = approval.Workflow.WorkflowID
	review.ExitApproval = &approval
	now := s.now().UTC()
	review.ProcessedAt = &now
	payload, err := json.Marshal(review)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='EXIT_APPROVED',payload=$3,processed_at=$2 WHERE review_id=$1 AND position_id=$4 AND status='EXIT_RECOMMENDED'`, review.ReviewID, now, payload, positionID)
	if err != nil {
		return failClosed("persist exit approval", err)
	}
	if result.RowsAffected() == 0 {
		return verifyPersistedExitDecision(ctx, positionID, review.ReviewID, "EXIT_APPROVED", approval, s.pool)
	}
	return nil
}

func (s *PostgresStore) PersistExitRejection(ctx context.Context, positionID string, review Review, rejection ApprovalSnapshot) error {
	if positionID == "" || review.PositionID != positionID || rejection.Workflow.State != workflow.StateHumanRejected {
		return fmt.Errorf("exit rejection identity is invalid")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load exit rejection lifecycle", err)
	}
	canonical, err := canonicalExitReview(record, review.ReviewID)
	if err != nil {
		return err
	}
	if err := validateExitApproval(record, canonical, rejection, s.now().UTC()); err != nil {
		return failClosed("validate exit rejection", err)
	}
	review = canonical
	review.Status = "EXIT_REJECTED"
	review.ExitApprovalWorkflowID = rejection.Workflow.WorkflowID
	review.ExitApproval = &rejection
	now := s.now().UTC()
	review.ProcessedAt = &now
	payload, err := json.Marshal(review)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='EXIT_REJECTED',payload=$3,processed_at=$2 WHERE review_id=$1 AND position_id=$4 AND status='EXIT_RECOMMENDED'`, review.ReviewID, now, payload, positionID)
	if err != nil {
		return failClosed("persist exit rejection", err)
	}
	if result.RowsAffected() == 0 {
		return verifyPersistedExitDecision(ctx, positionID, review.ReviewID, "EXIT_REJECTED", rejection, s.pool)
	}
	return nil
}

func verifyPersistedExitDecision(ctx context.Context, positionID, reviewID, status string, approval ApprovalSnapshot, pool *pgxpool.Pool) error {
	var storedStatus string
	var payload []byte
	if err := pool.QueryRow(ctx, `SELECT status,payload FROM exploratory_paper_reviews WHERE review_id=$1 AND position_id=$2`, reviewID, positionID).Scan(&storedStatus, &payload); err != nil {
		return failClosed("verify persisted exit decision", err)
	}
	if storedStatus != status {
		return failClosed("exit decision conflict", ErrPersistenceConflict)
	}
	var stored Review
	if err := json.Unmarshal(payload, &stored); err != nil || stored.ExitApproval == nil {
		if err != nil {
			return failClosed("decode persisted exit decision", err)
		}
		return failClosed("persisted exit decision is incomplete", ErrPersistenceConflict)
	}
	left, err := json.Marshal(*stored.ExitApproval)
	if err != nil {
		return failClosed("encode persisted exit decision", err)
	}
	right, err := json.Marshal(approval)
	if err != nil {
		return failClosed("encode exit decision", err)
	}
	if string(left) != string(right) {
		return failClosed("exit decision conflict", ErrPersistenceConflict)
	}
	return nil
}

// PersistExitDecision is the protected operator/API handoff. The request must
// carry the existing workflow snapshot plus the explicit exit binding; the
// server chooses the durable review from that binding and never trusts a
// caller-supplied review payload.
func (s *PostgresStore) PersistExitDecision(ctx context.Context, positionID string, approval ApprovalSnapshot) error {
	if approval.ExitBinding == nil || approval.ExitBinding.PositionID != positionID {
		return fmt.Errorf("exit decision binding is invalid")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load exit decision lifecycle", err)
	}
	review, err := canonicalExitReview(record, approval.ExitBinding.ReviewID)
	if err != nil {
		return err
	}
	switch approval.Workflow.State {
	case workflow.StatePaperIntentCreated:
		return s.PersistExitApproval(ctx, positionID, review, approval)
	case workflow.StateHumanRejected:
		return s.PersistExitRejection(ctx, positionID, review, approval)
	default:
		return fmt.Errorf("exit decision must be an explicit paper approval or rejection")
	}
}

func canonicalExitReview(record LifecycleRecord, reviewID string) (Review, error) {
	for _, review := range record.Reviews {
		if review.ReviewID == reviewID {
			if review.PositionID != record.Position.PositionID || review.ExitRecommendationID == "" {
				return Review{}, fmt.Errorf("review is not a durable exit recommendation")
			}
			return review, nil
		}
	}
	return Review{}, fmt.Errorf("exit review %q was not found", reviewID)
}

func validateExitApproval(record LifecycleRecord, review Review, approval ApprovalSnapshot, now time.Time) error {
	if approval.ExitBinding == nil || approval.ExitBinding.PositionID != record.Position.PositionID || approval.ExitBinding.ReviewID != review.ReviewID || approval.ExitBinding.RecommendationID != review.ExitRecommendationID || approval.ExitBinding.EntryWorkflowID != record.Binding.WorkflowID || approval.ExitBinding.EntryPaperIntentID != record.Binding.PaperIntentID {
		return fmt.Errorf("exit approval is not bound to the requested position, review, recommendation, and entry paper identity")
	}
	if err := approval.Workflow.Validate(); err != nil {
		return err
	}
	if err := workflow.ValidateAuditEvents(approval.Events); err != nil || len(approval.Events) == 0 || approval.Events[len(approval.Events)-1].WorkflowID != approval.Workflow.WorkflowID || approval.Events[len(approval.Events)-1].NextState != approval.Workflow.State || approval.Events[len(approval.Events)-1].Sequence != approval.Workflow.Revision {
		if err != nil {
			return err
		}
		return fmt.Errorf("exit approval workflow audit does not match its durable state")
	}
	if approval.Workflow.RecommendationID != review.ExitRecommendationID || approval.Workflow.Confirmation == nil {
		return fmt.Errorf("exit approval workflow is not bound to the recommendation")
	}
	confirmation := approval.Workflow.Confirmation
	if err := confirmation.ValidateFor(approval.Workflow, now); err != nil {
		return err
	}
	wantDirection := DirectionLong
	if record.Thesis.Thesis.Direction == DirectionLong {
		wantDirection = DirectionShort
	}
	if confirmation.InstrumentID != record.Thesis.Thesis.InstrumentID || confirmation.Direction != string(wantDirection) {
		return fmt.Errorf("exit approval confirmation is not bound to the open position")
	}
	switch approval.Workflow.State {
	case workflow.StatePaperIntentCreated:
		if confirmation.Decision != workflow.ConfirmationApprove || approval.PaperIntent.WorkflowID != approval.Workflow.WorkflowID || approval.PaperIntent.RecommendationID != review.ExitRecommendationID || approval.PaperIntent.InstrumentID != record.Thesis.Thesis.InstrumentID || approval.PaperIntent.Direction != string(wantDirection) {
			return fmt.Errorf("exit paper intent is not bound to the approved recommendation")
		}
		return approval.PaperIntent.Validate()
	case workflow.StateHumanRejected:
		if confirmation.Decision != workflow.ConfirmationReject || approval.PaperIntent.IntentID != "" {
			return fmt.Errorf("exit rejection contains an invalid paper intent")
		}
		return nil
	default:
		return fmt.Errorf("unsupported exit approval state %q", approval.Workflow.State)
	}
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
		return fmt.Errorf("outcome: %w", err)
	}
	if positionID != outcome.PositionID || exit.Fill.FillID != outcome.ExitFillID || exit.Order.OrderID != outcome.ExitOrderID {
		return fmt.Errorf("outcome exit identity is inconsistent")
	}
	record, err := s.Get(ctx, positionID)
	if err != nil {
		return failClosed("load position for outcome close", err)
	}
	if err := s.requireAccount(exit.Ledger.AccountID); err != nil {
		return failClosed("exit account binding", err)
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
		return fmt.Errorf("exit order: %w", err)
	}
	if err := exit.Fill.Validate(); err != nil {
		return fmt.Errorf("exit fill: %w", err)
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
	if _, err := tx.Exec(ctx, `UPDATE exploratory_paper_reviews SET status='COMPLETED',processed_at=$2 WHERE position_id=$1 AND status='EXIT_APPROVED'`, positionID, s.now().UTC()); err != nil {
		return failClosed("close exit review", err)
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
	if err := s.requireFillAccount(ctx, entryFillID); err != nil {
		return record, failClosed("verify lifecycle paper account", err)
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
		var storedStatus string
		if err := rows.Scan(&review.ReviewID, &review.PositionID, &review.SessionNumber, &review.ScheduledAt, &storedStatus, &payload, &review.ProcessedAt); err != nil {
			return record, err
		}
		if err := json.Unmarshal(payload, &review); err != nil {
			return record, failClosed("decode review", err)
		}
		review.Status = storedStatus
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
	// INVALIDATED and EXIT_RECOMMENDED lifecycles remain operationally active
	// until the separately approved simulated exit is persisted. Hiding them
	// would make a pending human decision invisible to the operator.
	query := `SELECT l.position_id FROM exploratory_paper_lifecycles l WHERE l.state <> 'CLOSED' ORDER BY l.updated_at DESC LIMIT $1`
	args := []any{limit}
	if s.accountID != "" {
		query = `SELECT l.position_id FROM exploratory_paper_lifecycles l JOIN paper_ledger_events e ON e.fill_id=l.entry_fill_id WHERE l.state <> 'CLOSED' AND e.account_id=$2 ORDER BY l.updated_at DESC LIMIT $1`
		args = append(args, s.accountID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
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
