package main

import "testing"

func TestNewMarketToolsPrefersIBBridgeForFrontendMarketData(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "alpaca-key")
	t.Setenv("ALPACA_API_SECRET", "alpaca-secret")
	t.Setenv("POLYGON_API_KEY", "polygon-key")
	t.Setenv("FINANCIAL_DATASETS_API_KEY", "fd-key")

	mt := newMarketTools(nil, "http://ib-bridge:8092")
	if mt == nil || mt.mdClient == nil {
		t.Fatal("expected market data client to be initialized")
	}

	providers := mt.mdClient.ProviderNames()
	if len(providers) != 4 {
		t.Fatalf("expected 4 providers, got %d", len(providers))
	}
	if providers[0] != "ib-bridge" {
		t.Fatalf("expected ib-bridge first, got %q", providers[0])
	}
}
