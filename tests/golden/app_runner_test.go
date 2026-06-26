package golden

import (
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/decisioning/app"
	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type appGoldenCase struct {
	Name                     string                  `json:"name"`
	Item                     triage.Item             `json:"item"`
	Request                  app.ReviewActionRequest `json:"request"`
	ExpectedStatus           triage.Status           `json:"expected_status"`
	ExpectedFollowUpCount    int                     `json:"expected_follow_up_count"`
	ExpectedValidationText   string                  `json:"expected_validation_text"`
	ExpectedAutoApplyAllowed bool                    `json:"expected_auto_apply_allowed"`
}

func TestReviewOperationsAppGoldenCases(t *testing.T) {
	files := []string{
		"accept_suggestion_internal_access.json",
		"live_execution_blocked_internal_access.json",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc appGoldenCase
			readJSON(t, filepath.Join("app", file), &tc)
			repo := operations.NewMemoryRepository()
			if err := repo.SaveTriageItem(tc.Item); err != nil {
				t.Fatalf("%s SaveTriageItem returned error: %v", tc.Name, err)
			}
			service, err := app.NewReviewOperationsService(app.DefaultReviewOperationsConfig(), repo)
			if err != nil {
				t.Fatalf("%s NewReviewOperationsService returned error: %v", tc.Name, err)
			}

			got, err := service.AcceptSuggestion(tc.Request)
			if err != nil {
				t.Fatalf("%s AcceptSuggestion returned error: %v", tc.Name, err)
			}
			if tc.ExpectedValidationText != "" {
				if got.Result.Succeeded {
					t.Fatalf("%s blocked request succeeded: %#v", tc.Name, got.Result)
				}
				if !strings.Contains(strings.Join(got.Result.ValidationErrors, " "), tc.ExpectedValidationText) {
					t.Fatalf("%s validation errors = %v, want containing %q", tc.Name, got.Result.ValidationErrors, tc.ExpectedValidationText)
				}
				item, ok := repo.GetTriageItem(tc.Item.TriageItemID)
				if !ok || item.Status != tc.Item.Status {
					t.Fatalf("%s item mutated after block: ok=%v item=%#v", tc.Name, ok, item)
				}
				assertContainsAll(t, got.Result.ForbiddenActions, feedback.ForbiddenActions())
				return
			}

			if !got.Result.Succeeded || len(got.Result.ValidationErrors) != 0 {
				t.Fatalf("%s result = %#v", tc.Name, got.Result)
			}
			if got.Action.NewStatus != tc.ExpectedStatus {
				t.Fatalf("%s status = %s, want %s", tc.Name, got.Action.NewStatus, tc.ExpectedStatus)
			}
			if len(got.Action.FollowUpActionIDs) != tc.ExpectedFollowUpCount {
				t.Fatalf("%s follow-up action ids = %v, want count %d", tc.Name, got.Action.FollowUpActionIDs, tc.ExpectedFollowUpCount)
			}
			if got.Result.AutoApplyAllowed != tc.ExpectedAutoApplyAllowed {
				t.Fatalf("%s auto apply allowed = %v, want %v", tc.Name, got.Result.AutoApplyAllowed, tc.ExpectedAutoApplyAllowed)
			}
			assertContainsAll(t, got.Result.ForbiddenActions, feedback.ForbiddenActions())
		})
	}
}
