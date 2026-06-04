package llmcontext

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryQuery struct {
	TaskType   TaskType
	Symbol     string
	StrategyID string
	Limit      int
	Now        time.Time
	MaxAge     time.Duration
}

type MemoryArtifactStore struct {
	mu        sync.Mutex
	artifacts []MemoryArtifact
}

func NewMemoryArtifactStore() *MemoryArtifactStore {
	return &MemoryArtifactStore{}
}

func (s *MemoryArtifactStore) Save(_ context.Context, artifact MemoryArtifact) error {
	if strings.TrimSpace(artifact.ID) == "" {
		return fmt.Errorf("memory artifact id required")
	}
	if len(artifact.SourceIDs) == 0 {
		return fmt.Errorf("memory artifact source IDs required")
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifacts = append(s.artifacts, artifact)
	return nil
}

func (s *MemoryArtifactStore) Retrieve(_ context.Context, query MemoryQuery) ([]MemoryArtifact, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 5
	}
	now := query.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MemoryArtifact, 0, limit)
	for _, artifact := range s.artifacts {
		if query.TaskType != "" && artifact.TaskType != query.TaskType {
			continue
		}
		if query.Symbol != "" && !strings.EqualFold(artifact.Symbol, query.Symbol) {
			continue
		}
		if query.StrategyID != "" && artifact.StrategyID != query.StrategyID {
			continue
		}
		if query.MaxAge > 0 && now.Sub(artifact.CreatedAt) > query.MaxAge {
			continue
		}
		out = append(out, artifact)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Quality == out[j].Quality {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].Quality > out[j].Quality
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type PostgresMemoryArtifactStore struct {
	db SQLExecutor
}

func NewPostgresMemoryArtifactStore(db SQLExecutor) PostgresMemoryArtifactStore {
	return PostgresMemoryArtifactStore{db: db}
}

func (s PostgresMemoryArtifactStore) Save(ctx context.Context, artifact MemoryArtifact) error {
	if s.db == nil {
		return fmt.Errorf("postgres memory artifact store requires database handle")
	}
	if strings.TrimSpace(artifact.ID) == "" {
		return fmt.Errorf("memory artifact id required")
	}
	if len(artifact.SourceIDs) == 0 {
		return fmt.Errorf("memory artifact source IDs required")
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	sourceIDs, err := json.Marshal(artifact.SourceIDs)
	if err != nil {
		return fmt.Errorf("marshal source IDs: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO llm_memory_artifacts (
			id, task_type, symbol, strategy_id, summary, source_ids, quality, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8
		)
		ON CONFLICT (id) DO UPDATE SET
			task_type = EXCLUDED.task_type,
			symbol = EXCLUDED.symbol,
			strategy_id = EXCLUDED.strategy_id,
			summary = EXCLUDED.summary,
			source_ids = EXCLUDED.source_ids,
			quality = EXCLUDED.quality,
			created_at = EXCLUDED.created_at
	`, artifact.ID, artifact.TaskType, artifact.Symbol, artifact.StrategyID, artifact.Summary, sourceIDs, artifact.Quality, artifact.CreatedAt)
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresMemoryArtifactStore.Save: %w", err)
	}
	return nil
}
