package golden

import (
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/decisioning/research"
)

type researchGoldenCase struct {
	Name                      string                    `json:"name"`
	Evidence                  research.BacktestEvidence `json:"evidence"`
	ExpectedValid             bool                      `json:"expected_valid"`
	ExpectedMaxPromotion      research.PromotionState   `json:"expected_max_promotion_state"`
	ExpectedPromotionDecision research.PromotionState   `json:"expected_promotion_decision"`
	ExpectedErrorsContain     []string                  `json:"expected_errors_contain"`
	ExpectedWarningsContain   []string                  `json:"expected_warnings_contain"`
}

func TestResearchGoldenCases(t *testing.T) {
	files := []string{
		"missing_dataset_hash.json",
		"missing_slippage_costs.json",
		"missing_out_of_sample.json",
		"promising_backtest.json",
		"paper_ready_evidence.json",
		"attempted_live_ready.json",
		"weak_sample_size.json",
		"missing_failure_modes.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc researchGoldenCase
			readJSON(t, filepath.Join("research", file), &tc)

			got := research.ValidateBacktestEvidence(tc.Evidence)

			if got.IsValid != tc.ExpectedValid {
				t.Fatalf("%s is_valid = %v, want %v; errors=%v warnings=%v", tc.Name, got.IsValid, tc.ExpectedValid, got.ValidationErrors, got.ValidationWarnings)
			}
			if got.MaxAllowedPromotionState != tc.ExpectedMaxPromotion {
				t.Fatalf("%s max promotion = %s, want %s", tc.Name, got.MaxAllowedPromotionState, tc.ExpectedMaxPromotion)
			}
			if got.PromotionDecision != tc.ExpectedPromotionDecision {
				t.Fatalf("%s promotion decision = %s, want %s", tc.Name, got.PromotionDecision, tc.ExpectedPromotionDecision)
			}
			assertTextContainsAll(t, got.ValidationErrors, tc.ExpectedErrorsContain)
			assertTextContainsAll(t, got.ValidationWarnings, tc.ExpectedWarningsContain)
			if got.PromotionDecision == "LIVE_READY" || got.MaxAllowedPromotionState == "LIVE_READY" {
				t.Fatalf("%s must not return LIVE_READY", tc.Name)
			}
		})
	}
}

func assertTextContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	for _, expected := range want {
		found := false
		for _, actual := range got {
			if strings.Contains(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing text containing %q in %v", expected, got)
		}
	}
}
