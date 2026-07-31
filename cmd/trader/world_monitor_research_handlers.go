package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type worldMonitorResearchIngestService interface {
	Ingest(ctx context.Context, trigger worldMonitorResearchTrigger) (worldMonitorResearchReceipt, error)
}

type worldMonitorOpportunityPromoteService interface {
	PromotePending(ctx context.Context, limit int) (worldMonitorPromotionResult, error)
}

type worldMonitorResearchStatusService interface {
	Status(ctx context.Context) (worldMonitorResearchStatus, error)
}

type worldMonitorResearchInboxListService interface {
	List(ctx context.Context, filter worldMonitorResearchInboxFilter) (worldMonitorResearchInboxList, error)
}

type worldMonitorResearchInboxFilter struct {
	Status string
	Limit  int
}

type worldMonitorResearchInboxList struct {
	Items     []worldMonitorResearchInboxItem `json:"items"`
	Total     int                             `json:"total"`
	Counts    worldMonitorEvidenceCounts      `json:"counts"`
	CheckedAt time.Time                       `json:"checkedAt"`
}

type worldMonitorEvidenceCounts struct {
	Genuine            int `json:"genuine"`
	SyntheticTests     int `json:"syntheticTests"`
	Rejected           int `json:"rejected"`
	Duplicates         int `json:"duplicates"`
	CandidatesCreated  int `json:"candidatesCreated"`
	NoTrade            int `json:"noTrade"`
	Watch              int `json:"watch"`
	Candidate          int `json:"candidate"`
	AwaitingProcessing int `json:"awaitingProcessing"`
}

type worldMonitorEventDecision struct {
	DecisionID             string         `json:"decisionId"`
	Decision               string         `json:"decision"`
	DecisionVersion        int            `json:"decisionVersion"`
	RulesetVersion         string         `json:"rulesetVersion"`
	ProcessorIdentity      string         `json:"processorIdentity"`
	ProcessingMode         string         `json:"processingMode"`
	DecisionAt             time.Time      `json:"decisionAt"`
	EvidenceScore          float64        `json:"evidenceScore"`
	EvidenceScoreSource    string         `json:"evidenceScoreSource"`
	AffectedAssets         []string       `json:"affectedAssets"`
	UnknownAssets          bool           `json:"unknownAssets"`
	AssetMappingProvenance map[string]any `json:"assetMappingProvenance"`
	Reasons                []string       `json:"reasons"`
	BlockingReasons        []string       `json:"blockingReasons"`
	MissingEvidence        []string       `json:"missingEvidence"`
	TrustGateState         string         `json:"trustGateState"`
	RiskReviewState        string         `json:"riskReviewState"`
	CandidateID            string         `json:"candidateId,omitempty"`
	ReplayIdentity         string         `json:"replayIdentity"`
	CreatedAt              time.Time      `json:"createdAt"`
	UpdatedAt              time.Time      `json:"updatedAt"`
}

type worldMonitorEvidenceSubjectSummary struct {
	SubjectID        string    `json:"subjectId"`
	SubjectType      string    `json:"subjectType"`
	CanonicalLabel   string    `json:"canonicalLabel"`
	CurrentDecision  string    `json:"currentDecision"`
	DecisionReason   string    `json:"decisionReason"`
	MissingEvidence  []string  `json:"missingEvidence"`
	LinkedEventCount int       `json:"linkedEventCount"`
	SourceGroupCount int       `json:"sourceGroupCount"`
	FirstObservedAt  time.Time `json:"firstObservedAt"`
	LatestEvidenceAt time.Time `json:"latestEvidenceAt"`
	LatestDecisionAt time.Time `json:"latestDecisionAt"`
	RulesetVersion   string    `json:"rulesetVersion"`
	ResolvedAssets   []string  `json:"resolvedAssets"`
	UnknownAssets    bool      `json:"unknownAssets"`
	CandidateID      string    `json:"candidateId,omitempty"`
}

