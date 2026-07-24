package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/libs/runtimepolicy"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type operatorEvidenceOverview struct {
	RuntimeMode            string    `json:"runtimeMode"`
	AllowLiveTrading       bool      `json:"allowLiveTrading"`
	ExecutionEnabled       bool      `json:"executionEnabled"`
	ExecutionWorkerEnabled bool      `json:"executionWorkerEnabled"`
	BrokerExecutionAllowed bool      `json:"brokerExecutionAllowed"`
	MaximumLeverage        float64   `json:"maximumLeverage"`
	GenuineEvents          int       `json:"genuineEvents"`
	SyntheticEvents        int       `json:"syntheticEvents"`
	RejectedEvents         int       `json:"rejectedEvents"`
	DeduplicatedEvents     int       `json:"deduplicatedEvents"`
	Candidates             int       `json:"candidates"`
	Approvals              int       `json:"approvals"`
	PaperTickets           int       `json:"paperTickets"`
	PendingCheckpoints     int       `json:"pendingCheckpoints"`
	CompletedCheckpoints   int       `json:"completedCheckpoints"`
	MissingDataCheckpoints int       `json:"missingDataCheckpoints"`
	AmbiguousCheckpoints   int       `json:"ambiguousCheckpoints"`
	CheckedAt              time.Time `json:"checkedAt"`
}

func operatorEvidenceOverviewHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mode := runtimepolicy.CurrentMode()
		out := operatorEvidenceOverview{
			RuntimeMode:            mode.String(),
			AllowLiveTrading:       strings.EqualFold(os.Getenv("ALLOW_LIVE_TRADING"), "true"),
			ExecutionEnabled:       strings.EqualFold(os.Getenv("EXECUTION_ENABLED"), "true"),
			ExecutionWorkerEnabled: strings.EqualFold(os.Getenv("EXECUTION_INSTRUCTION_WORKER_ENABLED"), "true"),
			BrokerExecutionAllowed: strings.EqualFold(os.Getenv("BROKER_EXECUTION_ALLOWED"), "true"),
			MaximumLeverage:        envFloat("MAX_LEVERAGE", 1), CheckedAt: time.Now().UTC(),
		}
		if pool == nil {
			jsonOK(w, out)
			return
		}
		if err := pool.QueryRow(r.Context(), `SELECT
			COUNT(*) FILTER (WHERE NOT COALESCE(er.is_synthetic, false))::int,
			COUNT(*) FILTER (WHERE COALESCE(er.is_synthetic, false))::int,
			COUNT(*) FILTER (WHERE w.status='rejected')::int,
			COUNT(*) FILTER (WHERE w.status='ignored' AND COALESCE(w.rejection_reason,'') ILIKE '%dedup%')::int,
			COUNT(*) FILTER (WHERE w.candidate_id IS NOT NULL)::int
			FROM world_monitor_research_inbox w
			LEFT JOIN event_normalized en ON en.id=w.normalized_event_id
			LEFT JOIN event_raw er ON er.id=en.raw_event_id`).Scan(&out.GenuineEvents, &out.SyntheticEvents, &out.RejectedEvents, &out.DeduplicatedEvents, &out.Candidates); err != nil {
			http.Error(w, fmt.Sprintf("operator event counts: %v", err), http.StatusInternalServerError)
			return
		}
		if err := pool.QueryRow(r.Context(), `SELECT
			(SELECT COUNT(*)::int FROM candidate_approvals),
			(SELECT COUNT(*)::int FROM candidate_paper_tickets),
			COUNT(*) FILTER (WHERE checkpoint_status IN ('pending_not_due','pending_market_data'))::int,
			COUNT(*) FILTER (WHERE checkpoint_status IN ('completed','stop_touched','target_touched'))::int,
			COUNT(*) FILTER (WHERE checkpoint_status IN ('pending_market_data','insufficient_data'))::int,
			COUNT(*) FILTER (WHERE checkpoint_status='ambiguous_same_candle')::int
			FROM paper_ticket_outcome_checkpoints`).Scan(&out.Approvals, &out.PaperTickets, &out.PendingCheckpoints, &out.CompletedCheckpoints, &out.MissingDataCheckpoints, &out.AmbiguousCheckpoints); err != nil {
			http.Error(w, fmt.Sprintf("operator activity counts: %v", err), http.StatusInternalServerError)
			return
		}
		jsonOK(w, out)
	}
}

