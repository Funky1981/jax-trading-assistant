package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"jax-trading-assistant/internal/modules/instruments"
)

func TestExecuteTrade_ETFPolicyBlocksUnsafeSubmissionsBeforeBroker(t *testing.T) {
	catalog, err := instruments.LoadCatalog("../../../config/etf-instruments.json")
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	checkedAt := time.Date(2026, 5, 13, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		mode       string
		signal     Signal
		quote      *instruments.QuoteSnapshot
		wantReason string
	}{
		{
			name: "excluded ETF class",
			mode: "paper",
			signal: Signal{
				Symbol:     "TQQQ",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "TQQQ",
				Bid:       100,
				Ask:       100.05,
				BidSize:   10,
				AskSize:   10,
				Timestamp: checkedAt,
			},
			wantReason: instruments.ReasonExcludedClass,
		},
		{
			name: "live mode",
			mode: "live",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       100,
				Ask:       100.05,
				BidSize:   10,
				AskSize:   10,
				Timestamp: checkedAt,
			},
			wantReason: instruments.ReasonModeNotAllowed,
		},
		{
			name: "stale quote",
			mode: "paper",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       100,
				Ask:       100.05,
				BidSize:   10,
				AskSize:   10,
				Timestamp: checkedAt.Add(-61 * time.Second),
			},
			wantReason: instruments.ReasonQuoteStale,
		},
		{
			name: "missing bid ask",
			mode: "paper",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       0,
				Ask:       100.05,
				BidSize:   0,
				AskSize:   10,
				Timestamp: checkedAt,
			},
			wantReason: instruments.ReasonBidAskMissing,
		},
		{
			name: "wide spread",
			mode: "paper",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       100,
				Ask:       100.20,
				BidSize:   10,
				AskSize:   10,
				Timestamp: checkedAt,
			},
			wantReason: instruments.ReasonSpreadTooWide,
		},
		{
			name: "outside RTH",
			mode: "paper",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   95,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       100,
				Ask:       100.05,
				BidSize:   10,
				AskSize:   10,
				Timestamp: time.Date(2026, 5, 13, 21, 0, 0, 0, time.UTC),
			},
			wantReason: instruments.ReasonOutsideSession,
		},
		{
			name: "missing stop loss",
			mode: "paper",
			signal: Signal{
				Symbol:     "SPY",
				SignalType: "BUY",
				EntryPrice: 100,
				StopLoss:   0,
				TakeProfit: 110,
				StrategyID: "test",
			},
			quote: &instruments.QuoteSnapshot{
				Symbol:    "SPY",
				Bid:       100,
				Ask:       100.05,
				BidSize:   10,
				AskSize:   10,
				Timestamp: checkedAt,
			},
			wantReason: instruments.ReasonStopLossRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signalID := uuid.New()
			tt.signal.ID = signalID
			store := &fakeStore{signal: &tt.signal, quote: tt.quote}
			broker := &fakeBroker{}
			service := NewService(NewEngine(RiskParameters{}), broker, store, "LMT", RiskParameters{}, nil).
				WithInstrumentPolicy(catalog, tt.mode, func() time.Time {
					if tt.name == "outside RTH" {
						return time.Date(2026, 5, 13, 21, 0, 0, 0, time.UTC)
					}
					return checkedAt
				})

			_, err := service.ExecuteTrade(context.Background(), signalID, "tester")
			if err == nil {
				t.Fatal("expected ETF policy error")
			}
			if !strings.Contains(err.Error(), tt.wantReason) {
				t.Fatalf("expected %s in error, got %v", tt.wantReason, err)
			}
			if broker.placeCalls != 0 {
				t.Fatalf("expected broker not called, got %d", broker.placeCalls)
			}
		})
	}
}

func TestExecuteTrade_ETFPolicyAllowsSafePaperSubmission(t *testing.T) {
	signalID := uuid.New()
	store := &fakeStore{
		signal: &Signal{
			ID:         signalID,
			Symbol:     "SPY",
			SignalType: "BUY",
			EntryPrice: 100,
			StopLoss:   95,
			TakeProfit: 110,
			StrategyID: "test",
		},
		quote: &instruments.QuoteSnapshot{
			Symbol:    "SPY",
			Bid:       100,
			Ask:       100.05,
			BidSize:   10,
			AskSize:   10,
			Timestamp: time.Date(2026, 5, 13, 15, 0, 0, 0, time.UTC),
		},
	}
	broker := &fakeBroker{}
	catalog, err := instruments.LoadCatalog("../../../config/etf-instruments.json")
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	service := NewService(NewEngine(RiskParameters{}), broker, store, "LMT", RiskParameters{}, nil).
		WithInstrumentPolicy(catalog, "paper", func() time.Time {
			return time.Date(2026, 5, 13, 15, 0, 0, 0, time.UTC)
		})

	trade, err := service.ExecuteTrade(context.Background(), signalID, "tester")
	if err != nil {
		t.Fatalf("expected safe ETF submission to pass, got %v", err)
	}
	if broker.placeCalls != 1 {
		t.Fatalf("expected broker called once, got %d", broker.placeCalls)
	}
	if trade.ETFPolicy == nil || trade.ETFPolicy["reasonCode"] != instruments.ReasonAllowed {
		t.Fatalf("expected ETF policy evidence on trade, got %#v", trade.ETFPolicy)
	}
}
