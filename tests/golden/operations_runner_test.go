package golden

import (
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type operationsGoldenCase struct {
	Name                          string                           `json:"name"`
	Item                          triage.Item                      `json:"item"`
	Decision                      operations.FeedbackDecisionInput `json:"decision"`
	ExpectedStatus                triage.Status                    `json:"expected_status"`
	ExpectedActionTypes           []operations.ActionType          `json:"expected_action_types"`
	ExpectedRequiresHumanApproval bool                             `json:"expected_requires_human_approval"`
	ExpectedAutoApplyAllowed      bool                             `json:"expected_auto_apply_allowed"`
	ExpectedForbiddenActions      []string                         `json:"expected_forbidden_actions"`
	ExpectedErrorContains         string                           `json:"expected_error_contains"`
}

func TestOperationsGoldenCases(t *testing.T) {
	files := []string{
		"research_gap_acceptance.json",
		"scoring_review_accepted_not_applied.json",
		"suggestion_rejected.json",
		"request_more_evidence.json",
		"close_no_action.json",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc operationsGoldenCase
			readJSON(t, filepath.Join("operations", file), &tc)

			got, err := operations.ApplyFeedbackDecision(tc.Item, tc.Decision)
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
			if got.Item.Status != tc.ExpectedStatus {
				t.Fatalf("%s status = %s, want %s", tc.Name, got.Item.Status, tc.ExpectedStatus)
			}
			if got.Item.RequiresHumanApproval != tc.ExpectedRequiresHumanApproval {
				t.Fatalf("%s item requires human approval = %v, want %v", tc.Name, got.Item.RequiresHumanApproval, tc.ExpectedRequiresHumanApproval)
			}
			if got.Item.AutoApplyAllowed != tc.ExpectedAutoApplyAllowed {
				t.Fatalf("%s item auto apply allowed = %v, want %v", tc.Name, got.Item.AutoApplyAllowed, tc.ExpectedAutoApplyAllowed)
			}
			assertContainsAll(t, got.Item.ForbiddenActions, tc.ExpectedForbiddenActions)
			if len(got.FollowUpActions) != len(tc.ExpectedActionTypes) {
				t.Fatalf("%s follow-up actions = %d, want %d", tc.Name, len(got.FollowUpActions), len(tc.ExpectedActionTypes))
			}
			for i, actionType := range tc.ExpectedActionTypes {
				action := got.FollowUpActions[i]
				if action.ActionType != actionType {
					t.Fatalf("%s action[%d] type = %s, want %s", tc.Name, i, action.ActionType, actionType)
				}
				if !action.RequiresHumanApproval {
					t.Fatalf("%s action[%d] requires human approval = false", tc.Name, i)
				}
				if action.AutoApplyAllowed {
					t.Fatalf("%s action[%d] auto apply allowed = true", tc.Name, i)
				}
				assertContainsAll(t, action.ForbiddenActions, feedback.ForbiddenActions())
			}
		})
	}
}
