package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-orchestrator/internal/reflection"
)

func main() {
	var providersPath string
	var windowDays int
	var maxItems int
	var period string
	var toRaw string
	var dryRun bool
	var printJSON bool

	flag.StringVar(&providersPath, "providers", "config/providers.json", "Path to providers.json")
	flag.IntVar(&windowDays, "window-days", 7, "Reflection window in days")
	flag.IntVar(&maxItems, "max-items", 200, "Max items to recall per bank")
	flag.StringVar(&period, "period", "", "Shortcut for window-days: daily or weekly")
	flag.StringVar(&toRaw, "to", "", "Optional RFC3339 end time (defaults to now)")
	flag.BoolVar(&dryRun, "dry-run", false, "Generate beliefs without retaining them")
	flag.BoolVar(&printJSON, "print", false, "Print generated beliefs as JSON")
	flag.Parse()

	switch strings.ToLower(strings.TrimSpace(period)) {
	case "":
		// no-op
	case "daily":
		windowDays = 1
	case "weekly":
		windowDays = 7
	default:
		log.Fatalf("unsupported period %q (use daily or weekly)", period)
	}

	var to time.Time
	if strings.TrimSpace(toRaw) != "" {
		parsed, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			log.Fatalf("invalid -to timestamp: %v", err)
		}
		to = parsed
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = observability.WithRunInfo(ctx, observability.RunInfo{
		RunID:  observability.NewRunID(),
		TaskID: "reflect",
	})

	client, err := utcp.NewUTCPClientFromFile(providersPath)
	if err != nil {
		log.Fatal(err)
	}

	memorySvc := utcp.NewMemoryService(client)
	job := reflection.NewJob(memoryAdapter{svc: memorySvc}, reflection.WithMaxItems(maxItems))

	result, err := job.Run(ctx, reflection.RunConfig{
		WindowDays: windowDays,
		MaxItems:   maxItems,
		To:         to,
		DryRun:     dryRun,
	})
	if err != nil {
		log.Fatal(err)
	}

	if dryRun || printJSON {
		if err := writeBeliefsJSON(os.Stdout, result.BeliefItems); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("jax-reflect completed: beliefs=%d retained=%d window=%s..%s",
		result.Beliefs,
		result.Retained,
		result.Window.From.Format(time.RFC3339),
		result.Window.To.Format(time.RFC3339),
	)
}

type memoryAdapter struct {
	svc *utcp.MemoryService
}

func (m memoryAdapter) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	out, err := m.svc.Recall(ctx, contracts.MemoryRecallRequest{Bank: bank, Query: query})
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (m memoryAdapter) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	out, err := m.svc.Retain(ctx, contracts.MemoryRetainRequest{Bank: bank, Item: item})
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

func writeBeliefsJSON(out *os.File, beliefs []contracts.MemoryItem) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(beliefs)
}
