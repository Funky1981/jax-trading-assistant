package services

import (
	"context"
	"jax-trading-assistant/libs/contracts/domain"
)

// TradeExecutor executes trading signals as orders
type TradeExecutor interface {
	// ExecuteSignal converts a signal into one or more orders
	ExecuteSignal(ctx context.Context, signal domain.Signal) ([]domain.Order, error)

	// GetPositions returns current positions
	GetPositions(ctx context.Context, accountID string) ([]domain.Position, error)

	// GetPortfolio returns full portfolio snapshot
	GetPortfolio(ctx context.Context, accountID string) (*domain.Portfolio, error)

	// Health checks if the service is healthy
	Health(ctx context.Context) error
}
