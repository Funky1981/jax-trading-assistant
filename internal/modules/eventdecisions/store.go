package eventdecisions

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
	"jax-trading-assistant/internal/modules/candidates"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type eventQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) LoadSelectedEvents(ctx context.Context, eventIdentity string, limit int) ([]Event, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("event decision store requires a database")
	}
	return loadSelectedEvents(ctx, s.pool, eventIdentity, limit)
}

func (s *Store) LoadSelectedEventsTx(ctx context.Context, tx pgx.Tx, eventIdentity string, limit int) ([]Event, error) {
	if tx == nil {
		return nil, fmt.Errorf("transactional event load requires a transaction")
	}
	return loadSelectedEvents(ctx, tx, eventIdentity, limit)
}

func loadSelectedEvents(ctx context.Context, db eventQueryer, eventIdentity string, limit int) ([]Event, error) {
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
			COALESCE(er.payload->'raw_payload'->>'article_url',''),
			COALESCE(er.payload->'raw_payload'->>'feed_url',''),
			COALESCE(er.payload->'raw_payload'->>'source_native_id',''),
			COALESCE(er.payload->'raw_payload'->>'source_name',''),
			COALESCE(er.payload->'raw_payload'->>'content_hash',''),
			COALESCE(w.region,''),COALESCE(w.asset_themes,'[]'::jsonb),
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
	rows, err := db.Query(ctx, query, args...)
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
	var sourceURLs, confidenceReasons, assets, rawPayload, mappingMethods, assetThemes []byte
	var normalizedID, rawEventID uuid.NullUUID
	var collectionAt sql.NullTime
	var candidateRaw, evidenceRaw []byte
	err := row.Scan(
		&event.InboxID, &event.Source, &event.SourceEventID, &event.Status, &event.EventType, &event.Headline,
		&event.Summary, &sourceURLs, &event.SourceCount, &event.PublicationAt, &event.ReceiptAt, &event.Severity,
		&event.SourceTier, &event.Confidence, &confidenceReasons, &assets, &event.MappingReason, &rawPayload,
		&event.ArticleURL, &event.FeedURL, &event.SourceNativeID, &event.SourceName, &event.ContentHash, &event.Region, &assetThemes,
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
	if err := json.Unmarshal(assetThemes, &event.AssetThemes); err != nil {
		return Event{}, fmt.Errorf("decode asset themes: %w", err)
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

func persistDecision(ctx context.Context, tx pgx.Tx, event Event, result Result, rules Ruleset, decisionAt time.Time, fingerprint, replayIdentity string, origin DecisionOrigin, decisionContext string) (PersistedDecision, bool, error) {
	lockKey := event.InboxID.String() + ":" + rules.Version
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("lock event decision: %w", err)
	}
	existing, found, err := findInitialDecision(ctx, tx, event.InboxID, rules.Version)
	if err != nil {
		return PersistedDecision{}, false, err
	}
	if found {
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
	replayMetadata, _ := json.Marshal(map[string]any{"candidateWriterInvoked": false, "sourceStatus": event.Status, "decisionOrigin": origin, "decisionContext": decisionContext})
	sourceURL := event.SourceURLs[0]
	row := tx.QueryRow(ctx, `
		INSERT INTO genuine_event_decisions (
			source_inbox_event_id,normalized_event_id,source_event_identity,decision,decision_version,
			ruleset_version,processor_identity,processing_mode,decision_at,event_publication_at,event_collection_at,event_receipt_at,
			source,source_url,event_type,severity,evidence_score,evidence_score_source,confidence,affected_assets,unknown_assets,
			asset_mapping_provenance,reasons,blocking_reasons,missing_evidence,trust_gate_state,risk_review_state,candidate_id,
			replay_identity,input_fingerprint,replay_metadata,is_current,created_at,updated_at,
			is_initial,decision_origin,decision_context
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'deterministic',$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,true,$8,$8,true,$31,$32)
		RETURNING id,created_at,updated_at
	`, event.InboxID, event.NormalizedEventID, event.Source+":"+event.SourceEventID, result.Decision, version,
		rules.Version, rules.ProcessorIdentity, decisionAt, event.PublicationAt, event.CollectionAt, event.ReceiptAt,
		event.Source, sourceURL, event.EventType, event.Severity, result.EvidenceScore, result.EvidenceScoreSource, event.Confidence,
		result.AffectedAssets, result.UnknownAssets, mappingJSON, result.Reasons, result.BlockingReasons, result.MissingEvidence,
		result.TrustGateState, result.RiskReviewState, result.CandidateID, replayIdentity, fingerprint, replayMetadata, origin, decisionContext)
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
		IsInitial: true, DecisionOrigin: origin, DecisionContext: decisionContext,
	}
	if err := row.Scan(&persisted.ID, &persisted.CreatedAt, &persisted.UpdatedAt); err != nil {
		return PersistedDecision{}, false, fmt.Errorf("insert genuine event decision: %w", err)
	}
	return persisted, false, nil
}

func findInitialDecision(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, rulesetVersion string) (PersistedDecision, bool, error) {
	var decision PersistedDecision
	var normalizedID, candidateID uuid.NullUUID
	var collectionAt sql.NullTime
	var mapping, replayMetadata []byte
	err := tx.QueryRow(ctx, `SELECT id,source_inbox_event_id,normalized_event_id,source_event_identity,decision,decision_version,
		ruleset_version,processor_identity,processing_mode,decision_at,event_publication_at,event_collection_at,event_receipt_at,
		source,source_url,event_type,severity,evidence_score::float8,evidence_score_source,confidence::float8,affected_assets,unknown_assets,
		asset_mapping_provenance,reasons,blocking_reasons,missing_evidence,trust_gate_state,risk_review_state,candidate_id,
		replay_identity,input_fingerprint,replay_metadata,created_at,updated_at,is_initial,decision_origin,decision_context
		FROM genuine_event_decisions WHERE source_inbox_event_id=$1 AND ruleset_version=$2 AND is_initial FOR UPDATE`, eventID, rulesetVersion).Scan(
		&decision.ID, &decision.SourceInboxEventID, &normalizedID, &decision.SourceEventIdentity, &decision.Decision, &decision.DecisionVersion,
		&decision.RulesetVersion, &decision.ProcessorIdentity, &decision.ProcessingMode, &decision.DecisionAt, &decision.PublicationAt,
		&collectionAt, &decision.ReceiptAt, &decision.Source, &decision.SourceURL, &decision.EventType, &decision.Severity,
		&decision.EvidenceScore, &decision.EvidenceScoreSource, &decision.Confidence, &decision.AffectedAssets, &decision.UnknownAssets,
		&mapping, &decision.Reasons, &decision.BlockingReasons, &decision.MissingEvidence, &decision.TrustGateState,
		&decision.RiskReviewState, &candidateID, &decision.ReplayIdentity, &decision.InputFingerprint, &replayMetadata,
		&decision.CreatedAt, &decision.UpdatedAt, &decision.IsInitial, &decision.DecisionOrigin, &decision.DecisionContext)
	if err == pgx.ErrNoRows {
		return PersistedDecision{}, false, nil
	}
	if err != nil {
		return PersistedDecision{}, false, fmt.Errorf("find initial decision: %w", err)
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
	return decision, true, nil
}

func persistAssetResolution(ctx context.Context, tx pgx.Tx, event Event, decision PersistedDecision, resolution assetresolution.Result, createdAt time.Time) error {
	var found bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM event_asset_resolutions WHERE decision_id=$1)`, decision.ID).Scan(&found); err != nil {
		return err
	}
	if found {
		return nil
	}
	knownAtDecision := decision.DecisionOrigin == DecisionOriginLive && resolution.KnowableAtOperationalAnchor
	sourceValues, _ := json.Marshal(resolution.SourceValues)
	provenance, _ := json.Marshal(map[string]any{
		"resolver": resolution.RulesetVersion, "decisionRuleset": decision.RulesetVersion,
		"inputFingerprint": decision.InputFingerprint, "contentBounded": true, "llmUsed": false,
	})
	fingerprintRaw, _ := json.Marshal(struct {
		DecisionID string                 `json:"decisionId"`
		Resolution assetresolution.Result `json:"resolution"`
	}{DecisionID: decision.ID.String(), Resolution: resolution})
	digest := sha256.Sum256(fingerprintRaw)
	fingerprint := hex.EncodeToString(digest[:])
	var effectiveFrom, effectiveTo any
	if resolution.EffectiveFrom != nil {
		effectiveFrom = resolution.EffectiveFrom.Format("2006-01-02")
	}
	if resolution.EffectiveTo != nil {
		effectiveTo = resolution.EffectiveTo.Format("2006-01-02")
	}
	_, err := tx.Exec(ctx, `INSERT INTO event_asset_resolutions (
		decision_id,source_inbox_event_id,normalized_event_id,resolution_status,resolved_symbol,benchmark_symbol,
		mapping_type,asset_relationship,confidence_class,deterministic_reason,source_fields,source_values,
		resolver_ruleset_version,decision_origin,known_at_initial_decision_time,knowable_at_operational_anchor,
		ambiguity_reason,rejection_reason,canonical_entity,asset_class,exchange_name,effective_from,effective_to,
		provenance,resolution_fingerprint,created_at
	) VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16,
		NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),NULLIF($20,''),NULLIF($21,''),$22::date,$23::date,$24::jsonb,$25,$26)`,
		decision.ID, event.InboxID, event.NormalizedEventID, resolution.Status, resolution.Symbol, resolution.Benchmark,
		resolution.MappingType, resolution.Relationship, resolution.ConfidenceClass, resolution.Reason, resolution.SourceFields, string(sourceValues),
		resolution.RulesetVersion, decision.DecisionOrigin, knownAtDecision, resolution.KnowableAtOperationalAnchor,
		resolution.AmbiguityReason, resolution.RejectionReason, resolution.CanonicalEntity, resolution.AssetClass, resolution.Exchange,
		effectiveFrom, effectiveTo, string(provenance), fingerprint, createdAt)
	if err != nil {
		return err
	}
	if resolution.Status == assetresolution.StatusResolved && event.NormalizedEventID != nil {
		_, err = tx.Exec(ctx, `INSERT INTO event_symbol_map (normalized_event_id,symbol,relevance,mapping_method,is_primary,created_at)
			VALUES ($1,$2,1,$3,true,$4) ON CONFLICT (normalized_event_id,symbol) DO NOTHING`, event.NormalizedEventID, resolution.Symbol, resolution.RulesetVersion+":"+resolution.MappingType, createdAt)
	}
	return err
}
