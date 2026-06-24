package swing

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

const BrainSwingV1 = "SWING_BRAIN_V1"

type EvaluationInput struct {
	Event                     core.Event
	Scores                    core.Scores
	Catalyst                  string
	SetupFamily               SetupFamily
	SupportingReasons         []string
	RequiredConfirmations     []string
	PresentConfirmations      []string
	MissingConfirmations      []string
	InvalidationConditions    []string
	RiskRewardRatio           float64
	UnresolvedEventRisk       []string
	MarketSectorAlignmentNote string
	GeneratedAt               time.Time
}

type Decision struct {
	core.Decision
	SetupFamily SetupFamily `json:"setup_family,omitempty"`
	Scores      core.Scores `json:"scores"`
}

func Evaluate(input EvaluationInput) Decision {
	scores := normaliseScores(input.Scores)
	family := normaliseSetupFamily(input.SetupFamily)
	outcome := decide(input, scores)

	return Decision{
		Decision: core.Decision{
			DecisionID:             "swing_" + input.Event.EventID,
			EventID:                input.Event.EventID,
			Brain:                  BrainSwingV1,
			Decision:               outcome.decision,
			ConfidenceScore:        scores.ConfidenceScore,
			ClarityScore:           scores.ClarityScore,
			EdgeScore:              scores.EdgeScore,
			ConflictScore:          scores.ConflictScore,
			RiskScore:              scores.RiskScore,
			PrimaryReason:          outcome.primaryReason,
			SupportingReasons:      supportingReasons(input, outcome, family),
			RequiredConfirmations:  requiredConfirmations(input, outcome),
			InvalidationConditions: cloneStrings(input.InvalidationConditions),
			AllowedActions:         allowedActions(outcome.decision),
			ForbiddenActions:       forbiddenActions(),
			ReviewAfter:            reviewAfter(),
		},
		SetupFamily: family,
		Scores:      scores,
	}
}

func normaliseScores(scores core.Scores) core.Scores {
	scores.ClarityScore = clamp(scores.ClarityScore)
	scores.EdgeScore = clamp(scores.EdgeScore)
	scores.ConflictScore = clamp(scores.ConflictScore)
	scores.RiskScore = clamp(scores.RiskScore)
	scores.ConfirmationScore = clamp(scores.ConfirmationScore)
	scores.TimingScore = clamp(scores.TimingScore)
	if scores.ConfidenceScore == 0 {
		scores.ConfidenceScore = max(scores.ClarityScore, scores.EdgeScore, 1-scores.ConflictScore, 1-scores.RiskScore)
	} else {
		scores.ConfidenceScore = clamp(scores.ConfidenceScore)
	}
	return scores
}

func clamp(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func max(values ...float64) float64 {
	var result float64
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}
