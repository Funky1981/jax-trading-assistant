package services

import (
	"context"
	"jax-trading-assistant/libs/contracts/domain"
)

// SignalGenerator analyzes market data and generates trading signals
type SignalGenerator interface {
	// GenerateSignals analyzes multiple symbols and returns signals
	GenerateSignals(ctx context.Context, symbols []string) ([]domain.Signal, error)

	// GetSignalHistory returns recent signals
	GetSignalHistory(ctx context.Context, symbol string, limit int) ([]domain.Signal, error)

	// Health checks if the service is healthy
	Health(ctx context.Context) error
}
