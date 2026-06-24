package feedback

import (
	"fmt"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/review"
)

type SuggestionType string

const (
	SuggestionScoringReview          SuggestionType = "SCORING_REVIEW"
	SuggestionRiskThresholdReview    SuggestionType = "RISK_THRESHOLD_REVIEW"
	SuggestionConfirmationRuleReview SuggestionType = "CONFIRMATION_RULE_REVIEW"
	SuggestionSetupFamilyResearch    SuggestionType = "SETUP_FAMILY_RESEARCH"
	SuggestionNoTradeRuleReview      SuggestionType = "NO_TRADE_RULE_REVIEW"
	SuggestionWatchlistReview        SuggestionType = "WATCHLIST_REVIEW"
	SuggestionDataQualityReview      SuggestionType = "DATA_QUALITY_REVIEW"
)

type RuleSuggestion struct {
	SuggestionID          string         `json:"suggestion_id"`
	SourceLessonIDs       []string       `json:"source_lesson_ids"`
	SuggestionType        SuggestionType `json:"suggestion_type"`
	TargetModule          string         `json:"target_module"`
	TargetSetupFamily     string         `json:"target_setup_family"`
	TargetEventType       string         `json:"target_event_type"`
	Summary               string         `json:"summary"`
	Rationale             string         `json:"rationale"`
	EvidenceRefs          []string       `json:"evidence_refs"`
	Confidence            float64        `json:"confidence"`
	RequiresHumanApproval bool           `json:"requires_human_approval"`
	AutoApplyAllowed      bool           `json:"auto_apply_allowed"`
	ForbiddenActions      []string       `json:"forbidden_actions"`
}

func ForbiddenActions() []string {
	return []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove}
}

func suggestionFromLesson(index int, lesson review.Lesson) (RuleSuggestion, bool) {
	suggestion := RuleSuggestion{
		SuggestionID:          fmt.Sprintf("suggestion_%03d", index),
		SourceLessonIDs:       []string{lesson.LessonID},
		TargetSetupFamily:     lesson.AppliesToSetupFamily,
		TargetEventType:       lesson.AppliesToEventType,
		EvidenceRefs:          append([]string{}, lesson.EvidenceRefs...),
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      ForbiddenActions(),
		Confidence:            0.6,
	}
	switch lesson.LessonType {
	case review.LessonMissedOpportunity:
		suggestion.SuggestionType = SuggestionNoTradeRuleReview
		suggestion.TargetModule = "NO_TRADE_RULES"
		suggestion.Summary = "Review no-trade or confirmation rules for missed opportunity."
		suggestion.Rationale = "A review lesson marked a missed opportunity; any threshold or rule change requires human approval."
	case review.LessonRiskVetoTooStrict:
		suggestion.SuggestionType = SuggestionRiskThresholdReview
		suggestion.TargetModule = "RISK_VETO"
		suggestion.Summary = "Review risk threshold strictness."
		suggestion.Rationale = "A risk veto lesson suggests the veto may have rejected a valid setup."
	case review.LessonConfirmationRuleTooStrict:
		suggestion.SuggestionType = SuggestionConfirmationRuleReview
		suggestion.TargetModule = "CONFIRMATION_RULES"
		suggestion.Summary = "Review confirmation rule strictness."
		suggestion.Rationale = "A confirmation lesson suggests required confirmations may be too restrictive."
	case review.LessonPaperSetupFailed, review.LessonResearchEvidenceInsufficient:
		suggestion.SuggestionType = SuggestionSetupFamilyResearch
		suggestion.TargetModule = "RESEARCH"
		suggestion.Summary = "Research setup family weakness before changing rules."
		suggestion.Rationale = "Weak or failed evidence should create a research action, not an automatic strategy update."
	default:
		return RuleSuggestion{}, false
	}
	return suggestion, true
}
