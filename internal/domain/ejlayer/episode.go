package ejlayer

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// EpisodeType represents what kind of decision was made
type EpisodeType string

const (
	EpisodeTrade      EpisodeType = "trade"
	EpisodeAbstention EpisodeType = "abstention"
	EpisodeDeferral   EpisodeType = "deferral"
)

// ContextDominance represents what factor dominated the market context
type ContextDominance string

const (
	DomTechnical  ContextDominance = "technical"
	DomVolatility ContextDominance = "volatility"
	DomLiquidity  ContextDominance = "liquidity"
	DomMacro      ContextDominance = "macro"
	DomRegime     ContextDominance = "regime_transition"
	DomMixed      ContextDominance = "mixed"
	DomUnclear    ContextDominance = "unclear"
)

// SequencePosition represents where in a pattern sequence this episode sits
type SequencePosition string

const (
	SeqEarly      SequencePosition = "early"
	SeqMid        SequencePosition = "mid"
	SeqLate       SequencePosition = "late"
	SeqExhaustion SequencePosition = "exhaustion"
	SeqUnknown    SequencePosition = "unknown"
)

// MarketContext is a snapshot of market conditions at decision time
type MarketContext struct {
	Price          float64        `json:"price"`
	Volume         float64        `json:"volume,omitempty"`
	Volatility     float64        `json:"volatility,omitempty"`
	Spread         float64        `json:"spread,omitempty"`
	Regime         string         `json:"regime,omitempty"`
	Indicators     map[string]any `json:"indicators,omitempty"`
	MacroFlags     []string       `json:"macro_flags,omitempty"`
	LiquidityScore float64        `json:"liquidity_score,omitempty"`
}

// Expectations holds pre-action predictions
type Expectations struct {
	Direction       string         `json:"direction"`     // "up", "down", "sideways"
	MagnitudePct    float64        `json:"magnitude_pct"` // expected move %
	TimeHorizonMins int            `json:"time_horizon_mins"`
	VolatilityBand  [2]float64     `json:"volatility_band"` // [low, high]
	FailureModes    []string       `json:"failure_modes,omitempty"`
	ExtraFields     map[string]any `json:"extra,omitempty"`
}

// Outcome is filled in post-resolution
type Outcome struct {
	ActualDirection    string         `json:"actual_direction"`
	ActualMagnitudePct float64        `json:"actual_magnitude_pct"`
	ActualDurationMins int            `json:"actual_duration_mins"`
	Profitable         bool           `json:"profitable"`
	PnL                float64        `json:"pnl,omitempty"`
	ResolvedAt         time.Time      `json:"resolved_at"`
	Notes              string         `json:"notes,omitempty"`
	ExtraFields        map[string]any `json:"extra,omitempty"`
}

// DecayMetadata holds information for memory decay
type DecayMetadata struct {
	HalfLifeDays       float64   `json:"half_life_days"`
	LastDecayAt        time.Time `json:"last_decay_at"`
	ReinforcementCount int       `json:"reinforcement_count"`
	CurrentWeight      float64   `json:"current_weight"` // [0,1]
}

