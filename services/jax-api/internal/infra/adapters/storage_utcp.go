package adapters

import (
	"context"
	"fmt"

	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-api/internal/app"
	"jax-trading-assistant/services/jax-api/internal/domain"
)

type UTCPStorageAdapter struct {
	storage *utcp.StorageService
}

func NewUTCPStorageAdapter(storage *utcp.StorageService) *UTCPStorageAdapter {
	return &UTCPStorageAdapter{storage: storage}
}

var _ app.TradeStore = (*UTCPStorageAdapter)(nil)
var _ app.Storage = (*UTCPStorageAdapter)(nil)

func (a *UTCPStorageAdapter) SaveTrade(ctx context.Context, setup domain.TradeSetup, risk domain.RiskResult, event *domain.Event) error {
	in := utcp.SaveTradeInput{
		Trade: utcp.StoredTrade{
			ID:         setup.ID,
			Symbol:     setup.Symbol,
			Direction:  string(setup.Direction),
			Entry:      setup.Entry,
			Stop:       setup.Stop,
			Targets:    setup.Targets,
			EventID:    setup.EventID,
			StrategyID: setup.StrategyID,
			Notes:      setup.Notes,
		},
		Risk: &utcp.StoredRisk{
			PositionSize: risk.PositionSize,
			RiskPerUnit:  risk.RiskPerUnit,
			TotalRisk:    risk.TotalRisk,
			RMultiple:    risk.RMultiple,
		},
	}
	if event != nil {
		in.Event = &utcp.StoredEvent{
			ID:      event.ID,
			Symbol:  event.Symbol,
			Type:    string(event.Type),
			Time:    event.Time,
			Payload: event.Payload,
		}
	}
	return a.storage.SaveTrade(ctx, in)
}

func (a *UTCPStorageAdapter) GetTrade(ctx context.Context, id string) (domain.TradeRecord, error) {
	out, err := a.storage.GetTrade(ctx, id)
	if err != nil {
		return domain.TradeRecord{}, err
	}
	return mapTradeRecord(out)
}

func (a *UTCPStorageAdapter) ListTrades(ctx context.Context, symbol string, strategyID string, limit int) ([]domain.TradeRecord, error) {
	out, err := a.storage.ListTrades(ctx, utcp.ListTradesInput{
		Symbol:     symbol,
		StrategyID: strategyID,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	records := make([]domain.TradeRecord, 0, len(out.Trades))
	for _, t := range out.Trades {
		rec, err := mapTradeRecord(t)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func mapTradeRecord(out utcp.GetTradeOutput) (domain.TradeRecord, error) {
	setup := domain.TradeSetup{
		ID:         out.Trade.ID,
		Symbol:     out.Trade.Symbol,
		Direction:  domain.TradeDirection(out.Trade.Direction),
		Entry:      out.Trade.Entry,
		Stop:       out.Trade.Stop,
		Targets:    out.Trade.Targets,
		EventID:    out.Trade.EventID,
		StrategyID: out.Trade.StrategyID,
		Notes:      out.Trade.Notes,
	}

	var risk domain.RiskResult
	if out.Risk != nil {
		risk = domain.RiskResult{
			PositionSize: out.Risk.PositionSize,
			RiskPerUnit:  out.Risk.RiskPerUnit,
			TotalRisk:    out.Risk.TotalRisk,
			RMultiple:    out.Risk.RMultiple,
		}
	}

	var event *domain.Event
	if out.Event != nil {
		event = &domain.Event{
			ID:      out.Event.ID,
			Symbol:  out.Event.Symbol,
			Type:    domain.EventType(out.Event.Type),
			Time:    out.Event.Time,
			Payload: out.Event.Payload,
		}
	}

	if setup.ID == "" {
		return domain.TradeRecord{}, fmt.Errorf("storage returned trade with empty id")
	}

	return domain.TradeRecord{
		Setup:     setup,
		Risk:      risk,
		Event:     event,
		CreatedAt: out.Trade.CreatedAt,
	}, nil
}
