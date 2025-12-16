package utcp

import "context"

type StorageService struct {
	client Client
}

func NewStorageService(c Client) *StorageService {
	return &StorageService{client: c}
}

func (s *StorageService) SaveEvent(ctx context.Context, e StoredEvent) error {
	var out SaveEventOutput
	if err := s.client.CallTool(ctx, StorageProviderID, ToolStorageSaveEvent, SaveEventInput{Event: e}, &out); err != nil {
		return err
	}
	return nil
}

func (s *StorageService) SaveTrade(ctx context.Context, in SaveTradeInput) error {
	var out SaveTradeOutput
	if err := s.client.CallTool(ctx, StorageProviderID, ToolStorageSaveTrade, in, &out); err != nil {
		return err
	}
	return nil
}

func (s *StorageService) GetTrade(ctx context.Context, id string) (GetTradeOutput, error) {
	var out GetTradeOutput
	if err := s.client.CallTool(ctx, StorageProviderID, ToolStorageGetTrade, GetTradeInput{ID: id}, &out); err != nil {
		return GetTradeOutput{}, err
	}
	return out, nil
}

func (s *StorageService) ListTrades(ctx context.Context, in ListTradesInput) (ListTradesOutput, error) {
	var out ListTradesOutput
	if err := s.client.CallTool(ctx, StorageProviderID, ToolStorageListTrades, in, &out); err != nil {
		return ListTradesOutput{}, err
	}
	return out, nil
}