type worldMonitorSubjectEvidenceItem struct {
	SourceEventID        string    `json:"sourceEventId"`
	Headline             string    `json:"headline"`
	Source               string    `json:"source"`
	SourceURL            string    `json:"sourceUrl,omitempty"`
	RelationshipType     string    `json:"relationshipType"`
	AssociationReason    string    `json:"associationReason"`
	SourceIndependence   string    `json:"sourceIndependence"`
	EvidenceContribution string    `json:"evidenceContribution"`
	ContradictionState   string    `json:"contradictionState"`
	PublicationAt        time.Time `json:"publicationAt"`
	ReceiptAt            time.Time `json:"receiptAt"`
	LinkedAt             time.Time `json:"linkedAt"`
}

type worldMonitorSubjectEvaluationItem struct {
	PreviousDecision      string    `json:"previousDecision"`
	NewDecision           string    `json:"newDecision"`
	DeterministicReason   string    `json:"deterministicReason"`
	MissingEvidence       []string  `json:"missingEvidence"`
	RulesetVersion        string    `json:"rulesetVersion"`
	EvaluatedAt           time.Time `json:"evaluatedAt"`
	TriggeringSourceEvent string    `json:"triggeringSourceEventId"`
}

type worldMonitorEvidenceSubjectDetail struct {
	Subject     worldMonitorEvidenceSubjectSummary  `json:"subject"`
	Evidence    []worldMonitorSubjectEvidenceItem   `json:"evidence"`
	Evaluations []worldMonitorSubjectEvaluationItem `json:"evaluations"`
}

type worldMonitorResearchInboxItem struct {
	ID                   string                              `json:"id"`
	Source               string                              `json:"source"`
	SourceEventID        string                              `json:"sourceEventId"`
	WorldMonitorEventID  string                              `json:"worldMonitorEventId"`
	Status               string                              `json:"status"`
	RejectionReason      string                              `json:"rejectionReason,omitempty"`
	EventType            string                              `json:"eventType"`
	Headline             string                              `json:"headline"`
	Summary              string                              `json:"summary,omitempty"`
	SourceURLs           []string                            `json:"sourceUrls"`
	SourceCount          int                                 `json:"sourceCount"`
	EventTime            time.Time                           `json:"eventTime"`
	PublishedAt          *time.Time                          `json:"publishedAt,omitempty"`
	ReceivedAt           time.Time                           `json:"receivedAt"`
	CollectedAt          *time.Time                          `json:"collectedAt,omitempty"`
	RawEventID           string                              `json:"rawEventId,omitempty"`
	ProvenanceAvailable  bool                                `json:"provenanceAvailable"`
	IsSynthetic          bool                                `json:"isSynthetic"`
	SyntheticReason      string                              `json:"syntheticReason,omitempty"`
	DiscoveryMethod      string                              `json:"discoveryMethod,omitempty"`
	AnalysisIdentity     string                              `json:"analysisIdentity,omitempty"`
	AIProvider           string                              `json:"aiProvider,omitempty"`
	AIModel              string                              `json:"aiModel,omitempty"`
	Region               string                              `json:"region,omitempty"`
	PossibleAffectedETFs []string                            `json:"possibleAffectedEtfs"`
	AssetThemes          []string                            `json:"assetThemes"`
	Severity             string                              `json:"severity"`
	SourceTier           string                              `json:"sourceTier"`
	Confidence           float64                             `json:"confidence"`
	ConfidenceReasons    []string                            `json:"confidenceReasons"`
	MappingReason        string                              `json:"mappingReason"`
	NormalizedEventID    string                              `json:"normalizedEventId,omitempty"`
	NormalizedAt         *time.Time                          `json:"normalizedAt,omitempty"`
	CandidateID          string                              `json:"candidateId,omitempty"`
	CandidateSymbol      string                              `json:"candidateSymbol,omitempty"`
	CandidateStatus      string                              `json:"candidateStatus,omitempty"`
	CandidateCreatedAt   *time.Time                          `json:"candidateCreatedAt,omitempty"`
	ApprovalID           string                              `json:"approvalId,omitempty"`
	ApprovalDecision     string                              `json:"approvalDecision,omitempty"`
	ApprovalAt           *time.Time                          `json:"approvalAt,omitempty"`
	PaperTicketID        string                              `json:"paperTicketId,omitempty"`
	PaperTicketCreatedAt *time.Time                          `json:"paperTicketCreatedAt,omitempty"`
	OutcomeCount         int                                 `json:"outcomeCount"`
	LatestOutcomeAt      *time.Time                          `json:"latestOutcomeAt,omitempty"`
	OperatorDecision     string                              `json:"operatorDecision,omitempty"`
	OperatorReason       string                              `json:"operatorReason,omitempty"`
	Decision             *worldMonitorEventDecision          `json:"decision,omitempty"`
	DecisionHistory      []worldMonitorEventDecision         `json:"decisionHistory"`
	Subject              *worldMonitorEvidenceSubjectSummary `json:"subject,omitempty"`
	RawPayload           map[string]any                      `json:"rawPayload"`
}

