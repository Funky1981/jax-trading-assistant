package ejlayer

import (
	"testing"
	"time"
)

func makeEpisode(confidence float64) *Episode {
	return NewEpisode(
		EpisodeTrade,
		"AAPL", "momentum",
		MarketContext{Price: 150.0},
		Expectations{Direction: "up", MagnitudePct: 2.0, TimeHorizonMins: 60},
		confidence, 0.8,
		DomTechnical, SeqMid,
		"buy",
	)
}

// TestNewEpisode verifies correct defaults on creation
func TestNewEpisode(t *testing.T) {
	ep := makeEpisode(0.7)
	if ep.ID.String() == "" {
		t.Error("expected non-empty ID")
	}
	if ep.EpisodeType != EpisodeTrade {
		t.Errorf("expected EpisodeTrade, got %s", ep.EpisodeType)
	}
	if ep.Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", ep.Symbol)
	}
	if ep.DecayWeight != 1.0 {
		t.Errorf("expected DecayWeight=1.0, got %f", ep.DecayWeight)
	}
	if ep.ReinforcementCount != 0 {
		t.Errorf("expected ReinforcementCount=0, got %d", ep.ReinforcementCount)
	}
	if ep.Confidence != 0.7 {
		t.Errorf("expected Confidence=0.7, got %f", ep.Confidence)
	}
}

// TestComputeSurprise verifies surprise computation
func TestComputeSurprise(t *testing.T) {
	ep := makeEpisode(0.7)

	// Direction match: low surprise (only magnitude and time deviation contribute)
	outcome := Outcome{
		ActualDirection:    "up",
		ActualMagnitudePct: 2.0,
		ActualDurationMins: 60,
	}
	s := ep.ComputeSurprise(outcome)
	if s > 0.2 {
		t.Errorf("expected low surprise for matching outcome, got %f", s)
	}

	// Direction opposite: high surprise
	outcome2 := Outcome{
		ActualDirection:    "down",
		ActualMagnitudePct: 2.0,
		ActualDurationMins: 60,
	}
	s2 := ep.ComputeSurprise(outcome2)
	if s2 < 0.4 {
		t.Errorf("expected high surprise for opposite direction, got %f", s2)
	}

	// No expectations: moderate
	epNoExp := &Episode{}
	s3 := epNoExp.ComputeSurprise(outcome)
	if s3 != 0.5 {
		t.Errorf("expected 0.5 for no expectations, got %f", s3)
	}
}

// TestApplyDecay verifies exponential decay
func TestApplyDecay(t *testing.T) {
	ep := makeEpisode(0.7)

	// Freshly minted should be near 1.0 after decay with 30-day half-life
	ep.ApplyDecay(30)
	if ep.DecayWeight < 0.99 {
		t.Errorf("fresh episode should have decay_weight near 1.0, got %f", ep.DecayWeight)
	}

	// Simulate 30 days ago — weight should be near 0.5
	ep2 := makeEpisode(0.7)
	ep2.EpisodeAt = time.Now().UTC().AddDate(0, 0, -30)
	ep2.ApplyDecay(30)
	if ep2.DecayWeight < 0.45 || ep2.DecayWeight > 0.55 {
		t.Errorf("30-day old episode with 30-day half-life should have weight ~0.5, got %f", ep2.DecayWeight)
	}
}

// TestReinforce verifies reinforcement increments weight and count
func TestReinforce(t *testing.T) {
	ep := makeEpisode(0.7)
	ep.DecayWeight = 0.5

	ep.Reinforce(0.2)
	if ep.ReinforcementCount != 1 {
		t.Errorf("expected ReinforcementCount=1, got %d", ep.ReinforcementCount)
	}
	if ep.DecayWeight != 0.7 {
		t.Errorf("expected DecayWeight=0.7, got %f", ep.DecayWeight)
	}

	// Reinforce to cap at 1.0
	ep.Reinforce(0.5)
	if ep.DecayWeight > 1.0 {
		t.Errorf("DecayWeight should not exceed 1.0, got %f", ep.DecayWeight)
	}
}

