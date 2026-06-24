package replay

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/review"
)

const (
	PaperOutcomeWorked = "WORKED"
	PaperOutcomeFailed = "FAILED"
)

type Record struct {
	RecordID       string             `json:"record_id"`
	DecisionID     string             `json:"decision_id"`
	EventID        string             `json:"event_id"`
	Asset          string             `json:"asset"`
	EventType      string             `json:"event_type"`
	SetupFamily    string             `json:"setup_family"`
	FinalDecision  core.DecisionValue `json:"final_decision"`
	RejectedByRisk bool               `json:"rejected_by_risk"`
	PaperApproved  bool               `json:"paper_approved"`
	PaperOutcome   string             `json:"paper_outcome"`
	ResearchStatus string             `json:"research_status"`
	Lessons        []review.Lesson    `json:"lessons"`
	CreatedAt      time.Time          `json:"created_at"`
}

func Run(input ReplayInput) ReplayResult {
	result := ReplayResult{
		ReplayID:  input.ReplayID,
		CreatedAt: input.CreatedAt,
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	if result.ReplayID == "" {
		result.Errors = append(result.Errors, "replay_id is required")
	}

	includePaper := includeDefault(input.IncludePaperOutcomes)
	includeLessons := includeDefault(input.IncludeLessons)
	for _, record := range input.Records {
		if !recordMatches(input, record) {
			continue
		}
		result.RecordsProcessed++
		countDecision(&result, record)
		countRecordSignals(&result, record, includePaper, includeLessons)
	}
	return result
}

func countDecision(result *ReplayResult, record Record) {
	switch record.FinalDecision {
	case core.DecisionNoTrade:
		result.NoTradeCount++
	case core.DecisionWatch:
		result.WatchCount++
	case core.DecisionSetupForming:
		result.SetupFormingCount++
	case core.DecisionTradeCandidate, core.DecisionPaperApprovalRequired:
		result.TradeCandidateCount++
	}
	if record.RejectedByRisk || record.FinalDecision == core.DecisionRejectedByRisk {
		result.RejectedByRiskCount++
	}
	if record.PaperApproved {
		result.PaperApprovedCount++
	}
}

func countRecordSignals(result *ReplayResult, record Record, includePaper bool, includeLessons bool) {
	paperWorked := false
	paperFailed := false
	if includePaper {
		paperWorked, paperFailed = countPaperOutcome(record)
	}
	if includeLessons {
		lessonWorked, lessonFailed := countLessons(result, record.Lessons)
		paperWorked = paperWorked || lessonWorked
		paperFailed = paperFailed || lessonFailed
	}
	if paperWorked {
		result.PaperSetupWorkedCount++
	}
	if paperFailed {
		result.PaperSetupFailedCount++
	}
}

func countPaperOutcome(record Record) (bool, bool) {
	switch record.PaperOutcome {
	case PaperOutcomeWorked:
		return true, false
	case PaperOutcomeFailed:
		return false, true
	}
	return false, false
}

func countLessons(result *ReplayResult, lessons []review.Lesson) (bool, bool) {
	paperWorked := false
	paperFailed := false
	for _, lesson := range lessons {
		result.LessonsGenerated++
		switch lesson.LessonType {
		case review.LessonCorrectNoTrade:
			result.CorrectNoTradeCount++
		case review.LessonMissedOpportunity:
			result.MissedOpportunityCount++
		case review.LessonAvoidedLoss, review.LessonBadCandidateRejected:
			result.AvoidedLossCount++
		case review.LessonRiskVetoHelped:
			result.RiskVetoHelpedCount++
		case review.LessonRiskVetoTooStrict:
			result.RiskVetoTooStrictCount++
		case review.LessonPaperSetupWorked:
			paperWorked = true
		case review.LessonPaperSetupFailed:
			paperFailed = true
		}
	}
	return paperWorked, paperFailed
}