type worldMonitorResearchStatus struct {
	Connected         bool                             `json:"connected"`
	LastReceivedAt    *time.Time                       `json:"lastReceivedAt,omitempty"`
	LastSourceEventID string                           `json:"lastSourceEventId,omitempty"`
	LastStatus        string                           `json:"lastStatus,omitempty"`
	LastHeadline      string                           `json:"lastHeadline,omitempty"`
	LastSymbols       []string                         `json:"lastSymbols,omitempty"`
	LastCandidateID   string                           `json:"lastCandidateId,omitempty"`
	Counts            worldMonitorResearchStatusCounts `json:"counts"`
	CheckedAt         time.Time                        `json:"checkedAt"`
}

type worldMonitorResearchStatusCounts struct {
	Total             int `json:"total"`
	Pending           int `json:"pending"`
	CandidatesCreated int `json:"candidatesCreated"`
	Rejected          int `json:"rejected"`
	Ignored           int `json:"ignored"`
}

var newWorldMonitorResearchIngestService = func(pool *pgxpool.Pool) worldMonitorResearchIngestService {
	return newWorldMonitorResearchInboxService(pool)
}

var newWorldMonitorOpportunityPromoteService = func(pool *pgxpool.Pool) worldMonitorOpportunityPromoteService {
	return newWorldMonitorOpportunityPromoter(pool)
}

var newWorldMonitorResearchStatusService = func(pool *pgxpool.Pool) worldMonitorResearchStatusService {
	return &worldMonitorResearchStatusStore{pool: pool}
}

var newWorldMonitorResearchInboxListService = func(pool *pgxpool.Pool) worldMonitorResearchInboxListService {
	return &worldMonitorResearchStatusStore{pool: pool}
}

type worldMonitorResearchStatusStore struct {
	pool *pgxpool.Pool
}

