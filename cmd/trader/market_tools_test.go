package main

import (
	"context"
	"testing"

	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/marketdata"
)

type marketToolsRawPayloadStoreProbe struct{}

func (marketToolsRawPayloadStoreProbe) Put(context.Context, providercontract.RawPayloadRef, []byte) (providercontract.RawPayloadLocation, error) {
	return providercontract.RawPayloadLocation{}, nil
}

func (marketToolsRawPayloadStoreProbe) Get(context.Context, providercontract.RawPayloadRef) ([]byte, error) {
	return nil, nil
}

func TestNewMarketToolsPrefersIBBridgeForFrontendMarketData(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "alpaca-key")
	t.Setenv("ALPACA_API_SECRET", "alpaca-secret")
	t.Setenv("POLYGON_API_KEY", "polygon-key")

	mt := newMarketTools(nil, "http://ib-bridge:8092")
	if mt == nil || mt.mdClient == nil {
		t.Fatal("expected market data client to be initialized")
	}

	providers := mt.mdClient.ProviderNames()
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}
	if providers[0] != "ib-bridge" {
		t.Fatalf("expected ib-bridge first, got %q", providers[0])
	}
	if providers[1] != "alpaca" || providers[2] != "polygon" {
		t.Fatalf("unexpected active market provider chain: %v", providers)
	}
}

func TestMarketDataProviderConfigsExcludeRetiredVendorPaths(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "alpaca-key")
	t.Setenv("ALPACA_API_SECRET", "alpaca-secret")
	t.Setenv("POLYGON_API_KEY", "polygon-key")

	providers := marketDataProviderConfigs("")
	if len(providers) != 2 {
		t.Fatalf("expected only Alpaca and canonical Polygon providers, got %d: %v", len(providers), providers)
	}
	for _, provider := range providers {
		if provider.Name != marketdata.ProviderAlpaca && provider.Name != marketdata.ProviderPolygon {
			t.Fatalf("retired provider entered active trader chain: %q", provider.Name)
		}
	}
}

func TestMarketDataProviderConfigsIncludeAlpacaFallbackForIngester(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "alpaca-key")
	t.Setenv("ALPACA_API_SECRET", "alpaca-secret")
	t.Setenv("POLYGON_API_KEY", "")

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

func TestNewMarketToolsCarriesSharedRawPayloadStore(t *testing.T) {
	store := marketToolsRawPayloadStoreProbe{}
	mt := newMarketTools(nil, "", store)
	if mt == nil || mt.rawPayloadStore == nil {
		t.Fatal("expected shared raw payload store to be composed")
	}
}
