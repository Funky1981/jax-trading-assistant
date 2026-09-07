package evaluation

import (
	"strings"
	"testing"
	"time"
)

func TestBacktestAdapterAssessmentSelectsBoundedCustomEventReplay(t *testing.T) {
	custom, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "Jax custom event replay", Version: "v1", Language: "Go", License: "Jax proprietary", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low: existing Go runtime", IntegrationBurden: "low: typed package boundary", DecisionStatus: AdapterSelected, AssessmentNotes: "Best fit for frozen evidence and no new runtime."})
	if err != nil {
		t.Fatal(err)
	}
	lean, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "LEAN", Version: "reference-only", Language: "C#", License: "Apache-2.0; compatibility review required", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "high: additional runtime and data model", IntegrationBurden: "high: adapter and process boundary", DecisionStatus: AdapterRejected, AssessmentNotes: "Conceptual reference; not adopted in this bounded package."})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := NewBacktestAdapterAssessment(time.Date(2026, 9, 7, 15, 0, 0, 0, time.UTC), "Phase 07 historical candidate evaluation", []BacktestAdapterCandidate{custom, lean}, custom.ID, "Prefer the existing deterministic event model and typed Go boundary; defer new runtime adoption until evidence requires it.")
	if err != nil {
		t.Fatal(err)
	}
	if err := assessment.Validate(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(assessment.ID, "assessment_") || assessment.ExecutionAuthority != "NONE" || assessment.InferenceUsed {
		t.Fatalf("assessment crossed a safety boundary: %#v", assessment)
	}
}

func TestBacktestAdapterAssessmentRejectsSelectedStatusAmbiguity(t *testing.T) {
	candidate, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "custom", Version: "v1", Language: "Go", License: "Jax proprietary", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low", IntegrationBurden: "low", DecisionStatus: AdapterSelected, AssessmentNotes: "selected"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "other", Version: "v1", Language: "Go", License: "MIT", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low", IntegrationBurden: "low", DecisionStatus: AdapterSelected, AssessmentNotes: "candidate"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewBacktestAdapterAssessment(time.Now().UTC(), "scope", []BacktestAdapterCandidate{candidate, other}, candidate.ID, "rationale"); err == nil {
		t.Fatal("expected multiple selected adapters to be rejected")
	}
}

func TestBacktestAdapterAssessmentRejectsNonUTCOrMissingRationale(t *testing.T) {
	candidate, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "custom", Version: "v1", Language: "Go", License: "Jax proprietary", EventDriven: true, PointInTimeSafe: true, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low", IntegrationBurden: "low", DecisionStatus: AdapterSelected, AssessmentNotes: "selected"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewBacktestAdapterAssessment(time.Date(2026, 9, 7, 15, 0, 0, 0, time.FixedZone("BST", 3600)), "scope", []BacktestAdapterCandidate{candidate}, candidate.ID, "rationale"); err == nil {
		t.Fatal("expected non-UTC evaluation timestamp to be rejected")
	}
	if _, err := NewBacktestAdapterAssessment(time.Now().UTC(), "scope", []BacktestAdapterCandidate{candidate}, candidate.ID, ""); err == nil {
		t.Fatal("expected missing selection rationale to be rejected")
	}
}

func TestBacktestAdapterAssessmentRejectsUnsafeSelectedAdapter(t *testing.T) {
	candidate, err := NewBacktestAdapterCandidate(BacktestAdapterCandidate{Name: "unsafe", Version: "v1", Language: "Go", License: "MIT", EventDriven: true, PointInTimeSafe: false, TransactionCostSupport: true, Reproducible: true, CustomDataSupport: true, OperationalBurden: "low", IntegrationBurden: "low", DecisionStatus: AdapterSelected, AssessmentNotes: "missing point-in-time guarantee"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewBacktestAdapterAssessment(time.Now().UTC(), "scope", []BacktestAdapterCandidate{candidate}, candidate.ID, "rationale"); err == nil {
		t.Fatal("expected unsafe selected adapter to be rejected")
	}
}
