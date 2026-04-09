package harness

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPolicyCheckToolAllowedRejectsNonReadOnlyTool(t *testing.T) {
	policy := DefaultPolicy(ModeResearch)
	err := policy.CheckToolAllowed(ToolDefinition{
		Name:     "approve_trade",
		ReadOnly: false,
		Handler:  func(context.Context, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("expected non-read-only tool to be rejected")
	}
}

func TestPolicyCheckAnswerAllowedRejectsActionClaims(t *testing.T) {
	policy := DefaultPolicy(ModeResearch)
	if err := policy.CheckAnswerAllowed("I executed the order for you."); err == nil {
		t.Fatal("expected forbidden action language to be rejected")
	}
}

func TestPromptBuilderIncludesRegisteredToolNames(t *testing.T) {
	reg := NewRegistry()
	if err := RegisterDefaultTools(reg, nil); err != nil {
		t.Fatalf("RegisterDefaultTools failed: %v", err)
	}

	toolNames := make([]string, 0, len(reg.AllTools()))
	for _, tool := range reg.AllTools() {
		toolNames = append(toolNames, tool.Name)
	}

	prompt := NewPromptBuilder().SystemPrompt(DefaultPolicy(ModeResearch), toolNames)
	if !containsInsensitive(prompt, "Approvals page") {
		t.Fatalf("prompt missing advisory boundary: %s", prompt)
	}
	if !containsInsensitive(prompt, "get_candidate_trade") {
		t.Fatalf("prompt missing registered tool names: %s", prompt)
	}
}

func TestPolicyCheckToolAllowedRejectsWeakInferenceInPaperMode(t *testing.T) {
	policy := DefaultPolicy(ModePaper)
	err := policy.CheckToolAllowed(ToolDefinition{
		Name:          "get_strategy",
		ReadOnly:      true,
		EvidenceLevel: EvidenceWeakInference,
		Handler:       func(context.Context, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("expected weak inference tool to be rejected in paper mode")
	}
}

func TestPolicyCheckToolAllowedRejectsHistoricalToolInLiveMode(t *testing.T) {
	policy := DefaultPolicy(ModeLive)
	err := policy.CheckToolAllowed(ToolDefinition{
		Name:                 "get_trade",
		ReadOnly:             true,
		EvidenceLevel:        EvidenceHardInternal,
		FreshnessExpectation: "historical_snapshot",
		Handler:              func(context.Context, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("expected historical tool to be rejected in live mode")
	}
}
