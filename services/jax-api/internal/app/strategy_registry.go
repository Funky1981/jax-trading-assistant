package app

import "jax-trading-assistant/services/jax-api/internal/domain"

type StrategyRegistry struct {
	Strategies map[string]domain.StrategyConfig
}

func NewStrategyRegistry(strategies map[string]domain.StrategyConfig) *StrategyRegistry {
	if strategies == nil {
		strategies = make(map[string]domain.StrategyConfig)
	}
	return &StrategyRegistry{Strategies: strategies}
}

func (r *StrategyRegistry) List() []domain.StrategyConfig {
	out := make([]domain.StrategyConfig, 0, len(r.Strategies))
	for _, s := range r.Strategies {
		out = append(out, s)
	}
	return out
}
