package pipeline

import (
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/classify"
	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/review"
	"jax-trading-assistant/internal/decisioning/risk"
)

func Run(input Input) Result {
	now, validationErrors, validationWarnings := validateInput(input)

	eventIntelligence := classify.EnrichEvent(input.Event)
	decisionCore := core.Evaluate(core.EvaluationInput{
		Event:                  eventIntelligence.Event,
		Scores:                 input.DecisionScores,
		MarketContext:          marketContextAny(input.MarketContext),
		GeneratedAt:            now,
		SupportingReasons:      input.Swing.SupportingReasons,
		RequiredConfirmations:  input.Swing.RequiredConfirmations,
		MissingConfirmations:   input.Swing.MissingConfirmations,
		InvalidationConditions: input.Swing.InvalidationConditions,
	})

	swingDecision := swing.Evaluate(swing.EvaluationInput{
		Event:                     eventIntelligence.Event,
		Scores:                    input.Swing.Scores,
		Catalyst:                  input.Swing.Catalyst,
		SetupFamily:               input.Swing.SetupFamily,
		SupportingReasons:         input.Swing.SupportingReasons,
		RequiredConfirmations:     input.Swing.RequiredConfirmations,
		PresentConfirmations:      input.Swing.PresentConfirmations,
		MissingConfirmations:      input.Swing.MissingConfirmations,
		InvalidationConditions:    input.Swing.InvalidationConditions,
		RiskRewardRatio:           input.Swing.RiskRewardRatio,
		UnresolvedEventRisk:       input.Swing.UnresolvedEventRisk,
		MarketSectorAlignmentNote: input.Swing.MarketSectorAlignmentNote,
		GeneratedAt:               now,
	})

	riskAssessment := risk.Assess(risk.AssessmentInput{
		SwingDecision:            swingDecision,
		Asset:                    candidateAsset(input, eventIntelligence),
		SetupFamily:              input.Swing.SetupFamily,
		ProposedEntry:            input.Swing.ProposedEntry,
		ProposedStop:             input.Swing.ProposedStop,
		ProposedTarget:           input.Swing.ProposedTarget,
		RiskRewardRatio:          input.Swing.RiskRewardRatio,
		AccountMode:              accountMode(input.AccountMode),
		Portfolio:                portfolioContext(input),
		CurrentExposure:          input.CurrentExposure,
		SectorExposure:           input.SectorExposure,
		CorrelatedExposure:       input.CorrelatedExposure,
		LiquiditySpreadNotes:     input.Swing.LiquiditySpreadNotes,
		UnresolvedEventRisk:      input.Swing.UnresolvedEventRisk,
		UpcomingMajorEventRisk:   input.Swing.UpcomingMajorEventRisk,
		BrokerExecutionRequested: false,
		LiveOrderRequested:       false,
	})

	var researchResult *research.ValidationResult
	if swingDecision.Decision.Decision == core.DecisionTradeCandidate &&
		riskAssessment.RiskDecision == risk.RiskDecisionPass &&
		riskAssessment.FinalDecision == core.DecisionTradeCandidate {
		researchResult = validateResearchEvidence(input.ResearchEvidence)
	}
	finalDecision := riskAssessment.FinalDecision
	finalStatus := statusBeforePaper(decisionCore.FinalDecision.Decision, swingDecision.Decision.Decision, riskAssessment)
	allowedActions := safeAllowedActions(riskAssessment.AllowedActions)
	forbiddenActions := normaliseForbiddenActions(
		decisionCore.FinalDecision.ForbiddenActions,
		swingDecision.Decision.ForbiddenActions,
		riskAssessment.ForbiddenActions,
	)

	var ticket *paper.PaperTicket
	if finalStatus == "" {
		finalStatus, ticket = preparePaperReview(input, now, swingDecision, riskAssessment, researchResult, &validationWarnings, &validationErrors)
	}
	if finalStatus == "" {
		finalStatus = StatusPaperReviewBlocked
	}
	if len(validationErrors) > 0 {
		finalStatus = StatusPipelineInvalid
	}

	reviewSchedule := review.NewReviewSchedule(swingDecision.Decision.DecisionID, now)
	return Result{
		PipelineID:              pipelineID(input.Event.EventID),
		EventID:                 input.Event.EventID,
		EventIntelligenceResult: eventIntelligence,
		DecisionCoreResult:      decisionCore,
		SwingBrainResult:        swingDecision,
		RiskAssessment:          riskAssessment,
		ResearchEvidenceResult:  researchResult,
		PaperTicketResult:       ticket,
		ReviewScheduleResult:    reviewSchedule,
		FinalDecision:           finalDecision,
		FinalStatus:             finalStatus,
		AllowedActions:          allowedActions,
		ForbiddenActions:        forbiddenActions,
		HumanApprovalRequired:   riskAssessment.RequiresHumanApproval,
		PaperOnly:               riskAssessment.PaperOnly,
		LiveTradingBlocked:      riskAssessment.LiveTradingBlocked,
		ValidationErrors:        validationErrors,
		ValidationWarnings:      validationWarnings,
		CreatedAt:               now,
	}
}

