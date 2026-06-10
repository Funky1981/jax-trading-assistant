package macroevents

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AnalysisCaseStudyRecord struct {
	ID                    string
	MacroEventID          string
	Symbol                string
	EventType             string
	PlaybookKey           string
	TechnicalSnapshotID   string
	FundamentalSnapshotID string
	AnalystDecisionID     string
	ReviewID              string
	Decision              AnalystDecision
	ExpectedOutcome       string
	ActualOutcome         string
	OutcomeR              *float64
	SurpriseBucket        string
	TechnicalSetup        string
	MarketRegime          string
	WhatWorked            []string
	WhatFailed            []string
	Lesson                string
	Tags                  []string
	CreatedAt             time.Time
	ReviewedAt            *time.Time
}

type AnalysisCaseStudyCreateInput struct {
	MacroEventID          string
	Symbol                string
	EventType             string
	PlaybookKey           string
	TechnicalSnapshotID   string
	FundamentalSnapshotID string
	AnalystDecisionID     string
	ReviewID              string
	Decision              AnalystDecision
	ExpectedOutcome       string
	SurpriseBucket        string
	TechnicalSetup        string
	MarketRegime          string
	Tags                  []string
}

type AnalysisCaseStudyOutcomeUpdate struct {
	CaseStudyID   string
	ActualOutcome string
	OutcomeR      *float64
	WhatWorked    []string
	WhatFailed    []string
	Lesson        string
}

type AnalystFeedbackRecord struct {
	ID             string
	CaseStudyID    string
	FeedbackSource string
	Rating         string
	Comment        string
	CreatedAt      time.Time
}

type SimilarCaseQuery struct {
	Symbol         string
	EventType      string
	PlaybookKey    string
	SurpriseBucket string
	TechnicalSetup string
	MarketRegime   string
	Limit          int
}

type analysisMemoryStore interface {
	SaveAnalysisCaseStudy(ctx context.Context, study AnalysisCaseStudyRecord) (AnalysisCaseStudyRecord, error)
	UpdateAnalysisCaseStudyOutcome(ctx context.Context, update AnalysisCaseStudyOutcomeUpdate) (AnalysisCaseStudyRecord, error)
	SaveAnalystFeedback(ctx context.Context, feedback AnalystFeedbackRecord) (AnalystFeedbackRecord, error)
	FindSimilarAnalysisCaseStudies(ctx context.Context, query SimilarCaseQuery) ([]AnalysisCaseStudyRecord, error)
}

type AnalysisMemoryService struct {
	store analysisMemoryStore
}

func NewAnalysisMemoryService(store analysisMemoryStore) *AnalysisMemoryService {
	return &AnalysisMemoryService{store: store}
}

func (s *AnalysisMemoryService) CreateCaseStudy(ctx context.Context, input AnalysisCaseStudyCreateInput) (AnalysisCaseStudyRecord, error) {
	record, err := normalizeCaseStudyInput(input)
	if err != nil {
		return AnalysisCaseStudyRecord{}, err
	}
	if s.store == nil {
		return record, nil
	}
	return s.store.SaveAnalysisCaseStudy(ctx, record)
}

func (s *AnalysisMemoryService) AttachOutcome(ctx context.Context, update AnalysisCaseStudyOutcomeUpdate) (AnalysisCaseStudyRecord, error) {
	update.CaseStudyID = strings.TrimSpace(update.CaseStudyID)
	if update.CaseStudyID == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("case study id is required")
	}
	update.ActualOutcome = strings.TrimSpace(update.ActualOutcome)
	update.Lesson = strings.TrimSpace(update.Lesson)
	if s.store == nil {
		now := time.Now().UTC()
		return AnalysisCaseStudyRecord{
			ID:            update.CaseStudyID,
			ActualOutcome: update.ActualOutcome,
			OutcomeR:      update.OutcomeR,
			WhatWorked:    append([]string(nil), update.WhatWorked...),
			WhatFailed:    append([]string(nil), update.WhatFailed...),
			Lesson:        update.Lesson,
			ReviewedAt:    &now,
		}, nil
	}
	return s.store.UpdateAnalysisCaseStudyOutcome(ctx, update)
}

