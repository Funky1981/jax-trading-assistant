package macroevents

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type RiskVerdict string

const (
	RiskVerdictPass                 RiskVerdict = "pass"
	RiskVerdictFail                 RiskVerdict = "fail"
	RiskVerdictInsufficientEvidence RiskVerdict = "insufficient_evidence"
)

type RiskReviewOutput struct {
	Verdict          RiskVerdict
	RiskScore        float64
	HardBlocks       []AnalystHardVeto
	Reasons          []string
	ApprovalRequired bool
}

type TradeReviewerOutput struct {
	Decision             AnalystDecision
	CandidateScore       float64
	Reasons              []string
	ApprovalRequired     bool
	LLMSummary           string
	LLMOverrideAttempted bool
}

type MultiAnalystReviewInput struct {
	MacroEventID         string
	Symbol               string
	Fundamental          FundamentalSnapshot
	Technical            TechnicalSnapshot
	AnalystDecision      AnalystDecisionRecord
	LLMSuggestedDecision AnalystDecision
	LLMSummary           string
}

type MultiAnalystReviewRecord struct {
	ID              string
	MacroEventID    string
	Symbol          string
	Fundamental     FundamentalSnapshot
	Technical       TechnicalSnapshot
	AnalystDecision AnalystDecisionRecord
	Risk            RiskReviewOutput
	Review          TradeReviewerOutput
	CreatedAt       time.Time
}

type multiAnalystReviewStore interface {
	SaveMultiAnalystReview(ctx context.Context, review MultiAnalystReviewRecord) (MultiAnalystReviewRecord, error)
}

type MultiAnalystReviewService struct {
	store multiAnalystReviewStore
}

func NewMultiAnalystReviewService(store multiAnalystReviewStore) *MultiAnalystReviewService {
	return &MultiAnalystReviewService{store: store}
}

func (s *MultiAnalystReviewService) EvaluateAndSave(ctx context.Context, input MultiAnalystReviewInput) (MultiAnalystReviewRecord, error) {
	review := EvaluateMultiAnalystReview(input)
	if s.store == nil {
		return review, nil
	}
	return s.store.SaveMultiAnalystReview(ctx, review)
}

func EvaluateMultiAnalystReview(input MultiAnalystReviewInput) MultiAnalystReviewRecord {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	record := MultiAnalystReviewRecord{
		MacroEventID:    strings.TrimSpace(input.MacroEventID),
		Symbol:          symbol,
		Fundamental:     input.Fundamental,
		Technical:       input.Technical,
		AnalystDecision: input.AnalystDecision,
		CreatedAt:       time.Now().UTC(),
	}

	risk := buildRiskReviewOutput(input.AnalystDecision)
	review := buildTradeReviewerOutput(input, risk)

	record.Risk = risk
	record.Review = review
	return record
}

func buildRiskReviewOutput(decision AnalystDecisionRecord) RiskReviewOutput {
	risk := RiskReviewOutput{
		Verdict:          RiskVerdictPass,
		RiskScore:        clampAnalystScore(decision.RiskScore),
		HardBlocks:       []AnalystHardVeto{},
		Reasons:          []string{},
		ApprovalRequired: true,
	}

	for _, veto := range decision.HardVetoes {
		switch veto {
		case AnalystHardVetoNoStop, AnalystHardVetoRewardRisk, AnalystHardVetoETFNotAllowlisted,
			AnalystHardVetoLiveTradingRequested, AnalystHardVetoMajorConfounder, AnalystHardVetoFundamentalConflicted,
			AnalystHardVetoPricedIn, AnalystHardVetoPricedInUnclear:
			risk.HardBlocks = append(risk.HardBlocks, veto)
		}
	}
	risk.HardBlocks = uniqueAnalystHardVetoes(risk.HardBlocks)

	if len(risk.HardBlocks) > 0 {
		risk.Verdict = RiskVerdictFail
		risk.Reasons = append(risk.Reasons, "risk manager blocked candidate due to hard vetoes")
	}
	if decision.Decision == AnalystDecisionInsufficientEvidence {
		risk.Verdict = RiskVerdictInsufficientEvidence
		risk.Reasons = append(risk.Reasons, "insufficient evidence for risk validation")
	}
	if len(risk.Reasons) == 0 {
		risk.Reasons = append(risk.Reasons, "risk checks passed")
	}

	return risk
}

func buildTradeReviewerOutput(input MultiAnalystReviewInput, risk RiskReviewOutput) TradeReviewerOutput {
	decision := input.AnalystDecision.Decision
	reasons := append([]string{}, input.AnalystDecision.Reasons...)
	candidateScore := clampAnalystScore(input.AnalystDecision.CandidateScore)

	if strings.TrimSpace(input.Fundamental.ID) == "" || strings.TrimSpace(input.Technical.ID) == "" {
		decision = AnalystDecisionInsufficientEvidence
		reasons = append(reasons, "missing analyst role output")
	}
	if isFundamentalBlocking(input.Fundamental.Verdict) || isTechnicalBlocking(input.Technical.Verdict) {
		decision = AnalystDecisionCandidateRejected
		reasons = append(reasons, "analyst role verdict conflict blocks candidate")
	}
	if risk.Verdict == RiskVerdictFail {
		decision = AnalystDecisionCandidateRejected
		reasons = append(reasons, "risk manager veto blocks candidate")
	}
	if risk.Verdict == RiskVerdictInsufficientEvidence {
		decision = AnalystDecisionInsufficientEvidence
		reasons = append(reasons, "risk manager could not validate evidence")
	}
	if len(input.AnalystDecision.HardVetoes) > 0 && decision == AnalystDecisionCandidateAllowed {
		decision = AnalystDecisionCandidateRejected
		reasons = append(reasons, "hard veto prevents candidate_allowed")
	}

	overrideAttempted := false
	if input.LLMSuggestedDecision != "" && input.LLMSuggestedDecision != decision {
		overrideAttempted = true
		reasons = append(reasons, fmt.Sprintf("llm suggestion %q ignored to preserve deterministic decision", input.LLMSuggestedDecision))
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "trade reviewer aligned with deterministic analyst decision")
	}

	return TradeReviewerOutput{
		Decision:             decision,
		CandidateScore:       math.Round(candidateScore*100) / 100,
		Reasons:              reasons,
		ApprovalRequired:     true,
		LLMSummary:           strings.TrimSpace(input.LLMSummary),
		LLMOverrideAttempted: overrideAttempted,
	}
}

func isFundamentalBlocking(verdict FundamentalVerdict) bool {
	switch verdict {
	case FundamentalVerdictConflicted, FundamentalVerdictNeutral, FundamentalVerdictStrongBullish, FundamentalVerdictModerateBullish:
		return true
	default:
		return false
	}
}

func isTechnicalBlocking(verdict TechnicalVerdict) bool {
	switch verdict {
	case TechnicalVerdictNoConfirmation, TechnicalVerdictTooExtended, TechnicalVerdictWhipsaw:
		return true
	default:
		return false
	}
}
