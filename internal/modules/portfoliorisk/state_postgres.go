package portfoliorisk

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// PostgresSnapshotStore uses the existing database/sql boundary and an
// append-only table. It does not connect to a broker or mutate account state.
type PostgresSnapshotStore struct{ db *sql.DB }

func NewPostgresSnapshotStore(db *sql.DB) (*PostgresSnapshotStore, error) {
	if db == nil {
		return nil, fmt.Errorf("portfolio snapshot store: db is nil")
	}
	return &PostgresSnapshotStore{db: db}, nil
}

func (s *PostgresSnapshotStore) Save(ctx context.Context, snapshot PortfolioSnapshot) error {
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Errorf("portfolio snapshot: encode: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO portfolio_snapshots
 (snapshot_id, contract_version, identity_algorithm, account_id, as_of, captured_at,
  provider, currency, synthetic, payload, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now())
ON CONFLICT (snapshot_id) DO NOTHING`, canonical.SnapshotID, canonical.ContractVersion,
		canonical.IdentityAlgorithm, canonical.AccountID, canonical.AsOf, canonical.CapturedAt,
		canonical.Provider, canonical.Currency, canonical.Synthetic, payload)
	if err != nil {
		return fmt.Errorf("portfolio snapshot: save: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		var existing []byte
		if err := s.db.QueryRowContext(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, canonical.SnapshotID).Scan(&existing); err != nil {
			return fmt.Errorf("portfolio snapshot: conflict lookup: %w", err)
		}
		var stored PortfolioSnapshot
		if err := json.Unmarshal(existing, &stored); err != nil {
			return fmt.Errorf("portfolio snapshot: decode conflict: %w", err)
		}
		if !snapshotsEqual(stored, canonical) {
			return ErrSnapshotConflict
		}
	}
	return nil
}

func (s *PostgresSnapshotStore) Get(ctx context.Context, id string) (PortfolioSnapshot, error) {
	if strings.TrimSpace(id) == "" {
		return PortfolioSnapshot{}, fmt.Errorf("portfolio snapshot: id is required")
	}
	var payload []byte
	if err := s.db.QueryRowContext(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, id).Scan(&payload); err != nil {
		return PortfolioSnapshot{}, err
	}
	var snapshot PortfolioSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return PortfolioSnapshot{}, fmt.Errorf("portfolio snapshot: decode: %w", err)
	}
	return snapshot, nil
}
