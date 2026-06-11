package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/instruments"
)

type aiSuggestionPromoteRequest struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
	Risk       string  `json:"risk"`
	Source     string  `json:"source"`
}

type aiSuggestionPromoteResponse struct {
	CandidateID string `json:"candidateId"`
	SignalID    string `json:"signalId"`
	Route       string `json:"route"`
	Status      string `json:"status"`
}

func aiSuggestionPromoteHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req aiSuggestionPromoteRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		response, err := promoteAISuggestion(r.Context(), pool, req)
		if err != nil {
			switch {
			case errors.Is(err, errAISuggestionPromotionInput):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, candidatesmod.ErrDuplicateCandidate):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				http.Error(w, fmt.Sprintf("promote ai suggestion: %v", err), http.StatusInternalServerError)
			}
			return
		}
		jsonOK(w, response)
	}
}

var errAISuggestionPromotionInput = errors.New("invalid ai suggestion promotion input")

func promoteAISuggestion(ctx context.Context, pool *pgxpool.Pool, req aiSuggestionPromoteRequest) (aiSuggestionPromoteResponse, error) {
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if symbol == "" {
		return aiSuggestionPromoteResponse{}, fmt.Errorf("%w: symbol is required", errAISuggestionPromotionInput)
	}
	if action != "BUY" && action != "SELL" {
		return aiSuggestionPromoteResponse{}, fmt.Errorf("%w: action must be BUY or SELL", errAISuggestionPromotionInput)
	}
	confidence := req.Confidence
	if confidence > 1 {
		confidence = confidence / 100
	}
	if confidence <= 0 || confidence > 1 {
		return aiSuggestionPromoteResponse{}, fmt.Errorf("%w: confidence must be between 0 and 1", errAISuggestionPromotionInput)
	}
	if pool == nil {
		return aiSuggestionPromoteResponse{}, fmt.Errorf("database pool is required")
	}

	catalog, _ := instruments.LoadDefaultCatalog()
	isETF := catalog != nil && catalog.IsKnownETF(symbol)
	instanceID, strategyID, err := findAISuggestionStrategyInstance(ctx, pool, symbol, isETF)
	if err != nil {
		return aiSuggestionPromoteResponse{}, err
	}

	entry, err := newWorldMonitorOpportunityPromoter(pool).latestEntryPrice(ctx, symbol)
	if err != nil {
		return aiSuggestionPromoteResponse{}, err
	}
	stop, target := aiSuggestionRiskPrices(action, entry)
	reasoning := strings.TrimSpace(req.Reasoning)
	if reasoning == "" {
		reasoning = fmt.Sprintf("AI suggestion promoted by operator for %s %s.", action, symbol)
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "agent0_manual_review"
	}
	expiresAt := time.Now().UTC().Add(worldMonitorCandidateTTL)

	signalID, err := createAISuggestionSignal(ctx, pool, instanceID, strategyID, symbol, action, confidence, entry, stop, target, reasoning, expiresAt)
	if err != nil {
		return aiSuggestionPromoteResponse{}, err
	}

	candidateSvc := candidatesmod.NewService(candidatesmod.NewStore(pool))
	candidate, err := candidateSvc.Propose(ctx, candidatesmod.ProposalRequest{
		StrategyInstanceID: instanceID,
		SignalID:           signalID.String(),
		StrategyID:         strategyID,
		Symbol:             symbol,
		SignalType:         action,
		EntryPrice:         &entry,
		StopLoss:           &stop,
		TakeProfit:         &target,
		Confidence:         &confidence,
		Reasoning:          &reasoning,
		DataProvenance:     source,
		TTL:                worldMonitorCandidateTTL,
	})
	if err != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM strategy_signals WHERE id = $1`, signalID)
		return aiSuggestionPromoteResponse{}, err
	}

	status := candidatesmod.StatusDetected
	route := "manual_allowed"
	if isETF {
		if err := candidateSvc.Qualify(ctx, candidate.ID); err != nil {
			return aiSuggestionPromoteResponse{}, err
		}
		status = candidatesmod.StatusAwaitingApproval
		route = "approval_required"
	}
	if err := attachAISuggestionCandidateMetadata(ctx, pool, candidate.ID, signalID, req, route); err != nil {
		return aiSuggestionPromoteResponse{}, err
	}

	return aiSuggestionPromoteResponse{
		CandidateID: candidate.ID.String(),
		SignalID:    signalID.String(),
		Route:       route,
		Status:      status,
	}, nil
}

func findAISuggestionStrategyInstance(ctx context.Context, pool *pgxpool.Pool, symbol string, isETF bool) (uuid.UUID, string, error) {
	if isETF {
		return newWorldMonitorOpportunityPromoter(pool).findStrategyInstance(ctx, symbol)
	}
	var id uuid.UUID
	var strategyID string
	err := pool.QueryRow(ctx, `
		SELECT id, COALESCE(NULLIF(strategy_id, ''), strategy_type_id)
		FROM strategy_instances
		WHERE name = 'legacy-default'
		LIMIT 1
	`).Scan(&id, &strategyID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("find AI suggestion strategy instance: %w", err)
	}
	return id, strategyID, nil
}

func aiSuggestionRiskPrices(action string, entry float64) (float64, float64) {
	if action == "SELL" {
		return roundPrice(entry * 1.02), roundPrice(entry * 0.96)
	}
	return roundPrice(entry * 0.98), roundPrice(entry * 1.04)
}

func createAISuggestionSignal(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID, strategyID, symbol, action string, confidence, entry, stop, target float64, reasoning string, expiresAt time.Time) (uuid.UUID, error) {
	signalID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_signals (
			id, symbol, strategy_id, signal_type, confidence, entry_price, stop_loss,
			take_profit, reasoning, generated_at, expires_at, status, instance_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10, 'pending', $11)
	`, signalID, symbol, strategyID, action, confidence, entry, stop, target, reasoning, expiresAt.UTC(), instanceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create AI suggestion signal: %w", err)
	}
	return signalID, nil
}

func attachAISuggestionCandidateMetadata(ctx context.Context, pool *pgxpool.Pool, candidateID, signalID uuid.UUID, req aiSuggestionPromoteRequest, route string) error {
	metadata := map[string]any{
		"aiSuggestion": map[string]any{
			"source":     strings.TrimSpace(req.Source),
			"action":     strings.ToUpper(strings.TrimSpace(req.Action)),
			"risk":       strings.TrimSpace(req.Risk),
			"signalId":   signalID.String(),
			"route":      route,
			"promotedAt": time.Now().UTC().Format(time.RFC3339),
		},
	}
	payload, _ := json.Marshal(metadata)
	_, err := pool.Exec(ctx, `
		UPDATE candidate_trades
		SET metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb,
		    updated_at = NOW()
		WHERE id = $1
	`, candidateID, string(payload))
	if err != nil {
		return fmt.Errorf("attach AI suggestion candidate metadata: %w", err)
	}
	return nil
}
