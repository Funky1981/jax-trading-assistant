package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type worldMonitorResearchIngestService interface {
	Ingest(ctx context.Context, trigger worldMonitorResearchTrigger) (worldMonitorResearchReceipt, error)
}

type worldMonitorOpportunityPromoteService interface {
	PromotePending(ctx context.Context, limit int) (worldMonitorPromotionResult, error)
}

var newWorldMonitorResearchIngestService = func(pool *pgxpool.Pool) worldMonitorResearchIngestService {
	return newWorldMonitorResearchInboxService(pool)
}

var newWorldMonitorOpportunityPromoteService = func(pool *pgxpool.Pool) worldMonitorOpportunityPromoteService {
	return newWorldMonitorOpportunityPromoter(pool)
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
