package eventdecisions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/candidates"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) LoadSelectedEvents(ctx context.Context, eventIdentity string, limit int) ([]Event, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("event decision store requires a database")
	}
	if limit <= 0 || limit > 250 {
		return nil, fmt.Errorf("bounded replay limit must be between 1 and 250")
	}
	args := []any{}
	where := "WHERE 1=1"
	if strings.TrimSpace(eventIdentity) != "" {
		args = append(args, strings.TrimSpace(eventIdentity))
		where += fmt.Sprintf(" AND (w.id::text=$%d OR w.source_event_id=$%d)", len(args), len(args))
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT
			w.id, w.source, w.source_event_id, w.status, w.event_type, w.headline,
			COALESCE(w.summary,''), COALESCE(w.source_urls,'[]'::jsonb), w.source_count,
			w.event_time, w.received_at, w.severity, w.source_tier, w.confidence,
			COALESCE(w.confidence_reasons,'[]'::jsonb), COALESCE(w.possible_affected_etfs,'[]'::jsonb),
			COALESCE(w.mapping_reason,''), COALESCE(w.raw_payload,'{}'::jsonb),
			en.id, er.id, er.id IS NOT NULL, COALESCE(er.is_synthetic,false),
			COALESCE(er.synthetic_reason,''), COALESCE(er.data_source_type,''), COALESCE(er.source_provider,''),
			NULLIF(COALESCE(er.payload->>'collection_timestamp_utc',er.payload->>'collected_at',er.payload->>'collectedAt',''),'')::timestamptz,
			COALESCE(er.payload->>'deterministic_analysis',er.payload->>'deterministic_analysis_identity',er.payload->>'analysis_identity',''),
			COALESCE(er.payload->>'analysis_provider',er.payload->>'ai_provider',''),
			COALESCE(mapping.methods,'[]'::jsonb),
			CASE WHEN ct.id IS NULL THEN NULL ELSE jsonb_build_object(
				'id',ct.id,'strategyInstanceId',ct.strategy_instance_id,'symbol',ct.symbol,'signalType',ct.signal_type,
				'status',ct.status,'source',ct.source,'instrumentType',ct.instrument_type,'setupType',ct.setup_type,
				'direction',ct.direction,'catalystSummary',ct.catalyst_summary,'entryPrice',ct.entry_price,
				'stopLoss',ct.stop_loss,'takeProfit',ct.take_profit,'invalidationReason',ct.invalidation_reason,
				'riskStatus',ct.risk_status,'gateStatus',ct.gate_status,'approvalStatus',ct.approval_status,
				'humanApprovalRequired',ct.human_approval_required,'hasContradictoryEvidence',ct.has_contradictory_evidence,
				'rejectReasons',ct.reject_reasons,'metadata',COALESCE(ct.metadata,'{}'::jsonb),
				'dataProvenance',ct.data_provenance,'sessionDate',ct.session_date,'createdAt',ct.created_at,'updatedAt',ct.updated_at,
				'executionInstructionId',execution_link.id,'tradeId',execution_link.trade_id
			) END,
			CASE WHEN ces.id IS NULL THEN NULL ELSE jsonb_build_object(
				'candidateId',ces.candidate_id,'supportScore',ces.support_score,'contradictionScore',ces.contradiction_score,
				'qualityScore',ces.quality_score,'freshnessScore',ces.freshness_score,'overallEvidenceScore',ces.overall_evidence_score,
				'evidenceItemCount',ces.evidence_item_count,'supportingItemCount',ces.supporting_item_count,
				'contradictoryItemCount',ces.contradictory_item_count,'staleItemCount',ces.stale_item_count,
				'evidenceStatus',ces.evidence_status,'evidenceReady',ces.evidence_ready,'evidenceGateReady',ces.evidence_gate_ready,
				'approvalGranted',ces.approval_granted,'brokerExecutionAllowed',ces.broker_execution_allowed,
				'executionInstructionCreated',ces.execution_instruction_created
			) END
		FROM world_monitor_research_inbox w
		LEFT JOIN event_normalized en ON en.id=w.normalized_event_id
		LEFT JOIN LATERAL (
			SELECT candidate.* FROM event_raw candidate
			WHERE candidate.id=en.raw_event_id OR (candidate.source_event_id=w.source_event_id AND candidate.source_id=w.source)
			ORDER BY (candidate.id=en.raw_event_id) DESC,candidate.received_at DESC LIMIT 1
		) er ON true
		LEFT JOIN LATERAL (
			SELECT jsonb_agg(DISTINCT esm.mapping_method ORDER BY esm.mapping_method) AS methods
			FROM event_symbol_map esm WHERE esm.normalized_event_id=en.id
		) mapping ON true
		LEFT JOIN candidate_trades ct ON ct.id=w.candidate_id
		LEFT JOIN LATERAL (SELECT id,trade_id FROM execution_instructions WHERE candidate_id=ct.id ORDER BY created_at LIMIT 1) execution_link ON true
		LEFT JOIN LATERAL (SELECT * FROM candidate_evidence_scores WHERE candidate_id=ct.id ORDER BY scored_at DESC LIMIT 1) ces ON true
		%s
		ORDER BY w.received_at,w.id
		LIMIT $%d`, where, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load selected genuine events: %w", err)
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("selected genuine events rows: %w", err)
	}
	return events, nil
}

func scanEvent(row pgx.Row) (Event, error) {
	var event Event
	var sourceURLs, confidenceReasons, assets, rawPayload, mappingMethods []byte
	var normalizedID, rawEventID uuid.NullUUID
	var collectionAt sql.NullTime
	var candidateRaw, evidenceRaw []byte
	err := row.Scan(
		&event.InboxID, &event.Source, &event.SourceEventID, &event.Status, &event.EventType, &event.Headline,
		&event.Summary, &sourceURLs, &event.SourceCount, &event.PublicationAt, &event.ReceiptAt, &event.Severity,
		&event.SourceTier, &event.Confidence, &confidenceReasons, &assets, &event.MappingReason, &rawPayload,
		&normalizedID, &rawEventID, &event.ProvenanceAvailable, &event.IsSynthetic, &event.SyntheticReason,
		&event.DataSourceType, &event.SourceProvider, &collectionAt, &event.DeterministicAnalysis, &event.AIAnalysisProvider,
		&mappingMethods, &candidateRaw, &evidenceRaw,
	)
	if err != nil {
		return Event{}, fmt.Errorf("scan selected genuine event: %w", err)
	}
	if normalizedID.Valid {
		id := normalizedID.UUID
		event.NormalizedEventID = &id
	}
	if rawEventID.Valid {
		id := rawEventID.UUID
		event.RawEventID = &id
	}
	if collectionAt.Valid {
		value := collectionAt.Time
		event.CollectionAt = &value
	}
	if err := json.Unmarshal(sourceURLs, &event.SourceURLs); err != nil {
		return Event{}, fmt.Errorf("decode source urls: %w", err)
	}
	if err := json.Unmarshal(confidenceReasons, &event.ConfidenceReasons); err != nil {
		return Event{}, fmt.Errorf("decode confidence reasons: %w", err)
	}
	if err := json.Unmarshal(assets, &event.AffectedAssets); err != nil {
		return Event{}, fmt.Errorf("decode affected assets: %w", err)
	}
	if err := json.Unmarshal(mappingMethods, &event.MappingMethods); err != nil {
		return Event{}, fmt.Errorf("decode mapping methods: %w", err)
	}
	if len(candidateRaw) > 0 {
		var candidate candidates.Candidate
		if err := json.Unmarshal(candidateRaw, &candidate); err != nil {
			return Event{}, fmt.Errorf("decode linked candidate: %w", err)
		}
		event.Candidate = &candidate
	}
	if len(evidenceRaw) > 0 {
		var score candidates.EvidenceScoreSummary
		if err := json.Unmarshal(evidenceRaw, &score); err != nil {
			return Event{}, fmt.Errorf("decode candidate evidence score: %w", err)
		}
		event.CandidateEvidenceScore = &score
	}
	return event, nil
}

func (s *Store) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
}

func persistDecision(ctx context.Context, tx pgx.Tx, event Event, result Result, rules Ruleset, decisionAt time.Time, fingerprint, replayIdentity string) (PersistedDecision, bool, error) {
	lockKey := event.InboxID.String() + ":" + rules.Version
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("lock event decision: %w", err)
	}
	existing, found, current, err := findDecisionByReplay(ctx, tx, replayIdentity)
	if err != nil {
		return PersistedDecision{}, false, err
	}
	if found {
		if !current {
			if _, err := tx.Exec(ctx, `UPDATE genuine_event_decisions SET is_current=false,updated_at=$2 WHERE source_inbox_event_id=$1 AND is_current`, event.InboxID, decisionAt); err != nil {
				return PersistedDecision{}, false, fmt.Errorf("retire current decision: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE genuine_event_decisions SET is_current=true,updated_at=$2 WHERE id=$1`, existing.ID, decisionAt); err != nil {
				return PersistedDecision{}, false, fmt.Errorf("restore replay decision projection: %w", err)
			}
			existing.UpdatedAt = decisionAt
		}
		return existing, true, nil
	}
	var version int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(decision_version),0)+1 FROM genuine_event_decisions WHERE source_inbox_event_id=$1`, event.InboxID).Scan(&version); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("next decision version: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE genuine_event_decisions SET is_current=false,updated_at=$2 WHERE source_inbox_event_id=$1 AND is_current`, event.InboxID, decisionAt); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("retire prior decision: %w", err)
	}
	mappingJSON, _ := json.Marshal(result.AssetMappingProvenance)
	replayMetadata, _ := json.Marshal(map[string]any{"candidateWriterInvoked": false, "sourceStatus": event.Status})
	sourceURL := event.SourceURLs[0]
	row := tx.QueryRow(ctx, `
		INSERT INTO genuine_event_decisions (
			source_inbox_event_id,normalized_event_id,source_event_identity,decision,decision_version,
			ruleset_version,processor_identity,processing_mode,decision_at,event_publication_at,event_collection_at,event_receipt_at,
			source,source_url,event_type,severity,evidence_score,evidence_score_source,confidence,affected_assets,unknown_assets,
			asset_mapping_provenance,reasons,blocking_reasons,missing_evidence,trust_gate_state,risk_review_state,candidate_id,
			replay_identity,input_fingerprint,replay_metadata,is_current,created_at,updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'deterministic',$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,true,$8,$8)
		RETURNING id,created_at,updated_at
	`, event.InboxID, event.NormalizedEventID, event.Source+":"+event.SourceEventID, result.Decision, version,
		rules.Version, rules.ProcessorIdentity, decisionAt, event.PublicationAt, event.CollectionAt, event.ReceiptAt,
		event.Source, sourceURL, event.EventType, event.Severity, result.EvidenceScore, result.EvidenceScoreSource, event.Confidence,
		result.AffectedAssets, result.UnknownAssets, mappingJSON, result.Reasons, result.BlockingReasons, result.MissingEvidence,
		result.TrustGateState, result.RiskReviewState, result.CandidateID, replayIdentity, fingerprint, replayMetadata)
	persisted := PersistedDecision{
		SourceInboxEventID: event.InboxID, NormalizedEventID: event.NormalizedEventID, SourceEventIdentity: event.Source + ":" + event.SourceEventID,
		Decision: result.Decision, DecisionVersion: version, RulesetVersion: rules.Version, ProcessorIdentity: rules.ProcessorIdentity,
		ProcessingMode: ProcessingModeDeterministic, DecisionAt: decisionAt, PublicationAt: event.PublicationAt, CollectionAt: event.CollectionAt,
		ReceiptAt: event.ReceiptAt, Source: event.Source, SourceURL: sourceURL, EventType: event.EventType, Severity: event.Severity,
		EvidenceScore: result.EvidenceScore, EvidenceScoreSource: result.EvidenceScoreSource, Confidence: event.Confidence,
		AffectedAssets: result.AffectedAssets, UnknownAssets: result.UnknownAssets, MappingProvenance: result.AssetMappingProvenance,
		Reasons: result.Reasons, BlockingReasons: result.BlockingReasons, MissingEvidence: result.MissingEvidence,
		TrustGateState: result.TrustGateState, RiskReviewState: result.RiskReviewState, CandidateID: result.CandidateID,
		ReplayIdentity: replayIdentity, InputFingerprint: fingerprint, ReplayMetadata: replayMetadata,
	}
	if err := row.Scan(&persisted.ID, &persisted.CreatedAt, &persisted.UpdatedAt); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("insert genuine event decision: %w", err)
	}
	return persisted, false, nil
}