// Episode represents a single decision point (trade, abstention, or deferral)
type Episode struct {
	ID                 uuid.UUID        `json:"id"`
	EpisodeType        EpisodeType      `json:"episode_type"`
	Symbol             string           `json:"symbol"`
	StrategyName       string           `json:"strategy_name"`
	ArtifactID         *uuid.UUID       `json:"artifact_id,omitempty"`
	EpisodeAt          time.Time        `json:"episode_at"`
	Context            MarketContext    `json:"context"`
	Expectations       Expectations     `json:"expectations"`
	Confidence         float64          `json:"confidence"`         // [0,1]
	UncertaintyBudget  float64          `json:"uncertainty_budget"` // [0,1]
	ContextDominance   ContextDominance `json:"context_dominance"`
	SequencePosition   SequencePosition `json:"sequence_position"`
	ActionTaken        string           `json:"action_taken"` // buy/sell/abstain/defer
	Outcome            *Outcome         `json:"outcome,omitempty"`
	SurpriseScore      *float64         `json:"surprise_score,omitempty"` // [0,1]
	HindsightNotes     string           `json:"hindsight_notes,omitempty"`
	DecayWeight        float64          `json:"decay_weight"` // [0,1]
	ReinforcementCount int              `json:"reinforcement_count"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// NewEpisode creates a new episode at decision time
func NewEpisode(
	episodeType EpisodeType,
	symbol, strategyName string,
	ctx MarketContext,
	exp Expectations,
	confidence, uncertaintyBudget float64,
	dominance ContextDominance,
	seqPos SequencePosition,
	actionTaken string,
) *Episode {
	now := time.Now().UTC()
	return &Episode{
		ID:                 uuid.New(),
		EpisodeType:        episodeType,
		Symbol:             symbol,
		StrategyName:       strategyName,
		EpisodeAt:          now,
		Context:            ctx,
		Expectations:       exp,
		Confidence:         clamp(confidence, 0, 1),
		UncertaintyBudget:  clamp(uncertaintyBudget, 0, 1),
		ContextDominance:   dominance,
		SequencePosition:   seqPos,
		ActionTaken:        actionTaken,
		DecayWeight:        1.0,
		ReinforcementCount: 0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// ComputeSurprise computes how surprised Jax should be given the outcome.
// Returns a score [0,1] where 0 = fully expected, 1 = completely surprising.
func (e *Episode) ComputeSurprise(outcome Outcome) float64 {
	if e.Expectations.Direction == "" {
		return 0.5 // no expectations set, moderate surprise
	}
	var score float64

	// Directional mismatch [0,1]
	dirMatch := 0.0
	if e.Expectations.Direction == outcome.ActualDirection {
		dirMatch = 1.0
	}
	dirSurprise := 1.0 - dirMatch

	// Magnitude deviation (normalize to [0,1])
	expectedMag := math.Abs(e.Expectations.MagnitudePct)
	actualMag := math.Abs(outcome.ActualMagnitudePct)
	magDev := 0.0
	if expectedMag > 0 {
		magDev = math.Min(math.Abs(expectedMag-actualMag)/expectedMag, 1.0)
	}

	// Timing deviation (normalize to [0,1])
	timeDev := 0.0
	expected := float64(e.Expectations.TimeHorizonMins)
	actual := float64(outcome.ActualDurationMins)
	if expected > 0 {
		timeDev = math.Min(math.Abs(expected-actual)/expected, 1.0)
	}

	// Weighted combination: direction is most important
	score = 0.5*dirSurprise + 0.3*magDev + 0.2*timeDev
	return clamp(score, 0, 1)
}

// ApplyDecay applies time-based memory decay to this episode.
// halfLifeDays: number of days for weight to halve.
func (e *Episode) ApplyDecay(halfLifeDays float64) {
	if halfLifeDays <= 0 {
		halfLifeDays = 30
	}
	daysSince := time.Since(e.EpisodeAt).Hours() / 24.0
	// Exponential decay: weight = exp(-lambda * t) where lambda = ln(2) / halfLife
	lambda := math.Log(2) / halfLifeDays
	e.DecayWeight = math.Exp(-lambda * daysSince)
	e.UpdatedAt = time.Now().UTC()
}

// Reinforce increases this episode's weight (called when a similar episode occurs)
func (e *Episode) Reinforce(amount float64) {
	e.ReinforcementCount++
	e.DecayWeight = math.Min(1.0, e.DecayWeight+amount)
	e.UpdatedAt = time.Now().UTC()
}

// IsAbstention returns true if no trade was taken
func (e *Episode) IsAbstention() bool {
	return e.EpisodeType == EpisodeAbstention || e.ActionTaken == "abstain" || e.ActionTaken == "defer"
}

// clamp restricts v to [min, max]
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ValidEpisodeTypes is the set of valid episode types
var ValidEpisodeTypes = map[EpisodeType]bool{
	EpisodeTrade:      true,
	EpisodeAbstention: true,
	EpisodeDeferral:   true,
}

// Validate checks required fields
func (e *Episode) Validate() error {
	if !ValidEpisodeTypes[e.EpisodeType] {
		return fmt.Errorf("invalid episode type: %s", e.EpisodeType)
	}
	if e.Symbol == "" {
		return fmt.Errorf("symbol required")
	}
	if e.StrategyName == "" {
		return fmt.Errorf("strategy_name required")
	}
	if e.Confidence < 0 || e.Confidence > 1 {
		return fmt.Errorf("confidence must be [0,1], got %f", e.Confidence)
	}
	if e.UncertaintyBudget < 0 || e.UncertaintyBudget > 1 {
		return fmt.Errorf("uncertainty_budget must be [0,1]")
	}
	return nil
}

// ContextJSON serializes context for DB storage
func (e *Episode) ContextJSON() ([]byte, error) {
	return json.Marshal(e.Context)
}

// ExpectationsJSON serializes expectations for DB storage
func (e *Episode) ExpectationsJSON() ([]byte, error) {
	return json.Marshal(e.Expectations)
}

// OutcomeJSON serializes outcome for DB storage (nil-safe)
func (e *Episode) OutcomeJSON() ([]byte, error) {
	if e.Outcome == nil {
		return nil, nil
	}
	return json.Marshal(e.Outcome)
}
