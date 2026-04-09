package harness

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRegistryRegisterRejectsInvalidDefinitions(t *testing.T) {
	reg := NewRegistry()

	if err := reg.Register(ToolDefinition{}); err == nil {
		t.Fatal("expected empty tool name to fail")
	}
	if err := reg.Register(ToolDefinition{
		Name:     "mutating_tool",
		ReadOnly: false,
		Handler:  func(context.Context, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	}); err == nil {
		t.Fatal("expected non-read-only tool to fail")
	}
	if err := reg.Register(ToolDefinition{
		Name:     "missing_handler",
		ReadOnly: true,
	}); err == nil {
		t.Fatal("expected missing handler to fail")
	}
}

func TestRegistryRegisterRejectsDuplicates(t *testing.T) {
	reg := NewRegistry()
	def := ToolDefinition{
		Name:     "query",
		ReadOnly: true,
		Handler:  func(context.Context, json.RawMessage) (json.RawMessage, error) { return json.RawMessage(`{}`), nil },
	}
	if err := reg.Register(def); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := reg.Register(def); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

func TestRegisterDefaultToolsRegistersSharedToolSet(t *testing.T) {
	reg := NewRegistry()
	if err := RegisterDefaultTools(reg, nil); err != nil {
		t.Fatalf("RegisterDefaultTools failed: %v", err)
	}

	tools := reg.AllTools()
	if len(tools) != 18 {
		t.Fatalf("expected 18 registered tools, got %d", len(tools))
	}

	def, ok := reg.Get("query_knowledge")
	if !ok {
		t.Fatal("expected query_knowledge to be registered")
	}
	if def.EvidenceLevel != EvidenceDerivedInternal {
		t.Fatalf("unexpected evidence level: %s", def.EvidenceLevel)
	}

	if _, ok := reg.Get("compare_runs"); !ok {
		t.Fatal("expected compare_runs to be registered")
	}
}
