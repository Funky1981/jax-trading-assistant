package macroevents

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type fakeAnalysisMemoryStore struct {
	caseStudies map[string]AnalysisCaseStudyRecord
	feedback    map[string]AnalystFeedbackRecord
	nextCaseID  int
	nextFeedID  int
}

func newFakeAnalysisMemoryStore() *fakeAnalysisMemoryStore {
	return &fakeAnalysisMemoryStore{
		caseStudies: map[string]AnalysisCaseStudyRecord{},
		feedback:    map[string]AnalystFeedbackRecord{},
	}
}

func (s *fakeAnalysisMemoryStore) SaveAnalysisCaseStudy(_ context.Context, study AnalysisCaseStudyRecord) (AnalysisCaseStudyRecord, error) {
	s.nextCaseID++
	study.ID = fmt.Sprintf("cs-%d", s.nextCaseID)
	s.caseStudies[study.ID] = study
	return study, nil
}

func (s *fakeAnalysisMemoryStore) UpdateAnalysisCaseStudyOutcome(_ context.Context, update AnalysisCaseStudyOutcomeUpdate) (AnalysisCaseStudyRecord, error) {
	study, ok := s.caseStudies[update.CaseStudyID]
	if !ok {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("case study not found")
	}
	now := time.Now().UTC()
	study.ActualOutcome = update.ActualOutcome
	study.OutcomeR = update.OutcomeR
	study.WhatWorked = append([]string(nil), update.WhatWorked...)
	study.WhatFailed = append([]string(nil), update.WhatFailed...)
	study.Lesson = update.Lesson
	study.ReviewedAt = &now
	s.caseStudies[study.ID] = study
	return study, nil
}

func (s *fakeAnalysisMemoryStore) SaveAnalystFeedback(_ context.Context, feedback AnalystFeedbackRecord) (AnalystFeedbackRecord, error) {
	s.nextFeedID++
	feedback.ID = fmt.Sprintf("fb-%d", s.nextFeedID)
	s.feedback[feedback.ID] = feedback
	return feedback, nil
}

func (s *fakeAnalysisMemoryStore) FindSimilarAnalysisCaseStudies(_ context.Context, query SimilarCaseQuery) ([]AnalysisCaseStudyRecord, error) {
	items := make([]AnalysisCaseStudyRecord, 0, len(s.caseStudies))
	for _, study := range s.caseStudies {
		if !strings.EqualFold(study.Symbol, query.Symbol) {
			continue
		}
		score := 0
		if query.EventType != "" && strings.EqualFold(study.EventType, query.EventType) {
			score += 3
		}
		if query.PlaybookKey != "" && strings.EqualFold(study.PlaybookKey, query.PlaybookKey) {
			score += 3
		}
		if query.SurpriseBucket != "" && strings.EqualFold(study.SurpriseBucket, query.SurpriseBucket) {
			score++
		}
		if query.TechnicalSetup != "" && strings.EqualFold(study.TechnicalSetup, query.TechnicalSetup) {
			score++
		}
		if query.MarketRegime != "" && strings.EqualFold(study.MarketRegime, query.MarketRegime) {
			score++
		}
		if score > 0 || (query.EventType == "" && query.PlaybookKey == "") {
			study.Tags = append([]string{fmt.Sprintf("similarity:%d", score)}, study.Tags...)
			items = append(items, study)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	return items, nil
}