func findDecisionByReplay(ctx context.Context, tx pgx.Tx, replayIdentity string) (PersistedDecision, bool, bool, error) {
	var decision PersistedDecision
	var normalizedID, candidateID uuid.NullUUID
	var collectionAt sql.NullTime
	var mapping, replayMetadata []byte
	var current bool
	err := tx.QueryRow(ctx, `SELECT id,source_inbox_event_id,normalized_event_id,source_event_identity,decision,decision_version,
		ruleset_version,processor_identity,processing_mode,decision_at,event_publication_at,event_collection_at,event_receipt_at,
		source,source_url,event_type,severity,evidence_score::float8,evidence_score_source,confidence::float8,affected_assets,unknown_assets,
		asset_mapping_provenance,reasons,blocking_reasons,missing_evidence,trust_gate_state,risk_review_state,candidate_id,
		replay_identity,input_fingerprint,replay_metadata,is_current,created_at,updated_at
		FROM genuine_event_decisions WHERE replay_identity=$1 FOR UPDATE`, replayIdentity).Scan(
		&decision.ID, &decision.SourceInboxEventID, &normalizedID, &decision.SourceEventIdentity, &decision.Decision, &decision.DecisionVersion,
		&decision.RulesetVersion, &decision.ProcessorIdentity, &decision.ProcessingMode, &decision.DecisionAt, &decision.PublicationAt,
		&collectionAt, &decision.ReceiptAt, &decision.Source, &decision.SourceURL, &decision.EventType, &decision.Severity,
		&decision.EvidenceScore, &decision.EvidenceScoreSource, &decision.Confidence, &decision.AffectedAssets, &decision.UnknownAssets,
		&mapping, &decision.Reasons, &decision.BlockingReasons, &decision.MissingEvidence, &decision.TrustGateState,
		&decision.RiskReviewState, &candidateID, &decision.ReplayIdentity, &decision.InputFingerprint, &replayMetadata,
		&current, &decision.CreatedAt, &decision.UpdatedAt)
	if err == pgx.ErrNoRows {
		return PersistedDecision{}, false, false, nil
	}
	if err != nil {
		return PersistedDecision{}, false, false, fmt.Errorf("find replay decision: %w", err)
	}
	if normalizedID.Valid {
		id := normalizedID.UUID
		decision.NormalizedEventID = &id
	}
	if candidateID.Valid {
		id := candidateID.UUID
		decision.CandidateID = &id
	}
	if collectionAt.Valid {
		value := collectionAt.Time
		decision.CollectionAt = &value
	}
	_ = json.Unmarshal(mapping, &decision.MappingProvenance)
	decision.ReplayMetadata = replayMetadata
	return decision, true, current, nil
}
