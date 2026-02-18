package ejlayer

import "math"

// UncertaintyFactor represents something draining the uncertainty budget
type UncertaintyFactor struct {
	Name   string
	Weight float64 // how much it drains [0,1]
	Score  float64 // current intensity [0,1]
}

// ComputeUncertaintyBudget computes the remaining uncertainty budget given draining factors.
// Returns [0,1]; lower = more uncertain = closer to abstention.
func ComputeUncertaintyBudget(factors []UncertaintyFactor) float64 {
	if len(factors) == 0 {
		return 1.0
	}
	totalDrain := 0.0
	for _, f := range factors {
		totalDrain += clamp(f.Weight, 0, 1) * clamp(f.Score, 0, 1)
	}
	budget := 1.0 - clamp(totalDrain, 0, 1)
	return clamp(budget, 0, 1)
}

// ShouldAbstain returns true if Jax should abstain given confidence and uncertainty budget.
// Thresholds are conservative by design.
func ShouldAbstain(confidence, uncertaintyBudget float64) bool {
	const (
		MinConfidence        = 0.25
		MinUncertaintyBudget = 0.30
	)
	return confidence < MinConfidence || uncertaintyBudget < MinUncertaintyBudget
}

// WeightedConfidence computes confidence from a slice of recent episodes.
// More recent and more reinforced episodes get higher weight.
func WeightedConfidence(episodes []*Episode) float64 {
	if len(episodes) == 0 {
		return 0.5 // neutral when no history
	}
	totalWeight := 0.0
	weightedSum := 0.0
	for _, ep := range episodes {
		w := ep.DecayWeight * float64(1+ep.ReinforcementCount)
		weightedSum += ep.Confidence * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0.5
	}
	return clamp(weightedSum/totalWeight, 0, 1)
}

// SurpriseAdjustedConfidence reduces confidence when recent surprise is high.
func SurpriseAdjustedConfidence(baseConfidence float64, recentSurpriseScores []float64) float64 {
	if len(recentSurpriseScores) == 0 {
		return baseConfidence
	}
	totalSurprise := 0.0
	for _, s := range recentSurpriseScores {
		totalSurprise += s
	}
	avgSurprise := totalSurprise / float64(len(recentSurpriseScores))
	// High surprise reduces confidence (penalty scales with surprise)
	penalty := avgSurprise * 0.3
	return clamp(baseConfidence-penalty, 0, 1)
}

// NegativePatternPenalty computes confidence reduction from active negative patterns.
func NegativePatternPenalty(patterns []NegativePattern, decayThreshold float64) float64 {
	total := 0.0
	for _, p := range patterns {
		if p.DecayWeight > decayThreshold {
			// Apply pattern's penalty, weighted by decay
			total += p.ReducesConfidenceBy * p.DecayWeight
		}
	}
	return math.Min(total, 0.5) // cap penalty at -0.5 confidence
}

// NegativePattern is an in-memory representation of a cautionary pattern
type NegativePattern struct {
	Name                string
	ReducesConfidenceBy float64
	DecayWeight         float64
	ReinforcementCount  int
	MatchCriteria       map[string]any
}
