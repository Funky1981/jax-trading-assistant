package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func Test02BTechnicalRoutingFixtureExistsOnlyInGuardedDisposableDatabase(t *testing.T) {
	pool := testFrontendAPIPool(t) // testsupport rejects every non-jax_paper02r_test database.
	database := pool.Config().ConnConfig.Database
	if database != "jax_paper02r_test" && !strings.HasPrefix(database, "jax_paper02r_test_") {
		t.Fatalf("routing fixture refuses non-disposable database identity")
	}
	policyPath, err := filepath.Abs("../../config/core-readiness-02b-candidate-economic-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, policyPath)
	policy, err := loadCandidateEconomicPolicy()
	if err != nil {
		t.Fatal(err)
	}
	symbols := make([]string, 0, len(policy.Instruments))
	for symbol := range policy.Instruments {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	config, err := json.Marshal(map[string]any{
		"symbols": symbols,
		"purpose": "CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY",
		"classification_labels": []string{
			"CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY",
			"NOT_STRATEGY_EVIDENCE",
			"NOT_HYPOTHESIS_SELECTION",
			"DISPOSABLE",
		},
		"strategy_evidence":    false,
		"hypothesis_selection": false,
		"disposable":           true,
	})
	if err != nil {
		t.Fatal(err)
	}
	instanceID := uuid.New()
	name := "CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY"
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,session_timezone,flatten_by_close_time,config,config_hash)
		VALUES($1,$2,'etf_news_sector_momentum_v1','etf_news_sector_momentum_v1',TRUE,'America/New_York','15:55',$3::jsonb,$4)
	`, instanceID, name, string(config), fmt.Sprintf("02b-technical-routing-%s", instanceID.String())); err != nil {
		t.Fatalf("create disposable technical routing fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM strategy_instances WHERE id=$1 AND name=$2`, instanceID, name)
	})
	var actualName string
	var actualConfig []byte
	if err := pool.QueryRow(context.Background(), `SELECT name,config FROM strategy_instances WHERE id=$1`, instanceID).Scan(&actualName, &actualConfig); err != nil {
		t.Fatal(err)
	}
	var labels map[string]any
	if err := json.Unmarshal(actualConfig, &labels); err != nil {
		t.Fatal(err)
	}
	if actualName != name || labels["purpose"] != name || labels["strategy_evidence"] != false || labels["hypothesis_selection"] != false || labels["disposable"] != true {
		t.Fatalf("routing fixture is not unmistakably marked: name=%q config=%s", actualName, actualConfig)
	}
	classificationLabels, ok := labels["classification_labels"].([]any)
	wantLabels := []string{"CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY", "NOT_STRATEGY_EVIDENCE", "NOT_HYPOTHESIS_SELECTION", "DISPOSABLE"}
	if !ok || len(classificationLabels) != len(wantLabels) {
		t.Fatalf("routing fixture classification labels=%v", labels["classification_labels"])
	}
	for index, want := range wantLabels {
		if classificationLabels[index] != want {
			t.Fatalf("routing fixture classification labels=%v, want %v", classificationLabels, wantLabels)
		}
	}
	gotID, gotStrategy, err := newWorldMonitorOpportunityPromoter(pool).findStrategyInstance(context.Background(), "QQQ")
	if err != nil || gotID != instanceID || gotStrategy != "etf_news_sector_momentum_v1" {
		t.Fatalf("promoter routing did not select isolated technical fixture: id=%s strategy=%q err=%v", gotID, gotStrategy, err)
	}
}