func (s *AnalysisMemoryService) AddFeedback(ctx context.Context, feedback AnalystFeedbackRecord) (AnalystFeedbackRecord, error) {
	feedback.CaseStudyID = strings.TrimSpace(feedback.CaseStudyID)
	feedback.FeedbackSource = strings.TrimSpace(feedback.FeedbackSource)
	feedback.Rating = strings.TrimSpace(feedback.Rating)
	feedback.Comment = strings.TrimSpace(feedback.Comment)
	if feedback.CaseStudyID == "" {
		return AnalystFeedbackRecord{}, fmt.Errorf("case study id is required")
	}
	if feedback.FeedbackSource == "" {
		return AnalystFeedbackRecord{}, fmt.Errorf("feedback source is required")
	}
	if feedback.Rating == "" {
		return AnalystFeedbackRecord{}, fmt.Errorf("rating is required")
	}
	if feedback.Comment == "" {
		return AnalystFeedbackRecord{}, fmt.Errorf("comment is required")
	}
	feedback.CreatedAt = time.Now().UTC()
	if s.store == nil {
		return feedback, nil
	}
	return s.store.SaveAnalystFeedback(ctx, feedback)
}

func (s *AnalysisMemoryService) FindSimilarCases(ctx context.Context, query SimilarCaseQuery) ([]AnalysisCaseStudyRecord, error) {
	query.Symbol = strings.ToUpper(strings.TrimSpace(query.Symbol))
	query.EventType = strings.TrimSpace(query.EventType)
	query.PlaybookKey = strings.TrimSpace(query.PlaybookKey)
	query.SurpriseBucket = strings.TrimSpace(query.SurpriseBucket)
	query.TechnicalSetup = strings.TrimSpace(query.TechnicalSetup)
	query.MarketRegime = strings.TrimSpace(query.MarketRegime)
	if query.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if query.Limit <= 0 {
		query.Limit = 5
	}
	if query.Limit > 20 {
		query.Limit = 20
	}
	if s.store == nil {
		return []AnalysisCaseStudyRecord{}, nil
	}
	return s.store.FindSimilarAnalysisCaseStudies(ctx, query)
}

func normalizeCaseStudyInput(input AnalysisCaseStudyCreateInput) (AnalysisCaseStudyRecord, error) {
	record := AnalysisCaseStudyRecord{
		MacroEventID:          strings.TrimSpace(input.MacroEventID),
		Symbol:                strings.ToUpper(strings.TrimSpace(input.Symbol)),
		EventType:             strings.TrimSpace(input.EventType),
		PlaybookKey:           strings.TrimSpace(input.PlaybookKey),
		TechnicalSnapshotID:   strings.TrimSpace(input.TechnicalSnapshotID),
		FundamentalSnapshotID: strings.TrimSpace(input.FundamentalSnapshotID),
		AnalystDecisionID:     strings.TrimSpace(input.AnalystDecisionID),
		ReviewID:              strings.TrimSpace(input.ReviewID),
		Decision:              input.Decision,
		ExpectedOutcome:       strings.TrimSpace(input.ExpectedOutcome),
		SurpriseBucket:        strings.TrimSpace(input.SurpriseBucket),
		TechnicalSetup:        strings.TrimSpace(input.TechnicalSetup),
		MarketRegime:          strings.TrimSpace(input.MarketRegime),
		Tags:                  append([]string(nil), input.Tags...),
		WhatWorked:            []string{},
		WhatFailed:            []string{},
		CreatedAt:             time.Now().UTC(),
	}
	if record.Symbol == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("symbol is required")
	}
	if record.EventType == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("event type is required")
	}
	if record.PlaybookKey == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("playbook key is required")
	}
	if record.Decision == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("decision is required")
	}
	if record.ExpectedOutcome == "" {
		return AnalysisCaseStudyRecord{}, fmt.Errorf("expected outcome is required")
	}
	return record, nil
}
