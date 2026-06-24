package triage

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/review"
)

func NewItemFromSuggestion(suggestion feedback.RuleSuggestion, now time.Time) (Item, error) {
	source := sourceFromSuggestion(suggestion)
	item := Item{
		TriageItemID:           "triage_" + suggestion.SuggestionID,
		SourceType:             source,
		SourceID:               suggestion.SuggestionID,
		SetupFamily:            suggestion.TargetSetupFamily,
		EventType:              suggestion.TargetEventType,
		Priority:               DefaultPriority(source),
		Status:                 StatusOpen,
		Reason:                 firstNonEmpty(suggestion.Rationale, suggestion.Summary),
		EvidenceRefs:           append([]string{}, suggestion.EvidenceRefs...),
		SuggestedAction:        suggestedActionForSource(source),
		AllowedFollowUpActions: allowedActionsForSource(source),
		ForbiddenActions:       mergeForbiddenActions(suggestion.ForbiddenActions),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		CreatedAt:              now,
		UpdatedAt:              now,
		DueAt:                  now,
	}
	if len(suggestion.SourceLessonIDs) > 0 {
		item.SourceDecisionID = strings.Join(suggestion.SourceLessonIDs, ",")
	}
	return item, ValidateItem(item)
}

func NewItemFromLesson(lesson review.Lesson, now time.Time) (Item, error) {
	source := sourceFromLesson(lesson.LessonType)
	item := Item{
		TriageItemID:           "triage_" + lesson.LessonID,
		SourceType:             source,
		SourceID:               lesson.LessonID,
		SourceDecisionID:       lesson.DecisionID,
		EventID:                lesson.EventID,
		SetupFamily:            lesson.AppliesToSetupFamily,
		EventType:              lesson.AppliesToEventType,
		Priority:               DefaultPriority(source),
		Status:                 StatusOpen,
		Reason:                 lesson.LessonSummary,
		EvidenceRefs:           append([]string{}, lesson.EvidenceRefs...),
		SuggestedAction:        suggestedActionForSource(source),
		AllowedFollowUpActions: allowedActionsForSource(source),
		ForbiddenActions:       mergeForbiddenActions(nil),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		CreatedAt:              now,
		UpdatedAt:              now,
		DueAt:                  now,
	}
	return item, ValidateItem(item)
}

func sourceFromSuggestion(suggestion feedback.RuleSuggestion) SourceType {
	text := strings.ToLower(strings.Join([]string{
		string(suggestion.SuggestionType),
		suggestion.SuggestionID,
		suggestion.Summary,
		suggestion.Rationale,
		strings.Join(suggestion.SourceLessonIDs, " "),
	}, " "))

	switch {
	case strings.Contains(text, "risk") && strings.Contains(text, "strict"):
		return SourceRiskVetoTooStrict
	case strings.Contains(text, "paper") && strings.Contains(text, "failed"):
		return SourcePaperSetupFailed
	case strings.Contains(text, "paper") && strings.Contains(text, "worked"):
		return SourcePaperSetupWorked
	case strings.Contains(text, "missed"):
		return SourceMissedOpportunity
	case strings.Contains(text, "data"):
		return SourceDataQualityReview
	case strings.Contains(text, "watchlist"):
		return SourceWatchlistReview
	case strings.Contains(text, "confirmation"):
		return SourceConfirmationRuleReview
	case strings.Contains(text, "scoring"):
		return SourceScoringReview
	case strings.Contains(text, "research") || strings.Contains(text, "setup_family"):
		return SourceResearchGap
	default:
		return SourceNoTradeRuleReview
	}
}

func sourceFromLesson(lessonType review.LessonType) SourceType {
	switch lessonType {
	case review.LessonMissedOpportunity:
		return SourceMissedOpportunity
	case review.LessonRiskVetoTooStrict:
		return SourceRiskVetoTooStrict
	case review.LessonRiskVetoHelped, review.LessonAvoidedLoss, review.LessonBadCandidateRejected:
		return SourceRiskVetoHelped
	case review.LessonPaperSetupWorked:
		return SourcePaperSetupWorked
	case review.LessonPaperSetupFailed:
		return SourcePaperSetupFailed
	case review.LessonResearchEvidenceInsufficient:
		return SourceResearchGap
	case review.LessonConfirmationRuleTooStrict, review.LessonConfirmationRuleHelped:
		return SourceConfirmationRuleReview
	case review.LessonCorrectNoTrade:
		return SourceNoTradeRuleReview
	default:
		return SourceScoringReview
	}
}

func suggestedActionForSource(source SourceType) string {
	switch source {
	case SourceRiskVetoTooStrict:
		return "Review risk threshold; do not change veto rules automatically."
	case SourcePaperSetupFailed, SourceResearchGap:
		return "Create research task before promotion or rule changes."
	case SourceMissedOpportunity, SourceNoTradeRuleReview:
		return "Review no-trade rule before any strategy change."
	case SourceScoringReview:
		return "Review scoring rule; do not apply changes automatically."
	case SourceConfirmationRuleReview:
		return "Review confirmation rule; keep human approval required."
	case SourceDataQualityReview:
		return "Review data quality before any decision change."
	case SourceWatchlistReview:
		return "Review watchlist manually."
	default:
		return "Manual review required; no automatic rule change allowed."
	}
}

func allowedActionsForSource(source SourceType) []string {
	switch source {
	case SourceResearchGap, SourcePaperSetupFailed:
		return []string{"CREATE_RESEARCH_TASK"}
	case SourceScoringReview:
		return []string{"REVIEW_SCORING_RULE"}
	case SourceRiskVetoTooStrict, SourceRiskVetoHelped:
		return []string{"REVIEW_RISK_THRESHOLD"}
	case SourceConfirmationRuleReview:
		return []string{"REVIEW_CONFIRMATION_RULE"}
	case SourceNoTradeRuleReview, SourceMissedOpportunity:
		return []string{"REVIEW_NO_TRADE_RULE"}
	case SourceWatchlistReview:
		return []string{"REVIEW_WATCHLIST"}
	case SourceDataQualityReview:
		return []string{"REVIEW_DATA_QUALITY"}
	default:
		return []string{"CLOSE_WITH_NO_ACTION"}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "manual triage required"
}

func mergeForbiddenActions(actions []string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, action := range append(actions, feedback.ForbiddenActions()...) {
		if action == "" || seen[action] {
			continue
		}
		seen[action] = true
		merged = append(merged, action)
	}
	return merged
}

func validationError(format string, args ...any) error {
	return fmt.Errorf("triage validation: "+format, args...)
}
