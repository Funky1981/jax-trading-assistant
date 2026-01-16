package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

type toolRequest struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

type toolResponse struct {
	Output json.RawMessage `json:"output"`
}

func TestIntegration_JaxMemory_RetainRecall(t *testing.T) {
	baseURL := os.Getenv("JAX_MEMORY_URL")
	if baseURL == "" {
		t.Skip("JAX_MEMORY_URL not set")
	}

	healthResp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	healthResp.Body.Close()
	if healthResp.StatusCode < 200 || healthResp.StatusCode >= 300 {
		t.Fatalf("health status: %d", healthResp.StatusCode)
	}

	bank := "trade_decisions"
	summary := "integration test memory"

	retainBody, _ := json.Marshal(toolRequest{
		Tool: "memory.retain",
		Input: mustJSON(map[string]any{
			"bank": bank,
			"item": map[string]any{
				"ts":      time.Now().UTC(),
				"type":    "decision",
				"summary": summary,
				"tags":    []string{"integration"},
				"data":    map[string]any{"source": "integration"},
				"source":  map[string]any{"system": "integration"},
			},
		}),
	})

	retainResp, err := http.Post(baseURL+"/tools", "application/json", bytes.NewReader(retainBody))
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	retainResp.Body.Close()
	if retainResp.StatusCode < 200 || retainResp.StatusCode >= 300 {
		t.Fatalf("retain status: %d", retainResp.StatusCode)
	}

	found := false
	for i := 0; i < 5; i++ {
		recallBody, _ := json.Marshal(toolRequest{
			Tool: "memory.recall",
			Input: mustJSON(map[string]any{
				"bank": bank,
				"query": map[string]any{
					"q":     summary,
					"limit": 5,
				},
			}),
		})

		recallResp, err := http.Post(baseURL+"/tools", "application/json", bytes.NewReader(recallBody))
		if err != nil {
			t.Fatalf("recall: %v", err)
		}
		if recallResp.StatusCode < 200 || recallResp.StatusCode >= 300 {
			recallResp.Body.Close()
			t.Fatalf("recall status: %d", recallResp.StatusCode)
		}

		var payload toolResponse
		if err := json.NewDecoder(recallResp.Body).Decode(&payload); err != nil {
			recallResp.Body.Close()
			t.Fatalf("decode: %v", err)
		}
		recallResp.Body.Close()

		var out struct {
			Items []struct {
				Summary string `json:"summary"`
			} `json:"items"`
		}
		if err := json.Unmarshal(payload.Output, &out); err != nil {
			t.Fatalf("output decode: %v", err)
		}
		for _, item := range out.Items {
			if item.Summary == summary {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !found {
		t.Fatalf("expected to recall retained memory")
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