// TestShouldAbstain covers the abstention thresholds
func TestShouldAbstain(t *testing.T) {
	tests := []struct {
		conf   float64
		budget float64
		want   bool
	}{
		{0.1, 0.8, true},    // low confidence
		{0.8, 0.1, true},    // depleted budget
		{0.24, 0.5, true},   // just below confidence threshold
		{0.25, 0.30, false}, // exactly at thresholds
		{0.8, 0.8, false},   // healthy
	}
	for _, tc := range tests {
		got := ShouldAbstain(tc.conf, tc.budget)
		if got != tc.want {
			t.Errorf("ShouldAbstain(%.2f, %.2f) = %v, want %v", tc.conf, tc.budget, got, tc.want)
		}
	}
}

// TestWeightedConfidence covers aggregation of episode history
func TestWeightedConfidence(t *testing.T) {
	// No history → 0.5
	if c := WeightedConfidence(nil); c != 0.5 {
		t.Errorf("no history: expected 0.5, got %f", c)
	}
	if c := WeightedConfidence([]*Episode{}); c != 0.5 {
		t.Errorf("empty history: expected 0.5, got %f", c)
	}

	// High-confidence episodes → high confidence
	eps := []*Episode{makeEpisode(0.9), makeEpisode(0.85), makeEpisode(0.88)}
	c := WeightedConfidence(eps)
	if c < 0.8 {
		t.Errorf("high-confidence episodes: expected >0.8, got %f", c)
	}

	// Low-confidence episodes → low confidence
	eps2 := []*Episode{makeEpisode(0.1), makeEpisode(0.15)}
	c2 := WeightedConfidence(eps2)
	if c2 > 0.3 {
		t.Errorf("low-confidence episodes: expected <0.3, got %f", c2)
	}
}

// TestSurpriseAdjustedConfidence verifies high surprise reduces confidence
func TestSurpriseAdjustedConfidence(t *testing.T) {
	base := 0.8
	// No surprise: unchanged
	c := SurpriseAdjustedConfidence(base, nil)
	if c != base {
		t.Errorf("no surprise: expected %f, got %f", base, c)
	}

	// High surprise: reduced
	c2 := SurpriseAdjustedConfidence(base, []float64{0.9, 0.9, 0.9})
	if c2 >= base {
		t.Errorf("high surprise should reduce confidence, got %f", c2)
	}
	if c2 < 0 {
		t.Errorf("confidence should not go negative, got %f", c2)
	}
}

// TestComputeUncertaintyBudget covers the budget calculator
func TestComputeUncertaintyBudget(t *testing.T) {
	// No factors → full budget
	if b := ComputeUncertaintyBudget(nil); b != 1.0 {
		t.Errorf("no factors: expected 1.0, got %f", b)
	}
	if b := ComputeUncertaintyBudget([]UncertaintyFactor{}); b != 1.0 {
		t.Errorf("empty factors: expected 1.0, got %f", b)
	}

	// Fully drained (one factor with weight=1, score=1)
	full := []UncertaintyFactor{{Name: "x", Weight: 1.0, Score: 1.0}}
	if b := ComputeUncertaintyBudget(full); b != 0.0 {
		t.Errorf("fully drained: expected 0.0, got %f", b)
	}

	// Partial drain
	partial := []UncertaintyFactor{{Name: "y", Weight: 0.5, Score: 0.5}}
	b := ComputeUncertaintyBudget(partial)
	if b < 0.7 || b > 0.8 {
		t.Errorf("partial drain: expected ~0.75, got %f", b)
	}
}

// TestEpisodeValidate covers the validation rules
func TestEpisodeValidate(t *testing.T) {
	// Invalid episode type
	ep := makeEpisode(0.5)
	ep.EpisodeType = "bad_type"
	if err := ep.Validate(); err == nil {
		t.Error("expected error for invalid episode type")
	}

	// Empty symbol
	ep2 := makeEpisode(0.5)
	ep2.Symbol = ""
	if err := ep2.Validate(); err == nil {
		t.Error("expected error for empty symbol")
	}

	// Empty strategy
	ep3 := makeEpisode(0.5)
	ep3.StrategyName = ""
	if err := ep3.Validate(); err == nil {
		t.Error("expected error for empty strategy_name")
	}

	// Out-of-range confidence
	ep4 := makeEpisode(0.5)
	ep4.Confidence = 1.5
	if err := ep4.Validate(); err == nil {
		t.Error("expected error for confidence > 1")
	}

	// Valid episode
	ep5 := makeEpisode(0.5)
	if err := ep5.Validate(); err != nil {
		t.Errorf("expected valid episode, got: %v", err)
	}
}
