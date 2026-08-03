package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jax-trading-assistant/internal/modules/aishadow"
	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/evidencequality"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	manifestPath := flag.String("manifest", "config/ai-shadow-benchmark-manifest-v1.json", "fixed benchmark manifest path")
	rulesetPath := flag.String("ruleset-file", "config/historical-evidence-quality-v1.json", "qualified outcome ruleset path")
	outputRoot := flag.String("output-root", ".runtime/ai-shadow", "gitignored benchmark output root")
	prepareManifest := flag.Bool("prepare-manifest", false, "write the fixed manifest without calling a model")
	preflight := flag.Bool("preflight", false, "verify configuration, database, manifest, Ollama, and model without inference")
	execute := flag.Bool("execute", false, "explicitly permit the configured bounded model batch")
	flag.Parse()

	config, err := aishadow.LoadConfig(os.LookupEnv)
	if err != nil {
		fail(err)
	}
	safety, err := eventdecisions.ReadEnvironmentSafetyState()
	if err != nil {
		fail(err)
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fail(fmt.Errorf("DATABASE_URL is required"))
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fail(fmt.Errorf("connect AI shadow database: %w", err))
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fail(fmt.Errorf("ping AI shadow database: %w", err))
	}

	rules, err := evidencequality.LoadRuleset(*rulesetPath)
	if err != nil {
		fail(err)
	}
	snapshot, err := evidencequality.NewStore(pool).LoadSnapshot(ctx)
	if err != nil {
		fail(err)
	}
	runtimeSafety := evidencequality.RuntimeSafety{
		RuntimeMode: safety.RuntimeMode, AllowLiveTrading: safety.AllowLiveTrading,
		ExecutionEnabled: safety.ExecutionEnabled, ExecutionWorker: safety.ExecutionWorker,
		BrokerExecution: safety.BrokerExecution, MaximumLeverage: safety.MaximumLeverage,
	}
	qualified, err := evidencequality.Evaluate(snapshot, rules, runtimeSafety)
	if err != nil {
		fail(err)
	}
	eligible, err := aishadow.EligibleEvents(qualified.Events)
	if err != nil {
		fail(err)
	}

	if *prepareManifest {
		count := len(eligible)
		if count > 60 {
			count = 60
		}
		manifest, err := aishadow.NewManifest(eligible, count)
		if err != nil {
			fail(err)
		}
		if err := os.MkdirAll(filepath.Dir(*manifestPath), 0o755); err != nil {
			fail(err)
		}
		if err := aishadow.WriteManifest(*manifestPath, manifest); err != nil {
			fail(err)
		}
		printJSON(map[string]any{"manifest": *manifestPath, "version": manifest.Version, "fingerprint": manifest.Fingerprint, "events": len(manifest.Events)})
		return
	}

	manifest, err := aishadow.LoadManifest(*manifestPath)
	if err != nil {
		fail(err)
	}
	events, err := aishadow.ResolveManifest(manifest, eligible, config.MaxEvents)
	if err != nil {
		fail(err)
	}
	plan := map[string]any{
		"manifest": *manifestPath, "manifest_version": manifest.Version,
		"manifest_fingerprint": manifest.Fingerprint, "manifest_events": len(manifest.Events),
		"selected_events": len(events), "provider": config.Provider, "model": config.Model,
		"base_url": config.BaseURL, "timeout_seconds": int(config.Timeout.Seconds()),
		"temperature": config.Temperature, "seed": config.Seed, "runtime_mode": safety.RuntimeMode,
	}
	if *preflight {
		if err := aishadow.VerifyOllama(config); err != nil {
			fail(err)
		}
		var tableExists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass('public.ai_shadow_benchmark_runs') IS NOT NULL`).Scan(&tableExists); err != nil {
			fail(err)
		}
		if !tableExists {
			fail(fmt.Errorf("AI shadow migration 000054 is not applied"))
		}
		plan["preflight"] = "ready"
		printJSON(plan)
		return
	}
	if !*execute {
		plan["execution"] = "blocked: pass --execute only after operator confirmation"
		printJSON(plan)
		fail(fmt.Errorf("explicit --execute is required"))
	}
	if err := aishadow.VerifyOllama(config); err != nil {
		fail(err)
	}
	knownTickers := map[string]bool{}
	for _, candle := range snapshot.Candles {
		knownTickers[strings.ToUpper(strings.TrimSpace(candle.Symbol))] = true
	}
	runner := aishadow.Runner{
		Config: config, Provider: aishadow.NewOllamaClient(config), Repository: aishadow.NewPGStore(pool),
		OutputRoot: *outputRoot, KnownTickers: knownTickers,
	}
	report, paths, err := runner.Run(manifest, events)
	if err != nil {
		fail(err)
	}
	printJSON(map[string]any{"run_id": report.RunID, "verdict": report.Verdict, "events": report.EventsAttempted, "accepted": report.Accepted, "rejected": report.Rejected, "artifacts": paths})
}

func printJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
