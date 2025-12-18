package utcp

import (
	"context"
	"testing"
	"time"
)

func TestBacktestTools_RunAndGet(t *testing.T) {
	registry := NewLocalRegistry()
	engine := NewBacktestEngine()
	if err := RegisterBacktestTools(registry, engine); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: BacktestProviderID, Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewBacktestService(client)
	run, err := svc.RunStrategy(context.Background(), RunStrategyInput{
		StrategyConfigID: "earnings_gap_v1",
		Symbols:          []string{"AAPL", "MSFT"},
		From:             time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:               time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run strategy: %v", err)
	}
	if run.RunID == "" {
		t.Fatalf("expected runID")
	}
	if run.Stats.Trades == 0 {
		t.Fatalf("expected trades > 0")
	}

	got, err := svc.GetRun(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.RunID != run.RunID {
		t.Fatalf("expected %q, got %q", run.RunID, got.RunID)
	}
	if len(got.BySymbol) != 2 {
		t.Fatalf("expected 2 bySymbol entries, got %d", len(got.BySymbol))
	}
}

func TestBacktestTools_GetUnknownRunIsError(t *testing.T) {
	registry := NewLocalRegistry()
	engine := NewBacktestEngine()
	if err := RegisterBacktestTools(registry, engine); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: BacktestProviderID, Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewBacktestService(client)
	_, err = svc.GetRun(context.Background(), "bt_missing")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
