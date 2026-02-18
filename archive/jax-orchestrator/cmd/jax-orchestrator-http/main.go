package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/services/jax-orchestrator/internal/app"
	httpapi "jax-trading-assistant/services/jax-orchestrator/internal/infra/http"
)

func main() {
	port := flag.String("port", "8091", "HTTP server port")
	dbDSN := flag.String("db", "", "Database DSN (if not set, uses DATABASE_URL env var)")
	flag.Parse()

	// Get database DSN
	databaseURL := *dbDSN
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		databaseURL = "postgresql://jax:jax@localhost:5432/jax?sslmode=disable"
	}

	// Connect to database
	dbConfig := database.DefaultConfig()
	dbConfig.DSN = databaseURL

	db, err := database.Connect(context.Background(), dbConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	// Create strategy registry (for future use)
	strategyRegistry := strategies.NewRegistry()

	// Get service URLs from environment
	memoryServiceURL := os.Getenv("MEMORY_SERVICE_URL")
	if memoryServiceURL == "" {
		memoryServiceURL = "http://jax-memory:8090"
	}

	agent0ServiceURL := os.Getenv("AGENT0_SERVICE_URL")
	if agent0ServiceURL == "" {
		agent0ServiceURL = "http://agent0-service:8093"
	}

	dexterServiceURL := os.Getenv("DEXTER_SERVICE_URL")
	if dexterServiceURL == "" {
		dexterServiceURL = "http://localhost:8094"
	}

	// Create real client dependencies for orchestrator
	memory, err := NewMemoryClient(memoryServiceURL)
	if err != nil {
		log.Fatalf("failed to create memory client: %v", err)
	}
	log.Printf("memory client connected to %s", memoryServiceURL)

	agent, err := NewAgent0Client(agent0ServiceURL)
	if err != nil {
		log.Fatalf("failed to create Agent0 client: %v", err)
	}
	log.Printf("Agent0 client connected to %s", agent0ServiceURL)

	dexter, err := NewDexterClient(dexterServiceURL)
	if err != nil {
		log.Printf("warning: failed to create Dexter client: %v (continuing without Dexter)", err)
		dexter = nil // Dexter is optional
	} else {
		log.Printf("Dexter client connected to %s", dexterServiceURL)
	}

	tools := NewToolRunner(dexter)

	// Create orchestrator
	orchestrator := app.NewOrchestrator(memory, agent, tools, strategyRegistry)
	if dexter != nil {
		orchestrator = orchestrator.WithDexter(dexter)
	}

	// Create HTTP server
	server := httpapi.NewServer(orchestrator, db.DB)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    ":" + *port,
		Handler: server,
	}

	// Run server in background
	go func() {
		log.Printf("jax-orchestrator HTTP server listening on :%s", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("shutdown signal received, gracefully shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("server stopped")
}
