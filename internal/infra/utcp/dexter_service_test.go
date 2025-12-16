package utcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDexterService_ResearchCompany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Tool  string `json:"tool"`
			Input struct {
				Ticker    string   `json:"ticker"`
				Questions []string `json:"questions"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Tool != ToolDexterResearchCompany {
			t.Fatalf("expected tool %q, got %q", ToolDexterResearchCompany, req.Tool)
		}
		if req.Input.Ticker != "AAPL" {
			t.Fatalf("expected ticker AAPL, got %q", req.Input.Ticker)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"ticker":       "AAPL",
				"summary":      "Test summary",
				"key_points":   []string{"kp1", "kp2"},
				"metrics":      map[string]any{"pe_ratio": 28.5},
				"raw_markdown": "## AAPL\n- kp1\n",
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: DexterProviderID, Transport: "http", Endpoint: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewDexterService(client)
	out, err := svc.ResearchCompany(context.Background(), "AAPL", []string{"q1"})
	if err != nil {
		t.Fatalf("research: %v", err)
	}
	if out.Ticker != "AAPL" || out.Summary == "" || len(out.KeyPoints) != 2 {
		t.Fatalf("unexpected output: %#v", out)
	}
	if out.Metrics["pe_ratio"] != 28.5 {
		t.Fatalf("expected pe_ratio=28.5, got %v", out.Metrics["pe_ratio"])
	}
}

func TestDexterService_CompareCompanies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Tool  string `json:"tool"`
			Input struct {
				Tickers []string `json:"tickers"`
				Focus   string   `json:"focus"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Tool != ToolDexterCompareCompanies {
			t.Fatalf("expected tool %q, got %q", ToolDexterCompareCompanies, req.Tool)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"comparison_axis": "growth_vs_margins",
				"items": []map[string]any{
					{"ticker": "AAPL", "thesis": "A", "notes": []string{"n1"}},
					{"ticker": "MSFT", "thesis": "M", "notes": []string{}},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: DexterProviderID, Transport: "http", Endpoint: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewDexterService(client)
	out, err := svc.CompareCompanies(context.Background(), []string{"AAPL", "MSFT"}, "growth_vs_margins")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if out.ComparisonAxis != "growth_vs_margins" || len(out.Items) != 2 {
		t.Fatalf("unexpected output: %#v", out)
	}
}
