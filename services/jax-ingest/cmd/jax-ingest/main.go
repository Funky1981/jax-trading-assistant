package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-ingest/internal/ingest"
)

func main() {
	var providersPath string
	var inputPath string
	var threshold float64

	flag.StringVar(&providersPath, "providers", "config/providers.json", "Path to providers.json")
	flag.StringVar(&inputPath, "input", "", "Path to Dexter JSON payload (defaults to stdin)")
	flag.Float64Var(&threshold, "threshold", 0.7, "Significance threshold for retention")
	flag.Parse()

	payload, err := readPayload(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	if payload.GeneratedAt.IsZero() {
		payload.GeneratedAt = time.Now().UTC()
	}
	for i := range payload.Observations {
		if payload.Observations[i].TS.IsZero() {
			payload.Observations[i].TS = payload.GeneratedAt
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = observability.WithRunInfo(ctx, observability.RunInfo{
		RunID:  observability.NewRunID(),
		TaskID: "ingest",
	})

	client, err := utcp.NewUTCPClientFromFile(providersPath)
	if err != nil {
		log.Fatal(err)
	}

	memorySvc := utcp.NewMemoryService(client)
	store := memoryAdapter{svc: memorySvc}

	result, err := ingest.RetainDexterObservations(ctx, store, payload.Observations, ingest.RetentionConfig{
		SignificanceThreshold: threshold,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("jax-ingest completed: retained=%d skipped=%d", result.Retained, result.Skipped)
}

type memoryAdapter struct {
	svc *utcp.MemoryService
}

func (m memoryAdapter) Ping(context.Context) error {
	return nil
}

func (m memoryAdapter) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	out, err := m.svc.Retain(ctx, contracts.MemoryRetainRequest{Bank: bank, Item: item})
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

func (m memoryAdapter) Recall(context.Context, string, contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	return nil, errors.New("memory recall not supported by jax-ingest")
}

func (m memoryAdapter) Reflect(context.Context, string, contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	return nil, errors.New("memory reflect not supported by jax-ingest")
}

func readPayload(path string) (ingest.DexterPayload, error) {
	var payload ingest.DexterPayload
	reader, err := openInput(path)
	if err != nil {
		return payload, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return payload, err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return payload, errors.New("dexter payload is empty")
	}

	if err := json.Unmarshal(raw, &payload); err == nil && len(payload.Observations) > 0 {
		return payload, nil
	}

	var observations []ingest.DexterObservation
	if err := json.Unmarshal(raw, &observations); err != nil {
		return payload, fmt.Errorf("parse dexter payload: %w", err)
	}
	payload.Observations = observations
	return payload, nil
}

func openInput(path string) (io.ReadCloser, error) {
	if path == "" {
		return io.NopCloser(os.Stdin), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}
