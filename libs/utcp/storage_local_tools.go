package utcp

import (
	"context"
	"fmt"
)

func RegisterStorageTools(registry *LocalRegistry, storage *PostgresStorage) error {
	if registry == nil {
		return fmt.Errorf("register storage tools: registry is nil")
	}
	if storage == nil {
		return fmt.Errorf("register storage tools: storage is nil")
	}

	if err := registry.Register(StorageProviderID, ToolStorageSaveEvent, storage.saveEventTool); err != nil {
		return err
	}
	if err := registry.Register(StorageProviderID, ToolStorageSaveTrade, storage.saveTradeTool); err != nil {
		return err
	}
	if err := registry.Register(StorageProviderID, ToolStorageGetTrade, storage.getTradeTool); err != nil {
		return err
	}
	if err := registry.Register(StorageProviderID, ToolStorageListTrades, storage.listTradesTool); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) saveEventTool(ctx context.Context, input any, output any) error {
	var in SaveEventInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("storage.save_event: %w", err)
	}
	if err := s.SaveEvent(ctx, in.Event); err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*SaveEventOutput)
	if !ok {
		return fmt.Errorf("storage.save_event: output must be *utcp.SaveEventOutput")
	}
	*typed = SaveEventOutput{}
	return nil
}

func (s *PostgresStorage) saveTradeTool(ctx context.Context, input any, output any) error {
	var in SaveTradeInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("storage.save_trade: %w", err)
	}
	if err := s.SaveTrade(ctx, in.Trade, in.Risk, in.Event); err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*SaveTradeOutput)
	if !ok {
		return fmt.Errorf("storage.save_trade: output must be *utcp.SaveTradeOutput")
	}
	*typed = SaveTradeOutput{}
	return nil
}

func (s *PostgresStorage) getTradeTool(ctx context.Context, input any, output any) error {
	var in GetTradeInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("storage.get_trade: %w", err)
	}
	out, err := s.GetTrade(ctx, in.ID)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*GetTradeOutput)
	if !ok {
		return fmt.Errorf("storage.get_trade: output must be *utcp.GetTradeOutput")
	}
	*typed = out
	return nil
}

func (s *PostgresStorage) listTradesTool(ctx context.Context, input any, output any) error {
	var in ListTradesInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("storage.list_trades: %w", err)
	}
	out, err := s.ListTrades(ctx, in)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*ListTradesOutput)
	if !ok {
		return fmt.Errorf("storage.list_trades: output must be *utcp.ListTradesOutput")
	}
	*typed = out
	return nil
}
