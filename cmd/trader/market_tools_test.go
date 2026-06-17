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

func TestMarketDataProviderConfigsIncludeAlpacaFallbackForIngester(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "alpaca-key")
	t.Setenv("ALPACA_API_SECRET", "alpaca-secret")
	t.Setenv("POLYGON_API_KEY", "")
	t.Setenv("FINANCIAL_DATASETS_API_KEY", "")

	providers := marketDataProviderConfigs("http://ib-bridge:8092")
	if len(providers) != 2 {
		t.Fatalf("expected ib-bridge plus alpaca fallback, got %d providers", len(providers))
	}
	if providers[0].Name != "ib-bridge" {
		t.Fatalf("expected ib-bridge first, got %q", providers[0].Name)
	}
	if providers[1].Name != "alpaca" {
		t.Fatalf("expected alpaca fallback second, got %q", providers[1].Name)
	}
	if providers[1].Feed != "iex" {
		t.Fatalf("expected alpaca free fallback to use iex feed, got %q", providers[1].Feed)
	}
	if providers[1].Priority <= providers[0].Priority {
		t.Fatalf("expected alpaca priority to be lower than ib-bridge: %+v", providers)
	}
}

func TestMarketDataSourceLabelShowsProviderChain(t *testing.T) {
	got := marketDataSourceLabel([]string{"ib-bridge", "alpaca"})
	if got != "provider-chain: ib-bridge,alpaca" {
		t.Fatalf("source label = %q", got)
	}
}
