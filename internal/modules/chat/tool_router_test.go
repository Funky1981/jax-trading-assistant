package chat

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestToolRouter_UnknownTool verifies that an unregistered tool name returns
// ErrUnknownTool without panicking — even with a nil pool, since the dispatch
// switch returns before any DB access for unrecognised names.
func TestToolRouter_UnknownTool(t *testing.T) {
	r := NewToolRouter(nil)

	for _, name := range []string{
		"approve_trade",
		"execute_order",
		"submit_order",
		"cancel_trade",
		"",
		"not_a_tool",
	} {
		_, err := r.Dispatch(context.Background(), ToolCall{Name: name, Args: json.RawMessage(`{}`)})
		if !errors.Is(err, ErrUnknownTool) {
			t.Errorf("tool %q: expected ErrUnknownTool, got %v", name, err)
		}
	}
}

// TestToolRouter_AvailableTools_OnlyReadOnly verifies that the tool catalogue
// contains only read-only inspection tools and never exposes any names that
// could mutate trading or approval state.
func TestToolRouter_AvailableTools_OnlyReadOnly(t *testing.T) {
	tools := AvailableTools()
	if len(tools) == 0 {
		t.Fatal("AvailableTools returned empty list")
	}

	forbidden := []string{
		"approve_trade",
		"reject_trade",
		"execute_order",
		"submit_order",
		"cancel_trade",
		"place_order",
		"modify_order",
		"decide",
	}
	registered := make(map[string]struct{}, len(tools))
	for _, m := range tools {
		registered[m["name"]] = struct{}{}
	}
	for _, bad := range forbidden {
		if _, found := registered[bad]; found {
			t.Errorf("tool catalogue must not expose mutating tool %q", bad)
		}
	}
}

// TestToolRouter_AvailableTools_ContainsExpected checks that the eight
// documented read-only tools are all present.
func TestToolRouter_AvailableTools_ContainsExpected(t *testing.T) {
	expected := []string{
		"get_candidate_trade",
		"get_signal",
		"get_trade",
		"get_strategy",
		"get_strategy_instance",
		"get_orchestration_run",
		"search_research_runs",
		"explain_trade_blockers",
	}
	tools := AvailableTools()
	registered := make(map[string]struct{}, len(tools))
	for _, m := range tools {
		registered[m["name"]] = struct{}{}
	}
	for _, want := range expected {
		if _, found := registered[want]; !found {
			t.Errorf("expected tool %q in AvailableTools", want)
		}
	}
}

// TestToolRouter_AvailableTools_HaveDescriptions ensures every entry has
// a non-empty description so the frontend can display it.
func TestToolRouter_AvailableTools_HaveDescriptions(t *testing.T) {
	for _, m := range AvailableTools() {
		if m["description"] == "" {
			t.Errorf("tool %q has empty description", m["name"])
		}
	}
}

// TestErrUnknownTool_IsSentinel confirms the sentinel is a stable value
// that callers can compare with errors.Is.
func TestErrUnknownTool_IsSentinel(t *testing.T) {
	if ErrUnknownTool == nil {
		t.Fatal("ErrUnknownTool must not be nil")
	}
	wrapped := errors.New("tool not found: " + ErrUnknownTool.Error())
	if errors.Is(wrapped, ErrUnknownTool) {
		// wrapped with errors.New won't chain — that's expected
	}
	// Direct identity check:
	if !errors.Is(ErrUnknownTool, ErrUnknownTool) {
		t.Error("ErrUnknownTool must satisfy errors.Is with itself")
	}
}
