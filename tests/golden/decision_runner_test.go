package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

type expectedDecisionCase struct {
	Decision                core.DecisionValue `json:"decision"`
	Brain                   string             `json:"brain"`
	PrimaryReasonContains   string             `json:"primary_reason_contains"`
	MinConflictScore        float64            `json:"min_conflict_score"`
	MaxEdgeScore            float64            `json:"max_edge_score"`
	AllowedActionsContains  []string           `json:"allowed_actions_contains"`
	ForbiddenActionsContain []string           `json:"forbidden_actions_contains"`
	ReviewAfterContains     []string           `json:"review_after_contains"`
}

func TestDecisionRunnerFTSEOilLabourConflictReturnsNoTrade(t *testing.T) {
	var event core.Event
	readJSON(t, filepath.Join("events", "ftse_oil_labour_conflict.input.json"), &event)

	var expected expectedDecisionCase
	readJSON(t, filepath.Join("events", "ftse_oil_labour_conflict.expected.json"), &expected)

	bundle := core.Evaluate(core.EvaluationInput{
		Event: event,
		Scores: core.Scores{
			ClarityScore:  0.38,
			EdgeScore:     0.22,
			ConflictScore: 0.79,
			RiskScore:     0.66,
		},
		GeneratedAt: time.Date(2026, 6, 18, 10, 35, 0, 0, time.UTC),
	})

	decision := bundle.FinalDecision
	if decision.Decision != expected.Decision {
		t.Fatalf("decision = %s, want %s", decision.Decision, expected.Decision)
	}
	if decision.Brain != expected.Brain {
		t.Fatalf("brain = %q, want %q", decision.Brain, expected.Brain)
	}
	if !strings.Contains(strings.ToLower(decision.PrimaryReason), strings.ToLower(expected.PrimaryReasonContains)) {
		t.Fatalf("primary reason = %q, want to contain %q", decision.PrimaryReason, expected.PrimaryReasonContains)
	}
	if decision.ConflictScore < expected.MinConflictScore {
		t.Fatalf("conflict score = %v, want >= %v", decision.ConflictScore, expected.MinConflictScore)
	}
	if decision.EdgeScore > expected.MaxEdgeScore {
		t.Fatalf("edge score = %v, want <= %v", decision.EdgeScore, expected.MaxEdgeScore)
	}
	assertContainsAll(t, decision.AllowedActions, expected.AllowedActionsContains)
	assertContainsAll(t, decision.ForbiddenActions, expected.ForbiddenActionsContain)
	assertContainsAll(t, decision.ReviewAfter, expected.ReviewAfterContains)
	if decision.IsError() {
		t.Fatal("NO_TRADE golden case must be a successful decision, not an error")
	}
}

func readJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func assertContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q in %v", item, got)
		}
	}
}
