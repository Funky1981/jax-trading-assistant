package review

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

type OutcomeSummary struct {
	Summary          string  `json:"summary"`
	Direction        string  `json:"direction,omitempty"`
	ReturnPct        float64 `json:"return_pct,omitempty"`
	HitTarget        bool    `json:"hit_target,omitempty"`
	HitInvalidation  bool    `json:"hit_invalidation,omitempty"`
	CleanSetupFormed bool    `json:"clean_setup_formed,omitempty"`
}

type OutcomeReviewInput struct {
	ReviewID                     string             `json:"review_id"`
	DecisionID                   string             `json:"decision_id"`
	EventID                      string             `json:"event_id"`
	ReviewWindow                 string             `json:"review_window"`
	OriginalDecision             core.DecisionValue `json:"original_decision"`
	FinalDecision                string             `json:"final_decision"`
	AssetOutcome                 OutcomeSummary     `json:"asset_outcome"`
	MarketOutcome                OutcomeSummary     `json:"market_outcome"`
	WasDecisionCorrect           bool               `json:"was_decision_correct"`
	MissedOpportunity            bool               `json:"missed_opportunity"`
	AvoidedLoss                  bool               `json:"avoided_loss"`
	LessonSummary                string             `json:"lesson_summary"`
	MemoryTags                   []string           `json:"memory_tags"`
	ScoringAdjustmentSuggestion  string             `json:"scoring_adjustment_suggestion,omitempty"`
	StrategyAdjustmentSuggestion string             `json:"strategy_adjustment_suggestion,omitempty"`
	RequiresHumanReview          bool               `json:"requires_human_review"`
	AttemptedPromotion           string             `json:"attempted_promotion,omitempty"`
	AttemptedLiveTradingApproval bool               `json:"attempted_live_trading_approval,omitempty"`
	AttemptedPaperExecution      bool               `json:"attempted_paper_execution,omitempty"`
	AppliesToSetupFamily         string             `json:"applies_to_setup_family,omitempty"`
	AppliesToEventType           string             `json:"applies_to_event_type,omitempty"`
	EvidenceRefs                 []string           `json:"evidence_refs,omitempty"`
	CreatedAt                    time.Time          `json:"created_at"`
}

type OutcomeReview struct {
	ReviewID                     string             `json:"review_id"`
	DecisionID                   string             `json:"decision_id"`
	EventID                      string             `json:"event_id"`
	ReviewWindow                 string             `json:"review_window"`
	OriginalDecision             core.DecisionValue `json:"original_decision"`
	FinalDecision                string             `json:"final_decision"`
	AssetOutcome                 OutcomeSummary     `json:"asset_outcome"`
	MarketOutcome                OutcomeSummary     `json:"market_outcome"`
	WasDecisionCorrect           bool               `json:"was_decision_correct"`
	MissedOpportunity            bool               `json:"missed_opportunity"`
	AvoidedLoss                  bool               `json:"avoided_loss"`
	Lesson                       Lesson             `json:"lesson"`
	MemoryTags                   []string           `json:"memory_tags"`
	ScoringAdjustmentSuggestion  string             `json:"scoring_adjustment_suggestion,omitempty"`
	StrategyAdjustmentSuggestion string             `json:"strategy_adjustment_suggestion,omitempty"`
	RequiresHumanReview          bool               `json:"requires_human_review"`
	PromotionBlocked             bool               `json:"promotion_blocked"`
	AttemptedPromotion           string             `json:"attempted_promotion,omitempty"`
	AttemptedLiveTradingApproval bool               `json:"attempted_live_trading_approval,omitempty"`
	AttemptedPaperExecution      bool               `json:"attempted_paper_execution,omitempty"`
	ForbiddenActions             []string           `json:"forbidden_actions"`
	CreatedAt                    time.Time          `json:"created_at"`
}