func envFloat(key string, fallback float64) float64 {
	var value float64
	if _, err := fmt.Sscan(strings.TrimSpace(os.Getenv(key)), &value); err == nil && value > 0 {
		return value
	}
	return fallback
}

type operatorCheckpoint struct {
	Name                string     `json:"name"`
	TrackingStartedAt   time.Time  `json:"trackingStartedAt"`
	TrackingStartSource string     `json:"trackingStartSource"`
	DueAt               time.Time  `json:"dueAt"`
	ObservationAt       *time.Time `json:"observationAt,omitempty"`
	EntryPrice          float64    `json:"entryPrice"`
	CheckpointPrice     *float64   `json:"checkpointPrice,omitempty"`
	PercentageReturn    *float64   `json:"percentageReturn,omitempty"`
	HypotheticalPnL     *float64   `json:"hypotheticalPnl,omitempty"`
	MFE                 *float64   `json:"maximumFavourableExcursion,omitempty"`
	MAE                 *float64   `json:"maximumAdverseExcursion,omitempty"`
	TargetTouched       bool       `json:"targetTouched"`
	StopTouched         bool       `json:"stopTouched"`
	FirstTargetTouchAt  *time.Time `json:"firstTargetTouchAt,omitempty"`
	FirstStopTouchAt    *time.Time `json:"firstStopTouchAt,omitempty"`
	Status              string     `json:"status"`
	DataQualityStatus   string     `json:"dataQualityStatus"`
	MarketDataSource    *string    `json:"marketDataSource,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type operatorCandidateSummary struct {
	CandidateID          string     `json:"candidateId"`
	Symbol               string     `json:"symbol"`
	SetupType            string     `json:"setupType"`
	CandidateStatus      string     `json:"candidateStatus"`
	HumanDecision        string     `json:"humanDecision"`
	DecisionProvenance   string     `json:"decisionProvenance"`
	PaperTicketID        string     `json:"paperTicketId,omitempty"`
	PaperTicketStatus    string     `json:"paperTicketStatus,omitempty"`
	LatestOutcomeStatus  string     `json:"latestOutcomeStatus,omitempty"`
	CompletedCheckpoints int        `json:"completedCheckpoints"`
	PendingCheckpoints   int        `json:"pendingCheckpoints"`
	MissingCheckpoints   int        `json:"missingCheckpoints"`
	AmbiguousCheckpoints int        `json:"ambiguousCheckpoints"`
	Reason               string     `json:"reason"`
	BlockReason          string     `json:"blockReason,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	ExpiresAt            *time.Time `json:"expiresAt,omitempty"`
}

func operatorCandidatesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if pool == nil {
			jsonOK(w, []operatorCandidateSummary{})
			return
		}
		rows, err := pool.Query(r.Context(), `SELECT
			ct.id::text,
			ct.symbol,
			COALESCE(NULLIF(ct.setup_type,''), NULLIF(ct.strategy_id,''), 'Not supplied'),
			ct.status,
			COALESCE(ca.decision, ''),
			CASE
				WHEN ca.id IS NULL THEN 'none'
				WHEN lower(COALESCE(ca.approved_by,'')) IN ('system','automation','auto','jax') THEN 'non_human'
				ELSE 'human'
			END,
			COALESCE(pt.paper_ticket_id,''),
			COALESCE(pt.status,''),
			COALESCE(latest.checkpoint_status,''),
			COALESCE(outcomes.completed,0)::int,
			COALESCE(outcomes.pending,0)::int,
			COALESCE(outcomes.missing,0)::int,
			COALESCE(outcomes.ambiguous,0)::int,
			COALESCE(NULLIF(ct.candidate_reason_summary,''), NULLIF(ct.reasoning,''), NULLIF(ct.catalyst_summary,''), 'No plain-language reason was persisted.'),
			COALESCE(ct.block_reason,''),
			ct.detected_at,
			ct.expires_at
			FROM candidate_trades ct
			LEFT JOIN LATERAL (
				SELECT id, decision, approved_by
				FROM candidate_approvals
				WHERE candidate_id=ct.id
				ORDER BY decided_at DESC
				LIMIT 1
			) ca ON true
			LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id=ct.id
			LEFT JOIN LATERAL (
				SELECT
					COUNT(*) FILTER (WHERE checkpoint_status IN ('completed','stop_touched','target_touched','ambiguous_same_candle')) AS completed,
					COUNT(*) FILTER (WHERE checkpoint_status='pending_not_due') AS pending,
					COUNT(*) FILTER (WHERE checkpoint_status IN ('pending_market_data','insufficient_data')) AS missing,
					COUNT(*) FILTER (WHERE checkpoint_status='ambiguous_same_candle') AS ambiguous
				FROM paper_ticket_outcome_checkpoints
				WHERE paper_ticket_id=pt.paper_ticket_id
			) outcomes ON true
			LEFT JOIN LATERAL (
				SELECT checkpoint_status
				FROM paper_ticket_outcome_checkpoints
				WHERE paper_ticket_id=pt.paper_ticket_id
				ORDER BY scheduled_at DESC
				LIMIT 1
			) latest ON true
			ORDER BY ct.detected_at DESC
			LIMIT 100`)
		if err != nil {
			http.Error(w, fmt.Sprintf("operator candidates: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := make([]operatorCandidateSummary, 0)
		for rows.Next() {
			var candidate operatorCandidateSummary
			if err := rows.Scan(
				&candidate.CandidateID,
				&candidate.Symbol,
				&candidate.SetupType,
				&candidate.CandidateStatus,
				&candidate.HumanDecision,
				&candidate.DecisionProvenance,
				&candidate.PaperTicketID,
				&candidate.PaperTicketStatus,
				&candidate.LatestOutcomeStatus,
				&candidate.CompletedCheckpoints,
				&candidate.PendingCheckpoints,
				&candidate.MissingCheckpoints,
				&candidate.AmbiguousCheckpoints,
				&candidate.Reason,
				&candidate.BlockReason,
				&candidate.CreatedAt,
				&candidate.ExpiresAt,
			); err != nil {
				http.Error(w, fmt.Sprintf("operator candidate row: %v", err), http.StatusInternalServerError)
				return
			}
			out = append(out, candidate)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("operator candidates rows: %v", err), http.StatusInternalServerError)
			return
		}
		jsonOK(w, out)
	}
}

