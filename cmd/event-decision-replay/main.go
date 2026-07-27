package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/instruments"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		dryRun      = flag.Bool("dry-run", false, "evaluate selected events without database writes")
		event       = flag.String("event", "", "one inbox UUID or source event ID (optional)")
		limit       = flag.Int("limit", 25, "bounded maximum number of inbox rows (1-250)")
		ruleset     = flag.String("ruleset", "", "required explicit ruleset version")
		rulesetFile = flag.String("ruleset-file", "config/genuine-event-decision-v1.json", "ruleset configuration path")
		databaseURL = flag.String("database-url", "", "PostgreSQL URL; defaults to DATABASE_URL")
	)
	flag.Parse()
	ctx := context.Background()

	rules, err := eventdecisions.LoadRuleset(*rulesetFile)
	if err != nil {
		fail(err)
	}
	if strings.TrimSpace(*ruleset) == "" || *ruleset != rules.Version {
		fail(fmt.Errorf("explicit --ruleset %s is required", rules.Version))
	}
	safety, err := eventdecisions.ReadEnvironmentSafetyState()
	if err != nil {
		fail(err)
	}
	if *limit <= 0 || *limit > 250 {
		fail(fmt.Errorf("--limit must be between 1 and 250"))
	}
	if strings.TrimSpace(*databaseURL) == "" {
		*databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if *databaseURL == "" {
		fail(fmt.Errorf("DATABASE_URL or --database-url is required"))
	}
	pool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		fail(fmt.Errorf("connect replay database: %w", err))
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fail(fmt.Errorf("ping replay database: %w", err))
	}
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		fail(fmt.Errorf("load instrument catalog: %w", err))
	}
	store := eventdecisions.NewStore(pool)
	events, err := store.LoadSelectedEvents(ctx, *event, *limit)
	if err != nil {
		fail(err)
	}
	replayer := eventdecisions.Replayer{
		Store: store,
		Evaluator: eventdecisions.Evaluator{
			Ruleset: rules,
			Catalog: catalog,
		},
		Now: func() time.Time { return time.Now().UTC() },
	}
	summary, runErr := replayer.Run(ctx, events, *dryRun)
	output := struct {
		Safety eventdecisions.SafetyState `json:"safety"`
		Replay eventdecisions.Summary     `json:"replay"`
	}{Safety: safety, Replay: summary}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fail(err)
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
