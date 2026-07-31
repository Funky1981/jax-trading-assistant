package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/evidencequality"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		rulesetVersion = flag.String("ruleset", "", "required explicit evaluation ruleset version")
		rulesetFile    = flag.String("ruleset-file", "config/historical-evidence-quality-v1.json", "evaluation ruleset path")
		outputDir      = flag.String("output-dir", ".runtime/evidence-quality", "isolated report output directory")
		databaseURL    = flag.String("database-url", "", "PostgreSQL URL; defaults to DATABASE_URL")
	)
	flag.Parse()
	rules, err := evidencequality.LoadRuleset(*rulesetFile)
	if err != nil {
		fail(err)
	}
	if strings.TrimSpace(*rulesetVersion) == "" || *rulesetVersion != rules.Version {
		fail(fmt.Errorf("explicit --ruleset %s is required", rules.Version))
	}
	safety, err := eventdecisions.ReadEnvironmentSafetyState()
	if err != nil {
		fail(err)
	}
	if strings.TrimSpace(*databaseURL) == "" {
		*databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if *databaseURL == "" {
		fail(fmt.Errorf("DATABASE_URL or --database-url is required"))
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		fail(fmt.Errorf("connect evaluation database: %w", err))
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fail(fmt.Errorf("ping evaluation database: %w", err))
	}
	snapshot, err := evidencequality.NewStore(pool).LoadSnapshot(ctx)
	if err != nil {
		fail(err)
	}
	runtime := evidencequality.RuntimeSafety{RuntimeMode: safety.RuntimeMode, AllowLiveTrading: safety.AllowLiveTrading, ExecutionEnabled: safety.ExecutionEnabled, ExecutionWorker: safety.ExecutionWorker, BrokerExecution: safety.BrokerExecution, MaximumLeverage: safety.MaximumLeverage}
	report, err := evidencequality.Evaluate(snapshot, rules, runtime)
	if err != nil {
		fail(err)
	}
	paths, err := evidencequality.WriteArtifacts(*outputDir, report)
	if err != nil {
		fail(err)
	}
	output := struct {
		RulesetVersion        string                            `json:"rulesetVersion"`
		InputFingerprint      string                            `json:"inputFingerprint"`
		Population            evidencequality.PopulationSummary `json:"population"`
		Conclusion            string                            `json:"conclusion"`
		ProductRecommendation string                            `json:"productRecommendation"`
		Verdict               string                            `json:"verdict"`
		Artifacts             evidencequality.ArtifactPaths     `json:"artifacts"`
	}{report.RulesetVersion, report.InputFingerprint, report.Population, report.Conclusion, report.ProductRecommendation, report.Verdict, paths}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
