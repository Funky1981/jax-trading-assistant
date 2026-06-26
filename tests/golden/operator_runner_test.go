package golden

import (
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/operator"
	"jax-trading-assistant/internal/decisioning/triage"
)

type operatorGoldenCase struct {
	Name                     string                   `json:"name"`
	Item                     triage.Item              `json:"item"`
	Action                   operations.HumanDecision `json:"action"`
	Request                  operator.ActionRequest   `json:"request"`
	ExpectedStatus           triage.Status            `json:"expected_status"`
	ExpectedFollowUpCount    int                      `json:"expected_follow_up_count"`
	ExpectedValidationText   string                   `json:"expected_validation_text"`
	ExpectedAutoApplyAllowed bool                     `json:"expected_auto_apply_allowed"`
}

func TestOperatorGoldenCases(t *testing.T) {
	files := []string{
		"accept_suggestion.json",
		"request_more_evidence.json",
		"live_promotion_blocked.json",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc operatorGoldenCase
			readJSON(t, filepath.Join("operator", file), &tc)
			repo := operations.NewMemoryRepository()
			if err := repo.SaveTriageItem(tc.Item); err != nil {
				t.Fatalf("%s SaveTriageItem returned error: %v", tc.Name, err)
			}
			service := operator.NewService(repo)
			got, err := runOperatorGoldenAction(service, tc.Action, tc.Request)
			if err != nil {
				t.Fatalf("%s operator action returned error: %v", tc.Name, err)
			}
			if tc.ExpectedValidationText != "" {
				if !strings.Contains(strings.Join(got.ValidationErrors, " "), tc.ExpectedValidationText) {
					t.Fatalf("%s validation errors = %v, want containing %q", tc.Name, got.ValidationErrors, tc.ExpectedValidationText)
				}
				item, ok := repo.GetTriageItem(tc.Item.TriageItemID)
				if !ok || item.Status != tc.Item.Status {
					t.Fatalf("%s item mutated after validation block: ok=%v item=%#v", tc.Name, ok, item)
				}
				return
			}
			if len(got.ValidationErrors) != 0 {
				t.Fatalf("%s validation errors = %v", tc.Name, got.ValidationErrors)
			}
			if got.NewStatus != tc.ExpectedStatus {
				t.Fatalf("%s status = %s, want %s", tc.Name, got.NewStatus, tc.ExpectedStatus)
			}
			if len(got.FollowUpActionIDs) != tc.ExpectedFollowUpCount {
				t.Fatalf("%s follow-up action ids = %v, want count %d", tc.Name, got.FollowUpActionIDs, tc.ExpectedFollowUpCount)
			}
			if got.AutoApplyAllowed != tc.ExpectedAutoApplyAllowed {
				t.Fatalf("%s auto apply allowed = %v, want %v", tc.Name, got.AutoApplyAllowed, tc.ExpectedAutoApplyAllowed)
			}
			assertContainsAll(t, got.ForbiddenActions, feedback.ForbiddenActions())
		})
	}
}

func runOperatorGoldenAction(service operator.Service, action operations.HumanDecision, request operator.ActionRequest) (operator.ActionResult, error) {
	switch action {
	case operations.DecisionAcceptSuggestion:
		return service.AcceptSuggestion(request)
	case operations.DecisionRequestMoreEvidence:
		return service.RequestMoreEvidence(request)
	case operations.DecisionRejectSuggestion:
		return service.RejectSuggestion(request)
	case operations.DecisionDeferDecision:
		return service.DeferSuggestion(request)
	case operations.DecisionCloseNoAction:
		return service.CloseNoAction(request)
	default:
		return operator.ActionResult{}, nil
	}
}