type operatorCandidateEvidence struct {
	EvidenceScore             *float64             `json:"evidenceScore,omitempty"`
	EvidenceStatus            string               `json:"evidenceStatus"`
	GateStatus                string               `json:"gateStatus"`
	RiskStatus                string               `json:"riskStatus"`
	ApprovalID                string               `json:"approvalId,omitempty"`
	ApprovalDecision          string               `json:"approvalDecision,omitempty"`
	DecisionProvenance        string               `json:"decisionProvenance"`
	ApprovedBy                string               `json:"approvedBy,omitempty"`
	ApprovalReason            string               `json:"approvalReason,omitempty"`
	ApprovalAt                *time.Time           `json:"approvalAt,omitempty"`
	PaperTicketID             string               `json:"paperTicketId,omitempty"`
	PaperTicketStatus         string               `json:"paperTicketStatus,omitempty"`
	Entry                     *float64             `json:"entry,omitempty"`
	Stop                      *float64             `json:"stop,omitempty"`
	Target                    *float64             `json:"target,omitempty"`
	Quantity                  *float64             `json:"quantity,omitempty"`
	PlannedRisk               *float64             `json:"plannedRisk,omitempty"`
	PlannedReward             *float64             `json:"plannedReward,omitempty"`
	RewardRisk                *float64             `json:"rewardRisk,omitempty"`
	Notional                  *float64             `json:"notional,omitempty"`
	AccountEquityAssumption   *float64             `json:"accountEquityAssumption,omitempty"`
	Leverage                  *float64             `json:"leverage,omitempty"`
	Checkpoints               []operatorCheckpoint `json:"checkpoints"`
	SelectedExecutionCounts   map[string]int       `json:"selectedExecutionCounts"`
	HistoricalExecutionCounts map[string]int       `json:"historicalExecutionCounts"`
}

func operatorCandidateEvidenceHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw := strings.TrimPrefix(r.URL.Path, "/api/v1/operator-evidence/candidates/")
		id, err := uuid.Parse(strings.Trim(raw, "/"))
		if err != nil {
			http.Error(w, "invalid candidate id", http.StatusBadRequest)
			return
		}
		out := operatorCandidateEvidence{Checkpoints: []operatorCheckpoint{}, SelectedExecutionCounts: map[string]int{}, HistoricalExecutionCounts: map[string]int{}}
		var score sql.NullFloat64
		err = pool.QueryRow(r.Context(), `SELECT ces.overall_evidence_score::float8, COALESCE(ces.evidence_status,'missing'), ct.gate_status, ct.risk_status,
			COALESCE(ca.id::text,''), COALESCE(ca.decision,''),
			CASE WHEN ca.id IS NULL THEN 'none' WHEN lower(COALESCE(ca.approved_by,'')) IN ('system','automation','auto','jax') THEN 'non_human' ELSE 'human' END,
			COALESCE(ca.approved_by,''), COALESCE(ca.notes,''), ca.decided_at,
			COALESCE(pt.paper_ticket_id,''), COALESCE(pt.status,''), pt.entry_price::float8, pt.stop_loss_price::float8, pt.target_price::float8, pt.position_size::float8,
			pt.max_normal_loss::float8, (pt.target_price-pt.entry_price)*pt.position_size, pt.reward_risk_ratio::float8,
			(pt.entry_price*pt.position_size)::float8
			FROM candidate_trades ct
			LEFT JOIN LATERAL (SELECT * FROM candidate_evidence_scores WHERE candidate_id=ct.id ORDER BY scored_at DESC LIMIT 1) ces ON true
			LEFT JOIN LATERAL (SELECT * FROM candidate_approvals WHERE candidate_id=ct.id ORDER BY decided_at DESC LIMIT 1) ca ON true
			LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id=ct.id WHERE ct.id=$1`, id).Scan(&score, &out.EvidenceStatus, &out.GateStatus, &out.RiskStatus, &out.ApprovalID, &out.ApprovalDecision, &out.DecisionProvenance, &out.ApprovedBy, &out.ApprovalReason, &out.ApprovalAt, &out.PaperTicketID, &out.PaperTicketStatus, &out.Entry, &out.Stop, &out.Target, &out.Quantity, &out.PlannedRisk, &out.PlannedReward, &out.RewardRisk, &out.Notional)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("candidate evidence: %v", err), 500)
			}
			return
		}
		if score.Valid {
			out.EvidenceScore = &score.Float64
		}
		if out.PaperTicketID != "" {
			rows, qerr := pool.Query(r.Context(), `SELECT checkpoint_name,tracking_started_at,tracking_start_source,scheduled_at,observation_at,hypothetical_entry_price::float8,checkpoint_price::float8,percentage_return::float8,hypothetical_pnl::float8,maximum_favourable_excursion::float8,maximum_adverse_excursion::float8,target_touched,stop_touched,first_target_touch_at,first_stop_touch_at,checkpoint_status,data_quality_status,market_data_source,created_at,updated_at FROM paper_ticket_outcome_checkpoints WHERE paper_ticket_id=$1 ORDER BY scheduled_at`, out.PaperTicketID)
			if qerr != nil {
				http.Error(w, qerr.Error(), 500)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var c operatorCheckpoint
				if err := rows.Scan(&c.Name, &c.TrackingStartedAt, &c.TrackingStartSource, &c.DueAt, &c.ObservationAt, &c.EntryPrice, &c.CheckpointPrice, &c.PercentageReturn, &c.HypotheticalPnL, &c.MFE, &c.MAE, &c.TargetTouched, &c.StopTouched, &c.FirstTargetTouchAt, &c.FirstStopTouchAt, &c.Status, &c.DataQualityStatus, &c.MarketDataSource, &c.CreatedAt, &c.UpdatedAt); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				out.Checkpoints = append(out.Checkpoints, c)
			}
		}
		var selectedInstructions, selectedIntents, selectedBrokerOrders, selectedTrades, selectedFills int
		if err := pool.QueryRow(r.Context(), `SELECT COUNT(*)::int, COUNT(DISTINCT broker_order_id) FILTER (WHERE broker_order_id IS NOT NULL AND broker_order_id<>'')::int, COUNT(DISTINCT trade_id) FILTER (WHERE trade_id IS NOT NULL)::int FROM execution_instructions WHERE candidate_id=$1`, id).Scan(&selectedInstructions, &selectedBrokerOrders, &selectedTrades); err != nil {
			http.Error(w, fmt.Sprintf("selected execution evidence: %v", err), http.StatusInternalServerError)
			return
		}
		if err := pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM order_intents WHERE metadata->>'candidateId'=$1 OR metadata->>'candidate_id'=$1`, id.String()).Scan(&selectedIntents); err != nil {
			http.Error(w, fmt.Sprintf("selected order-intent evidence: %v", err), http.StatusInternalServerError)
			return
		}
		if err := pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM fills f WHERE EXISTS (SELECT 1 FROM execution_instructions ei WHERE ei.candidate_id=$1 AND (ei.trade_id=f.trade_id OR ei.broker_order_id=f.broker_order_id))`, id).Scan(&selectedFills); err != nil {
			http.Error(w, fmt.Sprintf("selected fill evidence: %v", err), http.StatusInternalServerError)
			return
		}
		out.SelectedExecutionCounts = map[string]int{"executionInstructions": selectedInstructions, "orderIntents": selectedIntents, "brokerOrders": selectedBrokerOrders, "trades": selectedTrades, "fills": selectedFills}
		var historicalInstructions, historicalIntents, historicalBrokerOrders, historicalTrades, historicalFills int
		if err := pool.QueryRow(r.Context(), `SELECT (SELECT COUNT(*)::int FROM execution_instructions), (SELECT COUNT(*)::int FROM order_intents), (SELECT COUNT(DISTINCT broker_order_id)::int FROM execution_instructions WHERE broker_order_id IS NOT NULL AND broker_order_id<>''), (SELECT COUNT(*)::int FROM trades), (SELECT COUNT(*)::int FROM fills)`).Scan(&historicalInstructions, &historicalIntents, &historicalBrokerOrders, &historicalTrades, &historicalFills); err != nil {
			http.Error(w, fmt.Sprintf("historical execution evidence: %v", err), http.StatusInternalServerError)
			return
		}
		out.HistoricalExecutionCounts = map[string]int{"executionInstructions": historicalInstructions, "orderIntents": historicalIntents, "brokerOrders": historicalBrokerOrders, "trades": historicalTrades, "fills": historicalFills}
		jsonOK(w, out)
	}
}
