package review

import "time"

type LessonType string

const (
	LessonCorrectNoTrade               LessonType = "CORRECT_NO_TRADE"
	LessonMissedOpportunity            LessonType = "MISSED_OPPORTUNITY"
	LessonAvoidedLoss                  LessonType = "AVOIDED_LOSS"
	LessonBadCandidateRejected         LessonType = "BAD_CANDIDATE_REJECTED"
	LessonPaperSetupWorked             LessonType = "PAPER_SETUP_WORKED"
	LessonPaperSetupFailed             LessonType = "PAPER_SETUP_FAILED"
	LessonConfirmationRuleHelped       LessonType = "CONFIRMATION_RULE_HELPED"
	LessonConfirmationRuleTooStrict    LessonType = "CONFIRMATION_RULE_TOO_STRICT"
	LessonRiskVetoHelped               LessonType = "RISK_VETO_HELPED"
	LessonRiskVetoTooStrict            LessonType = "RISK_VETO_TOO_STRICT"
	LessonResearchEvidenceHelped       LessonType = "RESEARCH_EVIDENCE_HELPED"
	LessonResearchEvidenceInsufficient LessonType = "RESEARCH_EVIDENCE_INSUFFICIENT"
)

type Lesson struct {
	LessonID              string     `json:"lesson_id"`
	DecisionID            string     `json:"decision_id"`
	EventID               string     `json:"event_id"`
	LessonType            LessonType `json:"lesson_type"`
	LessonSummary         string     `json:"lesson_summary"`
	EvidenceRefs          []string   `json:"evidence_refs"`
	AppliesToSetupFamily  string     `json:"applies_to_setup_family"`
	AppliesToEventType    string     `json:"applies_to_event_type"`
	SuggestedAction       string     `json:"suggested_action"`
	RequiresHumanApproval bool       `json:"requires_human_approval"`
	CreatedAt             time.Time  `json:"created_at"`
}

func LessonTypeAllowed(lessonType LessonType) bool {
	switch lessonType {
	case LessonCorrectNoTrade,
		LessonMissedOpportunity,
		LessonAvoidedLoss,
		LessonBadCandidateRejected,
		LessonPaperSetupWorked,
		LessonPaperSetupFailed,
		LessonConfirmationRuleHelped,
		LessonConfirmationRuleTooStrict,
		LessonRiskVetoHelped,
		LessonRiskVetoTooStrict,
		LessonResearchEvidenceHelped,
		LessonResearchEvidenceInsufficient:
		return true
	default:
		return false
	}
}

func lessonRequiresHumanApproval(lesson Lesson, review OutcomeReview) bool {
	if lesson.SuggestedAction != "" ||
		review.ScoringAdjustmentSuggestion != "" ||
		review.StrategyAdjustmentSuggestion != "" {
		return true
	}
	switch lesson.LessonType {
	case LessonMissedOpportunity,
		LessonRiskVetoTooStrict,
		LessonConfirmationRuleTooStrict,
		LessonResearchEvidenceInsufficient:
		return true
	default:
		return false
	}
}
