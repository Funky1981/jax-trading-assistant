package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-orchestrator/internal/app"
)

func main() {
	var providersPath string
	var bank string
	var symbol string
	var strategy string
	var userContext string
	var tagsCSV string

	flag.StringVar(&providersPath, "providers", "config/providers.json", "Path to providers.json")
	flag.StringVar(&bank, "bank", "trade_decisions", "Memory bank to use")
	flag.StringVar(&symbol, "symbol", "", "Symbol to orchestrate")
	flag.StringVar(&strategy, "strategy", "", "Strategy identifier for tagging")
	flag.StringVar(&userContext, "context", "", "User context for planning")
	flag.StringVar(&tagsCSV, "tags", "", "Comma-separated tags")
	flag.Parse()

	if strings.TrimSpace(symbol) == "" {
		log.Fatal("symbol is required (use -symbol)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runID := observability.NewRunID()
	taskID := "orchestrate"
	if strings.TrimSpace(symbol) != "" {
		taskID = "orchestrate-" + strings.ToLower(symbol)
	}
	ctx = observability.WithRunInfo(ctx, observability.RunInfo{
		RunID:  runID,
		TaskID: taskID,
		Symbol: strings.ToUpper(symbol),
	})

	client, err := utcp.NewUTCPClientFromFile(providersPath)
	if err != nil {
		log.Fatal(err)
	}

	memorySvc := utcp.NewMemoryService(client)
	memory := memoryAdapter{svc: memorySvc}
	agent := stubAgent{}
	tools := stubToolRunner{}

	orch := app.NewOrchestrator(memory, agent, tools)
	result, err := orch.Run(ctx, app.OrchestrationRequest{
		Bank:        bank,
		Symbol:      symbol,
		Strategy:    strategy,
		UserContext: userContext,
		Tags:        parseTags(tagsCSV),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("orchestrator completed: action=%s confidence=%.2f tools=%d", result.Plan.Action, result.Plan.Confidence, len(result.Tools))
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

type stubAgent struct{}

func (stubAgent) Plan(_ context.Context, input app.PlanInput) (app.PlanResult, error) {
	summary := "Decision recorded."
	if strings.TrimSpace(input.Symbol) != "" {
		summary = "Reviewed " + strings.ToUpper(input.Symbol) + " and recorded a decision."
	}
	return app.PlanResult{
		Summary:        summary,
		Steps:          []string{"review inputs", "determine action"},
		Action:         "skipped",
		Confidence:     0.0,
		ReasoningNotes: "stub agent in use",
	}, nil
}

type stubToolRunner struct{}

func (stubToolRunner) Execute(_ context.Context, _ app.PlanResult) ([]app.ToolRun, error) {
	return nil, nil
}

func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	return contracts.NormalizeMemoryTags(parts)
}
