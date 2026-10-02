package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"jax-trading-assistant/internal/testsupport"
	"jax-trading-assistant/libs/runtimepolicy"
	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/libs/strategytypes"
)

func TestEnsureCoreReadiness02BTechnicalRoutingFixtureFailsClosedBeforeDatabaseUse(t *testing.T) {
	t.Setenv(coreReadiness02BProofModeEnv, "true")
	for _, tt := range []struct {
		name        string
		proofMode   bool
		databaseURL string
	}{
		{name: "proof mode false", proofMode: false, databaseURL: "postgres://jax@localhost/jax_paper02r_test_proof"},
		{name: "normal database", proofMode: true, databaseURL: "postgres://jax@localhost/jax"},
		{name: "non-postgres database", proofMode: true, databaseURL: "https://host/jax_paper02r_test_proof"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := ensureCoreReadiness02BTechnicalRoutingFixture(context.Background(), nil, tt.proofMode, tt.databaseURL); err == nil {
				t.Fatal("unsafe fixture request unexpectedly accepted")
			}
		})
	}
}

func TestValidateCoreReadiness02BTechnicalRoutingFixtureRejectsIncompatibleSemantics(t *testing.T) {
	policy := loadRoutingFixtureTestPolicy(t)
	want, err := buildCoreReadiness02BTechnicalRoutingConfig(policy)
	if err != nil {
		t.Fatal(err)
	}
	newRow := func() coreReadiness02BStrategyInstance {
		config, marshalErr := json.Marshal(want)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		hash := sha256.Sum256(config)
		return coreReadiness02BStrategyInstance{
			ID: uuid.New(), Name: coreReadiness02BTechnicalRoutingFixtureName,
			StrategyTypeID: coreReadiness02BTechnicalRoutingStrategyTypeID,
			StrategyID:     coreReadiness02BTechnicalRoutingStrategyTypeID, Enabled: true,
			SessionTimezone: "America/New_York", FlattenByCloseTime: "15:55", Config: config, ConfigHash: hex.EncodeToString(hash[:]),
		}
	}
	tests := []struct {
		name   string
		mutate func(*coreReadiness02BStrategyInstance)
	}{
		{name: "wrong strategy type", mutate: func(row *coreReadiness02BStrategyInstance) { row.StrategyTypeID = "other" }},
		{name: "wrong strategy id", mutate: func(row *coreReadiness02BStrategyInstance) { row.StrategyID = "other" }},
		{name: "disabled fixture", mutate: func(row *coreReadiness02BStrategyInstance) { row.Enabled = false }},
		{name: "missing labels", mutate: func(row *coreReadiness02BStrategyInstance) {
			var cfg coreReadiness02BTechnicalRoutingConfig
			_ = json.Unmarshal(row.Config, &cfg)
			cfg.ClassificationLabels = cfg.ClassificationLabels[:2]
			row.Config, _ = json.Marshal(cfg)
			hash := sha256.Sum256(row.Config)
			row.ConfigHash = hex.EncodeToString(hash[:])
		}},
		{name: "strategy evidence true", mutate: func(row *coreReadiness02BStrategyInstance) {
			updateRoutingFixtureConfig(t, row, func(c *coreReadiness02BTechnicalRoutingConfig) { c.StrategyEvidence = true })
		}},
		{name: "hypothesis selection true", mutate: func(row *coreReadiness02BStrategyInstance) {
			updateRoutingFixtureConfig(t, row, func(c *coreReadiness02BTechnicalRoutingConfig) { c.HypothesisSelection = true })
		}},
		{name: "disposable false", mutate: func(row *coreReadiness02BStrategyInstance) {
			updateRoutingFixtureConfig(t, row, func(c *coreReadiness02BTechnicalRoutingConfig) { c.Disposable = false })
		}},
		{name: "broader symbol universe", mutate: func(row *coreReadiness02BStrategyInstance) {
			updateRoutingFixtureConfig(t, row, func(c *coreReadiness02BTechnicalRoutingConfig) { c.Symbols = append(c.Symbols, "AAPL") })
		}},
		{name: "unexpected config field", mutate: func(row *coreReadiness02BStrategyInstance) {
			var cfg map[string]any
			_ = json.Unmarshal(row.Config, &cfg)
			cfg["extra"] = true
			row.Config, _ = json.Marshal(cfg)
			hash := sha256.Sum256(row.Config)
			row.ConfigHash = hex.EncodeToString(hash[:])
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := newRow()
			tt.mutate(&row)
			if err := validateCoreReadiness02BTechnicalRoutingFixture(row, want, row.ConfigHash); err == nil {
				t.Fatal("incompatible fixture unexpectedly accepted")
			}
		})
	}
}

