package ejlayer

import (
	"context"
	"fmt"

	domain "jax-trading-assistant/internal/domain/ejlayer"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Recorder writes episodes to the database
type Recorder struct {
	store *domain.Store
}

// NewRecorder creates a Recorder backed by the given pool
func NewRecorder(pool *pgxpool.Pool) *Recorder {
	return &Recorder{store: domain.NewStore(pool)}
}

// RecordDecision creates a new episode at decision time
func (r *Recorder) RecordDecision(ctx context.Context, episode *domain.Episode) error {
	if err := episode.Validate(); err != nil {
		return fmt.Errorf("ejlayer recorder: invalid episode: %w", err)
	}
	return r.store.CreateEpisode(ctx, episode)
}

// RecordOutcome fills in the outcome and computes surprise for a past episode
func (r *Recorder) RecordOutcome(ctx context.Context, episodeID uuid.UUID, outcome domain.Outcome, hindsightNotes string) error {
	// Load the episode to compute surprise
	ep, err := r.store.GetEpisodeByID(ctx, episodeID)
	if err != nil {
		return fmt.Errorf("ejlayer recorder: episode %s not found: %w", episodeID, err)
	}

	surprise := ep.ComputeSurprise(outcome)
	return r.store.UpdateOutcome(ctx, episodeID, outcome, surprise, hindsightNotes)
}

// ReinforceSimilar signals that a similar context occurred, reinforcing related episodes
func (r *Recorder) ReinforceSimilar(ctx context.Context, symbol, strategyName string, dominance domain.ContextDominance) (int, error) {
	n, err := r.store.ReinforceSimilarEpisodes(ctx, symbol, strategyName, string(dominance), 0.05)
	if err != nil {
		return 0, fmt.Errorf("ejlayer recorder: reinforce failed: %w", err)
	}
	return n, nil
}

// RunDecay triggers decay across all episodes
func (r *Recorder) RunDecay(ctx context.Context, halfLifeDays float64) (int, error) {
	n, err := r.store.ApplyDecayToAll(ctx, halfLifeDays)
	if err != nil {
		return 0, fmt.Errorf("ejlayer recorder: decay failed: %w", err)
	}
	return n, nil
}
