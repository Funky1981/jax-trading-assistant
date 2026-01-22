package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-orchestrator/internal/app"
	"jax-trading-assistant/services/jax-orchestrator/internal/config"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if strings.TrimSpace(cfg.Symbol) == "" {
		log.Fatal("symbol is required (use -symbol)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runID := observability.NewRunID()
	taskID := "orchestrate"
	if strings.TrimSpace(cfg.Symbol) != "" {
		taskID = "orchestrate-" + strings.ToLower(cfg.Symbol)
	}
	ctx = observability.WithRunInfo(ctx, observability.RunInfo{
		RunID:  runID,
		TaskID: taskID,
		Symbol: strings.ToUpper(cfg.Symbol),
	})

	client, err := utcp.NewUTCPClientFromFile(cfg.ProvidersPath)
	if err != nil {
		log.Fatal(err)
	}

	memorySvc := utcp.NewMemoryService(client)
	memory := memoryAdapter{svc: memorySvc}
	agent := stubAgent{}
	tools := stubToolRunner{}

	orch := app.NewOrchestrator(memory, agent, tools)
	result, err := orch.Run(ctx, app.OrchestrationRequest{
		Bank:        cfg.Bank,
		Symbol:      cfg.Symbol,
		Strategy:    cfg.Strategy,
		UserContext: cfg.UserContext,
		Tags:        parseTags(cfg.TagsCSV),
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