func Test02BTechnicalRoutingFixtureCreationRestartAndPromoterRouting(t *testing.T) {
	pool := coreReadiness02BRoutingTestPool(t)
	ctx := context.Background()
	dbName := pool.Config().ConnConfig.Database
	if _, err := guardedCoreReadiness02BDatabaseName("postgres://test/" + dbName); err != nil {
		t.Fatalf("test database is not guarded: %v", err)
	}
	databaseURL := (&url.URL{Scheme: "postgres", Host: "test", Path: "/" + dbName}).String()
	setValid02BProofEnvironment(t)
	setRoutingFixtureTestPolicy(t)
	t.Setenv("POLYGON_API_KEY", "")
	registry := strategytypes.DefaultRegistry()

	// The generic gate passes before the fixture exists; after installation its
	// normal event-provider contract still rejects the event-dependent fixture.
	if err := validateEventProviderReadiness(ctx, pool, registry, runtimepolicy.ModePaper, true); err != nil {
		t.Fatalf("generic readiness should pass before fixture insertion: %v", err)
	}
	if err := ensureCoreReadiness02BTechnicalRoutingFixture(ctx, pool, true, databaseURL); err != nil {
		t.Fatalf("create guarded routing fixture: %v", err)
	}
	var fixtureID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM strategy_instances WHERE name=$1`, coreReadiness02BTechnicalRoutingFixtureName).Scan(&fixtureID); err != nil {
		t.Fatalf("load fixture ID: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM strategy_instances WHERE id=$1`, fixtureID)
	})
	if err := ensureCoreReadiness02BTechnicalRoutingFixture(ctx, pool, true, databaseURL); err != nil {
		t.Fatalf("restart should reuse the validated fixture: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM strategy_instances WHERE name=$1`, coreReadiness02BTechnicalRoutingFixtureName).Scan(&count); err != nil || count != 1 {
		t.Fatalf("fixture count after restart=%d err=%v, want exactly one", count, err)
	}
	approvedRegistry := strategies.NewRegistry()
	if err := requireApprovedStrategiesForStartup(runtimepolicy.ModePaper, len(approvedRegistry.List()), true); err != nil {
		t.Fatalf("proof-mode empty approved registry rejected: %v", err)
	}
	if err := validateEventProviderReadiness(ctx, pool, registry, runtimepolicy.ModePaper); err == nil {
		t.Fatal("generic readiness changed: enabled event-dependent fixture unexpectedly bypassed missing Polygon")
	}
	if err := validateEventProviderReadiness(ctx, pool, registry, runtimepolicy.ModePaper, true); err != nil {
		t.Fatalf("proof-mode startup should exclude only its validated technical fixture on restart: %v", err)
	}

	policy, err := loadCandidateEconomicPolicy()
	if err != nil {
		t.Fatal(err)
	}
	promoter := newWorldMonitorOpportunityPromoter(pool)
	for symbol := range policy.Instruments {
		id, strategyID, err := promoter.findStrategyInstance(ctx, symbol)
		if err != nil || id != fixtureID || strategyID != coreReadiness02BTechnicalRoutingStrategyTypeID {
			t.Errorf("policy ETF %s did not resolve to technical fixture: id=%s strategy=%q err=%v", symbol, id, strategyID, err)
		}
	}
	if id, strategyID, err := promoter.findStrategyInstance(ctx, "AAPL"); err == nil {
		t.Fatalf("non-policy symbol acquired routing identity id=%s strategy=%q", id, strategyID)
	}
}

