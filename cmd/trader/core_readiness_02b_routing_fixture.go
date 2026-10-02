package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	coreReadiness02BTechnicalRoutingFixtureName          = "CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY"
	coreReadiness02BTechnicalRoutingStrategyTypeID       = "etf_news_sector_momentum_v1"
	coreReadiness02BTechnicalRoutingAdvisoryLock   int64 = 824902702
)

var coreReadiness02BTechnicalRoutingLabels = []string{
	coreReadiness02BTechnicalRoutingFixtureName,
	"NOT_STRATEGY_EVIDENCE",
	"NOT_HYPOTHESIS_SELECTION",
	"DISPOSABLE",
}

type coreReadiness02BTechnicalRoutingConfig struct {
	Symbols              []string `json:"symbols"`
	Purpose              string   `json:"purpose"`
	ClassificationLabels []string `json:"classification_labels"`
	StrategyEvidence     bool     `json:"strategy_evidence"`
	HypothesisSelection  bool     `json:"hypothesis_selection"`
	Disposable           bool     `json:"disposable"`
}

type coreReadiness02BStrategyInstance struct {
	ID                 uuid.UUID
	Name               string
	StrategyTypeID     string
	StrategyID         string
	Enabled            bool
	SessionTimezone    string
	FlattenByCloseTime string
	Config             []byte
	ConfigHash         string
}

func ensureCoreReadiness02BTechnicalRoutingFixture(ctx context.Context, pool *pgxpool.Pool, proofMode bool, databaseURL string) error {
	if !proofMode || !coreReadiness02BProofModeEnabled() {
		return errors.New("02B technical routing fixture requires CORE_READINESS_02B_PROOF_MODE=true")
	}
	urlDatabase, err := guardedCoreReadiness02BDatabaseName(databaseURL)
	if err != nil {
		return err
	}
	var connectedDatabase string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&connectedDatabase); err != nil {
		return fmt.Errorf("verify connected proof database: %w", err)
	}
	if connectedDatabase != urlDatabase {
		return fmt.Errorf("proof database URL names %q but connection selected %q", urlDatabase, connectedDatabase)
	}

	policy, err := loadCandidateEconomicPolicy()
	if err != nil {
		return fmt.Errorf("load reviewed 02B candidate economic policy for routing fixture: %w", err)
	}
	fixtureConfig, err := buildCoreReadiness02BTechnicalRoutingConfig(policy)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(fixtureConfig)
	if err != nil {
		return fmt.Errorf("encode technical routing fixture config: %w", err)
	}
	configHashBytes := sha256.Sum256(configJSON)
	configHash := hex.EncodeToString(configHashBytes[:])

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin technical routing fixture transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, coreReadiness02BTechnicalRoutingAdvisoryLock); err != nil {
		return fmt.Errorf("lock technical routing fixture identity: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, name, strategy_type_id, COALESCE(strategy_id, ''), enabled,
		       session_timezone, flatten_by_close_time, config, config_hash
		FROM strategy_instances
		WHERE name = $1 OR (strategy_type_id = $2 AND COALESCE(strategy_id, '') = $2)
		FOR UPDATE
	`, coreReadiness02BTechnicalRoutingFixtureName, coreReadiness02BTechnicalRoutingStrategyTypeID)
	if err != nil {
		return fmt.Errorf("inspect technical routing fixture identity: %w", err)
	}
	var matches []coreReadiness02BStrategyInstance
	for rows.Next() {
		var row coreReadiness02BStrategyInstance
		if err := rows.Scan(&row.ID, &row.Name, &row.StrategyTypeID, &row.StrategyID, &row.Enabled, &row.SessionTimezone, &row.FlattenByCloseTime, &row.Config, &row.ConfigHash); err != nil {
			rows.Close()
			return fmt.Errorf("scan technical routing fixture identity: %w", err)
		}
		matches = append(matches, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read technical routing fixture identities: %w", err)
	}
	rows.Close()
	if len(matches) > 1 {
		return fmt.Errorf("found %d duplicate semantic technical routing fixtures; refusing startup", len(matches))
	}

	var unexpectedEnabled int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM strategy_instances WHERE enabled=TRUE AND name<>$1`, coreReadiness02BTechnicalRoutingFixtureName).Scan(&unexpectedEnabled); err != nil {
		return fmt.Errorf("check for unexpected enabled strategy instances: %w", err)
	}
	if unexpectedEnabled != 0 {
		return fmt.Errorf("proof database contains %d unexpected enabled strategy instances; refusing startup", unexpectedEnabled)
	}

	if len(matches) == 1 {
		if err := validateCoreReadiness02BTechnicalRoutingFixture(matches[0], fixtureConfig, configHash); err != nil {
			return fmt.Errorf("existing technical routing fixture is incompatible: %w", err)
		}
	} else {
		instanceID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("jax:core-readiness-02b:technical-routing:"+connectedDatabase))
		if _, err := tx.Exec(ctx, `
			INSERT INTO strategy_instances
				(id, name, strategy_type_id, strategy_id, enabled, session_timezone, flatten_by_close_time, config, config_hash)
			VALUES ($1,$2,$3,$3,TRUE,'America/New_York','15:55',$4::jsonb,$5)
		`, instanceID, coreReadiness02BTechnicalRoutingFixtureName, coreReadiness02BTechnicalRoutingStrategyTypeID, string(configJSON), configHash); err != nil {
			return fmt.Errorf("create disposable technical routing fixture: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit technical routing fixture transaction: %w", err)
	}
	return nil
}

