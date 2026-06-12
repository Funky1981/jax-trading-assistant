package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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
