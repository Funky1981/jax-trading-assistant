package execution

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeBroker struct {
	placeCalls int
}

func (b *fakeBroker) GetAccount(ctx context.Context) (*BrokerAccountInfo, error) {
	return &BrokerAccountInfo{
		NetLiquidation: 100000,
		BuyingPower:    50000,
		Currency:       "USD",
	}, nil
}

func (b *fakeBroker) PlaceOrder(ctx context.Context, order *BrokerOrderRequest) (*BrokerOrderResponse, error) {
	b.placeCalls++
	return &BrokerOrderResponse{OrderID: 123, Success: true}, nil
}

func (b *fakeBroker) GetOrderStatus(ctx context.Context, orderID int) (*BrokerOrderStatus, error) {
	return &BrokerOrderStatus{OrderID: orderID, Status: "Submitted"}, nil
}

func (b *fakeBroker) GetPositions(ctx context.Context) (*BrokerPositionsResponse, error) {
	return &BrokerPositionsResponse{}, nil
}

type fakeStore struct {
	signal                   *Signal
	trade                    *TradeResult
	intent                   *OrderIntentSummary
	storeIntentCalls         int
	updateIntentByTradeCalls int
}

func (s *fakeStore) GetSignal(ctx context.Context, signalID uuid.UUID) (*Signal, error) {
	if s.signal == nil {
		return nil, errors.New("signal missing")
	}
	return s.signal, nil
}

func (s *fakeStore) GetTradeBySignal(ctx context.Context, signalID uuid.UUID) (*TradeResult, error) {
	return s.trade, nil
}

func (s *fakeStore) GetLatestOrderIntentBySignal(ctx context.Context, signalID uuid.UUID) (*OrderIntentSummary, error) {
	return s.intent, nil
}

func (s *fakeStore) StoreTrade(ctx context.Context, trade *TradeResult) error {
	s.trade = trade
	return nil
}

func (s *fakeStore) UpdateTradeApproval(ctx context.Context, signalID uuid.UUID, orderID string) error {
	return nil
}

func (s *fakeStore) UpdateTradeStatus(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error {
	return nil
}

func (s *fakeStore) UpdateOrderIntentStatusByTrade(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error {
	s.updateIntentByTradeCalls++
	return nil
}

func (s *fakeStore) GetDailyRisk(ctx context.Context) (float64, error) {
	return 0, nil
}

func (s *fakeStore) StoreOrderIntent(ctx context.Context, intent *OrderIntent) (string, error) {
	s.storeIntentCalls++
	return "intent-1", nil
}

func (s *fakeStore) UpdateOrderIntentStatus(ctx context.Context, signalID uuid.UUID, status *BrokerOrderStatus) error {
	return nil
}

func TestExecuteTrade_IdempotentExistingTrade(t *testing.T) {
	existingTrade := &TradeResult{
		TradeID:  uuid.New(),
		SignalID: uuid.New(),
		OrderID:  "existing",
		Symbol:   "AAPL",
		Status:   "Submitted",
	}

	store := &fakeStore{trade: existingTrade}
	broker := &fakeBroker{}
	engine := NewEngine(RiskParameters{})
	service := NewService(engine, broker, store, "LMT", RiskParameters{}, nil)

	trade, err := service.ExecuteTrade(context.Background(), existingTrade.SignalID, "tester")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if trade.OrderID != existingTrade.OrderID {
		t.Fatalf("expected existing trade to be returned")
	}
	if broker.placeCalls != 0 {
		t.Fatalf("expected broker not called, got %d", broker.placeCalls)
	}
}

func TestExecuteTrade_IdempotentExistingIntentBlocksBroker(t *testing.T) {
	signalID := uuid.New()
	store := &fakeStore{
		signal: &Signal{
			ID:         signalID,
			Symbol:     "AAPL",
			SignalType: "BUY",
			EntryPrice: 100,
			StopLoss:   95,
			TakeProfit: 110,
			StrategyID: "test",
		},
		intent: &OrderIntentSummary{
			ID:            "intent-1",
			Status:        "Submitted",
			BrokerOrderID: "123",
		},
	}
	broker := &fakeBroker{}
	engine := NewEngine(RiskParameters{})
	service := NewService(engine, broker, store, "LMT", RiskParameters{}, nil)

	_, err := service.ExecuteTrade(context.Background(), signalID, "tester")
	if err == nil {
		t.Fatalf("expected error due to existing intent")
	}
	if broker.placeCalls != 0 {
		t.Fatalf("expected broker not called, got %d", broker.placeCalls)
	}
}
