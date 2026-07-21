package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	Items     []worldMonitorResearchInboxItem  `json:"items"`
	Total     int                              `json:"total"`
	Counts    worldMonitorResearchStatusCounts `json:"counts"`
	CheckedAt time.Time                        `json:"checkedAt"`
}

type worldMonitorResearchInboxItem struct {
	ID                   string         `json:"id"`
	Source               string         `json:"source"`
	SourceEventID        string         `json:"sourceEventId"`
	WorldMonitorEventID  string         `json:"worldMonitorEventId"`
	Status               string         `json:"status"`
	RejectionReason      string         `json:"rejectionReason,omitempty"`
	EventType            string         `json:"eventType"`
	Headline             string         `json:"headline"`
	Summary              string         `json:"summary,omitempty"`
	SourceURLs           []string       `json:"sourceUrls"`
	SourceCount          int            `json:"sourceCount"`
	EventTime            time.Time      `json:"eventTime"`
	ReceivedAt           time.Time      `json:"receivedAt"`
	CollectedAt          *time.Time     `json:"collectedAt,omitempty"`
	RawEventID           string         `json:"rawEventId,omitempty"`
	IsSynthetic          bool           `json:"isSynthetic"`
	SyntheticReason      string         `json:"syntheticReason,omitempty"`
	DiscoveryMethod      string         `json:"discoveryMethod,omitempty"`
	AnalysisIdentity     string         `json:"analysisIdentity,omitempty"`
	AIProvider           string         `json:"aiProvider,omitempty"`
	AIModel              string         `json:"aiModel,omitempty"`
	Region               string         `json:"region,omitempty"`
	PossibleAffectedETFs []string       `json:"possibleAffectedEtfs"`
	AssetThemes          []string       `json:"assetThemes"`
	Severity             string         `json:"severity"`
	SourceTier           string         `json:"sourceTier"`
	Confidence           float64        `json:"confidence"`
	ConfidenceReasons    []string       `json:"confidenceReasons"`
	MappingReason        string         `json:"mappingReason"`
	NormalizedEventID    string         `json:"normalizedEventId,omitempty"`
	CandidateID          string         `json:"candidateId,omitempty"`
	OperatorDecision     string         `json:"operatorDecision,omitempty"`
	OperatorReason       string         `json:"operatorReason,omitempty"`
	RawPayload           map[string]any `json:"rawPayload"`
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

	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'new')::int,
			COUNT(*) FILTER (WHERE status = 'candidate_created')::int,
			COUNT(*) FILTER (WHERE status = 'rejected')::int,
			COUNT(*) FILTER (WHERE status = 'ignored')::int
		FROM world_monitor_research_inbox
	`).Scan(
		&out.Counts.Total,
		&out.Counts.Pending,
		&out.Counts.CandidatesCreated,
		&out.Counts.Rejected,
		&out.Counts.Ignored,
	)
	if err != nil {
		return worldMonitorResearchInboxList{}, fmt.Errorf("world monitor inbox counts: %w", err)
	}
	out.Total = out.Counts.Total

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
			COALESCE(w.source_urls, '[]'::jsonb), w.source_count, w.event_time, w.received_at,
			COALESCE(w.region, ''), COALESCE(w.possible_affected_etfs, '[]'::jsonb), COALESCE(w.asset_themes, '[]'::jsonb),
			w.severity, w.source_tier, w.confidence, COALESCE(w.confidence_reasons, '[]'::jsonb), w.mapping_reason,
			COALESCE(w.normalized_event_id::text, ''), COALESCE(w.candidate_id::text, ''),
			COALESCE(w.operator_decision, ''), COALESCE(w.operator_reason, ''),
			COALESCE(w.raw_payload, '{}'::jsonb),
			NULLIF(COALESCE(er.payload->>'collection_timestamp_utc', er.payload->>'collected_at', er.payload->>'collectedAt', ''), '')::timestamptz,
			COALESCE(er.id::text, ''), COALESCE(er.is_synthetic, false), COALESCE(er.synthetic_reason, ''),
			COALESCE(er.payload->>'discovery_method', er.payload->>'discoveryMethod', ''),
			COALESCE(er.payload->>'deterministic_analysis', er.payload->>'deterministic_analysis_identity', er.payload->>'analysis_identity',
				CASE WHEN COALESCE(er.payload->>'analysis_provider','')='' THEN er.payload->>'analysis_model' ELSE '' END, ''),
			COALESCE(er.payload->>'analysis_provider', er.payload->>'ai_provider', ''),
			CASE WHEN COALESCE(er.payload->>'analysis_provider', er.payload->>'ai_provider', '')<>'' THEN COALESCE(er.payload->>'analysis_model', er.payload->>'ai_model', '') ELSE '' END
		FROM world_monitor_research_inbox w
		LEFT JOIN event_normalized en ON en.id=w.normalized_event_id
		LEFT JOIN event_raw er ON er.id=en.raw_event_id
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
		var sourceURLsRaw, etfsRaw, themesRaw, confidenceReasonsRaw, rawPayload []byte
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
			&item.IsSynthetic,
			&item.SyntheticReason,
			&item.DiscoveryMethod,
			&item.AnalysisIdentity,
			&item.AIProvider,
			&item.AIModel,
		); err != nil {
			return worldMonitorResearchInboxList{}, err
		}
		_ = json.Unmarshal(sourceURLsRaw, &item.SourceURLs)
		_ = json.Unmarshal(etfsRaw, &item.PossibleAffectedETFs)
		_ = json.Unmarshal(themesRaw, &item.AssetThemes)
		_ = json.Unmarshal(confidenceReasonsRaw, &item.ConfidenceReasons)
		_ = json.Unmarshal(rawPayload, &item.RawPayload)
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
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return worldMonitorResearchInboxList{}, err
	}
	return out, nil
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