func statusBeforePaper(coreDecision core.DecisionValue, swingDecision core.DecisionValue, assessment risk.RiskAssessment) FinalStatus {
	if coreDecision == core.DecisionNoTrade || swingDecision == core.DecisionNoTrade {
		return StatusNoTradeRecorded
	}
	if swingDecision == core.DecisionWatch {
		return StatusWatchRecorded
	}
	if swingDecision == core.DecisionSetupForming {
		return StatusSetupFormingRecorded
	}
	if swingDecision != core.DecisionTradeCandidate {
		return StatusNoTradeRecorded
	}
	if assessment.FinalDecision == core.DecisionRejectedByRisk ||
		assessment.RiskDecision == risk.RiskDecisionReject ||
		assessment.RiskDecision == risk.RiskDecisionRejectedByRisk {
		return StatusTradeCandidateRejectedByRisk
	}
	if assessment.RiskDecision != risk.RiskDecisionPass || assessment.FinalDecision != core.DecisionTradeCandidate {
		return StatusPaperReviewBlocked
	}
	return ""
}

func preparePaperReview(
	input Input,
	now time.Time,
	swingDecision swing.Decision,
	assessment risk.RiskAssessment,
	researchResult *research.ValidationResult,
	warnings *[]string,
	errors *[]string,
) (FinalStatus, *paper.PaperTicket) {
	if !evidenceSufficient(researchResult) {
		if input.ResearchEvidence == nil && !input.Options.researchRequired() {
			*warnings = append(*warnings, "research evidence is explicitly not required for this pipeline run")
		} else {
			return StatusTradeCandidateNeedsResearch, nil
		}
	}

	ticketRequest := paper.TicketRequest{
		SourceDecision:                swingDecision.Decision,
		RiskAssessment:                assessment,
		ResearchEvidenceSummary:       researchSummary(input, researchResult, warnings),
		Asset:                         candidateAsset(input, classify.EventIntelligence{}),
		SetupFamily:                   string(swingDecision.SetupFamily),
		ProposedEntryZone:             entryZone(input),
		ProposedStop:                  input.Swing.ProposedStop,
		ProposedTarget:                input.Swing.ProposedTarget,
		RiskReward:                    input.Swing.RiskRewardRatio,
		MaxPaperPositionSize:          input.Swing.MaxPaperPositionSize,
		CreatedAt:                     now,
		ExpiresAt:                     now.Add(24 * time.Hour),
		RequiredConfirmations:         swingDecision.Decision.RequiredConfirmations,
		InvalidationConditions:        swingDecision.Decision.InvalidationConditions,
		ExplicitHumanApprovalRequired: true,
	}

	ticket, validation := paper.NewTicket(ticketRequest)
	*warnings = append(*warnings, validation.ValidationWarnings...)
	if !validation.CanCreateTicket {
		*errors = append(*errors, validation.ValidationErrors...)
		return StatusPaperReviewBlocked, nil
	}
	return StatusTradeCandidateReadyForPaperReview, &ticket
}

func marketContextAny(context map[string]string) map[string]any {
	if len(context) == 0 {
		return nil
	}
	result := make(map[string]any, len(context))
	for key, value := range context {
		result[key] = value
	}
	return result
}

func portfolioContext(input Input) risk.PortfolioContext {
	if input.PortfolioContext == nil {
		return risk.PortfolioContext{}
	}
	return *input.PortfolioContext
}

func accountMode(mode risk.AccountMode) risk.AccountMode {
	if mode == "" {
		return risk.AccountModePaper
	}
	return mode
}

func candidateAsset(input Input, intelligence classify.EventIntelligence) string {
	if input.Swing.Asset != "" {
		return input.Swing.Asset
	}
	if len(intelligence.AffectedAssets) > 0 {
		return intelligence.AffectedAssets[0]
	}
	if len(input.Event.AffectedAssets) > 0 {
		return input.Event.AffectedAssets[0]
	}
	return ""
}

func entryZone(input Input) paper.EntryZone {
	high := input.Swing.ProposedEntryHigh
	if high == 0 {
		high = input.Swing.ProposedEntry
	}
	return paper.EntryZone{
		Low:  input.Swing.ProposedEntry,
		High: high,
	}
}

func researchSummary(input Input, result *research.ValidationResult, warnings *[]string) paper.ResearchEvidenceSummary {
	if input.ResearchEvidence == nil {
		return paper.ResearchEvidenceSummary{}
	}
	if result != nil {
		*warnings = append(*warnings, result.ValidationWarnings...)
	}
	return paper.ResearchEvidenceSummary{
		HypothesisID:      input.ResearchEvidence.HypothesisID,
		SetupFamily:       input.ResearchEvidence.SetupFamily,
		PromotionDecision: input.ResearchEvidence.PromotionDecision,
		Summary:           "research evidence validated for paper review preparation",
		Warnings:          researchWarnings(result),
	}
}

func researchWarnings(result *research.ValidationResult) []string {
	if result == nil {
		return nil
	}
	return result.ValidationWarnings
}
