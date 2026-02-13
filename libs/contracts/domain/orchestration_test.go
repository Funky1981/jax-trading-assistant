package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOrchestrationRun_JSONMarshaling(t *testing.T) {
	completedAt := time.Date(2026, 2, 13, 10, 35, 0, 0, time.UTC)
	run := OrchestrationRun{
		ID:          "run-123",
		UserID:      "user-456",
		Query:       "Analyze AAPL for potential trades",
		Status:      "completed",
		StartedAt:   time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		CompletedAt: &completedAt,
		Response:    "Analysis complete. AAPL shows strong bullish momentum.",
		Metadata: map[string]interface{}{
			"signal_id":  "sig-789",
			"confidence": 0.85,
		},
		Providers: []ProviderUsage{
			{
				Name:     "openai",
				Duration: 1234.5,
				Success:  true,
			},
			{
				Name:     "anthropic",
				Duration: 987.3,
				Success:  true,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("Failed to marshal orchestration run: %v", err)
	}

	// Unmarshal back
	var decoded OrchestrationRun
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal orchestration run: %v", err)
	}

	// Verify fields
	if decoded.ID != run.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, run.ID)
	}
	if decoded.Query != run.Query {
		t.Errorf("Query mismatch: got %s, want %s", decoded.Query, run.Query)
	}
	if decoded.Status != run.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, run.Status)
	}
	if decoded.CompletedAt == nil {
		t.Errorf("CompletedAt should not be nil")
	}
	if len(decoded.Providers) != len(run.Providers) {
		t.Errorf("Providers length mismatch: got %d, want %d", len(decoded.Providers), len(run.Providers))
	}
}

func TestOrchestrationRun_Running(t *testing.T) {
	run := OrchestrationRun{
		ID:        "run-456",
		UserID:    "user-789",
		Query:     "What's the market sentiment for tech stocks?",
		Status:    "running",
		StartedAt: time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		Providers: []ProviderUsage{
			{
				Name:     "openai",
				Duration: 0,
				Success:  true,
			},
		},
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("Failed to marshal running orchestration: %v", err)
	}

	var decoded OrchestrationRun
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal running orchestration: %v", err)
	}

	if decoded.Status != "running" {
		t.Errorf("Status mismatch: got %s, want running", decoded.Status)
	}
	if decoded.CompletedAt != nil {
		t.Errorf("CompletedAt should be nil for running orchestration")
	}
}

func TestProviderUsage_Failed(t *testing.T) {
	provider := ProviderUsage{
		Name:     "anthropic",
		Duration: 500.0,
		Success:  false,
		ErrorMsg: "Rate limit exceeded",
	}

	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("Failed to marshal provider usage: %v", err)
	}

	var decoded ProviderUsage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal provider usage: %v", err)
	}

	if decoded.Success != false {
		t.Errorf("Success should be false")
	}
	if decoded.ErrorMsg != provider.ErrorMsg {
		t.Errorf("ErrorMsg mismatch: got %s, want %s", decoded.ErrorMsg, provider.ErrorMsg)
	}
}
