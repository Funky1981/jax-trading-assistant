package main

import (
	"log"
	"os"
	"time"

	"jax-trading-assistant/libs/ingest"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-ingest/internal/config"
	internalIngest "jax-trading-assistant/services/jax-ingest/internal/ingest"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	payload, err := ingest.ReadPayload(cfg.InputPath)
	if err != nil {
		log.Fatal(err)
	}
	ingest.NormalizeTimestamps(&payload)

	ctx, cancel := ingest.CreateContext(30 * time.Second)
	defer cancel()

	client, err := utcp.NewUTCPClientFromFile(cfg.ProvidersPath)
	if err != nil {
		log.Fatal(err)
	}

	store := ingest.NewMemoryAdapter(client)

	result, err := internalIngest.RetainDexterObservations(ctx, store, payload.Observations, internalIngest.RetentionConfig{
		SignificanceThreshold: cfg.Threshold,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("jax-ingest completed: retained=%d skipped=%d", result.Retained, result.Skipped)
}
