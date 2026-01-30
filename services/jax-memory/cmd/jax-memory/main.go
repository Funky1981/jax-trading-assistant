package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/testing"
	httpapi "jax-trading-assistant/services/jax-memory/internal/http"
	"jax-trading-assistant/services/jax-memory/internal/infra/hindsight"
)

func main() {
	store := buildStore()

	srv := httpapi.NewServer(store)
	srv.RegisterHealth()
	srv.RegisterTools()
	srv.RegisterMemoryAPI()

	port := getenvInt("PORT", 8090)
	addr := ":" + strconv.Itoa(port)
	log.Printf("jax-memory listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

func buildStore() contracts.MemoryStore {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if baseURL := os.Getenv("HINDSIGHT_URL"); baseURL != "" {
		client, err := hindsight.New(baseURL)
		if err != nil {
			log.Printf("invalid HINDSIGHT_URL, falling back to in-memory store: %v", err)
			return testing.NewInMemoryMemoryStore()
		}
		if err := client.Ping(ctx); err != nil {
			log.Printf("hindsight ping failed, falling back to in-memory store: %v", err)
			return testing.NewInMemoryMemoryStore()
		}
		return client
	}
	return testing.NewInMemoryMemoryStore()
}

func getenvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
