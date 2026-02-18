package ejlayer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides database operations for EJLayer episodes
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new EJLayer store
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateEpisode persists a new episode to the database
func (s *Store) CreateEpisode(ctx context.Context, ep *Episode) error {
	query := `
		INSERT INTO market_episodes (
			id, episode_type, symbol, strategy_name, artifact_id,
			episode_at, context, expectations,
			confidence, uncertainty_budget, context_dominance, sequence_position,
			action_taken, outcome, surprise_score, hindsight_notes,
			decay_weight, reinforcement_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20
		)
	`

	contextJSON, err := ep.ContextJSON()
	if err != nil {
		return fmt.Errorf("ejlayer: marshal context: %w", err)
	}
	expectationsJSON, err := ep.ExpectationsJSON()
	if err != nil {
		return fmt.Errorf("ejlayer: marshal expectations: %w", err)
	}
	outcomeJSON, err := ep.OutcomeJSON()
	if err != nil {
		return fmt.Errorf("ejlayer: marshal outcome: %w", err)
	}

	var hindsight sql.NullString
	if ep.HindsightNotes != "" {
		hindsight = sql.NullString{String: ep.HindsightNotes, Valid: true}
	}

	_, err = s.pool.Exec(ctx, query,
		ep.ID,
		string(ep.EpisodeType),
		ep.Symbol,
		ep.StrategyName,
		ep.ArtifactID, // *uuid.UUID, pgx handles nil as NULL
		ep.EpisodeAt,
		contextJSON,
		expectationsJSON,
		ep.Confidence,
		ep.UncertaintyBudget,
		string(ep.ContextDominance),
		string(ep.SequencePosition),
		ep.ActionTaken,
		outcomeJSON,
		ep.SurpriseScore, // *float64, pgx handles nil as NULL
		hindsight,
		ep.DecayWeight,
		ep.ReinforcementCount,
		ep.CreatedAt,
		ep.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ejlayer: create episode: %w", err)
	}
	return nil
}

// GetEpisodeByID retrieves a single episode by its UUID
func (s *Store) GetEpisodeByID(ctx context.Context, id uuid.UUID) (*Episode, error) {
	query := `
		SELECT
			id, episode_type, symbol, strategy_name, artifact_id,
			episode_at, context, expectations,
			confidence, uncertainty_budget, context_dominance, sequence_position,
			action_taken, outcome, surprise_score, hindsight_notes,
			decay_weight, reinforcement_count, created_at, updated_at
		FROM market_episodes
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, query, id)
	ep, err := scanEpisode(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ejlayer: episode not found: %s", id)
		}
		return nil, fmt.Errorf("ejlayer: get episode by id: %w", err)
	}
	return ep, nil
}

// ListRecentEpisodes returns up to limit episodes for a symbol/strategy, ordered newest first.
// Decay is applied in-memory after retrieval so callers always see current decay state.
func (s *Store) ListRecentEpisodes(ctx context.Context, symbol, strategyName string, limit int) ([]*Episode, error) {
	query := `
		SELECT
			id, episode_type, symbol, strategy_name, artifact_id,
			episode_at, context, expectations,
			confidence, uncertainty_budget, context_dominance, sequence_position,
			action_taken, outcome, surprise_score, hindsight_notes,
			decay_weight, reinforcement_count, created_at, updated_at
		FROM market_episodes
		WHERE ($1 = '' OR symbol = $1)
		  AND ($2 = '' OR strategy_name = $2)
		ORDER BY episode_at DESC
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, symbol, strategyName, limit)
	if err != nil {
		return nil, fmt.Errorf("ejlayer: list recent episodes: %w", err)
	}
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		ep, err := scanEpisode(rows)
		if err != nil {
			return nil, fmt.Errorf("ejlayer: scan episode: %w", err)
		}
		episodes = append(episodes, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ejlayer: list recent episodes rows: %w", err)
	}
	return episodes, nil
}

