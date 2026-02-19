package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type fakeTradeStore struct {
	list []domain.TradeRecord
	get  map[string]domain.TradeRecord
}

func (s *fakeTradeStore) SaveTrade(context.Context, domain.TradeSetup, domain.RiskResult, *domain.Event) error {
	return nil
}

func (s *fakeTradeStore) GetTrade(_ context.Context, id string) (domain.TradeRecord, error) {
	return s.get[id], nil
}

func (s *fakeTradeStore) ListTrades(_ context.Context, _ string, _ string, _ int) ([]domain.TradeRecord, error) {
	return s.list, nil
}

func TestTradesHandler_ListTrades_HappyPath(t *testing.T) {
	store := &fakeTradeStore{
		list: []domain.TradeRecord{
			{
				Setup: domain.TradeSetup{
					ID:         "t1",
					Symbol:     "AAPL",
					Direction:  domain.Long,
					Entry:      100,
					Stop:       95,
					Targets:    []float64{110},
					StrategyID: "earnings_gap_v1",
				},
				Risk: domain.RiskResult{PositionSize: 60, TotalRisk: 300},
			},
		},
	}

	srv := NewServer()
	srv.RegisterTrades(store)

	req := httptest.NewRequest(http.MethodGet, "/trades", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Trades []domain.TradeRecord `json:"trades"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Trades) != 1 || body.Trades[0].Setup.ID != "t1" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestTradesHandler_GetTrade_HappyPath(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	store := &fakeTradeStore{
		get: map[string]domain.TradeRecord{
			"t1": {
				Setup: domain.TradeSetup{ID: "t1", Symbol: "AAPL"},
				Risk:  domain.RiskResult{PositionSize: 1},
				Event: &domain.Event{ID: "e1", Symbol: "AAPL", Time: now},
			},
		},
	}

	srv := NewServer()
	srv.RegisterTrades(store)

	req := httptest.NewRequest(http.MethodGet, "/trades/t1", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got domain.TradeRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Setup.ID != "t1" || got.Event == nil || got.Event.ID != "e1" {
		t.Fatalf("unexpected: %#v", got)
	}
}
