package portfoliorisk

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"jax-trading-assistant/internal/testsupport"
)

func TestPostgresLegacySnapshotAndRiskDecisionKeepV1Identity(t *testing.T) {
	dsn := testsupport.PostgresDSN(t, "TEST_DATABASE_URL")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("open disposable PostgreSQL database")
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("connect to disposable PostgreSQL database")
	}

	snapshot := fixtureSnapshot()
	snapshot.AccountID = "legacy-v1-" + uuid.NewString()
	legacy := legacySnapshotFromCurrent(snapshot)
	legacyID, _, err := legacySnapshotIdentity(legacy)
	if err != nil {
		t.Fatal(err)
	}
	legacy.SnapshotID = legacyID
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"known_empty"`) {
		t.Fatal("test fixture is not a pre-KnownEmpty payload")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO portfolio_snapshots(snapshot_id,contract_version,identity_algorithm,account_id,as_of,captured_at,provider,currency,synthetic,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, legacyID, legacy.ContractVersion, legacy.IdentityAlgorithm, legacy.AccountID, legacy.AsOf, legacy.CapturedAt, legacy.Provider, legacy.Currency, legacy.Synthetic, payload); err != nil {
		t.Fatal("insert append-only legacy snapshot fixture")
	}

	snapshotStore, err := NewPostgresSnapshotStore(db)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := snapshotStore.Get(ctx, legacyID)
	if err != nil {
		t.Fatalf("PostgresSnapshotStore.Get legacy snapshot: %v", err)
	}
	canonical, err := BuildSnapshot(loaded)
	if err != nil || canonical.SnapshotID != legacyID {
		t.Fatalf("legacy snapshot verification changed identity: canonical=%+v err=%v want=%s", canonical, err, legacyID)
	}

	analytics, err := CalculateExposure(canonical, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	recommendationInput := recommendation(5000)
	recommendationInput.RecommendationID = "legacy-risk-" + uuid.NewString()
	decision := EvaluateRecommendation(recommendationInput, canonical, analytics, policy, stateNow, 10*time.Minute)
	if decision.Outcome != DecisionAccept || decision.PortfolioSnapshotID != legacyID || decision.Validate() != nil {
		t.Fatalf("historical-bound risk decision invalid: decision=%+v validation=%v", decision, decision.Validate())
	}
	riskStore, err := NewPostgresRiskDecisionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := riskStore.Save(ctx, decision); err != nil {
		t.Fatalf("persist legacy-bound risk decision: %v", err)
	}
	reloadedDecision, err := riskStore.Get(ctx, decision.DecisionID)
	if err != nil {
		t.Fatalf("reload legacy-bound risk decision: %v", err)
	}
	if reloadedDecision.DecisionID != decision.DecisionID || reloadedDecision.PortfolioSnapshotID != legacyID || reloadedDecision.Validate() != nil {
		t.Fatalf("risk decision binding changed after reload: %+v", reloadedDecision)
	}

	var storedPayload []byte
	if err := db.QueryRowContext(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, legacyID).Scan(&storedPayload); err != nil {
		t.Fatal("verify legacy row was not rewritten")
	}
	if strings.Contains(string(storedPayload), `"known_empty"`) {
		t.Fatal("legacy durable payload was rewritten to add known_empty")
	}
}