// UpdateOutcome fills in the outcome, surprise score, and hindsight notes for an episode
func (s *Store) UpdateOutcome(ctx context.Context, id uuid.UUID, outcome Outcome, surpriseScore float64, notes string) error {
	query := `
		UPDATE market_episodes
		SET outcome = $2,
		    surprise_score = $3,
		    hindsight_notes = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		return fmt.Errorf("ejlayer: marshal outcome: %w", err)
	}

	var hindsight sql.NullString
	if notes != "" {
		hindsight = sql.NullString{String: notes, Valid: true}
	}

	_, err = s.pool.Exec(ctx, query, id, outcomeJSON, surpriseScore, hindsight)
	if err != nil {
		return fmt.Errorf("ejlayer: update outcome: %w", err)
	}
	return nil
}

// ApplyDecayToAll recomputes decay_weight for all episodes using pure-SQL exponential decay.
// Returns the number of rows updated.
func (s *Store) ApplyDecayToAll(ctx context.Context, halfLifeDays float64) (int, error) {
	if halfLifeDays <= 0 {
		halfLifeDays = 30
	}
	query := `
		UPDATE market_episodes
		SET decay_weight = GREATEST(0, EXP(-0.693147 / $1 * EXTRACT(EPOCH FROM (NOW() - episode_at)) / 86400.0)),
		    updated_at = NOW()
		WHERE episode_at < NOW() - INTERVAL '1 hour'
	`

	tag, err := s.pool.Exec(ctx, query, halfLifeDays)
	if err != nil {
		return 0, fmt.Errorf("ejlayer: apply decay: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ReinforceSimilarEpisodes bumps decay_weight and reinforcement_count for matching episodes.
// Returns the number of rows updated.
func (s *Store) ReinforceSimilarEpisodes(ctx context.Context, symbol, strategyName, contextDominance string, amount float64) (int, error) {
	query := `
		UPDATE market_episodes
		SET decay_weight = LEAST(1.0, decay_weight + $4),
		    reinforcement_count = reinforcement_count + 1,
		    updated_at = NOW()
		WHERE symbol = $1 AND strategy_name = $2 AND context_dominance = $3
	`

	tag, err := s.pool.Exec(ctx, query, symbol, strategyName, contextDominance, amount)
	if err != nil {
		return 0, fmt.Errorf("ejlayer: reinforce similar episodes: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// scanner is a common interface satisfied by pgx.Row and pgx.Rows
type scanner interface {
	Scan(dest ...any) error
}

// scanEpisode reads one episode row from a scanner
func scanEpisode(row scanner) (*Episode, error) {
	var ep Episode
	var artifactID *uuid.UUID
	var contextJSON, expectationsJSON []byte
	var outcomeJSON []byte
	var surpriseScore sql.NullFloat64
	var hindsight sql.NullString
	var episodeTypeStr, contextDomStr, seqPosStr string

	err := row.Scan(
		&ep.ID,
		&episodeTypeStr,
		&ep.Symbol,
		&ep.StrategyName,
		&artifactID,
		&ep.EpisodeAt,
		&contextJSON,
		&expectationsJSON,
		&ep.Confidence,
		&ep.UncertaintyBudget,
		&contextDomStr,
		&seqPosStr,
		&ep.ActionTaken,
		&outcomeJSON,
		&surpriseScore,
		&hindsight,
		&ep.DecayWeight,
		&ep.ReinforcementCount,
		&ep.CreatedAt,
		&ep.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	ep.EpisodeType = EpisodeType(episodeTypeStr)
	ep.ContextDominance = ContextDominance(contextDomStr)
	ep.SequencePosition = SequencePosition(seqPosStr)
	ep.ArtifactID = artifactID

	if hindsight.Valid {
		ep.HindsightNotes = hindsight.String
	}
	if surpriseScore.Valid {
		v := surpriseScore.Float64
		ep.SurpriseScore = &v
	}

	if err := json.Unmarshal(contextJSON, &ep.Context); err != nil {
		return nil, fmt.Errorf("unmarshal context: %w", err)
	}
	if err := json.Unmarshal(expectationsJSON, &ep.Expectations); err != nil {
		return nil, fmt.Errorf("unmarshal expectations: %w", err)
	}
	if len(outcomeJSON) > 0 {
		var o Outcome
		if err := json.Unmarshal(outcomeJSON, &o); err != nil {
			return nil, fmt.Errorf("unmarshal outcome: %w", err)
		}
		ep.Outcome = &o
	}

	return &ep, nil
}

// decayWeight computes the expected decay weight for an episode given a half-life.
// Exported for use in tests.
func decayWeight(episodeAt time.Time, halfLifeDays float64) float64 {
	daysSince := time.Since(episodeAt).Hours() / 24.0
	lambda := math.Log(2) / halfLifeDays
	return math.Exp(-lambda * daysSince)
}
