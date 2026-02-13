package services

import (
	"context"
	"jax-trading-assistant/libs/contracts/domain"
)

// Orchestrator coordinates AI providers for trading analysis
type Orchestrator interface {
	// RunOrchestration executes an orchestration query
	RunOrchestration(ctx context.Context, userID, query string) (*domain.OrchestrationRun, error)

	// GetRunHistory returns recent orchestration runs
	GetRunHistory(ctx context.Context, userID string, limit int) ([]domain.OrchestrationRun, error)

	// Health checks if the service is healthy
	Health(ctx context.Context) error
}
