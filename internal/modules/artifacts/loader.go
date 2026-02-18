package artifacts

import (
	"context"
	"fmt"
	"log"

	"jax-trading-assistant/internal/domain/artifacts"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Loader loads approved strategy artifacts from the database
// and registers them with the strategy registry
type Loader struct {
	store    *artifacts.Store
	registry *strategies.Registry
}

// NewLoader creates a new artifact loader
func NewLoader(pool *pgxpool.Pool, registry *strategies.Registry) *Loader {
	return &Loader{
		store:    artifacts.NewStore(pool),
		registry: registry,
	}
}

// LoadApprovedStrategies loads all approved strategy artifacts and registers them
//
// Exit criteria: Trader refuses non-approved artifacts
// This method implements the "only load APPROVED artifacts" gate
func (l *Loader) LoadApprovedStrategies(ctx context.Context) error {
	log.Println("loading approved strategy artifacts...")

	// Load all approved artifacts from database
	approvedArtifacts, err := l.store.ListApprovedArtifacts(ctx)
	if err != nil {
		return fmt.Errorf("failed to load approved artifacts: %w", err)
	}

	if len(approvedArtifacts) == 0 {
		log.Println("⚠️  no approved artifacts found - trader will not generate signals")
		log.Println("   create and approve artifacts using the artifact API")
		return nil
	}

	// Load and register each artifact
	loaded := 0
	for _, artifact := range approvedArtifacts {
		if err := l.loadArtifact(ctx, artifact); err != nil {
			log.Printf("warning: failed to load artifact %s: %v (skipping)", artifact.ArtifactID, err)
			continue
		}
		loaded++
	}

	log.Printf("loaded %d approved strategy artifacts", loaded)
	return nil
}

// loadArtifact loads a single artifact and registers the strategy
func (l *Loader) loadArtifact(ctx context.Context, artifact *artifacts.Artifact) error {
	// 1. Verify hash (immutability guarantee)
	if err := artifact.VerifyHash(); err != nil {
		return fmt.Errorf("hash verification failed: %w", err)
	}

	// 2. Create and register strategy from artifact
	// Each strategy type knows how to register itself with metadata
	if err := l.registerStrategyFromArtifact(artifact); err != nil {
		return fmt.Errorf("failed to register strategy: %w", err)
	}

	log.Printf("  ✓ loaded %s v%s (artifact: %s, hash: %s...)",
		artifact.Strategy.Name,
		artifact.Strategy.Version,
		artifact.ArtifactID,
		artifact.Hash[:8])

	return nil
}

// registerStrategyFromArtifact creates and registers a strategy from an artifact
func (l *Loader) registerStrategyFromArtifact(artifact *artifacts.Artifact) error {
	// Map strategy name to strategy constructor and register
	// Note: We create strategies using their standard constructors.
	// Artifact parameters could be used to configure strategies in the future,
	// but for Phase 4, we're just loading approved versions of standard strategies.
	switch artifact.Strategy.Name {
	case "rsi_momentum":
		strategy := strategies.NewRSIMomentumStrategy()
		metadata := strategy.GetMetadata()
		// ADR-0012 Phase 4: Add artifact tracking to metadata
		metadata.Extra = map[string]interface{}{
			"artifact_id":   artifact.ArtifactID,
			"artifact_hash": artifact.Hash,
		}
		return l.registry.Register(strategy, metadata)
	case "macd_crossover":
		strategy := strategies.NewMACDCrossoverStrategy()
		metadata := strategy.GetMetadata()
		metadata.Extra = map[string]interface{}{
			"artifact_id":   artifact.ArtifactID,
			"artifact_hash": artifact.Hash,
		}
		return l.registry.Register(strategy, metadata)
	case "ma_crossover":
		strategy := strategies.NewMACrossoverStrategy()
		metadata := strategy.GetMetadata()
		metadata.Extra = map[string]interface{}{
			"artifact_id":   artifact.ArtifactID,
			"artifact_hash": artifact.Hash,
		}
		return l.registry.Register(strategy, metadata)
	default:
		return fmt.Errorf("unknown strategy: %s", artifact.Strategy.Name)
	}
}

// GetLatestApprovedArtifact returns the latest approved artifact for a strategy
func (l *Loader) GetLatestApprovedArtifact(ctx context.Context, strategyName string) (*artifacts.Artifact, error) {
	artifact, err := l.store.GetLatestApprovedArtifact(ctx, strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest approved artifact for %s: %w", strategyName, err)
	}

	// Verify hash before returning
	if err := artifact.VerifyHash(); err != nil {
		return nil, fmt.Errorf("artifact hash verification failed: %w", err)
	}

	return artifact, nil
}

// GetArtifactByID returns an artifact by ID
func (l *Loader) GetArtifactByID(ctx context.Context, id uuid.UUID) (*artifacts.Artifact, error) {
	artifact, err := l.store.GetArtifactByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact %s: %w", id, err)
	}

	// Verify hash before returning
	if err := artifact.VerifyHash(); err != nil {
		return nil, fmt.Errorf("artifact hash verification failed: %w", err)
	}

	return artifact, nil
}

// RefreshStrategies reloads approved artifacts (for SIGHUP or admin endpoint)
func (l *Loader) RefreshStrategies(ctx context.Context) error {
	log.Println("refreshing approved strategy artifacts...")

	// Clear existing strategies
	// Note: Registry doesn't have a Clear() method, so we'll just re-register
	// This will overwrite existing registrations with the same name

	// Reload approved artifacts
	if err := l.LoadApprovedStrategies(ctx); err != nil {
		return fmt.Errorf("failed to refresh strategies: %w", err)
	}

	return nil
}
