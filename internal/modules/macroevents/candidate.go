package macroevents

import (
	"context"
	"strings"
)

type CandidateSide string
type CandidateEntryType string
type MacroCandidateStatus string

const (
	CandidateSideLong      CandidateSide = "long"
	CandidateSideShortBias CandidateSide = "short_bias"
	CandidateSideWatchOnly CandidateSide = "watch_only"
	CandidateSideNoTrade   CandidateSide = "no_trade"
)

const (
	EntryTypeBreakoutContinuation CandidateEntryType = "breakout_continuation"
	EntryTypePullbackRetest       CandidateEntryType = "pullback_retest"
	EntryTypeRangeReclaim         CandidateEntryType = "range_reclaim"
	EntryTypeNoEntry              CandidateEntryType = "no_entry"
)

const (
	MacroCandidateStatusAwaitingHumanApproval MacroCandidateStatus = "awaiting_human_approval"
	MacroCandidateStatusWatchOnly             MacroCandidateStatus = "watch_only"
	MacroCandidateStatusBlocked               MacroCandidateStatus = "blocked"
)

type CandidatePlan struct {
	EntryType       CandidateEntryType
	EntryPrice      *float64
	StopPrice       *float64
	TargetPrice     *float64
	RiskPercent     float64
	TimeLimit       string
	RewardRiskRatio float64
}

type CandidateInput struct {
	Bundle EvidenceBundle
	Side   CandidateSide
	Plan   CandidatePlan
}

type MacroCandidate struct {
	ID                   string
	MacroEventID         string
	EvidenceBundleID     string
	Symbol               string
	Side                 CandidateSide
	Bias                 string
	EntryType            CandidateEntryType
	EntryReferencePrice  float64
	StopReferencePrice   float64
	TargetReferencePrice float64
	RiskPercent          float64
	TimeLimit            string
	Status               MacroCandidateStatus
	CreatedReason        string
	RejectionReason      string
	WalkawayReasons      []string
}

func GenerateCandidate(input CandidateInput) MacroCandidate {
	bundle := input.Bundle
	base := MacroCandidate{
		MacroEventID:     bundle.MacroEventID,
		EvidenceBundleID: bundle.ID,
		Symbol:           strings.ToUpper(strings.TrimSpace(bundle.Symbol)),
		CreatedReason:    bundle.Summary,
		WalkawayReasons:  append([]string(nil), bundle.WalkawayReasons...),
	}
	switch bundle.Verdict {
	case EvidenceVerdictCandidateAllowed:
		return candidateFromAllowedBundle(base, input)
	case EvidenceVerdictWatchOnly:
		base.Side = CandidateSideWatchOnly
		base.EntryType = EntryTypeNoEntry
		base.Status = MacroCandidateStatusWatchOnly
		base.RejectionReason = "evidence verdict is watch_only"
		return base
	default:
		base.Side = CandidateSideNoTrade
		base.EntryType = EntryTypeNoEntry
		base.Status = MacroCandidateStatusBlocked
		base.RejectionReason = "evidence verdict does not allow candidate"
		return base
	}
}

func candidateFromAllowedBundle(base MacroCandidate, input CandidateInput) MacroCandidate {
	plan := input.Plan
	base.Side = input.Side
	if base.Side == "" {
		base.Side = CandidateSideLong
	}
	base.EntryType = plan.EntryType
	if base.EntryType == "" {
		base.EntryType = EntryTypePullbackRetest
	}
	if plan.EntryPrice == nil {
		return blockMacroCandidate(base, "entry reference price is required")
	}
	if plan.StopPrice == nil {
		return blockMacroCandidate(base, "stop reference price is required")
	}
	if plan.TargetPrice == nil {
		return blockMacroCandidate(base, "target reference price is required")
	}
	if plan.RiskPercent > 0.5 {
		return blockMacroCandidate(base, "risk percent exceeds 0.5 limit")
	}
	if plan.RiskPercent <= 0 {
		return blockMacroCandidate(base, "risk percent must be greater than zero")
	}
	if plan.RewardRiskRatio < 1.5 {
		return blockMacroCandidate(base, "reward:risk is below 1.5")
	}
	base.EntryReferencePrice = *plan.EntryPrice
	base.StopReferencePrice = *plan.StopPrice
	base.TargetReferencePrice = *plan.TargetPrice
	base.RiskPercent = plan.RiskPercent
	base.TimeLimit = strings.TrimSpace(plan.TimeLimit)
	if base.TimeLimit == "" {
		base.TimeLimit = "end_of_session"
	}
	base.Bias = string(base.Side)
	base.Status = MacroCandidateStatusAwaitingHumanApproval
	return base
}

func blockMacroCandidate(candidate MacroCandidate, reason string) MacroCandidate {
	candidate.Side = CandidateSideNoTrade
	candidate.EntryType = EntryTypeNoEntry
	candidate.Status = MacroCandidateStatusBlocked
	candidate.RejectionReason = reason
	return candidate
}

type macroCandidateStore interface {
	SaveMacroCandidate(ctx context.Context, candidate MacroCandidate) (MacroCandidate, error)
}

type MacroCandidateService struct {
	store macroCandidateStore
}

func NewMacroCandidateService(store macroCandidateStore) *MacroCandidateService {
	return &MacroCandidateService{store: store}
}

func (s *MacroCandidateService) GenerateAndSave(ctx context.Context, input CandidateInput) (MacroCandidate, error) {
	candidate := GenerateCandidate(input)
	if s.store == nil {
		return candidate, nil
	}
	return s.store.SaveMacroCandidate(ctx, candidate)
}