func NewOutcomeReview(input OutcomeReviewInput) (OutcomeReview, ValidationResult) {
	lessonType := inferLessonType(input)
	reviewID := firstNonEmpty(input.ReviewID, "rev_"+input.DecisionID+"_"+input.ReviewWindow)
	lesson := Lesson{
		LessonID:             "lesson_" + reviewID,
		DecisionID:           input.DecisionID,
		EventID:              input.EventID,
		LessonType:           lessonType,
		LessonSummary:        input.LessonSummary,
		EvidenceRefs:         input.EvidenceRefs,
		AppliesToSetupFamily: input.AppliesToSetupFamily,
		AppliesToEventType:   input.AppliesToEventType,
		SuggestedAction:      firstNonEmpty(input.ScoringAdjustmentSuggestion, input.StrategyAdjustmentSuggestion),
		CreatedAt:            input.CreatedAt,
	}

	review := OutcomeReview{
		ReviewID:                     reviewID,
		DecisionID:                   input.DecisionID,
		EventID:                      input.EventID,
		ReviewWindow:                 input.ReviewWindow,
		OriginalDecision:             input.OriginalDecision,
		FinalDecision:                input.FinalDecision,
		AssetOutcome:                 input.AssetOutcome,
		MarketOutcome:                input.MarketOutcome,
		WasDecisionCorrect:           input.WasDecisionCorrect,
		MissedOpportunity:            input.MissedOpportunity,
		AvoidedLoss:                  input.AvoidedLoss,
		Lesson:                       lesson,
		MemoryTags:                   input.MemoryTags,
		ScoringAdjustmentSuggestion:  input.ScoringAdjustmentSuggestion,
		StrategyAdjustmentSuggestion: input.StrategyAdjustmentSuggestion,
		RequiresHumanReview:          input.RequiresHumanReview,
		PromotionBlocked:             input.AttemptedPromotion == "LIVE_READY" || input.AttemptedLiveTradingApproval,
		AttemptedPromotion:           input.AttemptedPromotion,
		AttemptedLiveTradingApproval: input.AttemptedLiveTradingApproval,
		AttemptedPaperExecution:      input.AttemptedPaperExecution,
		ForbiddenActions:             mandatoryForbiddenActions(nil),
		CreatedAt:                    input.CreatedAt,
	}
	review.Lesson.RequiresHumanApproval = lessonRequiresHumanApproval(review.Lesson, review) || review.RequiresHumanReview
	if review.Lesson.RequiresHumanApproval {
		review.RequiresHumanReview = true
	}
	if input.AttemptedPaperExecution {
		review.PromotionBlocked = true
	}
	return review, ValidateOutcomeReview(review)
}

func inferLessonType(input OutcomeReviewInput) LessonType {
	if (input.FinalDecision == "APPROVED_FOR_PAPER" || input.AttemptedPromotion == "LIVE_READY") && input.AssetOutcome.HitTarget && !input.AssetOutcome.HitInvalidation {
		return LessonPaperSetupWorked
	}
	if input.FinalDecision == "APPROVED_FOR_PAPER" && input.AssetOutcome.HitInvalidation {
		return LessonPaperSetupFailed
	}
	if input.OriginalDecision == core.DecisionRejectedByRisk && input.AvoidedLoss {
		return LessonRiskVetoHelped
	}
	if input.OriginalDecision == core.DecisionRejectedByRisk && input.MissedOpportunity {
		return LessonRiskVetoTooStrict
	}
	if input.OriginalDecision == core.DecisionNoTrade && input.MissedOpportunity {
		return LessonMissedOpportunity
	}
	if input.OriginalDecision == core.DecisionNoTrade && input.WasDecisionCorrect {
		return LessonCorrectNoTrade
	}
	if input.AvoidedLoss {
		return LessonAvoidedLoss
	}
	if input.MissedOpportunity {
		return LessonMissedOpportunity
	}
	return LessonResearchEvidenceHelped
}