func (s *worldMonitorResearchStatusStore) Status(ctx context.Context) (worldMonitorResearchStatus, error) {
	status := worldMonitorResearchStatus{
		LastSymbols: []string{},
		CheckedAt:   time.Now().UTC(),
	}
	if s.pool == nil {
		return status, nil
	}

	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'new')::int,
			COUNT(*) FILTER (WHERE status = 'candidate_created')::int,
			COUNT(*) FILTER (WHERE status = 'rejected')::int,
			COUNT(*) FILTER (WHERE status = 'ignored')::int
		FROM world_monitor_research_inbox
	`).Scan(
		&status.Counts.Total,
		&status.Counts.Pending,
		&status.Counts.CandidatesCreated,
		&status.Counts.Rejected,
		&status.Counts.Ignored,
	)
	if err != nil {
		return worldMonitorResearchStatus{}, fmt.Errorf("world monitor status counts: %w", err)
	}

	var (
		lastReceivedAt sql.NullTime
		sourceEventID  sql.NullString
		lastState      sql.NullString
		headline       sql.NullString
		candidateID    sql.NullString
		symbolsRaw     []byte
	)
	err = s.pool.QueryRow(ctx, `
		SELECT received_at, source_event_id, status, headline, COALESCE(possible_affected_etfs, '[]'::jsonb), candidate_id::text
		FROM world_monitor_research_inbox
		ORDER BY received_at DESC
		LIMIT 1
	`).Scan(&lastReceivedAt, &sourceEventID, &lastState, &headline, &symbolsRaw, &candidateID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return status, nil
		}
		return worldMonitorResearchStatus{}, fmt.Errorf("world monitor latest trigger: %w", err)
	}

	status.Connected = true
	if lastReceivedAt.Valid {
		status.LastReceivedAt = &lastReceivedAt.Time
	}
	status.LastSourceEventID = sourceEventID.String
	status.LastStatus = lastState.String
	status.LastHeadline = headline.String
	status.LastCandidateID = candidateID.String
	_ = json.Unmarshal(symbolsRaw, &status.LastSymbols)
	if status.LastSymbols == nil {
		status.LastSymbols = []string{}
	}
	return status, nil
}

func (s *worldMonitorResearchStatusStore) List(ctx context.Context, filter worldMonitorResearchInboxFilter) (worldMonitorResearchInboxList, error) {
	if filter.Limit <= 0 || filter.Limit > 250 {
		filter.Limit = 100
	}
	out := worldMonitorResearchInboxList{
		Items:     []worldMonitorResearchInboxItem{},
		CheckedAt: time.Now().UTC(),
	}
	if s.pool == nil {
		return out, nil
	}

	var total int
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE er.id IS NOT NULL AND NOT er.is_synthetic)::int,
			COUNT(*) FILTER (WHERE er.id IS NOT NULL AND er.is_synthetic)::int,
			COUNT(*) FILTER (WHERE w.status = 'rejected')::int,
			COUNT(*) FILTER (WHERE w.status = 'ignored' AND
				COALESCE(w.rejection_reason, w.operator_reason, '') ILIKE '%dedup%')::int,
			COUNT(*) FILTER (WHERE w.candidate_id IS NOT NULL)::int,
			COUNT(*) FILTER (WHERE gd.decision='NO_TRADE')::int,
			COUNT(*) FILTER (WHERE gd.decision='WATCH')::int,
			COUNT(*) FILTER (WHERE gd.decision='CANDIDATE')::int,
			COUNT(*) FILTER (WHERE er.id IS NOT NULL AND NOT er.is_synthetic AND w.status<>'rejected' AND gd.id IS NULL)::int
		FROM world_monitor_research_inbox w
		LEFT JOIN event_normalized en ON en.id=w.normalized_event_id
		LEFT JOIN LATERAL (
			SELECT candidate.id, candidate.is_synthetic
			FROM event_raw candidate
			WHERE candidate.id=en.raw_event_id
			   OR (candidate.source_event_id=w.source_event_id AND candidate.source_id=w.source)
			ORDER BY (candidate.id=en.raw_event_id) DESC, candidate.received_at DESC
			LIMIT 1
		) er ON true
		LEFT JOIN LATERAL (
			SELECT current_decision.id,current_decision.decision
			FROM genuine_event_decisions current_decision
			WHERE current_decision.source_inbox_event_id=w.id AND current_decision.is_current
			ORDER BY current_decision.decision_at DESC,current_decision.decision_version DESC
			LIMIT 1
		) gd ON true
	`).Scan(
		&total,
		&out.Counts.Genuine,
		&out.Counts.SyntheticTests,
		&out.Counts.Rejected,
		&out.Counts.Duplicates,
		&out.Counts.CandidatesCreated,
		&out.Counts.NoTrade,
		&out.Counts.Watch,
		&out.Counts.Candidate,
		&out.Counts.AwaitingProcessing,
	)
	if err != nil {
		return worldMonitorResearchInboxList{}, fmt.Errorf("world monitor inbox counts: %w", err)
	}
	out.Total = total

	args := []any{}
	where := "WHERE 1=1"
	if filter.Status != "" && filter.Status != "all" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(" AND w.status = $%d", len(args))
	}
	args = append(args, filter.Limit)
	query := fmt.Sprintf(`
		SELECT
			w.id::text, w.source, w.source_event_id, w.world_monitor_event_id, w.status,
			COALESCE(w.rejection_reason, ''), w.event_type, w.headline, COALESCE(w.summary, ''),
			COALESCE(w.source_urls, '[]'::jsonb), w.source_count, w.event_time,
			CASE WHEN COALESCE(er.payload->'raw_payload'->>'publication_time_supplied','true')='false' THEN NULL ELSE w.event_time END,
			w.received_at,
			COALESCE(w.region, ''), COALESCE(w.possible_affected_etfs, '[]'::jsonb), COALESCE(w.asset_themes, '[]'::jsonb),
			w.severity, w.source_tier, w.confidence, COALESCE(w.confidence_reasons, '[]'::jsonb), w.mapping_reason,
			COALESCE(w.normalized_event_id::text, ''), COALESCE(w.candidate_id::text, ''),
			COALESCE(w.operator_decision, ''), COALESCE(w.operator_reason, ''),
			COALESCE(w.raw_payload, '{}'::jsonb),
			NULLIF(COALESCE(er.payload->>'collection_timestamp_utc', er.payload->>'collected_at', er.payload->>'collectedAt', ''), '')::timestamptz,
			COALESCE(er.id::text, ''), er.id IS NOT NULL, COALESCE(er.is_synthetic, false), COALESCE(er.synthetic_reason, ''),
			COALESCE(er.payload->>'discovery_method', er.payload->>'discoveryMethod', ''),
			COALESCE(er.payload->>'deterministic_analysis', er.payload->>'deterministic_analysis_identity', er.payload->>'analysis_identity',
				CASE WHEN COALESCE(er.payload->>'analysis_provider','')='' THEN er.payload->>'analysis_model' ELSE '' END, ''),
			COALESCE(er.payload->>'analysis_provider', er.payload->>'ai_provider', ''),
			CASE WHEN COALESCE(er.payload->>'analysis_provider', er.payload->>'ai_provider', '')<>'' THEN COALESCE(er.payload->>'analysis_model', er.payload->>'ai_model', '') ELSE '' END,
			en.created_at,
			COALESCE(ct.symbol, ''), COALESCE(ct.status, ''), ct.created_at,
			COALESCE(ca.id::text, ''), COALESCE(ca.decision, ''), ca.decided_at,
			COALESCE(pt.paper_ticket_id, ''), pt.created_at,
			COALESCE(oc.outcome_count, 0), oc.latest_outcome_at,
			decision_current.payload, COALESCE(decision_history.payload,'[]'::jsonb), subject_current.payload
		FROM world_monitor_research_inbox w
		LEFT JOIN event_normalized en ON en.id=w.normalized_event_id
		LEFT JOIN LATERAL (
			SELECT candidate.* FROM event_raw candidate
			WHERE candidate.id=en.raw_event_id
			   OR (candidate.source_event_id=w.source_event_id AND candidate.source_id=w.source)
			ORDER BY (candidate.id=en.raw_event_id) DESC, candidate.received_at DESC
			LIMIT 1
		) er ON true
		LEFT JOIN candidate_trades ct ON ct.id=w.candidate_id
		LEFT JOIN LATERAL (
			SELECT id, decision, decided_at
			FROM candidate_approvals
			WHERE candidate_id=w.candidate_id
			ORDER BY decided_at DESC
			LIMIT 1
		) ca ON true
		LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id=w.candidate_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS outcome_count, MAX(updated_at) AS latest_outcome_at
			FROM paper_ticket_outcome_checkpoints
			WHERE paper_ticket_id=pt.paper_ticket_id
		) oc ON true
		LEFT JOIN LATERAL (
			SELECT jsonb_build_object(
				'decisionId',d.id,'decision',d.decision,'decisionVersion',d.decision_version,
				'rulesetVersion',d.ruleset_version,'processorIdentity',d.processor_identity,'processingMode',d.processing_mode,
				'decisionAt',d.decision_at,'evidenceScore',d.evidence_score,'evidenceScoreSource',d.evidence_score_source,
				'affectedAssets',d.affected_assets,'unknownAssets',d.unknown_assets,'assetMappingProvenance',d.asset_mapping_provenance,
				'reasons',d.reasons,'blockingReasons',d.blocking_reasons,'missingEvidence',d.missing_evidence,
				'trustGateState',d.trust_gate_state,'riskReviewState',d.risk_review_state,'candidateId',d.candidate_id,
				'replayIdentity',d.replay_identity,'createdAt',d.created_at,'updatedAt',d.updated_at
			) AS payload
			FROM genuine_event_decisions d
			WHERE d.source_inbox_event_id=w.id AND d.is_current
			ORDER BY d.decision_at DESC,d.decision_version DESC LIMIT 1
		) decision_current ON true
		LEFT JOIN LATERAL (
			SELECT jsonb_agg(jsonb_build_object(
				'decisionId',d.id,'decision',d.decision,'decisionVersion',d.decision_version,
				'rulesetVersion',d.ruleset_version,'processorIdentity',d.processor_identity,'processingMode',d.processing_mode,
				'decisionAt',d.decision_at,'evidenceScore',d.evidence_score,'evidenceScoreSource',d.evidence_score_source,
				'affectedAssets',d.affected_assets,'unknownAssets',d.unknown_assets,'assetMappingProvenance',d.asset_mapping_provenance,
				'reasons',d.reasons,'blockingReasons',d.blocking_reasons,'missingEvidence',d.missing_evidence,
				'trustGateState',d.trust_gate_state,'riskReviewState',d.risk_review_state,'candidateId',d.candidate_id,
				'replayIdentity',d.replay_identity,'createdAt',d.created_at,'updatedAt',d.updated_at
			) ORDER BY d.decision_at DESC,d.decision_version DESC) AS payload
			FROM genuine_event_decisions d WHERE d.source_inbox_event_id=w.id
		) decision_history ON true
		LEFT JOIN LATERAL (
			SELECT jsonb_build_object(
				'subjectId',s.public_id,'subjectType',s.subject_type,'canonicalLabel',s.canonical_label,
				'currentDecision',s.current_decision,'decisionReason',s.current_decision_reason,
				'missingEvidence',s.current_missing_evidence,'linkedEventCount',counts.linked_event_count,
				'sourceGroupCount',counts.source_group_count,'firstObservedAt',s.first_observed_at,
				'latestEvidenceAt',s.latest_evidence_at,'latestDecisionAt',s.latest_evaluation_at,
				'rulesetVersion',s.ruleset_version,'resolvedAssets',s.resolved_assets,
				'unknownAssets',s.unknown_assets,'candidateId',s.candidate_id
			) AS payload
			FROM evidence_subject_events link
			JOIN evidence_subjects s ON s.id=link.subject_id
			JOIN LATERAL (
				SELECT COUNT(*)::int linked_event_count,COUNT(DISTINCT source_group_key)::int source_group_count
				FROM evidence_subject_events WHERE subject_id=s.id
			) counts ON true
			WHERE link.genuine_event_id=w.id
		) subject_current ON true
		%s
		ORDER BY w.received_at DESC
		LIMIT $%d
	`, where, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return worldMonitorResearchInboxList{}, fmt.Errorf("world monitor inbox rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item worldMonitorResearchInboxItem
		var sourceURLsRaw, etfsRaw, themesRaw, confidenceReasonsRaw, rawPayload, decisionRaw, decisionHistoryRaw, subjectRaw []byte
		if err := rows.Scan(
			&item.ID,
			&item.Source,
			&item.SourceEventID,
			&item.WorldMonitorEventID,
			&item.Status,
			&item.RejectionReason,
			&item.EventType,
			&item.Headline,
			&item.Summary,
			&sourceURLsRaw,
			&item.SourceCount,
			&item.EventTime,
			&item.PublishedAt,
			&item.ReceivedAt,
			&item.Region,
			&etfsRaw,
			&themesRaw,
			&item.Severity,
			&item.SourceTier,
			&item.Confidence,
			&confidenceReasonsRaw,
			&item.MappingReason,
			&item.NormalizedEventID,
			&item.CandidateID,
			&item.OperatorDecision,
			&item.OperatorReason,
			&rawPayload,
			&item.CollectedAt,
			&item.RawEventID,
			&item.ProvenanceAvailable,
			&item.IsSynthetic,
			&item.SyntheticReason,
			&item.DiscoveryMethod,
			&item.AnalysisIdentity,
			&item.AIProvider,
			&item.AIModel,
			&item.NormalizedAt,
			&item.CandidateSymbol,
			&item.CandidateStatus,
			&item.CandidateCreatedAt,
			&item.ApprovalID,
			&item.ApprovalDecision,
			&item.ApprovalAt,
			&item.PaperTicketID,
			&item.PaperTicketCreatedAt,
			&item.OutcomeCount,
			&item.LatestOutcomeAt,
			&decisionRaw,
			&decisionHistoryRaw,
			&subjectRaw,
		); err != nil {
			return worldMonitorResearchInboxList{}, err
		}
		_ = json.Unmarshal(sourceURLsRaw, &item.SourceURLs)
		_ = json.Unmarshal(etfsRaw, &item.PossibleAffectedETFs)
		_ = json.Unmarshal(themesRaw, &item.AssetThemes)
		_ = json.Unmarshal(confidenceReasonsRaw, &item.ConfidenceReasons)
		_ = json.Unmarshal(rawPayload, &item.RawPayload)
		if len(decisionRaw) > 0 {
			item.Decision = &worldMonitorEventDecision{}
			_ = json.Unmarshal(decisionRaw, item.Decision)
		}
		_ = json.Unmarshal(decisionHistoryRaw, &item.DecisionHistory)
		if len(subjectRaw) > 0 {
			item.Subject = &worldMonitorEvidenceSubjectSummary{}
			_ = json.Unmarshal(subjectRaw, item.Subject)
		}
		if item.SourceURLs == nil {
			item.SourceURLs = []string{}
		}
		if item.PossibleAffectedETFs == nil {
			item.PossibleAffectedETFs = []string{}
		}
		if item.AssetThemes == nil {
			item.AssetThemes = []string{}
		}
		if item.ConfidenceReasons == nil {
			item.ConfidenceReasons = []string{}
		}
		if item.RawPayload == nil {
			item.RawPayload = map[string]any{}
		}
		if item.DecisionHistory == nil {
			item.DecisionHistory = []worldMonitorEventDecision{}
		}
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return worldMonitorResearchInboxList{}, err
	}
	return out, nil
}

