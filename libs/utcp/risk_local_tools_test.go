package utcp

import (
	"context"
	"testing"
)

func TestRiskTools_PositionSize_HappyPath(t *testing.T) {
	registry := NewLocalRegistry()
	if err := RegisterRiskTools(registry); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: RiskProviderID, Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	risk := NewRiskService(client)
	out, err := risk.PositionSize(context.Background(), PositionSizeInput{
		AccountSize: 10000,
		RiskPercent: 3,
		Entry:       100,
		Stop:        95,
	})
	if err != nil {
		t.Fatalf("position size: %v", err)
	}
	if out.PositionSize != 60 {
		t.Fatalf("expected positionSize=60, got %d", out.PositionSize)
	}
	if out.RiskPerUnit != 5 {
		t.Fatalf("expected riskPerUnit=5, got %v", out.RiskPerUnit)
	}
	if out.TotalRisk != 300 {
		t.Fatalf("expected totalRisk=300, got %v", out.TotalRisk)
	}
}

func TestRiskTools_RMultiple_HappyPath(t *testing.T) {
	registry := NewLocalRegistry()
	if err := RegisterRiskTools(registry); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: RiskProviderID, Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	risk := NewRiskService(client)
	out, err := risk.RMultiple(context.Background(), RMultipleInput{
		Entry:  100,
		Stop:   95,
		Target: 110,
	})
	if err != nil {
		t.Fatalf("r multiple: %v", err)
	}
	if out.RMultiple != 2.0 {
		t.Fatalf("expected rMultiple=2.0, got %v", out.RMultiple)
	}
}

func TestRiskTools_StopEqualsEntry_IsError(t *testing.T) {
	registry := NewLocalRegistry()
	if err := RegisterRiskTools(registry); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: RiskProviderID, Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	risk := NewRiskService(client)
	_, err = risk.PositionSize(context.Background(), PositionSizeInput{
		AccountSize: 10000,
		RiskPercent: 3,
		Entry:       100,
		Stop:        100,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
