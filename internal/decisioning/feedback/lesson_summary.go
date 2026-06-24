package feedback

import (
	"fmt"

	"jax-trading-assistant/internal/decisioning/review"
)

func applyLesson(report *FeedbackReport, lesson review.Lesson) {
	setupFamily := lesson.AppliesToSetupFamily
	if setupFamily == "" {
		setupFamily = "unspecified setup family"
	}
	eventType := lesson.AppliesToEventType
	if eventType == "" {
		eventType = "unspecified event type"
	}

	switch lesson.LessonType {
	case review.LessonCorrectNoTrade:
		report.NoTradeFindings = append(report.NoTradeFindings, fmt.Sprintf("No-trade rule helped for %s.", setupFamily))
		report.SetupFamilyFindings = append(report.SetupFamilyFindings, fmt.Sprintf("%s rejection evidence strengthened.", setupFamily))
	case review.LessonMissedOpportunity:
		report.NoTradeFindings = append(report.NoTradeFindings, fmt.Sprintf("Missed opportunity found for %s; review confirmation before changing rules.", setupFamily))
		report.EventTypeFindings = append(report.EventTypeFindings, fmt.Sprintf("%s produced a missed opportunity signal.", eventType))
	case review.LessonAvoidedLoss, review.LessonBadCandidateRejected:
		report.KeyFindings = append(report.KeyFindings, fmt.Sprintf("Avoided loss recorded for %s.", setupFamily))
	case review.LessonRiskVetoHelped:
		report.RiskVetoFindings = append(report.RiskVetoFindings, fmt.Sprintf("Risk Veto avoided bad setup for %s.", setupFamily))
	case review.LessonRiskVetoTooStrict:
		report.RiskVetoFindings = append(report.RiskVetoFindings, fmt.Sprintf("Risk Veto may have been too strict for %s.", setupFamily))
	case review.LessonPaperSetupWorked:
		report.PaperOutcomeFindings = append(report.PaperOutcomeFindings, fmt.Sprintf("%s paper setup is promising and worth further research.", setupFamily))
		report.SetupFamilyFindings = append(report.SetupFamilyFindings, fmt.Sprintf("%s showed positive paper outcome evidence.", setupFamily))
	case review.LessonPaperSetupFailed:
		report.PaperOutcomeFindings = append(report.PaperOutcomeFindings, fmt.Sprintf("%s paper setup showed weakness and needs research before promotion.", setupFamily))
		report.ResearchGaps = append(report.ResearchGaps, fmt.Sprintf("%s needs research after failed paper outcome.", setupFamily))
	case review.LessonResearchEvidenceInsufficient:
		report.ResearchGaps = append(report.ResearchGaps, fmt.Sprintf("%s has weak or insufficient research evidence.", setupFamily))
	}
}
