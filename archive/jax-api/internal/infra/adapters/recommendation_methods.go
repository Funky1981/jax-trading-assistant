package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"jax-trading-assistant/services/jax-api/internal/app"

	"github.com/google/uuid"
)

// GetRecommendations returns signals with orchestration analysis (pending recommendations)
func (s *PostgresSignalStore) GetRecommendations(ctx context.Context, limit, offset int) (*app.RecommendationListResponse, error) {
	// Query for signals with orchestration analysis that are pending
	query := `
		SELECT 
			s.id, s.symbol, s.strategy_id, s.signal_type, s.confidence, 
			s.entry_price, s.stop_loss, s.take_profit, s.reasoning,
			s.generated_at, s.expires_at, s.status, s.orchestration_run_id, s.created_at,
			o.id, o.symbol, o.trigger_type, o.trigger_id, o.agent_suggestion, o.confidence, o.reasoning,
			o.memories_recalled, o.status, o.started_at, o.completed_at, o.error
		FROM strategy_signals s
		LEFT JOIN orchestration_runs o ON s.orchestration_run_id = o.id
		WHERE s.status = 'pending' AND s.orchestration_run_id IS NOT NULL
		ORDER BY s.generated_at DESC
		LIMIT $1 OFFSET $2`

	countQuery := `
		SELECT COUNT(*) 
		FROM strategy_signals 
		WHERE status = 'pending' AND orchestration_run_id IS NOT NULL`

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count recommendations: %w", err)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query recommendations: %w", err)
	}
	defer rows.Close()

	recommendations := []app.Recommendation{}
	for rows.Next() {
		var rec app.Recommendation
		var orch app.OrchestrationRun

		err := rows.Scan(
			// Signal fields
			&rec.Signal.ID, &rec.Signal.Symbol, &rec.Signal.StrategyID, &rec.Signal.SignalType, &rec.Signal.Confidence,
			&rec.Signal.EntryPrice, &rec.Signal.StopLoss, &rec.Signal.TakeProfit, &rec.Signal.Reasoning,
			&rec.Signal.GeneratedAt, &rec.Signal.ExpiresAt, &rec.Signal.Status, &rec.Signal.OrchestrationRunID, &rec.Signal.CreatedAt,
			// Orchestration fields
			&orch.ID, &orch.Symbol, &orch.TriggerType, &orch.TriggerID, &orch.AgentSuggestion, &orch.Confidence, &orch.Reasoning,
			&orch.MemoriesRecalled, &orch.Status, &orch.StartedAt, &orch.CompletedAt, &orch.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation: %w", err)
		}

		rec.Analysis = &orch
		recommendations = append(recommendations, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recommendations: %w", err)
	}

	return &app.RecommendationListResponse{
		Recommendations: recommendations,
		Total:           total,
		Limit:           limit,
		Offset:          offset,
	}, nil
}

// GetRecommendation returns a single recommendation with full details
func (s *PostgresSignalStore) GetRecommendation(ctx context.Context, signalID uuid.UUID) (*app.Recommendation, error) {
	query := `
		SELECT 
			s.id, s.symbol, s.strategy_id, s.signal_type, s.confidence, 
			s.entry_price, s.stop_loss, s.take_profit, s.reasoning,
			s.generated_at, s.expires_at, s.status, s.orchestration_run_id, s.created_at,
			o.id, o.symbol, o.trigger_type, o.trigger_id, o.agent_suggestion, o.confidence, o.reasoning,
			o.memories_recalled, o.status, o.started_at, o.completed_at, o.error
		FROM strategy_signals s
		LEFT JOIN orchestration_runs o ON s.orchestration_run_id = o.id
		WHERE s.id = $1`

	var rec app.Recommendation
	var orch app.OrchestrationRun
	var orchID sql.NullString

	err := s.db.QueryRowContext(ctx, query, signalID).Scan(
		// Signal fields
		&rec.Signal.ID, &rec.Signal.Symbol, &rec.Signal.StrategyID, &rec.Signal.SignalType, &rec.Signal.Confidence,
		&rec.Signal.EntryPrice, &rec.Signal.StopLoss, &rec.Signal.TakeProfit, &rec.Signal.Reasoning,
		&rec.Signal.GeneratedAt, &rec.Signal.ExpiresAt, &rec.Signal.Status, &rec.Signal.OrchestrationRunID, &rec.Signal.CreatedAt,
		// Orchestration fields
		&orchID, &orch.Symbol, &orch.TriggerType, &orch.TriggerID, &orch.AgentSuggestion, &orch.Confidence, &orch.Reasoning,
		&orch.MemoriesRecalled, &orch.Status, &orch.StartedAt, &orch.CompletedAt, &orch.Error,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("recommendation not found: %s", signalID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendation: %w", err)
	}

	// Only include orchestration data if it exists
	if orchID.Valid {
		orch.ID = uuid.MustParse(orchID.String)
		rec.Analysis = &orch
	}

	return &rec, nil
}
