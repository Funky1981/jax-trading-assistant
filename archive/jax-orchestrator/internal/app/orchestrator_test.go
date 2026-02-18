package app

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/dexter"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategies"
	testfixtures "jax-trading-assistant/libs/testing"
)

type fakeMemory struct {
	lastRecallBank string
	lastRetainBank string
	lastRetainItem contracts.MemoryItem
	recallInvoked  bool
	retainInvoked  bool
}

func (m *fakeMemory) Recall(_ context.Context, bank string, _ contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	m.lastRecallBank = bank
	m.recallInvoked = true
	return []contracts.MemoryItem{
		{
			TS:      time.Now().UTC(),
			Type:    "decision",
			Summary: "prior memory",
			Data:    map[string]any{"ok": true},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}, nil
}

func (m *fakeMemory) Retain(_ context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	m.lastRetainBank = bank
	m.lastRetainItem = item
	m.retainInvoked = true
	return "mem_1", nil
}

type fakeAgent struct {
	lastPlanRequest agent0.PlanRequest
}

func (a *fakeAgent) Plan(_ context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error) {
	a.lastPlanRequest = req
	return agent0.PlanResponse{
		Summary:        "Plan summary",
		Steps:          []string{"step1", "step2"},
		Action:         "executed",
		Confidence:     0.7,
		ReasoningNotes: "short notes",
	}, nil
}

func (a *fakeAgent) Execute(_ context.Context, _ agent0.ExecuteRequest) (agent0.ExecuteResponse, error) {
	return agent0.ExecuteResponse{Success: true}, nil
}

type fakeTools struct {
	lastPlan PlanResult
}

func (t *fakeTools) Execute(_ context.Context, plan PlanResult) ([]ToolRun, error) {
	t.lastPlan = plan
	return []ToolRun{{Name: "risk.position_size", Success: true}}, nil
}

func TestOrchestrator_Run_RecallPlanExecuteRetain(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	orch := NewOrchestrator(memory, agent, tools, registry)

	var constraints map[string]any
	if err := json.Unmarshal(testfixtures.LoadFixture(t, "orchestrator_constraints.json"), &constraints); err != nil {
		t.Fatalf("constraints fixture: %v", err)
	}

	result, err := orch.Run(context.Background(), OrchestrationRequest{
		Bank:        "trade_decisions",
		Symbol:      "AAPL",
		Strategy:    "earnings_gap_v1",
		Constraints: constraints,
		UserContext: "user constraints",
		Tags:        []string{"earnings", "risk-high"},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !memory.recallInvoked || !memory.retainInvoked {
		t.Fatalf("expected recall and retain to be invoked")
	}
	if memory.lastRecallBank != "trade_decisions" {
		t.Fatalf("expected recall bank trade_decisions, got %q", memory.lastRecallBank)
	}
	if agent.lastPlanRequest.Symbol != "AAPL" {
		t.Fatalf("expected agent plan symbol AAPL, got %q", agent.lastPlanRequest.Symbol)
	}
	if len(agent.lastPlanRequest.Memories) == 0 {
		t.Fatalf("expected recalled memories passed to agent")
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "risk.position_size" {
		t.Fatalf("unexpected tools: %#v", result.Tools)
	}
	if memory.lastRetainItem.Type != "decision" || memory.lastRetainItem.Symbol != "AAPL" {
		t.Fatalf("unexpected retained item: %#v", memory.lastRetainItem)
	}
	if memory.lastRetainItem.TS.IsZero() {
		t.Fatalf("expected retained item timestamp")
	}
	if memory.lastRetainItem.Summary == "" {
		t.Fatalf("expected retained item summary")
	}
	if len(memory.lastRetainItem.Tags) == 0 {
		t.Fatalf("expected retained item tags")
	}
	if memory.lastRetainItem.Data == nil {
		t.Fatalf("expected retained item data")
	}
	if redacted := observability.RedactValue(memory.lastRetainItem.Data); !reflect.DeepEqual(memory.lastRetainItem.Data, redacted) {
		t.Fatalf("expected retained data to be redacted")
	}
}

func TestOrchestrator_WithStrategySignals(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	// Register RSI strategy
	rsiStrategy := strategies.NewRSIMomentumStrategy()
	registry.Register(rsiStrategy, rsiStrategy.GetMetadata())

	orch := NewOrchestrator(memory, agent, tools, registry)

	// Constraints with oversold RSI
	constraints := map[string]any{
		"price":        150.0,
		"rsi":          25.0,
		"atr":          2.5,
		"market_trend": "bullish",
		"volume":       int64(1000000),
		"avg_volume":   int64(900000),
	}

	result, err := orch.Run(context.Background(), OrchestrationRequest{
		Bank:        "trade_decisions",
		Symbol:      "AAPL",
		Strategy:    "rsi_momentum_v1",
		Constraints: constraints,
		UserContext: "Analyzing AAPL",
		Tags:        []string{"momentum"},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Check that plan was generated
	if result.Plan.Summary == "" {
		t.Error("expected non-empty plan summary")
	}

	// Check that agent received context with signals
	if !contains(agent.lastPlanRequest.Context, "Strategy signals") {
		t.Error("expected agent context to include strategy signals")
	}

	// Check that retained memory includes signals
	signalsData := memory.lastRetainItem.Data["signals"]
	t.Logf("signals type: %T, value: %+v", signalsData, signalsData)
	if data, ok := signalsData.([]map[string]interface{}); !ok || len(data) == 0 {
		// Try []interface{} as well since RedactValue might convert it
		if data2, ok2 := signalsData.([]interface{}); ok2 && len(data2) > 0 {
			// OK, RedactValue converted it
		} else {
			t.Errorf("expected retained memory to include signals, got type=%T", signalsData)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestOrchestrator_WithDexterResearch(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	// Mock Dexter client
	dexterMock := &dexter.MockClient{
		ResearchCompanyFunc: func(ctx context.Context, input dexter.ResearchCompanyInput) (dexter.ResearchCompanyOutput, error) {
			if input.Ticker != "TSLA" {
				t.Errorf("expected ticker TSLA, got %s", input.Ticker)
			}
			return dexter.ResearchCompanyOutput{
				Ticker:    input.Ticker,
				Summary:   "Tesla showing strong EV market growth",
				KeyPoints: []string{"Revenue up 25% YoY", "Production capacity expanding"},
				Metrics:   map[string]interface{}{"pe_ratio": 65.2, "growth_rate": 0.25},
			}, nil
		},
	}

	orch := NewOrchestrator(memory, agent, tools, registry).WithDexter(dexterMock)

	result, err := orch.Run(context.Background(), OrchestrationRequest{
		Bank:            "trade_decisions",
		Symbol:          "TSLA",
		Strategy:        "",
		Constraints:     map[string]any{"price": 250.0},
		UserContext:     "Analyzing TSLA earnings opportunity",
		Tags:            []string{"earnings"},
		ResearchQueries: []string{"What is revenue growth?", "What are production trends?"},
	})

	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Check that plan was generated
	if result.Plan.Summary == "" {
		t.Error("expected non-empty plan summary")
	}

	// Check that agent received context with research
	if !contains(agent.lastPlanRequest.Context, "Dexter research") {
		t.Error("expected agent context to include Dexter research")
	}

	if !contains(agent.lastPlanRequest.Context, "Tesla showing strong EV market growth") {
		t.Error("expected agent context to include research summary")
	}

	// Check that retained memory includes research
	if data, ok := memory.lastRetainItem.Data["research"].(map[string]interface{}); !ok || data == nil {
		t.Errorf("expected retained memory to include research data, got: %+v", memory.lastRetainItem.Data)
	} else {
		if summary, ok := data["summary"].(string); !ok || summary == "" {
			t.Errorf("expected research summary in retained memory, got: %+v", data)
		}
		// key_points can be []string or []interface{} depending on how it's stored
		keyPointsRaw := data["key_points"]
		if keyPointsRaw == nil {
			t.Error("expected research key points in retained memory, got nil")
		} else {
			// Try []string first
			if kp, ok := keyPointsRaw.([]string); ok && len(kp) > 0 {
				// OK
			} else if kp, ok := keyPointsRaw.([]interface{}); ok && len(kp) > 0 {
				// Also OK
			} else {
				t.Errorf("expected research key points (non-empty array), got type=%T, value=%+v", keyPointsRaw, keyPointsRaw)
			}
		}
	}
}
