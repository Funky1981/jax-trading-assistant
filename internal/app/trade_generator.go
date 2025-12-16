package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"jax-trading-assistant/internal/domain"
)

type TradeGenerator struct {
	market     MarketData
	strategies map[string]domain.StrategyConfig
}

func NewTradeGenerator(market MarketData, strategies map[string]domain.StrategyConfig) *TradeGenerator {
	if strategies == nil {
		strategies = make(map[string]domain.StrategyConfig)
	}
	return &TradeGenerator{market: market, strategies: strategies}
}

func (g *TradeGenerator) GenerateFromEvent(ctx context.Context, e domain.Event) ([]domain.TradeSetup, error) {
	var matched []domain.StrategyConfig
	for _, s := range g.strategies {
		for _, t := range s.EventTypes {
			if t == string(e.Type) {
				matched = append(matched, s)
				break
			}
		}
	}
	if len(matched) == 0 {
		return nil, nil
	}

	candles, err := g.market.GetDailyCandles(ctx, e.Symbol, 2)
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, nil
	}
	cur := candles[len(candles)-1]

	direction := inferDirection(e)
	entry := cur.Open
	if entry == 0 {
		entry = cur.Close
	}
	if entry == 0 {
		return nil, nil
	}

	stop := cur.Low
	if stop == 0 {
		stop = entry * 0.95
	}
	if direction == domain.Short {
		stop = cur.High
		if stop == 0 {
			stop = entry * 1.05
		}
	}

	riskPerUnit := abs(entry - stop)
	if riskPerUnit == 0 {
		return nil, nil
	}

	var setups []domain.TradeSetup
	for _, s := range matched {
		targetMultiples, err := parseTargetRuleRMultiples(s.TargetRule)
		if err != nil {
			return nil, err
		}
		targets := make([]float64, 0, len(targetMultiples))
		for _, m := range targetMultiples {
			if m <= 0 {
				continue
			}
			if direction == domain.Long {
				targets = append(targets, entry+m*riskPerUnit)
			} else {
				targets = append(targets, entry-m*riskPerUnit)
			}
		}

		setups = append(setups, domain.TradeSetup{
			ID:         fmt.Sprintf("ts_%s_%s_%d", s.ID, e.Symbol, e.Time.UTC().Unix()),
			Symbol:     e.Symbol,
			Direction:  direction,
			Entry:      entry,
			Stop:       stop,
			Targets:    targets,
			EventID:    e.ID,
			StrategyID: s.ID,
			Notes:      fmt.Sprintf("generated from event %s", e.Type),
		})
	}

	return setups, nil
}

func inferDirection(e domain.Event) domain.TradeDirection {
	if v, ok := e.Payload["gapPct"]; ok {
		switch n := v.(type) {
		case float64:
			if n < 0 {
				return domain.Short
			}
			return domain.Long
		case int:
			if n < 0 {
				return domain.Short
			}
			return domain.Long
		}
	}
	return domain.Long
}

func parseTargetRuleRMultiples(rule string) ([]float64, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil, nil
	}

	parts := strings.Split(rule, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimSuffix(p, "R")
		p = strings.TrimSuffix(p, "r")
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid targetRule %q", rule)
		}
		out = append(out, v)
	}
	return out, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
