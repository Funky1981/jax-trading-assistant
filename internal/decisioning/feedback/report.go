package feedback

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/decisioning/replay"
	"jax-trading-assistant/internal/decisioning/review"
)

type ReportInput struct {
	ReportID                  string              `json:"report_id"`
	ReplayResult              replay.ReplayResult `json:"replay_result"`
	Lessons                   []review.Lesson     `json:"lessons"`
	ResearchEvidence          []ResearchEvidence  `json:"research_evidence"`
	AttemptedPromotion        string              `json:"attempted_promotion"`
	AttemptedForbiddenActions []string            `json:"attempted_forbidden_actions"`
	CreatedAt                 time.Time           `json:"created_at"`
}

type ResearchEvidence struct {
	EvidenceID   string   `json:"evidence_id"`
	SetupFamily  string   `json:"setup_family"`
	EventType    string   `json:"event_type"`
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type FeedbackReport struct {
	ReportID                      string           `json:"report_id"`
	ReplayID                      string           `json:"replay_id"`
	Summary                       string           `json:"summary"`
	KeyFindings                   []string         `json:"key_findings"`
	SetupFamilyFindings           []string         `json:"setup_family_findings"`
	EventTypeFindings             []string         `json:"event_type_findings"`
	RiskVetoFindings              []string         `json:"risk_veto_findings"`
	NoTradeFindings               []string         `json:"no_trade_findings"`
	PaperOutcomeFindings          []string         `json:"paper_outcome_findings"`
	ResearchGaps                  []string         `json:"research_gaps"`
	SuggestedRuleChanges          []RuleSuggestion `json:"suggested_rule_changes"`
	SuggestedResearchActions      []string         `json:"suggested_research_actions"`
	SuggestedWatchlistAdjustments []string         `json:"suggested_watchlist_adjustments"`
	RequiresHumanApproval         bool             `json:"requires_human_approval"`
	ForbiddenActions              []string         `json:"forbidden_actions"`
	Warnings                      []string         `json:"warnings"`
	Errors                        []string         `json:"errors"`
	CreatedAt                     time.Time        `json:"created_at"`
}

func BuildReport(input ReportInput) FeedbackReport {
	report := FeedbackReport{
		ReportID:              input.ReportID,
		ReplayID:              input.ReplayResult.ReplayID,
		RequiresHumanApproval: true,
		ForbiddenActions:      ForbiddenActions(),
		CreatedAt:             input.CreatedAt,
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	if report.ReportID == "" {
		report.Errors = append(report.Errors, "report_id is required")
	}
	report.Summary = fmt.Sprintf("Replay %s processed %d records, including %d no-trades, %d risk rejections, and %d paper approvals.",
		report.ReplayID,
		input.ReplayResult.RecordsProcessed,
		input.ReplayResult.NoTradeCount,
		input.ReplayResult.RejectedByRiskCount,
		input.ReplayResult.PaperApprovedCount,
	)
	applyReplayCounts(&report, input.ReplayResult)
	for i, lesson := range input.Lessons {
		applyLesson(&report, lesson)
		if suggestion, ok := suggestionFromLesson(i+1, lesson); ok {
			report.SuggestedRuleChanges = append(report.SuggestedRuleChanges, suggestion)
		}
	}
	applyResearchEvidence(&report, input.ResearchEvidence)
	validateForbiddenInput(&report, input)
	return report
}

func applyReplayCounts(report *FeedbackReport, result replay.ReplayResult) {
	if result.CorrectNoTradeCount > 0 {
		report.NoTradeFindings = append(report.NoTradeFindings, "No-trade rule helped avoid weak setups.")
	}
	if result.MissedOpportunityCount > 0 {
		report.NoTradeFindings = append(report.NoTradeFindings, "Replay found missed opportunity lessons.")
	}
	if result.AvoidedLossCount > 0 {
		report.KeyFindings = append(report.KeyFindings, "Replay found avoided losses.")
	}
	if result.RiskVetoHelpedCount > 0 {
		report.RiskVetoFindings = append(report.RiskVetoFindings, "Risk Veto avoided bad setup.")
	}
	if result.RiskVetoTooStrictCount > 0 {
		report.RiskVetoFindings = append(report.RiskVetoFindings, "Risk Veto may have been too strict.")
	}
	if result.PaperSetupWorkedCount > 0 {
		report.PaperOutcomeFindings = append(report.PaperOutcomeFindings, "Paper-approved setup outcome was promising.")
	}
	if result.PaperSetupFailedCount > 0 {
		report.PaperOutcomeFindings = append(report.PaperOutcomeFindings, "Paper-approved setup outcome showed weakness.")
	}
}

func applyResearchEvidence(report *FeedbackReport, evidence []ResearchEvidence) {
	for _, item := range evidence {
		setupFamily := item.SetupFamily
		if setupFamily == "" {
			setupFamily = "unspecified setup family"
		}
		switch item.Status {
		case "", "MISSING":
			report.ResearchGaps = append(report.ResearchGaps, fmt.Sprintf("%s has missing research evidence.", setupFamily))
			report.SuggestedResearchActions = append(report.SuggestedResearchActions, fmt.Sprintf("Add research evidence for %s before rule changes.", setupFamily))
		case "BACKTESTED_WEAK":
			report.ResearchGaps = append(report.ResearchGaps, fmt.Sprintf("%s has weak backtest evidence.", setupFamily))
			report.SuggestedResearchActions = append(report.SuggestedResearchActions, fmt.Sprintf("Improve weak evidence for %s with additional validation.", setupFamily))
		}
	}
}

func validateForbiddenInput(report *FeedbackReport, input ReportInput) {
	if strings.EqualFold(input.AttemptedPromotion, "LIVE_READY") {
		report.Errors = append(report.Errors, "LIVE_READY promotion is forbidden in feedback reporting")
	}
	for _, action := range input.AttemptedForbiddenActions {
		if containsAny(action, ForbiddenActions()) {
			report.Warnings = append(report.Warnings, fmt.Sprintf("forbidden action %s preserved and not executed", action))
		}
	}
}

func contains(value string, want string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(want))
}

func containsAny(value string, values []string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
