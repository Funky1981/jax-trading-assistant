package utcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUTCPClient_LocalToolCall(t *testing.T) {
	registry := NewLocalRegistry()
	if err := registry.Register("risk", "risk.position_size", func(ctx context.Context, input any, output any) error {
		out := output.(*struct {
			PositionSize int `json:"positionSize"`
		})
		out.PositionSize = 123
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "risk", Transport: "local"},
		},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out struct {
		PositionSize int `json:"positionSize"`
	}
	if err := client.CallTool(context.Background(), "risk", "risk.position_size", map[string]any{"x": 1}, &out); err != nil {
		t.Fatalf("call: %v", err)
	}
	if out.PositionSize != 123 {
		t.Fatalf("expected 123, got %d", out.PositionSize)
	}
}

func TestUTCPClient_HTTPCall_WrappedOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Tool != "market.get_quote" {
			t.Fatalf("expected tool market.get_quote, got %q", req.Tool)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"symbol": "AAPL",
				"price":  123.45,
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "market-data", Transport: "http", Endpoint: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out struct {
		Symbol string  `json:"symbol"`
		Price  float64 `json:"price"`
	}
	if err := client.CallTool(context.Background(), "market-data", "market.get_quote", map[string]any{"symbol": "AAPL"}, &out); err != nil {
		t.Fatalf("call: %v", err)
	}
	if out.Symbol != "AAPL" || out.Price != 123.45 {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestUTCPClient_HTTPCall_RawOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rMultiple": 2.0,
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "risk", Transport: "http", Endpoint: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out struct {
		RMultiple float64 `json:"rMultiple"`
	}
	if err := client.CallTool(context.Background(), "risk", "risk.r_multiple", map[string]any{"entry": 1}, &out); err != nil {
		t.Fatalf("call: %v", err)
	}
	if out.RMultiple != 2.0 {
		t.Fatalf("expected 2.0, got %v", out.RMultiple)
	}
}
