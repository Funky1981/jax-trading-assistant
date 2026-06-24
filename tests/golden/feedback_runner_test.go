package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/feedback"
)

type feedbackGoldenCase struct {
	Name                          string                    `json:"name"`
	Input                         feedback.ReportInput      `json:"input"`
	ExpectedNoTradeFindings       []string                  `json:"expected_no_trade_findings_contain"`
	ExpectedRiskVetoFindings      []string                  `json:"expected_risk_veto_findings_contain"`
	ExpectedPaperOutcomeFindings  []string                  `json:"expected_paper_outcome_findings_contain"`
	ExpectedResearchGaps          []string                  `json:"expected_research_gaps_contain"`
	ExpectedSuggestionTypes       []feedback.SuggestionType `json:"expected_suggestion_types"`
	ExpectedRequiresHumanApproval bool                      `json:"expected_requires_human_approval"`
	ExpectedForbiddenActions      []string                  `json:"expected_forbidden_actions"`
	ExpectedErrorsContain         []string                  `json:"expected_errors_contain"`
}

func TestFeedbackGoldenCases(t *testing.T) {
	files := []string{"phase_10_feedback_report.json", "live_promotion_blocked.json"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc feedbackGoldenCase
			readJSON(t, filepath.Join("feedback", file), &tc)

			got := feedback.BuildReport(tc.Input)
			if got.RequiresHumanApproval != tc.ExpectedRequiresHumanApproval {
				t.Fatalf("%s requires human approval = %v, want %v", tc.Name, got.RequiresHumanApproval, tc.ExpectedRequiresHumanApproval)
			}
			assertTextContainsAll(t, got.NoTradeFindings, tc.ExpectedNoTradeFindings)
			assertTextContainsAll(t, got.RiskVetoFindings, tc.ExpectedRiskVetoFindings)
			assertTextContainsAll(t, got.PaperOutcomeFindings, tc.ExpectedPaperOutcomeFindings)
			assertTextContainsAll(t, got.ResearchGaps, tc.ExpectedResearchGaps)
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			assertTextContainsAll(t, got.Errors, tc.ExpectedErrorsContain)
			for _, suggestion := range got.SuggestedRuleChanges {
				if !suggestion.RequiresHumanApproval {
					t.Fatalf("%s suggestion %s requires human approval = false", tc.Name, suggestion.SuggestionID)
				}
				if suggestion.AutoApplyAllowed {
					t.Fatalf("%s suggestion %s auto apply allowed = true", tc.Name, suggestion.SuggestionID)
				}
				assertContainsAll(t, suggestion.ForbiddenActions, tc.ExpectedForbiddenActions)
			}
			for _, want := range tc.ExpectedSuggestionTypes {
				if !hasSuggestionType(got.SuggestedRuleChanges, want) {
					t.Fatalf("%s missing suggestion type %s in %#v", tc.Name, want, got.SuggestedRuleChanges)
				}
			}
		})
	}
}

func hasSuggestionType(suggestions []feedback.RuleSuggestion, want feedback.SuggestionType) bool {
	for _, suggestion := range suggestions {
		if suggestion.SuggestionType == want {
			return true
		}
	}
	return false
}
