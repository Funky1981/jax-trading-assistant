package app

import (
	"context"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type TradeStore interface {
	SaveTrade(ctx context.Context, setup domain.TradeSetup, risk domain.RiskResult, event *domain.Event) error
	GetTrade(ctx context.Context, id string) (domain.TradeRecord, error)
	ListTrades(ctx context.Context, symbol string, strategyID string, limit int) ([]domain.TradeRecord, error)
}
