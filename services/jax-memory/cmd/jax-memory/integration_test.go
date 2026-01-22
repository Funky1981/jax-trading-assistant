//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	if baseURL == "" && os.Getenv("JAX_MEMORY_COMPOSE") == "" {
		t.Skip("set JAX_MEMORY_URL or JAX_MEMORY_COMPOSE=1 to run")
	}
	if baseURL == "" {
		if err := startCompose(t); err != nil {
			t.Fatalf("compose: %v", err)
		}
		baseURL = "http://localhost:8090"
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

func startCompose(t *testing.T) error {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}
	composeFile := filepath.Join(root, "services", "jax-memory", "docker-compose.yml")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	up := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "up", "-d", "--build")
	up.Dir = root
	up.Env = os.Environ()
	if err := up.Run(); err != nil {
		return err
	}
	t.Cleanup(func() {
		down := exec.Command("docker", "compose", "-f", composeFile, "down")
		down.Dir = root
		_ = down.Run()
	})

	return waitForHealth(ctx, "http://localhost:8090/health")
}

func waitForHealth(ctx context.Context, url string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for health")
		case <-ticker.C:
		}
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", ".."))
	return root, nil
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
