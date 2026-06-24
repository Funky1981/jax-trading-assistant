package golden

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
	"jax-trading-assistant/internal/decisioning/triage"
	"jax-trading-assistant/internal/decisioning/workflow"
)

type exportGoldenCase struct {
	Name                     string                    `json:"name"`
	BatchID                  string                    `json:"batch_id"`
	GeneratedAt              time.Time                 `json:"generated_at"`
	AsOf                     time.Time                 `json:"as_of"`
	ExportID                 string                    `json:"export_id"`
	ExportType               reviewexport.ExportType   `json:"export_type"`
	Format                   reviewexport.ExportFormat `json:"format"`
	Items                    []triage.Item             `json:"items"`
	ExpectedContentContains  []string                  `json:"expected_content_contains"`
	ExpectedForbiddenActions []string                  `json:"expected_forbidden_actions"`
}

func TestExportGoldenCases(t *testing.T) {
	files := []string{
		"review_batch_json.json",
		"review_batch_markdown.json",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc exportGoldenCase
			readJSON(t, filepath.Join("export", file), &tc)

			batch, err := workflow.BuildReviewBatch(tc.Items, nil, workflow.BatchOptions{
				BatchID:         tc.BatchID,
				GeneratedAt:     tc.GeneratedAt,
				AsOf:            tc.AsOf,
				SelectionReason: workflow.SelectionActiveReview,
			})
			if err != nil {
				t.Fatalf("%s BuildReviewBatch returned error: %v", tc.Name, err)
			}
			got, err := reviewexport.ExportReviewBatch(batch, reviewexport.ExportOptions{
				ExportID:    tc.ExportID,
				ExportType:  tc.ExportType,
				Format:      tc.Format,
				GeneratedAt: tc.GeneratedAt,
			})
			if err != nil {
				t.Fatalf("%s ExportReviewBatch returned error: %v", tc.Name, err)
			}

			if !got.ReadOnly || got.AutoApplyAllowed {
				t.Fatalf("%s safety flags = read_only %v auto_apply_allowed %v", tc.Name, got.ReadOnly, got.AutoApplyAllowed)
			}
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			for _, want := range tc.ExpectedContentContains {
				if !strings.Contains(got.Content, want) {
					t.Fatalf("%s content missing %q:\n%s", tc.Name, want, got.Content)
				}
			}
			if strings.Contains(strings.ToLower(got.Content), "chain-of-thought") ||
				strings.Contains(strings.ToLower(got.Content), "hidden reasoning") {
				t.Fatalf("%s leaked hidden reasoning marker:\n%s", tc.Name, got.Content)
			}
		})
	}
}
