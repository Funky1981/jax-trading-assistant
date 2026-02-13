package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Snapshot struct {
	Captured time.Time              `json:"captured"`
	Service  string                 `json:"service"`
	Endpoint string                 `json:"endpoint"`
	Method   string                 `json:"method"`
	Request  interface{}            `json:"request,omitempty"`
	Response interface{}            `json:"response"`
	Metadata map[string]interface{} `json:"metadata"`
}

func main() {
	outputDir := flag.String("output", "tests/golden", "Output directory for golden files")
	baseURL := flag.String("base-url", "http://localhost", "Base URL for services")
	flag.Parse()

	log.Println("🎯 Capturing golden baseline snapshots...")

	// Capture signals
	if err := captureSignals(*baseURL, *outputDir); err != nil {
		log.Printf("⚠️  Warning: Failed to capture signals: %v", err)
	} else {
		log.Println("✅ Signals captured")
	}

	// Capture executions (recent trades)
	if err := captureExecutions(*baseURL, *outputDir); err != nil {
		log.Printf("⚠️  Warning: Failed to capture executions: %v", err)
	} else {
		log.Println("✅ Executions captured")
	}

	// Capture orchestration runs
	if err := captureOrchestration(*baseURL, *outputDir); err != nil {
		log.Printf("⚠️  Warning: Failed to capture orchestration: %v", err)
	} else {
		log.Println("✅ Orchestration captured")
	}

	log.Println("🎉 Golden baseline capture complete!")
}

func captureSignals(baseURL, outputDir string) error {
	url := fmt.Sprintf("%s:8096/api/v1/signals", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch signals: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var signals interface{}
	if err := json.Unmarshal(body, &signals); err != nil {
		return err
	}

	snapshot := Snapshot{
		Captured: time.Now(),
		Service:  "jax-signal-generator",
		Endpoint: "/api/v1/signals",
		Method:   "GET",
		Response: signals,
		Metadata: map[string]interface{}{
			"version": "baseline",
			"count":   countSignals(signals),
		},
	}

	return saveSnapshot(snapshot, filepath.Join(outputDir, "signals", fmt.Sprintf("baseline-%s.json", time.Now().Format("2006-01-02"))))
}

func captureExecutions(baseURL, outputDir string) error {
	// Query recent trades from database via jax-api
	url := fmt.Sprintf("%s:8081/api/v1/trades?limit=20", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch trades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var trades interface{}
	if err := json.Unmarshal(body, &trades); err != nil {
		return err
	}

	snapshot := Snapshot{
		Captured: time.Now(),
		Service:  "jax-trade-executor",
		Endpoint: "/api/v1/trades",
		Method:   "GET",
		Response: trades,
		Metadata: map[string]interface{}{
			"version": "baseline",
			"limit":   20,
		},
	}

	return saveSnapshot(snapshot, filepath.Join(outputDir, "executions", fmt.Sprintf("baseline-%s.json", time.Now().Format("2006-01-02"))))
}

func captureOrchestration(baseURL, outputDir string) error {
	// Query recent orchestration runs from database via jax-api
	url := fmt.Sprintf("%s:8081/api/v1/orchestration/runs?limit=10", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch orchestration runs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If endpoint doesn't exist, create empty snapshot
		log.Println("⚠️  Orchestration endpoint not available, creating empty snapshot")
		snapshot := Snapshot{
			Captured: time.Now(),
			Service:  "jax-orchestrator",
			Endpoint: "/api/v1/orchestration/runs",
			Method:   "GET",
			Response: []interface{}{},
			Metadata: map[string]interface{}{
				"version": "baseline",
				"note":    "endpoint not available",
			},
		}
		return saveSnapshot(snapshot, filepath.Join(outputDir, "orchestration", fmt.Sprintf("baseline-%s.json", time.Now().Format("2006-01-02"))))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var runs interface{}
	if err := json.Unmarshal(body, &runs); err != nil {
		return err
	}

	snapshot := Snapshot{
		Captured: time.Now(),
		Service:  "jax-orchestrator",
		Endpoint: "/api/v1/orchestration/runs",
		Method:   "GET",
		Response: runs,
		Metadata: map[string]interface{}{
			"version": "baseline",
			"limit":   10,
		},
	}

	return saveSnapshot(snapshot, filepath.Join(outputDir, "orchestration", fmt.Sprintf("baseline-%s.json", time.Now().Format("2006-01-02"))))
}

func saveSnapshot(snapshot Snapshot, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write JSON file with pretty formatting
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return err
	}

	log.Printf("  └─ Saved to: %s", path)
	return nil
}

func countSignals(data interface{}) int {
	switch v := data.(type) {
	case []interface{}:
		return len(v)
	case map[string]interface{}:
		if signals, ok := v["signals"].([]interface{}); ok {
			return len(signals)
		}
	}
	return 0
}
