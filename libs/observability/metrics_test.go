package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"testing"
	"time"
)

func captureLog(fn func()) map[string]interface{} {
	old := logger
	defer func() { logger = old }()

	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)

	fn()

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil
	}
	return result
}

func TestRecordStrategySignal(t *testing.T) {
	ctx := WithRunInfo(context.Background(), RunInfo{
		RunID:  "run_123",
		Symbol: "AAPL",
	})

	result := captureLog(func() {
		RecordStrategySignal(ctx, "rsi_momentum_v1", "buy", 0.85)
	})

	if result == nil {
		t.Fatal("expected JSON log output")
	}

	if result["event"] != "metric" {
		t.Errorf("expected event=metric, got %v", result["event"])
	}

	if result["name"] != "strategy_signal" {
		t.Errorf("expected name=strategy_signal, got %v", result["name"])
	}

	if result["strategy"] != "rsi_momentum_v1" {
		t.Errorf("expected strategy=rsi_momentum_v1, got %v", result["strategy"])
	}

	if result["type"] != "buy" {
		t.Errorf("expected type=buy, got %v", result["type"])
	}

	if result["confidence"] != 0.85 {
		t.Errorf("expected confidence=0.85, got %v", result["confidence"])
	}

	if result["run_id"] != "run_123" {
		t.Errorf("expected run_id=run_123, got %v", result["run_id"])
	}
}

func TestRecordOrchestrationRun_Success(t *testing.T) {
	ctx := WithRunInfo(context.Background(), RunInfo{
		RunID:  "orch_456",
		Symbol: "TSLA",
	})

	result := captureLog(func() {
		RecordOrchestrationRun(ctx, 250*time.Millisecond, 7, nil)
	})

	if result == nil {
		t.Fatal("expected JSON log output")
	}

	if result["name"] != "orchestration_run" {
		t.Errorf("expected name=orchestration_run, got %v", result["name"])
	}

	if result["stages"] != float64(7) {
		t.Errorf("expected stages=7, got %v", result["stages"])
	}

	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}

	latency := result["latency_ms"].(float64)
	if latency < 249 || latency > 251 {
		t.Errorf("expected latency_ms ~250, got %v", latency)
	}
}

func TestRecordOrchestrationRun_Failure(t *testing.T) {
	ctx := context.Background()

	result := captureLog(func() {
		RecordOrchestrationRun(ctx, 100*time.Millisecond, 3, io.EOF)
	})

	if result == nil {
		t.Fatal("expected JSON log output")
	}

	if result["success"] != false {
		t.Errorf("expected success=false, got %v", result["success"])
	}

	if result["error"] != "EOF" {
		t.Errorf("expected error=EOF, got %v", result["error"])
	}
}

func TestRecordResearchQuery(t *testing.T) {
	ctx := WithRunInfo(context.Background(), RunInfo{
		RunID: "research_789",
	})

	result := captureLog(func() {
		RecordResearchQuery(ctx, "dexter", 500*time.Millisecond, nil)
	})

	if result == nil {
		t.Fatal("expected JSON log output")
	}

	if result["name"] != "research_query" {
		t.Errorf("expected name=research_query, got %v", result["name"])
	}

	if result["service"] != "dexter" {
		t.Errorf("expected service=dexter, got %v", result["service"])
	}

	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}
}

func TestRecordAgent0Plan(t *testing.T) {
	ctx := WithRunInfo(context.Background(), RunInfo{
		RunID:  "agent_999",
		Symbol: "NVDA",
	})

	result := captureLog(func() {
		RecordAgent0Plan(ctx, 1200*time.Millisecond, 5, 0.92, nil)
	})

	if result == nil {
		t.Fatal("expected JSON log output")
	}

	if result["name"] != "agent0_plan" {
		t.Errorf("expected name=agent0_plan, got %v", result["name"])
	}

	if result["steps"] != float64(5) {
		t.Errorf("expected steps=5, got %v", result["steps"])
	}

	if result["confidence"] != 0.92 {
		t.Errorf("expected confidence=0.92, got %v", result["confidence"])
	}

	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}

	latency := result["latency_ms"].(float64)
	if latency < 1199 || latency > 1201 {
		t.Errorf("expected latency_ms ~1200, got %v", latency)
	}
}

func TestMain(m *testing.M) {
	// Suppress log output during tests unless VERBOSE=1
	if os.Getenv("VERBOSE") != "1" {
		logger = log.New(io.Discard, "", 0)
	}
	os.Exit(m.Run())
}
