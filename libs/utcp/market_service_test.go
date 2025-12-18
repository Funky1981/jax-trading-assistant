package utcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMarketDataService_GetQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Tool  string `json:"tool"`
			Input struct {
				Symbol string `json:"symbol"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Tool != ToolMarketGetQuote {
			t.Fatalf("expected tool %q, got %q", ToolMarketGetQuote, req.Tool)
		}
		if req.Input.Symbol != "AAPL" {
			t.Fatalf("expected symbol AAPL, got %q", req.Input.Symbol)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"symbol":    "AAPL",
				"price":     182.10,
				"currency":  "USD",
				"timestamp": time.Date(2025, 2, 1, 14, 30, 0, 0, time.UTC),
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: MarketDataProviderID, Transport: "http", Endpoint: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewMarketDataService(client)
	out, err := svc.GetQuote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("get quote: %v", err)
	}
	if out.Symbol != "AAPL" || out.Price != 182.10 || out.Currency != "USD" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if out.Timestamp.IsZero() {
		t.Fatalf("expected timestamp, got zero")
	}
}