func (s *worldMonitorResearchStatusStore) SubjectDetail(ctx context.Context, publicID string, limit int) (worldMonitorEvidenceSubjectDetail, error) {
	if s.pool == nil {
		return worldMonitorEvidenceSubjectDetail{}, pgx.ErrNoRows
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var detail worldMonitorEvidenceSubjectDetail
	var missing, assets []string
	err := s.pool.QueryRow(ctx, `
		SELECT s.public_id,s.subject_type,s.canonical_label,s.current_decision,s.current_decision_reason,
			s.current_missing_evidence,counts.linked_event_count,counts.source_group_count,s.first_observed_at,
			s.latest_evidence_at,s.latest_evaluation_at,s.ruleset_version,s.resolved_assets,s.unknown_assets,
			COALESCE(s.candidate_id::text,'')
		FROM evidence_subjects s
		JOIN LATERAL (
			SELECT COUNT(*)::int linked_event_count,COUNT(DISTINCT source_group_key)::int source_group_count
			FROM evidence_subject_events WHERE subject_id=s.id
		) counts ON true
		WHERE s.public_id=$1
	`, publicID).Scan(&detail.Subject.SubjectID, &detail.Subject.SubjectType, &detail.Subject.CanonicalLabel,
		&detail.Subject.CurrentDecision, &detail.Subject.DecisionReason, &missing, &detail.Subject.LinkedEventCount,
		&detail.Subject.SourceGroupCount, &detail.Subject.FirstObservedAt, &detail.Subject.LatestEvidenceAt,
		&detail.Subject.LatestDecisionAt, &detail.Subject.RulesetVersion, &assets, &detail.Subject.UnknownAssets,
		&detail.Subject.CandidateID)
	if err != nil {
		return worldMonitorEvidenceSubjectDetail{}, err
	}
	detail.Subject.MissingEvidence = missing
	detail.Subject.ResolvedAssets = assets
	detail.Evidence = []worldMonitorSubjectEvidenceItem{}
	detail.Evaluations = []worldMonitorSubjectEvaluationItem{}
	rows, err := s.pool.Query(ctx, `
		SELECT w.source_event_id,w.headline,w.source,COALESCE(w.source_urls->>0,''),l.relationship_type,
			l.association_reason,l.source_independence,l.evidence_contribution,l.contradiction_state,
			l.publication_at,l.receipt_at,l.linked_at
		FROM evidence_subject_events l JOIN evidence_subjects s ON s.id=l.subject_id
		JOIN world_monitor_research_inbox w ON w.id=l.genuine_event_id
		WHERE s.public_id=$1 ORDER BY l.publication_at DESC,l.genuine_event_id LIMIT $2
	`, publicID, limit)
	if err != nil {
		return worldMonitorEvidenceSubjectDetail{}, fmt.Errorf("subject evidence detail: %w", err)
	}
	for rows.Next() {
		var item worldMonitorSubjectEvidenceItem
		if err := rows.Scan(&item.SourceEventID, &item.Headline, &item.Source, &item.SourceURL, &item.RelationshipType,
			&item.AssociationReason, &item.SourceIndependence, &item.EvidenceContribution, &item.ContradictionState,
			&item.PublicationAt, &item.ReceiptAt, &item.LinkedAt); err != nil {
			rows.Close()
			return worldMonitorEvidenceSubjectDetail{}, err
		}
		detail.Evidence = append(detail.Evidence, item)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `
		SELECT e.previous_decision,e.new_decision,e.deterministic_reason,e.missing_evidence,
			e.ruleset_version,e.evaluated_at,w.source_event_id
		FROM evidence_subject_evaluations e JOIN evidence_subjects s ON s.id=e.subject_id
		JOIN world_monitor_research_inbox w ON w.id=e.triggering_event_id
		WHERE s.public_id=$1 ORDER BY e.evaluated_at DESC,e.id DESC LIMIT $2
	`, publicID, limit)
	if err != nil {
		return worldMonitorEvidenceSubjectDetail{}, fmt.Errorf("subject evaluation detail: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item worldMonitorSubjectEvaluationItem
		if err := rows.Scan(&item.PreviousDecision, &item.NewDecision, &item.DeterministicReason, &item.MissingEvidence,
			&item.RulesetVersion, &item.EvaluatedAt, &item.TriggeringSourceEvent); err != nil {
			return worldMonitorEvidenceSubjectDetail{}, err
		}
		detail.Evaluations = append(detail.Evaluations, item)
	}
	return detail, rows.Err()
}

func worldMonitorEvidenceSubjectHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		publicID := strings.TrimPrefix(r.URL.Path, "/api/v1/research/events/world-monitor/subjects/")
		if !regexp.MustCompile(`^es_[a-f0-9]{24}$`).MatchString(publicID) {
			http.Error(w, "invalid evidence subject", http.StatusBadRequest)
			return
		}
		detail, err := (&worldMonitorResearchStatusStore{pool: pool}).SubjectDetail(r.Context(), publicID, 20)
		if err == pgx.ErrNoRows {
			http.Error(w, "evidence subject not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load evidence subject", http.StatusInternalServerError)
			return
		}
		jsonOK(w, detail)
	}
}

func worldMonitorResearchIngestHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var trigger worldMonitorResearchTrigger
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&trigger); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		receipt, err := newWorldMonitorResearchIngestService(pool).Ingest(r.Context(), trigger)
		if err != nil {
			http.Error(w, fmt.Sprintf("ingest world monitor trigger: %v", err), http.StatusInternalServerError)
			return
		}

		statusCode := http.StatusAccepted
		if receipt.Duplicate {
			statusCode = http.StatusOK
		}
		if receipt.Status == worldMonitorInboxStatusRejected {
			statusCode = http.StatusUnprocessableEntity
		}
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(receipt)
	}
}

func worldMonitorResearchInboxHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		filter := worldMonitorResearchInboxFilter{
			Status: r.URL.Query().Get("status"),
			Limit:  limit,
		}
		result, err := newWorldMonitorResearchInboxListService(pool).List(r.Context(), filter)
		if err != nil {
			http.Error(w, fmt.Sprintf("world monitor inbox: %v", err), http.StatusInternalServerError)
			return
		}
		jsonOK(w, result)
	}
}

func worldMonitorResearchStatusHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status, err := newWorldMonitorResearchStatusService(pool).Status(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("world monitor status: %v", err), http.StatusInternalServerError)
			return
		}
		jsonOK(w, status)
	}
}

func worldMonitorOpportunityPromoteHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		result, err := newWorldMonitorOpportunityPromoteService(pool).PromotePending(r.Context(), 10)
		if err != nil {
			http.Error(w, fmt.Sprintf("promote world monitor opportunities: %v", err), http.StatusInternalServerError)
			return
		}
		jsonOK(w, result)
	}
}
