// Package artifacts provides Builder — a service layer for creating strategy artifacts
// from backtest results. This is the ADR-0012 Phase 5 artifact builder component.
//
// Typical flow:
//  1. Research runtime runs a backtest via internal/modules/backtest.Engine
//  2. Builder.BuildFromBacktest() converts the result to an immutable Artifact
//  3. Artifact is saved to Postgres in DRAFT state with a SHA-256 hash
//  4. Human approver uses cmd/artifact-approver to promote to APPROVED
//  5. Trader runtime loads only APPROVED artifacts
package artifacts

import (
	"context"
	"fmt"
	"time"

	domainArtifacts "jax-trading-assistant/internal/domain/artifacts"
	"jax-trading-assistant/internal/modules/backtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// artifactPersister is the minimal store surface the Builder needs.
// Using an interface keeps BuildFromBacktest testable without a real DB.
type artifactPersister interface {
	CreateArtifact(ctx context.Context, artifact *domainArtifacts.Artifact) error
	CreateApproval(ctx context.Context, approval *domainArtifacts.Approval) error
}

// Builder creates and persists strategy artifacts from backtest results.
type Builder struct {
	store artifactPersister
}

// NewBuilder creates a new Builder backed by the given database pool.
func NewBuilder(pool *pgxpool.Pool) *Builder {
	return &Builder{store: domainArtifacts.NewStore(pool)}
}

// NewBuilderWithStore creates a Builder using a custom persister — useful for testing.
func NewBuilderWithStore(s artifactPersister) *Builder {
	return &Builder{store: s}
}

// BuildFromBacktest converts a backtest Result into an immutable Artifact,
// persists it with a SHA-256 hash in DRAFT state, and returns it.
//
// Parameters:
//   - strategyVersion: semver string (e.g. "1.2.0")
//   - params: strategy configuration parameters stored in the artifact for replay
//   - result: the output of backtest.Engine.Run()
//   - riskProfile: risk constraints embedded in the artifact (read only by trader)
//   - createdBy: service or user identifier for audit trail
func (b *Builder) BuildFromBacktest(
	ctx context.Context,
	strategyVersion string,
	params map[string]any,
	result *backtest.Result,
	riskProfile domainArtifacts.RiskProfile,
	createdBy string,
) (*domainArtifacts.Artifact, error) {
	now := time.Now().UTC()

	// Derive a stable UUID from the run ID so the same backtest always maps to
	// the same ValidationInfo.BacktestRunID (deterministic replay validation).
	runUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(result.RunID))

	symbols := result.Symbols
	if len(symbols) == 0 && result.Symbol != "" {
		symbols = []string{result.Symbol}
	}

	artifact := &domainArtifacts.Artifact{
		ID:            uuid.New(),
		ArtifactID:    fmt.Sprintf("strat_%s_%s", result.StrategyID, now.Format(time.RFC3339)),
		SchemaVersion: "1.0.0",
		Strategy: domainArtifacts.StrategyInfo{
			Name:    result.StrategyID,
			Version: strategyVersion,
			Params:  params,
		},
		DataWindow: &domainArtifacts.DataWindow{
			From:    result.StartDate,
			To:      result.EndDate,
			Symbols: symbols,
		},
		Validation: &domainArtifacts.ValidationInfo{
			BacktestRunID:   runUUID,
			DeterminismSeed: int(result.Seed),
			Metrics: map[string]any{
				"total_trades":     result.TotalTrades,
				"winning_trades":   result.WinningTrades,
				"losing_trades":    result.LosingTrades,
				"win_rate":         result.WinRate,
				"total_return_pct": result.TotalReturnPct,
				"max_drawdown":     result.MaxDrawdown,
				"sharpe_ratio":     result.SharpeRatio,
				"profit_factor":    result.ProfitFactor,
				"avg_r":            result.AvgR,
				"duration_ms":      result.DurationMs,
			},
		},
		RiskProfile: riskProfile,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}

	// Compute and embed the SHA-256 hash (immutability guarantee).
	hash, err := artifact.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("failed to compute artifact hash: %w", err)
	}
	artifact.Hash = hash

	// Persist the artifact.
	if err := b.store.CreateArtifact(ctx, artifact); err != nil {
		return nil, fmt.Errorf("failed to save artifact: %w", err)
	}

	// Create the initial DRAFT approval record.
	approval := &domainArtifacts.Approval{
		ID:                uuid.New(),
		ArtifactID:        artifact.ID,
		State:             domainArtifacts.StateDraft,
		ValidationPassed:  true,
		StateChangedBy:    createdBy,
		StateChangedAt:    now,
		StateChangeReason: "artifact created from backtest run " + result.RunID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := b.store.CreateApproval(ctx, approval); err != nil {
		return nil, fmt.Errorf("failed to create draft approval record: %w", err)
	}

	return artifact, nil
}
