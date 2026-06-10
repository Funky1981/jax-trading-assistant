package macroevents

import (
	"context"
	"testing"
)

func TestAnalysisMemoryServiceCaseStudyCreatedAfterAnalystDecision(t *testing.T) {
	store := newFakeAnalysisMemoryStore()
	service := NewAnalysisMemoryService(store)

	record, err := service.CreateCaseStudy(context.Background(), AnalysisCaseStudyCreateInput{
		MacroEventID:          "macro-1",
		Symbol:                "QQQ",
		EventType:             "cpi",
		PlaybookKey:           "hot_cpi_bearish",
		TechnicalSnapshotID:   "ta-1",
		FundamentalSnapshotID: "fa-1",
		AnalystDecisionID:     "ad-1",
		ReviewID:              "mar-1",
		Decision:              AnalystDecisionCandidateRejected,
		ExpectedOutcome:       "expect bearish continuation if vwap fails",
		SurpriseBucket:        "hot",
		TechnicalSetup:        "vwap_reject",
		MarketRegime:          "risk_off",
		Tags:                  []string{"cpi", "qqq", "bearish"},
	})
	if err != nil {
		t.Fatalf("CreateCaseStudy returned error: %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected case study id")
	}
	if len(store.caseStudies) != 1 {
		t.Fatalf("case studies = %d, want 1", len(store.caseStudies))
	}
}

func TestAnalysisMemoryServiceOperatorRejectionCreatesFeedbackRecord(t *testing.T) {
	store := newFakeAnalysisMemoryStore()
	service := NewAnalysisMemoryService(store)

	feedback, err := service.AddFeedback(context.Background(), AnalystFeedbackRecord{
		CaseStudyID:    "cs-1",
		FeedbackSource: "operator",
		Rating:         "rejected",
		Comment:        "operator rejected due to event overlap",
	})
	if err != nil {
		t.Fatalf("AddFeedback returned error: %v", err)
	}
	if feedback.ID == "" {
		t.Fatal("expected feedback id")
	}
	if len(store.feedback) != 1 {
		t.Fatalf("feedback records = %d, want 1", len(store.feedback))
	}
}

func TestAnalysisMemoryServicePaperOutcomeUpdatesCaseStudy(t *testing.T) {
	store := newFakeAnalysisMemoryStore()
	service := NewAnalysisMemoryService(store)

	seed, err := service.CreateCaseStudy(context.Background(), AnalysisCaseStudyCreateInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		EventType:       "cpi",
		PlaybookKey:     "hot_cpi_bearish",
		Decision:        AnalystDecisionCandidateAllowed,
		ExpectedOutcome: "expected continuation",
	})
	if err != nil {
		t.Fatalf("CreateCaseStudy returned error: %v", err)
	}

	outcomeR := 1.25
	updated, err := service.AttachOutcome(context.Background(), AnalysisCaseStudyOutcomeUpdate{
		CaseStudyID:   seed.ID,
		ActualOutcome: "moved as expected then mean reverted",
		OutcomeR:      &outcomeR,
		WhatWorked:    []string{"playbook alignment"},
		WhatFailed:    []string{"late entry"},
		Lesson:        "wait for second confirmation candle",
	})
	if err != nil {
		t.Fatalf("AttachOutcome returned error: %v", err)
	}
	if updated.ActualOutcome == "" {
		t.Fatal("expected actual outcome to be set")
	}
	if updated.OutcomeR == nil || *updated.OutcomeR != outcomeR {
		t.Fatalf("outcome_r = %#v, want %v", updated.OutcomeR, outcomeR)
	}
}

func TestAnalysisMemoryServiceSimilarCaseRetrievalReturnsMatchingFields(t *testing.T) {
	store := newFakeAnalysisMemoryStore()
	service := NewAnalysisMemoryService(store)

	_, _ = service.CreateCaseStudy(context.Background(), AnalysisCaseStudyCreateInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		EventType:       "cpi",
		PlaybookKey:     "hot_cpi_bearish",
		Decision:        AnalystDecisionCandidateAllowed,
		ExpectedOutcome: "expected continuation",
		SurpriseBucket:  "hot",
		TechnicalSetup:  "vwap_reject",
		MarketRegime:    "risk_off",
	})
	_, _ = service.CreateCaseStudy(context.Background(), AnalysisCaseStudyCreateInput{
		MacroEventID:    "macro-2",
		Symbol:          "QQQ",
		EventType:       "nfp",
		PlaybookKey:     "jobs_hot_bearish",
		Decision:        AnalystDecisionWatchOnly,
		ExpectedOutcome: "watch only",
	})

	cases, err := service.FindSimilarCases(context.Background(), SimilarCaseQuery{
		Symbol:         "QQQ",
		EventType:      "cpi",
		PlaybookKey:    "hot_cpi_bearish",
		SurpriseBucket: "hot",
		TechnicalSetup: "vwap_reject",
		MarketRegime:   "risk_off",
		Limit:          3,
	})
	if err != nil {
		t.Fatalf("FindSimilarCases returned error: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("expected at least one similar case")
	}
	if cases[0].EventType != "cpi" || cases[0].PlaybookKey != "hot_cpi_bearish" {
		t.Fatalf("top case mismatch: event=%q playbook=%q", cases[0].EventType, cases[0].PlaybookKey)
	}
}

func TestAnalysisMemoryServiceReflectionCannotModifyImmutableDecision(t *testing.T) {
	store := newFakeAnalysisMemoryStore()
	service := NewAnalysisMemoryService(store)

	seed, err := service.CreateCaseStudy(context.Background(), AnalysisCaseStudyCreateInput{
		MacroEventID:    "macro-1",
		Symbol:          "QQQ",
		EventType:       "cpi",
		PlaybookKey:     "hot_cpi_bearish",
		Decision:        AnalystDecisionCandidateRejected,
		ExpectedOutcome: "expected rejection",
	})
	if err != nil {
		t.Fatalf("CreateCaseStudy returned error: %v", err)
	}
	originalDecision := seed.Decision

	_, err = service.AttachOutcome(context.Background(), AnalysisCaseStudyOutcomeUpdate{
		CaseStudyID:   seed.ID,
		ActualOutcome: "event faded; rejection was correct",
		Lesson:        "keep rejecting without confirmation",
	})
	if err != nil {
		t.Fatalf("AttachOutcome returned error: %v", err)
	}

	stored := store.caseStudies[seed.ID]
	if stored.Decision != originalDecision {
		t.Fatalf("decision mutated from %q to %q", originalDecision, stored.Decision)
	}
}