func guardedCoreReadiness02BDatabaseName(databaseURL string) (string, error) {
	dsn, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil || dsn.Scheme != "postgres" && dsn.Scheme != "postgresql" {
		return "", errors.New("technical routing fixture requires a guarded PostgreSQL proof database URL")
	}
	name := strings.TrimPrefix(dsn.Path, "/")
	if name != "jax_paper02r_test" && !strings.HasPrefix(name, "jax_paper02r_test_") {
		return "", errors.New("technical routing fixture refuses databases outside the jax_paper02r_test namespace")
	}
	return name, nil
}

func buildCoreReadiness02BTechnicalRoutingConfig(policy candidateEconomicPolicy) (coreReadiness02BTechnicalRoutingConfig, error) {
	if len(policy.Instruments) == 0 {
		return coreReadiness02BTechnicalRoutingConfig{}, errors.New("technical routing fixture requires a non-empty reviewed instrument policy")
	}
	symbols := make([]string, 0, len(policy.Instruments))
	for symbol := range policy.Instruments {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			return coreReadiness02BTechnicalRoutingConfig{}, errors.New("technical routing fixture policy contains an empty symbol")
		}
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	return coreReadiness02BTechnicalRoutingConfig{
		Symbols: symbols, Purpose: coreReadiness02BTechnicalRoutingFixtureName,
		ClassificationLabels: append([]string(nil), coreReadiness02BTechnicalRoutingLabels...),
		StrategyEvidence:     false, HypothesisSelection: false, Disposable: true,
	}, nil
}

func validateCoreReadiness02BTechnicalRoutingFixture(row coreReadiness02BStrategyInstance, expected coreReadiness02BTechnicalRoutingConfig, expectedHash string) error {
	if row.Name != coreReadiness02BTechnicalRoutingFixtureName || row.StrategyTypeID != coreReadiness02BTechnicalRoutingStrategyTypeID || row.StrategyID != coreReadiness02BTechnicalRoutingStrategyTypeID {
		return errors.New("fixture identity does not match the guarded technical routing identity")
	}
	if !row.Enabled || row.SessionTimezone != "America/New_York" || row.FlattenByCloseTime != "15:55" {
		return errors.New("fixture must be enabled with the reviewed session timezone and close time")
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(row.Config, &config); err != nil {
		return fmt.Errorf("decode fixture config: %w", err)
	}
	if len(config) != 6 {
		return errors.New("fixture config contains missing or unexpected fields")
	}
	for _, key := range []string{"symbols", "purpose", "classification_labels", "strategy_evidence", "hypothesis_selection", "disposable"} {
		if _, exists := config[key]; !exists {
			return fmt.Errorf("fixture config is missing required field %q", key)
		}
	}
	var actual coreReadiness02BTechnicalRoutingConfig
	if err := json.Unmarshal(row.Config, &actual); err != nil {
		return fmt.Errorf("decode fixture config fields: %w", err)
	}
	if !equalStrings(actual.Symbols, expected.Symbols) {
		return errors.New("fixture symbols do not exactly match the reviewed candidate economic policy")
	}
	if actual.Purpose != coreReadiness02BTechnicalRoutingFixtureName || !equalStrings(actual.ClassificationLabels, coreReadiness02BTechnicalRoutingLabels) {
		return errors.New("fixture is missing required technical-proof classification labels")
	}
	if actual.StrategyEvidence || actual.HypothesisSelection || !actual.Disposable {
		return errors.New("fixture evidence, hypothesis-selection, or disposable flags are incompatible")
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("encode validated fixture config: %w", err)
	}
	actualHash := sha256.Sum256(actualJSON)
	if row.ConfigHash != expectedHash || hex.EncodeToString(actualHash[:]) != expectedHash {
		return errors.New("fixture config hash does not match its reviewed semantic config")
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