func Test02BTechnicalRoutingFixtureRejectsUnexpectedEnabledAndDuplicateSemanticRows(t *testing.T) {
	for _, scenario := range []string{"unexpected enabled row", "duplicate semantic fixture"} {
		t.Run(scenario, func(t *testing.T) {
			pool := coreReadiness02BRoutingTestPool(t)
			ctx := context.Background()
			setValid02BProofEnvironment(t)
			setRoutingFixtureTestPolicy(t)
			name := fmt.Sprintf("02b-routing-reject-%s-%s", scenario, uuid.NewString())
			strategyType, strategyID := "unrelated_strategy", "unrelated_strategy"
			config := `{"symbols":["AAPL"]}`
			if scenario == "duplicate semantic fixture" {
				name = "02b-routing-duplicate-" + uuid.NewString()
				strategyType, strategyID = coreReadiness02BTechnicalRoutingStrategyTypeID, coreReadiness02BTechnicalRoutingStrategyTypeID
				policy, err := loadCandidateEconomicPolicy()
				if err != nil {
					t.Fatal(err)
				}
				cfg, err := buildCoreReadiness02BTechnicalRoutingConfig(policy)
				if err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(cfg)
				config = string(encoded)
			}
			id := uuid.New()
			if _, err := pool.Exec(ctx, `INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,config) VALUES($1,$2,$3,$4,TRUE,$5::jsonb)`, id, name, strategyType, strategyID, config); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM strategy_instances WHERE id=$1`, id) })
			databaseURL := (&url.URL{Scheme: "postgres", Host: "test", Path: "/" + pool.Config().ConnConfig.Database}).String()
			if err := ensureCoreReadiness02BTechnicalRoutingFixture(ctx, pool, true, databaseURL); err == nil {
				t.Fatal("unsafe enabled strategy state unexpectedly accepted")
			}
		})
	}
}

func Test02BNormalPaperReadinessStillRequiresPolygonForEnabledEventStrategy(t *testing.T) {
	pool := coreReadiness02BRoutingTestPool(t)
	t.Setenv("POLYGON_API_KEY", "")
	ctx := context.Background()
	id := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,config) VALUES($1,$2,'etf_news_sector_momentum_v1','etf_news_sector_momentum_v1',TRUE,'{"symbols":["QQQ"]}'::jsonb)`, id, "normal-paper-readiness-"+id.String()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM strategy_instances WHERE id=$1`, id) })
	if err := validateEventProviderReadiness(ctx, pool, strategytypes.DefaultRegistry(), runtimepolicy.ModePaper); err == nil {
		t.Fatal("ordinary PAPER readiness bypassed its required Polygon provider")
	}
}

func loadRoutingFixtureTestPolicy(t *testing.T) candidateEconomicPolicy {
	t.Helper()
	setRoutingFixtureTestPolicy(t)
	policy, err := loadCandidateEconomicPolicy()
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func setRoutingFixtureTestPolicy(t *testing.T) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "config", "core-readiness-02b-candidate-economic-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, path)
}

func coreReadiness02BRoutingTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := testsupport.PostgresDSN(t, "CORE02B_ROUTING_DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal("guarded CORE-READINESS-02B routing test database unavailable")
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatal("guarded CORE-READINESS-02B routing test database unavailable")
	}
	t.Cleanup(pool.Close)
	return pool
}

func updateRoutingFixtureConfig(t *testing.T, row *coreReadiness02BStrategyInstance, mutate func(*coreReadiness02BTechnicalRoutingConfig)) {
	t.Helper()
	var cfg coreReadiness02BTechnicalRoutingConfig
	if err := json.Unmarshal(row.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	mutate(&cfg)
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	row.Config = encoded
	hash := sha256.Sum256(encoded)
	row.ConfigHash = hex.EncodeToString(hash[:])
}
