package utcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func TestMemoryTools_HTTPRecallIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Tool != ToolMemoryRecall {
			t.Fatalf("expected tool %s, got %q", ToolMemoryRecall, req.Tool)
		}

		var in contracts.MemoryRecallRequest
		if err := json.Unmarshal(req.Input, &in); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if in.Bank != "trade_decisions" {
			t.Fatalf("expected bank trade_decisions, got %q", in.Bank)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": contracts.MemoryRecallResponse{Items: []contracts.MemoryItem{
				{
					TS:      time.Now().UTC(),
					Type:    "decision",
					Summary: "found prior decision",
					Data:    map[string]any{"ok": true},
					Source:  &contracts.MemorySource{System: "test"},
				},
			}},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{{ID: MemoryProviderID, Transport: "http", Endpoint: server.URL}},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewMemoryService(client)
	out, err := svc.Recall(context.Background(), contracts.MemoryRecallRequest{
		Bank: "trade_decisions",
		Query: contracts.MemoryQuery{
			Q: "gap",
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Summary != "found prior decision" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
