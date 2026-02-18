package ejlayer

import (
	"context"
	"fmt"
	"log"

	domain "jax-trading-assistant/internal/domain/ejlayer"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Evaluation holds the EJLayer's assessment before a decision
type Evaluation struct {
	Symbol            string
	StrategyName      string
	Confidence        float64
	UncertaintyBudget float64
	ShouldAbstain     bool
	AbstainReason     string
	ContextDominance  domain.ContextDominance
	RecentSurpriseAvg float64
	EpisodeCount      int
}

// Evaluator queries episode history to produce an evaluation
type Evaluator struct {
	store *domain.Store
}

// NewEvaluator creates an Evaluator backed by the given pool
func NewEvaluator(pool *pgxpool.Pool) *Evaluator {
	return &Evaluator{store: domain.NewStore(pool)}
}

// Evaluate produces a confidence/uncertainty assessment for a symbol + strategy
func (ev *Evaluator) Evaluate(ctx context.Context, symbol, strategyName string, factors []domain.UncertaintyFactor) (*Evaluation, error) {
	// Load recent episodes
	episodes, err := ev.store.ListRecentEpisodes(ctx, symbol, strategyName, 20)
	if err != nil {
		return nil, fmt.Errorf("ejlayer: failed to load episodes for %s/%s: %w", symbol, strategyName, err)
	}

	// Compute confidence from episode history
	baseConf := domain.WeightedConfidence(episodes)

	// Collect recent surprise scores
	var surprises []float64
	for _, ep := range episodes {
		if ep.SurpriseScore != nil {
			surprises = append(surprises, *ep.SurpriseScore)
		}
	}
	conf := domain.SurpriseAdjustedConfidence(baseConf, surprises)

	// Compute uncertainty budget
	budget := domain.ComputeUncertaintyBudget(factors)

	abstain := domain.ShouldAbstain(conf, budget)
	reason := ""
	if abstain {
		if conf < 0.25 {
			reason = fmt.Sprintf("confidence too low (%.2f < 0.25)", conf)
		} else {
			reason = fmt.Sprintf("uncertainty budget depleted (%.2f < 0.30)", budget)
		}
	}

	// Derive context dominance from most frequent tag in recent episodes
	dominance := deriveDominance(episodes)

	// Average surprise
	avgSurprise := 0.0
	for _, s := range surprises {
		avgSurprise += s
	}
	if len(surprises) > 0 {
		avgSurprise /= float64(len(surprises))
	}

	log.Printf("ejlayer: %s/%s conf=%.3f budget=%.3f abstain=%v", symbol, strategyName, conf, budget, abstain)

	return &Evaluation{
		Symbol:            symbol,
		StrategyName:      strategyName,
		Confidence:        conf,
		UncertaintyBudget: budget,
		ShouldAbstain:     abstain,
		AbstainReason:     reason,
		ContextDominance:  dominance,
		RecentSurpriseAvg: avgSurprise,
		EpisodeCount:      len(episodes),
	}, nil
}

// deriveDominance returns the most common ContextDominance among recent episodes
func deriveDominance(episodes []*domain.Episode) domain.ContextDominance {
	counts := map[domain.ContextDominance]int{}
	for _, ep := range episodes {
		counts[ep.ContextDominance]++
	}
	best := domain.DomUnclear
	maxCount := 0
	for k, v := range counts {
		if v > maxCount {
			maxCount = v
			best = k
		}
	}
	return best
}
