package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type operationsPersistenceGoldenCase struct {
	Name                     string                        `json:"name"`
	Items                    []triage.Item                 `json:"items"`
	Decisions                []operations.FeedbackDecision `json:"decisions"`
	Actions                  []operations.FollowUpAction   `json:"actions"`
	ExpectedOpenIDs          []string                      `json:"expected_open_ids"`
	ExpectedHighPriorityIDs  []string                      `json:"expected_high_priority_ids"`
	ExpectedAuditActions     []operations.AuditAction      `json:"expected_audit_actions"`
	ExpectedForbiddenActions []string                      `json:"expected_forbidden_actions"`
}

func TestOperationsPersistenceGoldenCases(t *testing.T) {
	files := []string{"persist_review_operations.json"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc operationsPersistenceGoldenCase
			readJSON(t, filepath.Join("operations_persistence", file), &tc)

			repo := operations.NewMemoryRepository()
			for _, item := range tc.Items {
				if err := repo.SaveTriageItem(item); err != nil {
					t.Fatalf("%s SaveTriageItem(%s): %v", tc.Name, item.TriageItemID, err)
				}
			}
			for _, decision := range tc.Decisions {
				if err := repo.SaveHumanFeedbackDecision(decision); err != nil {
					t.Fatalf("%s SaveHumanFeedbackDecision(%s): %v", tc.Name, decision.FeedbackDecisionID, err)
				}
			}
			for _, action := range tc.Actions {
				if err := repo.SaveFollowUpAction(action); err != nil {
					t.Fatalf("%s SaveFollowUpAction(%s): %v", tc.Name, action.ActionID, err)
				}
			}

			assertGoldenTriageIDs(t, repo.ListOpenTriageItems(), tc.ExpectedOpenIDs)
			assertGoldenTriageIDs(t, repo.ListHighPriorityTriageItems(), tc.ExpectedHighPriorityIDs)
			for _, item := range repo.ListTriageItems() {
				if item.AutoApplyAllowed {
					t.Fatalf("%s item %s auto apply allowed = true", tc.Name, item.TriageItemID)
				}
				if !item.RequiresHumanApproval {
					t.Fatalf("%s item %s requires human approval = false", tc.Name, item.TriageItemID)
				}
				assertContainsAll(t, item.ForbiddenActions, tc.ExpectedForbiddenActions)
			}
			assertGoldenAuditActions(t, repo.ListOperationAuditRecords(), tc.ExpectedAuditActions)
		})
	}
}

func assertGoldenTriageIDs(t *testing.T, items []triage.Item, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("items = %d, want %d: %#v", len(items), len(want), items)
	}
	for i, item := range items {
		if item.TriageItemID != want[i] {
			t.Fatalf("item[%d] = %s, want %s", i, item.TriageItemID, want[i])
		}
	}
}

func assertGoldenAuditActions(t *testing.T, records []operations.OperationAuditRecord, want []operations.AuditAction) {
	t.Helper()
	seen := map[operations.AuditAction]bool{}
	for _, record := range records {
		seen[record.Action] = true
	}
	for _, action := range want {
		if !seen[action] {
			t.Fatalf("missing audit action %s in %#v", action, records)
		}
	}
}
