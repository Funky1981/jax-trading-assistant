package golden

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

type triageGoldenCase struct {
	Name                          string                  `json:"name"`
	Input                         feedback.RuleSuggestion `json:"input"`
	ExpectedSourceType            triage.SourceType       `json:"expected_source_type"`
	ExpectedStatus                triage.Status           `json:"expected_status"`
	ExpectedPriority              triage.Priority         `json:"expected_priority"`
	ExpectedRequiresHumanApproval bool                    `json:"expected_requires_human_approval"`
	ExpectedAutoApplyAllowed      bool                    `json:"expected_auto_apply_allowed"`
	ExpectedForbiddenActions      []string                `json:"expected_forbidden_actions"`
	ExpectedErrorContains         string                  `json:"expected_error_contains"`
}

func TestTriageGoldenCases(t *testing.T) {
	files := []string{
		"missed_opportunity_triage.json",
		"risk_veto_too_strict_triage.json",
		"paper_setup_failed_triage.json",
		"critical_priority_no_auto_apply.json",
		"live_promotion_blocked.json",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc triageGoldenCase
			readJSON(t, filepath.Join("triage", file), &tc)

			got, err := triage.NewItemFromSuggestion(tc.Input, fixedGoldenPhase11Now())
			if tc.ExpectedErrorContains != "" {
				if err == nil {
					t.Fatalf("%s returned nil error, want %q", tc.Name, tc.ExpectedErrorContains)
				}
				if !strings.Contains(err.Error(), tc.ExpectedErrorContains) {
					t.Fatalf("%s error = %q, want containing %q", tc.Name, err.Error(), tc.ExpectedErrorContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s returned error: %v", tc.Name, err)
			}

			if got.SourceType != tc.ExpectedSourceType {
				t.Fatalf("%s source = %s, want %s", tc.Name, got.SourceType, tc.ExpectedSourceType)
			}
			if got.Status != tc.ExpectedStatus {
				t.Fatalf("%s status = %s, want %s", tc.Name, got.Status, tc.ExpectedStatus)
			}
			if got.Priority != tc.ExpectedPriority {
				t.Fatalf("%s priority = %s, want %s", tc.Name, got.Priority, tc.ExpectedPriority)
			}
			if got.RequiresHumanApproval != tc.ExpectedRequiresHumanApproval {
				t.Fatalf("%s requires human approval = %v, want %v", tc.Name, got.RequiresHumanApproval, tc.ExpectedRequiresHumanApproval)
			}
			if got.AutoApplyAllowed != tc.ExpectedAutoApplyAllowed {
				t.Fatalf("%s auto apply allowed = %v, want %v", tc.Name, got.AutoApplyAllowed, tc.ExpectedAutoApplyAllowed)
			}
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
		})
	}
}

func fixedGoldenPhase11Now() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}
