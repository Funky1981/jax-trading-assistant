package chat

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"jax-trading-assistant/internal/modules/harness"
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
	tools := AvailableTools(harness.DefaultPolicy(harness.ModeResearch))
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
		registered[m.Name] = struct{}{}
	}
	for _, bad := range forbidden {
		if _, found := registered[bad]; found {
			t.Errorf("tool catalogue must not expose mutating tool %q", bad)
		}
	}
}

// TestToolRouter_AvailableTools_ContainsExpected checks that the documented
// read-only tools are all present.
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
		"list_pending_approvals",
		"list_recent_blocked_candidates",
		"search_candidates",
		"query_knowledge",
		"compare_runs",
		"strategy_instance_summary",
		"blocked_candidate_analysis",
		"recent_research_narrative",
		"confidence_drift_summary",
		"signal_clustering_overview",
	}
	tools := AvailableTools(harness.DefaultPolicy(harness.ModeResearch))
	registered := make(map[string]struct{}, len(tools))
	for _, m := range tools {
		registered[m.Name] = struct{}{}
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
	for _, m := range AvailableTools(harness.DefaultPolicy(harness.ModeResearch)) {
		if m.Description == "" {
			t.Errorf("tool %q has empty description", m.Name)
		}
	}
}

func TestToolRouter_AvailableTools_ExposeMetadata(t *testing.T) {
	for _, m := range AvailableTools(harness.DefaultPolicy(harness.ModeResearch)) {
		if m.EvidenceLevel == "" {
			t.Errorf("tool %q missing evidenceLevel", m.Name)
		}
		if m.Freshness == "" {
			t.Errorf("tool %q missing freshness", m.Name)
		}
	}
}

func TestToolRouter_AvailableTools_ExposePolicyAvailability(t *testing.T) {
	tools := AvailableTools(harness.DefaultPolicy(harness.ModeLive))
	var foundBlocked bool
	for _, tool := range tools {
		if !tool.Allowed {
			foundBlocked = true
			if tool.PolicyReason == "" {
				t.Fatalf("tool %q missing policy reason", tool.Name)
			}
		}
	}
	if !foundBlocked {
		t.Fatal("expected at least one tool to be blocked in live mode")
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
		t.Fatal("errors.New should not chain ErrUnknownTool")
		// wrapped with errors.New won't chain — that's expected
	}
	// Direct identity check:
	if !errors.Is(ErrUnknownTool, ErrUnknownTool) {
		t.Error("ErrUnknownTool must satisfy errors.Is with itself")
	}
}
